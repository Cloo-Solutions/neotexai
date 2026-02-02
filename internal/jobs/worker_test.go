package jobs

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/cloo-solutions/neotexai/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockJobProcessor is a mock implementation of JobProcessor
type MockJobProcessor struct {
	mock.Mock
}

func (m *MockJobProcessor) ProcessJobs(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// MockEmbeddingJobRepository is a mock implementation of EmbeddingJobRepository
type MockEmbeddingJobRepository struct {
	mock.Mock
}

func (m *MockEmbeddingJobRepository) GetPendingJobs(ctx context.Context) ([]*domain.EmbeddingJob, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.EmbeddingJob), args.Error(1)
}

func (m *MockEmbeddingJobRepository) UpdateJobStatus(ctx context.Context, jobID string, status domain.EmbeddingJobStatus, errMsg string) error {
	args := m.Called(ctx, jobID, status, errMsg)
	return args.Error(0)
}

func (m *MockEmbeddingJobRepository) IncrementRetries(ctx context.Context, jobID string) error {
	args := m.Called(ctx, jobID)
	return args.Error(0)
}

// MockEmbeddingService is a mock implementation of EmbeddingService
type MockEmbeddingService struct {
	mock.Mock
}

func (m *MockEmbeddingService) GenerateEmbedding(ctx context.Context, knowledgeID string) error {
	args := m.Called(ctx, knowledgeID)
	return args.Error(0)
}

func (m *MockEmbeddingService) GenerateAssetEmbedding(ctx context.Context, assetID string) error {
	args := m.Called(ctx, assetID)
	return args.Error(0)
}

// TestWorker_StartStop tests the worker start and stop functionality
func TestWorker_StartStop(t *testing.T) {
	mockProcessor := new(MockJobProcessor)
	mockProcessor.On("ProcessJobs", mock.Anything).Return(nil)

	worker := NewWorker(mockProcessor, 100*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start worker in goroutine
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		worker.Start(ctx)
	}()

	// Let it run for a bit
	time.Sleep(250 * time.Millisecond)

	// Stop worker
	worker.Stop()
	wg.Wait()

	// Verify ProcessJobs was called at least once
	mockProcessor.AssertCalled(t, "ProcessJobs", mock.Anything)
}

// TestWorker_ContextCancellation tests worker stops on context cancellation
func TestWorker_ContextCancellation(t *testing.T) {
	mockProcessor := new(MockJobProcessor)
	mockProcessor.On("ProcessJobs", mock.Anything).Return(nil)

	worker := NewWorker(mockProcessor, 100*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())

	// Start worker in goroutine
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		worker.Start(ctx)
	}()

	// Let it run for a bit
	time.Sleep(150 * time.Millisecond)

	// Cancel context
	cancel()
	wg.Wait()

	// Verify ProcessJobs was called
	mockProcessor.AssertCalled(t, "ProcessJobs", mock.Anything)
}

// TestEmbeddingWorker_ProcessJobs_NoPendingJobs tests when there are no pending jobs
func TestEmbeddingWorker_ProcessJobs_NoPendingJobs(t *testing.T) {
	mockRepo := new(MockEmbeddingJobRepository)
	mockService := new(MockEmbeddingService)

	mockRepo.On("GetPendingJobs", mock.Anything).Return([]*domain.EmbeddingJob{}, nil)

	worker := NewEmbeddingWorker(mockRepo, mockService)
	err := worker.ProcessJobs(context.Background())

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockService.AssertNotCalled(t, "GenerateEmbedding", mock.Anything, mock.Anything)
}

// TestEmbeddingWorker_ProcessJobs_Success tests successful job processing
func TestEmbeddingWorker_ProcessJobs_Success(t *testing.T) {
	mockRepo := new(MockEmbeddingJobRepository)
	mockService := new(MockEmbeddingService)

	job := &domain.EmbeddingJob{
		ID:          "job-1",
		KnowledgeID: "knowledge-1",
		Status:      domain.EmbeddingJobStatusPending,
		Retries:     0,
	}

	mockRepo.On("GetPendingJobs", mock.Anything).Return([]*domain.EmbeddingJob{job}, nil)
	mockService.On("GenerateEmbedding", mock.Anything, "knowledge-1").Return(nil)
	mockRepo.On("UpdateJobStatus", mock.Anything, "job-1", domain.EmbeddingJobStatusCompleted, "").Return(nil)

	worker := NewEmbeddingWorker(mockRepo, mockService)
	err := worker.ProcessJobs(context.Background())

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockService.AssertExpectations(t)
}

// TestEmbeddingWorker_ProcessJobs_FailureWithRetry tests job failure with retry
func TestEmbeddingWorker_ProcessJobs_FailureWithRetry(t *testing.T) {
	mockRepo := new(MockEmbeddingJobRepository)
	mockService := new(MockEmbeddingService)

	job := &domain.EmbeddingJob{
		ID:          "job-1",
		KnowledgeID: "knowledge-1",
		Status:      domain.EmbeddingJobStatusPending,
		Retries:     0,
	}

	mockRepo.On("GetPendingJobs", mock.Anything).Return([]*domain.EmbeddingJob{job}, nil)
	mockService.On("GenerateEmbedding", mock.Anything, "knowledge-1").Return(errors.New("embedding failed"))
	mockRepo.On("IncrementRetries", mock.Anything, "job-1").Return(nil)
	mockRepo.On("UpdateJobStatus", mock.Anything, "job-1", domain.EmbeddingJobStatusPending, mock.MatchedBy(func(msg string) bool {
		return msg != ""
	})).Return(nil)

	worker := NewEmbeddingWorker(mockRepo, mockService)
	err := worker.ProcessJobs(context.Background())

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockService.AssertExpectations(t)
}

// TestEmbeddingWorker_ProcessJobs_MaxRetriesExceeded tests job failure after max retries
func TestEmbeddingWorker_ProcessJobs_MaxRetriesExceeded(t *testing.T) {
	mockRepo := new(MockEmbeddingJobRepository)
	mockService := new(MockEmbeddingService)

	job := &domain.EmbeddingJob{
		ID:          "job-1",
		KnowledgeID: "knowledge-1",
		Status:      domain.EmbeddingJobStatusPending,
		Retries:     2, // Already retried twice
	}

	mockRepo.On("GetPendingJobs", mock.Anything).Return([]*domain.EmbeddingJob{job}, nil)
	mockService.On("GenerateEmbedding", mock.Anything, "knowledge-1").Return(errors.New("embedding failed"))
	mockRepo.On("IncrementRetries", mock.Anything, "job-1").Return(nil)
	mockRepo.On("UpdateJobStatus", mock.Anything, "job-1", domain.EmbeddingJobStatusFailed, mock.MatchedBy(func(msg string) bool {
		return msg != ""
	})).Return(nil)

	worker := NewEmbeddingWorker(mockRepo, mockService)
	err := worker.ProcessJobs(context.Background())

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockService.AssertExpectations(t)
}

// TestEmbeddingWorker_ProcessJobs_MultipleJobs tests processing multiple jobs
func TestEmbeddingWorker_ProcessJobs_MultipleJobs(t *testing.T) {
	mockRepo := new(MockEmbeddingJobRepository)
	mockService := new(MockEmbeddingService)

	jobs := []*domain.EmbeddingJob{
		{
			ID:          "job-1",
			KnowledgeID: "knowledge-1",
			Status:      domain.EmbeddingJobStatusPending,
			Retries:     0,
		},
		{
			ID:          "job-2",
			KnowledgeID: "knowledge-2",
			Status:      domain.EmbeddingJobStatusPending,
			Retries:     0,
		},
	}

	mockRepo.On("GetPendingJobs", mock.Anything).Return(jobs, nil)

	// Job 1 succeeds
	mockService.On("GenerateEmbedding", mock.Anything, "knowledge-1").Return(nil)
	mockRepo.On("UpdateJobStatus", mock.Anything, "job-1", domain.EmbeddingJobStatusCompleted, "").Return(nil)

	// Job 2 succeeds
	mockService.On("GenerateEmbedding", mock.Anything, "knowledge-2").Return(nil)
	mockRepo.On("UpdateJobStatus", mock.Anything, "job-2", domain.EmbeddingJobStatusCompleted, "").Return(nil)

	worker := NewEmbeddingWorker(mockRepo, mockService)
	err := worker.ProcessJobs(context.Background())

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockService.AssertExpectations(t)
}

// TestEmbeddingWorker_ProcessJobs_RepositoryError tests repository error handling
func TestEmbeddingWorker_ProcessJobs_RepositoryError(t *testing.T) {
	mockRepo := new(MockEmbeddingJobRepository)
	mockService := new(MockEmbeddingService)

	mockRepo.On("GetPendingJobs", mock.Anything).Return(nil, errors.New("database error"))

	worker := NewEmbeddingWorker(mockRepo, mockService)
	err := worker.ProcessJobs(context.Background())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to fetch pending jobs")
	mockRepo.AssertExpectations(t)
}
