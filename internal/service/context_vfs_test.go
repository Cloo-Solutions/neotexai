package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cloo-solutions/neotexai/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Mock implementations

type MockVFSKnowledgeRepo struct {
	mock.Mock
}

func (m *MockVFSKnowledgeRepo) GetByID(ctx context.Context, id string) (*domain.Knowledge, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Knowledge), args.Error(1)
}

type MockVFSChunkRepo struct {
	mock.Mock
}

func (m *MockVFSChunkRepo) GetByID(ctx context.Context, chunkID string) (*domain.KnowledgeChunk, error) {
	args := m.Called(ctx, chunkID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.KnowledgeChunk), args.Error(1)
}

func (m *MockVFSChunkRepo) GetByKnowledgeID(ctx context.Context, knowledgeID string) ([]*domain.KnowledgeChunk, error) {
	args := m.Called(ctx, knowledgeID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.KnowledgeChunk), args.Error(1)
}

func (m *MockVFSChunkRepo) CountByKnowledgeID(ctx context.Context, knowledgeID string) (int, error) {
	args := m.Called(ctx, knowledgeID)
	return args.Int(0), args.Error(1)
}

type MockVFSAssetRepo struct {
	mock.Mock
}

func (m *MockVFSAssetRepo) GetByID(ctx context.Context, id string) (*domain.Asset, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Asset), args.Error(1)
}

type MockVFSStorage struct {
	mock.Mock
}

func (m *MockVFSStorage) GenerateDownloadURL(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)
	return args.String(0), args.Error(1)
}

type MockVFSListRepo struct {
	mock.Mock
}

func (m *MockVFSListRepo) ListKnowledge(ctx context.Context, input ListInput) ([]*ListItem, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*ListItem), args.Error(1)
}

func (m *MockVFSListRepo) ListAssets(ctx context.Context, input ListInput) ([]*ListItem, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*ListItem), args.Error(1)
}

// Tests

func TestVFSService_Open_Knowledge(t *testing.T) {
	t.Run("opens knowledge item and returns content", func(t *testing.T) {
		knowledgeRepo := new(MockVFSKnowledgeRepo)
		chunkRepo := new(MockVFSChunkRepo)
		assetRepo := new(MockVFSAssetRepo)
		storage := new(MockVFSStorage)
		listRepo := new(MockVFSListRepo)

		svc := NewVFSService(knowledgeRepo, chunkRepo, assetRepo, storage, listRepo)

		knowledge := &domain.Knowledge{
			ID:        "k-123",
			Title:     "Test Knowledge",
			BodyMD:    "Line 1\nLine 2\nLine 3",
			UpdatedAt: time.Now(),
		}

		knowledgeRepo.On("GetByID", mock.Anything, "k-123").Return(knowledge, nil)
		chunkRepo.On("CountByKnowledgeID", mock.Anything, "k-123").Return(3, nil)

		result, err := svc.Open(context.Background(), OpenInput{
			ID:         "k-123",
			SourceType: "knowledge",
		})

		require.NoError(t, err)
		assert.Equal(t, "k-123", result.ID)
		assert.Equal(t, "knowledge", result.SourceType)
		assert.Equal(t, "Test Knowledge", result.Title)
		assert.Equal(t, "Line 1\nLine 2\nLine 3", result.Content)
		assert.Equal(t, 3, result.TotalLines)
		assert.Equal(t, 3, result.ChunkCount)
		assert.Equal(t, -1, result.ChunkIndex) // No specific chunk

		knowledgeRepo.AssertExpectations(t)
		chunkRepo.AssertExpectations(t)
	})

	t.Run("opens knowledge with line range", func(t *testing.T) {
		knowledgeRepo := new(MockVFSKnowledgeRepo)
		chunkRepo := new(MockVFSChunkRepo)
		assetRepo := new(MockVFSAssetRepo)
		storage := new(MockVFSStorage)
		listRepo := new(MockVFSListRepo)

		svc := NewVFSService(knowledgeRepo, chunkRepo, assetRepo, storage, listRepo)

		knowledge := &domain.Knowledge{
			ID:        "k-123",
			Title:     "Test Knowledge",
			BodyMD:    "Line 0\nLine 1\nLine 2\nLine 3\nLine 4",
			UpdatedAt: time.Now(),
		}

		knowledgeRepo.On("GetByID", mock.Anything, "k-123").Return(knowledge, nil)
		chunkRepo.On("CountByKnowledgeID", mock.Anything, "k-123").Return(0, nil)

		result, err := svc.Open(context.Background(), OpenInput{
			ID:         "k-123",
			SourceType: "knowledge",
			Range: &ContentRange{
				StartLine: 1,
				EndLine:   3,
			},
		})

		require.NoError(t, err)
		assert.Equal(t, "Line 1\nLine 2", result.Content)
		assert.Equal(t, 5, result.TotalLines)
	})

	t.Run("opens knowledge with max chars limit", func(t *testing.T) {
		knowledgeRepo := new(MockVFSKnowledgeRepo)
		chunkRepo := new(MockVFSChunkRepo)
		assetRepo := new(MockVFSAssetRepo)
		storage := new(MockVFSStorage)
		listRepo := new(MockVFSListRepo)

		svc := NewVFSService(knowledgeRepo, chunkRepo, assetRepo, storage, listRepo)

		knowledge := &domain.Knowledge{
			ID:        "k-123",
			Title:     "Test Knowledge",
			BodyMD:    "This is a long content that should be truncated",
			UpdatedAt: time.Now(),
		}

		knowledgeRepo.On("GetByID", mock.Anything, "k-123").Return(knowledge, nil)
		chunkRepo.On("CountByKnowledgeID", mock.Anything, "k-123").Return(0, nil)

		result, err := svc.Open(context.Background(), OpenInput{
			ID:         "k-123",
			SourceType: "knowledge",
			Range: &ContentRange{
				MaxChars: 10,
			},
		})

		require.NoError(t, err)
		assert.Equal(t, "This is a ", result.Content)
		assert.Equal(t, 10, len(result.Content))
	})

	t.Run("returns error when knowledge not found", func(t *testing.T) {
		knowledgeRepo := new(MockVFSKnowledgeRepo)
		chunkRepo := new(MockVFSChunkRepo)
		assetRepo := new(MockVFSAssetRepo)
		storage := new(MockVFSStorage)
		listRepo := new(MockVFSListRepo)

		svc := NewVFSService(knowledgeRepo, chunkRepo, assetRepo, storage, listRepo)

		knowledgeRepo.On("GetByID", mock.Anything, "not-found").Return(nil, domain.ErrKnowledgeNotFound)

		_, err := svc.Open(context.Background(), OpenInput{
			ID:         "not-found",
			SourceType: "knowledge",
		})

		require.Error(t, err)
		assert.Equal(t, domain.ErrKnowledgeNotFound, err)
	})
}

func TestVFSService_Open_Chunk(t *testing.T) {
	t.Run("opens specific chunk by chunk_id", func(t *testing.T) {
		knowledgeRepo := new(MockVFSKnowledgeRepo)
		chunkRepo := new(MockVFSChunkRepo)
		assetRepo := new(MockVFSAssetRepo)
		storage := new(MockVFSStorage)
		listRepo := new(MockVFSListRepo)

		svc := NewVFSService(knowledgeRepo, chunkRepo, assetRepo, storage, listRepo)

		chunk := &domain.KnowledgeChunk{
			ID:          "c-456",
			KnowledgeID: "k-123",
			Title:       "Chunk Title",
			Content:     "Chunk content here",
			ChunkIndex:  2,
			UpdatedAt:   time.Now(),
		}

		chunkRepo.On("GetByID", mock.Anything, "c-456").Return(chunk, nil)
		chunkRepo.On("CountByKnowledgeID", mock.Anything, "k-123").Return(5, nil)

		result, err := svc.Open(context.Background(), OpenInput{
			ID:         "c-456",
			SourceType: "chunk",
		})

		require.NoError(t, err)
		assert.Equal(t, "k-123", result.ID)
		assert.Equal(t, "knowledge", result.SourceType)
		assert.Equal(t, "Chunk Title", result.Title)
		assert.Equal(t, "Chunk content here", result.Content)
		assert.Equal(t, "c-456", result.ChunkID)
		assert.Equal(t, 2, result.ChunkIndex)
		assert.Equal(t, 5, result.ChunkCount)

		chunkRepo.AssertExpectations(t)
	})

	t.Run("opens chunk via knowledge ID with chunk_id parameter", func(t *testing.T) {
		knowledgeRepo := new(MockVFSKnowledgeRepo)
		chunkRepo := new(MockVFSChunkRepo)
		assetRepo := new(MockVFSAssetRepo)
		storage := new(MockVFSStorage)
		listRepo := new(MockVFSListRepo)

		svc := NewVFSService(knowledgeRepo, chunkRepo, assetRepo, storage, listRepo)

		chunk := &domain.KnowledgeChunk{
			ID:          "c-456",
			KnowledgeID: "k-123",
			Title:       "Chunk Title",
			Content:     "Chunk content here",
			ChunkIndex:  1,
			UpdatedAt:   time.Now(),
		}

		chunkRepo.On("GetByID", mock.Anything, "c-456").Return(chunk, nil)
		chunkRepo.On("CountByKnowledgeID", mock.Anything, "k-123").Return(3, nil)

		result, err := svc.Open(context.Background(), OpenInput{
			ID:         "k-123",
			SourceType: "knowledge",
			ChunkID:    "c-456",
		})

		require.NoError(t, err)
		assert.Equal(t, "k-123", result.ID)
		assert.Equal(t, "c-456", result.ChunkID)
		assert.Equal(t, 1, result.ChunkIndex)
		assert.Equal(t, 3, result.ChunkCount)
	})
}

func TestVFSService_Open_Asset(t *testing.T) {
	t.Run("opens asset and returns metadata only", func(t *testing.T) {
		knowledgeRepo := new(MockVFSKnowledgeRepo)
		chunkRepo := new(MockVFSChunkRepo)
		assetRepo := new(MockVFSAssetRepo)
		storage := new(MockVFSStorage)
		listRepo := new(MockVFSListRepo)

		svc := NewVFSService(knowledgeRepo, chunkRepo, assetRepo, storage, listRepo)

		asset := &domain.Asset{
			ID:          "a-789",
			Filename:    "report.pdf",
			MimeType:    "application/pdf",
			Description: "Q4 financial report",
			Keywords:    []string{"finance", "quarterly"},
			StorageKey:  "assets/a-789/report.pdf",
			CreatedAt:   time.Now(),
		}

		assetRepo.On("GetByID", mock.Anything, "a-789").Return(asset, nil)

		result, err := svc.Open(context.Background(), OpenInput{
			ID:         "a-789",
			SourceType: "asset",
		})

		require.NoError(t, err)
		assert.Equal(t, "a-789", result.ID)
		assert.Equal(t, "asset", result.SourceType)
		assert.Equal(t, "report.pdf", result.Title)
		assert.Equal(t, "report.pdf", result.Filename)
		assert.Equal(t, "application/pdf", result.MimeType)
		assert.Equal(t, "Q4 financial report", result.Description)
		assert.Equal(t, []string{"finance", "quarterly"}, result.Keywords)
		assert.Empty(t, result.DownloadURL) // Not requested
		assert.Empty(t, result.Content)     // Assets don't have inline content
	})

	t.Run("opens asset with download URL when requested", func(t *testing.T) {
		knowledgeRepo := new(MockVFSKnowledgeRepo)
		chunkRepo := new(MockVFSChunkRepo)
		assetRepo := new(MockVFSAssetRepo)
		storage := new(MockVFSStorage)
		listRepo := new(MockVFSListRepo)

		svc := NewVFSService(knowledgeRepo, chunkRepo, assetRepo, storage, listRepo)

		asset := &domain.Asset{
			ID:         "a-789",
			Filename:   "report.pdf",
			MimeType:   "application/pdf",
			StorageKey: "assets/a-789/report.pdf",
			CreatedAt:  time.Now(),
		}

		assetRepo.On("GetByID", mock.Anything, "a-789").Return(asset, nil)
		storage.On("GenerateDownloadURL", mock.Anything, "assets/a-789/report.pdf").Return("https://s3.example.com/presigned-url", nil)

		result, err := svc.Open(context.Background(), OpenInput{
			ID:         "a-789",
			SourceType: "asset",
			IncludeURL: true,
		})

		require.NoError(t, err)
		assert.Equal(t, "https://s3.example.com/presigned-url", result.DownloadURL)

		storage.AssertExpectations(t)
	})

	t.Run("handles nil storage gracefully", func(t *testing.T) {
		knowledgeRepo := new(MockVFSKnowledgeRepo)
		chunkRepo := new(MockVFSChunkRepo)
		assetRepo := new(MockVFSAssetRepo)
		listRepo := new(MockVFSListRepo)

		// No storage client
		svc := NewVFSService(knowledgeRepo, chunkRepo, assetRepo, nil, listRepo)

		asset := &domain.Asset{
			ID:         "a-789",
			Filename:   "report.pdf",
			MimeType:   "application/pdf",
			StorageKey: "assets/a-789/report.pdf",
			CreatedAt:  time.Now(),
		}

		assetRepo.On("GetByID", mock.Anything, "a-789").Return(asset, nil)

		result, err := svc.Open(context.Background(), OpenInput{
			ID:         "a-789",
			SourceType: "asset",
			IncludeURL: true, // Requested but storage is nil
		})

		require.NoError(t, err)
		assert.Empty(t, result.DownloadURL) // Should be empty when storage is nil
	})
}

func TestVFSService_List(t *testing.T) {
	t.Run("lists knowledge items only", func(t *testing.T) {
		knowledgeRepo := new(MockVFSKnowledgeRepo)
		chunkRepo := new(MockVFSChunkRepo)
		assetRepo := new(MockVFSAssetRepo)
		storage := new(MockVFSStorage)
		listRepo := new(MockVFSListRepo)

		svc := NewVFSService(knowledgeRepo, chunkRepo, assetRepo, storage, listRepo)

		items := []*ListItem{
			{
				ID:         "k-1",
				Title:      "Knowledge 1",
				Scope:      "/docs",
				Type:       domain.KnowledgeTypeGuideline,
				Status:     domain.KnowledgeStatusApproved,
				SourceType: "knowledge",
				ChunkCount: 3,
				UpdatedAt:  time.Now(),
			},
			{
				ID:         "k-2",
				Title:      "Knowledge 2",
				Scope:      "/docs/api",
				Type:       domain.KnowledgeTypeLearning,
				Status:     domain.KnowledgeStatusApproved,
				SourceType: "knowledge",
				ChunkCount: 5,
				UpdatedAt:  time.Now(),
			},
		}

		listRepo.On("ListKnowledge", mock.Anything, mock.MatchedBy(func(input ListInput) bool {
			return input.SourceType == "knowledge" && input.OrgID == "org-1"
		})).Return(items, nil)

		result, err := svc.List(context.Background(), ListInput{
			OrgID:      "org-1",
			SourceType: "knowledge",
			Limit:      50,
		})

		require.NoError(t, err)
		assert.Len(t, result.Items, 2)
		assert.Equal(t, "k-1", result.Items[0].ID)
		assert.Equal(t, 3, result.Items[0].ChunkCount)
		assert.False(t, result.HasMore)
	})

	t.Run("lists assets only", func(t *testing.T) {
		knowledgeRepo := new(MockVFSKnowledgeRepo)
		chunkRepo := new(MockVFSChunkRepo)
		assetRepo := new(MockVFSAssetRepo)
		storage := new(MockVFSStorage)
		listRepo := new(MockVFSListRepo)

		svc := NewVFSService(knowledgeRepo, chunkRepo, assetRepo, storage, listRepo)

		items := []*ListItem{
			{
				ID:         "a-1",
				Title:      "report.pdf",
				SourceType: "asset",
				Filename:   "report.pdf",
				MimeType:   "application/pdf",
				UpdatedAt:  time.Now(),
			},
		}

		listRepo.On("ListAssets", mock.Anything, mock.MatchedBy(func(input ListInput) bool {
			return input.SourceType == "asset"
		})).Return(items, nil)

		result, err := svc.List(context.Background(), ListInput{
			OrgID:      "org-1",
			SourceType: "asset",
			Limit:      50,
		})

		require.NoError(t, err)
		assert.Len(t, result.Items, 1)
		assert.Equal(t, "a-1", result.Items[0].ID)
		assert.Equal(t, "asset", result.Items[0].SourceType)
	})

	t.Run("lists both knowledge and assets when source_type is empty", func(t *testing.T) {
		knowledgeRepo := new(MockVFSKnowledgeRepo)
		chunkRepo := new(MockVFSChunkRepo)
		assetRepo := new(MockVFSAssetRepo)
		storage := new(MockVFSStorage)
		listRepo := new(MockVFSListRepo)

		svc := NewVFSService(knowledgeRepo, chunkRepo, assetRepo, storage, listRepo)

		knowledgeItems := []*ListItem{
			{ID: "k-1", Title: "Knowledge", SourceType: "knowledge", UpdatedAt: time.Now()},
		}
		assetItems := []*ListItem{
			{ID: "a-1", Title: "Asset", SourceType: "asset", UpdatedAt: time.Now()},
		}

		listRepo.On("ListKnowledge", mock.Anything, mock.Anything).Return(knowledgeItems, nil)
		listRepo.On("ListAssets", mock.Anything, mock.Anything).Return(assetItems, nil)

		result, err := svc.List(context.Background(), ListInput{
			OrgID:      "org-1",
			SourceType: "", // Empty means both
			Limit:      50,
		})

		require.NoError(t, err)
		assert.Len(t, result.Items, 2)
	})

	t.Run("applies pagination", func(t *testing.T) {
		knowledgeRepo := new(MockVFSKnowledgeRepo)
		chunkRepo := new(MockVFSChunkRepo)
		assetRepo := new(MockVFSAssetRepo)
		storage := new(MockVFSStorage)
		listRepo := new(MockVFSListRepo)

		svc := NewVFSService(knowledgeRepo, chunkRepo, assetRepo, storage, listRepo)

		// Return more items than the limit
		items := []*ListItem{
			{ID: "k-1", Title: "Knowledge 1", SourceType: "knowledge", UpdatedAt: time.Now()},
			{ID: "k-2", Title: "Knowledge 2", SourceType: "knowledge", UpdatedAt: time.Now()},
			{ID: "k-3", Title: "Knowledge 3", SourceType: "knowledge", UpdatedAt: time.Now()},
		}

		listRepo.On("ListKnowledge", mock.Anything, mock.Anything).Return(items, nil)

		result, err := svc.List(context.Background(), ListInput{
			OrgID:      "org-1",
			SourceType: "knowledge",
			Limit:      2,
		})

		require.NoError(t, err)
		assert.Len(t, result.Items, 2)
		assert.True(t, result.HasMore)
		assert.NotEmpty(t, result.Cursor)
	})

	t.Run("filters by path prefix", func(t *testing.T) {
		knowledgeRepo := new(MockVFSKnowledgeRepo)
		chunkRepo := new(MockVFSChunkRepo)
		assetRepo := new(MockVFSAssetRepo)
		storage := new(MockVFSStorage)
		listRepo := new(MockVFSListRepo)

		svc := NewVFSService(knowledgeRepo, chunkRepo, assetRepo, storage, listRepo)

		listRepo.On("ListKnowledge", mock.Anything, mock.MatchedBy(func(input ListInput) bool {
			return input.PathPrefix == "/docs/api"
		})).Return([]*ListItem{}, nil)

		_, err := svc.List(context.Background(), ListInput{
			OrgID:      "org-1",
			SourceType: "knowledge",
			PathPrefix: "/docs/api",
			Limit:      50,
		})

		require.NoError(t, err)
		listRepo.AssertExpectations(t)
	})

	t.Run("returns error on repository failure", func(t *testing.T) {
		knowledgeRepo := new(MockVFSKnowledgeRepo)
		chunkRepo := new(MockVFSChunkRepo)
		assetRepo := new(MockVFSAssetRepo)
		storage := new(MockVFSStorage)
		listRepo := new(MockVFSListRepo)

		svc := NewVFSService(knowledgeRepo, chunkRepo, assetRepo, storage, listRepo)

		listRepo.On("ListKnowledge", mock.Anything, mock.Anything).Return(nil, errors.New("database error"))

		_, err := svc.List(context.Background(), ListInput{
			OrgID:      "org-1",
			SourceType: "knowledge",
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "database error")
	})
}

func TestApplyRange(t *testing.T) {
	content := "Line 0\nLine 1\nLine 2\nLine 3\nLine 4"

	t.Run("extracts line range", func(t *testing.T) {
		result := applyRange(content, &ContentRange{StartLine: 1, EndLine: 3})
		assert.Equal(t, "Line 1\nLine 2", result)
	})

	t.Run("handles start line at beginning", func(t *testing.T) {
		result := applyRange(content, &ContentRange{StartLine: 0, EndLine: 2})
		assert.Equal(t, "Line 0\nLine 1", result)
	})

	t.Run("handles end line past content", func(t *testing.T) {
		result := applyRange(content, &ContentRange{StartLine: 3, EndLine: 100})
		assert.Equal(t, "Line 3\nLine 4", result)
	})

	t.Run("returns empty for start past content", func(t *testing.T) {
		result := applyRange(content, &ContentRange{StartLine: 100, EndLine: 200})
		assert.Empty(t, result)
	})

	t.Run("returns empty for invalid range", func(t *testing.T) {
		result := applyRange(content, &ContentRange{StartLine: 3, EndLine: 1})
		assert.Empty(t, result)
	})

	t.Run("truncates at max chars", func(t *testing.T) {
		result := applyRange(content, &ContentRange{MaxChars: 10})
		assert.Equal(t, "Line 0\nLin", result)
		assert.Equal(t, 10, len(result))
	})

	t.Run("uses default max chars", func(t *testing.T) {
		longContent := ""
		for i := 0; i < 1000; i++ {
			longContent += "This is line " + string(rune('0'+i%10)) + "\n"
		}
		result := applyRange(longContent, &ContentRange{})
		assert.LessOrEqual(t, len(result), defaultMaxChars)
	})

	t.Run("caps max chars at maximum", func(t *testing.T) {
		result := applyRange(content, &ContentRange{MaxChars: 100000})
		// Should use content length since it's smaller than maxMaxChars
		assert.Equal(t, content, result)
	})

	t.Run("handles empty content", func(t *testing.T) {
		result := applyRange("", &ContentRange{StartLine: 0, EndLine: 10})
		assert.Empty(t, result)
	})
}

func TestCountLines(t *testing.T) {
	t.Run("counts single line", func(t *testing.T) {
		assert.Equal(t, 1, countLines("single line"))
	})

	t.Run("counts multiple lines", func(t *testing.T) {
		assert.Equal(t, 3, countLines("line 1\nline 2\nline 3"))
	})

	t.Run("counts empty string as zero", func(t *testing.T) {
		assert.Equal(t, 0, countLines(""))
	})

	t.Run("counts trailing newline", func(t *testing.T) {
		assert.Equal(t, 2, countLines("line 1\n"))
	})
}
