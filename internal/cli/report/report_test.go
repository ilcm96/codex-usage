package report

import (
	"math"
	"testing"

	"github.com/ilcm96/codex-usage/internal/core/codexlog"
	"github.com/ilcm96/codex-usage/internal/core/pricing"
)

func TestFinalizeRow_UnknownModelFallsBackToGPT5Pricing(t *testing.T) {
	pr, err := pricing.LoadEmbeddedPricing()
	if err != nil {
		t.Fatalf("LoadEmbeddedPricing: %v", err)
	}

	unknownModel := "some-new-model"
	u := codexlog.Usage{
		InputTokens:       100,
		CachedInputTokens: 20,
		OutputTokens:      50,
		TotalTokens:       150,
	}
	row := Row{
		Key: "2026-02-08",
		Models: map[string]codexlog.Usage{
			unknownModel: u,
		},
	}

	finalizeRow(&row, pr)

	expected := pr.CostForModelUSD(unknownModel, pricing.TokenUsage{
		InputTokens:     u.InputTokens,
		CacheReadTokens: u.CachedInputTokens,
		OutputTokens:    u.OutputTokens,
	}, codexlog.DefaultFallbackModel)

	if math.Abs(row.CostUSD-expected) > 1e-12 {
		t.Fatalf("unexpected cost: got %.15f expected %.15f", row.CostUSD, expected)
	}
}
