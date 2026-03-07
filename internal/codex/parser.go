package codex

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/tidwall/gjson"
)

type FileParseResult struct {
	Size  int64
	Mtime int64
	// Daily[dateKey][model]usage
	Daily map[string]map[string]Usage
}

type RawUsage struct {
	Input     int64
	Cached    int64
	Output    int64
	Reasoning int64
	Total     int64
}

func normalizeRawUsage(obj gjson.Result) (RawUsage, bool) {
	if !obj.Exists() || obj.Type != gjson.JSON {
		return RawUsage{}, false
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

	return RawUsage{
		Input:     input,
		Cached:    cached,
		Output:    output,
		Reasoning: reasoning,
		Total:     total,
	}, true
}

func subtractRawUsage(cur RawUsage, prev *RawUsage) RawUsage {
	p := RawUsage{}
	if prev != nil {
		p = *prev
	}

	sub := func(a, b int64) int64 {
		if a-b <= 0 {
			return 0
		}
		return a - b
	}

	return RawUsage{
		Input:     sub(cur.Input, p.Input),
		Cached:    sub(cur.Cached, p.Cached),
		Output:    sub(cur.Output, p.Output),
		Reasoning: sub(cur.Reasoning, p.Reasoning),
		Total:     sub(cur.Total, p.Total),
	}
}

func convertToDelta(raw RawUsage) Usage {
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

func extractModel(line []byte) string {
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

func dateKey(timestamp string, loc *time.Location) (string, bool) {
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

func ParseSessionFile(path string, size int64, mtime int64, loc *time.Location) (FileParseResult, error) {
	f, err := os.Open(path)
	if err != nil {
		return FileParseResult{}, fmt.Errorf("failed to open %s: %w", path, err)
	}
	defer f.Close()

	res := FileParseResult{
		Size:  size,
		Mtime: mtime,
		Daily: map[string]map[string]Usage{},
	}

	reader := bufio.NewScanner(f)
	reader.Buffer(make([]byte, 64*1024), 8*1024*1024)

	var prevTotals *RawUsage
	var currentModel string
	var currentModelIsFallback bool

	for reader.Scan() {
		line := bytes.TrimSpace(reader.Bytes())
		if len(line) == 0 {
			continue
		}

		entryType := gjson.GetBytes(line, "type").String()
		if entryType == "turn_context" {
			if m := extractModel(line); m != "" {
				currentModel = m
				currentModelIsFallback = false
			}
			continue
		}

		if entryType != "event_msg" {
			continue
		}

		if gjson.GetBytes(line, "payload.type").String() != "token_count" {
			continue
		}

		ts := gjson.GetBytes(line, "timestamp").String()
		day, ok := dateKey(ts, loc)
		if !ok {
			continue
		}

		lastObj := gjson.GetBytes(line, "payload.info.last_token_usage")
		totalObj := gjson.GetBytes(line, "payload.info.total_token_usage")

		lastRaw, hasLast := normalizeRawUsage(lastObj)
		totalRaw, hasTotal := normalizeRawUsage(totalObj)

		raw := RawUsage{}
		hasRaw := false

		if hasLast {
			raw = lastRaw
			hasRaw = true
		} else if hasTotal {
			raw = subtractRawUsage(totalRaw, prevTotals)
			hasRaw = true
		}

		if hasTotal {
			tmp := totalRaw
			prevTotals = &tmp
		}

		if !hasRaw {
			continue
		}

		delta := convertToDelta(raw)
		if delta.InputTokens == 0 && delta.CachedInputTokens == 0 && delta.OutputTokens == 0 && delta.ReasoningOutputTokens == 0 {
			continue
		}

		extractedModel := extractModel(line)
		isFallbackModel := false
		if extractedModel != "" {
			currentModel = extractedModel
			currentModelIsFallback = false
		}

		model := extractedModel
		if model == "" {
			model = currentModel
		}
		if model == "" {
			model = "gpt-5"
			isFallbackModel = true
			currentModel = model
			currentModelIsFallback = true
		} else if extractedModel == "" && currentModelIsFallback {
			isFallbackModel = true
		}

		delta.IsFallbackModel = isFallbackModel

		dayMap := res.Daily[day]
		if dayMap == nil {
			dayMap = map[string]Usage{}
			res.Daily[day] = dayMap
		}
		u := dayMap[model]
		u.Add(delta)
		dayMap[model] = u
	}

	if err := reader.Err(); err != nil {
		return FileParseResult{}, fmt.Errorf("failed to read %s: %w", path, err)
	}
	return res, nil
}
