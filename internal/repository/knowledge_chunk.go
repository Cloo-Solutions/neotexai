package repository

import (
	"context"
	"time"

	"github.com/cloo-solutions/neotexai/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
)

// KnowledgeChunkRepository handles persistence of chunked knowledge embeddings.
type KnowledgeChunkRepository struct {
	db dbtx
}

func NewKnowledgeChunkRepository(pool *pgxpool.Pool) *KnowledgeChunkRepository {
	return &KnowledgeChunkRepository{db: pool}
}

func NewKnowledgeChunkRepositoryWithTx(tx dbtx) *KnowledgeChunkRepository {
	return &KnowledgeChunkRepository{db: tx}
}

// GetByID returns a single chunk by its ID.
func (r *KnowledgeChunkRepository) GetByID(ctx context.Context, chunkID string) (*domain.KnowledgeChunk, error) {
	var chunk domain.KnowledgeChunk
	var projectID, scope *string
	err := r.db.QueryRow(ctx,
		`SELECT id, knowledge_id, org_id, project_id, type, status, title, summary, scope_path, chunk_index, content, created_at, updated_at
		 FROM knowledge_chunks WHERE id = $1`,
		chunkID,
	).Scan(&chunk.ID, &chunk.KnowledgeID, &chunk.OrgID, &projectID, &chunk.Type, &chunk.Status, &chunk.Title, &chunk.Summary, &scope, &chunk.ChunkIndex, &chunk.Content, &chunk.CreatedAt, &chunk.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if projectID != nil {
		chunk.ProjectID = *projectID
	}
	if scope != nil {
		chunk.Scope = *scope
	}
	return &chunk, nil
}

// GetByKnowledgeID returns all chunks for a knowledge item, ordered by chunk_index.
func (r *KnowledgeChunkRepository) GetByKnowledgeID(ctx context.Context, knowledgeID string) ([]*domain.KnowledgeChunk, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, knowledge_id, org_id, project_id, type, status, title, summary, scope_path, chunk_index, content, created_at, updated_at
		 FROM knowledge_chunks WHERE knowledge_id = $1 ORDER BY chunk_index`,
		knowledgeID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chunks []*domain.KnowledgeChunk
	for rows.Next() {
		var chunk domain.KnowledgeChunk
		var projectID, scope *string
		if err := rows.Scan(&chunk.ID, &chunk.KnowledgeID, &chunk.OrgID, &projectID, &chunk.Type, &chunk.Status, &chunk.Title, &chunk.Summary, &scope, &chunk.ChunkIndex, &chunk.Content, &chunk.CreatedAt, &chunk.UpdatedAt); err != nil {
			return nil, err
		}
		if projectID != nil {
			chunk.ProjectID = *projectID
		}
		if scope != nil {
			chunk.Scope = *scope
		}
		chunks = append(chunks, &chunk)
	}
	return chunks, rows.Err()
}

// CountByKnowledgeID returns the number of chunks for a knowledge item.
func (r *KnowledgeChunkRepository) CountByKnowledgeID(ctx context.Context, knowledgeID string) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM knowledge_chunks WHERE knowledge_id = $1`, knowledgeID).Scan(&count)
	return count, err
}

// ReplaceChunks deletes existing chunks for a knowledge item and inserts new ones.
func (r *KnowledgeChunkRepository) ReplaceChunks(ctx context.Context, knowledgeID string, chunks []domain.KnowledgeChunk) error {
	_, err := r.db.Exec(ctx, `DELETE FROM knowledge_chunks WHERE knowledge_id = $1`, knowledgeID)
	if err != nil {
		return err
	}

	if len(chunks) == 0 {
		return nil
	}

	for _, c := range chunks {
		createdAt := c.CreatedAt
		if createdAt.IsZero() {
			createdAt = time.Now().UTC()
		}
		updatedAt := c.UpdatedAt
		if updatedAt.IsZero() {
			updatedAt = createdAt
		}
		_, err := r.db.Exec(ctx,
			`INSERT INTO knowledge_chunks
				(knowledge_id, org_id, project_id, type, status, title, summary, scope_path, chunk_index, content, embedding, created_at, updated_at)
			 VALUES
				($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
			c.KnowledgeID,
			c.OrgID,
			nullableString(c.ProjectID),
			c.Type,
			c.Status,
			c.Title,
			c.Summary,
			nullableString(c.Scope),
			c.ChunkIndex,
			c.Content,
			pgvector.NewVector(c.Embedding),
			createdAt,
			updatedAt,
		)
		if err != nil {
			return err
		}
	}

	return nil
}
