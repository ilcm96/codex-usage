package codex

type Usage struct {
	InputTokens           int64 `json:"inputTokens"`
	CachedInputTokens     int64 `json:"cachedInputTokens"`
	OutputTokens          int64 `json:"outputTokens"`
	ReasoningOutputTokens int64 `json:"reasoningOutputTokens"`
	TotalTokens           int64 `json:"totalTokens"`
	IsFallbackModel       bool  `json:"isFallbackModel,omitempty"`
}

func (u *Usage) Add(v Usage) {
	u.InputTokens += v.InputTokens
	u.CachedInputTokens += v.CachedInputTokens
	u.OutputTokens += v.OutputTokens
	u.ReasoningOutputTokens += v.ReasoningOutputTokens
	u.TotalTokens += v.TotalTokens
	u.IsFallbackModel = u.IsFallbackModel || v.IsFallbackModel
}
