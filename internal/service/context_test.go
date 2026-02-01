package service

import (
	"context"
	"errors"
	"testing"

	"github.com/cloo-solutions/neotexai/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockContextRepository is a mock implementation of ContextRepositoryInterface
type MockContextRepository struct {
	mock.Mock
}

func (m *MockContextRepository) GetManifest(ctx context.Context, orgID, projectID string) ([]*KnowledgeManifestItem, error) {
	args := m.Called(ctx, orgID, projectID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*KnowledgeManifestItem), args.Error(1)
}

func (m *MockContextRepository) SearchByEmbedding(ctx context.Context, embedding []float32, filters SearchFilters, limit int) ([]*SearchResult, error) {
	args := m.Called(ctx, embedding, filters, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*SearchResult), args.Error(1)
}

func (m *MockContextRepository) GetByIDs(ctx context.Context, ids []string) ([]*domain.Knowledge, error) {
	args := m.Called(ctx, ids)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Knowledge), args.Error(1)
}

// MockEmbeddingService is a mock implementation of EmbeddingServiceInterface
type MockEmbeddingService struct {
	mock.Mock
}

func (m *MockEmbeddingService) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	args := m.Called(ctx, text)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]float32), args.Error(1)
}

func newContextServiceWithAgenticDisabled(repo ContextRepositoryInterface, embedding EmbeddingServiceInterface) *ContextService {
	cfg := DefaultContextServiceConfig()
	cfg.AgenticSearch.Enabled = false
	return NewContextServiceWithConfig(repo, embedding, cfg)
}

// TestContextService_GetManifest tests the GetManifest method
func TestContextService_GetManifest(t *testing.T) {
	ctx := context.Background()

	t.Run("returns manifest for org and project", func(t *testing.T) {
		mockRepo := new(MockContextRepository)
		mockEmbedding := new(MockEmbeddingService)
		service := newContextServiceWithAgenticDisabled(mockRepo, mockEmbedding)

		expectedManifest := []*KnowledgeManifestItem{
			{ID: "k1", Title: "Guidelines", Summary: "Coding guidelines", Type: domain.KnowledgeTypeGuideline, Scope: "/src"},
			{ID: "k2", Title: "Learnings", Summary: "Team learnings", Type: domain.KnowledgeTypeLearning, Scope: ""},
		}

		mockRepo.On("GetManifest", mock.Anything, "org-1", "project-1").Return(expectedManifest, nil)

		result, err := service.GetManifest(ctx, "org-1", "project-1")

		require.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, "k1", result[0].ID)
		assert.Equal(t, "Guidelines", result[0].Title)
		assert.Equal(t, domain.KnowledgeTypeGuideline, result[0].Type)
		mockRepo.AssertExpectations(t)
	})

	t.Run("returns manifest for org only (empty projectID)", func(t *testing.T) {
		mockRepo := new(MockContextRepository)
		mockEmbedding := new(MockEmbeddingService)
		service := newContextServiceWithAgenticDisabled(mockRepo, mockEmbedding)

		expectedManifest := []*KnowledgeManifestItem{
			{ID: "k1", Title: "Org Guidelines", Summary: "Org-wide guidelines", Type: domain.KnowledgeTypeGuideline, Scope: ""},
		}

		mockRepo.On("GetManifest", mock.Anything, "org-1", "").Return(expectedManifest, nil)

		result, err := service.GetManifest(ctx, "org-1", "")

		require.NoError(t, err)
		assert.Len(t, result, 1)
		mockRepo.AssertExpectations(t)
	})

	t.Run("returns empty manifest when no knowledge exists", func(t *testing.T) {
		mockRepo := new(MockContextRepository)
		mockEmbedding := new(MockEmbeddingService)
		service := newContextServiceWithAgenticDisabled(mockRepo, mockEmbedding)

		mockRepo.On("GetManifest", mock.Anything, "org-empty", "").Return([]*KnowledgeManifestItem{}, nil)

		result, err := service.GetManifest(ctx, "org-empty", "")

		require.NoError(t, err)
		assert.Empty(t, result)
		mockRepo.AssertExpectations(t)
	})

	t.Run("returns error on repository failure", func(t *testing.T) {
		mockRepo := new(MockContextRepository)
		mockEmbedding := new(MockEmbeddingService)
		service := newContextServiceWithAgenticDisabled(mockRepo, mockEmbedding)

		expectedErr := errors.New("database error")
		mockRepo.On("GetManifest", mock.Anything, "org-1", "").Return(nil, expectedErr)

		result, err := service.GetManifest(ctx, "org-1", "")

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, expectedErr, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("agentic fallback expands query when results are sparse", func(t *testing.T) {
		mockRepo := new(MockContextRepository)
		mockEmbedding := new(MockEmbeddingService)
		cfg := DefaultContextServiceConfig()
		cfg.AgenticSearch.Enabled = true
		cfg.AgenticSearch.MinResults = 2
		cfg.AgenticSearch.MaxIterations = 1
		cfg.AgenticSearch.MaxVariants = 1
		service := NewContextServiceWithConfig(mockRepo, mockEmbedding, cfg)

		queryEmbedding := make([]float32, 1536)
		expandedEmbedding := make([]float32, 1536)
		expandedEmbedding[0] = 0.2

		filters := SearchFilters{OrgID: "org-1"}
		initialResults := []*SearchResult{
			{ID: "k1", Title: "Auth Basics", Score: 0.5},
		}
		expandedResults := []*SearchResult{
			{ID: "k2", Title: "Token Handling", Score: 0.9},
		}

		mockEmbedding.On("GenerateEmbedding", mock.Anything, "auth and tokens").Return(queryEmbedding, nil)
		mockRepo.On("SearchByEmbedding", mock.Anything, queryEmbedding, filters, 6).Return(initialResults, nil)
		mockEmbedding.On("GenerateEmbedding", mock.Anything, "auth").Return(expandedEmbedding, nil)
		mockRepo.On("SearchByEmbedding", mock.Anything, expandedEmbedding, filters, 6).Return(expandedResults, nil)

		input := SearchInput{
			Query:   "auth and tokens",
			Filters: filters,
			Limit:   5,
		}
		result, err := service.Search(ctx, input)

		require.NoError(t, err)
		require.Len(t, result.Results, 2)
		assert.Equal(t, "k2", result.Results[0].ID)
		assert.Equal(t, "k1", result.Results[1].ID)
		mockEmbedding.AssertExpectations(t)
		mockRepo.AssertExpectations(t)
	})
}

// TestContextService_Search tests the Search method
func TestContextService_Search(t *testing.T) {
	ctx := context.Background()

	t.Run("searches with query embedding and filters", func(t *testing.T) {
		mockRepo := new(MockContextRepository)
		mockEmbedding := new(MockEmbeddingService)
		service := newContextServiceWithAgenticDisabled(mockRepo, mockEmbedding)

		queryEmbedding := make([]float32, 1536)
		queryEmbedding[0] = 0.1

		filters := SearchFilters{
			OrgID:     "org-1",
			ProjectID: "project-1",
			Type:      domain.KnowledgeTypeGuideline,
			Status:    domain.KnowledgeStatusApproved,
		}

		expectedResults := []*SearchResult{
			{ID: "k1", Title: "Guidelines", Summary: "Coding guidelines", Score: 0.95},
			{ID: "k2", Title: "More Guidelines", Summary: "More guidelines", Score: 0.85},
		}

		mockEmbedding.On("GenerateEmbedding", mock.Anything, "how to code").Return(queryEmbedding, nil)
		mockRepo.On("SearchByEmbedding", mock.Anything, queryEmbedding, filters, 11).Return(expectedResults, nil)

		input := SearchInput{
			Query:   "how to code",
			Filters: filters,
			Limit:   10,
		}
		result, err := service.Search(ctx, input)

		require.NoError(t, err)
		assert.Len(t, result.Results, 2)
		assert.Equal(t, "k1", result.Results[0].ID)
		assert.Equal(t, float32(0.95), result.Results[0].Score)
		mockEmbedding.AssertExpectations(t)
		mockRepo.AssertExpectations(t)
	})

	t.Run("uses default limit when not specified", func(t *testing.T) {
		mockRepo := new(MockContextRepository)
		mockEmbedding := new(MockEmbeddingService)
		service := newContextServiceWithAgenticDisabled(mockRepo, mockEmbedding)

		queryEmbedding := make([]float32, 1536)
		filters := SearchFilters{OrgID: "org-1"}

		mockEmbedding.On("GenerateEmbedding", mock.Anything, "test").Return(queryEmbedding, nil)
		mockRepo.On("SearchByEmbedding", mock.Anything, queryEmbedding, filters, 21).Return([]*SearchResult{}, nil)

		input := SearchInput{
			Query:   "test",
			Filters: filters,
			Limit:   0, // Should use default
		}
		result, err := service.Search(ctx, input)

		require.NoError(t, err)
		assert.Empty(t, result.Results)
		mockRepo.AssertExpectations(t)
	})

	t.Run("filters by path prefix", func(t *testing.T) {
		mockRepo := new(MockContextRepository)
		mockEmbedding := new(MockEmbeddingService)
		service := newContextServiceWithAgenticDisabled(mockRepo, mockEmbedding)

		queryEmbedding := make([]float32, 1536)
		filters := SearchFilters{
			OrgID:      "org-1",
			PathPrefix: "/src/api",
		}

		expectedResults := []*SearchResult{
			{ID: "k1", Title: "API Guidelines", Summary: "API specific guidelines", Scope: "/src/api", Score: 0.9},
		}

		mockEmbedding.On("GenerateEmbedding", mock.Anything, "api design").Return(queryEmbedding, nil)
		mockRepo.On("SearchByEmbedding", mock.Anything, queryEmbedding, filters, 21).Return(expectedResults, nil)

		input := SearchInput{
			Query:   "api design",
			Filters: filters,
		}
		result, err := service.Search(ctx, input)

		require.NoError(t, err)
		assert.Len(t, result.Results, 1)
		assert.Equal(t, "/src/api", result.Results[0].Scope)
		mockRepo.AssertExpectations(t)
	})

	t.Run("returns error on embedding generation failure", func(t *testing.T) {
		mockRepo := new(MockContextRepository)
		mockEmbedding := new(MockEmbeddingService)
		service := newContextServiceWithAgenticDisabled(mockRepo, mockEmbedding)

		expectedErr := errors.New("embedding service error")
		mockEmbedding.On("GenerateEmbedding", mock.Anything, "test").Return(nil, expectedErr)

		input := SearchInput{
			Query:   "test",
			Filters: SearchFilters{OrgID: "org-1"},
		}
		result, err := service.Search(ctx, input)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, expectedErr, err)
		mockEmbedding.AssertExpectations(t)
	})

	t.Run("returns error on repository failure", func(t *testing.T) {
		mockRepo := new(MockContextRepository)
		mockEmbedding := new(MockEmbeddingService)
		service := newContextServiceWithAgenticDisabled(mockRepo, mockEmbedding)

		queryEmbedding := make([]float32, 1536)
		filters := SearchFilters{OrgID: "org-1"}
		expectedErr := errors.New("database error")

		mockEmbedding.On("GenerateEmbedding", mock.Anything, "test").Return(queryEmbedding, nil)
		mockRepo.On("SearchByEmbedding", mock.Anything, queryEmbedding, filters, 21).Return(nil, expectedErr)

		input := SearchInput{
			Query:   "test",
			Filters: filters,
		}
		result, err := service.Search(ctx, input)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, expectedErr, err)
		mockRepo.AssertExpectations(t)
	})
}

// TestContextService_GetRelevantKnowledge tests the GetRelevantKnowledge method
func TestContextService_GetRelevantKnowledge(t *testing.T) {
	ctx := context.Background()

	t.Run("returns up to 3 items ranked by relevance", func(t *testing.T) {
		mockRepo := new(MockContextRepository)
		mockEmbedding := new(MockEmbeddingService)
		service := newContextServiceWithAgenticDisabled(mockRepo, mockEmbedding)

		queryEmbedding := make([]float32, 1536)
		input := RelevantKnowledgeInput{
			OrgID:     "org-1",
			ProjectID: "project-1",
			FilePath:  "/src/api/handler.go",
			Query:     "error handling",
		}

		searchResults := []*SearchResult{
			{ID: "k1", Title: "API Error Handling", Summary: "How to handle errors in API", Scope: "/src/api/handler.go", Score: 0.9},
			{ID: "k2", Title: "General Error Guidelines", Summary: "General guidelines", Scope: "/src/api", Score: 0.85},
			{ID: "k3", Title: "Logging Practices", Summary: "How to log", Scope: "/src", Score: 0.8},
			{ID: "k4", Title: "Database Guidelines", Summary: "DB patterns", Scope: "/src/db", Score: 0.7},
		}

		expectedKnowledge := []*domain.Knowledge{
			{ID: "k1", Title: "API Error Handling", Scope: "/src/api/handler.go"},
			{ID: "k2", Title: "General Error Guidelines", Scope: "/src/api"},
			{ID: "k3", Title: "Logging Practices", Scope: "/src"},
		}

		mockEmbedding.On("GenerateEmbedding", mock.Anything, "error handling").Return(queryEmbedding, nil)
		mockRepo.On("SearchByEmbedding", mock.Anything, queryEmbedding, mock.MatchedBy(func(f SearchFilters) bool {
			return f.OrgID == "org-1" && f.ProjectID == "project-1"
		}), 10).Return(searchResults, nil)
		mockRepo.On("GetByIDs", mock.Anything, []string{"k1", "k2", "k3"}).Return(expectedKnowledge, nil)

		result, err := service.GetRelevantKnowledge(ctx, input)

		require.NoError(t, err)
		assert.Len(t, result, 3)
		assert.Equal(t, "k1", result[0].ID) // Exact file match
		mockRepo.AssertExpectations(t)
		mockEmbedding.AssertExpectations(t)
	})

	t.Run("ranks exact file match highest", func(t *testing.T) {
		mockRepo := new(MockContextRepository)
		mockEmbedding := new(MockEmbeddingService)
		service := newContextServiceWithAgenticDisabled(mockRepo, mockEmbedding)

		queryEmbedding := make([]float32, 1536)
		input := RelevantKnowledgeInput{
			OrgID:     "org-1",
			ProjectID: "project-1",
			FilePath:  "/src/api/handler.go",
			Query:     "handler patterns",
		}

		// Semantic match is first, but file match should be promoted
		searchResults := []*SearchResult{
			{ID: "k1", Title: "General Patterns", Summary: "Patterns", Scope: "", Score: 0.95},
			{ID: "k2", Title: "Handler Guidelines", Summary: "Handler specific", Scope: "/src/api/handler.go", Score: 0.7},
			{ID: "k3", Title: "API Guidelines", Summary: "API wide", Scope: "/src/api", Score: 0.8},
		}

		expectedKnowledge := []*domain.Knowledge{
			{ID: "k2", Title: "Handler Guidelines", Scope: "/src/api/handler.go"},
			{ID: "k3", Title: "API Guidelines", Scope: "/src/api"},
			{ID: "k1", Title: "General Patterns", Scope: ""},
		}

		mockEmbedding.On("GenerateEmbedding", mock.Anything, "handler patterns").Return(queryEmbedding, nil)
		mockRepo.On("SearchByEmbedding", mock.Anything, queryEmbedding, mock.Anything, 10).Return(searchResults, nil)
		mockRepo.On("GetByIDs", mock.Anything, []string{"k2", "k3", "k1"}).Return(expectedKnowledge, nil)

		result, err := service.GetRelevantKnowledge(ctx, input)

		require.NoError(t, err)
		assert.Len(t, result, 3)
		// File match should be first
		assert.Equal(t, "k2", result[0].ID)
		// Path match should be second
		assert.Equal(t, "k3", result[1].ID)
		// Semantic match should be third
		assert.Equal(t, "k1", result[2].ID)
		mockRepo.AssertExpectations(t)
	})

	t.Run("ranks path prefix match over semantic", func(t *testing.T) {
		mockRepo := new(MockContextRepository)
		mockEmbedding := new(MockEmbeddingService)
		service := newContextServiceWithAgenticDisabled(mockRepo, mockEmbedding)

		queryEmbedding := make([]float32, 1536)
		input := RelevantKnowledgeInput{
			OrgID:     "org-1",
			ProjectID: "project-1",
			FilePath:  "/src/api/v2/users.go",
			Query:     "user management",
		}

		searchResults := []*SearchResult{
			{ID: "k1", Title: "General Users", Summary: "User management", Scope: "", Score: 0.95},
			{ID: "k2", Title: "API Users", Summary: "API user handling", Scope: "/src/api", Score: 0.8},
		}

		expectedKnowledge := []*domain.Knowledge{
			{ID: "k2", Title: "API Users", Scope: "/src/api"},
			{ID: "k1", Title: "General Users", Scope: ""},
		}

		mockEmbedding.On("GenerateEmbedding", mock.Anything, "user management").Return(queryEmbedding, nil)
		mockRepo.On("SearchByEmbedding", mock.Anything, queryEmbedding, mock.Anything, 10).Return(searchResults, nil)
		mockRepo.On("GetByIDs", mock.Anything, []string{"k2", "k1"}).Return(expectedKnowledge, nil)

		result, err := service.GetRelevantKnowledge(ctx, input)

		require.NoError(t, err)
		assert.Len(t, result, 2)
		// Path prefix match should come first
		assert.Equal(t, "k2", result[0].ID)
		assert.Equal(t, "k1", result[1].ID)
		mockRepo.AssertExpectations(t)
	})

	t.Run("returns fewer than 3 if not enough results", func(t *testing.T) {
		mockRepo := new(MockContextRepository)
		mockEmbedding := new(MockEmbeddingService)
		service := newContextServiceWithAgenticDisabled(mockRepo, mockEmbedding)

		queryEmbedding := make([]float32, 1536)
		input := RelevantKnowledgeInput{
			OrgID:     "org-1",
			ProjectID: "project-1",
			FilePath:  "/src/main.go",
			Query:     "main function",
		}

		searchResults := []*SearchResult{
			{ID: "k1", Title: "Main Guidelines", Summary: "Main file", Score: 0.9},
		}

		expectedKnowledge := []*domain.Knowledge{
			{ID: "k1", Title: "Main Guidelines"},
		}

		mockEmbedding.On("GenerateEmbedding", mock.Anything, "main function").Return(queryEmbedding, nil)
		mockRepo.On("SearchByEmbedding", mock.Anything, queryEmbedding, mock.Anything, 10).Return(searchResults, nil)
		mockRepo.On("GetByIDs", mock.Anything, []string{"k1"}).Return(expectedKnowledge, nil)

		result, err := service.GetRelevantKnowledge(ctx, input)

		require.NoError(t, err)
		assert.Len(t, result, 1)
		mockRepo.AssertExpectations(t)
	})

	t.Run("returns empty when no results", func(t *testing.T) {
		mockRepo := new(MockContextRepository)
		mockEmbedding := new(MockEmbeddingService)
		service := newContextServiceWithAgenticDisabled(mockRepo, mockEmbedding)

		queryEmbedding := make([]float32, 1536)
		input := RelevantKnowledgeInput{
			OrgID:     "org-1",
			ProjectID: "project-1",
			FilePath:  "/src/main.go",
			Query:     "something rare",
		}

		mockEmbedding.On("GenerateEmbedding", mock.Anything, "something rare").Return(queryEmbedding, nil)
		mockRepo.On("SearchByEmbedding", mock.Anything, queryEmbedding, mock.Anything, 10).Return([]*SearchResult{}, nil)

		result, err := service.GetRelevantKnowledge(ctx, input)

		require.NoError(t, err)
		assert.Empty(t, result)
		mockRepo.AssertExpectations(t)
	})

	t.Run("returns error on embedding generation failure", func(t *testing.T) {
		mockRepo := new(MockContextRepository)
		mockEmbedding := new(MockEmbeddingService)
		service := newContextServiceWithAgenticDisabled(mockRepo, mockEmbedding)

		expectedErr := errors.New("embedding error")
		input := RelevantKnowledgeInput{
			OrgID:     "org-1",
			ProjectID: "project-1",
			FilePath:  "/src/main.go",
			Query:     "test",
		}

		mockEmbedding.On("GenerateEmbedding", mock.Anything, "test").Return(nil, expectedErr)

		result, err := service.GetRelevantKnowledge(ctx, input)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, expectedErr, err)
		mockEmbedding.AssertExpectations(t)
	})

	t.Run("returns error on search failure", func(t *testing.T) {
		mockRepo := new(MockContextRepository)
		mockEmbedding := new(MockEmbeddingService)
		service := newContextServiceWithAgenticDisabled(mockRepo, mockEmbedding)

		queryEmbedding := make([]float32, 1536)
		expectedErr := errors.New("search error")
		input := RelevantKnowledgeInput{
			OrgID:     "org-1",
			ProjectID: "project-1",
			FilePath:  "/src/main.go",
			Query:     "test",
		}

		mockEmbedding.On("GenerateEmbedding", mock.Anything, "test").Return(queryEmbedding, nil)
		mockRepo.On("SearchByEmbedding", mock.Anything, queryEmbedding, mock.Anything, 10).Return(nil, expectedErr)

		result, err := service.GetRelevantKnowledge(ctx, input)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, expectedErr, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("returns error on GetByIDs failure", func(t *testing.T) {
		mockRepo := new(MockContextRepository)
		mockEmbedding := new(MockEmbeddingService)
		service := newContextServiceWithAgenticDisabled(mockRepo, mockEmbedding)

		queryEmbedding := make([]float32, 1536)
		expectedErr := errors.New("database error")
		input := RelevantKnowledgeInput{
			OrgID:     "org-1",
			ProjectID: "project-1",
			FilePath:  "/src/main.go",
			Query:     "test",
		}

		searchResults := []*SearchResult{
			{ID: "k1", Title: "Test", Score: 0.9},
		}

		mockEmbedding.On("GenerateEmbedding", mock.Anything, "test").Return(queryEmbedding, nil)
		mockRepo.On("SearchByEmbedding", mock.Anything, queryEmbedding, mock.Anything, 10).Return(searchResults, nil)
		mockRepo.On("GetByIDs", mock.Anything, []string{"k1"}).Return(nil, expectedErr)

		result, err := service.GetRelevantKnowledge(ctx, input)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, expectedErr, err)
		mockRepo.AssertExpectations(t)
	})
}

// TestContextService_GetRelevantKnowledge_RelevanceRanking tests relevance ranking in detail
func TestContextService_GetRelevantKnowledge_RelevanceRanking(t *testing.T) {
	ctx := context.Background()

	t.Run("file match > path match > semantic match", func(t *testing.T) {
		mockRepo := new(MockContextRepository)
		mockEmbedding := new(MockEmbeddingService)
		service := newContextServiceWithAgenticDisabled(mockRepo, mockEmbedding)

		queryEmbedding := make([]float32, 1536)
		input := RelevantKnowledgeInput{
			OrgID:     "org-1",
			ProjectID: "project-1",
			FilePath:  "/src/api/users/handler.go",
			Query:     "user handler",
		}

		// All have same semantic score, but different scope relevance
		searchResults := []*SearchResult{
			{ID: "semantic", Title: "Semantic Match", Scope: "/src/db", Score: 0.9},           // No path match
			{ID: "path", Title: "Path Match", Scope: "/src/api/users", Score: 0.9},            // Path prefix match
			{ID: "file", Title: "File Match", Scope: "/src/api/users/handler.go", Score: 0.9}, // Exact file match
			{ID: "root", Title: "Root Match", Scope: "/src", Score: 0.9},                      // Partial path match
		}

		expectedOrder := []string{"file", "path", "root"}

		expectedKnowledge := []*domain.Knowledge{
			{ID: "file", Title: "File Match", Scope: "/src/api/users/handler.go"},
			{ID: "path", Title: "Path Match", Scope: "/src/api/users"},
			{ID: "root", Title: "Root Match", Scope: "/src"},
		}

		mockEmbedding.On("GenerateEmbedding", mock.Anything, "user handler").Return(queryEmbedding, nil)
		mockRepo.On("SearchByEmbedding", mock.Anything, queryEmbedding, mock.Anything, 10).Return(searchResults, nil)
		mockRepo.On("GetByIDs", mock.Anything, expectedOrder).Return(expectedKnowledge, nil)

		result, err := service.GetRelevantKnowledge(ctx, input)

		require.NoError(t, err)
		assert.Len(t, result, 3)
		assert.Equal(t, "file", result[0].ID)
		assert.Equal(t, "path", result[1].ID)
		assert.Equal(t, "root", result[2].ID)
	})
}
