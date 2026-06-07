package usage

import (
	"errors"
	"net/http"
	"strings"

	"github.com/ilcm96/codex-usage/internal/server/httpapi"
)

// handleTotals returns global usage totals and entity counts.
// @Summary Global usage totals
// @Tags Usage
// @Produce json
// @Security CookieAuth
// @Success 200 {object} GlobalTotals
// @Failure 401 {object} httpapi.ErrorResponse
// @Failure 500 {object} httpapi.ErrorResponse
// @Router /api/usage/totals [get]
func (c Controller) handleTotals(w http.ResponseWriter, r *http.Request) {
	out, err := c.service.Totals(r.Context())
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "failed to load usage totals")
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, out)
}

// handleWindows returns fixed recent usage windows.
// @Summary Recent usage windows
// @Tags Usage
// @Produce json
// @Security CookieAuth
// @Success 200 {array} Window
// @Failure 401 {object} httpapi.ErrorResponse
// @Failure 500 {object} httpapi.ErrorResponse
// @Router /api/usage/windows [get]
func (c Controller) handleWindows(w http.ResponseWriter, r *http.Request) {
	out, err := c.service.Windows(r.Context())
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "failed to load usage windows")
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, out)
}

// handleSeries returns usage buckets grouped by day, week, or month.
// @Summary Usage series
// @Tags Usage
// @Produce json
// @Security CookieAuth
// @Param bucket query string false "Bucket size: day, week, or month"
// @Param from query string false "Start date in YYYY-MM-DD format"
// @Param to query string false "End date in YYYY-MM-DD format"
// @Param deviceId query string false "Device ID filter"
// @Param repositoryId query string false "Repository ID filter"
// @Param projectId query string false "Project ID filter"
// @Param model query string false "Model filter"
// @Success 200 {array} SeriesBucket
// @Failure 400 {object} httpapi.ErrorResponse
// @Failure 401 {object} httpapi.ErrorResponse
// @Failure 500 {object} httpapi.ErrorResponse
// @Router /api/usage/series [get]
func (c Controller) handleSeries(w http.ResponseWriter, r *http.Request) {
	out, err := c.service.Series(r.Context(), SeriesParams{
		Bucket:  strings.TrimSpace(r.URL.Query().Get("bucket")),
		Filters: filtersFromRequest(r),
	})
	if errors.Is(err, ErrUnsupportedBucket) {
		httpapi.WriteError(w, http.StatusBadRequest, "unsupported bucket")
		return
	}
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "failed to load usage series")
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, out)
}

// handleBreakdown returns usage totals grouped by an entity.
// @Summary Usage breakdown
// @Tags Usage
// @Produce json
// @Security CookieAuth
// @Param groupBy query string false "Grouping key: model, repository, project, device, or language"
// @Param limit query int false "Maximum number of rows"
// @Param from query string false "Start date in YYYY-MM-DD format"
// @Param to query string false "End date in YYYY-MM-DD format"
// @Param deviceId query string false "Device ID filter"
// @Param repositoryId query string false "Repository ID filter"
// @Param projectId query string false "Project ID filter"
// @Param model query string false "Model filter"
// @Success 200 {array} BreakdownItem
// @Failure 400 {object} httpapi.ErrorResponse
// @Failure 401 {object} httpapi.ErrorResponse
// @Failure 500 {object} httpapi.ErrorResponse
// @Router /api/usage/breakdown [get]
func (c Controller) handleBreakdown(w http.ResponseWriter, r *http.Request) {
	out, err := c.service.Breakdown(r.Context(), BreakdownParams{
		GroupBy: strings.TrimSpace(r.URL.Query().Get("groupBy")),
		Limit:   httpapi.QueryInt(r, "limit", 12),
		Filters: filtersFromRequest(r),
	})
	if errors.Is(err, ErrUnsupportedGroup) {
		httpapi.WriteError(w, http.StatusBadRequest, "unsupported groupBy")
		return
	}
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "failed to load usage breakdown")
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, out)
}

// handleSummary returns usage summary for selected filters.
// @Summary Usage summary
// @Tags Usage
// @Produce json
// @Security CookieAuth
// @Param from query string false "Start date in YYYY-MM-DD format"
// @Param to query string false "End date in YYYY-MM-DD format"
// @Param deviceId query string false "Device ID filter"
// @Param repositoryId query string false "Repository ID filter"
// @Param projectId query string false "Project ID filter"
// @Param model query string false "Model filter"
// @Success 200 {object} Summary
// @Failure 401 {object} httpapi.ErrorResponse
// @Failure 500 {object} httpapi.ErrorResponse
// @Router /api/usage/summary [get]
func (c Controller) handleSummary(w http.ResponseWriter, r *http.Request) {
	out, err := c.service.Summary(r.Context(), filtersFromRequest(r))
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "failed to load usage summary")
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, out)
}

// handleCalendar returns daily token and cost values for calendar visualizations.
// @Summary Usage calendar
// @Tags Usage
// @Produce json
// @Security CookieAuth
// @Param days query int false "Number of days to include"
// @Param from query string false "Start date in YYYY-MM-DD format"
// @Param to query string false "End date in YYYY-MM-DD format"
// @Param deviceId query string false "Device ID filter"
// @Param repositoryId query string false "Repository ID filter"
// @Param projectId query string false "Project ID filter"
// @Param model query string false "Model filter"
// @Success 200 {array} CalendarDay
// @Failure 401 {object} httpapi.ErrorResponse
// @Failure 500 {object} httpapi.ErrorResponse
// @Router /api/usage/calendar [get]
func (c Controller) handleCalendar(w http.ResponseWriter, r *http.Request) {
	out, err := c.service.Calendar(r.Context(), CalendarParams{
		Days:    httpapi.QueryInt(r, "days", 120),
		Filters: filtersFromRequest(r),
	})
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "failed to load usage calendar")
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, out)
}

func filtersFromRequest(r *http.Request) Filters {
	return Filters{
		From:         httpapi.ValidDateParam(r.URL.Query().Get("from")),
		To:           httpapi.ValidDateParam(r.URL.Query().Get("to")),
		DeviceID:     strings.TrimSpace(r.URL.Query().Get("deviceId")),
		RepositoryID: strings.TrimSpace(r.URL.Query().Get("repositoryId")),
		ProjectID:    strings.TrimSpace(r.URL.Query().Get("projectId")),
		Model:        strings.TrimSpace(r.URL.Query().Get("model")),
	}
}
