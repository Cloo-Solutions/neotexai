package openai

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockOpenAIAPI is a mock for the OpenAI API
type MockOpenAIAPI struct {
	mock.Mock
}

func (m *MockOpenAIAPI) CreateEmbeddings(ctx context.Context, text string) ([]float32, error) {
	args := m.Called(ctx, text)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]float32), args.Error(1)
}

func TestClient_GenerateEmbedding_Success(t *testing.T) {
	mockAPI := new(MockOpenAIAPI)
	client := &Client{api: mockAPI}

	ctx := context.Background()
	text := "This is a test document about Go programming."
	expectedEmbedding := make([]float32, 1536)
	for i := range expectedEmbedding {
		expectedEmbedding[i] = float32(i) * 0.001
	}

	mockAPI.On("CreateEmbeddings", ctx, text).Return(expectedEmbedding, nil)

	embedding, err := client.GenerateEmbedding(ctx, text)

	assert.NoError(t, err)
	assert.Len(t, embedding, 1536)
	assert.Equal(t, expectedEmbedding, embedding)
	mockAPI.AssertExpectations(t)
}

func TestClient_GenerateEmbedding_EmptyText(t *testing.T) {
	client := NewClient("")

	ctx := context.Background()
	embedding, err := client.GenerateEmbedding(ctx, "")

	assert.Error(t, err)
	assert.Nil(t, embedding)
	assert.Equal(t, ErrEmptyText, err)
}

func TestClient_GenerateEmbedding_APIError(t *testing.T) {
	mockAPI := new(MockOpenAIAPI)
	client := &Client{api: mockAPI}

	ctx := context.Background()
	text := "Test text"
	apiErr := errors.New("API rate limit exceeded")

	mockAPI.On("CreateEmbeddings", ctx, text).Return(nil, apiErr)

	embedding, err := client.GenerateEmbedding(ctx, text)

	assert.Error(t, err)
	assert.Nil(t, embedding)
	assert.Contains(t, err.Error(), "failed to create embedding")
	mockAPI.AssertExpectations(t)
}

func TestNewClient(t *testing.T) {
	apiKey := "test-api-key"
	client := NewClient(apiKey)

	assert.NotNil(t, client)
	assert.NotNil(t, client.api)
}

func TestClient_GenerateEmbedding_WrongDimensions(t *testing.T) {
	mockAPI := new(MockOpenAIAPI)
	client := &Client{api: mockAPI}

	ctx := context.Background()
	text := "Test text"
	// Return embedding with wrong dimensions
	wrongEmbedding := make([]float32, 512)

	mockAPI.On("CreateEmbeddings", ctx, text).Return(wrongEmbedding, nil)

	embedding, err := client.GenerateEmbedding(ctx, text)

	assert.Error(t, err)
	assert.Nil(t, embedding)
	assert.Equal(t, ErrWrongDimensions, err)
	mockAPI.AssertExpectations(t)
}

func TestNewClientFromEnv_NoAPIKey(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")

	client, err := NewClientFromEnv()

	assert.Nil(t, client)
	assert.Error(t, err)
	assert.Equal(t, ErrNoAPIKey, err)
}

func TestNewClientFromEnv_WithAPIKey(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-api-key")

	client, err := NewClientFromEnv()

	assert.NotNil(t, client)
	assert.NoError(t, err)
}
