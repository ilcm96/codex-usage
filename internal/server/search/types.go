package search

import "time"

type Params struct {
	Query        string
	Limit        int
	Offset       int
	IncludeTotal bool
	From         string
	To           string
	DeviceID     string
	RepositoryID string
	ProjectID    string
	Kind         string
	Model        string
}

type Result struct {
	Kind           string     `json:"kind"`
	DocumentScope  string     `json:"documentScope"`
	SessionID      string     `json:"sessionId"`
	Seq            int        `json:"seq"`
	TurnIndex      *int       `json:"turnIndex"`
	OccurredAt     *time.Time `json:"occurredAt"`
	Role           string     `json:"role"`
	ToolName       string     `json:"toolName"`
	Title          string     `json:"title"`
	Text           string     `json:"text"`
	Snippet        string     `json:"snippet"`
	RankWeight     int        `json:"rankWeight"`
	DefaultSearch  bool       `json:"defaultSearchable"`
	MatchStart     int        `json:"matchStart"`
	MatchEnd       int        `json:"matchEnd"`
	CWD            string     `json:"cwd"`
	Branch         string     `json:"branch"`
	Repository     string     `json:"repository"`
	RepoURL        string     `json:"repositoryUrl"`
	Project        string     `json:"project"`
	SessionTitle   string     `json:"sessionTitle"`
	SessionSummary string     `json:"sessionSummary"`
}

type SearchResult struct {
	Items      []Result `json:"items"`
	Limit      int      `json:"limit"`
	NextOffset int      `json:"nextOffset"`
	Offset     int      `json:"offset"`
	Total      int64    `json:"total"`
	TotalKnown bool     `json:"totalKnown"`
}
