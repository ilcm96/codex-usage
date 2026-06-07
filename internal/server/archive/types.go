package archive

import "time"

type Status struct {
	Sessions          int64      `json:"sessions"`
	Devices           int64      `json:"devices"`
	RawBytes          int64      `json:"rawBytes"`
	SessionEvents     int64      `json:"sessionEvents"`
	Messages          int64      `json:"messages"`
	ToolEvents        int64      `json:"toolEvents"`
	UsageEvents       int64      `json:"usageEvents"`
	MissingRawFiles   int64      `json:"missingRawFiles"`
	MissingRawSHA     int64      `json:"missingRawSha"`
	OldestIngestedAt  *time.Time `json:"oldestIngestedAt"`
	NewestIngestedAt  *time.Time `json:"newestIngestedAt"`
	OldestSessionTime *time.Time `json:"oldestSessionTime"`
	NewestSessionTime *time.Time `json:"newestSessionTime"`
}

type Health struct {
	Status              string     `json:"status"`
	Sessions            int64      `json:"sessions"`
	ArchiveRows         int64      `json:"archiveRows"`
	SessionSummaries    int64      `json:"sessionSummaries"`
	ConversationTurns   int64      `json:"conversationTurns"`
	SearchDocuments     int64      `json:"searchDocuments"`
	DefaultSearchDocs   int64      `json:"defaultSearchDocs"`
	Messages            int64      `json:"messages"`
	ToolEvents          int64      `json:"toolEvents"`
	VerifiedArchiveRows int64      `json:"verifiedArchiveRows"`
	MissingRawFiles     int64      `json:"missingRawFiles"`
	MissingArchiveRows  int64      `json:"missingArchiveRows"`
	LatestIngestedAt    *time.Time `json:"latestIngestedAt"`
	OldestIngestedAt    *time.Time `json:"oldestIngestedAt"`
}

type DeviceSummary struct {
	ID             string     `json:"id"`
	Name           string     `json:"name"`
	Hostname       string     `json:"hostname"`
	Sessions       int64      `json:"sessions"`
	RawBytes       int64      `json:"rawBytes"`
	LastIngestedAt *time.Time `json:"lastIngestedAt"`
}

type RepositorySummary struct {
	ID             string     `json:"id"`
	Name           string     `json:"name"`
	URL            string     `json:"url"`
	Sessions       int64      `json:"sessions"`
	RawBytes       int64      `json:"rawBytes"`
	LastIngestedAt *time.Time `json:"lastIngestedAt"`
}

type IntegrityIssue struct {
	SessionID string `json:"sessionId"`
	Path      string `json:"path"`
	Problem   string `json:"problem"`
}

type IntegrityResult struct {
	Checked      int64            `json:"checked"`
	OK           int64            `json:"ok"`
	MissingPath  int64            `json:"missingPath"`
	MissingFile  int64            `json:"missingFile"`
	SizeMismatch int64            `json:"sizeMismatch"`
	MissingSHA   int64            `json:"missingSha"`
	Issues       []IntegrityIssue `json:"issues"`
}
