package usage

import "time"

type Filters struct {
	From         string
	To           string
	DeviceID     string
	RepositoryID string
	ProjectID    string
	Model        string
}

type SeriesParams struct {
	Bucket  string
	Filters Filters
}

type BreakdownParams struct {
	GroupBy string
	Limit   int
	Filters Filters
}

type CalendarParams struct {
	Days    int
	Filters Filters
}

type Totals struct {
	InputTokens           int64   `json:"inputTokens"`
	CachedInputTokens     int64   `json:"cachedInputTokens"`
	OutputTokens          int64   `json:"outputTokens"`
	ReasoningOutputTokens int64   `json:"reasoningOutputTokens"`
	TotalTokens           int64   `json:"totalTokens"`
	CostUSD               float64 `json:"costUsd"`
	Sessions              int64   `json:"sessions"`
	Messages              int64   `json:"messages"`
	ToolCalls             int64   `json:"toolCalls"`
	PatchAddedLines       int64   `json:"patchAddedLines"`
}

type GlobalTotals struct {
	TotalTokens     int64   `json:"totalTokens"`
	InputTokens     int64   `json:"inputTokens"`
	OutputTokens    int64   `json:"outputTokens"`
	CostUSD         float64 `json:"costUsd"`
	Sessions        int64   `json:"sessions"`
	Projects        int64   `json:"projects"`
	Devices         int64   `json:"devices"`
	PatchAddedLines int64   `json:"patchAddedLines"`
}

type Summary struct {
	Current        Totals  `json:"current"`
	ActiveDays     int64   `json:"activeDays"`
	CacheHitRate   float64 `json:"cacheHitRate"`
	AvgSessionCost float64 `json:"avgSessionCost"`
}

type SeriesBucket struct {
	Bucket                string  `json:"bucket"`
	InputTokens           int64   `json:"inputTokens"`
	CachedInputTokens     int64   `json:"cachedInputTokens"`
	OutputTokens          int64   `json:"outputTokens"`
	ReasoningOutputTokens int64   `json:"reasoningOutputTokens"`
	TotalTokens           int64   `json:"totalTokens"`
	CostUSD               float64 `json:"costUsd"`
	PatchAddedLines       int64   `json:"patchAddedLines"`
}

type BreakdownItem struct {
	ID                    string  `json:"id"`
	Label                 string  `json:"label"`
	Detail                string  `json:"detail"`
	Sessions              int64   `json:"sessions"`
	InputTokens           int64   `json:"inputTokens"`
	CachedInputTokens     int64   `json:"cachedInputTokens"`
	OutputTokens          int64   `json:"outputTokens"`
	ReasoningOutputTokens int64   `json:"reasoningOutputTokens"`
	TotalTokens           int64   `json:"totalTokens"`
	CostUSD               float64 `json:"costUsd"`
	PatchAddedLines       int64   `json:"patchAddedLines"`
}

type CalendarDay struct {
	Date        string  `json:"date"`
	TotalTokens int64   `json:"totalTokens"`
	CostUSD     float64 `json:"costUsd"`
	Projects    int64   `json:"projects"`
}

type Window struct {
	Label        string     `json:"label"`
	Days         int        `json:"days"`
	From         *time.Time `json:"from"`
	To           *time.Time `json:"to"`
	Totals       Totals     `json:"totals"`
	CacheHitRate float64    `json:"cacheHitRate"`
}
