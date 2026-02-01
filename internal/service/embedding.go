package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloo-solutions/neotexai/internal/domain"
)

// EmbeddingClient defines the interface for generating embeddings
type EmbeddingClient interface {
	GenerateEmbedding(ctx context.Context, text string) ([]float32, error)
}

// EmbeddingKnowledgeRepository defines the repository interface for embedding operations
type EmbeddingKnowledgeRepository interface {
	GetByID(ctx context.Context, id string) (*domain.Knowledge, error)
	UpdateEmbedding(ctx context.Context, id string, embedding []float32) error
}

// EmbeddingAssetRepository defines the repository interface for asset embedding operations
type EmbeddingAssetRepository interface {
	GetByID(ctx context.Context, id string) (*domain.Asset, error)
	UpdateEmbedding(ctx context.Context, id string, embedding []float32) error
}

// EmbeddingService handles embedding generation for knowledge and asset items
type EmbeddingService struct {
	client    EmbeddingClient
	repo      EmbeddingKnowledgeRepository
	assetRepo EmbeddingAssetRepository
}

// NewEmbeddingService creates a new EmbeddingService instance
func NewEmbeddingService(client EmbeddingClient, repo EmbeddingKnowledgeRepository) *EmbeddingService {
	return &EmbeddingService{
		client: client,
		repo:   repo,
	}
}

func NewEmbeddingServiceWithAssets(client EmbeddingClient, repo EmbeddingKnowledgeRepository, assetRepo EmbeddingAssetRepository) *EmbeddingService {
	return &EmbeddingService{
		client:    client,
		repo:      repo,
		assetRepo: assetRepo,
	}
}

// GenerateEmbedding generates and stores an embedding for the given knowledge ID
// This method is called by the background worker
func (s *EmbeddingService) GenerateEmbedding(ctx context.Context, knowledgeID string) error {
	// Fetch the knowledge item
	knowledge, err := s.repo.GetByID(ctx, knowledgeID)
	if err != nil {
		return err
	}

	// Build the text to embed from title, summary, and body
	text := buildEmbeddingText(knowledge)

	// Generate embedding using OpenAI
	embedding, err := s.client.GenerateEmbedding(ctx, text)
	if err != nil {
		return fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Store the embedding
	if err := s.repo.UpdateEmbedding(ctx, knowledgeID, embedding); err != nil {
		return fmt.Errorf("failed to update embedding: %w", err)
	}

	return nil
}

func (s *EmbeddingService) GenerateAssetEmbedding(ctx context.Context, assetID string) error {
	if s.assetRepo == nil {
		return fmt.Errorf("asset repository not configured")
	}

	asset, err := s.assetRepo.GetByID(ctx, assetID)
	if err != nil {
		return err
	}

	text := buildAssetEmbeddingText(asset)
	if text == "" {
		return fmt.Errorf("asset has no description or keywords to embed")
	}

	embedding, err := s.client.GenerateEmbedding(ctx, text)
	if err != nil {
		return fmt.Errorf("failed to generate embedding: %w", err)
	}

	if err := s.assetRepo.UpdateEmbedding(ctx, assetID, embedding); err != nil {
		return fmt.Errorf("failed to update embedding: %w", err)
	}

	return nil
}

func buildEmbeddingText(k *domain.Knowledge) string {
	var parts []string

	if k.Title != "" {
		parts = append(parts, k.Title)
	}
	if k.Summary != "" {
		parts = append(parts, k.Summary)
	}
	if k.BodyMD != "" {
		parts = append(parts, k.BodyMD)
	}

	return strings.Join(parts, "\n\n")
}

func buildAssetEmbeddingText(a *domain.Asset) string {
	var parts []string

	parts = append(parts, fmt.Sprintf("File: %s", a.Filename))
	if a.Description != "" {
		parts = append(parts, a.Description)
	}
	if len(a.Keywords) > 0 {
		parts = append(parts, fmt.Sprintf("Keywords: %s", strings.Join(a.Keywords, ", ")))
	}

	return strings.Join(parts, "\n\n")
}
