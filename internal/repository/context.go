package repository

import (
	"context"
	"fmt"
	"strings"

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

func (r *ContextRepository) SearchKnowledgeChunksSemantic(ctx context.Context, embedding []float32, filters service.SearchFilters, limit int) ([]*service.ChunkSearchResult, error) {
	if limit <= 0 {
		limit = 20
	}

	vec := pgvector.NewVector(embedding)
	args := []interface{}{vec}
	argIdx := 2

	where := []string{"embedding IS NOT NULL"}
	where = append(where, buildKnowledgeFilters(filters, &args, &argIdx, "")...)

	query := fmt.Sprintf(`
		SELECT id, knowledge_id, chunk_index, title, summary, scope_path, content, updated_at,
		       1.0 / (1.0 + (embedding <=> $1)) AS score
		FROM knowledge_chunks
		WHERE %s
		ORDER BY embedding <=> $1
		LIMIT $%d`, strings.Join(where, " AND "), argIdx)

	args = append(args, limit)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]*service.ChunkSearchResult, 0)
	for rows.Next() {
		var result service.ChunkSearchResult
		var scope *string
		if err := rows.Scan(&result.ChunkID, &result.KnowledgeID, &result.ChunkIndex, &result.Title, &result.Summary, &scope, &result.Content, &result.UpdatedAt, &result.Score); err != nil {
			return nil, err
		}
		if scope != nil {
			result.Scope = *scope
		}
		results = append(results, &result)
	}

	return results, rows.Err()
}

func (r *ContextRepository) SearchKnowledgeChunksLexical(ctx context.Context, queryText string, filters service.SearchFilters, limit int) ([]*service.ChunkSearchResult, error) {
	if limit <= 0 {
		limit = 20
	}

	args := []interface{}{queryText}
	argIdx := 2

	where := []string{"search_tsv @@ websearch_to_tsquery('english', $1)"}
	where = append(where, buildKnowledgeFilters(filters, &args, &argIdx, "")...)

	query := fmt.Sprintf(`
		SELECT id, knowledge_id, chunk_index, title, summary, scope_path, content, updated_at,
		       ts_rank_cd(search_tsv, websearch_to_tsquery('english', $1)) AS score
		FROM knowledge_chunks
		WHERE %s
		ORDER BY score DESC
		LIMIT $%d`, strings.Join(where, " AND "), argIdx)

	args = append(args, limit)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]*service.ChunkSearchResult, 0)
	for rows.Next() {
		var result service.ChunkSearchResult
		var scope *string
		if err := rows.Scan(&result.ChunkID, &result.KnowledgeID, &result.ChunkIndex, &result.Title, &result.Summary, &scope, &result.Content, &result.UpdatedAt, &result.Score); err != nil {
			return nil, err
		}
		if scope != nil {
			result.Scope = *scope
		}
		results = append(results, &result)
	}

	return results, rows.Err()
}

func (r *ContextRepository) SearchKnowledgeSemantic(ctx context.Context, embedding []float32, filters service.SearchFilters, limit int) ([]*service.SearchResult, error) {
	if limit <= 0 {
		limit = 20
	}

	vec := pgvector.NewVector(embedding)
	args := []interface{}{vec}
	argIdx := 2

	where := []string{"embedding IS NOT NULL"}
	where = append(where, buildKnowledgeFilters(filters, &args, &argIdx, "")...)

	query := fmt.Sprintf(`
		SELECT id, title, summary, scope_path, updated_at,
		       1.0 / (1.0 + (embedding <=> $1)) AS score
		FROM knowledge
		WHERE %s
		ORDER BY embedding <=> $1
		LIMIT $%d`, strings.Join(where, " AND "), argIdx)

	args = append(args, limit)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]*service.SearchResult, 0)
	for rows.Next() {
		var result service.SearchResult
		var scope *string
		if err := rows.Scan(&result.ID, &result.Title, &result.Summary, &scope, &result.UpdatedAt, &result.Score); err != nil {
			return nil, err
		}
		if scope != nil {
			result.Scope = *scope
		}
		result.SourceType = "knowledge"
		results = append(results, &result)
	}

	return results, rows.Err()
}

func (r *ContextRepository) SearchKnowledgeLexical(ctx context.Context, queryText string, filters service.SearchFilters, limit int) ([]*service.SearchResult, error) {
	if limit <= 0 {
		limit = 20
	}

	args := []interface{}{queryText}
	argIdx := 2

	where := []string{"search_tsv @@ websearch_to_tsquery('english', $1)"}
	where = append(where, buildKnowledgeFilters(filters, &args, &argIdx, "")...)

	query := fmt.Sprintf(`
		SELECT id, title, summary, scope_path, updated_at,
		       ts_rank_cd(search_tsv, websearch_to_tsquery('english', $1)) AS score
		FROM knowledge
		WHERE %s
		ORDER BY score DESC
		LIMIT $%d`, strings.Join(where, " AND "), argIdx)

	args = append(args, limit)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]*service.SearchResult, 0)
	for rows.Next() {
		var result service.SearchResult
		var scope *string
		if err := rows.Scan(&result.ID, &result.Title, &result.Summary, &scope, &result.UpdatedAt, &result.Score); err != nil {
			return nil, err
		}
		if scope != nil {
			result.Scope = *scope
		}
		result.SourceType = "knowledge"
		results = append(results, &result)
	}

	return results, rows.Err()
}

func (r *ContextRepository) SearchAssetsSemantic(ctx context.Context, embedding []float32, filters service.SearchFilters, limit int) ([]*service.SearchResult, error) {
	if limit <= 0 {
		limit = 20
	}

	vec := pgvector.NewVector(embedding)
	args := []interface{}{vec}
	argIdx := 2

	where := []string{"embedding IS NOT NULL"}
	where = append(where, buildAssetFilters(filters, &args, &argIdx, "")...)

	query := fmt.Sprintf(`
		SELECT id, filename as title, description as summary, created_at,
		       1.0 / (1.0 + (embedding <=> $1)) AS score
		FROM assets
		WHERE %s
		ORDER BY embedding <=> $1
		LIMIT $%d`, strings.Join(where, " AND "), argIdx)

	args = append(args, limit)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]*service.SearchResult, 0)
	for rows.Next() {
		var result service.SearchResult
		if err := rows.Scan(&result.ID, &result.Title, &result.Summary, &result.UpdatedAt, &result.Score); err != nil {
			return nil, err
		}
		result.SourceType = "asset"
		results = append(results, &result)
	}

	return results, rows.Err()
}

func (r *ContextRepository) SearchAssetsLexical(ctx context.Context, queryText string, filters service.SearchFilters, limit int) ([]*service.SearchResult, error) {
	if limit <= 0 {
		limit = 20
	}

	args := []interface{}{queryText}
	argIdx := 2

	where := []string{"search_tsv @@ websearch_to_tsquery('english', $1)"}
	where = append(where, buildAssetFilters(filters, &args, &argIdx, "")...)

	query := fmt.Sprintf(`
		SELECT id, filename as title, description as summary, created_at,
		       ts_rank_cd(search_tsv, websearch_to_tsquery('english', $1)) AS score
		FROM assets
		WHERE %s
		ORDER BY score DESC
		LIMIT $%d`, strings.Join(where, " AND "), argIdx)

	args = append(args, limit)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]*service.SearchResult, 0)
	for rows.Next() {
		var result service.SearchResult
		if err := rows.Scan(&result.ID, &result.Title, &result.Summary, &result.UpdatedAt, &result.Score); err != nil {
			return nil, err
		}
		result.SourceType = "asset"
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

func (r *ContextRepository) GetAssetsByIDs(ctx context.Context, ids []string) ([]*domain.Asset, error) {
	if len(ids) == 0 {
		return []*domain.Asset{}, nil
	}

	rows, err := r.pool.Query(ctx,
		`SELECT id, org_id, project_id, filename, mime_type, sha256, storage_key, keywords, description, created_at
		 FROM assets WHERE id = ANY($1)`,
		ids,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assets []*domain.Asset
	for rows.Next() {
		var a domain.Asset
		var projectID *string
		if err := rows.Scan(&a.ID, &a.OrgID, &projectID, &a.Filename, &a.MimeType, &a.SHA256, &a.StorageKey, &a.Keywords, &a.Description, &a.CreatedAt); err != nil {
			return nil, err
		}
		if projectID != nil {
			a.ProjectID = *projectID
		}
		assets = append(assets, &a)
	}

	return assets, rows.Err()
}

func buildKnowledgeFilters(filters service.SearchFilters, args *[]interface{}, argIdx *int, tableAlias string) []string {
	where := []string{}
	column := func(name string) string {
		if tableAlias == "" {
			return name
		}
		return tableAlias + "." + name
	}

	where = append(where, fmt.Sprintf("%s = $%d", column("org_id"), *argIdx))
	*args = append(*args, filters.OrgID)
	*argIdx++

	if filters.ProjectID != "" {
		where = append(where, fmt.Sprintf("%s = $%d", column("project_id"), *argIdx))
		*args = append(*args, filters.ProjectID)
		*argIdx++
	}
	if filters.Type != "" {
		where = append(where, fmt.Sprintf("%s = $%d", column("type"), *argIdx))
		*args = append(*args, filters.Type)
		*argIdx++
	}
	if filters.Status != "" {
		where = append(where, fmt.Sprintf("%s = $%d", column("status"), *argIdx))
		*args = append(*args, filters.Status)
		*argIdx++
	}
	if filters.PathPrefix != "" {
		where = append(where, fmt.Sprintf("%s LIKE $%d", column("scope_path"), *argIdx))
		*args = append(*args, filters.PathPrefix+"%")
		*argIdx++
	}
	return where
}

func buildAssetFilters(filters service.SearchFilters, args *[]interface{}, argIdx *int, tableAlias string) []string {
	where := []string{}
	column := func(name string) string {
		if tableAlias == "" {
			return name
		}
		return tableAlias + "." + name
	}

	where = append(where, fmt.Sprintf("%s = $%d", column("org_id"), *argIdx))
	*args = append(*args, filters.OrgID)
	*argIdx++

	if filters.ProjectID != "" {
		where = append(where, fmt.Sprintf("%s = $%d", column("project_id"), *argIdx))
		*args = append(*args, filters.ProjectID)
		*argIdx++
	}
	return where
}

// ListKnowledge returns metadata-only knowledge items for the List operation
func (r *ContextRepository) ListKnowledge(ctx context.Context, input service.ListInput) ([]*service.ListItem, error) {
	args := []interface{}{input.OrgID}
	argIdx := 2

	where := []string{"k.org_id = $1"}

	if input.ProjectID != "" {
		where = append(where, fmt.Sprintf("k.project_id = $%d", argIdx))
		args = append(args, input.ProjectID)
		argIdx++
	}
	if input.Type != "" {
		where = append(where, fmt.Sprintf("k.type = $%d", argIdx))
		args = append(args, input.Type)
		argIdx++
	}
	if input.Status != "" {
		where = append(where, fmt.Sprintf("k.status = $%d", argIdx))
		args = append(args, input.Status)
		argIdx++
	}
	if input.PathPrefix != "" {
		where = append(where, fmt.Sprintf("k.scope_path LIKE $%d", argIdx))
		args = append(args, input.PathPrefix+"%")
		argIdx++
	}
	if input.UpdatedSince != nil {
		where = append(where, fmt.Sprintf("k.updated_at >= $%d", argIdx))
		args = append(args, *input.UpdatedSince)
		argIdx++
	}

	limit := input.Limit
	if limit <= 0 {
		limit = 50
	}

	query := fmt.Sprintf(`
		SELECT k.id, k.title, k.scope_path, k.type, k.status, k.updated_at,
		       COALESCE((SELECT COUNT(*) FROM knowledge_chunks WHERE knowledge_id = k.id), 0) AS chunk_count
		FROM knowledge k
		WHERE %s
		ORDER BY k.updated_at DESC
		LIMIT $%d`, strings.Join(where, " AND "), argIdx)

	args = append(args, limit+1) // fetch one extra to detect hasMore

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*service.ListItem
	for rows.Next() {
		var item service.ListItem
		var scope *string
		if err := rows.Scan(&item.ID, &item.Title, &scope, &item.Type, &item.Status, &item.UpdatedAt, &item.ChunkCount); err != nil {
			return nil, err
		}
		if scope != nil {
			item.Scope = *scope
		}
		item.SourceType = "knowledge"
		items = append(items, &item)
	}

	return items, rows.Err()
}

// ListAssets returns metadata-only asset items for the List operation
func (r *ContextRepository) ListAssets(ctx context.Context, input service.ListInput) ([]*service.ListItem, error) {
	args := []interface{}{input.OrgID}
	argIdx := 2

	where := []string{"org_id = $1"}

	if input.ProjectID != "" {
		where = append(where, fmt.Sprintf("project_id = $%d", argIdx))
		args = append(args, input.ProjectID)
		argIdx++
	}
	if input.UpdatedSince != nil {
		where = append(where, fmt.Sprintf("created_at >= $%d", argIdx))
		args = append(args, *input.UpdatedSince)
		argIdx++
	}

	limit := input.Limit
	if limit <= 0 {
		limit = 50
	}

	query := fmt.Sprintf(`
		SELECT id, filename, mime_type, created_at
		FROM assets
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d`, strings.Join(where, " AND "), argIdx)

	args = append(args, limit+1) // fetch one extra to detect hasMore

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*service.ListItem
	for rows.Next() {
		var item service.ListItem
		if err := rows.Scan(&item.ID, &item.Filename, &item.MimeType, &item.UpdatedAt); err != nil {
			return nil, err
		}
		item.Title = item.Filename
		item.SourceType = "asset"
		item.ChunkCount = 0
		items = append(items, &item)
	}

	return items, rows.Err()
}
