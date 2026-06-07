package projects

type RepositorySummary struct {
	ID            string  `json:"id"`
	RepositoryURL string  `json:"repositoryUrl"`
	Owner         string  `json:"owner"`
	Name          string  `json:"name"`
	Sessions      int64   `json:"sessions"`
	TotalTokens   int64   `json:"totalTokens"`
	CostUSD       float64 `json:"costUsd"`
}

type ProjectSummary struct {
	ID            string  `json:"id"`
	DisplayName   string  `json:"displayName"`
	CWD           string  `json:"cwd"`
	RelativePath  string  `json:"relativePath"`
	Repository    string  `json:"repository"`
	RepositoryURL string  `json:"repositoryUrl"`
	Sessions      int64   `json:"sessions"`
	TotalTokens   int64   `json:"totalTokens"`
	CostUSD       float64 `json:"costUsd"`
}
