package ingest

type BatchResponse struct {
	Status   string         `json:"status"`
	Ingested int            `json:"ingested"`
	Skipped  int            `json:"skipped"`
	Failed   int            `json:"failed"`
	Items    []ItemResponse `json:"items"`
}

type ItemResponse struct {
	Status       string `json:"status"`
	SessionID    string `json:"session_id"`
	RawFilePath  string `json:"raw_file_path,omitempty"`
	RawSizeBytes int64  `json:"raw_size_bytes,omitempty"`
	RawSHA256    string `json:"raw_sha256"`
	Filename     string `json:"filename,omitempty"`
	Error        string `json:"error,omitempty"`
}
