package server

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ilcm96/codex-usage/internal/server/archive"
	"github.com/ilcm96/codex-usage/internal/server/auth"
	"github.com/ilcm96/codex-usage/internal/server/config"
	"github.com/ilcm96/codex-usage/internal/server/projects"
	"github.com/ilcm96/codex-usage/internal/server/search"
	"github.com/ilcm96/codex-usage/internal/server/servertest"
	"github.com/ilcm96/codex-usage/internal/server/sessions"
)

func TestServerRoutes_RequireSessionAndExposeDataElementAPIs(t *testing.T) {
	ctx := context.Background()
	db := servertest.StartPostgres(t)
	servertest.Reset(t, ctx, db)
	servertest.SeedFixture(t, ctx, db)

	client := newRouteClient(newTestHandler(t, db))

	swaggerSpec := client.get(t, "/swagger/doc.json")
	assertStatus(t, swaggerSpec, http.StatusOK)
	if !strings.Contains(swaggerSpec.Body.String(), `"/api/usage/series"`) {
		t.Fatalf("swagger spec does not include usage series path: %s", swaggerSpec.Body.String())
	}

	swaggerUI := client.get(t, "/swagger/index.html")
	assertStatus(t, swaggerUI, http.StatusOK)
	if !strings.Contains(swaggerUI.Body.String(), "SwaggerUIBundle") {
		t.Fatalf("swagger ui response does not look like swagger ui: %s", swaggerUI.Body.String())
	}

	unauthorized := client.get(t, "/api/usage/totals")
	assertStatus(t, unauthorized, http.StatusUnauthorized)

	badLogin := httptest.NewRecorder()
	client.handler.ServeHTTP(badLogin, httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"password":"wrong"}`)))
	assertStatus(t, badLogin, http.StatusUnauthorized)

	client.login(t)

	var summary struct {
		TotalTokens     int64   `json:"totalTokens"`
		InputTokens     int64   `json:"inputTokens"`
		CostUSD         float64 `json:"costUsd"`
		Sessions        int64   `json:"sessions"`
		Projects        int64   `json:"projects"`
		Devices         int64   `json:"devices"`
		PatchAddedLines int64   `json:"patchAddedLines"`
	}
	client.getJSON(t, "/api/usage/totals", &summary)
	if summary.TotalTokens != 380 || summary.InputTokens != 130 || summary.Sessions != 2 || summary.Projects != 1 || summary.Devices != 1 || summary.PatchAddedLines != 4 {
		t.Fatalf("unexpected summary: %+v", summary)
	}

	var windows []struct {
		Days   int `json:"days"`
		Totals struct {
			TotalTokens int64 `json:"totalTokens"`
			Sessions    int64 `json:"sessions"`
		} `json:"totals"`
	}
	client.getJSON(t, "/api/usage/windows", &windows)
	if len(windows) != 3 || windows[0].Days != 1 || windows[0].Totals.TotalTokens != 300 || windows[1].Totals.Sessions != 2 {
		t.Fatalf("unexpected usage windows: %+v", windows)
	}

	var options struct {
		Devices      []filterOption `json:"devices"`
		Repositories []filterOption `json:"repositories"`
		Projects     []filterOption `json:"projects"`
		Models       []filterOption `json:"models"`
		Branches     []filterOption `json:"branches"`
	}
	client.getJSON(t, "/api/filter-options", &options)
	if len(options.Devices) != 1 || options.Devices[0].Count != 2 {
		t.Fatalf("unexpected device options: %+v", options.Devices)
	}
	if len(options.Models) != 2 || options.Models[0].Label != "gpt-5" {
		t.Fatalf("unexpected model options: %+v", options.Models)
	}
	if len(options.Branches) != 2 {
		t.Fatalf("unexpected branch options: %+v", options.Branches)
	}

	var usage []struct {
		Bucket      string `json:"bucket"`
		TotalTokens int64  `json:"totalTokens"`
	}
	client.getJSON(t, "/api/usage/series?bucket=day", &usage)
	if len(usage) != 2 || usage[0].TotalTokens != 80 || usage[1].TotalTokens != 300 {
		t.Fatalf("unexpected usage buckets: %+v", usage)
	}

	var breakdown []struct {
		Label       string `json:"label"`
		Sessions    int64  `json:"sessions"`
		TotalTokens int64  `json:"totalTokens"`
	}
	client.getJSON(t, "/api/usage/breakdown?groupBy=model", &breakdown)
	if len(breakdown) != 2 || breakdown[0].Label != "gpt-5" || breakdown[0].Sessions != 2 || breakdown[0].TotalTokens != 330 {
		t.Fatalf("unexpected usage breakdown: %+v", breakdown)
	}

	var usageSummary struct {
		Current struct {
			TotalTokens     int64 `json:"totalTokens"`
			Messages        int64 `json:"messages"`
			ToolCalls       int64 `json:"toolCalls"`
			PatchAddedLines int64 `json:"patchAddedLines"`
		} `json:"current"`
		ActiveDays   int64   `json:"activeDays"`
		CacheHitRate float64 `json:"cacheHitRate"`
	}
	client.getJSON(t, "/api/usage/summary", &usageSummary)
	if usageSummary.Current.TotalTokens != 380 || usageSummary.Current.Messages != 3 || usageSummary.Current.ToolCalls != 1 || usageSummary.Current.PatchAddedLines != 4 || usageSummary.ActiveDays != 2 {
		t.Fatalf("unexpected usage summary: %+v", usageSummary)
	}
	if usageSummary.CacheHitRate <= 0 {
		t.Fatalf("expected positive cache hit rate, got %+v", usageSummary)
	}

	unsupported := client.get(t, "/api/usage/breakdown?groupBy=branch")
	assertStatus(t, unsupported, http.StatusBadRequest)
}

func TestServerRoutes_DisablesSwaggerInProduction(t *testing.T) {
	ctx := context.Background()
	db := servertest.StartPostgres(t)
	servertest.Reset(t, ctx, db)

	client := newRouteClient(newTestHandlerWithConfig(t, db, config.Config{
		Environment: "PROD",
	}))

	swaggerSpec := client.get(t, "/swagger/doc.json")
	assertStatus(t, swaggerSpec, http.StatusNotFound)

	swaggerUI := client.get(t, "/swagger/index.html")
	assertStatus(t, swaggerUI, http.StatusNotFound)
}

func TestServerRoutes_SessionsSearchProjectsArchiveAndExports(t *testing.T) {
	ctx := context.Background()
	db := servertest.StartPostgres(t)
	servertest.Reset(t, ctx, db)
	fixture := servertest.SeedFixture(t, ctx, db)

	client := newRouteClient(newTestHandler(t, db))
	client.login(t)

	var list sessions.ListResult
	client.getJSON(t, "/api/sessions?sort=cost_desc&limit=1", &list)
	if list.Total != 2 || list.NextOffset != 1 || len(list.Items) != 1 || list.Items[0].ID != fixture.SessionAlpha {
		t.Fatalf("unexpected sessions list: %+v", list)
	}
	if list.Totals.TotalTokens != 380 || list.Totals.Sessions != 2 {
		t.Fatalf("unexpected sessions totals: %+v", list.Totals)
	}
	if list.Totals.PatchAddedLines != 4 {
		t.Fatalf("unexpected sessions patch totals: %+v", list.Totals)
	}

	var simple []sessions.SimpleSession
	client.getJSON(t, "/api/sessions/compact?q=cleanup", &simple)
	if len(simple) != 1 || simple[0].ID != fixture.SessionBeta {
		t.Fatalf("unexpected simple session search: %+v", simple)
	}

	var detail sessions.DetailResult
	client.getJSON(t, "/api/sessions/"+fixture.SessionAlpha, &detail)
	if detail.Session.ID != fixture.SessionAlpha || detail.Session.DisplayTitle != "Cache tuning" || len(detail.Models) != 2 || detail.Models[0].Model != "gpt-5" {
		t.Fatalf("unexpected detail: %+v", detail)
	}
	if detail.Session.PatchAddedLines != 4 {
		t.Fatalf("unexpected detail patch added lines: %+v", detail.Session)
	}

	notFound := client.get(t, "/api/sessions/not-found")
	assertStatus(t, notFound, http.StatusNotFound)

	var reader sessions.ReaderResult
	client.getJSON(t, "/api/sessions/"+fixture.SessionAlpha+"/reader?q=cache", &reader)
	if reader.Total != 1 || len(reader.Items) != 1 || !strings.Contains(reader.Items[0].UserText, "cache") {
		t.Fatalf("unexpected reader result: %+v", reader)
	}

	var timeline sessions.TimelineResult
	client.getJSON(t, "/api/sessions/"+fixture.SessionAlpha+"/timeline?kind=tool", &timeline)
	if timeline.Total != 1 || len(timeline.Items) != 1 || timeline.Items[0].ToolName != "shell" {
		t.Fatalf("unexpected timeline: %+v", timeline)
	}

	badTimeline := client.get(t, "/api/sessions/"+fixture.SessionAlpha+"/timeline?kind=unknown")
	assertStatus(t, badTimeline, http.StatusBadRequest)

	var searchResult search.SearchResult
	client.getJSON(t, "/api/search?q=postgres&kind=tool", &searchResult)
	if searchResult.Total != 1 || len(searchResult.Items) != 1 || searchResult.Items[0].ToolName != "shell" {
		t.Fatalf("unexpected search result: %+v", searchResult)
	}

	badSearch := client.get(t, "/api/search?q=postgres&kind=unknown")
	assertStatus(t, badSearch, http.StatusBadRequest)

	var repositories []projects.RepositorySummary
	client.getJSON(t, "/api/repositories", &repositories)
	if len(repositories) != 1 || repositories[0].Name != "codex-usage" || repositories[0].TotalTokens != 380 {
		t.Fatalf("unexpected repositories: %+v", repositories)
	}

	var projectList []projects.ProjectSummary
	client.getJSON(t, "/api/projects", &projectList)
	if len(projectList) != 1 || projectList[0].DisplayName != "codex-usage" || projectList[0].Sessions != 2 {
		t.Fatalf("unexpected projects: %+v", projectList)
	}

	var daily []struct {
		TotalTokens int64 `json:"totalTokens"`
	}
	client.getJSON(t, "/api/usage/series?bucket=day", &daily)
	if len(daily) != 2 || daily[0].TotalTokens != 80 || daily[1].TotalTokens != 300 {
		t.Fatalf("unexpected usage series: %+v", daily)
	}

	var models []struct {
		Label       string `json:"label"`
		TotalTokens int64  `json:"totalTokens"`
	}
	client.getJSON(t, "/api/usage/breakdown?groupBy=model&limit=1", &models)
	if len(models) != 1 || models[0].Label != "gpt-5" || models[0].TotalTokens != 330 {
		t.Fatalf("unexpected model usage breakdown: %+v", models)
	}

	var status archive.Status
	client.getJSON(t, "/api/archive/status", &status)
	if status.Sessions != 2 || status.Devices != 1 || status.RawBytes <= 0 || status.MissingRawFiles != 1 || status.MissingRawSHA != 1 {
		t.Fatalf("unexpected archive status: %+v", status)
	}

	var health archive.Health
	client.getJSON(t, "/api/archive/health", &health)
	if health.Status != "attention" || health.MissingArchiveRows != 1 || health.VerifiedArchiveRows != 1 {
		t.Fatalf("unexpected archive health: %+v", health)
	}

	var byDevice []archive.DeviceSummary
	client.getJSON(t, "/api/archive/devices", &byDevice)
	if len(byDevice) != 1 || byDevice[0].Sessions != 2 {
		t.Fatalf("unexpected archive by-device: %+v", byDevice)
	}

	var byRepository []archive.RepositorySummary
	client.getJSON(t, "/api/archive/repositories?limit=1", &byRepository)
	if len(byRepository) != 1 || byRepository[0].Sessions != 2 {
		t.Fatalf("unexpected archive by-repository: %+v", byRepository)
	}

	var integrity archive.IntegrityResult
	client.getJSON(t, "/api/archive/integrity", &integrity)
	if integrity.Checked != 2 || integrity.OK != 1 || integrity.MissingSHA != 1 {
		t.Fatalf("unexpected archive integrity: %+v", integrity)
	}

	csvExport := client.get(t, "/api/exports/sessions?format=csv")
	assertStatus(t, csvExport, http.StatusOK)
	if !strings.Contains(csvExport.Body.String(), "session-alpha") || !strings.Contains(csvExport.Header().Get("Content-Disposition"), "codex-sessions.csv") {
		t.Fatalf("unexpected csv export headers/body: headers=%v body=%s", csvExport.Header(), csvExport.Body.String())
	}

	markdownExport := client.get(t, "/api/sessions/"+fixture.SessionAlpha+"/export?format=markdown")
	assertStatus(t, markdownExport, http.StatusOK)
	if !strings.Contains(markdownExport.Body.String(), "Use Testcontainers with Postgres") {
		t.Fatalf("unexpected markdown export: %s", markdownExport.Body.String())
	}

	var rawEvents []map[string]any
	client.getJSON(t, "/api/sessions/"+fixture.SessionAlpha+"/export?format=json", &rawEvents)
	if len(rawEvents) != 1 || rawEvents[0]["type"] != "message" {
		t.Fatalf("unexpected json export: %+v", rawEvents)
	}
}

func TestServerRoutes_IngestSessionEndToEnd(t *testing.T) {
	ctx := context.Background()
	db := servertest.StartPostgres(t)
	servertest.Reset(t, ctx, db)

	client := newRouteClient(newTestHandler(t, db))
	rawJSONL := strings.Join([]string{
		`{"timestamp":"2026-02-08T00:00:00Z","type":"session_meta","payload":{"id":"ingest-session","timestamp":"2026-02-08T00:00:00Z","cwd":"/repo/ingested","originator":"codex","source":"cli","cli_version":"1.0.0","model_provider":"openai","git":{"repository_url":"git@github.com:acme/ingested.git","branch":"main","commit_hash":"abc123"}}}`,
		`{"timestamp":"2026-02-08T00:00:01Z","type":"turn_context","payload":{"model":"gpt-5"}}`,
		`{"timestamp":"2026-02-08T00:00:02Z","type":"message","role":"user","content":"Please add uploaded session tests."}`,
		`{"timestamp":"2026-02-08T00:00:03Z","type":"function_call","name":"shell","call_id":"call-1","status":"completed","arguments":"go test ./internal/server/..."}`,
		`{"timestamp":"2026-02-08T00:00:04Z","type":"function_call_output","call_id":"call-1","status":"completed","output":"tests passed"}`,
		`{"timestamp":"2026-02-08T00:00:04Z","type":"response_item","payload":{"type":"custom_tool_call","name":"apply_patch","call_id":"call-2","status":"completed","input":"*** Begin Patch\n*** Add File: main.go\n+package main\n+\n+func main() {}\n*** End Patch\n"}}`,
		`{"timestamp":"2026-02-08T00:00:05Z","type":"message","role":"assistant","content":"Uploaded session tests are ready."}`,
		`{"timestamp":"2026-02-08T00:00:06Z","type":"event_msg","payload":{"type":"token_count","info":{"last_token_usage":{"input_tokens":100,"cached_input_tokens":20,"output_tokens":50,"reasoning_output_tokens":10,"total_tokens":160}}}}`,
	}, "\n") + "\n"
	rawBytes := []byte(rawJSONL)
	rawSum := sha256.Sum256(rawBytes)

	metadata := map[string]any{
		"device": map[string]any{
			"name":     "Upload MacBook",
			"hostname": "upload-macbook",
			"platform": "darwin",
		},
		"session": map[string]any{
			"id":             "ingest-session",
			"path":           "/source/ingest-session.jsonl",
			"raw_sha256":     hex.EncodeToString(rawSum[:]),
			"raw_size_bytes": len(rawBytes),
		},
		"workspace": map[string]any{
			"cwd":            "/repo/ingested",
			"git_root":       "/repo/ingested",
			"relative_path":  ".",
			"repository_url": "git@github.com:acme/ingested.git",
			"branch":         "main",
			"commit_hash":    "abc123",
		},
	}

	unauthorized := httptest.NewRecorder()
	client.handler.ServeHTTP(unauthorized, newIngestRequest(t, metadata, rawBytes, ""))
	assertStatus(t, unauthorized, http.StatusUnauthorized)

	accepted := httptest.NewRecorder()
	client.handler.ServeHTTP(accepted, newIngestRequest(t, metadata, rawBytes, "device-token"))
	assertStatus(t, accepted, http.StatusAccepted)
	var ingestResponse struct {
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
	decodeRecorderJSON(t, accepted, &ingestResponse)
	if ingestResponse.Status != "accepted" || ingestResponse.Ingested != 1 || ingestResponse.Skipped != 0 || ingestResponse.Failed != 0 || len(ingestResponse.Items) != 1 {
		t.Fatalf("unexpected ingest response: %+v", ingestResponse)
	}
	if ingestResponse.Items[0].Status != "ingested" || ingestResponse.Items[0].SessionID != "ingest-session" || ingestResponse.Items[0].RawFilePath == "" || ingestResponse.Items[0].RawSizeBytes != int64(len(rawBytes)) {
		t.Fatalf("unexpected ingest item response: %+v", ingestResponse.Items[0])
	}

	client.login(t)
	var detail sessions.DetailResult
	client.getJSON(t, "/api/sessions/ingest-session", &detail)
	if detail.Session.ID != "ingest-session" || detail.Session.Repository != "ingested" || detail.Session.TotalTokens != 160 || detail.Session.ToolCallCount != 3 || detail.Session.PatchAddedLines != 3 {
		t.Fatalf("unexpected ingested session detail: %+v", detail)
	}

	var reader sessions.ReaderResult
	client.getJSON(t, "/api/sessions/ingest-session/reader", &reader)
	if reader.Total != 1 || !strings.Contains(reader.Items[0].UserText, "uploaded session tests") {
		t.Fatalf("unexpected ingested reader result: %+v", reader)
	}

	var searchResult search.SearchResult
	client.getJSON(t, "/api/search?q=uploaded&kind=message", &searchResult)
	if searchResult.Total != 2 {
		t.Fatalf("unexpected ingested search result: %+v", searchResult)
	}

	var archiveHealth archive.Health
	client.getJSON(t, "/api/archive/health", &archiveHealth)
	if archiveHealth.Status != "ok" || archiveHealth.Sessions != 1 || archiveHealth.ArchiveRows != 1 || archiveHealth.SearchDocuments == 0 {
		t.Fatalf("unexpected archive health after ingest: %+v", archiveHealth)
	}

	duplicate := httptest.NewRecorder()
	client.handler.ServeHTTP(duplicate, newIngestRequest(t, metadata, rawBytes, "device-token"))
	assertStatus(t, duplicate, http.StatusAccepted)
	var duplicateResponse struct {
		Status  string `json:"status"`
		Skipped int    `json:"skipped"`
		Items   []struct {
			Status    string `json:"status"`
			SessionID string `json:"session_id"`
		} `json:"items"`
	}
	decodeRecorderJSON(t, duplicate, &duplicateResponse)
	if duplicateResponse.Status != "accepted" || duplicateResponse.Skipped != 1 || len(duplicateResponse.Items) != 1 {
		t.Fatalf("unexpected duplicate ingest response: %+v", duplicateResponse)
	}
	if duplicateResponse.Items[0].Status != "skipped" || duplicateResponse.Items[0].SessionID != "ingest-session" {
		t.Fatalf("unexpected duplicate ingest item response: %+v", duplicateResponse.Items[0])
	}
}

type filterOption struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Detail string `json:"detail"`
	Count  int64  `json:"count"`
}

type routeClient struct {
	handler http.Handler
	cookie  *http.Cookie
}

func newRouteClient(handler http.Handler) *routeClient {
	return &routeClient{handler: handler}
}

func (c *routeClient) login(t *testing.T) {
	t.Helper()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"password":"admin-password"}`))
	c.handler.ServeHTTP(recorder, request)
	assertStatus(t, recorder, http.StatusOK)
	cookies := recorder.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatalf("expected login cookie")
	}
	c.cookie = cookies[0]
}

func (c *routeClient) get(t *testing.T, target string) *httptest.ResponseRecorder {
	t.Helper()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, target, nil)
	if c.cookie != nil {
		request.AddCookie(c.cookie)
	}
	c.handler.ServeHTTP(recorder, request)
	return recorder
}

func (c *routeClient) getJSON(t *testing.T, target string, out any) {
	t.Helper()

	recorder := c.get(t, target)
	assertStatus(t, recorder, http.StatusOK)
	decodeRecorderJSON(t, recorder, out)
}

func decodeRecorderJSON(t *testing.T, recorder *httptest.ResponseRecorder, out any) {
	t.Helper()

	body, err := io.ReadAll(recorder.Result().Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(out); err != nil {
		t.Fatalf("decode json: %v\nbody: %s", err, string(body))
	}
}

func assertStatus(t *testing.T, recorder *httptest.ResponseRecorder, want int) {
	t.Helper()

	if recorder.Code != want {
		t.Fatalf("unexpected status: got %d want %d body=%s", recorder.Code, want, recorder.Body.String())
	}
}

func newTestHandler(t *testing.T, db *pgxpool.Pool) http.Handler {
	t.Helper()

	return newTestHandlerWithConfig(t, db, config.Config{})
}

func newTestHandlerWithConfig(t *testing.T, db *pgxpool.Pool, overrides config.Config) http.Handler {
	t.Helper()

	rawDir := overrides.RawDir
	if rawDir == "" {
		rawDir = t.TempDir()
	}
	maxUploadBytes := overrides.MaxUploadBytes
	if maxUploadBytes == 0 {
		maxUploadBytes = 1 << 20
	}
	deviceTokens := overrides.DeviceTokens
	if deviceTokens == nil {
		deviceTokens = map[string]struct{}{"device-token": {}}
	}

	srv := &Server{
		cfg: config.Config{
			Environment:    overrides.Environment,
			AdminPassword:  "admin-password",
			SessionSecret:  []byte("test-session-secret-with-enough-bytes"),
			SessionTTL:     time.Hour,
			DeviceTokens:   deviceTokens,
			CookieSecure:   false,
			RawDir:         rawDir,
			MaxUploadBytes: maxUploadBytes,
		},
		db:       db,
		sessions: auth.NewSessionManager([]byte("test-session-secret-with-enough-bytes"), time.Hour, false),
	}
	return srv.routes()
}

func newIngestRequest(t *testing.T, metadata map[string]any, raw []byte, token string) *http.Request {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	metadata["session"].(map[string]any)["raw_field"] = "raw_0"
	metadataBytes, err := json.Marshal([]map[string]any{metadata})
	if err != nil {
		t.Fatalf("marshal ingest metadata: %v", err)
	}
	if err := writer.WriteField("metadata", string(metadataBytes)); err != nil {
		t.Fatalf("write ingest metadata: %v", err)
	}
	fileWriter, err := writer.CreateFormFile("raw_0", "session.jsonl")
	if err != nil {
		t.Fatalf("create raw form file: %v", err)
	}
	if _, err := fileWriter.Write(raw); err != nil {
		t.Fatalf("write raw form file: %v", err)
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
