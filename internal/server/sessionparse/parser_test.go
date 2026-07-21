package sessionparse

import (
	"testing"

	"github.com/ilcm96/codex-usage/internal/core/pricing"
	"github.com/ilcm96/codex-usage/internal/server/servertest"
)

func TestParseRaw(t *testing.T) {
	raw := []byte(
		`{"timestamp":"2026-02-08T00:00:00Z","type":"session_meta","payload":{"id":"parsed-session","timestamp":"2026-02-08T00:00:00Z","cwd":"/repo/parsed","originator":"codex","source":"cli","cli_version":"1.0.0","model_provider":"openai","git":{"repository_url":"https://github.com/acme/parsed","branch":"main","commit_hash":"abc123"}}}` + "\n" +
			`{"timestamp":"2026-02-08T00:00:01Z","type":"turn_context","payload":{"model":"gpt-5.6-sol"}}` + "\n" +
			`{"timestamp":"2026-02-08T00:00:02Z","type":"message","role":"user","content":"Parse this session."}` + "\n" +
			`{"timestamp":"2026-02-08T00:00:03Z","type":"function_call","name":"shell","call_id":"call-1","status":"completed","arguments":"go test"}` + "\n" +
			`{"timestamp":"2026-02-08T00:00:04Z","type":"function_call_output","call_id":"call-1","status":"completed","output":"ok"}` + "\n" +
			`{"timestamp":"2026-02-08T00:00:05Z","type":"message","role":"assistant","content":"Parsed."}` + "\n" +
			`{"timestamp":"2026-02-08T00:00:06Z","type":"event_msg","payload":{"type":"token_count","info":{"last_token_usage":{"input_tokens":10,"cached_input_tokens":2,"cache_write_input_tokens":3,"output_tokens":5,"reasoning_output_tokens":1,"total_tokens":16}}}}` + "\n",
	)
	rawFixture := servertest.WriteRaw(t, raw, "parsed-session.jsonl")
	pr, err := pricing.LoadEmbeddedPricing()
	if err != nil {
		t.Fatalf("LoadEmbeddedPricing failed: %v", err)
	}

	got, err := ParseRaw(rawFixture.Path, FallbackMeta{
		SessionID: "fallback-session",
	}, pr)
	if err != nil {
		t.Fatalf("ParseRaw failed: %v", err)
	}
	if got.Meta.ID != "parsed-session" || got.Meta.CWD != "/repo/parsed" || got.Meta.Branch != "main" {
		t.Fatalf("unexpected parsed meta: %+v", got.Meta)
	}
	if len(got.Events) != 7 || len(got.Messages) != 2 || len(got.Tools) != 2 || len(got.Usages) != 1 {
		t.Fatalf("unexpected parsed counts: events=%d messages=%d tools=%d usages=%d", len(got.Events), len(got.Messages), len(got.Tools), len(got.Usages))
	}
	if got.Usages[0].Model != "gpt-5.6-sol" || got.Usages[0].CacheWriteInputTokens != 3 || got.Usages[0].TotalTokens != 16 || got.Usages[0].CostUSD <= 0 {
		t.Fatalf("unexpected usage: %+v", got.Usages[0])
	}
	if got.StartedAt == nil || got.UpdatedAt == nil || !got.UpdatedAt.After(*got.StartedAt) {
		t.Fatalf("unexpected parsed times: started=%v updated=%v", got.StartedAt, got.UpdatedAt)
	}
}
