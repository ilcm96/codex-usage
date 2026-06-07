package projects

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
	mux.Handle("GET /api/repositories", protect(http.HandlerFunc(c.handleRepositories)))
	mux.Handle("GET /api/projects", protect(http.HandlerFunc(c.handleProjects)))
}

// handleRepositories returns repository usage summaries.
// @Summary List repositories
// @Tags Projects
// @Produce json
// @Security CookieAuth
// @Success 200 {array} RepositorySummary
// @Failure 401 {object} httpapi.ErrorResponse
// @Failure 500 {object} httpapi.ErrorResponse
// @Router /api/repositories [get]
func (c Controller) handleRepositories(w http.ResponseWriter, r *http.Request) {
	out, err := c.service.ListRepositories(r.Context())
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "failed to load repositories")
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, out)
}

// handleProjects returns project usage summaries.
// @Summary List projects
// @Tags Projects
// @Produce json
// @Security CookieAuth
// @Success 200 {array} ProjectSummary
// @Failure 401 {object} httpapi.ErrorResponse
// @Failure 500 {object} httpapi.ErrorResponse
// @Router /api/projects [get]
func (c Controller) handleProjects(w http.ResponseWriter, r *http.Request) {
	out, err := c.service.ListProjects(r.Context())
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "failed to load projects")
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, out)
}
