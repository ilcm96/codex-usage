package syncclient

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/tidwall/gjson"

	"github.com/ilcm96/codex-usage/internal/cli/cache"
	"github.com/ilcm96/codex-usage/internal/cli/codex"
)

type Options struct {
	SessionsDir string
	CacheDir    string
	ServerURL   string
	DeviceToken string
	Limit       int
	BatchSize   int
}

type Result struct {
	Scanned  int
	Uploaded int
	Skipped  int
	Failed   int
	Errors   []string
}

type stateEntry struct {
	RawSHA256 string `json:"rawSha256"`
	Uploaded  bool   `json:"uploaded"`
	UpdatedAt string `json:"updatedAt"`
}

type metadata struct {
	Device struct {
		Name     string `json:"name"`
		Hostname string `json:"hostname"`
		Platform string `json:"platform"`
	} `json:"device"`
	Session struct {
		ID           string `json:"id"`
		Path         string `json:"path"`
		RawSHA256    string `json:"raw_sha256"`
		RawSizeBytes int64  `json:"raw_size_bytes"`
		RawField     string `json:"raw_field"`
	} `json:"session"`
	Workspace struct {
		CWD           string `json:"cwd"`
		GitRoot       string `json:"git_root"`
		RelativePath  string `json:"relative_path"`
		RepositoryURL string `json:"repository_url"`
		Branch        string `json:"branch"`
		CommitHash    string `json:"commit_hash"`
	} `json:"workspace"`
}

type sessionMeta struct {
	ID            string
	CWD           string
	RepositoryURL string
	Branch        string
	CommitHash    string
}

type uploadItem struct {
	Path string
	Meta metadata
	Raw  []byte
}

type batchItemResponse struct {
	SessionID string `json:"session_id"`
	Status    string `json:"status"`
	RawSHA256 string `json:"raw_sha256"`
	Error     string `json:"error"`
}

type batchResponse struct {
	Status   string              `json:"status"`
	Ingested int                 `json:"ingested"`
	Skipped  int                 `json:"skipped"`
	Failed   int                 `json:"failed"`
	Items    []batchItemResponse `json:"items"`
}

func Run(ctx context.Context, opt Options) (Result, error) {
	if opt.ServerURL == "" {
		return Result{}, fmt.Errorf("server URL is required")
	}
	if opt.DeviceToken == "" {
		return Result{}, fmt.Errorf("device token is required")
	}
	files, err := codex.DiscoverSessionFilesWithInfo(opt.SessionsDir)
	if err != nil {
		return Result{}, err
	}
	for i, j := 0, len(files)-1; i < j; i, j = i+1, j-1 {
		files[i], files[j] = files[j], files[i]
	}

	statePath := filepath.Join(opt.CacheDir, "sync-v1.json")
	st, err := cache.LoadCache[stateEntry](statePath)
	if err != nil {
		st = cache.CacheV1[stateEntry]{Version: cache.CacheVersion, Files: map[string]stateEntry{}}
	}
	if st.Files == nil {
		st.Files = map[string]stateEntry{}
	}

	if opt.BatchSize <= 0 {
		opt.BatchSize = 50
	}

	client := &http.Client{Timeout: 60 * time.Second}
	var result Result
	var pending []uploadItem
	flush := func() {
		if len(pending) == 0 {
			return
		}
		res, err := uploadBatch(ctx, client, opt.ServerURL, opt.DeviceToken, pending)
		if err != nil {
			for _, item := range pending {
				result.Failed++
				result.addError(item.Path, err)
			}
			pending = pending[:0]
			return
		}
		bySHA := make(map[string]batchItemResponse, len(res.Items))
		for _, item := range res.Items {
			bySHA[item.RawSHA256] = item
		}
		for _, item := range pending {
			itemRes, ok := bySHA[item.Meta.Session.RawSHA256]
			if !ok {
				result.Failed++
				result.addError(item.Path, fmt.Errorf("missing batch response item"))
				continue
			}
			switch itemRes.Status {
			case "ingested":
				st.Files[item.Path] = stateEntry{
					RawSHA256: item.Meta.Session.RawSHA256,
					Uploaded:  true,
					UpdatedAt: time.Now().Format(time.RFC3339),
				}
				result.Uploaded++
			case "skipped":
				st.Files[item.Path] = stateEntry{
					RawSHA256: item.Meta.Session.RawSHA256,
					Uploaded:  true,
					UpdatedAt: time.Now().Format(time.RFC3339),
				}
				result.Skipped++
			default:
				result.Failed++
				errText := itemRes.Error
				if errText == "" {
					errText = "batch item failed"
				}
				result.addError(item.Path, fmt.Errorf("%s", errText))
			}
		}
		pending = pending[:0]
	}

	for _, file := range files {
		if opt.Limit > 0 && result.Scanned >= opt.Limit {
			break
		}
		result.Scanned++

		raw, rawSHA, err := readAndHash(file.Path)
		if err != nil {
			result.Failed++
			result.addError(file.Path, err)
			continue
		}
		if ent := st.Files[file.Path]; ent.Uploaded && ent.RawSHA256 == rawSHA {
			result.Skipped++
			continue
		}

		pending = append(pending, uploadItem{
			Path: file.Path,
			Meta: buildMetadata(file.Path, raw, rawSHA),
			Raw:  raw,
		})
		if len(pending) >= opt.BatchSize {
			flush()
		}
	}
	flush()

	if err := cache.SaveCache(statePath, st); err != nil {
		return result, err
	}
	return result, nil
}

func (r *Result) addError(path string, err error) {
	if len(r.Errors) >= 5 {
		return
	}
	r.Errors = append(r.Errors, fmt.Sprintf("%s: %v", path, err))
}

func readAndHash(path string) ([]byte, string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}
	sum := sha256.Sum256(raw)
	return raw, hex.EncodeToString(sum[:]), nil
}

func buildMetadata(path string, raw []byte, rawSHA string) metadata {
	sm := extractSessionMeta(raw)
	hostname, _ := os.Hostname()
	out := metadata{}
	out.Device.Name = hostname
	out.Device.Hostname = hostname
	out.Device.Platform = runtime.GOOS + "/" + runtime.GOARCH
	out.Session.ID = firstNonEmpty(sm.ID, sessionIDFromPath(path))
	out.Session.Path = path
	out.Session.RawSHA256 = rawSHA
	out.Session.RawSizeBytes = int64(len(raw))
	out.Workspace.CWD = sm.CWD
	out.Workspace.RepositoryURL = sm.RepositoryURL
	out.Workspace.Branch = sm.Branch
	out.Workspace.CommitHash = sm.CommitHash
	out.enrichGit()
	return out
}

func extractSessionMeta(raw []byte) sessionMeta {
	for _, line := range bytes.Split(raw, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if len(line) == 0 || !gjson.ValidBytes(line) {
			continue
		}
		if gjson.GetBytes(line, "type").String() != "session_meta" {
			continue
		}
		payload := gjson.GetBytes(line, "payload")
		return sessionMeta{
			ID:            payload.Get("id").String(),
			CWD:           payload.Get("cwd").String(),
			RepositoryURL: payload.Get("git.repository_url").String(),
			Branch:        payload.Get("git.branch").String(),
			CommitHash:    payload.Get("git.commit_hash").String(),
		}
	}
	return sessionMeta{}
}

func (m *metadata) enrichGit() {
	cwd := m.Workspace.CWD
	if cwd == "" {
		return
	}
	if st, err := os.Stat(cwd); err != nil || !st.IsDir() {
		return
	}
	gitRoot := runGit(cwd, "rev-parse", "--show-toplevel")
	m.Workspace.GitRoot = gitRoot
	if gitRoot != "" {
		if rel, err := filepath.Rel(gitRoot, cwd); err == nil && rel != "." {
			m.Workspace.RelativePath = rel
		}
	}
	m.Workspace.RepositoryURL = firstNonEmpty(m.Workspace.RepositoryURL, runGit(cwd, "remote", "get-url", "origin"))
	m.Workspace.Branch = firstNonEmpty(m.Workspace.Branch, runGit(cwd, "branch", "--show-current"))
	m.Workspace.CommitHash = firstNonEmpty(m.Workspace.CommitHash, runGit(cwd, "rev-parse", "HEAD"))
}

func uploadBatch(ctx context.Context, client *http.Client, serverURL string, token string, items []uploadItem) (batchResponse, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	metas := make([]metadata, 0, len(items))
	for i, item := range items {
		meta := item.Meta
		meta.Session.RawField = fmt.Sprintf("raw_%d", i)
		metas = append(metas, meta)
	}
	metaPart, err := writer.CreateFormField("metadata")
	if err != nil {
		return batchResponse{}, err
	}
	if err := json.NewEncoder(metaPart).Encode(metas); err != nil {
		return batchResponse{}, err
	}

	for i, item := range items {
		filePart, err := writer.CreateFormFile(fmt.Sprintf("raw_%d", i), item.Meta.Session.ID+".jsonl")
		if err != nil {
			return batchResponse{}, err
		}
		if _, err := io.Copy(filePart, bytes.NewReader(item.Raw)); err != nil {
			return batchResponse{}, err
		}
	}
	if err := writer.Close(); err != nil {
		return batchResponse{}, err
	}

	endpoint := strings.TrimRight(serverURL, "/") + "/api/ingest/sessions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, &body)
	if err != nil {
		return batchResponse{}, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Device-Token", token)

	res, err := client.Do(req)
	if err != nil {
		return batchResponse{}, err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
		return batchResponse{}, fmt.Errorf("upload failed: %s: %s", res.Status, strings.TrimSpace(string(b)))
	}
	var out batchResponse
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return batchResponse{}, err
	}
	return out, nil
}

func runGit(cwd string, args ...string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = cwd
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func sessionIDFromPath(path string) string {
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	if strings.HasPrefix(base, "rollout-") {
		parts := strings.Split(base, "-")
		if len(parts) > 4 {
			return strings.Join(parts[4:], "-")
		}
	}
	return base
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
