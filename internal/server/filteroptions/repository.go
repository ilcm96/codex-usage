package filteroptions

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) PostgresRepository {
	return PostgresRepository{db: db}
}

func (r PostgresRepository) List(ctx context.Context) (Result, error) {
	var out Result
	var oldest, newest *time.Time
	var devicesJSON, repositoriesJSON, projectsJSON, modelsJSON, branchesJSON []byte
	err := r.db.QueryRow(ctx, `
		SELECT
			(SELECT min(bucket_date) FROM usage_rollups),
			(SELECT max(bucket_date) FROM usage_rollups),
			(
				SELECT COALESCE(jsonb_agg(jsonb_build_object('id', id, 'label', label, 'detail', detail, 'count', count) ORDER BY label), '[]'::jsonb)
				FROM (
					SELECT devices.id::text AS id, devices.name AS label, devices.hostname AS detail, COALESCE(sum(session_rollups.session_count), 0)::bigint AS count
					FROM devices
					LEFT JOIN session_rollups ON session_rollups.device_id = devices.id
					GROUP BY devices.id
				) device_facets
			),
			(
				SELECT COALESCE(jsonb_agg(jsonb_build_object('id', id, 'label', label, 'detail', detail, 'count', count) ORDER BY label), '[]'::jsonb)
				FROM (
					SELECT
						repositories.id::text AS id,
						COALESCE(NULLIF(repositories.repository_name, ''), repositories.repository_url) AS label,
						repositories.repository_url AS detail,
						COALESCE(sum(session_rollups.session_count), 0)::bigint AS count
					FROM repositories
					LEFT JOIN session_rollups ON session_rollups.repository_id = repositories.id
					GROUP BY repositories.id
				) repository_facets
			),
			(
				SELECT COALESCE(jsonb_agg(jsonb_build_object('id', id, 'label', label, 'detail', detail, 'count', count) ORDER BY label), '[]'::jsonb)
				FROM (
					SELECT min(projects.id::text) AS id, projects.display_name AS label, projects.cwd AS detail, COALESCE(sum(session_rollups.session_count), 0)::bigint AS count
					FROM projects
					LEFT JOIN session_rollups ON session_rollups.project_id = projects.id
					GROUP BY projects.display_name, projects.cwd, projects.relative_path
				) project_facets
			),
			(
				SELECT COALESCE(jsonb_agg(jsonb_build_object('id', id, 'label', label, 'detail', detail, 'count', count) ORDER BY label), '[]'::jsonb)
				FROM (
					SELECT model AS id, model AS label, '' AS detail, count(DISTINCT session_id)::bigint AS count
					FROM usage_events
					WHERE model <> ''
					GROUP BY model
				) model_facets
			),
			(
				SELECT COALESCE(jsonb_agg(jsonb_build_object('id', id, 'label', label, 'detail', detail, 'count', count) ORDER BY label), '[]'::jsonb)
				FROM (
					SELECT branch AS id, branch AS label, '' AS detail, count(*)::bigint AS count
					FROM sessions
					WHERE branch <> ''
					GROUP BY branch
				) branch_facets
			)
	`).Scan(&oldest, &newest, &devicesJSON, &repositoriesJSON, &projectsJSON, &modelsJSON, &branchesJSON)
	if err != nil {
		return Result{}, err
	}
	if oldest != nil {
		out.DateRange.Oldest = oldest.Format(time.DateOnly)
	}
	if newest != nil {
		out.DateRange.Newest = newest.Format(time.DateOnly)
	}
	for _, item := range []struct {
		raw    []byte
		target *[]Option
	}{
		{devicesJSON, &out.Devices},
		{repositoriesJSON, &out.Repositories},
		{projectsJSON, &out.Projects},
		{modelsJSON, &out.Models},
		{branchesJSON, &out.Branches},
	} {
		if err := json.Unmarshal(item.raw, item.target); err != nil {
			return Result{}, err
		}
	}
	return out, nil
}
