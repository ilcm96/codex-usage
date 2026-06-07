package filteroptions

import (
	"net/http"

	"github.com/ilcm96/codex-usage/internal/server/httpapi"
)

type Controller struct {
	service Service
}

func NewController(service Service) Controller {
	return Controller{service: service}
}

func (c Controller) Register(mux *http.ServeMux, protect func(http.Handler) http.Handler) {
	mux.Handle("GET /api/filter-options", protect(http.HandlerFunc(c.handleList)))
}

// handleList returns available filter option values.
// @Summary Filter options
// @Tags Filter Options
// @Produce json
// @Security CookieAuth
// @Success 200 {object} Result
// @Failure 401 {object} httpapi.ErrorResponse
// @Failure 500 {object} httpapi.ErrorResponse
// @Router /api/filter-options [get]
func (c Controller) handleList(w http.ResponseWriter, r *http.Request) {
	out, err := c.service.List(r.Context())
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "failed to load filter options")
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, out)
}
