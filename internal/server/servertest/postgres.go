package servertest

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

type Fixture struct {
	DeviceID     string
	RepositoryID string
	ProjectID    string
	SessionAlpha string
	SessionBeta  string
	RawPath      string
}

type RawFixture struct {
	Path      string
	RawBytes  []byte
	RawSize   int64
	RawSHA256 string
}

func StartPostgres(t *testing.T) *pgxpool.Pool {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	root := repoRoot(t)
	container, err := postgres.Run(ctx,
		"postgres:18-trixie",
		postgres.WithDatabase("codex_usage_test"),
		postgres.WithUsername("codex_usage"),
		postgres.WithPassword("codex_usage"),
		postgres.WithInitScripts(filepath.Join(root, "migrations", "001_initial.sql")),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}
	t.Cleanup(func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		_ = container.Terminate(shutdownCtx)
	})

	connString, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("postgres connection string: %v", err)
	}

	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := pingPostgres(ctx, pool); err != nil {
		t.Fatalf("ping postgres: %v", err)
	}
	return pool
}

func pingPostgres(ctx context.Context, pool *pgxpool.Pool) error {
	var err error
	for range 20 {
		if err = pool.Ping(ctx); err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(250 * time.Millisecond):
		}
	}
	return err
}

func Reset(t *testing.T, ctx context.Context, db *pgxpool.Pool) {
	t.Helper()

	_, err := db.Exec(ctx, `
		TRUNCATE
			session_rollups,
			usage_rollups,
			usage_events,
			search_documents,
			conversation_turns,
			session_summaries,
			tool_events,
			messages,
			session_events,
			import_runs,
			archive_files,
			sessions,
			projects,
			repository_aliases,
			repositories,
			devices
		RESTART IDENTITY CASCADE
	`)
	if err != nil {
		t.Fatalf("reset test database: %v", err)
	}
}

func SeedFixture(t *testing.T, ctx context.Context, db *pgxpool.Pool) Fixture {
	t.Helper()

	fixture := Fixture{
		DeviceID:     "00000000-0000-0000-0000-000000000001",
		RepositoryID: "00000000-0000-0000-0000-000000000101",
		ProjectID:    "00000000-0000-0000-0000-000000000201",
		SessionAlpha: "session-alpha",
		SessionBeta:  "session-beta",
	}
	rawPath, rawSize := writeRawSession(t)
	fixture.RawPath = rawPath

	batch := []struct {
		sql  string
		args []any
	}{
		{
			sql: `INSERT INTO devices (id, name, hostname, platform)
			      VALUES ($1, 'Work MacBook', 'work-macbook', 'darwin')`,
			args: []any{fixture.DeviceID},
		},
		{
			sql: `INSERT INTO repositories (id, repository_url, repository_host, repository_owner, repository_name)
			      VALUES ($1, 'https://github.com/acme/codex-usage', 'github.com', 'acme', 'codex-usage')`,
			args: []any{fixture.RepositoryID},
		},
		{
			sql: `INSERT INTO projects (id, repository_id, cwd, git_root, relative_path, display_name)
			      VALUES ($1, $2, '/repo/codex-usage', '/repo/codex-usage', '.', 'codex-usage')`,
			args: []any{fixture.ProjectID, fixture.RepositoryID},
		},
		{
			sql: `INSERT INTO sessions (
					id, device_id, repository_id, project_id, started_at, updated_at, cwd,
					originator, source, cli_version, model_provider, branch, commit_hash,
					raw_sha256, raw_size_bytes, raw_file_path
			      )
			      VALUES (
					$1, $2, $3, $4, CURRENT_DATE + TIME '10:00', CURRENT_DATE + TIME '10:20',
					'/repo/codex-usage', 'codex', 'cli', '1.0.0', 'openai', 'main', 'abc123',
					'raw-alpha-sha', $5, $6
			      )`,
			args: []any{fixture.SessionAlpha, fixture.DeviceID, fixture.RepositoryID, fixture.ProjectID, rawSize, rawPath},
		},
		{
			sql: `INSERT INTO sessions (
					id, device_id, repository_id, project_id, started_at, updated_at, cwd,
					originator, source, cli_version, model_provider, branch, commit_hash,
					raw_sha256, raw_size_bytes, raw_file_path
			      )
			      VALUES (
					$1, $2, $3, $4, CURRENT_DATE - INTERVAL '1 day' + TIME '09:00',
					CURRENT_DATE - INTERVAL '1 day' + TIME '09:05',
					'/repo/codex-usage', 'codex', 'cli', '1.0.0', 'openai', 'feature/cache', 'def456',
					'', 80, ''
			      )`,
			args: []any{fixture.SessionBeta, fixture.DeviceID, fixture.RepositoryID, fixture.ProjectID},
		},
		{
			sql: `INSERT INTO archive_files (
					session_id, device_id, raw_file_path, raw_sha256,
					raw_size_bytes, verified_at, status
			      )
			      VALUES ($1, $2, $3, 'raw-alpha-sha', $4, now(), 'verified')`,
			args: []any{fixture.SessionAlpha, fixture.DeviceID, rawPath, rawSize},
		},
	}
	for _, item := range batch {
		if _, err := db.Exec(ctx, item.sql, item.args...); err != nil {
			t.Fatalf("seed fixture: %v\nsql: %s", err, item.sql)
		}
	}

	insertSummary(t, ctx, db, fixture.SessionAlpha, "Cache test session", "Cache tuning", "Improve cache usage", "gpt-5", 100, 20, 150, 50, 300, 0.03, 2, 1, 4)
	insertSummary(t, ctx, db, fixture.SessionBeta, "Repository cleanup", "Repository cleanup", "Clean repository", "gpt-5", 30, 5, 40, 5, 80, 0.01, 1, 0, 0)
	insertTimeline(t, ctx, db)
	insertUsage(t, ctx, db, fixture.SessionAlpha, fixture.SessionBeta, fixture.DeviceID, fixture.RepositoryID, fixture.ProjectID)

	return fixture
}

func insertSummary(t *testing.T, ctx context.Context, db *pgxpool.Pool, sessionID string, title string, displayTitle string, intent string, model string, input int, cached int, output int, reasoning int, total int, cost float64, messages int, tools int, patchAddedLines int) {
	t.Helper()

	_, err := db.Exec(ctx, `
		INSERT INTO session_summaries (
			session_id, title, display_title, display_subtitle, user_intent, dominant_language,
			first_user_message, last_user_message, short_summary, message_count,
			user_message_count, assistant_message_count, tool_call_count, patch_added_lines, conversation_turn_count,
			searchable_message_count, searchable_tool_count, main_model, models,
			input_tokens, cached_input_tokens, output_tokens, reasoning_output_tokens,
			total_tokens, cost_usd, duration_seconds, cache_hit_rate, started_at, updated_at
		)
		SELECT
			$1, $2, $3, 'server integration test', $4, 'go',
			'How can we improve cache usage?', 'Use Testcontainers with Postgres.',
			'Fixture summary', $12::bigint, 1, GREATEST($12::bigint - 1, 0), $13::bigint, $14::bigint, 1,
			$12::bigint, $13::bigint, $5, $5, $6, $7, $8, $9, $10, $11, 120, 0.2,
			sessions.started_at, sessions.updated_at
		FROM sessions
		WHERE sessions.id = $1
	`, sessionID, title, displayTitle, intent, model, input, cached, output, reasoning, total, cost, messages, tools, patchAddedLines)
	if err != nil {
		t.Fatalf("insert summary: %v", err)
	}
}

func insertTimeline(t *testing.T, ctx context.Context, db *pgxpool.Pool) {
	t.Helper()

	statements := []string{
		`INSERT INTO session_events (session_id, seq, event_hash, occurred_at, event_type, payload_type, role, content_text, payload_jsonb)
		 VALUES ('session-alpha', 1, 'event-alpha-1', CURRENT_DATE + TIME '10:01', 'message', 'message', 'user', 'How can we improve cache usage?', '{"role":"user"}')`,
		`INSERT INTO messages (session_id, seq, occurred_at, role, content_text, content_jsonb)
		 VALUES ('session-alpha', 1, CURRENT_DATE + TIME '10:01', 'user', 'How can we improve cache usage?', '{"role":"user"}')`,
		`INSERT INTO tool_events (session_id, seq, occurred_at, kind, tool_name, call_id, status, input_text, output_text, payload_jsonb)
		 VALUES ('session-alpha', 2, CURRENT_DATE + TIME '10:02', 'call', 'shell', 'call-1', 'ok', 'docker ps', 'postgres container is running', '{"tool":"shell"}')`,
		`INSERT INTO messages (session_id, seq, occurred_at, role, content_text, content_jsonb)
		 VALUES ('session-alpha', 3, CURRENT_DATE + TIME '10:03', 'assistant', 'Use Testcontainers with Postgres for realistic tests.', '{"role":"assistant"}')`,
		`INSERT INTO conversation_turns (
			session_id, turn_index, user_seq, assistant_seq, started_at, ended_at, user_text,
			assistant_text, tool_call_count, tool_result_count, tool_names
		 )
		 VALUES (
			'session-alpha', 0, 1, 3, CURRENT_DATE + TIME '10:01', CURRENT_DATE + TIME '10:03',
			'How can we improve cache usage?', 'Use Testcontainers with Postgres for realistic tests.',
			1, 1, ARRAY['shell']
		 )`,
		`INSERT INTO search_documents (
			session_id, seq, turn_index, occurred_at, kind, document_scope, role, title, body,
			snippet, rank_weight, default_searchable
		 )
		 VALUES (
			'session-alpha', 1, 0, CURRENT_DATE + TIME '10:01', 'message', 'user', 'user',
			'Cache usage question', 'How can we improve cache usage?', '', 10, true
		 )`,
		`INSERT INTO search_documents (
			session_id, seq, turn_index, occurred_at, kind, document_scope, tool_name, title, body,
			snippet, rank_weight, default_searchable
		 )
		 VALUES (
			'session-alpha', 2, 0, CURRENT_DATE + TIME '10:02', 'tool', 'tool', 'shell',
			'Postgres container', 'postgres container is running in docker', '', 8, true
		 )`,
		`INSERT INTO messages (session_id, seq, occurred_at, role, content_text, content_jsonb)
		 VALUES ('session-beta', 1, CURRENT_DATE - INTERVAL '1 day' + TIME '09:01', 'user', 'Clean repository metadata.', '{"role":"user"}')`,
		`INSERT INTO search_documents (
			session_id, seq, occurred_at, kind, document_scope, role, title, body,
			snippet, rank_weight, default_searchable
		 )
		 VALUES (
			'session-beta', 1, CURRENT_DATE - INTERVAL '1 day' + TIME '09:01', 'message', 'user', 'user',
			'Repository cleanup', 'Clean repository metadata.', '', 5, true
		 )`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(ctx, statement); err != nil {
			t.Fatalf("insert timeline: %v\nsql: %s", err, statement)
		}
	}

}

func insertUsage(t *testing.T, ctx context.Context, db *pgxpool.Pool, alpha string, beta string, deviceID string, repositoryID string, projectID string) {
	t.Helper()

	statements := []struct {
		sql  string
		args []any
	}{
		{
			sql: `INSERT INTO usage_events (
					session_id, seq, occurred_at, model, input_tokens, cached_input_tokens,
					output_tokens, reasoning_output_tokens, total_tokens, cost_usd
			      )
			      VALUES ($1, 10, CURRENT_DATE + TIME '10:04', 'gpt-5', 80, 15, 130, 40, 250, 0.025)`,
			args: []any{alpha},
		},
		{
			sql: `INSERT INTO usage_events (
					session_id, seq, occurred_at, model, input_tokens, cached_input_tokens,
					output_tokens, reasoning_output_tokens, total_tokens, cost_usd
			      )
			      VALUES ($1, 11, CURRENT_DATE + TIME '10:05', 'gpt-5-mini', 20, 5, 20, 10, 50, 0.005)`,
			args: []any{alpha},
		},
		{
			sql: `INSERT INTO usage_events (
					session_id, seq, occurred_at, model, input_tokens, cached_input_tokens,
					output_tokens, reasoning_output_tokens, total_tokens, cost_usd
			      )
			      VALUES ($1, 10, CURRENT_DATE - INTERVAL '1 day' + TIME '09:04', 'gpt-5', 30, 5, 40, 5, 80, 0.01)`,
			args: []any{beta},
		},
		{
			sql: `INSERT INTO usage_rollups (
					bucket_date, bucket_month, device_id, repository_id, project_id, model,
					input_tokens, cached_input_tokens, output_tokens, reasoning_output_tokens,
					total_tokens, cost_usd
			      )
			      VALUES (
					CURRENT_DATE, date_trunc('month', CURRENT_DATE)::date, $1, $2, $3, 'gpt-5',
					80, 15, 130, 40, 250, 0.025
			      )`,
			args: []any{deviceID, repositoryID, projectID},
		},
		{
			sql: `INSERT INTO usage_rollups (
					bucket_date, bucket_month, device_id, repository_id, project_id, model,
					input_tokens, cached_input_tokens, output_tokens, reasoning_output_tokens,
					total_tokens, cost_usd
			      )
			      VALUES (
					CURRENT_DATE, date_trunc('month', CURRENT_DATE)::date, $1, $2, $3, 'gpt-5-mini',
					20, 5, 20, 10, 50, 0.005
			      )`,
			args: []any{deviceID, repositoryID, projectID},
		},
		{
			sql: `INSERT INTO usage_rollups (
					bucket_date, bucket_month, device_id, repository_id, project_id, model,
					input_tokens, cached_input_tokens, output_tokens, reasoning_output_tokens,
					total_tokens, cost_usd
			      )
			      VALUES (
					CURRENT_DATE - 1, date_trunc('month', CURRENT_DATE - 1)::date, $1, $2, $3, 'gpt-5',
					30, 5, 40, 5, 80, 0.01
			      )`,
			args: []any{deviceID, repositoryID, projectID},
		},
		{
			sql: `INSERT INTO session_rollups (
					bucket_date, bucket_month, device_id, repository_id, project_id, session_count,
					input_tokens, cached_input_tokens, output_tokens, reasoning_output_tokens,
					total_tokens, cost_usd, patch_added_lines
			      )
			      VALUES (
					CURRENT_DATE, date_trunc('month', CURRENT_DATE)::date, $1, $2, $3, 1,
					100, 20, 150, 50, 300, 0.03, 4
			      )`,
			args: []any{deviceID, repositoryID, projectID},
		},
		{
			sql: `INSERT INTO session_rollups (
					bucket_date, bucket_month, device_id, repository_id, project_id, session_count,
					input_tokens, cached_input_tokens, output_tokens, reasoning_output_tokens,
					total_tokens, cost_usd, patch_added_lines
			      )
			      VALUES (
					CURRENT_DATE - 1, date_trunc('month', CURRENT_DATE - 1)::date, $1, $2, $3, 1,
					30, 5, 40, 5, 80, 0.01, 0
			      )`,
			args: []any{deviceID, repositoryID, projectID},
		},
	}
	for _, statement := range statements {
		if _, err := db.Exec(ctx, statement.sql, statement.args...); err != nil {
			t.Fatalf("insert usage: %v\nsql: %s", err, statement.sql)
		}
	}
}

func writeRawSession(t *testing.T) (string, int64) {
	t.Helper()

	raw := []byte("{\"type\":\"message\",\"role\":\"user\",\"content\":\"hello\"}\n")
	fixture := WriteRaw(t, raw, "session-alpha.jsonl")
	return fixture.Path, fixture.RawSize
}

func WriteRaw(t *testing.T, raw []byte, filename string) RawFixture {
	t.Helper()

	path := filepath.Join(t.TempDir(), filename)
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatalf("write raw fixture: %v", err)
	}
	stat, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat raw fixture: %v", err)
	}
	rawSum := sha256.Sum256(raw)
	return RawFixture{
		Path:      path,
		RawBytes:  raw,
		RawSize:   stat.Size(),
		RawSHA256: hex.EncodeToString(rawSum[:]),
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not find repository root from %s", dir)
		}
		dir = parent
	}
}
