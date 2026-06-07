package archive

import (
	"context"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) PostgresRepository {
	return PostgresRepository{db: db}
}

func (r PostgresRepository) Status(ctx context.Context) (Status, error) {
	var out Status
	err := r.db.QueryRow(ctx, `
		SELECT
			count(*)::bigint,
			COALESCE(sum(raw_size_bytes), 0)::bigint,
			min(ingested_at),
			max(ingested_at),
			min(COALESCE(started_at, updated_at)),
			max(COALESCE(updated_at, started_at))
		FROM sessions
	`).Scan(
		&out.Sessions,
		&out.RawBytes,
		&out.OldestIngestedAt,
		&out.NewestIngestedAt,
		&out.OldestSessionTime,
		&out.NewestSessionTime,
	)
	if err != nil {
		return Status{}, err
	}
	_ = r.db.QueryRow(ctx, `SELECT count(*) FROM devices`).Scan(&out.Devices)
	_ = r.db.QueryRow(ctx, `SELECT count(*) FROM session_events`).Scan(&out.SessionEvents)
	_ = r.db.QueryRow(ctx, `SELECT count(*) FROM messages`).Scan(&out.Messages)
	_ = r.db.QueryRow(ctx, `SELECT count(*) FROM tool_events`).Scan(&out.ToolEvents)
	_ = r.db.QueryRow(ctx, `SELECT count(*) FROM usage_events`).Scan(&out.UsageEvents)
	_ = r.db.QueryRow(ctx, `
		SELECT
			count(*) FILTER (WHERE raw_file_path = '')::bigint,
			count(*) FILTER (WHERE raw_sha256 = '')::bigint
		FROM sessions
	`).Scan(&out.MissingRawFiles, &out.MissingRawSHA)
	return out, nil
}

func (r PostgresRepository) Health(ctx context.Context) (Health, error) {
	var out Health
	err := r.db.QueryRow(ctx, `
		SELECT
			(SELECT count(*)::bigint FROM sessions),
			(SELECT count(*)::bigint FROM archive_files),
			(SELECT count(*)::bigint FROM session_summaries),
			(SELECT count(*)::bigint FROM conversation_turns),
			(SELECT count(*)::bigint FROM search_documents),
			(SELECT count(*)::bigint FROM search_documents WHERE default_searchable),
			(SELECT count(*)::bigint FROM messages),
			(SELECT count(*)::bigint FROM tool_events),
			(SELECT count(*)::bigint FROM archive_files WHERE verified_at IS NOT NULL AND status = 'verified'),
			(SELECT count(*)::bigint FROM sessions WHERE raw_file_path = ''),
			(SELECT count(*)::bigint FROM sessions LEFT JOIN archive_files ON archive_files.session_id = sessions.id WHERE archive_files.session_id IS NULL),
			(SELECT max(ingested_at) FROM sessions),
			(SELECT min(ingested_at) FROM sessions)
	`).Scan(
		&out.Sessions,
		&out.ArchiveRows,
		&out.SessionSummaries,
		&out.ConversationTurns,
		&out.SearchDocuments,
		&out.DefaultSearchDocs,
		&out.Messages,
		&out.ToolEvents,
		&out.VerifiedArchiveRows,
		&out.MissingRawFiles,
		&out.MissingArchiveRows,
		&out.LatestIngestedAt,
		&out.OldestIngestedAt,
	)
	if err != nil {
		return Health{}, err
	}
	if out.MissingRawFiles > 0 || out.MissingArchiveRows > 0 || out.SessionSummaries < out.Sessions || out.SearchDocuments == 0 || out.VerifiedArchiveRows < out.ArchiveRows {
		out.Status = "attention"
	} else {
		out.Status = "ok"
	}
	return out, nil
}

func (r PostgresRepository) ByDevice(ctx context.Context) ([]DeviceSummary, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			devices.id::text,
			devices.name,
			devices.hostname,
			count(sessions.id)::bigint,
			COALESCE(sum(sessions.raw_size_bytes), 0)::bigint,
			max(sessions.ingested_at)
		FROM devices
		LEFT JOIN sessions ON sessions.device_id = devices.id
		GROUP BY devices.id
		ORDER BY COALESCE(sum(sessions.raw_size_bytes), 0) DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []DeviceSummary
	for rows.Next() {
		var row DeviceSummary
		if err := rows.Scan(&row.ID, &row.Name, &row.Hostname, &row.Sessions, &row.RawBytes, &row.LastIngestedAt); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r PostgresRepository) ByRepository(ctx context.Context, limit int) ([]RepositorySummary, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			COALESCE(sessions.repository_id::text, ''),
			COALESCE(NULLIF(repositories.repository_name, ''), 'local'),
			COALESCE(repositories.repository_url, ''),
			count(sessions.id)::bigint,
			COALESCE(sum(sessions.raw_size_bytes), 0)::bigint,
			max(sessions.ingested_at)
		FROM sessions
		LEFT JOIN repositories ON repositories.id = sessions.repository_id
		GROUP BY sessions.repository_id, repositories.repository_name, repositories.repository_url
		ORDER BY COALESCE(sum(sessions.raw_size_bytes), 0) DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []RepositorySummary
	for rows.Next() {
		var row RepositorySummary
		if err := rows.Scan(&row.ID, &row.Name, &row.URL, &row.Sessions, &row.RawBytes, &row.LastIngestedAt); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r PostgresRepository) Integrity(ctx context.Context) (IntegrityResult, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, raw_file_path, raw_size_bytes, raw_sha256
		FROM sessions
		ORDER BY ingested_at DESC
	`)
	if err != nil {
		return IntegrityResult{}, err
	}
	defer rows.Close()

	var out IntegrityResult
	for rows.Next() {
		var id, path, rawSHA string
		var expectedSize int64
		if err := rows.Scan(&id, &path, &expectedSize, &rawSHA); err != nil {
			return IntegrityResult{}, err
		}
		out.Checked++
		if rawSHA == "" {
			out.MissingSHA++
			appendIssue(&out, IntegrityIssue{SessionID: id, Path: path, Problem: "missing sha256 metadata"})
			continue
		}
		if path == "" {
			out.MissingPath++
			appendIssue(&out, IntegrityIssue{SessionID: id, Problem: "missing raw file path"})
			continue
		}
		stat, err := os.Stat(path)
		if err != nil {
			out.MissingFile++
			appendIssue(&out, IntegrityIssue{SessionID: id, Path: path, Problem: "raw file not found"})
			continue
		}
		if expectedSize > 0 && stat.Size() != expectedSize {
			out.SizeMismatch++
			appendIssue(&out, IntegrityIssue{SessionID: id, Path: path, Problem: "raw size mismatch"})
			continue
		}
		out.OK++
	}
	return out, rows.Err()
}

func appendIssue(result *IntegrityResult, issue IntegrityIssue) {
	if len(result.Issues) < 20 {
		result.Issues = append(result.Issues, issue)
	}
}
