package codexlog

import (
	"testing"

	"github.com/tidwall/gjson"
)

func TestUsageNormalizer_TotalUsageSubtract(t *testing.T) {
	normalizer := NewUsageNormalizer(DefaultFallbackModel)
	normalizer.ObserveModel([]byte(`{"type":"turn_context","payload":{"model":"gpt-5.2-codex"}}`))

	first, ok := normalizer.NormalizeUsageLine([]byte(`{"timestamp":"2026-02-08T00:00:01.000Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":100,"cached_input_tokens":20,"cache_write_input_tokens":30,"output_tokens":50,"reasoning_output_tokens":0,"total_tokens":150}}}}`))
	if !ok {
		t.Fatal("expected first usage event")
	}
	second, ok := normalizer.NormalizeUsageLine([]byte(`{"timestamp":"2026-02-08T00:00:02.000Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":150,"cached_input_tokens":25,"cache_write_input_tokens":40,"output_tokens":70,"reasoning_output_tokens":0,"total_tokens":220}}}}`))
	if !ok {
		t.Fatal("expected second usage event")
	}

	if first.Model != "gpt-5.2-codex" {
		t.Fatalf("unexpected first model: %s", first.Model)
	}
	if first.Usage.InputTokens != 100 || first.Usage.CachedInputTokens != 20 || first.Usage.CacheWriteInputTokens != 30 || first.Usage.OutputTokens != 50 || first.Usage.TotalTokens != 150 {
		t.Fatalf("unexpected first usage: %#v", first.Usage)
	}
	if second.Usage.InputTokens != 50 || second.Usage.CachedInputTokens != 5 || second.Usage.CacheWriteInputTokens != 10 || second.Usage.OutputTokens != 20 || second.Usage.TotalTokens != 70 {
		t.Fatalf("unexpected second usage: %#v", second.Usage)
	}
}

func TestUsageNormalizer_FallbackModel(t *testing.T) {
	normalizer := NewUsageNormalizer(DefaultFallbackModel)

	got, ok := normalizer.NormalizeUsageLine([]byte(`{"timestamp":"2026-02-08T00:00:01.000Z","type":"event_msg","payload":{"type":"token_count","info":{"last_token_usage":{"input_tokens":10,"cached_input_tokens":0,"output_tokens":5,"reasoning_output_tokens":0,"total_tokens":15}}}}`))
	if !ok {
		t.Fatal("expected usage event")
	}
	if got.Model != DefaultFallbackModel {
		t.Fatalf("unexpected model: %s", got.Model)
	}
	if !got.Usage.IsFallbackModel {
		t.Fatalf("expected fallback model flag: %#v", got.Usage)
	}
}

func TestNormalizeRawUsage_UsesCacheReadWhenCachedInputMissing(t *testing.T) {
	raw, ok := normalizeRawUsage(gjson.Parse(`{"input_tokens":100,"cache_read_input_tokens":30,"output_tokens":50}`))
	if !ok {
		t.Fatal("expected raw usage")
	}
	if raw.Cached != 30 {
		t.Fatalf("unexpected cached tokens: %d", raw.Cached)
	}
}

func TestUsageNormalizer_TracksCacheWriteTokens(t *testing.T) {
	normalizer := NewUsageNormalizer(DefaultFallbackModel)
	normalizer.ObserveModel([]byte(`{"type":"turn_context","payload":{"model":"gpt-5.6-sol"}}`))

	got, ok := normalizer.NormalizeUsageLine([]byte(`{"timestamp":"2026-07-21T00:00:01Z","type":"event_msg","payload":{"type":"token_count","info":{"last_token_usage":{"input_tokens":100,"cached_input_tokens":20,"cache_write_input_tokens":30,"output_tokens":10,"total_tokens":110}}}}`))
	if !ok {
		t.Fatal("expected usage event")
	}
	if got.Usage.CacheWriteInputTokens != 30 {
		t.Fatalf("unexpected cache write tokens: %#v", got.Usage)
	}
}
