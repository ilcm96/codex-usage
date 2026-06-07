package main

import (
	"context"
	"fmt"
	"os"

	"github.com/ilcm96/codex-usage/internal/server"
	"github.com/joho/godotenv"
)

// @title codex-usage API
// @version 0.1.0
// @description API for Codex usage ingestion, session browsing, usage reporting, and archive inspection.
// @BasePath /
// @securityDefinitions.apikey CookieAuth
// @in header
// @name Cookie
// @description Admin session cookie. Login first, then send the codex_usage_session cookie.
// @securityDefinitions.apikey DeviceTokenAuth
// @in header
// @name X-Device-Token
// @description Device ingest token. Authorization: Bearer <token> is also accepted by the server.
func main() {
	_ = godotenv.Load()

	if err := server.Run(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
