package service

import "context"

// SearchLogResult captures a single result entry for logging.
type SearchLogResult struct {
	ID         string  `json:"id"`
	SourceType string  `json:"source_type"`
	Score      float32 `json:"score"`
}

// SearchLogEntry captures a search request and its results.
type SearchLogEntry struct {
	OrgID      string
	ProjectID  string
	Query      string
	Filters    SearchFilters
	Mode       SearchMode
	Exact      bool
	Limit      int
	DurationMs int
	Results    []SearchLogResult
}

// SearchLogRepository persists search logs and feedback.
type SearchLogRepository interface {
	CreateSearchLog(ctx context.Context, entry SearchLogEntry) (string, error)
	RecordSearchSelection(ctx context.Context, orgID, searchID, selectedID, sourceType string) error
}
