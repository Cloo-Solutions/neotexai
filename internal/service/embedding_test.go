package service

import (
	"context"
	"errors"
	"testing"

	"github.com/cloo-solutions/neotexai/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockEmbeddingClient mocks the OpenAI client
type MockEmbeddingClient struct {
	mock.Mock
}

func (m *MockEmbeddingClient) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	args := m.Called(ctx, text)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]float32), args.Error(1)
}

// MockEmbeddingKnowledgeRepo mocks the knowledge repository for embedding service
type MockEmbeddingKnowledgeRepo struct {
	mock.Mock
}

func (m *MockEmbeddingKnowledgeRepo) GetByID(ctx context.Context, id string) (*domain.Knowledge, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Knowledge), args.Error(1)
}

func (m *MockEmbeddingKnowledgeRepo) UpdateEmbedding(ctx context.Context, id string, embedding []float32) error {
	args := m.Called(ctx, id, embedding)
	return args.Error(0)
}

func TestEmbeddingService_GenerateEmbedding_Success(t *testing.T) {
	mockClient := new(MockEmbeddingClient)
	mockRepo := new(MockEmbeddingKnowledgeRepo)
	service := NewEmbeddingService(mockClient, mockRepo)

	ctx := context.Background()
	knowledgeID := "knowledge-123"
	knowledge := &domain.Knowledge{
		ID:      knowledgeID,
		Title:   "Test Knowledge",
		Summary: "This is a summary",
		BodyMD:  "This is the body content in markdown.",
	}

	// Create a 1536-dim embedding
	embedding := make([]float32, 1536)
	for i := range embedding {
		embedding[i] = float32(i) * 0.001
	}

	// Expected text to embed: Title + Summary + BodyMD
	expectedText := "Test Knowledge\n\nThis is a summary\n\nThis is the body content in markdown."

	mockRepo.On("GetByID", ctx, knowledgeID).Return(knowledge, nil)
	mockClient.On("GenerateEmbedding", ctx, expectedText).Return(embedding, nil)
	mockRepo.On("UpdateEmbedding", ctx, knowledgeID, embedding).Return(nil)

	err := service.GenerateEmbedding(ctx, knowledgeID)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockClient.AssertExpectations(t)
}

func TestEmbeddingService_GenerateEmbedding_KnowledgeNotFound(t *testing.T) {
	mockClient := new(MockEmbeddingClient)
	mockRepo := new(MockEmbeddingKnowledgeRepo)
	service := NewEmbeddingService(mockClient, mockRepo)

	ctx := context.Background()
	knowledgeID := "nonexistent-id"

	mockRepo.On("GetByID", ctx, knowledgeID).Return(nil, domain.ErrKnowledgeNotFound)

	err := service.GenerateEmbedding(ctx, knowledgeID)

	assert.Error(t, err)
	assert.Equal(t, domain.ErrKnowledgeNotFound, err)
	mockRepo.AssertExpectations(t)
	mockClient.AssertNotCalled(t, "GenerateEmbedding")
}

func TestEmbeddingService_GenerateEmbedding_OpenAIError(t *testing.T) {
	mockClient := new(MockEmbeddingClient)
	mockRepo := new(MockEmbeddingKnowledgeRepo)
	service := NewEmbeddingService(mockClient, mockRepo)

	ctx := context.Background()
	knowledgeID := "knowledge-123"
	knowledge := &domain.Knowledge{
		ID:      knowledgeID,
		Title:   "Test Knowledge",
		Summary: "Summary",
		BodyMD:  "Body",
	}

	apiError := errors.New("OpenAI API rate limit exceeded")

	mockRepo.On("GetByID", ctx, knowledgeID).Return(knowledge, nil)
	mockClient.On("GenerateEmbedding", ctx, mock.Anything).Return(nil, apiError)

	err := service.GenerateEmbedding(ctx, knowledgeID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to generate embedding")
	mockRepo.AssertExpectations(t)
	mockClient.AssertExpectations(t)
	mockRepo.AssertNotCalled(t, "UpdateEmbedding")
}

func TestEmbeddingService_GenerateEmbedding_UpdateError(t *testing.T) {
	mockClient := new(MockEmbeddingClient)
	mockRepo := new(MockEmbeddingKnowledgeRepo)
	service := NewEmbeddingService(mockClient, mockRepo)

	ctx := context.Background()
	knowledgeID := "knowledge-123"
	knowledge := &domain.Knowledge{
		ID:      knowledgeID,
		Title:   "Test",
		Summary: "Summary",
		BodyMD:  "Body",
	}

	embedding := make([]float32, 1536)
	dbError := errors.New("database connection lost")

	mockRepo.On("GetByID", ctx, knowledgeID).Return(knowledge, nil)
	mockClient.On("GenerateEmbedding", ctx, mock.Anything).Return(embedding, nil)
	mockRepo.On("UpdateEmbedding", ctx, knowledgeID, embedding).Return(dbError)

	err := service.GenerateEmbedding(ctx, knowledgeID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update embedding")
	mockRepo.AssertExpectations(t)
	mockClient.AssertExpectations(t)
}

func TestEmbeddingService_GenerateEmbedding_EmptyBody(t *testing.T) {
	mockClient := new(MockEmbeddingClient)
	mockRepo := new(MockEmbeddingKnowledgeRepo)
	service := NewEmbeddingService(mockClient, mockRepo)

	ctx := context.Background()
	knowledgeID := "knowledge-123"
	knowledge := &domain.Knowledge{
		ID:      knowledgeID,
		Title:   "Title Only",
		Summary: "",
		BodyMD:  "",
	}

	embedding := make([]float32, 1536)
	expectedText := "Title Only"

	mockRepo.On("GetByID", ctx, knowledgeID).Return(knowledge, nil)
	mockClient.On("GenerateEmbedding", ctx, expectedText).Return(embedding, nil)
	mockRepo.On("UpdateEmbedding", ctx, knowledgeID, embedding).Return(nil)

	err := service.GenerateEmbedding(ctx, knowledgeID)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockClient.AssertExpectations(t)
}

func TestNewEmbeddingService(t *testing.T) {
	mockClient := new(MockEmbeddingClient)
	mockRepo := new(MockEmbeddingKnowledgeRepo)

	service := NewEmbeddingService(mockClient, mockRepo)

	assert.NotNil(t, service)
}
