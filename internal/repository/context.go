package repository

import (
	"context"

	"github.com/cloo-solutions/neotexai/internal/domain"
	"github.com/cloo-solutions/neotexai/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
)

// ContextRepository implements context lookups and vector search.
type ContextRepository struct {
	pool *pgxpool.Pool
}

func NewContextRepository(pool *pgxpool.Pool) *ContextRepository {
	return &ContextRepository{pool: pool}
}

func (r *ContextRepository) GetManifest(ctx context.Context, orgID, projectID string) ([]*service.KnowledgeManifestItem, error) {
	query := `
		SELECT id, title, summary, type, scope_path
		FROM knowledge
		WHERE org_id = $1`
	args := []interface{}{orgID}

	if projectID != "" {
		query += " AND project_id = $2"
		args = append(args, projectID)
	}

	query += " ORDER BY updated_at DESC"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*service.KnowledgeManifestItem
	for rows.Next() {
		var item service.KnowledgeManifestItem
		var scope *string
		var knowledgeType string
		if err := rows.Scan(&item.ID, &item.Title, &item.Summary, &knowledgeType, &scope); err != nil {
			return nil, err
		}
		item.Type = domain.KnowledgeType(knowledgeType)
		if scope != nil {
			item.Scope = *scope
		}
		results = append(results, &item)
	}

	return results, rows.Err()
}

func (r *ContextRepository) SearchByEmbedding(ctx context.Context, embedding []float32, filters service.SearchFilters, limit int) ([]*service.SearchResult, error) {
	if limit <= 0 {
		limit = 20
	}

	vec := pgvector.NewVector(embedding)

	query := `
		WITH combined AS (
			SELECT id, title, summary, scope_path, 'knowledge' as source_type,
			       1.0 / (1.0 + (embedding <=> $1)) AS score
			FROM knowledge
			WHERE org_id = $2 AND embedding IS NOT NULL
			UNION ALL
			SELECT id, filename as title, description as summary, NULL as scope_path, 'asset' as source_type,
			       1.0 / (1.0 + (embedding <=> $1)) AS score
			FROM assets
			WHERE org_id = $2 AND embedding IS NOT NULL
		)
		SELECT id, title, summary, scope_path, source_type, score
		FROM combined
		ORDER BY score DESC
		LIMIT $3`

	rows, err := r.pool.Query(ctx, query, vec, filters.OrgID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]*service.SearchResult, 0)
	for rows.Next() {
		var result service.SearchResult
		var scope, sourceType *string
		if err := rows.Scan(&result.ID, &result.Title, &result.Summary, &scope, &sourceType, &result.Score); err != nil {
			return nil, err
		}
		if scope != nil {
			result.Scope = *scope
		}
		if sourceType != nil && *sourceType == "asset" {
			result.Scope = "asset"
		}
		results = append(results, &result)
	}

	return results, rows.Err()
}

func (r *ContextRepository) GetByIDs(ctx context.Context, ids []string) ([]*domain.Knowledge, error) {
	if len(ids) == 0 {
		return []*domain.Knowledge{}, nil
	}

	rows, err := r.pool.Query(ctx,
		`SELECT id, org_id, project_id, type, status, title, summary, body_md, scope_path, created_at, updated_at
		 FROM knowledge WHERE id = ANY($1)`,
		ids,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanKnowledgeRows(rows)
}
