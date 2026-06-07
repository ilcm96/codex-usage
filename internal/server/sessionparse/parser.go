package sessionparse

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/tidwall/gjson"

	"github.com/ilcm96/codex-usage/internal/core/codexlog"
	"github.com/ilcm96/codex-usage/internal/core/pricing"
)

type Session struct {
	Meta      SessionMeta
	Events    []Event
	Messages  []Message
	Tools     []ToolEvent
	Usages    []UsageEvent
	StartedAt *time.Time
	UpdatedAt *time.Time
}

type SessionMeta struct {
	ID            string
	StartedAt     *time.Time
	CWD           string
	Originator    string
	Source        string
	CLIVersion    string
	ModelProvider string
	RepositoryURL string
	Branch        string
	CommitHash    string
}

type Event struct {
	Seq         int
	Hash        string
	OccurredAt  *time.Time
	EventType   string
	PayloadType string
	Role        string
	ToolName    string
	CallID      string
	ContentText string
	PayloadJSON []byte
}

type Message struct {
	Seq         int
	OccurredAt  *time.Time
	Role        string
	ContentText string
	ContentJSON []byte
}

type ToolEvent struct {
	Seq         int
	OccurredAt  *time.Time
	Kind        string
	ToolName    string
	CallID      string
	Status      string
	InputText   string
	OutputText  string
	PayloadJSON []byte
}

type UsageEvent struct {
	Seq                   int
	OccurredAt            *time.Time
	Model                 string
	InputTokens           int64
	CachedInputTokens     int64
	OutputTokens          int64
	ReasoningOutputTokens int64
	TotalTokens           int64
	CostUSD               float64
}

type FallbackMeta struct {
	SessionID     string
	CWD           string
	RepositoryURL string
	Branch        string
	CommitHash    string
}

func ParseRaw(path string, fallbackMeta FallbackMeta, pr pricing.Pricing) (Session, error) {
	f, err := os.Open(path)
	if err != nil {
		return Session{}, err
	}
	defer f.Close()

	return parseJSONL(f, fallbackMeta, pr)
}

func parseJSONL(r io.Reader, fallbackMeta FallbackMeta, pr pricing.Pricing) (Session, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)

	out := Session{
		Meta: SessionMeta{
			ID:            fallbackMeta.SessionID,
			CWD:           fallbackMeta.CWD,
			RepositoryURL: fallbackMeta.RepositoryURL,
			Branch:        fallbackMeta.Branch,
			CommitHash:    fallbackMeta.CommitHash,
		},
	}

	normalizer := codexlog.NewUsageNormalizer(codexlog.DefaultFallbackModel)
	sessionMetaSeen := false

	seq := 0
	for scanner.Scan() {
		seq++
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		if !gjson.ValidBytes(line) {
			continue
		}

		ts := parseTimePtr(gjson.GetBytes(line, "timestamp").String())
		out.trackTime(ts)

		eventType := gjson.GetBytes(line, "type").String()
		payload := gjson.GetBytes(line, "payload")
		payloadType := payload.Get("type").String()
		if eventType == "session_meta" && !sessionMetaSeen {
			out.mergeSessionMeta(payload)
			sessionMetaSeen = true
		}

		event := Event{
			Seq:         seq,
			Hash:        hashEvent(seq, line),
			OccurredAt:  ts,
			EventType:   eventType,
			PayloadType: payloadType,
			Role:        firstJSON(line, "role", "payload.role"),
			ToolName:    firstJSON(line, "name", "payload.name"),
			CallID:      firstJSON(line, "call_id", "payload.call_id"),
			ContentText: eventContentText(line),
			PayloadJSON: cloneJSON(line),
		}
		out.Events = append(out.Events, event)

		if msg, ok := normalizeMessage(seq, ts, line); ok {
			out.Messages = append(out.Messages, msg)
		}
		if tool, ok := normalizeToolEvent(seq, ts, line); ok {
			out.Tools = append(out.Tools, tool)
		}

		if eventType == "turn_context" {
			normalizer.ObserveModel(line)
			continue
		}
		if eventType != "event_msg" || payloadType != "token_count" {
			continue
		}

		usageEvent, ok := normalizer.NormalizeUsageLine(line)
		if !ok {
			continue
		}
		usage := usageEvent.Usage

		out.Usages = append(out.Usages, UsageEvent{
			Seq:                   seq,
			OccurredAt:            ts,
			Model:                 usageEvent.Model,
			InputTokens:           usage.InputTokens,
			CachedInputTokens:     usage.CachedInputTokens,
			OutputTokens:          usage.OutputTokens,
			ReasoningOutputTokens: usage.ReasoningOutputTokens,
			TotalTokens:           usage.TotalTokens,
			CostUSD: pr.CostForModelUSD(usageEvent.Model, pricing.TokenUsage{
				InputTokens:     usage.InputTokens,
				CacheReadTokens: usage.CachedInputTokens,
				OutputTokens:    usage.OutputTokens,
			}, codexlog.DefaultFallbackModel),
		})
	}

	if err := scanner.Err(); err != nil {
		return Session{}, err
	}
	if out.Meta.ID == "" {
		return Session{}, fmt.Errorf("session id is missing")
	}
	if out.Meta.CWD == "" {
		out.Meta.CWD = fallbackMeta.CWD
	}
	if out.Meta.RepositoryURL == "" {
		out.Meta.RepositoryURL = fallbackMeta.RepositoryURL
	}
	if out.Meta.Branch == "" {
		out.Meta.Branch = fallbackMeta.Branch
	}
	if out.Meta.CommitHash == "" {
		out.Meta.CommitHash = fallbackMeta.CommitHash
	}
	return out, nil
}

func (s *Session) trackTime(t *time.Time) {
	if t == nil {
		return
	}
	if s.StartedAt == nil || t.Before(*s.StartedAt) {
		v := *t
		s.StartedAt = &v
	}
	if s.UpdatedAt == nil || t.After(*s.UpdatedAt) {
		v := *t
		s.UpdatedAt = &v
	}
}

func (s *Session) mergeSessionMeta(payload gjson.Result) {
	if id := payload.Get("id").String(); id != "" {
		s.Meta.ID = id
	}
	if t := parseTimePtr(payload.Get("timestamp").String()); t != nil {
		s.Meta.StartedAt = t
		s.trackTime(t)
	}
	if cwd := payload.Get("cwd").String(); cwd != "" {
		s.Meta.CWD = cwd
	}
	s.Meta.Originator = firstNonEmpty(s.Meta.Originator, payload.Get("originator").String())
	s.Meta.Source = firstNonEmpty(s.Meta.Source, payload.Get("source").String())
	s.Meta.CLIVersion = firstNonEmpty(s.Meta.CLIVersion, payload.Get("cli_version").String())
	s.Meta.ModelProvider = firstNonEmpty(s.Meta.ModelProvider, payload.Get("model_provider").String())
	s.Meta.RepositoryURL = firstNonEmpty(s.Meta.RepositoryURL, payload.Get("git.repository_url").String())
	s.Meta.Branch = firstNonEmpty(s.Meta.Branch, payload.Get("git.branch").String())
	s.Meta.CommitHash = firstNonEmpty(s.Meta.CommitHash, payload.Get("git.commit_hash").String())
}

func normalizeMessage(seq int, ts *time.Time, line []byte) (Message, bool) {
	eventType := gjson.GetBytes(line, "type").String()
	var obj gjson.Result
	switch eventType {
	case "message":
		obj = gjson.ParseBytes(line)
	case "response_item":
		payload := gjson.GetBytes(line, "payload")
		if payload.Get("type").String() != "message" {
			return Message{}, false
		}
		obj = payload
	case "event_msg":
		payload := gjson.GetBytes(line, "payload")
		pt := payload.Get("type").String()
		if pt != "user_message" && pt != "agent_message" {
			return Message{}, false
		}
		role := "assistant"
		if pt == "user_message" {
			role = "user"
		}
		return Message{
			Seq:         seq,
			OccurredAt:  ts,
			Role:        role,
			ContentText: extractText(payload),
			ContentJSON: resultJSON(payload),
		}, true
	default:
		return Message{}, false
	}

	role := obj.Get("role").String()
	text := extractText(obj.Get("content"))
	if role == "" || text == "" {
		return Message{}, false
	}
	return Message{
		Seq:         seq,
		OccurredAt:  ts,
		Role:        role,
		ContentText: text,
		ContentJSON: resultJSON(obj.Get("content")),
	}, true
}

func normalizeToolEvent(seq int, ts *time.Time, line []byte) (ToolEvent, bool) {
	eventType := gjson.GetBytes(line, "type").String()
	payload := gjson.GetBytes(line, "payload")
	obj := gjson.ParseBytes(line)
	if eventType == "response_item" {
		obj = payload
		eventType = payload.Get("type").String()
	}

	switch eventType {
	case "function_call", "custom_tool_call", "tool_search_call":
		return ToolEvent{
			Seq:         seq,
			OccurredAt:  ts,
			Kind:        eventType,
			ToolName:    firstNonEmpty(obj.Get("name").String(), obj.Get("namespace").String()),
			CallID:      obj.Get("call_id").String(),
			Status:      obj.Get("status").String(),
			InputText:   firstNonEmpty(obj.Get("arguments").String(), obj.Get("input").String()),
			PayloadJSON: resultJSON(obj),
		}, true
	case "function_call_output", "custom_tool_call_output", "tool_search_output":
		return ToolEvent{
			Seq:         seq,
			OccurredAt:  ts,
			Kind:        eventType,
			CallID:      obj.Get("call_id").String(),
			Status:      obj.Get("status").String(),
			OutputText:  firstNonEmpty(obj.Get("output").String(), extractText(obj)),
			PayloadJSON: resultJSON(obj),
		}, true
	case "event_msg":
		pt := payload.Get("type").String()
		if !strings.Contains(pt, "exec_command") && !strings.Contains(pt, "patch_apply") && !strings.Contains(pt, "mcp_tool") && !strings.Contains(pt, "web_search") {
			return ToolEvent{}, false
		}
		return ToolEvent{
			Seq:         seq,
			OccurredAt:  ts,
			Kind:        pt,
			ToolName:    firstNonEmpty(payload.Get("name").String(), payload.Get("command.0").String()),
			CallID:      payload.Get("call_id").String(),
			Status:      firstNonEmpty(payload.Get("status").String(), payload.Get("exit_code").String()),
			InputText:   firstNonEmpty(payload.Get("command").Raw, payload.Get("input").String()),
			OutputText:  firstNonEmpty(payload.Get("aggregated_output").String(), payload.Get("stdout").String(), payload.Get("stderr").String(), payload.Get("output").String()),
			PayloadJSON: resultJSON(payload),
		}, true
	default:
		return ToolEvent{}, false
	}
}

func eventContentText(line []byte) string {
	eventType := gjson.GetBytes(line, "type").String()
	if eventType == "response_item" || eventType == "event_msg" {
		return extractText(gjson.GetBytes(line, "payload"))
	}
	return extractText(gjson.ParseBytes(line))
}

func extractText(value gjson.Result) string {
	var parts []string
	collectText(value, &parts, 0)
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

func collectText(value gjson.Result, parts *[]string, depth int) {
	if depth > 8 || !value.Exists() {
		return
	}
	switch {
	case value.IsArray():
		value.ForEach(func(_, v gjson.Result) bool {
			collectText(v, parts, depth+1)
			return true
		})
	case value.IsObject():
		for _, key := range []string{"text", "content", "output", "aggregated_output", "stdout", "stderr", "summary"} {
			if v := value.Get(key); v.Exists() {
				collectText(v, parts, depth+1)
			}
		}
	case value.Type == gjson.String:
		text := strings.TrimSpace(value.String())
		if text != "" {
			*parts = append(*parts, text)
		}
	}
}

func firstJSON(line []byte, paths ...string) string {
	for _, path := range paths {
		if v := gjson.GetBytes(line, path).String(); v != "" {
			return v
		}
	}
	return ""
}

func parseTimePtr(value string) *time.Time {
	if value == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		t, err = time.Parse(time.RFC3339, value)
		if err != nil {
			return nil
		}
	}
	return &t
}

func hashEvent(seq int, value []byte) string {
	hasher := sha256.New()
	_, _ = fmt.Fprintf(hasher, "%d\n", seq)
	_, _ = hasher.Write(value)
	sum := hasher.Sum(nil)
	return hex.EncodeToString(sum[:])
}

func cloneJSON(value []byte) []byte {
	out := make([]byte, len(value))
	copy(out, value)
	return out
}

func resultJSON(value gjson.Result) []byte {
	if value.Raw != "" {
		return []byte(value.Raw)
	}
	b, _ := json.Marshal(value.Value())
	return b
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
