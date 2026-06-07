package sessions

import (
	"errors"
	"net/http"
	"strings"

	"github.com/ilcm96/codex-usage/internal/server/httpapi"
)

type Controller struct {
	service Service
}

func NewController(service Service) Controller {
	return Controller{service: service}
}

func (c Controller) Register(mux *http.ServeMux, protect func(http.Handler) http.Handler) {
	mux.Handle("GET /api/sessions", protect(http.HandlerFunc(c.handleSessionsList)))
	mux.Handle("GET /api/sessions/compact", protect(http.HandlerFunc(c.handleCompactSessions)))
	mux.Handle("GET /api/sessions/{id}", protect(http.HandlerFunc(c.handleSessionDetail)))
	mux.Handle("GET /api/sessions/{id}/reader", protect(http.HandlerFunc(c.handleSessionReader)))
	mux.Handle("GET /api/sessions/{id}/timeline", protect(http.HandlerFunc(c.handleSessionTimeline)))
	mux.Handle("GET /api/exports/sessions", protect(http.HandlerFunc(c.handleExportSessions)))
	mux.Handle("GET /api/sessions/{id}/export", protect(http.HandlerFunc(c.handleExportSession)))
}

// handleCompactSessions returns a compact session list.
// @Summary List compact sessions
// @Tags Sessions
// @Produce json
// @Security CookieAuth
// @Param limit query int false "Maximum number of rows"
// @Param offset query int false "Pagination offset"
// @Param sort query string false "Sort key"
// @Param from query string false "Start date in YYYY-MM-DD format"
// @Param to query string false "End date in YYYY-MM-DD format"
// @Param deviceId query string false "Device ID filter"
// @Param repositoryId query string false "Repository ID filter"
// @Param projectId query string false "Project ID filter"
// @Param branch query string false "Branch filter"
// @Param model query string false "Model filter"
// @Param q query string false "Text search query"
// @Success 200 {array} SimpleSession
// @Failure 400 {object} httpapi.ErrorResponse
// @Failure 401 {object} httpapi.ErrorResponse
// @Failure 500 {object} httpapi.ErrorResponse
// @Router /api/sessions/compact [get]
func (c Controller) handleCompactSessions(w http.ResponseWriter, r *http.Request) {
	items, err := c.service.ListSimple(r.Context(), listParamsFromRequest(r))
	if errors.Is(err, ErrUnsupportedSort) {
		httpapi.WriteError(w, http.StatusBadRequest, "unsupported sort")
		return
	}
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "failed to load sessions")
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, items)
}

// handleSessionsList returns paginated sessions with aggregate totals.
// @Summary List sessions with totals
// @Tags Sessions
// @Produce json
// @Security CookieAuth
// @Param limit query int false "Maximum number of rows"
// @Param offset query int false "Pagination offset"
// @Param sort query string false "Sort key"
// @Param from query string false "Start date in YYYY-MM-DD format"
// @Param to query string false "End date in YYYY-MM-DD format"
// @Param deviceId query string false "Device ID filter"
// @Param repositoryId query string false "Repository ID filter"
// @Param projectId query string false "Project ID filter"
// @Param branch query string false "Branch filter"
// @Param model query string false "Model filter"
// @Param q query string false "Text search query"
// @Success 200 {object} ListResult
// @Failure 400 {object} httpapi.ErrorResponse
// @Failure 401 {object} httpapi.ErrorResponse
// @Failure 500 {object} httpapi.ErrorResponse
// @Router /api/sessions [get]
func (c Controller) handleSessionsList(w http.ResponseWriter, r *http.Request) {
	result, err := c.service.List(r.Context(), listParamsFromRequest(r))
	if errors.Is(err, ErrUnsupportedSort) {
		httpapi.WriteError(w, http.StatusBadRequest, "unsupported sort")
		return
	}
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "failed to load session list")
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, result)
}

// handleSessionDetail returns detail for one session.
// @Summary Session detail
// @Tags Sessions
// @Produce json
// @Security CookieAuth
// @Param id path string true "Session ID"
// @Success 200 {object} DetailResult
// @Failure 401 {object} httpapi.ErrorResponse
// @Failure 404 {object} httpapi.ErrorResponse
// @Failure 500 {object} httpapi.ErrorResponse
// @Router /api/sessions/{id} [get]
func (c Controller) handleSessionDetail(w http.ResponseWriter, r *http.Request) {
	result, err := c.service.Detail(r.Context(), r.PathValue("id"))
	if errors.Is(err, ErrNotFound) {
		httpapi.WriteError(w, http.StatusNotFound, "session not found")
		return
	}
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "failed to load session")
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, result)
}

// handleSessionReader returns conversation turns for reader mode.
// @Summary Session reader
// @Tags Sessions
// @Produce json
// @Security CookieAuth
// @Param id path string true "Session ID"
// @Param limit query int false "Maximum number of turns"
// @Param offset query int false "Pagination offset"
// @Param q query string false "Text search query"
// @Success 200 {object} ReaderResult
// @Failure 401 {object} httpapi.ErrorResponse
// @Failure 404 {object} httpapi.ErrorResponse
// @Failure 500 {object} httpapi.ErrorResponse
// @Router /api/sessions/{id}/reader [get]
func (c Controller) handleSessionReader(w http.ResponseWriter, r *http.Request) {
	result, err := c.service.Reader(r.Context(), r.PathValue("id"), ReaderParams{
		Limit:  httpapi.QueryInt(r, "limit", 30),
		Offset: httpapi.QueryInt(r, "offset", 0),
		Query:  strings.TrimSpace(r.URL.Query().Get("q")),
	})
	if errors.Is(err, ErrNotFound) {
		httpapi.WriteError(w, http.StatusNotFound, "session not found")
		return
	}
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "failed to load reader")
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, result)
}

// handleSessionTimeline returns raw timeline events.
// @Summary Session timeline
// @Tags Sessions
// @Produce json
// @Security CookieAuth
// @Param id path string true "Session ID"
// @Param limit query int false "Maximum number of timeline rows"
// @Param offset query int false "Pagination offset"
// @Param kind query string false "Timeline kind filter"
// @Param q query string false "Text search query"
// @Success 200 {object} TimelineResult
// @Failure 400 {object} httpapi.ErrorResponse
// @Failure 401 {object} httpapi.ErrorResponse
// @Failure 500 {object} httpapi.ErrorResponse
// @Router /api/sessions/{id}/timeline [get]
func (c Controller) handleSessionTimeline(w http.ResponseWriter, r *http.Request) {
	result, err := c.service.Timeline(r.Context(), r.PathValue("id"), TimelineParams{
		Limit:  httpapi.QueryInt(r, "limit", 100),
		Offset: httpapi.QueryInt(r, "offset", 0),
		Kind:   strings.TrimSpace(r.URL.Query().Get("kind")),
		Query:  strings.TrimSpace(r.URL.Query().Get("q")),
	})
	if errors.Is(err, ErrUnsupportedKind) {
		httpapi.WriteError(w, http.StatusBadRequest, "unsupported kind")
		return
	}
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "failed to load timeline")
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, result)
}

func listParamsFromRequest(r *http.Request) ListParams {
	return ListParams{
		Limit:        httpapi.QueryInt(r, "limit", 50),
		Offset:       httpapi.QueryInt(r, "offset", 0),
		Sort:         r.URL.Query().Get("sort"),
		From:         httpapi.ValidDateParam(r.URL.Query().Get("from")),
		To:           httpapi.ValidDateParam(r.URL.Query().Get("to")),
		DeviceID:     strings.TrimSpace(r.URL.Query().Get("deviceId")),
		RepositoryID: strings.TrimSpace(r.URL.Query().Get("repositoryId")),
		ProjectID:    strings.TrimSpace(r.URL.Query().Get("projectId")),
		Branch:       strings.TrimSpace(r.URL.Query().Get("branch")),
		Model:        strings.TrimSpace(r.URL.Query().Get("model")),
		Query:        strings.TrimSpace(r.URL.Query().Get("q")),
	}
}
