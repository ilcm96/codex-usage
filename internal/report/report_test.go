package report

import (
	"math"
	"testing"

	"github.com/ilcm96/codex-usage/internal/codex"
	"github.com/ilcm96/codex-usage/internal/pricing"
)

func TestFinalizeRow_UnknownModelFallsBackToGPT5Pricing(t *testing.T) {
	pr, err := pricing.LoadEmbeddedPricing()
	if err != nil {
		t.Fatalf("LoadEmbeddedPricing: %v", err)
	}

	unknownModel := "some-new-model"
	u := codex.Usage{
		InputTokens:       100,
		CachedInputTokens: 20,
		OutputTokens:      50,
		TotalTokens:       150,
	}
	row := Row{
		Key: "2026-02-08",
		Models: map[string]codex.Usage{
			unknownModel: u,
		},
	}

	finalizeRow(&row, pr)

	gpt5Pricing, ok := pr.GetModelPricing("gpt-5")
	if !ok {
		t.Fatalf("expected embedded gpt-5 pricing to exist")
	}

	expected := pricing.CostUSD(struct {
		InputTokens     int64
		CacheReadTokens int64
		OutputTokens    int64
	}{
		InputTokens:     u.InputTokens,
		CacheReadTokens: u.CachedInputTokens,
		OutputTokens:    u.OutputTokens,
	}, gpt5Pricing)

	if math.Abs(row.CostUSD-expected) > 1e-12 {
		t.Fatalf("unexpected cost: got %.15f expected %.15f", row.CostUSD, expected)
	}
}
