package jobs

import (
	"context"
	"fmt"
	"log"

	"github.com/cloo-solutions/neotexai/internal/domain"
)

const (
	// MaxRetries is the maximum number of retries for a failed job
	MaxRetries = 3
)

// EmbeddingJobRepository defines the interface for embedding job persistence
type EmbeddingJobRepository interface {
	// GetPendingJobs retrieves and claims pending embedding jobs
	GetPendingJobs(ctx context.Context) ([]*domain.EmbeddingJob, error)

	// UpdateJobStatus updates the status of an embedding job
	UpdateJobStatus(ctx context.Context, jobID string, status domain.EmbeddingJobStatus, errMsg string) error

	// IncrementRetries increments the retry count for a job
	IncrementRetries(ctx context.Context, jobID string) error

}

// EmbeddingService defines the interface for generating embeddings
type EmbeddingService interface {
	GenerateEmbedding(ctx context.Context, knowledgeID string) error
	GenerateAssetEmbedding(ctx context.Context, assetID string) error
}

// EmbeddingWorker processes embedding jobs
type EmbeddingWorker struct {
	repo    EmbeddingJobRepository
	service EmbeddingService
}

// NewEmbeddingWorker creates a new EmbeddingWorker instance
func NewEmbeddingWorker(repo EmbeddingJobRepository, service EmbeddingService) *EmbeddingWorker {
	return &EmbeddingWorker{
		repo:    repo,
		service: service,
	}
}

// ProcessJobs implements the JobProcessor interface
func (w *EmbeddingWorker) ProcessJobs(ctx context.Context) error {
	// Fetch pending jobs
	jobs, err := w.repo.GetPendingJobs(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch pending jobs: %w", err)
	}

	if len(jobs) == 0 {
		return nil
	}

	log.Printf("Processing %d pending embedding jobs", len(jobs))

	// Process each job
	for _, job := range jobs {
		if err := w.processJob(ctx, job); err != nil {
			log.Printf("Error processing job %s: %v", job.ID, err)
		}
	}

	return nil
}

func (w *EmbeddingWorker) processJob(ctx context.Context, job *domain.EmbeddingJob) error {
	var err error
	if job.KnowledgeID != "" {
		log.Printf("Processing job %s for knowledge %s", job.ID, job.KnowledgeID)
		err = w.service.GenerateEmbedding(ctx, job.KnowledgeID)
	} else if job.AssetID != "" {
		log.Printf("Processing job %s for asset %s", job.ID, job.AssetID)
		err = w.service.GenerateAssetEmbedding(ctx, job.AssetID)
	} else {
		return fmt.Errorf("job %s has neither knowledge_id nor asset_id", job.ID)
	}

	if err != nil {
		return w.handleJobFailure(ctx, job, err)
	}

	if err := w.repo.UpdateJobStatus(ctx, job.ID, domain.EmbeddingJobStatusCompleted, ""); err != nil {
		return fmt.Errorf("failed to update job status to completed: %w", err)
	}

	log.Printf("Job %s completed successfully", job.ID)
	return nil
}

// handleJobFailure handles a failed job with retry logic
func (w *EmbeddingWorker) handleJobFailure(ctx context.Context, job *domain.EmbeddingJob, jobErr error) error {
	log.Printf("Job %s failed: %v", job.ID, jobErr)

	// Increment retry count
	if err := w.repo.IncrementRetries(ctx, job.ID); err != nil {
		return fmt.Errorf("failed to increment retries: %w", err)
	}

	// Check if max retries exceeded
	if job.Retries+1 >= MaxRetries {
		log.Printf("Job %s exceeded max retries (%d), marking as failed", job.ID, MaxRetries)
		errMsg := fmt.Sprintf("max retries exceeded: %v", jobErr)
		if err := w.repo.UpdateJobStatus(ctx, job.ID, domain.EmbeddingJobStatusFailed, errMsg); err != nil {
			return fmt.Errorf("failed to update job status to failed: %w", err)
		}
		return nil
	}

	// Reset to pending for retry
	log.Printf("Job %s will be retried (attempt %d/%d)", job.ID, job.Retries+1, MaxRetries)
	errMsg := fmt.Sprintf("retry %d: %v", job.Retries+1, jobErr)
	if err := w.repo.UpdateJobStatus(ctx, job.ID, domain.EmbeddingJobStatusPending, errMsg); err != nil {
		return fmt.Errorf("failed to reset job status to pending: %w", err)
	}

	return nil
}
