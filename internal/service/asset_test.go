package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/cloo-solutions/neotexai/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockStorageClient is a mock implementation of StorageClientInterface
type MockStorageClient struct {
	mock.Mock
}

func (m *MockStorageClient) GenerateUploadURL(ctx context.Context, key string, contentType string) (string, error) {
	args := m.Called(ctx, key, contentType)
	return args.String(0), args.Error(1)
}

func (m *MockStorageClient) GenerateDownloadURL(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)
	return args.String(0), args.Error(1)
}

func (m *MockStorageClient) DeleteObject(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockStorageClient) HeadObject(ctx context.Context, key string) (*ObjectMetadata, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ObjectMetadata), args.Error(1)
}

// MockAssetRepository is a mock implementation of AssetRepositoryInterface
type MockAssetRepository struct {
	mock.Mock
}

func (m *MockAssetRepository) Create(ctx context.Context, a *domain.Asset) error {
	args := m.Called(ctx, a)
	return args.Error(0)
}

func (m *MockAssetRepository) GetByID(ctx context.Context, id string) (*domain.Asset, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Asset), args.Error(1)
}

func (m *MockAssetRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockAssetRepository) LinkToKnowledge(ctx context.Context, knowledgeID, assetID string) error {
	args := m.Called(ctx, knowledgeID, assetID)
	return args.Error(0)
}

func (m *MockAssetRepository) UnlinkFromKnowledge(ctx context.Context, knowledgeID, assetID string) error {
	args := m.Called(ctx, knowledgeID, assetID)
	return args.Error(0)
}

// MockUUIDGeneratorAsset is a mock UUID generator for asset tests
type MockUUIDGeneratorAsset struct {
	uuids []string
	index int
}

func NewMockUUIDGeneratorAsset(uuids ...string) *MockUUIDGeneratorAsset {
	return &MockUUIDGeneratorAsset{uuids: uuids}
}

func (m *MockUUIDGeneratorAsset) NewString() string {
	if m.index >= len(m.uuids) {
		return "default-uuid"
	}
	uuid := m.uuids[m.index]
	m.index++
	return uuid
}

func TestAssetService_InitUpload_Success(t *testing.T) {
	ctx := context.Background()
	mockStorage := new(MockStorageClient)
	mockRepo := new(MockAssetRepository)
	uuidGen := NewMockUUIDGeneratorAsset("asset-id-123")

	svc := NewAssetServiceWithUUIDGen(mockRepo, mockStorage, uuidGen)

	input := InitUploadInput{
		OrgID:       "org-123",
		ProjectID:   "proj-456",
		Filename:    "document.pdf",
		ContentType: "application/pdf",
	}

	expectedKey := "org-123/asset-id-123/document.pdf"
	expectedURL := "https://s3.example.com/presigned-upload-url"

	mockStorage.On("GenerateUploadURL", ctx, expectedKey, "application/pdf").
		Return(expectedURL, nil)

	result, err := svc.InitUpload(ctx, input)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "asset-id-123", result.AssetID)
	assert.Equal(t, expectedKey, result.StorageKey)
	assert.Equal(t, expectedURL, result.UploadURL)

	mockStorage.AssertExpectations(t)
}

func TestAssetService_CompleteUpload_Success(t *testing.T) {
	ctx := context.Background()
	mockStorage := new(MockStorageClient)
	mockRepo := new(MockAssetRepository)
	uuidGen := NewMockUUIDGeneratorAsset()

	svc := NewAssetServiceWithUUIDGen(mockRepo, mockStorage, uuidGen)

	input := CompleteUploadInput{
		AssetID:     "asset-id-123",
		OrgID:       "org-123",
		ProjectID:   "proj-456",
		Filename:    "document.pdf",
		ContentType: "application/pdf",
		StorageKey:  "org-123/asset-id-123/document.pdf",
		SHA256:      "abc123def456",
		Keywords:    []string{"document", "pdf"},
		Description: "Test document",
	}

	mockStorage.On("HeadObject", ctx, input.StorageKey).
		Return(&ObjectMetadata{
			ContentLength: 1024,
			ContentType:   "application/pdf",
			ETag:          "\"abc123def456\"",
		}, nil)

	mockRepo.On("Create", ctx, mock.MatchedBy(func(a *domain.Asset) bool {
		return a.ID == input.AssetID &&
			a.OrgID == input.OrgID &&
			a.ProjectID == input.ProjectID &&
			a.Filename == input.Filename &&
			a.MimeType == input.ContentType &&
			a.SHA256 == input.SHA256 &&
			a.StorageKey == input.StorageKey
	})).Return(nil)

	asset, err := svc.CompleteUpload(ctx, input)

	assert.NoError(t, err)
	assert.NotNil(t, asset)
	assert.Equal(t, input.AssetID, asset.ID)
	assert.Equal(t, input.OrgID, asset.OrgID)
	assert.Equal(t, input.SHA256, asset.SHA256)

	mockStorage.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestAssetService_CompleteUpload_FileNotFound(t *testing.T) {
	ctx := context.Background()
	mockStorage := new(MockStorageClient)
	mockRepo := new(MockAssetRepository)
	uuidGen := NewMockUUIDGeneratorAsset()

	svc := NewAssetServiceWithUUIDGen(mockRepo, mockStorage, uuidGen)

	input := CompleteUploadInput{
		AssetID:     "asset-id-123",
		OrgID:       "org-123",
		Filename:    "document.pdf",
		ContentType: "application/pdf",
		StorageKey:  "org-123/asset-id-123/document.pdf",
		SHA256:      "expected-hash-123",
	}

	mockStorage.On("HeadObject", ctx, input.StorageKey).
		Return(nil, fmt.Errorf("file not found"))

	asset, err := svc.CompleteUpload(ctx, input)

	assert.Error(t, err)
	assert.Nil(t, asset)
	assert.Contains(t, err.Error(), "failed to verify uploaded file")

	mockStorage.AssertExpectations(t)
}

func TestAssetService_CompleteUpload_LinkToKnowledge(t *testing.T) {
	ctx := context.Background()
	mockStorage := new(MockStorageClient)
	mockRepo := new(MockAssetRepository)
	uuidGen := NewMockUUIDGeneratorAsset()

	svc := NewAssetServiceWithUUIDGen(mockRepo, mockStorage, uuidGen)

	knowledgeID := "knowledge-789"
	input := CompleteUploadInput{
		AssetID:     "asset-id-123",
		OrgID:       "org-123",
		Filename:    "document.pdf",
		ContentType: "application/pdf",
		StorageKey:  "org-123/asset-id-123/document.pdf",
		SHA256:      "abc123def456",
		KnowledgeID: &knowledgeID,
	}

	mockStorage.On("HeadObject", ctx, input.StorageKey).
		Return(&ObjectMetadata{
			ContentLength: 1024,
			ContentType:   "application/pdf",
			ETag:          "\"abc123def456\"",
		}, nil)

	mockRepo.On("Create", ctx, mock.AnythingOfType("*domain.Asset")).Return(nil)
	mockRepo.On("LinkToKnowledge", ctx, knowledgeID, input.AssetID).Return(nil)

	asset, err := svc.CompleteUpload(ctx, input)

	assert.NoError(t, err)
	assert.NotNil(t, asset)

	mockStorage.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestAssetService_GetDownloadURL_Success(t *testing.T) {
	ctx := context.Background()
	mockStorage := new(MockStorageClient)
	mockRepo := new(MockAssetRepository)
	uuidGen := NewMockUUIDGeneratorAsset()

	svc := NewAssetServiceWithUUIDGen(mockRepo, mockStorage, uuidGen)

	asset := &domain.Asset{
		ID:         "asset-id-123",
		OrgID:      "org-123",
		Filename:   "document.pdf",
		StorageKey: "org-123/asset-id-123/document.pdf",
		CreatedAt:  time.Now(),
	}

	expectedURL := "https://s3.example.com/presigned-download-url"

	mockRepo.On("GetByID", ctx, "asset-id-123").Return(asset, nil)
	mockStorage.On("GenerateDownloadURL", ctx, asset.StorageKey).Return(expectedURL, nil)

	url, err := svc.GetDownloadURL(ctx, "asset-id-123")

	assert.NoError(t, err)
	assert.Equal(t, expectedURL, url)

	mockRepo.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
}

func TestAssetService_GetDownloadURL_AssetNotFound(t *testing.T) {
	ctx := context.Background()
	mockStorage := new(MockStorageClient)
	mockRepo := new(MockAssetRepository)
	uuidGen := NewMockUUIDGeneratorAsset()

	svc := NewAssetServiceWithUUIDGen(mockRepo, mockStorage, uuidGen)

	mockRepo.On("GetByID", ctx, "nonexistent").Return(nil, domain.ErrAssetNotFound)

	url, err := svc.GetDownloadURL(ctx, "nonexistent")

	assert.Error(t, err)
	assert.Equal(t, domain.ErrAssetNotFound, err)
	assert.Empty(t, url)

	mockRepo.AssertExpectations(t)
}

func TestAssetService_Delete_Success(t *testing.T) {
	ctx := context.Background()
	mockStorage := new(MockStorageClient)
	mockRepo := new(MockAssetRepository)
	uuidGen := NewMockUUIDGeneratorAsset()

	svc := NewAssetServiceWithUUIDGen(mockRepo, mockStorage, uuidGen)

	asset := &domain.Asset{
		ID:         "asset-id-123",
		OrgID:      "org-123",
		Filename:   "document.pdf",
		StorageKey: "org-123/asset-id-123/document.pdf",
	}

	mockRepo.On("GetByID", ctx, "asset-id-123").Return(asset, nil)
	mockStorage.On("DeleteObject", ctx, asset.StorageKey).Return(nil)
	mockRepo.On("Delete", ctx, "asset-id-123").Return(nil)

	err := svc.Delete(ctx, "asset-id-123")

	assert.NoError(t, err)

	mockRepo.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
}

func TestAssetService_Delete_AssetNotFound(t *testing.T) {
	ctx := context.Background()
	mockStorage := new(MockStorageClient)
	mockRepo := new(MockAssetRepository)
	uuidGen := NewMockUUIDGeneratorAsset()

	svc := NewAssetServiceWithUUIDGen(mockRepo, mockStorage, uuidGen)

	mockRepo.On("GetByID", ctx, "nonexistent").Return(nil, domain.ErrAssetNotFound)

	err := svc.Delete(ctx, "nonexistent")

	assert.Error(t, err)
	assert.Equal(t, domain.ErrAssetNotFound, err)

	mockRepo.AssertExpectations(t)
}

func TestAssetService_GetByID_Success(t *testing.T) {
	ctx := context.Background()
	mockStorage := new(MockStorageClient)
	mockRepo := new(MockAssetRepository)
	uuidGen := NewMockUUIDGeneratorAsset()

	svc := NewAssetServiceWithUUIDGen(mockRepo, mockStorage, uuidGen)

	expectedAsset := &domain.Asset{
		ID:          "asset-id-123",
		OrgID:       "org-123",
		ProjectID:   "proj-456",
		Filename:    "document.pdf",
		MimeType:    "application/pdf",
		SHA256:      "abc123",
		StorageKey:  "org-123/asset-id-123/document.pdf",
		Keywords:    []string{"test"},
		Description: "Test document",
		CreatedAt:   time.Now(),
	}

	mockRepo.On("GetByID", ctx, "asset-id-123").Return(expectedAsset, nil)

	asset, err := svc.GetByID(ctx, "asset-id-123")

	assert.NoError(t, err)
	assert.Equal(t, expectedAsset, asset)

	mockRepo.AssertExpectations(t)
}
