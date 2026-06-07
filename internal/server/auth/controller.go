package auth

import (
	"encoding/json"
	"net/http"

	"github.com/ilcm96/codex-usage/internal/server/httpapi"
)

type Controller struct {
	adminPassword string
	sessions      SessionManager
}

func NewController(adminPassword string, sessions SessionManager) Controller {
	return Controller{
		adminPassword: adminPassword,
		sessions:      sessions,
	}
}

func (c Controller) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/auth/login", c.handleLogin)
	mux.HandleFunc("POST /api/auth/logout", c.handleLogout)
	mux.Handle("GET /api/auth/me", RequireSession(c.sessions, http.HandlerFunc(c.handleMe)))
}

// handleLogin creates an admin session cookie.
// @Summary Login
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body object{password=string} true "Login request"
// @Success 200 {object} httpapi.StatusResponse
// @Failure 400 {object} httpapi.ErrorResponse
// @Failure 401 {object} httpapi.ErrorResponse
// @Router /api/auth/login [post]
func (c Controller) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.Password != c.adminPassword {
		httpapi.WriteError(w, http.StatusUnauthorized, "invalid password")
		return
	}
	if err := c.sessions.Set(w); err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "failed to create session")
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleLogout clears the admin session cookie.
// @Summary Logout
// @Tags Auth
// @Produce json
// @Success 200 {object} httpapi.StatusResponse
// @Router /api/auth/logout [post]
func (c Controller) handleLogout(w http.ResponseWriter, r *http.Request) {
	c.sessions.Clear(w)
	httpapi.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleMe returns the authenticated admin user.
// @Summary Current user
// @Tags Auth
// @Produce json
// @Security CookieAuth
// @Success 200 {object} httpapi.MeResponse
// @Failure 401 {object} httpapi.ErrorResponse
// @Router /api/auth/me [get]
func (c Controller) handleMe(w http.ResponseWriter, r *http.Request) {
	httpapi.WriteJSON(w, http.StatusOK, map[string]string{"role": "admin"})
}
