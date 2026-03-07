package codex

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseSessionFile_LastUsage(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.jsonl")
	content := "" +
		`{"timestamp":"2026-02-08T00:00:00.000Z","type":"turn_context","payload":{"model":"gpt-5"}}` + "\n" +
		`{"timestamp":"2026-02-08T00:00:01.000Z","type":"event_msg","payload":{"type":"token_count","info":{"last_token_usage":{"input_tokens":100,"cached_input_tokens":20,"output_tokens":50,"reasoning_output_tokens":10,"total_tokens":150}}}}` + "\n"
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	st, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ParseSessionFile(p, st.Size(), st.ModTime().Unix(), time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	day := "2026-02-08"
	dayMap := got.Daily[day]
	if dayMap == nil {
		t.Fatalf("missing day %s", day)
	}
	u := dayMap["gpt-5"]
	if u.InputTokens != 100 || u.CachedInputTokens != 20 || u.OutputTokens != 50 || u.ReasoningOutputTokens != 10 || u.TotalTokens != 150 {
		t.Fatalf("unexpected usage: %#v", u)
	}
}

func TestParseSessionFile_TotalUsageSubtract(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "b.jsonl")
	content := "" +
		`{"timestamp":"2026-02-08T00:00:00.000Z","type":"turn_context","payload":{"model":"gpt-5.2-codex"}}` + "\n" +
		`{"timestamp":"2026-02-08T00:00:01.000Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":100,"cached_input_tokens":20,"output_tokens":50,"reasoning_output_tokens":0,"total_tokens":150}}}}` + "\n" +
		`{"timestamp":"2026-02-08T00:00:02.000Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":150,"cached_input_tokens":25,"output_tokens":70,"reasoning_output_tokens":0,"total_tokens":220}}}}` + "\n"
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ParseSessionFile(p, st.Size(), st.ModTime().Unix(), time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	u := got.Daily["2026-02-08"]["gpt-5.2-codex"]
	// First event: 100/20/50/150, Second event: delta 50/5/20/70 -> totals 150/25/70/220
	if u.InputTokens != 150 || u.CachedInputTokens != 25 || u.OutputTokens != 70 || u.TotalTokens != 220 {
		t.Fatalf("unexpected usage: %#v", u)
	}
}

func TestParseSessionFile_FallbackModel(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "c.jsonl")
	content := "" +
		`{"timestamp":"2026-02-08T00:00:01.000Z","type":"event_msg","payload":{"type":"token_count","info":{"last_token_usage":{"input_tokens":10,"cached_input_tokens":0,"output_tokens":5,"reasoning_output_tokens":0,"total_tokens":15}}}}` + "\n"
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ParseSessionFile(p, st.Size(), st.ModTime().Unix(), time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	u := got.Daily["2026-02-08"]["gpt-5"]
	if !u.IsFallbackModel {
		t.Fatalf("expected fallback model flag: %#v", u)
	}
}
