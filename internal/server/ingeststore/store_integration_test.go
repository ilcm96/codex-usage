package ingeststore_test

import (
	"context"
	"testing"

	"github.com/ilcm96/codex-usage/internal/server/ingeststore"
	"github.com/ilcm96/codex-usage/internal/server/servertest"
)

func TestStore_StoreRaw(t *testing.T) {
	ctx := context.Background()
	db := servertest.StartPostgres(t)
	servertest.Reset(t, ctx, db)

	raw := []byte(
		`{"timestamp":"2026-02-08T00:00:00Z","type":"session_meta","payload":{"id":"store-session","timestamp":"2026-02-08T00:00:00Z","cwd":"/repo/store","originator":"codex","source":"cli","cli_version":"1.0.0","model_provider":"openai","git":{"repository_url":"git@github.com:acme/store.git","branch":"main","commit_hash":"abc123"}}}` + "\n" +
			`{"timestamp":"2026-02-08T00:00:01Z","type":"turn_context","payload":{"model":"gpt-5"}}` + "\n" +
			`{"timestamp":"2026-02-08T00:00:02Z","type":"message","role":"user","content":"Please store this raw session."}` + "\n" +
			`{"timestamp":"2026-02-08T00:00:02Z","type":"response_item","payload":{"type":"custom_tool_call","name":"apply_patch","call_id":"call-1","status":"completed","input":"*** Begin Patch\n*** Add File: main.go\n+package main\n+\n+func main() {}\n*** End Patch\n"}}` + "\n" +
			`{"timestamp":"2026-02-08T00:00:03Z","type":"message","role":"assistant","content":"Stored with projections."}` + "\n" +
			`{"timestamp":"2026-02-08T00:00:04Z","type":"event_msg","payload":{"type":"token_count","info":{"last_token_usage":{"input_tokens":10,"cached_input_tokens":2,"output_tokens":5,"reasoning_output_tokens":1,"total_tokens":16}}}}` + "\n",
	)
	rawFixture := servertest.WriteRaw(t, raw, "store-session.jsonl")
	meta := ingeststore.Metadata{}
	meta.Device.Name = "Store Device"
	meta.Device.Hostname = "store-device"
	meta.Device.Platform = "darwin"
	meta.Session.ID = "store-session"
	meta.Session.Path = "/source/store-session.jsonl"
	meta.Session.RawSHA256 = rawFixture.RawSHA256
	meta.Session.RawSizeBytes = int64(len(rawFixture.RawBytes))
	meta.Workspace.CWD = "/repo/store"
	meta.Workspace.GitRoot = "/repo/store"
	meta.Workspace.RelativePath = "."
	meta.Workspace.RepositoryURL = "git@github.com:acme/store.git"
	meta.Workspace.Branch = "main"
	meta.Workspace.CommitHash = "abc123"

	store := ingeststore.New(db)
	result, err := store.StoreRaw(ctx, meta, rawFixture.Path, int64(len(rawFixture.RawBytes)))
	if err != nil {
		t.Fatalf("StoreRaw failed: %v", err)
	}
	if result.Status != "ingested" || result.SessionID != "store-session" {
		t.Fatalf("unexpected StoreRaw result: %+v", result)
	}

	var sessions, archives, summaries, turns, searchDocuments, usageRollups, sessionRollups int64
	if err := db.QueryRow(ctx, `
		SELECT
			(SELECT count(*)::bigint FROM sessions),
			(SELECT count(*)::bigint FROM archive_files WHERE status = 'verified'),
			(SELECT count(*)::bigint FROM session_summaries),
			(SELECT count(*)::bigint FROM conversation_turns),
			(SELECT count(*)::bigint FROM search_documents),
			(SELECT count(*)::bigint FROM usage_rollups),
			(SELECT count(*)::bigint FROM session_rollups)
	`).Scan(&sessions, &archives, &summaries, &turns, &searchDocuments, &usageRollups, &sessionRollups); err != nil {
		t.Fatalf("load stored counts: %v", err)
	}
	if sessions != 1 || archives != 1 || summaries != 1 || turns != 1 || searchDocuments != 3 || usageRollups != 1 || sessionRollups != 1 {
		t.Fatalf("unexpected stored counts: sessions=%d archives=%d summaries=%d turns=%d searchDocuments=%d usageRollups=%d sessionRollups=%d", sessions, archives, summaries, turns, searchDocuments, usageRollups, sessionRollups)
	}

	var totalTokens int64
	var patchAddedLines int64
	var repositoryName string
	if err := db.QueryRow(ctx, `
		SELECT session_summaries.total_tokens, session_summaries.patch_added_lines, repositories.repository_name
		FROM session_summaries
		JOIN sessions ON sessions.id = session_summaries.session_id
		JOIN repositories ON repositories.id = sessions.repository_id
		WHERE sessions.id = 'store-session'
	`).Scan(&totalTokens, &patchAddedLines, &repositoryName); err != nil {
		t.Fatalf("load stored summary: %v", err)
	}
	if totalTokens != 16 || patchAddedLines != 3 || repositoryName != "store" {
		t.Fatalf("unexpected stored summary: totalTokens=%d patchAddedLines=%d repositoryName=%q", totalTokens, patchAddedLines, repositoryName)
	}

	duplicate, err := store.StoreRaw(ctx, meta, rawFixture.Path, int64(len(rawFixture.RawBytes)))
	if err != nil {
		t.Fatalf("duplicate StoreRaw failed: %v", err)
	}
	if duplicate.Status != "skipped" || duplicate.SessionID != "store-session" {
		t.Fatalf("unexpected duplicate StoreRaw result: %+v", duplicate)
	}

	replacementRaw := []byte(
		`{"timestamp":"2026-02-09T00:00:00Z","type":"session_meta","payload":{"id":"store-session","timestamp":"2026-02-09T00:00:00Z","cwd":"/repo/store","originator":"codex","source":"cli","cli_version":"1.0.0","model_provider":"openai","git":{"repository_url":"git@github.com:acme/store.git","branch":"main","commit_hash":"def456"}}}` + "\n" +
			`{"timestamp":"2026-02-09T00:00:01Z","type":"turn_context","payload":{"model":"gpt-5"}}` + "\n" +
			`{"timestamp":"2026-02-09T00:00:02Z","type":"message","role":"user","content":"Please replace this raw session."}` + "\n" +
			`{"timestamp":"2026-02-09T00:00:03Z","type":"message","role":"assistant","content":"Replaced with new usage."}` + "\n" +
			`{"timestamp":"2026-02-09T00:00:04Z","type":"event_msg","payload":{"type":"token_count","info":{"last_token_usage":{"input_tokens":12,"cached_input_tokens":3,"output_tokens":6,"reasoning_output_tokens":2,"total_tokens":20}}}}` + "\n",
	)
	replacementFixture := servertest.WriteRaw(t, replacementRaw, "store-session-replacement.jsonl")
	replacementMeta := meta
	replacementMeta.Session.RawSHA256 = replacementFixture.RawSHA256
	replacementMeta.Session.RawSizeBytes = int64(len(replacementFixture.RawBytes))
	replacement, err := store.StoreRaw(ctx, replacementMeta, replacementFixture.Path, int64(len(replacementFixture.RawBytes)))
	if err != nil {
		t.Fatalf("replacement StoreRaw failed: %v", err)
	}
	if replacement.Status != "ingested" || replacement.SessionID != "store-session" {
		t.Fatalf("unexpected replacement StoreRaw result: %+v", replacement)
	}

	var oldDateRollups, newDateRollups, replacementTotal, oldDateSessionRollups, newDateSessionRollups, replacementSessions int64
	if err := db.QueryRow(ctx, `
		SELECT
			(SELECT count(*)::bigint FROM usage_rollups WHERE bucket_date = DATE '2026-02-08'),
			(SELECT count(*)::bigint FROM usage_rollups WHERE bucket_date = DATE '2026-02-09'),
			(SELECT COALESCE(sum(total_tokens), 0)::bigint FROM usage_rollups),
			(SELECT count(*)::bigint FROM session_rollups WHERE bucket_date = DATE '2026-02-08'),
			(SELECT count(*)::bigint FROM session_rollups WHERE bucket_date = DATE '2026-02-09'),
			(SELECT COALESCE(sum(session_count), 0)::bigint FROM session_rollups)
	`).Scan(&oldDateRollups, &newDateRollups, &replacementTotal, &oldDateSessionRollups, &newDateSessionRollups, &replacementSessions); err != nil {
		t.Fatalf("load replacement rollups: %v", err)
	}
	if oldDateRollups != 0 || newDateRollups != 1 || replacementTotal != 20 || oldDateSessionRollups != 0 || newDateSessionRollups != 1 || replacementSessions != 1 {
		t.Fatalf("unexpected replacement rollups: old=%d new=%d total=%d oldSessions=%d newSessions=%d sessions=%d", oldDateRollups, newDateRollups, replacementTotal, oldDateSessionRollups, newDateSessionRollups, replacementSessions)
	}
}
