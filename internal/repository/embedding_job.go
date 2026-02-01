package repository

import (
	"context"
	"errors"
	"time"

	"github.com/cloo-solutions/neotexai/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrEmbeddingJobNotFound = errors.New("embedding job not found")

type EmbeddingJobRepository struct {
	pool *pgxpool.Pool
}

func NewEmbeddingJobRepository(pool *pgxpool.Pool) *EmbeddingJobRepository {
	return &EmbeddingJobRepository{pool: pool}
}

func (r *EmbeddingJobRepository) Create(ctx context.Context, job *domain.EmbeddingJob) error {
	var knowledgeID, assetID *string
	if job.KnowledgeID != "" {
		knowledgeID = &job.KnowledgeID
	}
	if job.AssetID != "" {
		assetID = &job.AssetID
	}
	_, err := r.pool.Exec(ctx,
		`INSERT INTO embedding_jobs (id, knowledge_id, asset_id, status, retries, error, created_at, processed_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		job.ID, knowledgeID, assetID, job.Status, job.Retries, job.Error, job.CreatedAt, job.ProcessedAt,
	)
	return err
}

func (r *EmbeddingJobRepository) GetByID(ctx context.Context, id string) (*domain.EmbeddingJob, error) {
	var job domain.EmbeddingJob
	var errMsg, knowledgeID, assetID pgtype.Text
	err := r.pool.QueryRow(ctx,
		`SELECT id, knowledge_id, asset_id, status, retries, error, created_at, processed_at
		 FROM embedding_jobs WHERE id = $1`,
		id,
	).Scan(&job.ID, &knowledgeID, &assetID, &job.Status, &job.Retries, &errMsg, &job.CreatedAt, &job.ProcessedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrEmbeddingJobNotFound
		}
		return nil, err
	}
	if knowledgeID.Valid {
		job.KnowledgeID = knowledgeID.String
	}
	if assetID.Valid {
		job.AssetID = assetID.String
	}
	if errMsg.Valid {
		job.Error = errMsg.String
	}
	return &job, nil
}

func (r *EmbeddingJobRepository) GetPending(ctx context.Context, limit int) ([]*domain.EmbeddingJob, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, knowledge_id, asset_id, status, retries, error, created_at, processed_at
		 FROM embedding_jobs
		 WHERE status = $1
		 ORDER BY created_at ASC
		 LIMIT $2`,
		domain.EmbeddingJobStatusPending, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*domain.EmbeddingJob
	for rows.Next() {
		var job domain.EmbeddingJob
		var errMsg, knowledgeID, assetID pgtype.Text
		if err := rows.Scan(&job.ID, &knowledgeID, &assetID, &job.Status, &job.Retries, &errMsg, &job.CreatedAt, &job.ProcessedAt); err != nil {
			return nil, err
		}
		if knowledgeID.Valid {
			job.KnowledgeID = knowledgeID.String
		}
		if assetID.Valid {
			job.AssetID = assetID.String
		}
		if errMsg.Valid {
			job.Error = errMsg.String
		}
		jobs = append(jobs, &job)
	}
	return jobs, rows.Err()
}

func (r *EmbeddingJobRepository) UpdateStatus(ctx context.Context, id string, status domain.EmbeddingJobStatus, errMsg string) error {
	var processedAt *time.Time
	if status == domain.EmbeddingJobStatusCompleted || status == domain.EmbeddingJobStatusFailed {
		now := time.Now().UTC()
		processedAt = &now
	}

	var errPtr *string
	if errMsg != "" {
		errPtr = &errMsg
	}

	cmdTag, err := r.pool.Exec(ctx,
		`UPDATE embedding_jobs SET status = $1, error = $2, processed_at = $3 WHERE id = $4`,
		status, errPtr, processedAt, id,
	)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrEmbeddingJobNotFound
	}
	return nil
}

func (r *EmbeddingJobRepository) IncrementRetries(ctx context.Context, id string) error {
	cmdTag, err := r.pool.Exec(ctx,
		`UPDATE embedding_jobs SET retries = retries + 1 WHERE id = $1`,
		id,
	)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrEmbeddingJobNotFound
	}
	return nil
}

func (r *EmbeddingJobRepository) GetPendingJobs(ctx context.Context) ([]*domain.EmbeddingJob, error) {
	return r.GetPending(ctx, 100)
}

func (r *EmbeddingJobRepository) UpdateJobStatus(ctx context.Context, jobID string, status domain.EmbeddingJobStatus, errMsg string) error {
	return r.UpdateStatus(ctx, jobID, status, errMsg)
}

func (r *EmbeddingJobRepository) MarkProcessed(ctx context.Context, jobID string, processedAt time.Time) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE embedding_jobs SET processed_at = $1 WHERE id = $2`,
		processedAt, jobID,
	)
	return err
}
