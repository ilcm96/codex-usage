package usage

import "net/http"

type Controller struct {
	service Service
}

func NewController(service Service) Controller {
	return Controller{service: service}
}

func (c Controller) Register(mux *http.ServeMux, protect func(http.Handler) http.Handler) {
	mux.Handle("GET /api/usage/totals", protect(http.HandlerFunc(c.handleTotals)))
	mux.Handle("GET /api/usage/windows", protect(http.HandlerFunc(c.handleWindows)))
	mux.Handle("GET /api/usage/series", protect(http.HandlerFunc(c.handleSeries)))
	mux.Handle("GET /api/usage/breakdown", protect(http.HandlerFunc(c.handleBreakdown)))
	mux.Handle("GET /api/usage/summary", protect(http.HandlerFunc(c.handleSummary)))
	mux.Handle("GET /api/usage/calendar", protect(http.HandlerFunc(c.handleCalendar)))
}
