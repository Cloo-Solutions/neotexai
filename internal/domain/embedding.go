package domain

import (
	"fmt"
	"time"
)

// EmbeddingJobStatus represents the status of an embedding job
type EmbeddingJobStatus string

const (
	EmbeddingJobStatusPending    EmbeddingJobStatus = "pending"
	EmbeddingJobStatusProcessing EmbeddingJobStatus = "processing"
	EmbeddingJobStatusCompleted  EmbeddingJobStatus = "completed"
	EmbeddingJobStatusFailed     EmbeddingJobStatus = "failed"
)

// EmbeddingJob represents an async embedding generation job
type EmbeddingJob struct {
	ID          string
	KnowledgeID string // Set for knowledge embeddings
	AssetID     string // Set for asset embeddings
	Status      EmbeddingJobStatus
	Retries     int32
	Error       string
	CreatedAt   time.Time
	ProcessedAt *time.Time
}

// NewEmbeddingJob creates a new EmbeddingJob instance
func NewEmbeddingJob(
	id, knowledgeID string,
	status EmbeddingJobStatus,
	retries int32,
	errMsg string,
	createdAt time.Time,
	processedAt *time.Time,
) *EmbeddingJob {
	return &EmbeddingJob{
		ID:          id,
		KnowledgeID: knowledgeID,
		Status:      status,
		Retries:     retries,
		Error:       errMsg,
		CreatedAt:   createdAt,
		ProcessedAt: processedAt,
	}
}

// ValidateEmbeddingJob validates an EmbeddingJob instance
func ValidateEmbeddingJob(j *EmbeddingJob) error {
	if j == nil {
		return fmt.Errorf("embedding job cannot be nil")
	}

	if j.ID == "" {
		return fmt.Errorf("embedding job ID is required")
	}

	if j.KnowledgeID == "" && j.AssetID == "" {
		return fmt.Errorf("embedding job must have either KnowledgeID or AssetID")
	}

	if j.KnowledgeID != "" && j.AssetID != "" {
		return fmt.Errorf("embedding job cannot have both KnowledgeID and AssetID")
	}

	if !isValidEmbeddingJobStatus(j.Status) {
		return fmt.Errorf("embedding job Status is invalid: %s", j.Status)
	}

	if j.Retries < 0 {
		return fmt.Errorf("embedding job Retries cannot be negative")
	}

	return nil
}

// isValidEmbeddingJobStatus checks if an EmbeddingJobStatus is valid
func isValidEmbeddingJobStatus(s EmbeddingJobStatus) bool {
	switch s {
	case EmbeddingJobStatusPending, EmbeddingJobStatusProcessing,
		EmbeddingJobStatusCompleted, EmbeddingJobStatusFailed:
		return true
	}
	return false
}
