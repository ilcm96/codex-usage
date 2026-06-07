package sessions

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ilcm96/codex-usage/internal/server/httpapi"
)

// handleExportSessions exports multiple sessions.
// @Summary Export sessions
// @Tags Exports
// @Produce json
// @Produce text/csv
// @Produce text/markdown
// @Security CookieAuth
// @Param format query string false "Export format: csv, json, or markdown"
// @Param limit query int false "Maximum number of rows"
// @Param offset query int false "Pagination offset"
// @Param sort query string false "Sort key"
// @Param from query string false "Start date in YYYY-MM-DD format"
// @Param to query string false "End date in YYYY-MM-DD format"
// @Param deviceId query string false "Device ID filter"
// @Param repositoryId query string false "Repository ID filter"
// @Param projectId query string false "Project ID filter"
// @Param branch query string false "Branch filter"
// @Param model query string false "Model filter"
// @Param q query string false "Text search query"
// @Success 200 {array} ListItem
// @Failure 400 {object} httpapi.ErrorResponse
// @Failure 401 {object} httpapi.ErrorResponse
// @Failure 500 {object} httpapi.ErrorResponse
// @Router /api/exports/sessions [get]
func (c Controller) handleExportSessions(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "csv"
	}
	limit := httpapi.QueryInt(r, "limit", 1000)
	if limit > 5000 {
		limit = 5000
	}

	params := listParamsFromRequest(r)
	params.Limit = limit
	if _, ok := sessionOrderBy(params.Sort); !ok {
		httpapi.WriteError(w, http.StatusBadRequest, "unsupported sort")
		return
	}
	result, err := c.service.repository.List(r.Context(), params)
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "failed to load export sessions")
		return
	}
	items := result.Items

	switch format {
	case "csv":
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="codex-sessions.csv"`)
		writer := csv.NewWriter(w)
		_ = writer.Write([]string{
			"id",
			"repository",
			"repository_url",
			"project",
			"cwd",
			"branch",
			"device",
			"models",
			"started_at",
			"updated_at",
			"input_tokens",
			"cached_input_tokens",
			"output_tokens",
			"reasoning_output_tokens",
			"total_tokens",
			"cost_usd",
			"messages",
			"tool_calls",
		})
		for _, item := range items {
			_ = writer.Write([]string{
				item.ID,
				item.Repository,
				item.RepositoryURL,
				item.Project,
				item.CWD,
				item.Branch,
				item.Device,
				item.Models,
				formatTimeForExport(item.StartedAt),
				formatTimeForExport(item.UpdatedAt),
				strconv.FormatInt(item.InputTokens, 10),
				strconv.FormatInt(item.CachedTokens, 10),
				strconv.FormatInt(item.OutputTokens, 10),
				strconv.FormatInt(item.Reasoning, 10),
				strconv.FormatInt(item.TotalTokens, 10),
				strconv.FormatFloat(item.CostUSD, 'f', 8, 64),
				strconv.FormatInt(item.MessageCount, 10),
				strconv.FormatInt(item.ToolCallCount, 10),
			})
		}
		writer.Flush()
	case "json":
		w.Header().Set("Content-Disposition", `attachment; filename="codex-sessions.json"`)
		httpapi.WriteJSON(w, http.StatusOK, items)
	case "markdown":
		w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="codex-sessions.md"`)
		var b strings.Builder
		b.WriteString("# Codex sessions export\n\n")
		for _, item := range items {
			fmt.Fprintf(&b, "## %s\n\n", item.Project)
			fmt.Fprintf(&b, "- Session: `%s`\n", item.ID)
			fmt.Fprintf(&b, "- Repository: %s\n", firstNonEmptyString(item.Repository, "local"))
			fmt.Fprintf(&b, "- CWD: `%s`\n", item.CWD)
			fmt.Fprintf(&b, "- Models: %s\n", firstNonEmptyString(item.Models, "-"))
			fmt.Fprintf(&b, "- Tokens: %d\n", item.TotalTokens)
			fmt.Fprintf(&b, "- Cost: %.8f\n\n", item.CostUSD)
		}
		_, _ = io.WriteString(w, b.String())
	default:
		httpapi.WriteError(w, http.StatusBadRequest, "unsupported export format")
	}
}

// handleExportSession exports one session.
// @Summary Export session
// @Tags Exports
// @Produce json
// @Produce text/markdown
// @Produce application/octet-stream
// @Security CookieAuth
// @Param id path string true "Session ID"
// @Param format query string false "Export format: raw, json, or markdown"
// @Success 200 {object} object
// @Failure 400 {object} httpapi.ErrorResponse
// @Failure 401 {object} httpapi.ErrorResponse
// @Failure 404 {object} httpapi.ErrorResponse
// @Failure 500 {object} httpapi.ErrorResponse
// @Router /api/sessions/{id}/export [get]
func (c Controller) handleExportSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "markdown"
	}
	switch format {
	case "raw":
		c.exportRaw(w, r, id)
	case "json":
		c.exportJSON(w, r, id)
	case "markdown":
		c.exportMarkdown(w, r, id)
	default:
		httpapi.WriteError(w, http.StatusBadRequest, "unsupported export format")
	}
}

func (c Controller) exportRaw(w http.ResponseWriter, r *http.Request, id string) {
	path, err := c.service.repository.RawPath(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		httpapi.WriteError(w, http.StatusNotFound, "session not found")
		return
	}
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "failed to load raw session")
		return
	}
	w.Header().Set("Content-Type", "application/x-ndjson; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.jsonl"`, id))
	http.ServeFile(w, r, path)
}

func (c Controller) exportJSON(w http.ResponseWriter, r *http.Request, id string) {
	path, err := c.service.repository.RawPath(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		httpapi.WriteError(w, http.StatusNotFound, "session not found")
		return
	}
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "failed to load raw session")
		return
	}
	events, err := readRawEvents(path)
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "failed to read raw session")
		return
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.clean.json"`, id))
	httpapi.WriteJSON(w, http.StatusOK, events)
}

func (c Controller) exportMarkdown(w http.ResponseWriter, r *http.Request, id string) {
	timeline, err := c.service.repository.FullTimeline(r.Context(), id)
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "failed to load timeline")
		return
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# Codex session %s\n\n", id)
	for _, item := range timeline {
		if item.Text == "" {
			continue
		}
		fmt.Fprintf(&b, "## %s: %s\n\n%s\n\n", titleWord(item.Kind), item.Role, item.Text)
	}
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.md"`, id))
	_, _ = io.WriteString(w, b.String())
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func titleWord(value string) string {
	if value == "" {
		return value
	}
	return strings.ToUpper(value[:1]) + value[1:]
}

func readRawEvents(path string) ([]map[string]any, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var out []map[string]any
	dec := json.NewDecoder(f)
	for {
		var item map[string]any
		if err := dec.Decode(&item); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		out = append(out, item)
	}
	return out, nil
}

func formatTimeForExport(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.Format(time.RFC3339)
}
