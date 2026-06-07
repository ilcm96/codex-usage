package server

import (
	"net/http"

	"github.com/ilcm96/codex-usage/internal/server/archive"
	"github.com/ilcm96/codex-usage/internal/server/auth"
	"github.com/ilcm96/codex-usage/internal/server/filteroptions"
	"github.com/ilcm96/codex-usage/internal/server/ingest"
	"github.com/ilcm96/codex-usage/internal/server/projects"
	"github.com/ilcm96/codex-usage/internal/server/search"
	"github.com/ilcm96/codex-usage/internal/server/sessions"
	"github.com/ilcm96/codex-usage/internal/server/swagger"
	"github.com/ilcm96/codex-usage/internal/server/usage"
)

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealthz)

	auth.NewController(s.cfg.AdminPassword, s.sessions).Register(mux)
	if !s.cfg.IsProduction() {
		swagger.Register(mux)
	}

	sessionAuth := func(next http.Handler) http.Handler {
		return auth.RequireSession(s.sessions, next)
	}

	ingestController := ingest.NewController(
		ingest.NewService(ingest.NewPostgresRepository(s.db), s.cfg.RawDir),
		s.cfg.MaxUploadBytes,
	)
	ingestController.Register(mux, func(next http.Handler) http.Handler {
		return auth.RequireDeviceToken(s.cfg.DeviceTokens, next)
	})

	sessionService := sessions.NewService(sessions.NewPostgresRepository(s.db))
	sessions.NewController(sessionService).Register(mux, sessionAuth)

	projects.NewController(
		projects.NewService(projects.NewPostgresRepository(s.db)),
	).Register(mux, sessionAuth)

	search.NewController(
		search.NewService(search.NewPostgresRepository(s.db)),
	).Register(mux, sessionAuth)

	archive.NewController(
		archive.NewService(archive.NewPostgresRepository(s.db)),
	).Register(mux, sessionAuth)

	filteroptions.NewController(
		filteroptions.NewService(filteroptions.NewPostgresRepository(s.db)),
	).Register(mux, sessionAuth)

	usage.NewController(
		usage.NewService(usage.NewPostgresRepository(s.db)),
	).Register(mux, sessionAuth)

	var handler http.Handler = mux
	handler = withSecurityHeaders(handler)
	handler = withCORS(s.cfg.AllowedOrigins, handler)
	handler = withRequestLog(handler)
	return handler
}
