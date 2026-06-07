package search

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
	mux.Handle("GET /api/search", protect(http.HandlerFunc(c.handleSearch)))
}

// handleSearch searches indexed messages and tool events.
// @Summary Search
// @Tags Search
// @Produce json
// @Security CookieAuth
// @Param q query string false "Text search query"
// @Param limit query int false "Maximum number of rows"
// @Param offset query int false "Pagination offset"
// @Param includeTotal query bool false "Set false to skip total count"
// @Param from query string false "Start date in YYYY-MM-DD format"
// @Param to query string false "End date in YYYY-MM-DD format"
// @Param deviceId query string false "Device ID filter"
// @Param repositoryId query string false "Repository ID filter"
// @Param projectId query string false "Project ID filter"
// @Param kind query string false "Search kind filter"
// @Param model query string false "Model filter"
// @Success 200 {object} SearchResult
// @Failure 400 {object} httpapi.ErrorResponse
// @Failure 401 {object} httpapi.ErrorResponse
// @Failure 500 {object} httpapi.ErrorResponse
// @Router /api/search [get]
func (c Controller) handleSearch(w http.ResponseWriter, r *http.Request) {
	result, err := c.service.Search(r.Context(), Params{
		Query:        strings.TrimSpace(r.URL.Query().Get("q")),
		Limit:        httpapi.QueryInt(r, "limit", 50),
		Offset:       httpapi.QueryInt(r, "offset", 0),
		IncludeTotal: r.URL.Query().Get("includeTotal") != "false",
		From:         httpapi.ValidDateParam(r.URL.Query().Get("from")),
		To:           httpapi.ValidDateParam(r.URL.Query().Get("to")),
		DeviceID:     strings.TrimSpace(r.URL.Query().Get("deviceId")),
		RepositoryID: strings.TrimSpace(r.URL.Query().Get("repositoryId")),
		ProjectID:    strings.TrimSpace(r.URL.Query().Get("projectId")),
		Kind:         strings.TrimSpace(r.URL.Query().Get("kind")),
		Model:        strings.TrimSpace(r.URL.Query().Get("model")),
	})
	if errors.Is(err, ErrUnsupportedKind) {
		httpapi.WriteError(w, http.StatusBadRequest, "unsupported kind")
		return
	}
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "failed to search")
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, result)
}
