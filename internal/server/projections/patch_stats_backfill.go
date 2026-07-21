package projections

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ilcm96/codex-usage/internal/server/sessionparse"
)

type PatchStatsBackfillResult struct {
	DryRun             bool     `json:"dry_run"`
	Since              string   `json:"since"`
	Sessions           int      `json:"sessions"`
	PreviousAddedLines int64    `json:"previous_added_lines"`
	UpdatedAddedLines  int64    `json:"updated_added_lines"`
	RollupDates        []string `json:"rollup_dates"`
}

func BackfillPatchStats(ctx context.Context, db *pgxpool.Pool, since time.Time, dryRun bool) (PatchStatsBackfillResult, error) {
	result := PatchStatsBackfillResult{
		DryRun: dryRun,
		Since:  since.Format(time.RFC3339),
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		return result, fmt.Errorf("begin patch stats backfill: %w", err)
	}
	defer tx.Rollback(ctx)

	sessionIDs, err := patchStatsBackfillSessionIDs(ctx, tx, since)
	if err != nil {
		return result, err
	}
	result.Sessions = len(sessionIDs)
	if len(sessionIDs) == 0 {
		if !dryRun {
			if err := tx.Commit(ctx); err != nil {
				return result, fmt.Errorf("commit empty patch stats backfill: %w", err)
			}
		}
		return result, nil
	}

	toolsBySession, err := patchStatsBackfillTools(ctx, tx, sessionIDs)
	if err != nil {
		return result, err
	}
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(sum(patch_added_lines), 0)::bigint
		FROM session_summaries
		WHERE session_id = ANY($1)
	`, sessionIDs).Scan(&result.PreviousAddedLines); err != nil {
		return result, fmt.Errorf("load existing patch stats: %w", err)
	}

	dates, err := SessionRollupDatesForSessions(ctx, tx, sessionIDs)
	if err != nil {
		return result, fmt.Errorf("load patch stats rollup dates: %w", err)
	}
	for _, date := range uniqueDates(dates) {
		result.RollupDates = append(result.RollupDates, date.Format(time.DateOnly))
	}

	batch := &pgx.Batch{}
	for _, sessionID := range sessionIDs {
		stats := collectPatchStats(toolsBySession[sessionID])
		result.UpdatedAddedLines += stats.AddedLines
		languageStats, err := json.Marshal(stats.LanguageLines)
		if err != nil {
			return result, fmt.Errorf("encode patch language stats for session %s: %w", sessionID, err)
		}
		batch.Queue(`
			UPDATE session_summaries
			SET patch_added_lines = $2,
				patch_language_stats = $3::jsonb,
				dominant_language = CASE WHEN $4 <> '' THEN $4 ELSE dominant_language END,
				generated_at = now()
			WHERE session_id = $1
		`, sessionID, stats.AddedLines, string(languageStats), stats.DominantLanguage)
	}

	if dryRun {
		return result, nil
	}

	batchResults := tx.SendBatch(ctx, batch)
	for _, sessionID := range sessionIDs {
		commandTag, err := batchResults.Exec()
		if err != nil {
			_ = batchResults.Close()
			return result, fmt.Errorf("update patch stats for session %s: %w", sessionID, err)
		}
		if commandTag.RowsAffected() != 1 {
			_ = batchResults.Close()
			return result, fmt.Errorf("update patch stats for session %s: updated %d summaries", sessionID, commandTag.RowsAffected())
		}
	}
	if err := batchResults.Close(); err != nil {
		return result, fmt.Errorf("close patch stats update batch: %w", err)
	}
	if err := RebuildSessionRollupsForDates(ctx, tx, dates); err != nil {
		return result, fmt.Errorf("rebuild patch stats session rollups: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return result, fmt.Errorf("commit patch stats backfill: %w", err)
	}
	return result, nil
}

func patchStatsBackfillSessionIDs(ctx context.Context, tx pgx.Tx, since time.Time) ([]string, error) {
	rows, err := tx.Query(ctx, `
		SELECT DISTINCT session_id
		FROM tool_events
		WHERE occurred_at >= $1
		  AND kind LIKE '%patch_apply%'
		  AND payload_jsonb ? 'changes'
		ORDER BY session_id
	`, since)
	if err != nil {
		return nil, fmt.Errorf("load patch stats backfill sessions: %w", err)
	}
	defer rows.Close()

	var sessionIDs []string
	for rows.Next() {
		var sessionID string
		if err := rows.Scan(&sessionID); err != nil {
			return nil, fmt.Errorf("scan patch stats backfill session: %w", err)
		}
		sessionIDs = append(sessionIDs, sessionID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate patch stats backfill sessions: %w", err)
	}
	return sessionIDs, nil
}

func patchStatsBackfillTools(ctx context.Context, tx pgx.Tx, sessionIDs []string) (map[string][]sessionparse.ToolEvent, error) {
	rows, err := tx.Query(ctx, `
		SELECT session_id, seq, occurred_at, kind, tool_name, call_id, status, input_text, output_text, payload_jsonb
		FROM tool_events
		WHERE session_id = ANY($1)
		ORDER BY session_id, seq
	`, sessionIDs)
	if err != nil {
		return nil, fmt.Errorf("load patch stats backfill tools: %w", err)
	}
	defer rows.Close()

	toolsBySession := make(map[string][]sessionparse.ToolEvent, len(sessionIDs))
	for rows.Next() {
		var sessionID string
		var tool sessionparse.ToolEvent
		if err := rows.Scan(
			&sessionID,
			&tool.Seq,
			&tool.OccurredAt,
			&tool.Kind,
			&tool.ToolName,
			&tool.CallID,
			&tool.Status,
			&tool.InputText,
			&tool.OutputText,
			&tool.PayloadJSON,
		); err != nil {
			return nil, fmt.Errorf("scan patch stats backfill tool: %w", err)
		}
		toolsBySession[sessionID] = append(toolsBySession[sessionID], tool)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate patch stats backfill tools: %w", err)
	}
	return toolsBySession, nil
}
