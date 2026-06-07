package usage

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) PostgresRepository {
	return PostgresRepository{db: db}
}

func (r PostgresRepository) Totals(ctx context.Context) (GlobalTotals, error) {
	var out GlobalTotals
	if err := r.db.QueryRow(ctx, `
		SELECT
			COALESCE(sum(total_tokens), 0)::bigint,
			COALESCE(sum(input_tokens), 0)::bigint,
			COALESCE(sum(output_tokens), 0)::bigint,
			COALESCE(sum(cost_usd), 0)::float8,
			COALESCE(sum(patch_added_lines), 0)::bigint
		FROM session_summaries
	`).Scan(&out.TotalTokens, &out.InputTokens, &out.OutputTokens, &out.CostUSD, &out.PatchAddedLines); err != nil {
		return GlobalTotals{}, err
	}
	if err := r.db.QueryRow(ctx, `SELECT count(*)::bigint FROM sessions`).Scan(&out.Sessions); err != nil {
		return GlobalTotals{}, err
	}
	if err := r.db.QueryRow(ctx, `
		SELECT count(*)::bigint
		FROM (
			SELECT 1
			FROM projects
			GROUP BY display_name, cwd, relative_path
		) grouped_projects
	`).Scan(&out.Projects); err != nil {
		return GlobalTotals{}, err
	}
	if err := r.db.QueryRow(ctx, `SELECT count(*)::bigint FROM devices`).Scan(&out.Devices); err != nil {
		return GlobalTotals{}, err
	}
	return out, nil
}

func (r PostgresRepository) Windows(ctx context.Context) ([]Window, error) {
	rows, err := r.db.Query(ctx, `
		WITH windows(label, days) AS (
			VALUES ('Today', 1), ('Last 7 days', 7), ('Last 30 days', 30)
		),
		usage_totals AS (
			SELECT
				windows.days,
				COALESCE(sum(usage_events.input_tokens), 0)::bigint AS input_tokens,
				COALESCE(sum(usage_events.cached_input_tokens), 0)::bigint AS cached_input_tokens,
				COALESCE(sum(usage_events.output_tokens), 0)::bigint AS output_tokens,
				COALESCE(sum(usage_events.reasoning_output_tokens), 0)::bigint AS reasoning_output_tokens,
				COALESCE(sum(usage_events.total_tokens), 0)::bigint AS total_tokens,
				COALESCE(sum(usage_events.cost_usd), 0)::float8 AS cost_usd,
				count(DISTINCT usage_events.session_id)::bigint AS sessions,
				min(usage_events.occurred_at) AS from_time,
				max(usage_events.occurred_at) AS to_time
			FROM windows
			LEFT JOIN usage_events
				ON usage_events.occurred_at >= (CURRENT_DATE - ((windows.days - 1) * INTERVAL '1 day'))
			GROUP BY windows.days
		),
		message_counts AS (
			SELECT windows.days, count(messages.id)::bigint AS messages
			FROM windows
			LEFT JOIN messages
				ON messages.occurred_at >= (CURRENT_DATE - ((windows.days - 1) * INTERVAL '1 day'))
			GROUP BY windows.days
		),
		tool_counts AS (
			SELECT windows.days, count(tool_events.id)::bigint AS tool_calls
			FROM windows
			LEFT JOIN tool_events
				ON tool_events.occurred_at >= (CURRENT_DATE - ((windows.days - 1) * INTERVAL '1 day'))
			GROUP BY windows.days
		),
		filtered_sessions AS (
			SELECT DISTINCT windows.days, usage_events.session_id
			FROM windows
			JOIN usage_events
				ON usage_events.occurred_at >= (CURRENT_DATE - ((windows.days - 1) * INTERVAL '1 day'))
		),
		patch_totals AS (
			SELECT
				filtered_sessions.days,
				COALESCE(sum(session_summaries.patch_added_lines), 0)::bigint AS patch_added_lines
			FROM filtered_sessions
			JOIN session_summaries ON session_summaries.session_id = filtered_sessions.session_id
			GROUP BY filtered_sessions.days
		)
		SELECT
			windows.label,
			windows.days,
			usage_totals.input_tokens,
			usage_totals.cached_input_tokens,
			usage_totals.output_tokens,
			usage_totals.reasoning_output_tokens,
			usage_totals.total_tokens,
			usage_totals.cost_usd,
			usage_totals.sessions,
			COALESCE(message_counts.messages, 0)::bigint,
			COALESCE(tool_counts.tool_calls, 0)::bigint,
			COALESCE(patch_totals.patch_added_lines, 0)::bigint,
			usage_totals.from_time,
			usage_totals.to_time
		FROM windows
		JOIN usage_totals ON usage_totals.days = windows.days
		LEFT JOIN message_counts ON message_counts.days = windows.days
		LEFT JOIN tool_counts ON tool_counts.days = windows.days
		LEFT JOIN patch_totals ON patch_totals.days = windows.days
		ORDER BY windows.days ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Window
	for rows.Next() {
		var item Window
		if err := rows.Scan(
			&item.Label,
			&item.Days,
			&item.Totals.InputTokens,
			&item.Totals.CachedInputTokens,
			&item.Totals.OutputTokens,
			&item.Totals.ReasoningOutputTokens,
			&item.Totals.TotalTokens,
			&item.Totals.CostUSD,
			&item.Totals.Sessions,
			&item.Totals.Messages,
			&item.Totals.ToolCalls,
			&item.Totals.PatchAddedLines,
			&item.From,
			&item.To,
		); err != nil {
			return nil, err
		}
		if item.Totals.InputTokens > 0 {
			item.CacheHitRate = float64(item.Totals.CachedInputTokens) / float64(item.Totals.InputTokens)
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r PostgresRepository) Series(ctx context.Context, params SeriesParams) ([]SeriesBucket, error) {
	bucketExpr := "bucket_date"
	patchBucketExpr := "usage_events.occurred_at::date"
	switch params.Bucket {
	case "week":
		bucketExpr = "date_trunc('week', bucket_date)::date"
		patchBucketExpr = "date_trunc('week', usage_events.occurred_at)::date"
	case "month":
		bucketExpr = "bucket_month"
		patchBucketExpr = "date_trunc('month', usage_events.occurred_at)::date"
	}

	where := []string{"1=1"}
	var args []any
	addDateFilters(params.Filters, &where, &args, "bucket_date")
	addIDFilter(params.Filters.DeviceID, &where, &args, "device_id")
	addIDFilter(params.Filters.RepositoryID, &where, &args, "repository_id")
	addIDFilter(params.Filters.ProjectID, &where, &args, "project_id")
	addTextFilter(params.Filters.Model, &where, &args, "model")

	patchWhere := []string{"usage_events.occurred_at IS NOT NULL"}
	addDateFilters(params.Filters, &patchWhere, &args, "usage_events.occurred_at")
	addIDFilter(params.Filters.DeviceID, &patchWhere, &args, "sessions.device_id")
	addIDFilter(params.Filters.RepositoryID, &patchWhere, &args, "sessions.repository_id")
	addIDFilter(params.Filters.ProjectID, &patchWhere, &args, "sessions.project_id")
	addTextFilter(params.Filters.Model, &patchWhere, &args, "usage_events.model")

	query := fmt.Sprintf(`
		WITH usage_buckets AS (
			SELECT
				%s AS bucket,
				COALESCE(sum(input_tokens), 0)::bigint AS input_tokens,
				COALESCE(sum(cached_input_tokens), 0)::bigint AS cached_input_tokens,
				COALESCE(sum(output_tokens), 0)::bigint AS output_tokens,
				COALESCE(sum(reasoning_output_tokens), 0)::bigint AS reasoning_output_tokens,
				COALESCE(sum(total_tokens), 0)::bigint AS total_tokens,
				COALESCE(sum(cost_usd), 0)::float8 AS cost_usd
			FROM usage_rollups
			WHERE %s
			GROUP BY bucket
		),
		patch_bucket_sessions AS (
			SELECT DISTINCT ON (usage_events.session_id)
				%s AS bucket,
				usage_events.session_id
			FROM usage_events
			JOIN sessions ON sessions.id = usage_events.session_id
			LEFT JOIN repositories ON repositories.id = sessions.repository_id
			LEFT JOIN projects ON projects.id = sessions.project_id
			WHERE %s
			ORDER BY usage_events.session_id, usage_events.occurred_at DESC
		),
		patch_buckets AS (
			SELECT
				patch_bucket_sessions.bucket,
				COALESCE(sum(session_summaries.patch_added_lines), 0)::bigint AS patch_added_lines
			FROM patch_bucket_sessions
			JOIN session_summaries ON session_summaries.session_id = patch_bucket_sessions.session_id
			GROUP BY patch_bucket_sessions.bucket
		)
		SELECT
			usage_buckets.bucket,
			usage_buckets.input_tokens,
			usage_buckets.cached_input_tokens,
			usage_buckets.output_tokens,
			usage_buckets.reasoning_output_tokens,
			usage_buckets.total_tokens,
			usage_buckets.cost_usd,
			COALESCE(patch_buckets.patch_added_lines, 0)::bigint
		FROM usage_buckets
		LEFT JOIN patch_buckets ON patch_buckets.bucket = usage_buckets.bucket
		ORDER BY usage_buckets.bucket ASC
	`, bucketExpr, strings.Join(where, " AND "), patchBucketExpr, strings.Join(patchWhere, " AND "))

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []SeriesBucket
	for rows.Next() {
		var item SeriesBucket
		var date time.Time
		if err := rows.Scan(
			&date,
			&item.InputTokens,
			&item.CachedInputTokens,
			&item.OutputTokens,
			&item.ReasoningOutputTokens,
			&item.TotalTokens,
			&item.CostUSD,
			&item.PatchAddedLines,
		); err != nil {
			return nil, err
		}
		item.Bucket = date.Format(time.DateOnly)
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r PostgresRepository) Breakdown(ctx context.Context, params BreakdownParams) ([]BreakdownItem, error) {
	if params.GroupBy == "language" {
		return r.languageBreakdown(ctx, params)
	}

	type groupSpec struct {
		ID     string
		Label  string
		Detail string
		Group  string
		Order  string
	}
	specs := map[string]groupSpec{
		"model": {
			ID:     "usage_events.model",
			Label:  "usage_events.model",
			Detail: "''",
			Group:  "usage_events.model",
			Order:  "total_tokens DESC",
		},
		"repository": {
			ID:     "COALESCE(sessions.repository_id::text, '')",
			Label:  "COALESCE(NULLIF(repositories.repository_name, ''), 'local')",
			Detail: "COALESCE(repositories.repository_url, '')",
			Group:  "sessions.repository_id, repositories.repository_name, repositories.repository_url",
			Order:  "total_tokens DESC",
		},
		"project": {
			ID:     "min(COALESCE(sessions.project_id::text, ''))",
			Label:  "COALESCE(NULLIF(projects.display_name, ''), NULLIF(projects.cwd, ''), 'unknown')",
			Detail: "COALESCE(projects.cwd, '')",
			Group:  "projects.display_name, projects.cwd, projects.relative_path",
			Order:  "total_tokens DESC",
		},
		"device": {
			ID:     "sessions.device_id::text",
			Label:  "devices.name",
			Detail: "devices.hostname",
			Group:  "sessions.device_id, devices.name, devices.hostname",
			Order:  "total_tokens DESC",
		},
	}
	spec := specs[params.GroupBy]

	where := []string{"1=1"}
	var args []any
	addDateFilters(params.Filters, &where, &args, "usage_events.occurred_at")
	addIDFilter(params.Filters.DeviceID, &where, &args, "sessions.device_id")
	addIDFilter(params.Filters.RepositoryID, &where, &args, "sessions.repository_id")
	addIDFilter(params.Filters.ProjectID, &where, &args, "sessions.project_id")
	addTextFilter(params.Filters.Model, &where, &args, "usage_events.model")
	limitPlaceholder := appendSQLArg(&args, params.Limit)

	query := fmt.Sprintf(`
		SELECT
			%s AS id,
			%s AS label,
			%s AS detail,
			count(DISTINCT sessions.id)::bigint,
			COALESCE(sum(usage_events.input_tokens), 0)::bigint,
			COALESCE(sum(usage_events.cached_input_tokens), 0)::bigint,
			COALESCE(sum(usage_events.output_tokens), 0)::bigint,
			COALESCE(sum(usage_events.reasoning_output_tokens), 0)::bigint,
			COALESCE(sum(usage_events.total_tokens), 0)::bigint AS total_tokens,
			COALESCE(sum(usage_events.cost_usd), 0)::float8,
			0::bigint AS patch_added_lines
		FROM usage_events
		JOIN sessions ON sessions.id = usage_events.session_id
		JOIN devices ON devices.id = sessions.device_id
		LEFT JOIN repositories ON repositories.id = sessions.repository_id
		LEFT JOIN projects ON projects.id = sessions.project_id
		WHERE %s
		GROUP BY %s
		ORDER BY %s
		LIMIT %s
	`, spec.ID, spec.Label, spec.Detail, strings.Join(where, " AND "), spec.Group, spec.Order, limitPlaceholder)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []BreakdownItem
	for rows.Next() {
		var item BreakdownItem
		if err := rows.Scan(
			&item.ID,
			&item.Label,
			&item.Detail,
			&item.Sessions,
			&item.InputTokens,
			&item.CachedInputTokens,
			&item.OutputTokens,
			&item.ReasoningOutputTokens,
			&item.TotalTokens,
			&item.CostUSD,
			&item.PatchAddedLines,
		); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r PostgresRepository) languageBreakdown(ctx context.Context, params BreakdownParams) ([]BreakdownItem, error) {
	where := []string{"1=1"}
	var args []any
	addDateFilters(params.Filters, &where, &args, "usage_events.occurred_at")
	addIDFilter(params.Filters.DeviceID, &where, &args, "sessions.device_id")
	addIDFilter(params.Filters.RepositoryID, &where, &args, "sessions.repository_id")
	addIDFilter(params.Filters.ProjectID, &where, &args, "sessions.project_id")
	addTextFilter(params.Filters.Model, &where, &args, "usage_events.model")
	limitPlaceholder := appendSQLArg(&args, params.Limit)

	query := fmt.Sprintf(`
		WITH filtered_sessions AS (
			SELECT DISTINCT sessions.id
			FROM usage_events
			JOIN sessions ON sessions.id = usage_events.session_id
			LEFT JOIN repositories ON repositories.id = sessions.repository_id
			LEFT JOIN projects ON projects.id = sessions.project_id
			WHERE %s
		)
		SELECT
			language_stats.key AS id,
			language_stats.key AS label,
			'apply_patch added lines' AS detail,
			count(DISTINCT filtered_sessions.id)::bigint AS sessions,
			0::bigint AS input_tokens,
			0::bigint AS cached_input_tokens,
			0::bigint AS output_tokens,
			0::bigint AS reasoning_output_tokens,
			0::bigint AS total_tokens,
			0::float8 AS cost_usd,
			COALESCE(sum(language_stats.value::bigint), 0)::bigint AS patch_added_lines
		FROM filtered_sessions
		JOIN session_summaries ON session_summaries.session_id = filtered_sessions.id
		CROSS JOIN LATERAL jsonb_each_text(session_summaries.patch_language_stats) AS language_stats(key, value)
		WHERE language_stats.value::bigint > 0
		GROUP BY language_stats.key
		ORDER BY patch_added_lines DESC, language_stats.key ASC
		LIMIT %s
	`, strings.Join(where, " AND "), limitPlaceholder)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []BreakdownItem
	for rows.Next() {
		var item BreakdownItem
		if err := rows.Scan(
			&item.ID,
			&item.Label,
			&item.Detail,
			&item.Sessions,
			&item.InputTokens,
			&item.CachedInputTokens,
			&item.OutputTokens,
			&item.ReasoningOutputTokens,
			&item.TotalTokens,
			&item.CostUSD,
			&item.PatchAddedLines,
		); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r PostgresRepository) Summary(ctx context.Context, filters Filters) (Totals, int64, error) {
	where := []string{"1=1"}
	var args []any
	addDateFilters(filters, &where, &args, "usage_events.occurred_at")
	addIDFilter(filters.DeviceID, &where, &args, "sessions.device_id")
	addIDFilter(filters.RepositoryID, &where, &args, "sessions.repository_id")
	addIDFilter(filters.ProjectID, &where, &args, "sessions.project_id")
	addTextFilter(filters.Model, &where, &args, "usage_events.model")

	var out Totals
	var activeDays int64
	query := fmt.Sprintf(`
		WITH filtered_usage AS (
			SELECT usage_events.*
			FROM usage_events
			JOIN sessions ON sessions.id = usage_events.session_id
			LEFT JOIN repositories ON repositories.id = sessions.repository_id
			LEFT JOIN projects ON projects.id = sessions.project_id
			WHERE %s
		),
		filtered_sessions AS (
			SELECT DISTINCT session_id
			FROM filtered_usage
		)
		SELECT
			COALESCE(sum(input_tokens), 0)::bigint,
			COALESCE(sum(cached_input_tokens), 0)::bigint,
			COALESCE(sum(output_tokens), 0)::bigint,
			COALESCE(sum(reasoning_output_tokens), 0)::bigint,
			COALESCE(sum(total_tokens), 0)::bigint,
			COALESCE(sum(cost_usd), 0)::float8,
			count(DISTINCT occurred_at::date)::bigint,
			(SELECT count(*)::bigint FROM filtered_sessions),
			(SELECT count(*)::bigint FROM messages JOIN filtered_sessions ON filtered_sessions.session_id = messages.session_id),
			(SELECT count(*)::bigint FROM tool_events JOIN filtered_sessions ON filtered_sessions.session_id = tool_events.session_id),
			(
				SELECT COALESCE(sum(session_summaries.patch_added_lines), 0)::bigint
				FROM session_summaries
				JOIN filtered_sessions ON filtered_sessions.session_id = session_summaries.session_id
			)
		FROM filtered_usage
	`, strings.Join(where, " AND "))
	err := r.db.QueryRow(ctx, query, args...).Scan(
		&out.InputTokens,
		&out.CachedInputTokens,
		&out.OutputTokens,
		&out.ReasoningOutputTokens,
		&out.TotalTokens,
		&out.CostUSD,
		&activeDays,
		&out.Sessions,
		&out.Messages,
		&out.ToolCalls,
		&out.PatchAddedLines,
	)
	return out, activeDays, err
}

func (r PostgresRepository) Calendar(ctx context.Context, params CalendarParams) ([]CalendarDay, error) {
	where := []string{"bucket_date >= CURRENT_DATE - ($1::int - 1)"}
	args := []any{params.Days}
	addDateFilters(params.Filters, &where, &args, "bucket_date")
	addIDFilter(params.Filters.DeviceID, &where, &args, "device_id")
	addIDFilter(params.Filters.RepositoryID, &where, &args, "repository_id")
	addIDFilter(params.Filters.ProjectID, &where, &args, "project_id")
	addTextFilter(params.Filters.Model, &where, &args, "model")

	rows, err := r.db.Query(ctx, fmt.Sprintf(`
		SELECT
			bucket_date,
			COALESCE(sum(total_tokens), 0)::bigint,
			COALESCE(sum(cost_usd), 0)::float8,
			count(DISTINCT NULLIF(project_id::text, ''))::bigint
		FROM usage_rollups
		WHERE %s
		GROUP BY bucket_date
		ORDER BY bucket_date ASC
	`, strings.Join(where, " AND ")), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []CalendarDay
	for rows.Next() {
		var item CalendarDay
		var date time.Time
		if err := rows.Scan(&date, &item.TotalTokens, &item.CostUSD, &item.Projects); err != nil {
			return nil, err
		}
		item.Date = date.Format(time.DateOnly)
		out = append(out, item)
	}
	return out, rows.Err()
}
