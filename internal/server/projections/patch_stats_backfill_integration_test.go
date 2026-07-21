package projections_test

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ilcm96/codex-usage/internal/server/projections"
	"github.com/ilcm96/codex-usage/internal/server/servertest"
)

func TestBackfillPatchStats(t *testing.T) {
	ctx := context.Background()
	db := servertest.StartPostgres(t)
	servertest.Reset(t, ctx, db)

	if _, err := db.Exec(ctx, `
		INSERT INTO devices (id, name, hostname)
		VALUES ('00000000-0000-0000-0000-000000000001', 'Backfill Device', 'backfill-device');

		INSERT INTO sessions (
			id, device_id, started_at, updated_at, raw_sha256, raw_file_path
		)
		VALUES (
			'backfill-session', '00000000-0000-0000-0000-000000000001',
			'2026-07-10T00:00:00Z', '2026-07-10T00:10:00Z', 'backfill-sha', '/tmp/backfill.jsonl'
		);

		INSERT INTO session_summaries (
			session_id, dominant_language, patch_added_lines, patch_language_stats,
			started_at, updated_at
		)
		VALUES (
			'backfill-session', 'Unknown', 0, '{}'::jsonb,
			'2026-07-10T00:00:00Z', '2026-07-10T00:10:00Z'
		);

		INSERT INTO tool_events (
			session_id, seq, occurred_at, kind, status, payload_jsonb
		)
		VALUES (
			'backfill-session', 1, '2026-07-10T00:05:00Z', 'patch_apply_end', 'completed',
			'{"type":"patch_apply_end","success":true,"changes":{"main.go":{"type":"add","content":"package main\n\nfunc main() {}\n"}}}'::jsonb
		);
	`); err != nil {
		t.Fatalf("seed patch stats backfill: %v", err)
	}

	since := time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC)
	preview, err := projections.BackfillPatchStats(ctx, db, since, true)
	if err != nil {
		t.Fatalf("preview BackfillPatchStats: %v", err)
	}
	if !preview.DryRun || preview.Sessions != 1 || preview.PreviousAddedLines != 0 || preview.UpdatedAddedLines != 3 {
		t.Fatalf("unexpected preview: %+v", preview)
	}
	assertBackfilledPatchStats(t, ctx, db, 0)

	result, err := projections.BackfillPatchStats(ctx, db, since, false)
	if err != nil {
		t.Fatalf("BackfillPatchStats: %v", err)
	}
	if result.DryRun || result.Sessions != 1 || result.PreviousAddedLines != 0 || result.UpdatedAddedLines != 3 {
		t.Fatalf("unexpected result: %+v", result)
	}
	assertBackfilledPatchStats(t, ctx, db, 3)

	secondResult, err := projections.BackfillPatchStats(ctx, db, since, false)
	if err != nil {
		t.Fatalf("second BackfillPatchStats: %v", err)
	}
	if secondResult.PreviousAddedLines != 3 || secondResult.UpdatedAddedLines != 3 {
		t.Fatalf("backfill is not idempotent: %+v", secondResult)
	}
	assertBackfilledPatchStats(t, ctx, db, 3)
}

func assertBackfilledPatchStats(t *testing.T, ctx context.Context, db *pgxpool.Pool, want int64) {
	t.Helper()

	var summaryLines, goLines, rollupLines int64
	var dominantLanguage string
	err := db.QueryRow(ctx, `
		SELECT
			ss.patch_added_lines,
			COALESCE((ss.patch_language_stats->>'Go')::bigint, 0),
			ss.dominant_language,
			COALESCE((SELECT sum(patch_added_lines)::bigint FROM session_rollups), 0)
		FROM session_summaries ss
		WHERE ss.session_id = 'backfill-session'
	`).Scan(&summaryLines, &goLines, &dominantLanguage, &rollupLines)
	if err != nil {
		t.Fatalf("load backfilled patch stats: %v", err)
	}
	if summaryLines != want || goLines != want || rollupLines != want {
		t.Fatalf("patch stats summary=%d go=%d rollup=%d, want %d", summaryLines, goLines, rollupLines, want)
	}
	if want > 0 && dominantLanguage != "Go" {
		t.Fatalf("dominant language = %q, want Go", dominantLanguage)
	}
}
