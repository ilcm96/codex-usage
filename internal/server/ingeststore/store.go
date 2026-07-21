package ingeststore

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ilcm96/codex-usage/internal/core/pricing"
	"github.com/ilcm96/codex-usage/internal/server/projections"
	"github.com/ilcm96/codex-usage/internal/server/sessionparse"
)

type Store struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) Store {
	return Store{db: db}
}

type Metadata struct {
	Device struct {
		Name     string `json:"name"`
		Hostname string `json:"hostname"`
		Platform string `json:"platform"`
	} `json:"device"`
	Session struct {
		ID           string `json:"id"`
		Path         string `json:"path"`
		RawSHA256    string `json:"raw_sha256"`
		RawSizeBytes int64  `json:"raw_size_bytes"`
		RawField     string `json:"raw_field"`
	} `json:"session"`
	Workspace struct {
		CWD           string `json:"cwd"`
		GitRoot       string `json:"git_root"`
		RelativePath  string `json:"relative_path"`
		RepositoryURL string `json:"repository_url"`
		Branch        string `json:"branch"`
		CommitHash    string `json:"commit_hash"`
	} `json:"workspace"`
}

type Result struct {
	SessionID     string
	Status        string
	RawSHA256     string
	RawPath       string
	RawSizeBytes  int64
	Error         string
	AffectedDates []time.Time
}

type RawInput struct {
	Metadata     Metadata
	RawPath      string
	RawSizeBytes int64
}

type sessionInsertInput struct {
	Parsed       sessionparse.Session
	Meta         Metadata
	DeviceID     string
	RepositoryID string
	ProjectID    string
	StartedAt    *time.Time
	UpdatedAt    *time.Time
	RawPath      string
	RawSize      int64
}

func (s Store) StoreRaw(ctx context.Context, meta Metadata, rawPath string, rawSize int64) (Result, error) {
	results, err := s.StoreRawBatch(ctx, []RawInput{{
		Metadata:     meta,
		RawPath:      rawPath,
		RawSizeBytes: rawSize,
	}})
	if err != nil {
		return Result{}, err
	}
	if len(results) == 0 {
		return Result{}, fmt.Errorf("missing ingest result")
	}
	return results[0], nil
}

func (s Store) StoreRawBatch(ctx context.Context, inputs []RawInput) ([]Result, error) {
	if len(inputs) == 0 {
		return nil, nil
	}
	pr, err := pricing.LoadEmbeddedPricing()
	if err != nil {
		return nil, err
	}

	rawSHAs := make([]string, 0, len(inputs))
	for _, input := range inputs {
		rawSHAs = append(rawSHAs, input.Metadata.Session.RawSHA256)
	}
	existing, err := s.existingSessionsByRawSHA(ctx, rawSHAs)
	if err != nil {
		return nil, err
	}

	type parsedInput struct {
		Input     RawInput
		Parsed    sessionparse.Session
		StartedAt *time.Time
		UpdatedAt *time.Time
	}

	results := make([]Result, len(inputs))
	parsedInputs := make([]parsedInput, 0, len(inputs))
	resultBySHA := make(map[string]int, len(inputs))
	for i, input := range inputs {
		meta := input.Metadata
		resultBySHA[meta.Session.RawSHA256] = i
		results[i] = Result{
			SessionID:    meta.Session.ID,
			Status:       "pending",
			RawSHA256:    meta.Session.RawSHA256,
			RawPath:      input.RawPath,
			RawSizeBytes: input.RawSizeBytes,
		}
		if existingID := existing[meta.Session.RawSHA256]; existingID != "" {
			results[i].SessionID = existingID
			results[i].Status = "skipped"
			continue
		}

		parsed, err := sessionparse.ParseRaw(input.RawPath, sessionparse.FallbackMeta{
			SessionID:     meta.Session.ID,
			CWD:           meta.Workspace.CWD,
			RepositoryURL: meta.Workspace.RepositoryURL,
			Branch:        meta.Workspace.Branch,
			CommitHash:    meta.Workspace.CommitHash,
		}, pr)
		if err != nil {
			results[i].Status = "failed"
			results[i].Error = err.Error()
			continue
		}

		startedAt := firstTime(parsed.Meta.StartedAt, parsed.StartedAt)
		updatedAt := firstTime(parsed.UpdatedAt, startedAt)
		parsedInputs = append(parsedInputs, parsedInput{
			Input:     input,
			Parsed:    parsed,
			StartedAt: startedAt,
			UpdatedAt: updatedAt,
		})
	}

	if len(parsedInputs) == 0 {
		return results, nil
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	additiveRollupSessionIDs := make([]string, 0, len(parsedInputs))
	rebuildRollupSessionIDs := make([]string, 0)
	rebuildRollupDates := make([]time.Time, 0)
	rebuildSessionRollupDates := make([]time.Time, 0)
	for _, item := range parsedInputs {
		meta := item.Input.Metadata
		parsed := item.Parsed
		resultIndex := resultBySHA[meta.Session.RawSHA256]

		deviceID, err := upsertDevice(ctx, tx, meta)
		if err != nil {
			return nil, err
		}
		repositoryID, err := upsertRepository(ctx, tx, parsed.Meta.RepositoryURL)
		if err != nil {
			return nil, err
		}
		projectID, err := upsertProject(ctx, tx, repositoryID, meta, parsed.Meta)
		if err != nil {
			return nil, err
		}

		existingDates, err := projections.UsageRollupDatesForSessions(ctx, tx, []string{parsed.Meta.ID})
		if err != nil {
			return nil, err
		}
		existingSessionDates, err := projections.SessionRollupDatesForSessions(ctx, tx, []string{parsed.Meta.ID})
		if err != nil {
			return nil, err
		}
		if _, err := tx.Exec(ctx, `DELETE FROM sessions WHERE id = $1`, parsed.Meta.ID); err != nil {
			return nil, err
		}

		insertedSessionID, inserted, err := insertSession(ctx, tx, sessionInsertInput{
			Parsed:       parsed,
			Meta:         meta,
			DeviceID:     deviceID,
			RepositoryID: repositoryID,
			ProjectID:    projectID,
			StartedAt:    item.StartedAt,
			UpdatedAt:    item.UpdatedAt,
			RawPath:      item.Input.RawPath,
			RawSize:      item.Input.RawSizeBytes,
		})
		if err != nil {
			return nil, err
		}
		results[resultIndex].SessionID = insertedSessionID
		if !inserted {
			results[resultIndex].Status = "skipped"
			continue
		}

		if _, err := tx.Exec(ctx, `
			INSERT INTO archive_files (
				session_id, device_id, raw_file_path, raw_sha256,
				raw_size_bytes, archived_at, verified_at, status
			)
			VALUES ($1, $2, $3, $4, $5, now(), now(), 'verified')
		`, parsed.Meta.ID, deviceID, item.Input.RawPath, meta.Session.RawSHA256, item.Input.RawSizeBytes); err != nil {
			return nil, err
		}

		if err := insertParsedEvents(ctx, tx, parsed); err != nil {
			return nil, err
		}
		if err := projections.RefreshSessionWithoutUsageRollups(ctx, tx, parsed, item.StartedAt, item.UpdatedAt); err != nil {
			return nil, err
		}

		results[resultIndex].Status = "ingested"
		if len(existingDates) > 0 {
			rebuildRollupDates = append(rebuildRollupDates, existingDates...)
			rebuildSessionRollupDates = append(rebuildSessionRollupDates, existingSessionDates...)
			rebuildRollupSessionIDs = append(rebuildRollupSessionIDs, parsed.Meta.ID)
		} else {
			additiveRollupSessionIDs = append(additiveRollupSessionIDs, parsed.Meta.ID)
		}
	}

	if err := projections.AddUsageRollupsForSessions(ctx, tx, additiveRollupSessionIDs); err != nil {
		return nil, err
	}
	if err := projections.AddSessionRollupsForSessions(ctx, tx, additiveRollupSessionIDs); err != nil {
		return nil, err
	}
	newRebuildDates, err := projections.UsageRollupDatesForSessions(ctx, tx, rebuildRollupSessionIDs)
	if err != nil {
		return nil, err
	}
	rebuildRollupDates = append(rebuildRollupDates, newRebuildDates...)
	if err := projections.RebuildUsageRollupsForDates(ctx, tx, rebuildRollupDates); err != nil {
		return nil, err
	}
	newSessionRebuildDates, err := projections.SessionRollupDatesForSessions(ctx, tx, rebuildRollupSessionIDs)
	if err != nil {
		return nil, err
	}
	rebuildSessionRollupDates = append(rebuildSessionRollupDates, newSessionRebuildDates...)
	if err := projections.RebuildSessionRollupsForDates(ctx, tx, rebuildSessionRollupDates); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return results, nil
}

func (s Store) existingSessionsByRawSHA(ctx context.Context, rawSHAs []string) (map[string]string, error) {
	out := make(map[string]string, len(rawSHAs))
	rows, err := s.db.Query(ctx, `SELECT raw_sha256, id FROM sessions WHERE raw_sha256 = ANY($1)`, rawSHAs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var rawSHA string
		var id string
		if err := rows.Scan(&rawSHA, &id); err != nil {
			return nil, err
		}
		out[rawSHA] = id
	}
	return out, rows.Err()
}

func insertSession(ctx context.Context, tx pgx.Tx, input sessionInsertInput) (string, bool, error) {
	var insertedID string
	err := tx.QueryRow(ctx, `
		INSERT INTO sessions (
			id, device_id, repository_id, project_id, started_at, updated_at, cwd,
			originator, source, cli_version, model_provider, branch, commit_hash,
			raw_sha256, raw_size_bytes, raw_file_path
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		ON CONFLICT (raw_sha256) DO NOTHING
		RETURNING id
	`, input.Parsed.Meta.ID, input.DeviceID, nullableString(input.RepositoryID), nullableString(input.ProjectID), nullableTime(input.StartedAt), nullableTime(input.UpdatedAt),
		input.Parsed.Meta.CWD, input.Parsed.Meta.Originator, input.Parsed.Meta.Source, input.Parsed.Meta.CLIVersion, input.Parsed.Meta.ModelProvider,
		input.Parsed.Meta.Branch, input.Parsed.Meta.CommitHash, input.Meta.Session.RawSHA256, input.RawSize, input.RawPath).Scan(&insertedID)
	if err == nil {
		return insertedID, true, nil
	}
	if err != pgx.ErrNoRows {
		return "", false, err
	}

	err = tx.QueryRow(ctx, `SELECT id FROM sessions WHERE raw_sha256 = $1`, input.Meta.Session.RawSHA256).Scan(&insertedID)
	if err != nil {
		return "", false, err
	}
	return insertedID, false, nil
}

func upsertDevice(ctx context.Context, tx pgx.Tx, meta Metadata) (string, error) {
	name := firstNonEmpty(meta.Device.Name, meta.Device.Hostname, "unknown-device")
	hostname := firstNonEmpty(meta.Device.Hostname, meta.Device.Name, "unknown-device")
	platform := meta.Device.Platform
	var id string
	err := tx.QueryRow(ctx, `
		INSERT INTO devices (name, hostname, platform, last_seen_at)
		VALUES ($1, $2, $3, now())
		ON CONFLICT (hostname)
		DO UPDATE SET name = EXCLUDED.name, platform = EXCLUDED.platform, last_seen_at = now()
		RETURNING id::text
	`, name, hostname, platform).Scan(&id)
	return id, err
}

func upsertRepository(ctx context.Context, tx pgx.Tx, rawURL string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return "", nil
	}
	canonicalURL := canonicalRepositoryURL(rawURL)
	host, owner, name := splitRepositoryURL(canonicalURL)
	var id string
	err := tx.QueryRow(ctx, `
		INSERT INTO repositories (repository_url, repository_host, repository_owner, repository_name)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (repository_url)
		DO UPDATE SET repository_host = EXCLUDED.repository_host, repository_owner = EXCLUDED.repository_owner, repository_name = EXCLUDED.repository_name
		RETURNING id::text
	`, canonicalURL, host, owner, name).Scan(&id)
	if err != nil {
		return "", err
	}
	for _, alias := range uniqueStrings(canonicalURL, rawURL) {
		if _, err := tx.Exec(ctx, `
			INSERT INTO repository_aliases (repository_id, alias_url, source)
			VALUES ($1, $2, 'ingest')
			ON CONFLICT (alias_url)
			DO UPDATE SET repository_id = EXCLUDED.repository_id
		`, id, alias); err != nil {
			return "", err
		}
	}
	return id, nil
}

func upsertProject(ctx context.Context, tx pgx.Tx, repositoryID string, meta Metadata, parsedMeta sessionparse.SessionMeta) (string, error) {
	cwd := firstNonEmpty(parsedMeta.CWD, meta.Workspace.CWD)
	if cwd == "" {
		return "", nil
	}
	gitRoot := meta.Workspace.GitRoot
	relativePath := meta.Workspace.RelativePath
	if relativePath == "" && gitRoot != "" {
		if rel, err := filepath.Rel(gitRoot, cwd); err == nil && rel != "." {
			relativePath = rel
		}
	}
	displayName := firstNonEmpty(relativePath, filepath.Base(cwd), cwd)

	var id string
	if repositoryID == "" {
		err := tx.QueryRow(ctx, `
			SELECT id::text FROM projects WHERE repository_id IS NULL AND cwd = $1 LIMIT 1
		`, cwd).Scan(&id)
		if err == nil {
			return id, nil
		}
		if err != pgx.ErrNoRows {
			return "", err
		}
		err = tx.QueryRow(ctx, `
			INSERT INTO projects (repository_id, cwd, git_root, relative_path, display_name)
			VALUES (NULL, $1, $2, $3, $4)
			RETURNING id::text
		`, cwd, gitRoot, relativePath, displayName).Scan(&id)
		return id, err
	}

	err := tx.QueryRow(ctx, `
		INSERT INTO projects (repository_id, cwd, git_root, relative_path, display_name)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (repository_id, cwd)
		DO UPDATE SET git_root = EXCLUDED.git_root, relative_path = EXCLUDED.relative_path, display_name = EXCLUDED.display_name
		RETURNING id::text
	`, repositoryID, cwd, gitRoot, relativePath, displayName).Scan(&id)
	return id, err
}

func insertParsedEvents(ctx context.Context, tx pgx.Tx, parsed sessionparse.Session) error {
	if len(parsed.Events) > 0 {
		rows := make([][]any, 0, len(parsed.Events))
		for _, ev := range parsed.Events {
			rows = append(rows, []any{
				parsed.Meta.ID,
				ev.Seq,
				ev.Hash,
				nullableTime(ev.OccurredAt),
				ev.EventType,
				ev.PayloadType,
				sanitizeDBText(ev.Role),
				sanitizeDBText(ev.ToolName),
				sanitizeDBText(ev.CallID),
				sanitizeDBText(ev.ContentText),
				sanitizeJSONB(ev.PayloadJSON),
			})
		}
		if _, err := tx.CopyFrom(ctx, pgx.Identifier{"session_events"}, []string{
			"session_id", "seq", "event_hash", "occurred_at", "event_type", "payload_type",
			"role", "tool_name", "call_id", "content_text", "payload_jsonb",
		}, pgx.CopyFromRows(rows)); err != nil {
			return fmt.Errorf("insert session events: %w", err)
		}
	}

	if len(parsed.Messages) > 0 {
		rows := make([][]any, 0, len(parsed.Messages))
		for _, msg := range parsed.Messages {
			rows = append(rows, []any{
				parsed.Meta.ID,
				msg.Seq,
				nullableTime(msg.OccurredAt),
				sanitizeDBText(msg.Role),
				sanitizeDBText(msg.ContentText),
				sanitizeJSONB(msg.ContentJSON),
			})
		}
		if _, err := tx.CopyFrom(ctx, pgx.Identifier{"messages"}, []string{
			"session_id", "seq", "occurred_at", "role", "content_text", "content_jsonb",
		}, pgx.CopyFromRows(rows)); err != nil {
			return fmt.Errorf("insert messages: %w", err)
		}
	}

	if len(parsed.Tools) > 0 {
		rows := make([][]any, 0, len(parsed.Tools))
		for _, tool := range parsed.Tools {
			rows = append(rows, []any{
				parsed.Meta.ID,
				tool.Seq,
				nullableTime(tool.OccurredAt),
				sanitizeDBText(tool.Kind),
				sanitizeDBText(tool.ToolName),
				sanitizeDBText(tool.CallID),
				sanitizeDBText(tool.Status),
				sanitizeDBText(tool.InputText),
				sanitizeDBText(tool.OutputText),
				sanitizeJSONB(tool.PayloadJSON),
			})
		}
		if _, err := tx.CopyFrom(ctx, pgx.Identifier{"tool_events"}, []string{
			"session_id", "seq", "occurred_at", "kind", "tool_name", "call_id", "status",
			"input_text", "output_text", "payload_jsonb",
		}, pgx.CopyFromRows(rows)); err != nil {
			return fmt.Errorf("insert tool events: %w", err)
		}
	}

	if len(parsed.Usages) > 0 {
		rows := make([][]any, 0, len(parsed.Usages))
		for _, usage := range parsed.Usages {
			rows = append(rows, []any{
				parsed.Meta.ID,
				usage.Seq,
				nullableTime(usage.OccurredAt),
				usage.Model,
				usage.InputTokens,
				usage.CachedInputTokens,
				usage.CacheWriteInputTokens,
				usage.OutputTokens,
				usage.ReasoningOutputTokens,
				usage.TotalTokens,
				usage.CostUSD,
			})
		}
		if _, err := tx.CopyFrom(ctx, pgx.Identifier{"usage_events"}, []string{
			"session_id", "seq", "occurred_at", "model", "input_tokens", "cached_input_tokens", "cache_write_input_tokens",
			"output_tokens", "reasoning_output_tokens", "total_tokens", "cost_usd",
		}, pgx.CopyFromRows(rows)); err != nil {
			return fmt.Errorf("insert usage events: %w", err)
		}
	}
	return nil
}

func splitRepositoryURL(raw string) (host string, owner string, name string) {
	raw = strings.TrimSuffix(strings.TrimSpace(raw), ".git")
	if strings.Contains(raw, "://") {
		if u, err := url.Parse(raw); err == nil {
			host = u.Host
			parts := strings.Split(strings.Trim(u.Path, "/"), "/")
			if len(parts) >= 2 {
				owner = parts[len(parts)-2]
				name = parts[len(parts)-1]
			}
			return host, owner, name
		}
	}
	if strings.Contains(raw, ":") {
		left, right, _ := strings.Cut(raw, ":")
		host = strings.TrimPrefix(left, "git@")
		parts := strings.Split(strings.Trim(right, "/"), "/")
		if len(parts) >= 2 {
			owner = parts[len(parts)-2]
			name = parts[len(parts)-1]
		}
		return host, owner, name
	}
	name = filepath.Base(raw)
	return "", "", name
}

func canonicalRepositoryURL(raw string) string {
	raw = strings.TrimSpace(strings.TrimSuffix(raw, ".git"))
	if raw == "" {
		return ""
	}
	if strings.Contains(raw, "://") {
		if u, err := url.Parse(raw); err == nil {
			u.Scheme = strings.ToLower(u.Scheme)
			u.Host = strings.ToLower(u.Host)
			u.Path = strings.TrimSuffix(strings.TrimRight(u.Path, "/"), ".git")
			u.Path = canonicalRepositoryPath(u.Host, u.Path)
			u.RawQuery = ""
			u.Fragment = ""
			return u.String()
		}
	}
	if strings.HasPrefix(raw, "git@") && strings.Contains(raw, ":") {
		left, right, _ := strings.Cut(raw, ":")
		host := strings.TrimPrefix(strings.ToLower(left), "git@")
		path := strings.TrimSuffix(strings.Trim(right, "/"), ".git")
		if path != "" {
			path = strings.TrimPrefix(canonicalRepositoryPath(host, "/"+path), "/")
			return "https://" + host + "/" + path
		}
	}
	return raw
}

func canonicalRepositoryPath(host string, path string) string {
	if strings.EqualFold(host, "github.com") {
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if len(parts) >= 2 && strings.EqualFold(parts[0], "dk-aegis") {
			parts[0] = "dkuaegis"
			return "/" + strings.Join(parts, "/")
		}
	}
	return path
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
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

func sanitizeJSONB(value []byte) []byte {
	value = bytes.ReplaceAll(value, []byte{0}, nil)
	var decoded any
	if err := json.Unmarshal(value, &decoded); err != nil {
		return []byte("null")
	}
	encoded, err := json.Marshal(sanitizeJSONValue(decoded))
	if err != nil {
		return []byte("null")
	}
	return encoded
}

func sanitizeJSONValue(value any) any {
	switch v := value.(type) {
	case string:
		return sanitizeDBText(v)
	case []any:
		for i, item := range v {
			v[i] = sanitizeJSONValue(item)
		}
		return v
	case map[string]any:
		out := make(map[string]any, len(v))
		for key, item := range v {
			out[sanitizeDBText(key)] = sanitizeJSONValue(item)
		}
		return out
	default:
		return value
	}
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func nullableTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return *value
}

func firstTime(values ...*time.Time) *time.Time {
	for _, v := range values {
		if v != nil {
			return v
		}
	}
	return nil
}
