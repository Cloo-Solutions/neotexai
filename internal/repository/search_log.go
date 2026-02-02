package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/cloo-solutions/neotexai/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SearchLogRepository stores search logs for evaluation/feedback loops.
type SearchLogRepository struct {
	pool *pgxpool.Pool
}

func NewSearchLogRepository(pool *pgxpool.Pool) *SearchLogRepository {
	return &SearchLogRepository{pool: pool}
}

func (r *SearchLogRepository) CreateSearchLog(ctx context.Context, entry service.SearchLogEntry) (string, error) {
	filters := map[string]any{}
	filters["query_length"] = len(entry.Query)
	if entry.Filters.ProjectID != "" {
		filters["project_id"] = entry.Filters.ProjectID
	}
	if entry.Filters.Type != "" {
		filters["type"] = entry.Filters.Type
	}
	if entry.Filters.Status != "" {
		filters["status"] = entry.Filters.Status
	}
	if entry.Filters.PathPrefix != "" {
		filters["path_prefix"] = entry.Filters.PathPrefix
	}
	if entry.Filters.SourceType != "" {
		filters["source_type"] = entry.Filters.SourceType
	}

	filtersJSON, _ := json.Marshal(filters)
	resultsJSON, _ := json.Marshal(entry.Results)

	var id string
	err := r.pool.QueryRow(ctx,
		`INSERT INTO search_logs (org_id, project_id, query, filters, mode, exact, results, result_count, duration_ms)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 RETURNING id`,
		entry.OrgID,
		nullableString(entry.ProjectID),
		entry.Query,
		filtersJSON,
		string(entry.Mode),
		entry.Exact,
		resultsJSON,
		len(entry.Results),
		entry.DurationMs,
	).Scan(&id)
	if err != nil {
		return "", err
	}
	return id, nil
}

func (r *SearchLogRepository) RecordSearchSelection(ctx context.Context, orgID, searchID, selectedID, sourceType string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE search_logs
		 SET chosen_id = $1, chosen_source = $2, chosen_at = $3
		 WHERE id = $4 AND org_id = $5`,
		selectedID,
		sourceType,
		time.Now().UTC(),
		searchID,
		orgID,
	)
	return err
}
