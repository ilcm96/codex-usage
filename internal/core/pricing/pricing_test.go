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
	if _, ok := pr.GetModelPricing("gpt-5.5"); !ok {
		t.Fatalf("expected pricing for gpt-5.5")
	}
	for _, model := range []string{"gpt-5.6", "gpt-5.6-sol", "gpt-5.6-terra", "gpt-5.6-luna"} {
		if _, ok := pr.GetModelPricing(model); !ok {
			t.Fatalf("expected pricing for %s", model)
		}
	}
	if _, ok := pr.GetModelPricing("gpt-5.4-codex"); ok {
		t.Fatalf("expected no pricing for gpt-5.4-codex")
	}
	// Should not fuzzy-match into gpt-5.2.
	if _, ok := pr.GetModelPricing("gpt-5.2-pro"); ok {
		t.Fatalf("expected no pricing for gpt-5.2-pro (exact match only)")
	}
	if _, ok := pr.GetModelPricing("gpt-5.4-pro"); ok {
		t.Fatalf("expected no pricing for gpt-5.4-pro (not in embedded snapshot)")
	}
}

func TestGetModelPricing_GPT56FamilyPricing(t *testing.T) {
	pr, err := LoadEmbeddedPricing()
	if err != nil {
		t.Fatalf("LoadEmbeddedPricing: %v", err)
	}

	tests := []struct {
		model      string
		input      float64
		cacheRead  float64
		cacheWrite float64
		output     float64
	}{
		{model: "gpt-5.6", input: 5e-06, cacheRead: 5e-07, cacheWrite: 6.25e-06, output: 3e-05},
		{model: "gpt-5.6-sol", input: 5e-06, cacheRead: 5e-07, cacheWrite: 6.25e-06, output: 3e-05},
		{model: "gpt-5.6-terra", input: 2.5e-06, cacheRead: 2.5e-07, cacheWrite: 3.125e-06, output: 1.5e-05},
		{model: "gpt-5.6-luna", input: 1e-06, cacheRead: 1e-07, cacheWrite: 1.25e-06, output: 6e-06},
	}
	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			got, ok := pr.GetModelPricing(tt.model)
			if !ok {
				t.Fatalf("expected pricing for %s", tt.model)
			}
			if got.InputCostPerToken != tt.input ||
				got.CacheReadInputTokenCost != tt.cacheRead ||
				got.CacheWriteInputTokenCost != tt.cacheWrite ||
				got.OutputCostPerToken != tt.output {
				t.Fatalf("unexpected pricing for %s: %+v", tt.model, got)
			}
		})
	}
}

func TestCostUSD_SeparatesCacheReadAndWrite(t *testing.T) {
	got := CostUSD(TokenUsage{
		InputTokens:      100,
		CacheReadTokens:  20,
		CacheWriteTokens: 30,
		OutputTokens:     10,
	}, ModelPricing{
		InputCostPerToken:        5,
		CacheReadInputTokenCost:  0.5,
		CacheWriteInputTokenCost: 6.25,
		OutputCostPerToken:       30,
	})
	want := float64(50*5 + 20*0.5 + 30*6.25 + 10*30)
	if got != want {
		t.Fatalf("unexpected cost: got %v want %v", got, want)
	}
}

func TestGetModelPricing_GPT55Pricing(t *testing.T) {
	pr, err := LoadEmbeddedPricing()
	if err != nil {
		t.Fatalf("LoadEmbeddedPricing: %v", err)
	}

	got, ok := pr.GetModelPricing("gpt-5.5")
	if !ok {
		t.Fatalf("expected pricing for gpt-5.5")
	}

	if got.InputCostPerToken != 5e-06 {
		t.Fatalf("unexpected input cost: got %v", got.InputCostPerToken)
	}
	if got.CacheReadInputTokenCost != 5e-07 {
		t.Fatalf("unexpected cache read cost: got %v", got.CacheReadInputTokenCost)
	}
	if got.OutputCostPerToken != 3e-05 {
		t.Fatalf("unexpected output cost: got %v", got.OutputCostPerToken)
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

func TestCostForModelUSD_FallsBackToDefaultModel(t *testing.T) {
	pr, err := LoadEmbeddedPricing()
	if err != nil {
		t.Fatalf("LoadEmbeddedPricing: %v", err)
	}

	got := pr.CostForModelUSD("unknown-model", TokenUsage{
		InputTokens:     100,
		CacheReadTokens: 20,
		OutputTokens:    50,
	}, "gpt-5")

	gpt5Pricing, ok := pr.GetModelPricing("gpt-5")
	if !ok {
		t.Fatalf("expected pricing for gpt-5")
	}
	want := CostUSD(TokenUsage{
		InputTokens:     100,
		CacheReadTokens: 20,
		OutputTokens:    50,
	}, gpt5Pricing)

	if got != want {
		t.Fatalf("unexpected cost: got %v want %v", got, want)
	}
}
