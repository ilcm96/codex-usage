package ingest_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ilcm96/codex-usage/internal/server/auth"
	"github.com/ilcm96/codex-usage/internal/server/ingest"
	"github.com/ilcm96/codex-usage/internal/server/servertest"
)

func TestController_RegisterAndHandleSession(t *testing.T) {
	ctx := context.Background()
	db := servertest.StartPostgres(t)
	servertest.Reset(t, ctx, db)

	repository := ingest.NewPostgresRepository(db)
	service := ingest.NewService(repository, t.TempDir())
	controller := ingest.NewController(service, 1<<20)
	mux := http.NewServeMux()
	controller.Register(mux, func(next http.Handler) http.Handler {
		return auth.RequireDeviceToken(map[string]struct{}{"device-token": {}}, next)
	})

	rawJSONL := strings.Join([]string{
		`{"timestamp":"2026-02-08T00:00:00Z","type":"session_meta","payload":{"id":"ingest-controller-session","timestamp":"2026-02-08T00:00:00Z","cwd":"/repo/ingest-controller","originator":"codex","source":"cli","cli_version":"1.0.0","model_provider":"openai","git":{"repository_url":"git@github.com:acme/ingest-controller.git","branch":"main","commit_hash":"abc123"}}}`,
		`{"timestamp":"2026-02-08T00:00:01Z","type":"turn_context","payload":{"model":"gpt-5"}}`,
		`{"timestamp":"2026-02-08T00:00:02Z","type":"message","role":"user","content":"Please ingest this session."}`,
		`{"timestamp":"2026-02-08T00:00:03Z","type":"message","role":"assistant","content":"Session ingested."}`,
		`{"timestamp":"2026-02-08T00:00:04Z","type":"event_msg","payload":{"type":"token_count","info":{"last_token_usage":{"input_tokens":10,"cached_input_tokens":2,"output_tokens":5,"reasoning_output_tokens":1,"total_tokens":16}}}}`,
	}, "\n") + "\n"
	rawBytes := []byte(rawJSONL)
	rawSum := sha256.Sum256(rawBytes)
	rawFixture := servertest.WriteRaw(t, rawBytes, "ingest-controller-session.jsonl")

	metadata := map[string]any{
		"device": map[string]any{
			"name":     "Ingest Device",
			"hostname": "ingest-device",
			"platform": "darwin",
		},
		"session": map[string]any{
			"id":             "ingest-controller-session",
			"path":           "/source/ingest-controller-session.jsonl",
			"raw_sha256":     hex.EncodeToString(rawSum[:]),
			"raw_size_bytes": len(rawBytes),
		},
		"workspace": map[string]any{
			"cwd":            "/repo/ingest-controller",
			"git_root":       "/repo/ingest-controller",
			"relative_path":  ".",
			"repository_url": "git@github.com:acme/ingest-controller.git",
			"branch":         "main",
			"commit_hash":    "abc123",
		},
	}

	t.Run("requires device token", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		mux.ServeHTTP(recorder, newMultipartRequest(t, metadata, rawFixture.RawBytes, ""))
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("unexpected status: got %d body=%s", recorder.Code, recorder.Body.String())
		}
	})

	t.Run("rejects invalid metadata", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/api/ingest/sessions", strings.NewReader("not multipart"))
		request.Header.Set("X-Device-Token", "device-token")
		mux.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("unexpected status: got %d body=%s", recorder.Code, recorder.Body.String())
		}
	})

	t.Run("ingests valid upload and skips duplicate", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		mux.ServeHTTP(recorder, newMultipartRequest(t, metadata, rawFixture.RawBytes, "device-token"))
		if recorder.Code != http.StatusAccepted {
			t.Fatalf("unexpected status: got %d body=%s", recorder.Code, recorder.Body.String())
		}
		var response struct {
			Status   string `json:"status"`
			Ingested int    `json:"ingested"`
			Skipped  int    `json:"skipped"`
			Failed   int    `json:"failed"`
			Items    []struct {
				Status       string `json:"status"`
				SessionID    string `json:"session_id"`
				RawFilePath  string `json:"raw_file_path"`
				RawSizeBytes int64  `json:"raw_size_bytes"`
			} `json:"items"`
		}
		decodeJSON(t, recorder, &response)
		if response.Status != "accepted" || response.Ingested != 1 || response.Skipped != 0 || response.Failed != 0 || len(response.Items) != 1 {
			t.Fatalf("unexpected ingest response: %+v", response)
		}
		item := response.Items[0]
		if item.Status != "ingested" || item.SessionID != "ingest-controller-session" || item.RawFilePath == "" || item.RawSizeBytes != int64(len(rawBytes)) {
			t.Fatalf("unexpected ingest item response: %+v", item)
		}

		var sessions, archives, summaries, searchDocuments, rollups int64
		if err := db.QueryRow(ctx, `
			SELECT
				(SELECT count(*)::bigint FROM sessions),
				(SELECT count(*)::bigint FROM archive_files),
				(SELECT count(*)::bigint FROM session_summaries),
				(SELECT count(*)::bigint FROM search_documents),
				(SELECT count(*)::bigint FROM usage_rollups)
		`).Scan(&sessions, &archives, &summaries, &searchDocuments, &rollups); err != nil {
			t.Fatalf("load stored counts: %v", err)
		}
		if sessions != 1 || archives != 1 || summaries != 1 || searchDocuments != 2 || rollups != 1 {
			t.Fatalf("unexpected stored counts: sessions=%d archives=%d summaries=%d searchDocuments=%d rollups=%d", sessions, archives, summaries, searchDocuments, rollups)
		}

		duplicate := httptest.NewRecorder()
		mux.ServeHTTP(duplicate, newMultipartRequest(t, metadata, rawFixture.RawBytes, "device-token"))
		if duplicate.Code != http.StatusAccepted {
			t.Fatalf("unexpected duplicate status: got %d body=%s", duplicate.Code, duplicate.Body.String())
		}
		var duplicateResponse struct {
			Status  string `json:"status"`
			Skipped int    `json:"skipped"`
			Items   []struct {
				Status    string `json:"status"`
				SessionID string `json:"session_id"`
			} `json:"items"`
		}
		decodeJSON(t, duplicate, &duplicateResponse)
		if duplicateResponse.Status != "accepted" || duplicateResponse.Skipped != 1 || len(duplicateResponse.Items) != 1 {
			t.Fatalf("unexpected duplicate response: %+v", duplicateResponse)
		}
		if duplicateResponse.Items[0].Status != "skipped" || duplicateResponse.Items[0].SessionID != "ingest-controller-session" {
			t.Fatalf("unexpected duplicate item response: %+v", duplicateResponse.Items[0])
		}
	})
}

func TestController_HandleSessionBatch(t *testing.T) {
	ctx := context.Background()
	db := servertest.StartPostgres(t)
	servertest.Reset(t, ctx, db)

	repository := ingest.NewPostgresRepository(db)
	service := ingest.NewService(repository, t.TempDir())
	controller := ingest.NewController(service, 1<<20)
	mux := http.NewServeMux()
	controller.Register(mux, func(next http.Handler) http.Handler {
		return auth.RequireDeviceToken(map[string]struct{}{"device-token": {}}, next)
	})

	firstRaw := []byte(strings.Join([]string{
		`{"timestamp":"2026-02-08T00:00:00Z","type":"session_meta","payload":{"id":"batch-session-a","timestamp":"2026-02-08T00:00:00Z","cwd":"/repo/batch","originator":"codex","source":"cli","cli_version":"1.0.0","model_provider":"openai","git":{"repository_url":"git@github.com:acme/batch.git","branch":"main","commit_hash":"abc123"}}}`,
		`{"timestamp":"2026-02-08T00:00:01Z","type":"turn_context","payload":{"model":"gpt-5"}}`,
		`{"timestamp":"2026-02-08T00:00:02Z","type":"message","role":"user","content":"First batch session."}`,
		`{"timestamp":"2026-02-08T00:00:03Z","type":"event_msg","payload":{"type":"token_count","info":{"last_token_usage":{"input_tokens":10,"cached_input_tokens":2,"output_tokens":5,"reasoning_output_tokens":1,"total_tokens":16}}}}`,
	}, "\n") + "\n")
	secondRaw := []byte(strings.Join([]string{
		`{"timestamp":"2026-02-08T01:00:00Z","type":"session_meta","payload":{"id":"batch-session-b","timestamp":"2026-02-08T01:00:00Z","cwd":"/repo/batch","originator":"codex","source":"cli","cli_version":"1.0.0","model_provider":"openai","git":{"repository_url":"git@github.com:acme/batch.git","branch":"main","commit_hash":"def456"}}}`,
		`{"timestamp":"2026-02-08T01:00:01Z","type":"turn_context","payload":{"model":"gpt-5"}}`,
		`{"timestamp":"2026-02-08T01:00:02Z","type":"message","role":"user","content":"Second batch session."}`,
		`{"timestamp":"2026-02-08T01:00:03Z","type":"event_msg","payload":{"type":"token_count","info":{"last_token_usage":{"input_tokens":20,"cached_input_tokens":4,"output_tokens":10,"reasoning_output_tokens":2,"total_tokens":32}}}}`,
	}, "\n") + "\n")

	metadata := []map[string]any{
		batchMetadata("batch-session-a", firstRaw, "raw_0"),
		batchMetadata("batch-session-b", secondRaw, "raw_1"),
	}
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, newMultipartBatchRequest(t, metadata, [][]byte{firstRaw, secondRaw}, "device-token"))
	if recorder.Code != http.StatusAccepted {
		t.Fatalf("unexpected status: got %d body=%s", recorder.Code, recorder.Body.String())
	}

	var response struct {
		Ingested int `json:"ingested"`
		Skipped  int `json:"skipped"`
		Failed   int `json:"failed"`
		Items    []struct {
			Status string `json:"status"`
		} `json:"items"`
	}
	decodeJSON(t, recorder, &response)
	if response.Ingested != 2 || response.Skipped != 0 || response.Failed != 0 || len(response.Items) != 2 {
		t.Fatalf("unexpected batch response: %+v", response)
	}

	var sessions, usageEvents, rollups, rollupTokens int64
	if err := db.QueryRow(ctx, `
		SELECT
			(SELECT count(*)::bigint FROM sessions),
			(SELECT count(*)::bigint FROM usage_events),
			(SELECT count(*)::bigint FROM usage_rollups),
			(SELECT total_tokens FROM usage_rollups WHERE bucket_date = '2026-02-08')
	`).Scan(&sessions, &usageEvents, &rollups, &rollupTokens); err != nil {
		t.Fatalf("load batch counts: %v", err)
	}
	if sessions != 2 || usageEvents != 2 || rollups != 1 || rollupTokens != 48 {
		t.Fatalf("unexpected batch counts: sessions=%d usageEvents=%d rollups=%d rollupTokens=%d", sessions, usageEvents, rollups, rollupTokens)
	}
}

func newMultipartRequest(t *testing.T, metadata map[string]any, raw []byte, token string) *http.Request {
	t.Helper()

	metadata["session"].(map[string]any)["raw_field"] = "raw_0"
	return newMultipartBatchRequest(t, []map[string]any{metadata}, [][]byte{raw}, token)
}

func newMultipartBatchRequest(t *testing.T, metadata []map[string]any, raws [][]byte, token string) *http.Request {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}
	if err := writer.WriteField("metadata", string(metadataBytes)); err != nil {
		t.Fatalf("write metadata field: %v", err)
	}
	for i, raw := range raws {
		rawWriter, err := writer.CreateFormFile("raw_"+fmt.Sprint(i), "session.jsonl")
		if err != nil {
			t.Fatalf("create raw field: %v", err)
		}
		if _, err := rawWriter.Write(raw); err != nil {
			t.Fatalf("write raw field: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/ingest/sessions", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	if token != "" {
		request.Header.Set("X-Device-Token", token)
	}
	return request
}

func batchMetadata(sessionID string, raw []byte, rawField string) map[string]any {
	rawSum := sha256.Sum256(raw)
	return map[string]any{
		"device": map[string]any{
			"name":     "Batch Device",
			"hostname": "batch-device",
			"platform": "darwin",
		},
		"session": map[string]any{
			"id":             sessionID,
			"path":           "/source/" + sessionID + ".jsonl",
			"raw_sha256":     hex.EncodeToString(rawSum[:]),
			"raw_size_bytes": len(raw),
			"raw_field":      rawField,
		},
		"workspace": map[string]any{
			"cwd":            "/repo/batch",
			"git_root":       "/repo/batch",
			"relative_path":  ".",
			"repository_url": "git@github.com:acme/batch.git",
			"branch":         "main",
			"commit_hash":    "abc123",
		},
	}
}

func decodeJSON(t *testing.T, recorder *httptest.ResponseRecorder, out any) {
	t.Helper()

	if err := json.NewDecoder(recorder.Body).Decode(out); err != nil {
		t.Fatalf("decode json: %v\nbody=%s", err, recorder.Body.String())
	}
}
