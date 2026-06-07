package codexlog

import (
	"strings"
	"time"

	"github.com/tidwall/gjson"
)

const DefaultFallbackModel = "gpt-5"

type Usage struct {
	InputTokens           int64 `json:"inputTokens"`
	CachedInputTokens     int64 `json:"cachedInputTokens"`
	OutputTokens          int64 `json:"outputTokens"`
	ReasoningOutputTokens int64 `json:"reasoningOutputTokens"`
	TotalTokens           int64 `json:"totalTokens"`
	IsFallbackModel       bool  `json:"isFallbackModel,omitempty"`
}

func (u *Usage) Add(v Usage) {
	u.InputTokens += v.InputTokens
	u.CachedInputTokens += v.CachedInputTokens
	u.OutputTokens += v.OutputTokens
	u.ReasoningOutputTokens += v.ReasoningOutputTokens
	u.TotalTokens += v.TotalTokens
	u.IsFallbackModel = u.IsFallbackModel || v.IsFallbackModel
}

func (u Usage) IsZero() bool {
	return u.InputTokens == 0 &&
		u.CachedInputTokens == 0 &&
		u.OutputTokens == 0 &&
		u.ReasoningOutputTokens == 0
}

type UsageEvent struct {
	Timestamp string
	Model     string
	Usage     Usage
}

type rawUsage struct {
	Input     int64
	Cached    int64
	Output    int64
	Reasoning int64
	Total     int64
}

type UsageNormalizer struct {
	prevTotals             *rawUsage
	currentModel           string
	currentModelIsFallback bool
	fallbackModel          string
}

func NewUsageNormalizer(fallbackModel string) *UsageNormalizer {
	if strings.TrimSpace(fallbackModel) == "" {
		fallbackModel = DefaultFallbackModel
	}
	return &UsageNormalizer{fallbackModel: fallbackModel}
}

func (n *UsageNormalizer) ObserveModel(line []byte) bool {
	if gjson.GetBytes(line, "type").String() != "turn_context" {
		return false
	}
	if m := ExtractModel(line); m != "" {
		n.currentModel = m
		n.currentModelIsFallback = false
	}
	return true
}

func (n *UsageNormalizer) NormalizeUsageLine(line []byte) (UsageEvent, bool) {
	if !IsTokenCountEvent(line) {
		return UsageEvent{}, false
	}

	usage, total, ok := normalizeUsageFromLine(line, n.prevTotals)
	if total != nil {
		n.prevTotals = total
	}
	if !ok || usage.IsZero() {
		return UsageEvent{}, false
	}

	extractedModel := ExtractModel(line)
	isFallbackModel := false
	if extractedModel != "" {
		n.currentModel = extractedModel
		n.currentModelIsFallback = false
	}

	model := extractedModel
	if model == "" {
		model = n.currentModel
	}
	if model == "" {
		model = n.fallbackModel
		isFallbackModel = true
		n.currentModel = model
		n.currentModelIsFallback = true
	} else if extractedModel == "" && n.currentModelIsFallback {
		isFallbackModel = true
	}

	usage.IsFallbackModel = isFallbackModel

	return UsageEvent{
		Timestamp: Timestamp(line),
		Model:     model,
		Usage:     usage,
	}, true
}

func IsTokenCountEvent(line []byte) bool {
	return gjson.GetBytes(line, "type").String() == "event_msg" &&
		gjson.GetBytes(line, "payload.type").String() == "token_count"
}

func Timestamp(line []byte) string {
	return gjson.GetBytes(line, "timestamp").String()
}

func DateKey(timestamp string, loc *time.Location) (string, bool) {
	if timestamp == "" {
		return "", false
	}
	t, err := time.Parse(time.RFC3339Nano, timestamp)
	if err != nil {
		t, err = time.Parse(time.RFC3339, timestamp)
		if err != nil {
			return "", false
		}
	}
	return t.In(loc).Format("2006-01-02"), true
}

func ExtractModel(line []byte) string {
	candidates := []string{
		"payload.info.model",
		"payload.info.model_name",
		"payload.info.metadata.model",
		"payload.model",
		"payload.metadata.model",
	}
	for _, path := range candidates {
		v := strings.TrimSpace(gjson.GetBytes(line, path).String())
		if v != "" {
			return v
		}
	}
	return ""
}

func normalizeUsageFromLine(line []byte, prevTotals *rawUsage) (Usage, *rawUsage, bool) {
	lastObj := gjson.GetBytes(line, "payload.info.last_token_usage")
	totalObj := gjson.GetBytes(line, "payload.info.total_token_usage")

	lastRaw, hasLast := normalizeRawUsage(lastObj)
	totalRaw, hasTotal := normalizeRawUsage(totalObj)

	var totalPtr *rawUsage
	if hasTotal {
		tmp := totalRaw
		totalPtr = &tmp
	}

	if hasLast {
		return convertRawUsage(lastRaw), totalPtr, true
	}
	if hasTotal {
		return convertRawUsage(subtractRawUsage(totalRaw, prevTotals)), totalPtr, true
	}
	return Usage{}, totalPtr, false
}

func normalizeRawUsage(obj gjson.Result) (rawUsage, bool) {
	if !obj.Exists() || obj.Type != gjson.JSON {
		return rawUsage{}, false
	}

	input := obj.Get("input_tokens").Int()
	// Match JS-style "nullish coalescing" behavior:
	// use cache_read_input_tokens only when cached_input_tokens is missing, not when it's 0.
	cachedRes := obj.Get("cached_input_tokens")
	cached := int64(0)
	if cachedRes.Exists() {
		cached = cachedRes.Int()
	} else {
		cached = obj.Get("cache_read_input_tokens").Int()
	}
	output := obj.Get("output_tokens").Int()
	reasoning := obj.Get("reasoning_output_tokens").Int()
	total := obj.Get("total_tokens").Int()
	if total <= 0 {
		total = input + output
	}

	return rawUsage{
		Input:     input,
		Cached:    cached,
		Output:    output,
		Reasoning: reasoning,
		Total:     total,
	}, true
}

func subtractRawUsage(cur rawUsage, prev *rawUsage) rawUsage {
	p := rawUsage{}
	if prev != nil {
		p = *prev
	}

	sub := func(a, b int64) int64 {
		if a-b <= 0 {
			return 0
		}
		return a - b
	}

	return rawUsage{
		Input:     sub(cur.Input, p.Input),
		Cached:    sub(cur.Cached, p.Cached),
		Output:    sub(cur.Output, p.Output),
		Reasoning: sub(cur.Reasoning, p.Reasoning),
		Total:     sub(cur.Total, p.Total),
	}
}

func convertRawUsage(raw rawUsage) Usage {
	total := raw.Total
	if total <= 0 {
		total = raw.Input + raw.Output
	}
	cached := raw.Cached
	if cached > raw.Input {
		cached = raw.Input
	}
	return Usage{
		InputTokens:           raw.Input,
		CachedInputTokens:     cached,
		OutputTokens:          raw.Output,
		ReasoningOutputTokens: raw.Reasoning,
		TotalTokens:           total,
	}
}
