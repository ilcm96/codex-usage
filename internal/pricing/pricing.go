package pricing

import (
	"strings"
)

type ModelPricing struct {
	InputCostPerToken       float64 `json:"input_cost_per_token"`
	OutputCostPerToken      float64 `json:"output_cost_per_token"`
	CacheReadInputTokenCost float64 `json:"cache_read_input_token_cost"`
}

type Pricing struct {
	models map[string]ModelPricing
}

func LoadEmbeddedPricing() (Pricing, error) {
	models := make(map[string]ModelPricing, len(embeddedSnapshot))
	for _, e := range embeddedSnapshot {
		models[e.Model] = e.Pricing
	}
	return Pricing{models: models}, nil
}

// GetModelPricing returns pricing only when the model name matches exactly.
func (p Pricing) GetModelPricing(model string) (ModelPricing, bool) {
	m := strings.TrimSpace(model)
	if m == "" {
		return ModelPricing{}, false
	}

	if p.models == nil {
		return ModelPricing{}, false
	}
	if v, ok := p.models[m]; ok {
		return v, true
	}
	return ModelPricing{}, false
}

func CostUSD(tokens struct {
	InputTokens     int64
	CacheReadTokens int64
	OutputTokens    int64
}, pricing ModelPricing) float64 {
	nonCached := float64(tokens.InputTokens - tokens.CacheReadTokens)
	if nonCached < 0 {
		nonCached = 0
	}
	cached := float64(tokens.CacheReadTokens)
	if cached < 0 {
		cached = 0
	}
	output := float64(tokens.OutputTokens)
	if output < 0 {
		output = 0
	}

	cacheReadCost := pricing.CacheReadInputTokenCost
	if cacheReadCost == 0 {
		cacheReadCost = pricing.InputCostPerToken
	}

	// Pricing fields are in USD per token.
	return nonCached*pricing.InputCostPerToken + cached*cacheReadCost + output*pricing.OutputCostPerToken
}
