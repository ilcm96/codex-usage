package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/ilcm96/codex-usage/internal/server/config"
	"github.com/ilcm96/codex-usage/internal/server/projections"
)

func main() {
	sinceValue := flag.String("since", "", "RFC3339 lower bound for patch events")
	dryRun := flag.Bool("dry-run", true, "calculate changes without updating the database")
	flag.Parse()

	if *sinceValue == "" {
		exitWithError(fmt.Errorf("--since is required"))
	}
	since, err := time.Parse(time.RFC3339, *sinceValue)
	if err != nil {
		exitWithError(fmt.Errorf("parse --since: %w", err))
	}

	_ = godotenv.Load()
	cfg, err := config.Load()
	if err != nil {
		exitWithError(err)
	}
	ctx := context.Background()
	db, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		exitWithError(fmt.Errorf("connect database: %w", err))
	}
	defer db.Close()
	if err := db.Ping(ctx); err != nil {
		exitWithError(fmt.Errorf("ping database: %w", err))
	}

	result, err := projections.BackfillPatchStats(ctx, db, since, *dryRun)
	if err != nil {
		exitWithError(err)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		exitWithError(fmt.Errorf("write result: %w", err))
	}
}

func exitWithError(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
