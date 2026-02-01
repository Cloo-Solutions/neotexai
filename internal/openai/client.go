package openai

import (
	"context"
	"errors"
	"fmt"
	"os"

	openai "github.com/sashabaranov/go-openai"
)

const (
	// DefaultEmbeddingModel is the OpenAI model used for generating embeddings
	DefaultEmbeddingModel = openai.AdaEmbeddingV2
	// DefaultEmbeddingDimensions is the expected dimension of embeddings from ada-002
	DefaultEmbeddingDimensions = 1536
)

var (
	// ErrEmptyText is returned when text is empty
	ErrEmptyText = errors.New("text cannot be empty")
	// ErrWrongDimensions is returned when embedding has wrong dimensions
	ErrWrongDimensions = errors.New("embedding has wrong dimensions, expected 1536")
	// ErrNoAPIKey is returned when OpenAI API key is not set
	ErrNoAPIKey = errors.New("OPENAI_API_KEY environment variable not set")
)

// EmbeddingAPI defines the interface for embedding generation
type EmbeddingAPI interface {
	CreateEmbeddings(ctx context.Context, text string) ([]float32, error)
}

// Client wraps the OpenAI API client
type Client struct {
	api        EmbeddingAPI
	dimensions int
}

type OpenAIAdapter struct {
	client *openai.Client
	model  openai.EmbeddingModel
}

func NewOpenAIAdapter(apiKey string, model openai.EmbeddingModel) *OpenAIAdapter {
	if model == "" {
		model = DefaultEmbeddingModel
	}
	return &OpenAIAdapter{
		client: openai.NewClient(apiKey),
		model:  model,
	}
}

// CreateEmbeddings calls the OpenAI API to create embeddings
func (a *OpenAIAdapter) CreateEmbeddings(ctx context.Context, text string) ([]float32, error) {
	resp, err := a.client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Input: []string{text},
		Model: a.model,
	})
	if err != nil {
		return nil, err
	}

	if len(resp.Data) == 0 {
		return nil, errors.New("no embedding data returned")
	}

	return resp.Data[0].Embedding, nil
}

type Config struct {
	APIKey              string
	EmbeddingModel      openai.EmbeddingModel
	EmbeddingDimensions int
}

// NewClient creates a new OpenAI client using defaults.
func NewClient(apiKey string) *Client {
	return NewClientWithConfig(Config{APIKey: apiKey})
}

// NewClientWithConfig creates a new OpenAI client with explicit configuration.
func NewClientWithConfig(cfg Config) *Client {
	dimensions := cfg.EmbeddingDimensions
	if dimensions <= 0 {
		dimensions = DefaultEmbeddingDimensions
	}
	return &Client{
		api:        NewOpenAIAdapter(cfg.APIKey, cfg.EmbeddingModel),
		dimensions: dimensions,
	}
}

// NewClientFromEnv creates a new OpenAI client using OPENAI_API_KEY environment variable
func NewClientFromEnv() (*Client, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, ErrNoAPIKey
	}
	return NewClient(apiKey), nil
}

// GenerateEmbedding generates an embedding for the given text
func (c *Client) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	if text == "" {
		return nil, ErrEmptyText
	}

	embedding, err := c.api.CreateEmbeddings(ctx, text)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding: %w", err)
	}

	expected := c.dimensions
	if expected <= 0 {
		expected = DefaultEmbeddingDimensions
	}
	if len(embedding) != expected {
		return nil, ErrWrongDimensions
	}

	return embedding, nil
}
