package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteJSONAndWriteError(t *testing.T) {
	recorder := httptest.NewRecorder()
	WriteJSON(recorder, http.StatusCreated, map[string]string{"status": "ok"})
	if recorder.Code != http.StatusCreated {
		t.Fatalf("unexpected status: %d", recorder.Code)
	}
	if recorder.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("unexpected content type: %q", recorder.Header().Get("Content-Type"))
	}
	var body map[string]string
	if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("unexpected json body: %+v", body)
	}

	errorRecorder := httptest.NewRecorder()
	WriteError(errorRecorder, http.StatusBadRequest, "bad request")
	if errorRecorder.Code != http.StatusBadRequest {
		t.Fatalf("unexpected error status: %d", errorRecorder.Code)
	}
	body = map[string]string{}
	if err := json.NewDecoder(errorRecorder.Body).Decode(&body); err != nil {
		t.Fatalf("decode error json: %v", err)
	}
	if body["error"] != "bad request" {
		t.Fatalf("unexpected error body: %+v", body)
	}
}

func TestQueryInt(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/?limit=25&zero=0&bad=x", nil)
	if got := QueryInt(request, "limit", 10); got != 25 {
		t.Fatalf("QueryInt limit = %d", got)
	}
	if got := QueryInt(request, "zero", 10); got != 10 {
		t.Fatalf("QueryInt zero = %d", got)
	}
	if got := QueryInt(request, "bad", 10); got != 10 {
		t.Fatalf("QueryInt bad = %d", got)
	}
	if got := QueryInt(request, "missing", 10); got != 10 {
		t.Fatalf("QueryInt missing = %d", got)
	}
}

func TestValidDateParam(t *testing.T) {
	if got := ValidDateParam(" 2026-02-08 "); got != "2026-02-08" {
		t.Fatalf("ValidDateParam valid = %q", got)
	}
	if got := ValidDateParam("2026-02-30"); got != "" {
		t.Fatalf("ValidDateParam invalid = %q", got)
	}
	if got := ValidDateParam(""); got != "" {
		t.Fatalf("ValidDateParam empty = %q", got)
	}
}
