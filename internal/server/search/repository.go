package search

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) PostgresRepository {
	return PostgresRepository{db: db}
}

func (r PostgresRepository) Search(ctx context.Context, params Params) (SearchResult, error) {
	where := []string{"1=1"}
	var args []any
	searchArg := appendSQLArg(&args, params.Query)
	where = append(where, fmt.Sprintf(`(
		search_documents.search_vector @@ plainto_tsquery('simple', %[1]s)
		OR search_documents.title ILIKE '%%' || %[1]s || '%%'
		OR search_documents.body ILIKE '%%' || %[1]s || '%%'
	)`, searchArg))
	addDateFilters(params, &where, &args, "search_documents.occurred_at")
	addIDFilter(params.DeviceID, &where, &args, "sessions.device_id")
	addIDFilter(params.RepositoryID, &where, &args, "sessions.repository_id")
	addIDFilter(params.ProjectID, &where, &args, "sessions.project_id")
	switch params.Kind {
	case "", "message":
		where = append(where, "search_documents.kind = 'message'")
	case "user", "assistant":
		where = append(where, "search_documents.kind = 'message'")
		where = append(where, fmt.Sprintf("search_documents.document_scope = %s", appendSQLArg(&args, params.Kind)))
	case "tool":
		where = append(where, "search_documents.kind = 'tool'")
	case "all":
	}
	if model := strings.TrimSpace(params.Model); model != "" {
		where = append(where, fmt.Sprintf(`EXISTS (
			SELECT 1 FROM usage_events filtered_usage
			WHERE filtered_usage.session_id = sessions.id AND filtered_usage.model = %s
		)`, appendSQLArg(&args, model)))
	}

	total := int64(-1)
	if params.IncludeTotal {
		countQuery := fmt.Sprintf(`
			SELECT count(*)::bigint
			FROM search_documents
			JOIN sessions ON sessions.id = search_documents.session_id
			LEFT JOIN repositories ON repositories.id = sessions.repository_id
			LEFT JOIN projects ON projects.id = sessions.project_id
			LEFT JOIN session_summaries ON session_summaries.session_id = sessions.id
			WHERE %s
		`, strings.Join(where, " AND "))
		if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
			return SearchResult{}, err
		}
	}

	itemArgs := append([]any{}, args...)
	fetchLimit := params.Limit
	if !params.IncludeTotal {
		fetchLimit = params.Limit + 1
	}
	limitPlaceholder := appendSQLArg(&itemArgs, fetchLimit)
	offsetPlaceholder := appendSQLArg(&itemArgs, params.Offset)
	query := fmt.Sprintf(`
		SELECT
			search_documents.kind,
			search_documents.document_scope,
			search_documents.session_id,
			search_documents.seq,
			search_documents.turn_index,
			search_documents.occurred_at,
			search_documents.role,
			search_documents.tool_name,
			search_documents.title,
			search_documents.body,
			search_documents.snippet,
			search_documents.rank_weight,
			search_documents.default_searchable,
			sessions.cwd,
			sessions.branch,
			COALESCE(repositories.repository_name, ''),
			COALESCE(repositories.repository_url, ''),
			COALESCE(projects.display_name, ''),
			COALESCE(session_summaries.title, ''),
			COALESCE(session_summaries.short_summary, '')
		FROM search_documents
		JOIN sessions ON sessions.id = search_documents.session_id
		LEFT JOIN repositories ON repositories.id = sessions.repository_id
		LEFT JOIN projects ON projects.id = sessions.project_id
		LEFT JOIN session_summaries ON session_summaries.session_id = sessions.id
		WHERE %s
		ORDER BY
			search_documents.rank_weight DESC,
			ts_rank(search_documents.search_vector, plainto_tsquery('simple', %s)) DESC,
			search_documents.occurred_at DESC NULLS LAST,
			search_documents.session_id,
			search_documents.seq
		LIMIT %s OFFSET %s
	`, strings.Join(where, " AND "), searchArg, limitPlaceholder, offsetPlaceholder)
	rows, err := r.db.Query(ctx, query, itemArgs...)
	if err != nil {
		return SearchResult{}, err
	}
	defer rows.Close()

	var out []Result
	for rows.Next() {
		var item Result
		if err := rows.Scan(
			&item.Kind,
			&item.DocumentScope,
			&item.SessionID,
			&item.Seq,
			&item.TurnIndex,
			&item.OccurredAt,
			&item.Role,
			&item.ToolName,
			&item.Title,
			&item.Text,
			&item.Snippet,
			&item.RankWeight,
			&item.DefaultSearch,
			&item.CWD,
			&item.Branch,
			&item.Repository,
			&item.RepoURL,
			&item.Project,
			&item.SessionTitle,
			&item.SessionSummary,
		); err != nil {
			return SearchResult{}, err
		}
		item.Snippet, item.MatchStart, item.MatchEnd = buildSearchSnippet(firstNonEmptyString(item.Snippet, item.Text), params.Query, 700)
		item.Text = truncate(item.Text, 1000)
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return SearchResult{}, err
	}

	hasMore := false
	if !params.IncludeTotal && len(out) > params.Limit {
		hasMore = true
		out = out[:params.Limit]
	}
	nextOffset := 0
	switch {
	case params.IncludeTotal && int64(params.Offset+len(out)) < total:
		nextOffset = params.Offset + len(out)
	case !params.IncludeTotal && hasMore:
		nextOffset = params.Offset + len(out)
	}
	if !params.IncludeTotal {
		total = int64(params.Offset + len(out))
		if hasMore {
			total++
		}
	}
	return SearchResult{
		Items:      out,
		Limit:      params.Limit,
		NextOffset: nextOffset,
		Offset:     params.Offset,
		Total:      total,
		TotalKnown: params.IncludeTotal,
	}, nil
}

func addDateFilters(params Params, where *[]string, args *[]any, column string) {
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

func buildSearchSnippet(text string, query string, maxLength int) (string, int, int) {
	if maxLength <= 0 || len(text) <= maxLength {
		start := strings.Index(strings.ToLower(text), strings.ToLower(query))
		if start < 0 {
			return truncate(text, maxLength), -1, -1
		}
		return text, start, start + len(query)
	}

	lowerText := strings.ToLower(text)
	lowerQuery := strings.ToLower(query)
	match := strings.Index(lowerText, lowerQuery)
	if match < 0 {
		return truncate(text, maxLength), -1, -1
	}

	start := match - maxLength/3
	if start < 0 {
		start = 0
	}
	end := start + maxLength
	if end > len(text) {
		end = len(text)
		start = end - maxLength
		if start < 0 {
			start = 0
		}
	}
	snippet := text[start:end]
	prefix := ""
	suffix := ""
	if start > 0 {
		prefix = "..."
	}
	if end < len(text) {
		suffix = "..."
	}
	matchStart := match - start + len(prefix)
	matchEnd := matchStart + len(query)
	return prefix + snippet + suffix, matchStart, matchEnd
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func truncate(value string, max int) string {
	value = strings.TrimSpace(value)
	if len(value) <= max {
		return value
	}
	return value[:max] + "..."
}
