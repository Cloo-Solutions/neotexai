package service

import (
	"context"
	"fmt"
	"time"

	"github.com/cloo-solutions/neotexai/internal/domain"
	"github.com/google/uuid"
)

type StorageClientInterface interface {
	GenerateUploadURL(ctx context.Context, key string, contentType string) (string, error)
	GenerateDownloadURL(ctx context.Context, key string) (string, error)
	DeleteObject(ctx context.Context, key string) error
	HeadObject(ctx context.Context, key string) (*ObjectMetadata, error)
}

type ObjectMetadata struct {
	ContentLength int64
	ContentType   string
	ETag          string
}

type AssetRepositoryInterface interface {
	Create(ctx context.Context, a *domain.Asset) error
	GetByID(ctx context.Context, id string) (*domain.Asset, error)
	Delete(ctx context.Context, id string) error
	LinkToKnowledge(ctx context.Context, knowledgeID, assetID string) error
	UnlinkFromKnowledge(ctx context.Context, knowledgeID, assetID string) error
}

type AssetEmbeddingJobRepository interface {
	Create(ctx context.Context, job *domain.EmbeddingJob) error
}

type AssetService struct {
	assetRepo        AssetRepositoryInterface
	storageClient    StorageClientInterface
	embeddingJobRepo AssetEmbeddingJobRepository
	uuidGen          UUIDGenerator
	txRunner         TxRunner
}

func NewAssetService(assetRepo AssetRepositoryInterface, storageClient StorageClientInterface) *AssetService {
	return &AssetService{
		assetRepo:     assetRepo,
		storageClient: storageClient,
		uuidGen:       &DefaultUUIDGenerator{},
		txRunner:      nil,
	}
}

func NewAssetServiceWithEmbeddings(assetRepo AssetRepositoryInterface, storageClient StorageClientInterface, embeddingJobRepo AssetEmbeddingJobRepository) *AssetService {
	return &AssetService{
		assetRepo:        assetRepo,
		storageClient:    storageClient,
		embeddingJobRepo: embeddingJobRepo,
		uuidGen:          &DefaultUUIDGenerator{},
		txRunner:         nil,
	}
}

func NewAssetServiceWithEmbeddingsAndTx(assetRepo AssetRepositoryInterface, storageClient StorageClientInterface, embeddingJobRepo AssetEmbeddingJobRepository, txRunner TxRunner) *AssetService {
	return &AssetService{
		assetRepo:        assetRepo,
		storageClient:    storageClient,
		embeddingJobRepo: embeddingJobRepo,
		uuidGen:          &DefaultUUIDGenerator{},
		txRunner:         txRunner,
	}
}

func NewAssetServiceWithUUIDGen(
	assetRepo AssetRepositoryInterface,
	storageClient StorageClientInterface,
	uuidGen UUIDGenerator,
) *AssetService {
	return &AssetService{
		assetRepo:     assetRepo,
		storageClient: storageClient,
		uuidGen:       uuidGen,
		txRunner:      nil,
	}
}

type InitUploadInput struct {
	OrgID       string
	ProjectID   string
	Filename    string
	ContentType string
}

type InitUploadResult struct {
	AssetID    string
	StorageKey string
	UploadURL  string
}

func (s *AssetService) InitUpload(ctx context.Context, input InitUploadInput) (*InitUploadResult, error) {
	assetID := s.uuidGen.NewString()

	storageKey := buildStorageKey(input.OrgID, assetID, input.Filename)

	uploadURL, err := s.storageClient.GenerateUploadURL(ctx, storageKey, input.ContentType)
	if err != nil {
		return nil, fmt.Errorf("failed to generate upload URL: %w", err)
	}

	return &InitUploadResult{
		AssetID:    assetID,
		StorageKey: storageKey,
		UploadURL:  uploadURL,
	}, nil
}

type CompleteUploadInput struct {
	AssetID     string
	OrgID       string
	ProjectID   string
	Filename    string
	ContentType string
	StorageKey  string
	SHA256      string
	Keywords    []string
	Description string
	KnowledgeID *string
}

func (s *AssetService) CompleteUpload(ctx context.Context, input CompleteUploadInput) (*domain.Asset, error) {
	_, err := s.storageClient.HeadObject(ctx, input.StorageKey)
	if err != nil {
		return nil, fmt.Errorf("failed to verify uploaded file: %w", err)
	}

	now := time.Now().UTC()
	asset := &domain.Asset{
		ID:          input.AssetID,
		OrgID:       input.OrgID,
		ProjectID:   input.ProjectID,
		Filename:    input.Filename,
		MimeType:    input.ContentType,
		SHA256:      input.SHA256,
		StorageKey:  input.StorageKey,
		Keywords:    input.Keywords,
		Description: input.Description,
		CreatedAt:   now,
	}

	if s.txRunner != nil {
		if err := s.txRunner.WithTx(ctx, func(repos TxRepositories) error {
			assetRepo := repos.Assets()

			if err := assetRepo.Create(ctx, asset); err != nil {
				return fmt.Errorf("failed to create asset record: %w", err)
			}

			if input.KnowledgeID != nil && *input.KnowledgeID != "" {
				if err := assetRepo.LinkToKnowledge(ctx, *input.KnowledgeID, input.AssetID); err != nil {
					return fmt.Errorf("failed to link asset to knowledge: %w", err)
				}
			}

			if s.embeddingJobRepo != nil && (input.Description != "" || len(input.Keywords) > 0) {
				job := &domain.EmbeddingJob{
					ID:        uuid.NewString(),
					AssetID:   input.AssetID,
					Status:    domain.EmbeddingJobStatusPending,
					CreatedAt: now,
				}
				if err := repos.EmbeddingJobs().Create(ctx, job); err != nil {
					return fmt.Errorf("failed to create embedding job: %w", err)
				}
			}
			return nil
		}); err != nil {
			return nil, err
		}
		return asset, nil
	}

	if err := s.assetRepo.Create(ctx, asset); err != nil {
		return nil, fmt.Errorf("failed to create asset record: %w", err)
	}

	if input.KnowledgeID != nil && *input.KnowledgeID != "" {
		if err := s.assetRepo.LinkToKnowledge(ctx, *input.KnowledgeID, input.AssetID); err != nil {
			return nil, fmt.Errorf("failed to link asset to knowledge: %w", err)
		}
	}

	if s.embeddingJobRepo != nil && (input.Description != "" || len(input.Keywords) > 0) {
		job := &domain.EmbeddingJob{
			ID:        uuid.NewString(),
			AssetID:   input.AssetID,
			Status:    domain.EmbeddingJobStatusPending,
			CreatedAt: now,
		}
		if err := s.embeddingJobRepo.Create(ctx, job); err != nil {
			return nil, fmt.Errorf("failed to create embedding job: %w", err)
		}
	}

	return asset, nil
}

func (s *AssetService) GetDownloadURL(ctx context.Context, assetID string) (string, error) {
	asset, err := s.assetRepo.GetByID(ctx, assetID)
	if err != nil {
		return "", err
	}

	url, err := s.storageClient.GenerateDownloadURL(ctx, asset.StorageKey)
	if err != nil {
		return "", fmt.Errorf("failed to generate download URL: %w", err)
	}

	return url, nil
}

func (s *AssetService) Delete(ctx context.Context, assetID string) error {
	asset, err := s.assetRepo.GetByID(ctx, assetID)
	if err != nil {
		return err
	}

	if err := s.storageClient.DeleteObject(ctx, asset.StorageKey); err != nil {
		return fmt.Errorf("failed to delete from storage: %w", err)
	}

	if err := s.assetRepo.Delete(ctx, assetID); err != nil {
		return fmt.Errorf("failed to delete asset record: %w", err)
	}

	return nil
}

func (s *AssetService) GetByID(ctx context.Context, assetID string) (*domain.Asset, error) {
	return s.assetRepo.GetByID(ctx, assetID)
}

func buildStorageKey(orgID, assetID, filename string) string {
	return fmt.Sprintf("%s/%s/%s", orgID, assetID, filename)
}

var _ UUIDGenerator = (*DefaultUUIDGenerator)(nil)

func init() {
	_ = uuid.NewString
}
