package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEmbeddingJob(t *testing.T) {
	now := time.Now()
	job := NewEmbeddingJob("job1", "k1", EmbeddingJobStatusPending, 0, "", now, nil)

	assert.Equal(t, "job1", job.ID)
	assert.Equal(t, "k1", job.KnowledgeID)
	assert.Equal(t, EmbeddingJobStatusPending, job.Status)
	assert.Equal(t, int32(0), job.Retries)
	assert.Equal(t, "", job.Error)
	assert.Equal(t, now, job.CreatedAt)
	assert.Nil(t, job.ProcessedAt)
}

func TestNewEmbeddingJobWithProcessedAt(t *testing.T) {
	now := time.Now()
	processedAt := now.Add(1 * time.Hour)
	job := NewEmbeddingJob("job1", "k1", EmbeddingJobStatusCompleted, 0, "", now, &processedAt)

	assert.Equal(t, "job1", job.ID)
	assert.NotNil(t, job.ProcessedAt)
	assert.Equal(t, processedAt, *job.ProcessedAt)
}

func TestEmbeddingJobStatusConstants(t *testing.T) {
	tests := []struct {
		name     string
		status   EmbeddingJobStatus
		expected string
	}{
		{"Pending", EmbeddingJobStatusPending, "pending"},
		{"Processing", EmbeddingJobStatusProcessing, "processing"},
		{"Completed", EmbeddingJobStatusCompleted, "completed"},
		{"Failed", EmbeddingJobStatusFailed, "failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.status))
		})
	}
}

func TestValidateEmbeddingJob(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name    string
		job     *EmbeddingJob
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid job",
			job: &EmbeddingJob{
				ID:          "job1",
				KnowledgeID: "k1",
				Status:      EmbeddingJobStatusPending,
				Retries:     0,
				Error:       "",
				CreatedAt:   now,
				ProcessedAt: nil,
			},
			wantErr: false,
		},
		{
			name: "missing ID",
			job: &EmbeddingJob{
				KnowledgeID: "k1",
				Status:      EmbeddingJobStatusPending,
				Retries:     0,
				Error:       "",
				CreatedAt:   now,
				ProcessedAt: nil,
			},
			wantErr: true,
			errMsg:  "ID",
		},
		{
			name: "missing KnowledgeID",
			job: &EmbeddingJob{
				ID:          "job1",
				Status:      EmbeddingJobStatusPending,
				Retries:     0,
				Error:       "",
				CreatedAt:   now,
				ProcessedAt: nil,
			},
			wantErr: true,
			errMsg:  "KnowledgeID",
		},
		{
			name: "invalid Status",
			job: &EmbeddingJob{
				ID:          "job1",
				KnowledgeID: "k1",
				Status:      EmbeddingJobStatus("invalid"),
				Retries:     0,
				Error:       "",
				CreatedAt:   now,
				ProcessedAt: nil,
			},
			wantErr: true,
			errMsg:  "Status",
		},
		{
			name: "negative Retries",
			job: &EmbeddingJob{
				ID:          "job1",
				KnowledgeID: "k1",
				Status:      EmbeddingJobStatusPending,
				Retries:     -1,
				Error:       "",
				CreatedAt:   now,
				ProcessedAt: nil,
			},
			wantErr: true,
			errMsg:  "Retries",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEmbeddingJob(tt.job)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
