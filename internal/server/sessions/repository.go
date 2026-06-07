package sessions

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) PostgresRepository {
	return PostgresRepository{db: db}
}

func (r PostgresRepository) ListSimple(ctx context.Context, params ListParams) ([]SimpleSession, error) {
	where, args := buildListFilters(params)
	orderBy, _ := sessionOrderBy(params.Sort)
	limitPlaceholder := appendSQLArg(&args, params.Limit)

	query := fmt.Sprintf(`
		SELECT
			sessions.id,
			sessions.started_at,
			sessions.updated_at,
			sessions.cwd,
			sessions.branch,
			COALESCE(repositories.repository_name, ''),
			COALESCE(repositories.repository_url, ''),
			COALESCE(projects.display_name, ''),
			devices.name,
			COALESCE(session_summaries.input_tokens, 0)::bigint,
			COALESCE(session_summaries.cached_input_tokens, 0)::bigint,
			COALESCE(session_summaries.output_tokens, 0)::bigint,
			COALESCE(session_summaries.reasoning_output_tokens, 0)::bigint,
			COALESCE(session_summaries.total_tokens, 0)::bigint,
			COALESCE(session_summaries.cost_usd, 0)::float8,
			COALESCE(session_summaries.models, '')
		FROM sessions
		JOIN devices ON devices.id = sessions.device_id
		LEFT JOIN repositories ON repositories.id = sessions.repository_id
		LEFT JOIN projects ON projects.id = sessions.project_id
		LEFT JOIN session_summaries ON session_summaries.session_id = sessions.id
		WHERE %s
		ORDER BY %s
		LIMIT %s
	`, strings.Join(where, " AND "), orderBy, limitPlaceholder)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []SimpleSession
	for rows.Next() {
		var item SimpleSession
		if err := rows.Scan(
			&item.ID,
			&item.StartedAt,
			&item.UpdatedAt,
			&item.CWD,
			&item.Branch,
			&item.Repository,
			&item.RepositoryURL,
			&item.Project,
			&item.Device,
			&item.InputTokens,
			&item.CachedTokens,
			&item.OutputTokens,
			&item.Reasoning,
			&item.TotalTokens,
			&item.CostUSD,
			&item.Models,
		); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r PostgresRepository) List(ctx context.Context, params ListParams) (ListResult, error) {
	where, args := buildListFilters(params)
	orderBy, _ := sessionOrderBy(params.Sort)

	totals, err := r.loadListTotals(ctx, where, args)
	if err != nil {
		return ListResult{}, err
	}
	total := totals.Sessions

	itemArgs := append([]any{}, args...)
	limitPlaceholder := appendSQLArg(&itemArgs, params.Limit)
	offsetPlaceholder := appendSQLArg(&itemArgs, params.Offset)
	items, err := r.loadListItems(ctx, where, itemArgs, orderBy, limitPlaceholder, offsetPlaceholder)
	if err != nil {
		return ListResult{}, err
	}

	nextOffset := 0
	if int64(params.Offset+len(items)) < total {
		nextOffset = params.Offset + len(items)
	}
	return ListResult{
		Items:      items,
		Limit:      params.Limit,
		NextOffset: nextOffset,
		Offset:     params.Offset,
		Total:      total,
		Totals:     totals,
	}, nil
}

func (r PostgresRepository) ListItems(ctx context.Context, params ListParams) ([]ListItem, error) {
	where, args := buildListFilters(params)
	orderBy, _ := sessionOrderBy(params.Sort)

	itemArgs := append([]any{}, args...)
	limitPlaceholder := appendSQLArg(&itemArgs, params.Limit)
	offsetPlaceholder := appendSQLArg(&itemArgs, params.Offset)
	return r.loadListItems(ctx, where, itemArgs, orderBy, limitPlaceholder, offsetPlaceholder)
}

func (r PostgresRepository) Detail(ctx context.Context, id string) (DetailResult, error) {
	var session Detail
	err := r.db.QueryRow(ctx, `
		SELECT
			sessions.id,
			sessions.started_at,
			sessions.updated_at,
			sessions.cwd,
			COALESCE(repositories.repository_name, ''),
			COALESCE(repositories.repository_url, ''),
			COALESCE(projects.display_name, ''),
			devices.name,
			sessions.branch,
			sessions.commit_hash,
			COALESCE(session_summaries.title, ''),
			COALESCE(session_summaries.display_title, ''),
			COALESCE(session_summaries.display_subtitle, ''),
			COALESCE(session_summaries.user_intent, ''),
			COALESCE(session_summaries.dominant_language, ''),
			COALESCE(session_summaries.first_user_message, ''),
			COALESCE(session_summaries.last_user_message, ''),
			COALESCE(session_summaries.short_summary, ''),
			COALESCE(session_summaries.main_model, ''),
			COALESCE(session_summaries.duration_seconds, 0)::bigint,
			COALESCE(session_summaries.cache_hit_rate, 0)::float8,
			COALESCE(session_summaries.conversation_turn_count, 0)::bigint,
			COALESCE(session_summaries.searchable_message_count, 0)::bigint,
			COALESCE(session_summaries.searchable_tool_count, 0)::bigint,
			COALESCE(session_summaries.input_tokens, 0)::bigint,
			COALESCE(session_summaries.cached_input_tokens, 0)::bigint,
			COALESCE(session_summaries.output_tokens, 0)::bigint,
			COALESCE(session_summaries.reasoning_output_tokens, 0)::bigint,
			COALESCE(session_summaries.total_tokens, 0)::bigint,
			COALESCE(session_summaries.cost_usd, 0)::float8,
			COALESCE(session_summaries.message_count, 0)::bigint,
			COALESCE(session_summaries.tool_call_count, 0)::bigint,
			COALESCE(session_summaries.patch_added_lines, 0)::bigint
		FROM sessions
		JOIN devices ON devices.id = sessions.device_id
		LEFT JOIN repositories ON repositories.id = sessions.repository_id
		LEFT JOIN projects ON projects.id = sessions.project_id
		LEFT JOIN session_summaries ON session_summaries.session_id = sessions.id
		WHERE sessions.id = $1
	`, id).Scan(
		&session.ID,
		&session.StartedAt,
		&session.UpdatedAt,
		&session.CWD,
		&session.Repository,
		&session.RepositoryURL,
		&session.Project,
		&session.Device,
		&session.Branch,
		&session.CommitHash,
		&session.Title,
		&session.DisplayTitle,
		&session.DisplaySubtitle,
		&session.UserIntent,
		&session.DominantLanguage,
		&session.FirstUserMessage,
		&session.LastUserMessage,
		&session.ShortSummary,
		&session.MainModel,
		&session.DurationSeconds,
		&session.CacheHitRate,
		&session.ConversationTurns,
		&session.SearchableMessages,
		&session.SearchableTools,
		&session.InputTokens,
		&session.CachedInputTokens,
		&session.OutputTokens,
		&session.ReasoningOutputTokens,
		&session.TotalTokens,
		&session.CostUSD,
		&session.MessageCount,
		&session.ToolCallCount,
		&session.PatchAddedLines,
	)
	if err == pgx.ErrNoRows {
		return DetailResult{}, ErrNotFound
	}
	if err != nil {
		return DetailResult{}, err
	}

	models, err := r.loadModels(ctx, id)
	if err != nil {
		return DetailResult{}, err
	}
	return DetailResult{Session: session, Models: models, Timeline: []any{}}, nil
}

func (r PostgresRepository) Reader(ctx context.Context, id string, params ReaderParams) (ReaderResult, error) {
	var summary ReaderSummary
	err := r.db.QueryRow(ctx, `
		SELECT
			session_summaries.session_id,
			session_summaries.title,
			session_summaries.display_title,
			session_summaries.display_subtitle,
			session_summaries.user_intent,
			session_summaries.dominant_language,
			session_summaries.first_user_message,
			session_summaries.last_user_message,
			session_summaries.short_summary,
			session_summaries.message_count,
			session_summaries.user_message_count,
			session_summaries.assistant_message_count,
			session_summaries.tool_call_count,
			session_summaries.main_model,
			session_summaries.duration_seconds,
			session_summaries.cache_hit_rate,
			session_summaries.started_at,
			session_summaries.updated_at,
			session_summaries.conversation_turn_count,
			(session_summaries.searchable_message_count + session_summaries.searchable_tool_count)::bigint
		FROM session_summaries
		WHERE session_summaries.session_id = $1
	`, id).Scan(
		&summary.SessionID,
		&summary.Title,
		&summary.DisplayTitle,
		&summary.DisplaySubtitle,
		&summary.UserIntent,
		&summary.DominantLanguage,
		&summary.FirstUserMessage,
		&summary.LastUserMessage,
		&summary.ShortSummary,
		&summary.MessageCount,
		&summary.UserMessageCount,
		&summary.AssistantMessageCount,
		&summary.ToolCallCount,
		&summary.MainModel,
		&summary.DurationSeconds,
		&summary.CacheHitRate,
		&summary.StartedAt,
		&summary.UpdatedAt,
		&summary.ConversationTurnCount,
		&summary.SearchableDocumentRows,
	)
	if err == pgx.ErrNoRows {
		return ReaderResult{}, ErrNotFound
	}
	if err != nil {
		return ReaderResult{}, err
	}

	where := []string{"session_id = $1"}
	args := []any{id}
	if q := strings.TrimSpace(params.Query); q != "" {
		p := appendSQLArg(&args, "%"+q+"%")
		where = append(where, fmt.Sprintf("(user_text ILIKE %[1]s OR assistant_text ILIKE %[1]s)", p))
	}

	var total int64
	countQuery := fmt.Sprintf(`SELECT count(*)::bigint FROM conversation_turns WHERE %s`, strings.Join(where, " AND "))
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return ReaderResult{}, err
	}

	itemArgs := append([]any{}, args...)
	limitPlaceholder := appendSQLArg(&itemArgs, params.Limit)
	offsetPlaceholder := appendSQLArg(&itemArgs, params.Offset)
	query := fmt.Sprintf(`
		SELECT
			turn_index, user_seq, assistant_seq, started_at, ended_at, user_text,
			assistant_text, tool_call_count, tool_result_count, tool_names
		FROM conversation_turns
		WHERE %s
		ORDER BY turn_index ASC
		LIMIT %s OFFSET %s
	`, strings.Join(where, " AND "), limitPlaceholder, offsetPlaceholder)
	rows, err := r.db.Query(ctx, query, itemArgs...)
	if err != nil {
		return ReaderResult{}, err
	}
	defer rows.Close()

	var items []ReaderTurn
	for rows.Next() {
		var item ReaderTurn
		if err := rows.Scan(
			&item.TurnIndex,
			&item.UserSeq,
			&item.AssistantSeq,
			&item.StartedAt,
			&item.EndedAt,
			&item.UserText,
			&item.AssistantText,
			&item.ToolCallCount,
			&item.ToolResultCount,
			&item.ToolNames,
		); err != nil {
			return ReaderResult{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return ReaderResult{}, err
	}

	nextOffset := 0
	if int64(params.Offset+len(items)) < total {
		nextOffset = params.Offset + len(items)
	}
	return ReaderResult{
		Summary:    summary,
		Items:      items,
		Limit:      params.Limit,
		NextOffset: nextOffset,
		Offset:     params.Offset,
		Total:      total,
	}, nil
}

func (r PostgresRepository) Timeline(ctx context.Context, id string, params TimelineParams) (TimelineResult, error) {
	where := []string{"session_id = $1"}
	args := []any{id}
	switch params.Kind {
	case "message":
		where = append(where, "kind = 'message'")
	case "tool":
		where = append(where, "kind = 'tool'")
	}
	if q := strings.TrimSpace(params.Query); q != "" {
		where = append(where, fmt.Sprintf("text ILIKE %s", appendSQLArg(&args, "%"+q+"%")))
	}

	timelineSource := `
		SELECT session_id, seq, occurred_at, 'message' AS kind, role, '' AS tool_name, '' AS status, content_text AS text
		FROM messages
		UNION ALL
		SELECT session_id, seq, occurred_at, 'tool' AS kind, kind AS role, tool_name, status, COALESCE(NULLIF(output_text, ''), input_text) AS text
		FROM tool_events
	`
	var total int64
	countQuery := fmt.Sprintf(`
		SELECT count(*)::bigint
		FROM (%s) timeline
		WHERE %s
	`, timelineSource, strings.Join(where, " AND "))
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return TimelineResult{}, err
	}

	itemArgs := append([]any{}, args...)
	limitPlaceholder := appendSQLArg(&itemArgs, params.Limit)
	offsetPlaceholder := appendSQLArg(&itemArgs, params.Offset)
	query := fmt.Sprintf(`
		SELECT seq, occurred_at, kind, role, tool_name, status, text
		FROM (%s) timeline
		WHERE %s
		ORDER BY seq ASC
		LIMIT %s OFFSET %s
	`, timelineSource, strings.Join(where, " AND "), limitPlaceholder, offsetPlaceholder)

	rows, err := r.db.Query(ctx, query, itemArgs...)
	if err != nil {
		return TimelineResult{}, err
	}
	defer rows.Close()

	var items []TimelineItem
	for rows.Next() {
		var item TimelineItem
		if err := rows.Scan(&item.Seq, &item.OccurredAt, &item.Kind, &item.Role, &item.ToolName, &item.Status, &item.Text); err != nil {
			return TimelineResult{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return TimelineResult{}, err
	}

	nextOffset := 0
	if int64(params.Offset+len(items)) < total {
		nextOffset = params.Offset + len(items)
	}
	return TimelineResult{
		Items:      items,
		Limit:      params.Limit,
		NextOffset: nextOffset,
		Offset:     params.Offset,
		Total:      total,
	}, nil
}

func (r PostgresRepository) RawPath(ctx context.Context, id string) (string, error) {
	var path string
	if err := r.db.QueryRow(ctx, `SELECT raw_file_path FROM sessions WHERE id = $1`, id).Scan(&path); err != nil {
		if err == pgx.ErrNoRows {
			return "", ErrNotFound
		}
		return "", err
	}
	return path, nil
}

func (r PostgresRepository) FullTimeline(ctx context.Context, id string) ([]TimelineItem, error) {
	rows, err := r.db.Query(ctx, `
		SELECT seq, occurred_at, 'message' AS kind, role, '' AS tool_name, '' AS status, content_text AS text
		FROM messages
		WHERE session_id = $1
		UNION ALL
		SELECT seq, occurred_at, 'tool' AS kind, kind AS role, tool_name, status, COALESCE(NULLIF(output_text, ''), input_text) AS text
		FROM tool_events
		WHERE session_id = $1
		ORDER BY seq ASC
	`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []TimelineItem
	for rows.Next() {
		var item TimelineItem
		if err := rows.Scan(&item.Seq, &item.OccurredAt, &item.Kind, &item.Role, &item.ToolName, &item.Status, &item.Text); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r PostgresRepository) countList(ctx context.Context, where []string, args []any) (int64, error) {
	var total int64
	countQuery := fmt.Sprintf(`
		SELECT count(*)::bigint
		FROM sessions
		JOIN devices ON devices.id = sessions.device_id
		LEFT JOIN repositories ON repositories.id = sessions.repository_id
		LEFT JOIN projects ON projects.id = sessions.project_id
		LEFT JOIN session_summaries ON session_summaries.session_id = sessions.id
		WHERE %s
	`, strings.Join(where, " AND "))
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	return total, err
}

func (r PostgresRepository) loadListTotals(ctx context.Context, where []string, args []any) (UsageTotals, error) {
	var out UsageTotals
	query := fmt.Sprintf(`
		SELECT
			count(sessions.id)::bigint,
			COALESCE(sum(session_summaries.input_tokens), 0)::bigint,
			COALESCE(sum(session_summaries.cached_input_tokens), 0)::bigint,
			COALESCE(sum(session_summaries.output_tokens), 0)::bigint,
			COALESCE(sum(session_summaries.reasoning_output_tokens), 0)::bigint,
			COALESCE(sum(session_summaries.total_tokens), 0)::bigint,
			COALESCE(sum(session_summaries.cost_usd), 0)::float8,
			COALESCE(sum(session_summaries.message_count), 0)::bigint,
			COALESCE(sum(session_summaries.tool_call_count), 0)::bigint,
			COALESCE(sum(session_summaries.patch_added_lines), 0)::bigint
		FROM sessions
		JOIN devices ON devices.id = sessions.device_id
		LEFT JOIN repositories ON repositories.id = sessions.repository_id
		LEFT JOIN projects ON projects.id = sessions.project_id
		LEFT JOIN session_summaries ON session_summaries.session_id = sessions.id
		WHERE %s
	`, strings.Join(where, " AND "))
	err := r.db.QueryRow(ctx, query, args...).Scan(
		&out.Sessions,
		&out.InputTokens,
		&out.CachedInputTokens,
		&out.OutputTokens,
		&out.ReasoningOutputTokens,
		&out.TotalTokens,
		&out.CostUSD,
		&out.Messages,
		&out.ToolCalls,
		&out.PatchAddedLines,
	)
	return out, err
}

func (r PostgresRepository) loadListItems(ctx context.Context, where []string, itemArgs []any, orderBy string, limitPlaceholder string, offsetPlaceholder string) ([]ListItem, error) {
	query := fmt.Sprintf(`
		SELECT
			sessions.id,
			sessions.started_at,
			sessions.updated_at,
			sessions.cwd,
			sessions.branch,
			COALESCE(repositories.repository_name, ''),
			COALESCE(repositories.repository_url, ''),
			COALESCE(projects.display_name, ''),
			devices.name,
			COALESCE(session_summaries.title, ''),
			COALESCE(session_summaries.display_title, ''),
			COALESCE(session_summaries.display_subtitle, ''),
			COALESCE(session_summaries.user_intent, ''),
			COALESCE(session_summaries.dominant_language, ''),
			COALESCE(session_summaries.first_user_message, ''),
			COALESCE(session_summaries.last_user_message, ''),
			COALESCE(session_summaries.short_summary, ''),
			COALESCE(session_summaries.main_model, ''),
			COALESCE(session_summaries.duration_seconds, 0)::bigint,
			COALESCE(session_summaries.cache_hit_rate, 0)::float8,
			COALESCE(session_summaries.conversation_turn_count, 0)::bigint,
			COALESCE(session_summaries.searchable_message_count, 0)::bigint,
			COALESCE(session_summaries.searchable_tool_count, 0)::bigint,
			COALESCE(session_summaries.input_tokens, 0)::bigint,
			COALESCE(session_summaries.cached_input_tokens, 0)::bigint,
			COALESCE(session_summaries.output_tokens, 0)::bigint,
			COALESCE(session_summaries.reasoning_output_tokens, 0)::bigint,
			COALESCE(session_summaries.total_tokens, 0)::bigint,
			COALESCE(session_summaries.cost_usd, 0)::float8,
			COALESCE(session_summaries.models, ''),
			COALESCE(session_summaries.message_count, 0)::bigint,
			COALESCE(session_summaries.user_message_count, 0)::bigint,
			COALESCE(session_summaries.assistant_message_count, 0)::bigint,
			COALESCE(session_summaries.tool_call_count, 0)::bigint,
			COALESCE(session_summaries.patch_added_lines, 0)::bigint
		FROM sessions
		JOIN devices ON devices.id = sessions.device_id
		LEFT JOIN repositories ON repositories.id = sessions.repository_id
		LEFT JOIN projects ON projects.id = sessions.project_id
		LEFT JOIN session_summaries ON session_summaries.session_id = sessions.id
		WHERE %s
		ORDER BY %s, sessions.id DESC
		LIMIT %s OFFSET %s
	`, strings.Join(where, " AND "), orderBy, limitPlaceholder, offsetPlaceholder)

	rows, err := r.db.Query(ctx, query, itemArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []ListItem
	for rows.Next() {
		var item ListItem
		if err := rows.Scan(
			&item.ID,
			&item.StartedAt,
			&item.UpdatedAt,
			&item.CWD,
			&item.Branch,
			&item.Repository,
			&item.RepositoryURL,
			&item.Project,
			&item.Device,
			&item.Title,
			&item.DisplayTitle,
			&item.DisplaySubtitle,
			&item.UserIntent,
			&item.DominantLanguage,
			&item.FirstUserMessage,
			&item.LastUserMessage,
			&item.ShortSummary,
			&item.MainModel,
			&item.DurationSeconds,
			&item.CacheHitRate,
			&item.ConversationTurns,
			&item.SearchableMessages,
			&item.SearchableTools,
			&item.InputTokens,
			&item.CachedTokens,
			&item.OutputTokens,
			&item.Reasoning,
			&item.TotalTokens,
			&item.CostUSD,
			&item.Models,
			&item.MessageCount,
			&item.UserMessageCount,
			&item.AssistantMessageCount,
			&item.ToolCallCount,
			&item.PatchAddedLines,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r PostgresRepository) loadModels(ctx context.Context, id string) ([]ModelUsage, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			model,
			COALESCE(sum(input_tokens), 0)::bigint,
			COALESCE(sum(cached_input_tokens), 0)::bigint,
			COALESCE(sum(output_tokens), 0)::bigint,
			COALESCE(sum(reasoning_output_tokens), 0)::bigint,
			COALESCE(sum(total_tokens), 0)::bigint,
			COALESCE(sum(cost_usd), 0)::float8
		FROM usage_events
		WHERE session_id = $1
		GROUP BY model
		ORDER BY COALESCE(sum(total_tokens), 0) DESC
	`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ModelUsage
	for rows.Next() {
		var item ModelUsage
		if err := rows.Scan(
			&item.Model,
			&item.InputTokens,
			&item.CachedInputTokens,
			&item.OutputTokens,
			&item.ReasoningOutputTokens,
			&item.TotalTokens,
			&item.CostUSD,
		); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func buildListFilters(params ListParams) ([]string, []any) {
	where := []string{"1=1"}
	var args []any
	addDateFilters(params, &where, &args, "COALESCE(sessions.updated_at, sessions.started_at, sessions.ingested_at)")
	addIDFilter(params.DeviceID, &where, &args, "sessions.device_id")
	addIDFilter(params.RepositoryID, &where, &args, "sessions.repository_id")
	addIDFilter(params.ProjectID, &where, &args, "sessions.project_id")
	addTextFilter(params.Branch, &where, &args, "sessions.branch")
	if params.OnlyWithInputTokens {
		where = append(where, "COALESCE(session_summaries.input_tokens, 0) > 0")
	}
	if model := strings.TrimSpace(params.Model); model != "" {
		where = append(where, fmt.Sprintf(`EXISTS (
			SELECT 1 FROM usage_events filtered_usage
			WHERE filtered_usage.session_id = sessions.id AND filtered_usage.model = %s
		)`, appendSQLArg(&args, model)))
	}
	if q := strings.TrimSpace(params.Query); q != "" {
		p := appendSQLArg(&args, "%"+q+"%")
		where = append(where, fmt.Sprintf(`(
			sessions.id ILIKE %[1]s OR sessions.cwd ILIKE %[1]s OR sessions.branch ILIKE %[1]s OR
			COALESCE(repositories.repository_name, '') ILIKE %[1]s OR
			COALESCE(repositories.repository_url, '') ILIKE %[1]s OR
			COALESCE(projects.display_name, '') ILIKE %[1]s OR
			COALESCE(session_summaries.title, '') ILIKE %[1]s OR
			COALESCE(session_summaries.display_title, '') ILIKE %[1]s OR
			COALESCE(session_summaries.user_intent, '') ILIKE %[1]s OR
			COALESCE(session_summaries.short_summary, '') ILIKE %[1]s OR
			COALESCE(session_summaries.first_user_message, '') ILIKE %[1]s OR
			COALESCE(session_summaries.last_user_message, '') ILIKE %[1]s
		)`, p))
	}
	return where, args
}

func addDateFilters(params ListParams, where *[]string, args *[]any, column string) {
	if params.From != "" {
		*where = append(*where, fmt.Sprintf("%s >= %s::date", column, appendSQLArg(args, params.From)))
	}
	if params.To != "" {
		*where = append(*where, fmt.Sprintf("%s < (%s::date + INTERVAL '1 day')", column, appendSQLArg(args, params.To)))
	}
}

func addIDFilter(value string, where *[]string, args *[]any, column string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	if !isUUIDText(value) {
		*where = append(*where, "1=0")
		return
	}
	*where = append(*where, fmt.Sprintf("%s = %s::uuid", column, appendSQLArg(args, value)))
}

func addTextFilter(value string, where *[]string, args *[]any, column string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	*where = append(*where, fmt.Sprintf("%s = %s", column, appendSQLArg(args, value)))
}

func appendSQLArg(args *[]any, value any) string {
	*args = append(*args, value)
	return fmt.Sprintf("$%d", len(*args))
}

func isUUIDText(value string) bool {
	if len(value) != 36 {
		return false
	}
	for i, r := range value {
		switch i {
		case 8, 13, 18, 23:
			if r != '-' {
				return false
			}
		default:
			if !(r >= '0' && r <= '9') && !(r >= 'a' && r <= 'f') && !(r >= 'A' && r <= 'F') {
				return false
			}
		}
	}
	return true
}

func sessionOrderBy(sort string) (string, bool) {
	switch sort {
	case "", "updated_desc":
		return "COALESCE(sessions.updated_at, sessions.started_at, sessions.ingested_at) DESC", true
	case "updated_asc":
		return "COALESCE(sessions.updated_at, sessions.started_at, sessions.ingested_at) ASC", true
	case "started_desc":
		return "COALESCE(sessions.started_at, sessions.updated_at, sessions.ingested_at) DESC", true
	case "tokens_desc":
		return "COALESCE(session_summaries.total_tokens, 0) DESC", true
	case "tokens_asc":
		return "COALESCE(session_summaries.total_tokens, 0) ASC", true
	case "cost_desc":
		return "COALESCE(session_summaries.cost_usd, 0) DESC", true
	case "cost_asc":
		return "COALESCE(session_summaries.cost_usd, 0) ASC", true
	case "tool_calls_desc":
		return "COALESCE(session_summaries.tool_call_count, 0) DESC", true
	default:
		return "", false
	}
}
