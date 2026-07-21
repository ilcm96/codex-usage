package projections

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ilcm96/codex-usage/internal/server/sessionparse"
)

type derivedTurn struct {
	TurnIndex       int
	UserSeq         int
	AssistantSeq    int
	StartedAt       *time.Time
	EndedAt         *time.Time
	UserText        string
	AssistantText   string
	ToolCallCount   int64
	ToolResultCount int64
	ToolNames       []string
	StartSeq        int
	EndSeq          int
}

func RefreshSession(ctx context.Context, tx pgx.Tx, parsed sessionparse.Session, startedAt *time.Time, updatedAt *time.Time) error {
	if err := RefreshSessionWithoutUsageRollups(ctx, tx, parsed, startedAt, updatedAt); err != nil {
		return err
	}
	return RefreshRollupsForSessions(ctx, tx, []string{parsed.Meta.ID})
}

func RefreshRollupsForSessions(ctx context.Context, tx pgx.Tx, sessionIDs []string) error {
	if err := RefreshUsageRollupsForSessions(ctx, tx, sessionIDs); err != nil {
		return err
	}
	return RefreshSessionRollupsForSessions(ctx, tx, sessionIDs)
}

func RefreshSessionWithoutUsageRollups(ctx context.Context, tx pgx.Tx, parsed sessionparse.Session, startedAt *time.Time, updatedAt *time.Time) error {
	turns := buildConversationTurns(parsed)
	if err := insertSessionSummary(ctx, tx, parsed, turns, startedAt, updatedAt); err != nil {
		return err
	}
	if err := insertConversationTurns(ctx, tx, parsed.Meta.ID, turns); err != nil {
		return err
	}
	if err := insertSearchDocuments(ctx, tx, parsed, turns); err != nil {
		return err
	}
	return nil
}

func insertSessionSummary(ctx context.Context, tx pgx.Tx, parsed sessionparse.Session, turns []derivedTurn, startedAt *time.Time, updatedAt *time.Time) error {
	var firstUser, lastUser, lastAssistant string
	var userCount, assistantCount int64
	for _, msg := range parsed.Messages {
		switch msg.Role {
		case "user":
			userCount++
			cleaned := userFacingText(msg.ContentText)
			if cleaned == "" {
				continue
			}
			if firstUser == "" {
				firstUser = cleaned
			}
			lastUser = cleaned
		case "assistant":
			assistantCount++
			if cleaned := userFacingText(msg.ContentText); cleaned != "" {
				lastAssistant = cleaned
			}
		}
	}

	var inputTokens, cachedInputTokens, cacheWriteInputTokens, outputTokens, reasoningOutputTokens, totalTokens int64
	var costUSD float64
	modelTotals := map[string]int64{}
	for _, usage := range parsed.Usages {
		modelTotals[usage.Model] += usage.TotalTokens
		inputTokens += usage.InputTokens
		cachedInputTokens += usage.CachedInputTokens
		cacheWriteInputTokens += usage.CacheWriteInputTokens
		outputTokens += usage.OutputTokens
		reasoningOutputTokens += usage.ReasoningOutputTokens
		totalTokens += usage.TotalTokens
		costUSD += usage.CostUSD
	}
	mainModel := ""
	var mainModelTokens int64
	for model, tokens := range modelTotals {
		if tokens > mainModelTokens {
			mainModel = model
			mainModelTokens = tokens
		}
	}
	models := make([]string, 0, len(modelTotals))
	for model := range modelTotals {
		if strings.TrimSpace(model) != "" {
			models = append(models, model)
		}
	}
	sort.Strings(models)

	durationSeconds := int64(0)
	if startedAt != nil && updatedAt != nil && updatedAt.After(*startedAt) {
		durationSeconds = int64(updatedAt.Sub(*startedAt).Seconds())
	}

	title := deriveSessionTitle(firstUser, parsed.Meta.CWD)
	shortSummary := deriveSessionSummary(firstUser, lastUser, lastAssistant)
	displayTitle := title
	userIntent := deriveUserIntent(firstUser, lastUser)
	displaySubtitle := deriveDisplaySubtitle(parsed.Meta.CWD, parsed.Meta.Branch, mainModel)
	patchStats := collectPatchStats(parsed.Tools)
	dominantLanguage := patchStats.DominantLanguage
	if dominantLanguage == "" {
		dominantLanguage = detectDominantLanguage(firstUser + "\n" + lastUser + "\n" + lastAssistant)
	}
	patchLanguageStats, err := json.Marshal(patchStats.LanguageLines)
	if err != nil {
		return err
	}
	cacheHitRate := 0.0
	if inputTokens > 0 {
		cacheHitRate = float64(cachedInputTokens) / float64(inputTokens)
	}
	searchableMessageCount := int64(0)
	searchableToolCount := int64(0)
	for _, msg := range parsed.Messages {
		if userFacingText(msg.ContentText) != "" {
			searchableMessageCount++
		}
	}
	for _, tool := range parsed.Tools {
		if strings.TrimSpace(joinText(tool.InputText, tool.OutputText)) != "" {
			searchableToolCount++
		}
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO session_summaries (
			session_id, title, display_title, display_subtitle, user_intent, dominant_language,
			first_user_message, last_user_message, short_summary,
			message_count, user_message_count, assistant_message_count, tool_call_count, patch_added_lines,
			patch_language_stats, conversation_turn_count, searchable_message_count, searchable_tool_count,
			main_model, models, input_tokens, cached_input_tokens, cache_write_input_tokens, output_tokens,
			reasoning_output_tokens, total_tokens, cost_usd,
			duration_seconds, cache_hit_rate, started_at, updated_at, generated_at
		)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17, $18, $19, $20,
			$21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31, now()
		)
	`, parsed.Meta.ID, sanitizeDBText(title), sanitizeDBText(displayTitle), sanitizeDBText(displaySubtitle),
		sanitizeDBText(userIntent), sanitizeDBText(dominantLanguage), sanitizeDBText(truncate(firstUser, 4000)),
		sanitizeDBText(truncate(lastUser, 4000)), sanitizeDBText(shortSummary),
		int64(len(parsed.Messages)), userCount, assistantCount, int64(len(parsed.Tools)),
		patchStats.AddedLines, patchLanguageStats, int64(len(turns)), searchableMessageCount, searchableToolCount,
		mainModel, strings.Join(models, ", "), inputTokens, cachedInputTokens, cacheWriteInputTokens, outputTokens,
		reasoningOutputTokens, totalTokens, costUSD,
		durationSeconds, cacheHitRate, nullableTime(startedAt), nullableTime(updatedAt))
	return err
}

func buildConversationTurns(parsed sessionparse.Session) []derivedTurn {
	var turns []derivedTurn
	var cur *derivedTurn
	maxSeq := 0
	for _, ev := range parsed.Events {
		if ev.Seq > maxSeq {
			maxSeq = ev.Seq
		}
	}

	flush := func(endSeq int) {
		if cur == nil {
			return
		}
		cur.EndSeq = endSeq
		for _, tool := range parsed.Tools {
			if tool.Seq < cur.StartSeq || tool.Seq > cur.EndSeq {
				continue
			}
			if strings.Contains(tool.Kind, "output") {
				cur.ToolResultCount++
			} else {
				cur.ToolCallCount++
			}
			if tool.ToolName != "" {
				cur.ToolNames = append(cur.ToolNames, tool.ToolName)
			}
			if cur.EndedAt == nil && tool.OccurredAt != nil {
				t := *tool.OccurredAt
				cur.EndedAt = &t
			}
		}
		cur.ToolNames = uniqueStrings(cur.ToolNames...)
		if cur.EndedAt == nil {
			cur.EndedAt = cur.StartedAt
		}
		turns = append(turns, *cur)
		cur = nil
	}

	for _, msg := range parsed.Messages {
		switch msg.Role {
		case "user":
			userText := userFacingText(msg.ContentText)
			if userText == "" {
				continue
			}
			if cur != nil {
				flush(msg.Seq - 1)
			}
			t := derivedTurn{
				TurnIndex: len(turns),
				UserSeq:   msg.Seq,
				StartedAt: msg.OccurredAt,
				UserText:  userText,
				StartSeq:  msg.Seq,
			}
			cur = &t
		case "assistant":
			if cur == nil {
				continue
			}
			cur.AssistantSeq = msg.Seq
			cur.AssistantText = joinText(cur.AssistantText, msg.ContentText)
			cur.EndedAt = msg.OccurredAt
		}
	}
	if cur != nil {
		flush(maxSeq)
	}
	return turns
}

func insertConversationTurns(ctx context.Context, tx pgx.Tx, sessionID string, turns []derivedTurn) error {
	if len(turns) == 0 {
		return nil
	}
	rows := make([][]any, 0, len(turns))
	for _, turn := range turns {
		rows = append(rows, []any{
			sessionID,
			turn.TurnIndex,
			nullableInt(turn.UserSeq),
			nullableInt(turn.AssistantSeq),
			nullableTime(turn.StartedAt),
			nullableTime(turn.EndedAt),
			sanitizeDBText(turn.UserText),
			sanitizeDBText(turn.AssistantText),
			turn.ToolCallCount,
			turn.ToolResultCount,
			turn.ToolNames,
		})
	}
	if _, err := tx.CopyFrom(ctx, pgx.Identifier{"conversation_turns"}, []string{
		"session_id", "turn_index", "user_seq", "assistant_seq", "started_at", "ended_at",
		"user_text", "assistant_text", "tool_call_count", "tool_result_count", "tool_names",
	}, pgx.CopyFromRows(rows)); err != nil {
		return fmt.Errorf("insert conversation turns: %w", err)
	}
	return nil
}

func insertSearchDocuments(ctx context.Context, tx pgx.Tx, parsed sessionparse.Session, turns []derivedTurn) error {
	turnBySeq := make(map[int]int, len(parsed.Events))
	for _, turn := range turns {
		for seq := turn.StartSeq; seq <= turn.EndSeq; seq++ {
			turnBySeq[seq] = turn.TurnIndex
		}
	}

	rows := make([][]any, 0, len(parsed.Messages)+len(parsed.Tools))
	for _, msg := range parsed.Messages {
		body := userFacingText(msg.ContentText)
		if body == "" {
			continue
		}
		turnIndex, ok := turnBySeq[msg.Seq]
		var turnArg any
		if ok {
			turnArg = turnIndex
		}
		documentScope := "assistant"
		rankWeight := 70
		if msg.Role == "user" {
			documentScope = "user"
			rankWeight = 110
		}
		rows = append(rows, []any{
			parsed.Meta.ID,
			msg.Seq,
			turnArg,
			nullableTime(msg.OccurredAt),
			"message",
			sanitizeDBText(documentScope),
			sanitizeDBText(msg.Role),
			"",
			sanitizeDBText(titleWord(msg.Role)),
			sanitizeDBText(body),
			sanitizeDBText(truncate(body, 700)),
			rankWeight,
			true,
		})
	}

	for _, tool := range parsed.Tools {
		body := strings.TrimSpace(joinText(tool.InputText, tool.OutputText))
		if body == "" {
			continue
		}
		turnIndex, ok := turnBySeq[tool.Seq]
		var turnArg any
		if ok {
			turnArg = turnIndex
		}
		title := firstNonEmpty(tool.ToolName, tool.Kind, "tool")
		rows = append(rows, []any{
			parsed.Meta.ID,
			tool.Seq,
			turnArg,
			nullableTime(tool.OccurredAt),
			"tool",
			"tool",
			sanitizeDBText(tool.Kind),
			sanitizeDBText(tool.ToolName),
			sanitizeDBText(title),
			sanitizeDBText(body),
			sanitizeDBText(truncate(body, 700)),
			15,
			false,
		})
	}
	if len(rows) == 0 {
		return nil
	}
	if _, err := tx.CopyFrom(ctx, pgx.Identifier{"search_documents"}, []string{
		"session_id", "seq", "turn_index", "occurred_at", "kind", "document_scope", "role", "tool_name",
		"title", "body", "snippet", "rank_weight", "default_searchable",
	}, pgx.CopyFromRows(rows)); err != nil {
		return fmt.Errorf("insert search documents: %w", err)
	}
	return nil
}

func RefreshUsageRollupsForSessions(ctx context.Context, tx pgx.Tx, sessionIDs []string) error {
	if len(sessionIDs) == 0 {
		return nil
	}
	dates, err := UsageRollupDatesForSessions(ctx, tx, sessionIDs)
	if err != nil {
		return err
	}
	return RebuildUsageRollupsForDates(ctx, tx, dates)
}

func RefreshSessionRollupsForSessions(ctx context.Context, tx pgx.Tx, sessionIDs []string) error {
	if len(sessionIDs) == 0 {
		return nil
	}
	dates, err := SessionRollupDatesForSessions(ctx, tx, sessionIDs)
	if err != nil {
		return err
	}
	return RebuildSessionRollupsForDates(ctx, tx, dates)
}

func UsageRollupDatesForSessions(ctx context.Context, tx pgx.Tx, sessionIDs []string) ([]time.Time, error) {
	if len(sessionIDs) == 0 {
		return nil, nil
	}
	rows, err := tx.Query(ctx, `
		SELECT DISTINCT occurred_at::date
		FROM usage_events
		WHERE session_id = ANY($1)
		  AND occurred_at IS NOT NULL
	`, sessionIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dates []time.Time
	for rows.Next() {
		var date time.Time
		if err := rows.Scan(&date); err != nil {
			return nil, err
		}
		dates = append(dates, date)
	}
	return dates, rows.Err()
}

func SessionRollupDatesForSessions(ctx context.Context, tx pgx.Tx, sessionIDs []string) ([]time.Time, error) {
	if len(sessionIDs) == 0 {
		return nil, nil
	}
	rows, err := tx.Query(ctx, `
		SELECT DISTINCT COALESCE(session_summaries.updated_at, session_summaries.started_at, sessions.updated_at, sessions.started_at, sessions.ingested_at)::date
		FROM sessions
		LEFT JOIN session_summaries ON session_summaries.session_id = sessions.id
		WHERE sessions.id = ANY($1)
	`, sessionIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dates []time.Time
	for rows.Next() {
		var date time.Time
		if err := rows.Scan(&date); err != nil {
			return nil, err
		}
		dates = append(dates, date)
	}
	return dates, rows.Err()
}

func AddUsageRollupsForSessions(ctx context.Context, tx pgx.Tx, sessionIDs []string) error {
	if len(sessionIDs) == 0 {
		return nil
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO usage_rollups (
			bucket_date, bucket_month, device_id, repository_id, project_id, model,
			input_tokens, cached_input_tokens, cache_write_input_tokens, output_tokens, reasoning_output_tokens, total_tokens, cost_usd
		)
		SELECT
			usage_events.occurred_at::date AS bucket_date,
			date_trunc('month', usage_events.occurred_at)::date AS bucket_month,
			sessions.device_id,
			sessions.repository_id,
			sessions.project_id,
			usage_events.model,
			sum(usage_events.input_tokens)::bigint,
			sum(usage_events.cached_input_tokens)::bigint,
			sum(usage_events.cache_write_input_tokens)::bigint,
			sum(usage_events.output_tokens)::bigint,
			sum(usage_events.reasoning_output_tokens)::bigint,
			sum(usage_events.total_tokens)::bigint,
			sum(usage_events.cost_usd)
		FROM usage_events
		JOIN sessions ON sessions.id = usage_events.session_id
		WHERE usage_events.occurred_at IS NOT NULL
		  AND usage_events.session_id = ANY($1)
		GROUP BY bucket_date, bucket_month, sessions.device_id, sessions.repository_id, sessions.project_id, usage_events.model
		ON CONFLICT (bucket_date, device_id, repository_id, project_id, model)
		DO UPDATE SET
			input_tokens = usage_rollups.input_tokens + EXCLUDED.input_tokens,
			cached_input_tokens = usage_rollups.cached_input_tokens + EXCLUDED.cached_input_tokens,
			cache_write_input_tokens = usage_rollups.cache_write_input_tokens + EXCLUDED.cache_write_input_tokens,
			output_tokens = usage_rollups.output_tokens + EXCLUDED.output_tokens,
			reasoning_output_tokens = usage_rollups.reasoning_output_tokens + EXCLUDED.reasoning_output_tokens,
			total_tokens = usage_rollups.total_tokens + EXCLUDED.total_tokens,
			cost_usd = usage_rollups.cost_usd + EXCLUDED.cost_usd,
			updated_at = now()
	`, sessionIDs)
	return err
}

func AddSessionRollupsForSessions(ctx context.Context, tx pgx.Tx, sessionIDs []string) error {
	if len(sessionIDs) == 0 {
		return nil
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO session_rollups (
			bucket_date, bucket_month, device_id, repository_id, project_id, session_count,
			input_tokens, cached_input_tokens, cache_write_input_tokens, output_tokens, reasoning_output_tokens,
			total_tokens, cost_usd, patch_added_lines
		)
		SELECT
			COALESCE(session_summaries.updated_at, session_summaries.started_at, sessions.updated_at, sessions.started_at, sessions.ingested_at)::date AS bucket_date,
			date_trunc('month', COALESCE(session_summaries.updated_at, session_summaries.started_at, sessions.updated_at, sessions.started_at, sessions.ingested_at))::date AS bucket_month,
			sessions.device_id,
			sessions.repository_id,
			sessions.project_id,
			count(sessions.id)::bigint,
			COALESCE(sum(session_summaries.input_tokens), 0)::bigint,
			COALESCE(sum(session_summaries.cached_input_tokens), 0)::bigint,
			COALESCE(sum(session_summaries.cache_write_input_tokens), 0)::bigint,
			COALESCE(sum(session_summaries.output_tokens), 0)::bigint,
			COALESCE(sum(session_summaries.reasoning_output_tokens), 0)::bigint,
			COALESCE(sum(session_summaries.total_tokens), 0)::bigint,
			COALESCE(sum(session_summaries.cost_usd), 0),
			COALESCE(sum(session_summaries.patch_added_lines), 0)::bigint
		FROM sessions
		LEFT JOIN session_summaries ON session_summaries.session_id = sessions.id
		WHERE sessions.id = ANY($1)
		GROUP BY bucket_date, bucket_month, sessions.device_id, sessions.repository_id, sessions.project_id
		ON CONFLICT (bucket_date, device_id, repository_id, project_id)
		DO UPDATE SET
			session_count = session_rollups.session_count + EXCLUDED.session_count,
			input_tokens = session_rollups.input_tokens + EXCLUDED.input_tokens,
			cached_input_tokens = session_rollups.cached_input_tokens + EXCLUDED.cached_input_tokens,
			cache_write_input_tokens = session_rollups.cache_write_input_tokens + EXCLUDED.cache_write_input_tokens,
			output_tokens = session_rollups.output_tokens + EXCLUDED.output_tokens,
			reasoning_output_tokens = session_rollups.reasoning_output_tokens + EXCLUDED.reasoning_output_tokens,
			total_tokens = session_rollups.total_tokens + EXCLUDED.total_tokens,
			cost_usd = session_rollups.cost_usd + EXCLUDED.cost_usd,
			patch_added_lines = session_rollups.patch_added_lines + EXCLUDED.patch_added_lines,
			updated_at = now()
	`, sessionIDs)
	return err
}

func RebuildUsageRollupsForDates(ctx context.Context, tx pgx.Tx, dates []time.Time) error {
	dates = uniqueDates(dates)
	if len(dates) == 0 {
		return nil
	}
	dateValues := dateStrings(dates)
	if _, err := tx.Exec(ctx, `
		DELETE FROM usage_rollups
		WHERE bucket_date = ANY($1::date[])
	`, dateValues); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO usage_rollups (
			bucket_date, bucket_month, device_id, repository_id, project_id, model,
			input_tokens, cached_input_tokens, cache_write_input_tokens, output_tokens, reasoning_output_tokens, total_tokens, cost_usd
		)
		SELECT
			usage_events.occurred_at::date AS bucket_date,
			date_trunc('month', usage_events.occurred_at)::date AS bucket_month,
			sessions.device_id,
			sessions.repository_id,
			sessions.project_id,
			usage_events.model,
			sum(usage_events.input_tokens)::bigint,
			sum(usage_events.cached_input_tokens)::bigint,
			sum(usage_events.cache_write_input_tokens)::bigint,
			sum(usage_events.output_tokens)::bigint,
			sum(usage_events.reasoning_output_tokens)::bigint,
			sum(usage_events.total_tokens)::bigint,
			sum(usage_events.cost_usd)
		FROM usage_events
		JOIN sessions ON sessions.id = usage_events.session_id
		WHERE usage_events.occurred_at IS NOT NULL
		  AND usage_events.occurred_at::date = ANY($1::date[])
		GROUP BY bucket_date, bucket_month, sessions.device_id, sessions.repository_id, sessions.project_id, usage_events.model
		ON CONFLICT (bucket_date, device_id, repository_id, project_id, model)
		DO UPDATE SET
			input_tokens = EXCLUDED.input_tokens,
			cached_input_tokens = EXCLUDED.cached_input_tokens,
			cache_write_input_tokens = EXCLUDED.cache_write_input_tokens,
			output_tokens = EXCLUDED.output_tokens,
			reasoning_output_tokens = EXCLUDED.reasoning_output_tokens,
			total_tokens = EXCLUDED.total_tokens,
			cost_usd = EXCLUDED.cost_usd,
			updated_at = now()
	`, dateValues)
	return err
}

func RebuildSessionRollupsForDates(ctx context.Context, tx pgx.Tx, dates []time.Time) error {
	dates = uniqueDates(dates)
	if len(dates) == 0 {
		return nil
	}
	dateValues := dateStrings(dates)
	if _, err := tx.Exec(ctx, `
		DELETE FROM session_rollups
		WHERE bucket_date = ANY($1::date[])
	`, dateValues); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO session_rollups (
			bucket_date, bucket_month, device_id, repository_id, project_id, session_count,
			input_tokens, cached_input_tokens, cache_write_input_tokens, output_tokens, reasoning_output_tokens,
			total_tokens, cost_usd, patch_added_lines
		)
		SELECT
			COALESCE(session_summaries.updated_at, session_summaries.started_at, sessions.updated_at, sessions.started_at, sessions.ingested_at)::date AS bucket_date,
			date_trunc('month', COALESCE(session_summaries.updated_at, session_summaries.started_at, sessions.updated_at, sessions.started_at, sessions.ingested_at))::date AS bucket_month,
			sessions.device_id,
			sessions.repository_id,
			sessions.project_id,
			count(sessions.id)::bigint,
			COALESCE(sum(session_summaries.input_tokens), 0)::bigint,
			COALESCE(sum(session_summaries.cached_input_tokens), 0)::bigint,
			COALESCE(sum(session_summaries.cache_write_input_tokens), 0)::bigint,
			COALESCE(sum(session_summaries.output_tokens), 0)::bigint,
			COALESCE(sum(session_summaries.reasoning_output_tokens), 0)::bigint,
			COALESCE(sum(session_summaries.total_tokens), 0)::bigint,
			COALESCE(sum(session_summaries.cost_usd), 0),
			COALESCE(sum(session_summaries.patch_added_lines), 0)::bigint
		FROM sessions
		LEFT JOIN session_summaries ON session_summaries.session_id = sessions.id
		WHERE COALESCE(session_summaries.updated_at, session_summaries.started_at, sessions.updated_at, sessions.started_at, sessions.ingested_at)::date = ANY($1::date[])
		GROUP BY bucket_date, bucket_month, sessions.device_id, sessions.repository_id, sessions.project_id
		ON CONFLICT (bucket_date, device_id, repository_id, project_id)
		DO UPDATE SET
			session_count = EXCLUDED.session_count,
			input_tokens = EXCLUDED.input_tokens,
			cached_input_tokens = EXCLUDED.cached_input_tokens,
			cache_write_input_tokens = EXCLUDED.cache_write_input_tokens,
			output_tokens = EXCLUDED.output_tokens,
			reasoning_output_tokens = EXCLUDED.reasoning_output_tokens,
			total_tokens = EXCLUDED.total_tokens,
			cost_usd = EXCLUDED.cost_usd,
			patch_added_lines = EXCLUDED.patch_added_lines,
			updated_at = now()
	`, dateValues)
	return err
}

func uniqueDates(dates []time.Time) []time.Time {
	seen := make(map[string]struct{}, len(dates))
	out := make([]time.Time, 0, len(dates))
	for _, date := range dates {
		day := date.UTC().Format(time.DateOnly)
		if _, ok := seen[day]; ok {
			continue
		}
		seen[day] = struct{}{}
		out = append(out, time.Date(date.UTC().Year(), date.UTC().Month(), date.UTC().Day(), 0, 0, 0, 0, time.UTC))
	}
	return out
}

func dateStrings(dates []time.Time) []string {
	out := make([]string, 0, len(dates))
	for _, date := range dates {
		out = append(out, date.UTC().Format(time.DateOnly))
	}
	return out
}

func firstNonEmpty(values ...string) string {
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

func titleWord(value string) string {
	if value == "" {
		return value
	}
	return strings.ToUpper(value[:1]) + value[1:]
}

func deriveSessionTitle(firstUser string, cwd string) string {
	firstUser = strings.TrimSpace(firstUser)
	if firstUser != "" {
		return truncate(oneLine(firstUser), 96)
	}
	if tail := filepath.Base(strings.TrimRight(cwd, string(filepath.Separator))); tail != "." && tail != "" {
		return tail
	}
	return "Untitled session"
}

func deriveSessionSummary(firstUser string, lastUser string, lastAssistant string) string {
	switch {
	case strings.TrimSpace(firstUser) != "" && strings.TrimSpace(lastAssistant) != "":
		return truncate(oneLine(firstUser)+" → "+oneLine(lastAssistant), 220)
	case strings.TrimSpace(lastUser) != "":
		return truncate(oneLine(lastUser), 220)
	case strings.TrimSpace(lastAssistant) != "":
		return truncate(oneLine(lastAssistant), 220)
	default:
		return ""
	}
}

func deriveUserIntent(firstUser string, lastUser string) string {
	intent := strings.TrimSpace(firstUser)
	if intent == "" {
		intent = strings.TrimSpace(lastUser)
	}
	return truncate(oneLine(intent), 180)
}

func deriveDisplaySubtitle(cwd string, branch string, model string) string {
	var parts []string
	if cwd != "" {
		parts = append(parts, cwd)
	}
	if branch != "" {
		parts = append(parts, branch)
	}
	if model != "" {
		parts = append(parts, model)
	}
	return truncate(strings.Join(parts, " · "), 220)
}

func detectDominantLanguage(value string) string {
	var hangul, latin int
	for _, r := range value {
		switch {
		case r >= 0xAC00 && r <= 0xD7A3:
			hangul++
		case (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z'):
			latin++
		}
	}
	if hangul > latin/3 && hangul > 0 {
		return "ko"
	}
	if latin > 0 {
		return "en"
	}
	return ""
}

func oneLine(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func userFacingText(value string) string {
	value = strings.TrimSpace(value)
	for _, marker := range []string{"</environment_context>", "</INSTRUCTIONS>", "</user_instructions>", "</goal_context>"} {
		if idx := strings.LastIndex(value, marker); idx >= 0 {
			value = strings.TrimSpace(value[idx+len(marker):])
		}
	}
	contextPrefixes := []string{
		"# AGENTS.md instructions",
		"# Review Guidelines",
		"# 사용 언어",
		"## Code review guidelines",
		"<INSTRUCTIONS>",
		"<environment_context>",
		"<goal_context>",
		"<user_instructions>",
		"<turn_aborted>",
	}
	for _, prefix := range contextPrefixes {
		if strings.HasPrefix(value, prefix) {
			return ""
		}
	}
	if strings.HasPrefix(value, "/goal ") {
		return ""
	}
	return value
}

func joinText(values ...string) string {
	var parts []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			parts = append(parts, value)
		}
	}
	return strings.Join(parts, "\n\n")
}

func uniqueStrings(values ...string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func sanitizeDBText(value string) string {
	value = strings.ToValidUTF8(value, "")
	return strings.ReplaceAll(value, "\x00", "")
}

func nullableTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableInt(value int) any {
	if value == 0 {
		return nil
	}
	return value
}
