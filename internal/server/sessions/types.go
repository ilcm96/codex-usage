package sessions

import "time"

type ListParams struct {
	Limit               int
	Offset              int
	Sort                string
	From                string
	To                  string
	DeviceID            string
	RepositoryID        string
	ProjectID           string
	Branch              string
	Model               string
	Query               string
	OnlyWithInputTokens bool
}

type ReaderParams struct {
	Limit  int
	Offset int
	Query  string
}

type TimelineParams struct {
	Limit  int
	Offset int
	Kind   string
	Query  string
}

type UsageTotals struct {
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

type SimpleSession struct {
	ID            string     `json:"id"`
	StartedAt     *time.Time `json:"startedAt"`
	UpdatedAt     *time.Time `json:"updatedAt"`
	CWD           string     `json:"cwd"`
	Branch        string     `json:"branch"`
	Repository    string     `json:"repository"`
	RepositoryURL string     `json:"repositoryUrl"`
	Project       string     `json:"project"`
	Device        string     `json:"device"`
	InputTokens   int64      `json:"inputTokens"`
	CachedTokens  int64      `json:"cachedInputTokens"`
	OutputTokens  int64      `json:"outputTokens"`
	Reasoning     int64      `json:"reasoningOutputTokens"`
	TotalTokens   int64      `json:"totalTokens"`
	CostUSD       float64    `json:"costUsd"`
	Models        string     `json:"models"`
}

type ListItem struct {
	ID                    string     `json:"id"`
	StartedAt             *time.Time `json:"startedAt"`
	UpdatedAt             *time.Time `json:"updatedAt"`
	CWD                   string     `json:"cwd"`
	Branch                string     `json:"branch"`
	Repository            string     `json:"repository"`
	RepositoryURL         string     `json:"repositoryUrl"`
	Project               string     `json:"project"`
	Device                string     `json:"device"`
	Title                 string     `json:"title"`
	DisplayTitle          string     `json:"displayTitle"`
	DisplaySubtitle       string     `json:"displaySubtitle"`
	UserIntent            string     `json:"userIntent"`
	DominantLanguage      string     `json:"dominantLanguage"`
	FirstUserMessage      string     `json:"firstUserMessage"`
	LastUserMessage       string     `json:"lastUserMessage"`
	ShortSummary          string     `json:"shortSummary"`
	MainModel             string     `json:"mainModel"`
	DurationSeconds       int64      `json:"durationSeconds"`
	CacheHitRate          float64    `json:"cacheHitRate"`
	ConversationTurns     int64      `json:"conversationTurns"`
	SearchableMessages    int64      `json:"searchableMessages"`
	SearchableTools       int64      `json:"searchableTools"`
	InputTokens           int64      `json:"inputTokens"`
	CachedTokens          int64      `json:"cachedInputTokens"`
	OutputTokens          int64      `json:"outputTokens"`
	Reasoning             int64      `json:"reasoningOutputTokens"`
	TotalTokens           int64      `json:"totalTokens"`
	CostUSD               float64    `json:"costUsd"`
	Models                string     `json:"models"`
	MessageCount          int64      `json:"messageCount"`
	UserMessageCount      int64      `json:"userMessageCount"`
	AssistantMessageCount int64      `json:"assistantMessageCount"`
	ToolCallCount         int64      `json:"toolCallCount"`
	PatchAddedLines       int64      `json:"patchAddedLines"`
}

type ListResult struct {
	Items      []ListItem  `json:"items"`
	Limit      int         `json:"limit"`
	NextOffset int         `json:"nextOffset"`
	Offset     int         `json:"offset"`
	Total      int64       `json:"total"`
	Totals     UsageTotals `json:"totals"`
}

type Detail struct {
	ID                    string     `json:"id"`
	StartedAt             *time.Time `json:"startedAt"`
	UpdatedAt             *time.Time `json:"updatedAt"`
	CWD                   string     `json:"cwd"`
	Repository            string     `json:"repository"`
	RepositoryURL         string     `json:"repositoryUrl"`
	Project               string     `json:"project"`
	Device                string     `json:"device"`
	Branch                string     `json:"branch"`
	CommitHash            string     `json:"commitHash"`
	Title                 string     `json:"title"`
	DisplayTitle          string     `json:"displayTitle"`
	DisplaySubtitle       string     `json:"displaySubtitle"`
	UserIntent            string     `json:"userIntent"`
	DominantLanguage      string     `json:"dominantLanguage"`
	FirstUserMessage      string     `json:"firstUserMessage"`
	LastUserMessage       string     `json:"lastUserMessage"`
	ShortSummary          string     `json:"shortSummary"`
	MainModel             string     `json:"mainModel"`
	DurationSeconds       int64      `json:"durationSeconds"`
	CacheHitRate          float64    `json:"cacheHitRate"`
	ConversationTurns     int64      `json:"conversationTurns"`
	SearchableMessages    int64      `json:"searchableMessages"`
	SearchableTools       int64      `json:"searchableTools"`
	InputTokens           int64      `json:"inputTokens"`
	CachedInputTokens     int64      `json:"cachedInputTokens"`
	OutputTokens          int64      `json:"outputTokens"`
	ReasoningOutputTokens int64      `json:"reasoningOutputTokens"`
	TotalTokens           int64      `json:"totalTokens"`
	CostUSD               float64    `json:"costUsd"`
	MessageCount          int64      `json:"messageCount"`
	ToolCallCount         int64      `json:"toolCallCount"`
	PatchAddedLines       int64      `json:"patchAddedLines"`
}

type ModelUsage struct {
	Model                 string  `json:"model"`
	InputTokens           int64   `json:"inputTokens"`
	CachedInputTokens     int64   `json:"cachedInputTokens"`
	OutputTokens          int64   `json:"outputTokens"`
	ReasoningOutputTokens int64   `json:"reasoningOutputTokens"`
	TotalTokens           int64   `json:"totalTokens"`
	CostUSD               float64 `json:"costUsd"`
}

type DetailResult struct {
	Session  Detail       `json:"session"`
	Models   []ModelUsage `json:"models"`
	Timeline []any        `json:"timeline"`
}

type ReaderSummary struct {
	SessionID              string     `json:"sessionId"`
	Title                  string     `json:"title"`
	DisplayTitle           string     `json:"displayTitle"`
	DisplaySubtitle        string     `json:"displaySubtitle"`
	UserIntent             string     `json:"userIntent"`
	DominantLanguage       string     `json:"dominantLanguage"`
	FirstUserMessage       string     `json:"firstUserMessage"`
	LastUserMessage        string     `json:"lastUserMessage"`
	ShortSummary           string     `json:"shortSummary"`
	MessageCount           int64      `json:"messageCount"`
	UserMessageCount       int64      `json:"userMessageCount"`
	AssistantMessageCount  int64      `json:"assistantMessageCount"`
	ToolCallCount          int64      `json:"toolCallCount"`
	MainModel              string     `json:"mainModel"`
	DurationSeconds        int64      `json:"durationSeconds"`
	CacheHitRate           float64    `json:"cacheHitRate"`
	StartedAt              *time.Time `json:"startedAt"`
	UpdatedAt              *time.Time `json:"updatedAt"`
	ConversationTurnCount  int64      `json:"conversationTurnCount"`
	SearchableDocumentRows int64      `json:"searchableDocumentRows"`
}

type ReaderTurn struct {
	TurnIndex       int        `json:"turnIndex"`
	UserSeq         *int       `json:"userSeq"`
	AssistantSeq    *int       `json:"assistantSeq"`
	StartedAt       *time.Time `json:"startedAt"`
	EndedAt         *time.Time `json:"endedAt"`
	UserText        string     `json:"userText"`
	AssistantText   string     `json:"assistantText"`
	ToolCallCount   int64      `json:"toolCallCount"`
	ToolResultCount int64      `json:"toolResultCount"`
	ToolNames       []string   `json:"toolNames"`
}

type ReaderResult struct {
	Summary    ReaderSummary `json:"summary"`
	Items      []ReaderTurn  `json:"items"`
	Limit      int           `json:"limit"`
	NextOffset int           `json:"nextOffset"`
	Offset     int           `json:"offset"`
	Total      int64         `json:"total"`
}

type TimelineItem struct {
	Seq        int        `json:"seq"`
	OccurredAt *time.Time `json:"occurredAt"`
	Kind       string     `json:"kind"`
	Role       string     `json:"role"`
	ToolName   string     `json:"toolName"`
	Status     string     `json:"status"`
	Text       string     `json:"text"`
}

type TimelineResult struct {
	Items      []TimelineItem `json:"items"`
	Limit      int            `json:"limit"`
	NextOffset int            `json:"nextOffset"`
	Offset     int            `json:"offset"`
	Total      int64          `json:"total"`
}
