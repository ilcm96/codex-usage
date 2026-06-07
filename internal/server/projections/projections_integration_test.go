package projections_test

import (
	"context"
	"testing"
	"time"

	"github.com/ilcm96/codex-usage/internal/server/projections"
	"github.com/ilcm96/codex-usage/internal/server/servertest"
	"github.com/ilcm96/codex-usage/internal/server/sessionparse"
)

func TestRefreshSession(t *testing.T) {
	ctx := context.Background()
	db := servertest.StartPostgres(t)
	servertest.Reset(t, ctx, db)

	if _, err := db.Exec(ctx, `
		INSERT INTO devices (id, name, hostname, platform)
		VALUES ('00000000-0000-0000-0000-000000000001', 'Projection Device', 'projection-device', 'darwin')
	`); err != nil {
		t.Fatalf("insert device: %v", err)
	}
	if _, err := db.Exec(ctx, `
		INSERT INTO sessions (
			id, device_id, started_at, updated_at, cwd, branch,
			raw_sha256, raw_size_bytes, raw_file_path
		)
		VALUES (
			'projection-session', '00000000-0000-0000-0000-000000000001',
			'2026-02-08T00:00:00Z', '2026-02-08T00:00:05Z', '/repo/projection', 'main',
			'raw-sha', 100, '/tmp/projection-session.jsonl'
		)
	`); err != nil {
		t.Fatalf("insert session: %v", err)
	}
	if _, err := db.Exec(ctx, `
		INSERT INTO usage_events (
			session_id, seq, occurred_at, model, input_tokens, cached_input_tokens,
			output_tokens, reasoning_output_tokens, total_tokens, cost_usd
		)
		VALUES ('projection-session', 4, '2026-02-08T00:00:04Z', 'gpt-5', 10, 2, 5, 1, 16, 0.001)
	`); err != nil {
		t.Fatalf("insert usage event: %v", err)
	}

	startedAt := time.Date(2026, 2, 8, 0, 0, 0, 0, time.UTC)
	toolAt := startedAt.Add(2 * time.Second)
	assistantAt := startedAt.Add(3 * time.Second)
	updatedAt := startedAt.Add(5 * time.Second)
	parsed := sessionparse.Session{
		Meta: sessionparse.SessionMeta{
			ID:            "projection-session",
			CWD:           "/repo/projection",
			RepositoryURL: "https://github.com/acme/projection",
			Branch:        "main",
			CommitHash:    "abc123",
		},
		Events: []sessionparse.Event{
			{Seq: 1, OccurredAt: &startedAt, EventType: "message", Role: "user", ContentText: "Please refresh projections.", PayloadJSON: []byte(`{"role":"user"}`)},
			{Seq: 2, OccurredAt: &toolAt, EventType: "function_call", ToolName: "shell", ContentText: "go test", PayloadJSON: []byte(`{"name":"shell"}`)},
			{Seq: 3, OccurredAt: &assistantAt, EventType: "message", Role: "assistant", ContentText: "Projections refreshed.", PayloadJSON: []byte(`{"role":"assistant"}`)},
		},
		Messages: []sessionparse.Message{
			{Seq: 1, OccurredAt: &startedAt, Role: "user", ContentText: "Please refresh projections.", ContentJSON: []byte(`"Please refresh projections."`)},
			{Seq: 3, OccurredAt: &assistantAt, Role: "assistant", ContentText: "Projections refreshed.", ContentJSON: []byte(`"Projections refreshed."`)},
		},
		Tools: []sessionparse.ToolEvent{
			{Seq: 2, OccurredAt: &toolAt, Kind: "function_call", ToolName: "shell", Status: "completed", InputText: "go test", PayloadJSON: []byte(`{"name":"shell"}`)},
			{Seq: 5, OccurredAt: &toolAt, Kind: "custom_tool_call", ToolName: "apply_patch", Status: "completed", InputText: "*** Begin Patch\n*** Add File: main.go\n+package main\n+\n+func main() {}\n*** End Patch\n", PayloadJSON: []byte(`{"name":"apply_patch"}`)},
		},
		Usages: []sessionparse.UsageEvent{
			{Seq: 4, OccurredAt: &updatedAt, Model: "gpt-5", InputTokens: 10, CachedInputTokens: 2, OutputTokens: 5, ReasoningOutputTokens: 1, TotalTokens: 16, CostUSD: 0.001},
		},
		StartedAt: &startedAt,
		UpdatedAt: &updatedAt,
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	if err := projections.RefreshSession(ctx, tx, parsed, &startedAt, &updatedAt); err != nil {
		_ = tx.Rollback(ctx)
		t.Fatalf("RefreshSession failed: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit tx: %v", err)
	}

	var summaries, turns, searchDocuments, rollups, sessionRollups, totalTokens, patchAddedLines int64
	if err := db.QueryRow(ctx, `
		SELECT
			(SELECT count(*)::bigint FROM session_summaries),
			(SELECT count(*)::bigint FROM conversation_turns),
			(SELECT count(*)::bigint FROM search_documents),
			(SELECT count(*)::bigint FROM usage_rollups),
			(SELECT count(*)::bigint FROM session_rollups),
			(SELECT total_tokens FROM session_summaries WHERE session_id = 'projection-session'),
			(SELECT patch_added_lines FROM session_summaries WHERE session_id = 'projection-session')
	`).Scan(&summaries, &turns, &searchDocuments, &rollups, &sessionRollups, &totalTokens, &patchAddedLines); err != nil {
		t.Fatalf("load projection counts: %v", err)
	}
	if summaries != 1 || turns != 1 || searchDocuments != 4 || rollups != 1 || sessionRollups != 1 || totalTokens != 16 || patchAddedLines != 3 {
		t.Fatalf("unexpected projection counts: summaries=%d turns=%d searchDocuments=%d rollups=%d sessionRollups=%d totalTokens=%d patchAddedLines=%d", summaries, turns, searchDocuments, rollups, sessionRollups, totalTokens, patchAddedLines)
	}
}
