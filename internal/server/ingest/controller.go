package ingest

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ilcm96/codex-usage/internal/server/httpapi"
	"github.com/ilcm96/codex-usage/internal/server/ingeststore"
)

type Controller struct {
	service        Service
	maxUploadBytes int64
}

func NewController(service Service, maxUploadBytes int64) Controller {
	return Controller{service: service, maxUploadBytes: maxUploadBytes}
}

func (c Controller) Register(mux *http.ServeMux, protect func(http.Handler) http.Handler) {
	mux.Handle("POST /api/ingest/sessions", protect(http.HandlerFunc(c.handleSessions)))
}

// handleSessions ingests uploaded Codex sessions in a batch.
// @Summary Ingest sessions
// @Tags Ingest
// @Accept multipart/form-data
// @Produce json
// @Security DeviceTokenAuth
// @Param metadata formData string true "JSON-encoded ingest metadata array"
// @Param raw_0 formData file true "Raw JSONL file"
// @Success 202 {object} BatchResponse
// @Failure 400 {object} httpapi.ErrorResponse
// @Failure 401 {object} httpapi.ErrorResponse
// @Failure 500 {object} httpapi.ErrorResponse
// @Router /api/ingest/sessions [post]
func (c Controller) handleSessions(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, c.maxUploadBytes)
	if err := r.ParseMultipartForm(c.maxUploadBytes); err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "invalid multipart upload")
		return
	}

	var metadata []ingeststore.Metadata
	if err := json.Unmarshal([]byte(r.FormValue("metadata")), &metadata); err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "invalid metadata")
		return
	}
	if len(metadata) == 0 {
		httpapi.WriteError(w, http.StatusBadRequest, "metadata must include at least one session")
		return
	}

	inputs := make([]ingeststore.RawInput, 0, len(metadata))
	responses := make([]ItemResponse, 0, len(metadata))
	for i, meta := range metadata {
		if meta.Session.RawField == "" {
			meta.Session.RawField = fmt.Sprintf("raw_%d", i)
		}
		item, input, ok := c.prepareRawInput(r, meta)
		if ok {
			inputs = append(inputs, input)
		}
		responses = append(responses, item)
	}

	results, err := c.service.StoreRawBatch(r.Context(), inputs)
	if err != nil {
		slog.Error("failed to ingest session batch", "error", err)
		httpapi.WriteError(w, http.StatusInternalServerError, "failed to ingest session")
		return
	}

	bySHA := make(map[string]ingeststore.Result, len(results))
	for _, result := range results {
		bySHA[result.RawSHA256] = result
	}
	for i, item := range responses {
		if item.Status == "failed" {
			continue
		}
		result, ok := bySHA[item.RawSHA256]
		if !ok {
			responses[i].Status = "failed"
			responses[i].Error = "missing ingest result"
			continue
		}
		responses[i].Status = result.Status
		responses[i].SessionID = result.SessionID
		responses[i].RawFilePath = result.RawPath
		responses[i].RawSizeBytes = result.RawSizeBytes
		if result.Error != "" {
			responses[i].Status = "failed"
			responses[i].Error = result.Error
		}
	}

	response := BatchResponse{Status: "accepted", Items: responses}
	for _, item := range responses {
		switch item.Status {
		case "ingested":
			response.Ingested++
		case "skipped":
			response.Skipped++
		default:
			response.Failed++
		}
	}
	httpapi.WriteJSON(w, http.StatusAccepted, response)
}

func (c Controller) prepareRawInput(r *http.Request, meta ingeststore.Metadata) (ItemResponse, ingeststore.RawInput, bool) {
	item := ItemResponse{
		SessionID: meta.Session.ID,
		RawSHA256: meta.Session.RawSHA256,
	}
	if meta.Session.ID == "" || meta.Session.RawSHA256 == "" {
		item.Status = "failed"
		item.Error = "session.id and session.raw_sha256 are required"
		return item, ingeststore.RawInput{}, false
	}

	raw, header, err := r.FormFile(meta.Session.RawField)
	if err != nil {
		item.Status = "failed"
		item.Error = "raw JSONL file is required"
		return item, ingeststore.RawInput{}, false
	}
	defer raw.Close()
	item.Filename = header.Filename

	sessionID := safePathPart(meta.Session.ID)
	deviceName := safePathPart(firstNonEmpty(meta.Device.Hostname, meta.Device.Name, "unknown-device"))
	dir := filepath.Join(c.service.rawDir, deviceName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		item.Status = "failed"
		item.Error = "failed to create raw storage"
		return item, ingeststore.RawInput{}, false
	}

	path := filepath.Join(dir, sessionID+".jsonl")
	tmp := path + fmt.Sprintf(".%d.tmp", time.Now().UnixNano())
	f, err := os.Create(tmp)
	if err != nil {
		item.Status = "failed"
		item.Error = "failed to write raw file"
		return item, ingeststore.RawInput{}, false
	}

	hasher := sha256.New()
	size, copyErr := io.Copy(io.MultiWriter(f, hasher), raw)
	closeErr := f.Close()
	if copyErr != nil || closeErr != nil {
		_ = os.Remove(tmp)
		item.Status = "failed"
		item.Error = "failed to store raw file"
		return item, ingeststore.RawInput{}, false
	}

	rawSHA256 := hex.EncodeToString(hasher.Sum(nil))
	if !strings.EqualFold(rawSHA256, meta.Session.RawSHA256) {
		_ = os.Remove(tmp)
		item.Status = "failed"
		item.Error = "raw_sha256 does not match payload"
		return item, ingeststore.RawInput{}, false
	}
	if meta.Session.RawSizeBytes > 0 && meta.Session.RawSizeBytes != size {
		_ = os.Remove(tmp)
		item.Status = "failed"
		item.Error = "raw_size_bytes does not match payload"
		return item, ingeststore.RawInput{}, false
	}

	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		item.Status = "failed"
		item.Error = "failed to finalize raw file"
		return item, ingeststore.RawInput{}, false
	}

	item.Status = "pending"
	item.RawFilePath = path
	item.RawSizeBytes = size
	input := ingeststore.RawInput{
		Metadata:     meta,
		RawPath:      path,
		RawSizeBytes: size,
	}
	return item, input, true
}

func safePathPart(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "/", "_")
	value = strings.ReplaceAll(value, "\\", "_")
	value = strings.ReplaceAll(value, ":", "_")
	if value == "" || value == "." || value == ".." {
		return "unknown"
	}
	return value
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
