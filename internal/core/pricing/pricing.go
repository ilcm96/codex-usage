package pricing

import (
	"strings"
)

type ModelPricing struct {
	InputCostPerToken        float64 `json:"input_cost_per_token"`
	OutputCostPerToken       float64 `json:"output_cost_per_token"`
	CacheReadInputTokenCost  float64 `json:"cache_read_input_token_cost"`
	CacheWriteInputTokenCost float64 `json:"cache_write_input_token_cost"`
}

type Pricing struct {
	models map[string]ModelPricing
}

type TokenUsage struct {
	InputTokens      int64
	CacheReadTokens  int64
	CacheWriteTokens int64
	OutputTokens     int64
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

func (p Pricing) CostForModelUSD(model string, tokens TokenUsage, fallbackModel string) float64 {
	prc, ok := p.GetModelPricing(model)
	if !ok {
		prc, ok = p.GetModelPricing(fallbackModel)
	}
	if !ok {
		return 0
	}
	return CostUSD(tokens, prc)
}

func CostUSD(tokens TokenUsage, pricing ModelPricing) float64 {
	input := max(tokens.InputTokens, 0)
	cacheRead := min(max(tokens.CacheReadTokens, 0), input)
	cacheWrite := min(max(tokens.CacheWriteTokens, 0), input-cacheRead)
	nonCached := float64(input - cacheRead - cacheWrite)
	output := float64(tokens.OutputTokens)
	if output < 0 {
		output = 0
	}

	cacheReadCost := pricing.CacheReadInputTokenCost
	if cacheReadCost == 0 {
		cacheReadCost = pricing.InputCostPerToken
	}
	cacheWriteCost := pricing.CacheWriteInputTokenCost
	if cacheWriteCost == 0 {
		cacheWriteCost = pricing.InputCostPerToken
	}

	// Pricing fields are in USD per token.
	return nonCached*pricing.InputCostPerToken +
		float64(cacheRead)*cacheReadCost +
		float64(cacheWrite)*cacheWriteCost +
		output*pricing.OutputCostPerToken
}
