package pricing

type embeddedEntry struct {
	Model   string
	Pricing ModelPricing
}

var embeddedSnapshot = []embeddedEntry{
	{Model: "gpt-5", Pricing: ModelPricing{InputCostPerToken: 1.25e-06, OutputCostPerToken: 1e-05, CacheReadInputTokenCost: 1.25e-07}},
	{Model: "gpt-5-codex", Pricing: ModelPricing{InputCostPerToken: 1.25e-06, OutputCostPerToken: 1e-05, CacheReadInputTokenCost: 1.25e-07}},

	{Model: "gpt-5.1", Pricing: ModelPricing{InputCostPerToken: 1.25e-06, OutputCostPerToken: 1e-05, CacheReadInputTokenCost: 1.25e-07}},
	{Model: "gpt-5.1-codex", Pricing: ModelPricing{InputCostPerToken: 1.25e-06, OutputCostPerToken: 1e-05, CacheReadInputTokenCost: 1.25e-07}},
	{Model: "gpt-5.1-codex-max", Pricing: ModelPricing{InputCostPerToken: 1.25e-06, OutputCostPerToken: 1e-05, CacheReadInputTokenCost: 1.25e-07}},
	{Model: "gpt-5.1-codex-mini", Pricing: ModelPricing{InputCostPerToken: 2.5e-07, OutputCostPerToken: 2e-06, CacheReadInputTokenCost: 2.5e-08}},

	{Model: "gpt-5.2", Pricing: ModelPricing{InputCostPerToken: 1.75e-06, OutputCostPerToken: 1.4e-05, CacheReadInputTokenCost: 1.75e-07}},
	{Model: "gpt-5.2-codex", Pricing: ModelPricing{InputCostPerToken: 1.75e-06, OutputCostPerToken: 1.4e-05, CacheReadInputTokenCost: 1.75e-07}},

	{Model: "gpt-5.3", Pricing: ModelPricing{InputCostPerToken: 1.75e-06, OutputCostPerToken: 1.4e-05, CacheReadInputTokenCost: 1.75e-07}},
	{Model: "gpt-5.3-codex", Pricing: ModelPricing{InputCostPerToken: 1.75e-06, OutputCostPerToken: 1.4e-05, CacheReadInputTokenCost: 1.75e-07}},

	{Model: "gpt-5.4", Pricing: ModelPricing{InputCostPerToken: 2.5e-06, OutputCostPerToken: 1.5e-05, CacheReadInputTokenCost: 2.5e-07}},
	{Model: "gpt-5.4-codex", Pricing: ModelPricing{InputCostPerToken: 2.5e-06, OutputCostPerToken: 1.5e-05, CacheReadInputTokenCost: 2.5e-07}},
}
