package filteroptions_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ilcm96/codex-usage/internal/server/filteroptions"
	"github.com/ilcm96/codex-usage/internal/server/servertest"
)

func TestController_RegisterAndListFilterOptions(t *testing.T) {
	ctx := context.Background()
	db := servertest.StartPostgres(t)
	servertest.Reset(t, ctx, db)
	servertest.SeedFixture(t, ctx, db)

	repository := filteroptions.NewPostgresRepository(db)
	service := filteroptions.NewService(repository)
	controller := filteroptions.NewController(service)
	mux := http.NewServeMux()
	controller.Register(mux, func(next http.Handler) http.Handler {
		return next
	})

	var got struct {
		DateRange struct {
			Oldest string `json:"oldest"`
			Newest string `json:"newest"`
		} `json:"dateRange"`
		Devices      []filterOption `json:"devices"`
		Repositories []filterOption `json:"repositories"`
		Projects     []filterOption `json:"projects"`
		Models       []filterOption `json:"models"`
		Branches     []filterOption `json:"branches"`
	}
	getJSON(t, mux, "/api/filter-options", &got)
	if got.DateRange.Oldest == "" || got.DateRange.Newest == "" {
		t.Fatalf("unexpected date range: %+v", got.DateRange)
	}
	if len(got.Devices) != 1 || got.Devices[0].Count != 2 {
		t.Fatalf("unexpected device options: %+v", got.Devices)
	}
	if len(got.Repositories) != 1 || got.Repositories[0].Label != "codex-usage" {
		t.Fatalf("unexpected repository options: %+v", got.Repositories)
	}
	if len(got.Projects) != 1 || got.Projects[0].Label != "codex-usage" {
		t.Fatalf("unexpected project options: %+v", got.Projects)
	}
	if len(got.Models) != 2 || got.Models[0].Label != "gpt-5" {
		t.Fatalf("unexpected model options: %+v", got.Models)
	}
	if len(got.Branches) != 2 {
		t.Fatalf("unexpected branch options: %+v", got.Branches)
	}
}

type filterOption struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Detail string `json:"detail"`
	Count  int64  `json:"count"`
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
