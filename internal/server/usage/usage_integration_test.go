package usage_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ilcm96/codex-usage/internal/server/servertest"
	"github.com/ilcm96/codex-usage/internal/server/usage"
)

func TestController_RegisterAndUsageEndpoints(t *testing.T) {
	ctx := context.Background()
	db := servertest.StartPostgres(t)
	servertest.Reset(t, ctx, db)
	fixture := servertest.SeedFixture(t, ctx, db)

	mux := http.NewServeMux()
	repository := usage.NewPostgresRepository(db)
	service := usage.NewService(repository)
	controller := usage.NewController(service)
	controller.Register(mux, func(next http.Handler) http.Handler {
		return next
	})

	t.Run("totals returns aggregate usage and entity counts", func(t *testing.T) {
		var got struct {
			TotalTokens     int64   `json:"totalTokens"`
			InputTokens     int64   `json:"inputTokens"`
			OutputTokens    int64   `json:"outputTokens"`
			CostUSD         float64 `json:"costUsd"`
			Sessions        int64   `json:"sessions"`
			Projects        int64   `json:"projects"`
			Devices         int64   `json:"devices"`
			PatchAddedLines int64   `json:"patchAddedLines"`
		}
		getJSON(t, mux, "/api/usage/totals", &got)
		if got.TotalTokens != 380 || got.InputTokens != 130 || got.OutputTokens != 190 || got.Sessions != 2 || got.Projects != 1 || got.Devices != 1 || got.PatchAddedLines != 4 {
			t.Fatalf("unexpected totals: %+v", got)
		}
		if got.CostUSD != 0.04 {
			t.Fatalf("unexpected totals cost: %+v", got)
		}
	})

	t.Run("windows returns fixed recent usage windows", func(t *testing.T) {
		var got []struct {
			Label  string `json:"label"`
			Days   int    `json:"days"`
			Totals struct {
				TotalTokens     int64 `json:"totalTokens"`
				Sessions        int64 `json:"sessions"`
				PatchAddedLines int64 `json:"patchAddedLines"`
			} `json:"totals"`
		}
		getJSON(t, mux, "/api/usage/windows", &got)
		if len(got) != 3 {
			t.Fatalf("unexpected windows: %+v", got)
		}
		if got[0].Days != 1 || got[0].Totals.TotalTokens != 300 || got[0].Totals.Sessions != 1 || got[0].Totals.PatchAddedLines != 4 {
			t.Fatalf("unexpected today window: %+v", got[0])
		}
		if got[1].Days != 7 || got[1].Totals.TotalTokens != 380 || got[1].Totals.Sessions != 2 || got[1].Totals.PatchAddedLines != 4 {
			t.Fatalf("unexpected last7 window: %+v", got[1])
		}
	})

	t.Run("usage groups rollups by day", func(t *testing.T) {
		var got []struct {
			Bucket                string  `json:"bucket"`
			InputTokens           int64   `json:"inputTokens"`
			CachedInputTokens     int64   `json:"cachedInputTokens"`
			OutputTokens          int64   `json:"outputTokens"`
			ReasoningOutputTokens int64   `json:"reasoningOutputTokens"`
			TotalTokens           int64   `json:"totalTokens"`
			CostUSD               float64 `json:"costUsd"`
			PatchAddedLines       int64   `json:"patchAddedLines"`
		}
		getJSON(t, mux, "/api/usage/series?bucket=day", &got)
		if len(got) != 2 || got[0].TotalTokens != 80 || got[0].PatchAddedLines != 0 || got[1].TotalTokens != 300 || got[1].PatchAddedLines != 4 {
			t.Fatalf("unexpected usage buckets: %+v", got)
		}
	})

	t.Run("usage filters by repository and model", func(t *testing.T) {
		var got []struct {
			TotalTokens     int64 `json:"totalTokens"`
			PatchAddedLines int64 `json:"patchAddedLines"`
		}
		getJSON(t, mux, "/api/usage/series?repositoryId="+fixture.RepositoryID+"&model=gpt-5-mini", &got)
		if len(got) != 1 || got[0].TotalTokens != 50 || got[0].PatchAddedLines != 4 {
			t.Fatalf("unexpected filtered usage: %+v", got)
		}
	})

	t.Run("usage returns empty result for invalid id filter", func(t *testing.T) {
		var got []struct {
			TotalTokens int64 `json:"totalTokens"`
		}
		getJSON(t, mux, "/api/usage/series?repositoryId=not-a-uuid", &got)
		if len(got) != 0 {
			t.Fatalf("unexpected invalid id usage: %+v", got)
		}
	})

	t.Run("usage rejects unsupported bucket", func(t *testing.T) {
		recorder := get(t, mux, "/api/usage/series?bucket=year")
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("unexpected status: got %d body=%s", recorder.Code, recorder.Body.String())
		}
	})

	t.Run("breakdown defaults to model", func(t *testing.T) {
		var got []struct {
			ID          string  `json:"id"`
			Label       string  `json:"label"`
			Detail      string  `json:"detail"`
			Sessions    int64   `json:"sessions"`
			TotalTokens int64   `json:"totalTokens"`
			CostUSD     float64 `json:"costUsd"`
		}
		getJSON(t, mux, "/api/usage/breakdown", &got)
		if len(got) != 2 || got[0].Label != "gpt-5" || got[0].Sessions != 2 || got[0].TotalTokens != 330 {
			t.Fatalf("unexpected model breakdown: %+v", got)
		}
	})

	t.Run("breakdown groups by repository", func(t *testing.T) {
		var got []struct {
			Label       string `json:"label"`
			Detail      string `json:"detail"`
			Sessions    int64  `json:"sessions"`
			TotalTokens int64  `json:"totalTokens"`
		}
		getJSON(t, mux, "/api/usage/breakdown?groupBy=repository", &got)
		if len(got) != 1 || got[0].Label != "codex-usage" || got[0].Sessions != 2 || got[0].TotalTokens != 380 {
			t.Fatalf("unexpected repository breakdown: %+v", got)
		}
	})

	t.Run("breakdown rejects unsupported group", func(t *testing.T) {
		recorder := get(t, mux, "/api/usage/breakdown?groupBy=branch")
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("unexpected status: got %d body=%s", recorder.Code, recorder.Body.String())
		}
	})

	t.Run("summary returns current totals and derived metrics", func(t *testing.T) {
		var got struct {
			Current struct {
				InputTokens       int64   `json:"inputTokens"`
				CachedInputTokens int64   `json:"cachedInputTokens"`
				TotalTokens       int64   `json:"totalTokens"`
				CostUSD           float64 `json:"costUsd"`
				Sessions          int64   `json:"sessions"`
				Messages          int64   `json:"messages"`
				ToolCalls         int64   `json:"toolCalls"`
				PatchAddedLines   int64   `json:"patchAddedLines"`
			} `json:"current"`
			ActiveDays     int64   `json:"activeDays"`
			CacheHitRate   float64 `json:"cacheHitRate"`
			AvgSessionCost float64 `json:"avgSessionCost"`
		}
		recorder := get(t, mux, "/api/usage/summary")
		if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if got.Current.TotalTokens != 380 || got.Current.Sessions != 2 || got.Current.Messages != 3 || got.Current.ToolCalls != 1 || got.Current.PatchAddedLines != 4 {
			t.Fatalf("unexpected summary: %+v", got)
		}
		if got.ActiveDays != 2 || got.CacheHitRate <= 0 || got.AvgSessionCost <= 0 {
			t.Fatalf("unexpected derived summary metrics: %+v", got)
		}
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(recorder.Body.Bytes(), &raw); err != nil {
			t.Fatalf("failed to decode raw response: %v", err)
		}
		if _, ok := raw["previous"]; ok {
			t.Fatalf("summary should not include previous comparison: %s", recorder.Body.String())
		}
	})

	t.Run("calendar returns recent rollup days", func(t *testing.T) {
		var got []struct {
			Date        string  `json:"date"`
			TotalTokens int64   `json:"totalTokens"`
			CostUSD     float64 `json:"costUsd"`
			Projects    int64   `json:"projects"`
		}
		getJSON(t, mux, "/api/usage/calendar?days=2", &got)
		if len(got) != 2 || got[0].TotalTokens != 80 || got[1].TotalTokens != 300 || got[1].Projects != 1 {
			t.Fatalf("unexpected calendar: %+v", got)
		}
	})

}

func get(t *testing.T, handler http.Handler, target string) *httptest.ResponseRecorder {
	t.Helper()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, target, nil)
	handler.ServeHTTP(recorder, request)
	return recorder
}

func getJSON(t *testing.T, handler http.Handler, target string, out any) {
	t.Helper()

	recorder := get(t, handler, target)
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d body=%s", recorder.Code, recorder.Body.String())
	}
	if err := json.NewDecoder(recorder.Body).Decode(out); err != nil {
		t.Fatalf("decode json: %v\nbody=%s", err, recorder.Body.String())
	}
}
