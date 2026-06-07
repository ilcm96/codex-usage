package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ilcm96/codex-usage/internal/server/auth"
	"github.com/ilcm96/codex-usage/internal/server/config"
	"github.com/ilcm96/codex-usage/internal/server/httpapi"
)

type Server struct {
	cfg      config.Config
	db       *pgxpool.Pool
	sessions auth.SessionManager
}

func Run(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	db, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer db.Close()

	if err := db.Ping(ctx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}

	if err := os.MkdirAll(cfg.RawDir, 0o755); err != nil {
		return fmt.Errorf("create raw dir: %w", err)
	}

	srv := &Server{
		cfg:      cfg,
		db:       db,
		sessions: auth.NewSessionManager(cfg.SessionSecret, cfg.SessionTTL, cfg.CookieSecure),
	}

	httpServer := &http.Server{
		Addr:              cfg.Addr,
		Handler:           srv.routes(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	slog.Info("starting codex-usage api", "addr", cfg.Addr)
	return httpServer.ListenAndServe()
}

// handleHealthz checks database availability.
// @Summary Health check
// @Tags System
// @Produce json
// @Success 200 {object} httpapi.HealthResponse
// @Failure 503 {object} httpapi.ErrorResponse
// @Router /healthz [get]
func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := s.db.Ping(ctx); err != nil {
		httpapi.WriteError(w, http.StatusServiceUnavailable, "database unavailable")
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
