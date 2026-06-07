package projects

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) PostgresRepository {
	return PostgresRepository{db: db}
}

func (r PostgresRepository) ListRepositories(ctx context.Context) ([]RepositorySummary, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			repositories.id::text,
			repositories.repository_url,
			repositories.repository_owner,
			repositories.repository_name,
			COALESCE(sum(session_rollups.session_count), 0)::bigint,
			COALESCE(sum(session_rollups.total_tokens), 0)::bigint,
			COALESCE(sum(session_rollups.cost_usd), 0)::float8
		FROM repositories
		LEFT JOIN session_rollups ON session_rollups.repository_id = repositories.id
		GROUP BY repositories.id
		ORDER BY COALESCE(sum(session_rollups.total_tokens), 0) DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []RepositorySummary
	for rows.Next() {
		var item RepositorySummary
		if err := rows.Scan(&item.ID, &item.RepositoryURL, &item.Owner, &item.Name, &item.Sessions, &item.TotalTokens, &item.CostUSD); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r PostgresRepository) ListProjects(ctx context.Context) ([]ProjectSummary, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			min(projects.id::text),
			projects.display_name,
			projects.cwd,
			projects.relative_path,
			COALESCE(
				NULLIF(string_agg(DISTINCT NULLIF(repositories.repository_name, ''), ', '), ''),
				''
			),
			CASE
				WHEN count(DISTINCT NULLIF(repositories.repository_url, '')) = 1
				THEN COALESCE(max(NULLIF(repositories.repository_url, '')), '')
				ELSE ''
			END,
			COALESCE(sum(session_rollups.session_count), 0)::bigint,
			COALESCE(sum(session_rollups.total_tokens), 0)::bigint,
			COALESCE(sum(session_rollups.cost_usd), 0)::float8
		FROM projects
		LEFT JOIN repositories ON repositories.id = projects.repository_id
		LEFT JOIN session_rollups ON session_rollups.project_id = projects.id
		GROUP BY projects.display_name, projects.cwd, projects.relative_path
		ORDER BY COALESCE(sum(session_rollups.total_tokens), 0) DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ProjectSummary
	for rows.Next() {
		var item ProjectSummary
		if err := rows.Scan(&item.ID, &item.DisplayName, &item.CWD, &item.RelativePath, &item.Repository, &item.RepositoryURL, &item.Sessions, &item.TotalTokens, &item.CostUSD); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}
