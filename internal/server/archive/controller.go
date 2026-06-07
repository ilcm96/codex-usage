package archive

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
	mux.Handle("GET /api/archive/status", protect(http.HandlerFunc(c.handleStatus)))
	mux.Handle("GET /api/archive/health", protect(http.HandlerFunc(c.handleHealth)))
	mux.Handle("GET /api/archive/devices", protect(http.HandlerFunc(c.handleDevices)))
	mux.Handle("GET /api/archive/repositories", protect(http.HandlerFunc(c.handleRepositories)))
	mux.Handle("GET /api/archive/integrity", protect(http.HandlerFunc(c.handleIntegrity)))
}

// handleStatus returns archive storage and integrity counters.
// @Summary Archive status
// @Tags Archive
// @Produce json
// @Security CookieAuth
// @Success 200 {object} Status
// @Failure 401 {object} httpapi.ErrorResponse
// @Failure 500 {object} httpapi.ErrorResponse
// @Router /api/archive/status [get]
func (c Controller) handleStatus(w http.ResponseWriter, r *http.Request) {
	out, err := c.service.Status(r.Context())
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "failed to load archive status")
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, out)
}

// handleHealth returns archive health and verification counters.
// @Summary Archive health
// @Tags Archive
// @Produce json
// @Security CookieAuth
// @Success 200 {object} Health
// @Failure 401 {object} httpapi.ErrorResponse
// @Failure 500 {object} httpapi.ErrorResponse
// @Router /api/archive/health [get]
func (c Controller) handleHealth(w http.ResponseWriter, r *http.Request) {
	out, err := c.service.Health(r.Context())
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "failed to load archive health")
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, out)
}

// handleDevices returns archive storage grouped by device.
// @Summary Archive devices
// @Tags Archive
// @Produce json
// @Security CookieAuth
// @Success 200 {array} DeviceSummary
// @Failure 401 {object} httpapi.ErrorResponse
// @Failure 500 {object} httpapi.ErrorResponse
// @Router /api/archive/devices [get]
func (c Controller) handleDevices(w http.ResponseWriter, r *http.Request) {
	out, err := c.service.ByDevice(r.Context())
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "failed to load archive devices")
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, out)
}

// handleRepositories returns archive storage grouped by repository.
// @Summary Archive repositories
// @Tags Archive
// @Produce json
// @Security CookieAuth
// @Param limit query int false "Maximum number of repositories"
// @Success 200 {array} RepositorySummary
// @Failure 401 {object} httpapi.ErrorResponse
// @Failure 500 {object} httpapi.ErrorResponse
// @Router /api/archive/repositories [get]
func (c Controller) handleRepositories(w http.ResponseWriter, r *http.Request) {
	out, err := c.service.ByRepository(r.Context(), httpapi.QueryInt(r, "limit", 20))
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "failed to load archive repositories")
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, out)
}

// handleIntegrity returns archive integrity check results.
// @Summary Archive integrity
// @Tags Archive
// @Produce json
// @Security CookieAuth
// @Success 200 {object} IntegrityResult
// @Failure 401 {object} httpapi.ErrorResponse
// @Failure 500 {object} httpapi.ErrorResponse
// @Router /api/archive/integrity [get]
func (c Controller) handleIntegrity(w http.ResponseWriter, r *http.Request) {
	out, err := c.service.Integrity(r.Context())
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "failed to load archive integrity")
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, out)
}
