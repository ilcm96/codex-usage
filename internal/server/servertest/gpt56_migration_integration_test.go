package servertest_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ilcm96/codex-usage/internal/server/servertest"
)

func TestNormalizeGPT56PricingMigration(t *testing.T) {
	ctx := context.Background()
	db := servertest.StartPostgres(t)
	servertest.Reset(t, ctx, db)

	if _, err := db.Exec(ctx, `
		INSERT INTO devices (id, name, hostname)
		VALUES ('00000000-0000-0000-0000-000000000001', 'GPT-5.6 Device', 'gpt-5-6-device');

		INSERT INTO sessions (
			id, device_id, started_at, updated_at, raw_sha256, raw_file_path
		)
		VALUES (
			'gpt-5-6-session', '00000000-0000-0000-0000-000000000001',
			'2026-07-21T00:00:00Z', '2026-07-21T00:01:00Z', 'gpt-5-6-raw', '/tmp/gpt-5-6.jsonl'
		);

		INSERT INTO session_events (
			session_id, seq, event_hash, occurred_at, event_type, payload_type, payload_jsonb
		)
		VALUES (
			'gpt-5-6-session', 1, 'gpt-5-6-event', '2026-07-21T00:00:30Z',
			'event_msg', 'token_count',
			'{"payload":{"info":{"last_token_usage":{"cache_write_input_tokens":30}}}}'::jsonb
		);

		INSERT INTO usage_events (
			session_id, seq, occurred_at, model, input_tokens, cached_input_tokens,
			output_tokens, total_tokens, cost_usd
		)
		VALUES (
			'gpt-5-6-session', 1, '2026-07-21T00:00:30Z', 'gpt-5.6-sol',
			100, 20, 10, 110, 0.0000275
		);

		INSERT INTO session_summaries (
			session_id, main_model, models, input_tokens, cached_input_tokens,
			output_tokens, total_tokens, cost_usd, started_at, updated_at
		)
		VALUES (
			'gpt-5-6-session', 'gpt-5.6-sol', 'gpt-5.6-sol', 100, 20,
			10, 110, 0.0000275, '2026-07-21T00:00:00Z', '2026-07-21T00:01:00Z'
		);
	`); err != nil {
		t.Fatalf("seed GPT-5.6 usage: %v", err)
	}

	migrationPath := filepath.Join("..", "..", "..", "migrations", "003_normalize_gpt_5_6_pricing.sql")
	migration, err := os.ReadFile(migrationPath)
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	for range 2 {
		if _, err := db.Exec(ctx, string(migration)); err != nil {
			t.Fatalf("run migration: %v", err)
		}
	}

	var eventWrite, summaryWrite, usageRollupWrite, sessionRollupWrite int64
	var eventCost, summaryCost, usageRollupCost, sessionRollupCost float64
	if err := db.QueryRow(ctx, `
		SELECT
			(SELECT cache_write_input_tokens FROM usage_events WHERE session_id = 'gpt-5-6-session'),
			(SELECT cost_usd::float8 FROM usage_events WHERE session_id = 'gpt-5-6-session'),
			(SELECT cache_write_input_tokens FROM session_summaries WHERE session_id = 'gpt-5-6-session'),
			(SELECT cost_usd::float8 FROM session_summaries WHERE session_id = 'gpt-5-6-session'),
			(SELECT cache_write_input_tokens FROM usage_rollups WHERE model = 'gpt-5.6-sol'),
			(SELECT cost_usd::float8 FROM usage_rollups WHERE model = 'gpt-5.6-sol'),
			(SELECT cache_write_input_tokens FROM session_rollups),
			(SELECT cost_usd::float8 FROM session_rollups)
	`).Scan(
		&eventWrite,
		&eventCost,
		&summaryWrite,
		&summaryCost,
		&usageRollupWrite,
		&usageRollupCost,
		&sessionRollupWrite,
		&sessionRollupCost,
	); err != nil {
		t.Fatalf("read normalized usage: %v", err)
	}

	const expectedCost = 0.0007475
	if eventWrite != 30 || summaryWrite != 30 || usageRollupWrite != 30 || sessionRollupWrite != 30 {
		t.Fatalf(
			"unexpected cache writes: event=%d summary=%d usageRollup=%d sessionRollup=%d",
			eventWrite,
			summaryWrite,
			usageRollupWrite,
			sessionRollupWrite,
		)
	}
	if eventCost != expectedCost || summaryCost != expectedCost || usageRollupCost != expectedCost || sessionRollupCost != expectedCost {
		t.Fatalf(
			"unexpected costs: event=%v summary=%v usageRollup=%v sessionRollup=%v",
			eventCost,
			summaryCost,
			usageRollupCost,
			sessionRollupCost,
		)
	}
}
