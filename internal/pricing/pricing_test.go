package pricing

import "testing"

func TestGetModelPricing_ExactMatchOnly(t *testing.T) {
	pr, err := LoadEmbeddedPricing()
	if err != nil {
		t.Fatalf("LoadEmbeddedPricing: %v", err)
	}

	if _, ok := pr.GetModelPricing("gpt-5"); !ok {
		t.Fatalf("expected pricing for gpt-5")
	}
	if _, ok := pr.GetModelPricing("gpt-5.2"); !ok {
		t.Fatalf("expected pricing for gpt-5.2")
	}
	if _, ok := pr.GetModelPricing("gpt-5.4"); !ok {
		t.Fatalf("expected pricing for gpt-5.4")
	}
	if _, ok := pr.GetModelPricing("gpt-5.4-codex"); !ok {
		t.Fatalf("expected pricing for gpt-5.4-codex")
	}
	// Should not fuzzy-match into gpt-5.2.
	if _, ok := pr.GetModelPricing("gpt-5.2-pro"); ok {
		t.Fatalf("expected no pricing for gpt-5.2-pro (exact match only)")
	}
	if _, ok := pr.GetModelPricing("gpt-5.4-pro"); ok {
		t.Fatalf("expected no pricing for gpt-5.4-pro (not in embedded snapshot)")
	}
}

func TestGetModelPricing_GPT54Pricing(t *testing.T) {
	pr, err := LoadEmbeddedPricing()
	if err != nil {
		t.Fatalf("LoadEmbeddedPricing: %v", err)
	}

	got, ok := pr.GetModelPricing("gpt-5.4")
	if !ok {
		t.Fatalf("expected pricing for gpt-5.4")
	}

	if got.InputCostPerToken != 2.5e-06 {
		t.Fatalf("unexpected input cost: got %v", got.InputCostPerToken)
	}
	if got.CacheReadInputTokenCost != 2.5e-07 {
		t.Fatalf("unexpected cache read cost: got %v", got.CacheReadInputTokenCost)
	}
	if got.OutputCostPerToken != 1.5e-05 {
		t.Fatalf("unexpected output cost: got %v", got.OutputCostPerToken)
	}
}
