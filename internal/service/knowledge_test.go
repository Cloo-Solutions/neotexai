package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cloo-solutions/neotexai/internal/domain"
	"github.com/cloo-solutions/neotexai/internal/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockKnowledgeRepository is a mock implementation of KnowledgeRepositoryInterface
type MockKnowledgeRepository struct {
	mock.Mock
}

func (m *MockKnowledgeRepository) Create(ctx context.Context, k *domain.Knowledge) error {
	args := m.Called(ctx, k)
	return args.Error(0)
}

func (m *MockKnowledgeRepository) GetByID(ctx context.Context, id string) (*domain.Knowledge, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Knowledge), args.Error(1)
}

func (m *MockKnowledgeRepository) ListByOrg(ctx context.Context, orgID string) ([]*domain.Knowledge, error) {
	args := m.Called(ctx, orgID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Knowledge), args.Error(1)
}

func (m *MockKnowledgeRepository) ListByProject(ctx context.Context, projectID string) ([]*domain.Knowledge, error) {
	args := m.Called(ctx, projectID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Knowledge), args.Error(1)
}

func (m *MockKnowledgeRepository) Update(ctx context.Context, k *domain.Knowledge) error {
	args := m.Called(ctx, k)
	return args.Error(0)
}

func (m *MockKnowledgeRepository) CreateVersion(ctx context.Context, v *domain.KnowledgeVersion) error {
	args := m.Called(ctx, v)
	return args.Error(0)
}

func (m *MockKnowledgeRepository) GetLatestVersion(ctx context.Context, knowledgeID string) (*domain.KnowledgeVersion, error) {
	args := m.Called(ctx, knowledgeID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.KnowledgeVersion), args.Error(1)
}

func (m *MockKnowledgeRepository) GetVersions(ctx context.Context, knowledgeID string) ([]*domain.KnowledgeVersion, error) {
	args := m.Called(ctx, knowledgeID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.KnowledgeVersion), args.Error(1)
}

func (m *MockKnowledgeRepository) ListByOrgWithCursor(ctx context.Context, orgID string, cursor *pagination.Cursor, limit int) (*KnowledgePageResult, error) {
	args := m.Called(ctx, orgID, cursor, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*KnowledgePageResult), args.Error(1)
}

func (m *MockKnowledgeRepository) ListByProjectWithCursor(ctx context.Context, projectID string, cursor *pagination.Cursor, limit int) (*KnowledgePageResult, error) {
	args := m.Called(ctx, projectID, cursor, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*KnowledgePageResult), args.Error(1)
}

// MockEmbeddingJobRepository is a mock implementation of EmbeddingJobRepositoryInterface
type MockEmbeddingJobRepository struct {
	mock.Mock
}

func (m *MockEmbeddingJobRepository) Create(ctx context.Context, job *domain.EmbeddingJob) error {
	args := m.Called(ctx, job)
	return args.Error(0)
}

// MockUUIDGenerator is a mock implementation of UUIDGenerator
type MockUUIDGenerator struct {
	mock.Mock
	callCount int
	uuids     []string
}

func NewMockUUIDGenerator(uuids ...string) *MockUUIDGenerator {
	return &MockUUIDGenerator{uuids: uuids}
}

func (m *MockUUIDGenerator) NewString() string {
	if m.callCount < len(m.uuids) {
		uuid := m.uuids[m.callCount]
		m.callCount++
		return uuid
	}
	return "default-uuid"
}

// TestKnowledgeService_Create tests the Create method
func TestKnowledgeService_Create(t *testing.T) {
	ctx := context.Background()

	t.Run("creates knowledge with first version and queues embedding job", func(t *testing.T) {
		mockKnowledgeRepo := new(MockKnowledgeRepository)
		mockEmbeddingJobRepo := new(MockEmbeddingJobRepository)
		mockUUIDGen := NewMockUUIDGenerator("knowledge-id-1", "version-id-1", "job-id-1")

		service := NewKnowledgeServiceWithUUIDGen(mockKnowledgeRepo, mockEmbeddingJobRepo, mockUUIDGen)

		input := CreateInput{
			OrgID:     "org-1",
			ProjectID: "project-1",
			Type:      domain.KnowledgeTypeGuideline,
			Title:     "Test Knowledge",
			Summary:   "Test summary",
			BodyMD:    "# Test Body",
			Scope:     "/src/main.go",
		}

		// Setup expectations
		mockKnowledgeRepo.On("Create", mock.Anything, mock.MatchedBy(func(k *domain.Knowledge) bool {
			return k.ID == "knowledge-id-1" &&
				k.OrgID == "org-1" &&
				k.ProjectID == "project-1" &&
				k.Type == domain.KnowledgeTypeGuideline &&
				k.Status == domain.KnowledgeStatusDraft &&
				k.Title == "Test Knowledge" &&
				k.Summary == "Test summary" &&
				k.BodyMD == "# Test Body" &&
				k.Scope == "/src/main.go"
		})).Return(nil)

		mockKnowledgeRepo.On("CreateVersion", mock.Anything, mock.MatchedBy(func(v *domain.KnowledgeVersion) bool {
			return v.ID == "version-id-1" &&
				v.KnowledgeID == "knowledge-id-1" &&
				v.VersionNumber == 1 &&
				v.Title == "Test Knowledge" &&
				v.Summary == "Test summary" &&
				v.BodyMD == "# Test Body"
		})).Return(nil)

		mockEmbeddingJobRepo.On("Create", mock.Anything, mock.MatchedBy(func(job *domain.EmbeddingJob) bool {
			return job.ID == "job-id-1" &&
				job.KnowledgeID == "knowledge-id-1" &&
				job.Status == domain.EmbeddingJobStatusPending &&
				job.Retries == 0
		})).Return(nil)

		// Execute
		result, err := service.Create(ctx, input)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "knowledge-id-1", result.ID)
		assert.Equal(t, "org-1", result.OrgID)
		assert.Equal(t, "project-1", result.ProjectID)
		assert.Equal(t, domain.KnowledgeTypeGuideline, result.Type)
		assert.Equal(t, domain.KnowledgeStatusDraft, result.Status)
		assert.Equal(t, "Test Knowledge", result.Title)

		mockKnowledgeRepo.AssertExpectations(t)
		mockEmbeddingJobRepo.AssertExpectations(t)
	})

	t.Run("returns error on validation failure - missing title", func(t *testing.T) {
		mockKnowledgeRepo := new(MockKnowledgeRepository)
		mockEmbeddingJobRepo := new(MockEmbeddingJobRepository)
		mockUUIDGen := NewMockUUIDGenerator("knowledge-id-1", "version-id-1", "job-id-1")

		service := NewKnowledgeServiceWithUUIDGen(mockKnowledgeRepo, mockEmbeddingJobRepo, mockUUIDGen)

		input := CreateInput{
			OrgID:     "org-1",
			ProjectID: "project-1",
			Type:      domain.KnowledgeTypeGuideline,
			Title:     "", // Empty title - validation should fail
			Summary:   "Test summary",
			BodyMD:    "# Test Body",
		}

		// Execute
		result, err := service.Create(ctx, input)

		// Assert
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "Title")
	})

	t.Run("returns error on validation failure - missing body", func(t *testing.T) {
		mockKnowledgeRepo := new(MockKnowledgeRepository)
		mockEmbeddingJobRepo := new(MockEmbeddingJobRepository)
		mockUUIDGen := NewMockUUIDGenerator("knowledge-id-1", "version-id-1", "job-id-1")

		service := NewKnowledgeServiceWithUUIDGen(mockKnowledgeRepo, mockEmbeddingJobRepo, mockUUIDGen)

		input := CreateInput{
			OrgID:     "org-1",
			ProjectID: "project-1",
			Type:      domain.KnowledgeTypeGuideline,
			Title:     "Test Title",
			Summary:   "Test summary",
			BodyMD:    "", // Empty body - validation should fail
		}

		// Execute
		result, err := service.Create(ctx, input)

		// Assert
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "BodyMD")
	})

	t.Run("returns error on knowledge repository failure", func(t *testing.T) {
		mockKnowledgeRepo := new(MockKnowledgeRepository)
		mockEmbeddingJobRepo := new(MockEmbeddingJobRepository)
		mockUUIDGen := NewMockUUIDGenerator("knowledge-id-1", "version-id-1", "job-id-1")

		service := NewKnowledgeServiceWithUUIDGen(mockKnowledgeRepo, mockEmbeddingJobRepo, mockUUIDGen)

		input := CreateInput{
			OrgID:     "org-1",
			ProjectID: "project-1",
			Type:      domain.KnowledgeTypeGuideline,
			Title:     "Test Knowledge",
			Summary:   "Test summary",
			BodyMD:    "# Test Body",
		}

		expectedErr := errors.New("database error")
		mockKnowledgeRepo.On("Create", mock.Anything, mock.Anything).Return(expectedErr)

		// Execute
		result, err := service.Create(ctx, input)

		// Assert
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, expectedErr, err)
		mockKnowledgeRepo.AssertExpectations(t)
	})

	t.Run("returns error on version creation failure", func(t *testing.T) {
		mockKnowledgeRepo := new(MockKnowledgeRepository)
		mockEmbeddingJobRepo := new(MockEmbeddingJobRepository)
		mockUUIDGen := NewMockUUIDGenerator("knowledge-id-1", "version-id-1", "job-id-1")

		service := NewKnowledgeServiceWithUUIDGen(mockKnowledgeRepo, mockEmbeddingJobRepo, mockUUIDGen)

		input := CreateInput{
			OrgID:     "org-1",
			ProjectID: "project-1",
			Type:      domain.KnowledgeTypeGuideline,
			Title:     "Test Knowledge",
			Summary:   "Test summary",
			BodyMD:    "# Test Body",
		}

		mockKnowledgeRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
		expectedErr := errors.New("version creation error")
		mockKnowledgeRepo.On("CreateVersion", mock.Anything, mock.Anything).Return(expectedErr)

		// Execute
		result, err := service.Create(ctx, input)

		// Assert
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, expectedErr, err)
		mockKnowledgeRepo.AssertExpectations(t)
	})

	t.Run("returns error on embedding job creation failure", func(t *testing.T) {
		mockKnowledgeRepo := new(MockKnowledgeRepository)
		mockEmbeddingJobRepo := new(MockEmbeddingJobRepository)
		mockUUIDGen := NewMockUUIDGenerator("knowledge-id-1", "version-id-1", "job-id-1")

		service := NewKnowledgeServiceWithUUIDGen(mockKnowledgeRepo, mockEmbeddingJobRepo, mockUUIDGen)

		input := CreateInput{
			OrgID:     "org-1",
			ProjectID: "project-1",
			Type:      domain.KnowledgeTypeGuideline,
			Title:     "Test Knowledge",
			Summary:   "Test summary",
			BodyMD:    "# Test Body",
		}

		mockKnowledgeRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
		mockKnowledgeRepo.On("CreateVersion", mock.Anything, mock.Anything).Return(nil)
		expectedErr := errors.New("embedding job creation error")
		mockEmbeddingJobRepo.On("Create", mock.Anything, mock.Anything).Return(expectedErr)

		// Execute
		result, err := service.Create(ctx, input)

		// Assert
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, expectedErr, err)
		mockKnowledgeRepo.AssertExpectations(t)
		mockEmbeddingJobRepo.AssertExpectations(t)
	})
}

// TestKnowledgeService_GetByID tests the GetByID method
func TestKnowledgeService_GetByID(t *testing.T) {
	ctx := context.Background()

	t.Run("returns knowledge when found", func(t *testing.T) {
		mockKnowledgeRepo := new(MockKnowledgeRepository)
		mockEmbeddingJobRepo := new(MockEmbeddingJobRepository)

		service := NewKnowledgeService(mockKnowledgeRepo, mockEmbeddingJobRepo)

		expectedKnowledge := &domain.Knowledge{
			ID:        "knowledge-1",
			OrgID:     "org-1",
			ProjectID: "project-1",
			Type:      domain.KnowledgeTypeGuideline,
			Status:    domain.KnowledgeStatusDraft,
			Title:     "Test Knowledge",
			Summary:   "Test summary",
			BodyMD:    "# Test Body",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		mockKnowledgeRepo.On("GetByID", mock.Anything, "knowledge-1").Return(expectedKnowledge, nil)

		// Execute
		result, err := service.GetByID(ctx, "knowledge-1")

		// Assert
		require.NoError(t, err)
		assert.Equal(t, expectedKnowledge, result)
		mockKnowledgeRepo.AssertExpectations(t)
	})

	t.Run("returns error when not found", func(t *testing.T) {
		mockKnowledgeRepo := new(MockKnowledgeRepository)
		mockEmbeddingJobRepo := new(MockEmbeddingJobRepository)

		service := NewKnowledgeService(mockKnowledgeRepo, mockEmbeddingJobRepo)

		mockKnowledgeRepo.On("GetByID", mock.Anything, "non-existent").Return(nil, domain.ErrKnowledgeNotFound)

		// Execute
		result, err := service.GetByID(ctx, "non-existent")

		// Assert
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, domain.ErrKnowledgeNotFound, err)
		mockKnowledgeRepo.AssertExpectations(t)
	})
}

// TestKnowledgeService_GetLatestVersion tests the GetLatestVersion method
func TestKnowledgeService_GetLatestVersion(t *testing.T) {
	ctx := context.Background()

	t.Run("returns latest version when found", func(t *testing.T) {
		mockKnowledgeRepo := new(MockKnowledgeRepository)
		mockEmbeddingJobRepo := new(MockEmbeddingJobRepository)

		service := NewKnowledgeService(mockKnowledgeRepo, mockEmbeddingJobRepo)

		expectedVersion := &domain.KnowledgeVersion{
			ID:            "version-3",
			KnowledgeID:   "knowledge-1",
			VersionNumber: 3,
			Title:         "Test Knowledge v3",
			Summary:       "Test summary v3",
			BodyMD:        "# Test Body v3",
			CreatedAt:     time.Now(),
		}

		mockKnowledgeRepo.On("GetLatestVersion", mock.Anything, "knowledge-1").Return(expectedVersion, nil)

		// Execute
		result, err := service.GetLatestVersion(ctx, "knowledge-1")

		// Assert
		require.NoError(t, err)
		assert.Equal(t, expectedVersion, result)
		assert.Equal(t, int64(3), result.VersionNumber)
		mockKnowledgeRepo.AssertExpectations(t)
	})

	t.Run("returns error when knowledge not found", func(t *testing.T) {
		mockKnowledgeRepo := new(MockKnowledgeRepository)
		mockEmbeddingJobRepo := new(MockEmbeddingJobRepository)

		service := NewKnowledgeService(mockKnowledgeRepo, mockEmbeddingJobRepo)

		mockKnowledgeRepo.On("GetLatestVersion", mock.Anything, "non-existent").Return(nil, domain.ErrKnowledgeNotFound)

		// Execute
		result, err := service.GetLatestVersion(ctx, "non-existent")

		// Assert
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, domain.ErrKnowledgeNotFound, err)
		mockKnowledgeRepo.AssertExpectations(t)
	})
}

// TestKnowledgeService_Update tests the Update method
func TestKnowledgeService_Update(t *testing.T) {
	ctx := context.Background()

	t.Run("creates new version on update and queues embedding job", func(t *testing.T) {
		mockKnowledgeRepo := new(MockKnowledgeRepository)
		mockEmbeddingJobRepo := new(MockEmbeddingJobRepository)
		mockUUIDGen := NewMockUUIDGenerator("version-id-2", "job-id-2")

		service := NewKnowledgeServiceWithUUIDGen(mockKnowledgeRepo, mockEmbeddingJobRepo, mockUUIDGen)

		existingKnowledge := &domain.Knowledge{
			ID:        "knowledge-1",
			OrgID:     "org-1",
			ProjectID: "project-1",
			Type:      domain.KnowledgeTypeGuideline,
			Status:    domain.KnowledgeStatusDraft,
			Title:     "Original Title",
			Summary:   "Original summary",
			BodyMD:    "# Original Body",
			CreatedAt: time.Now().Add(-24 * time.Hour),
			UpdatedAt: time.Now().Add(-24 * time.Hour),
		}

		existingVersion := &domain.KnowledgeVersion{
			ID:            "version-id-1",
			KnowledgeID:   "knowledge-1",
			VersionNumber: 1,
			Title:         "Original Title",
			Summary:       "Original summary",
			BodyMD:        "# Original Body",
			CreatedAt:     time.Now().Add(-24 * time.Hour),
		}

		input := UpdateInput{
			KnowledgeID: "knowledge-1",
			Title:       "Updated Title",
			Summary:     "Updated summary",
			BodyMD:      "# Updated Body",
			Scope:       "/src/updated.go",
		}

		mockKnowledgeRepo.On("GetByID", mock.Anything, "knowledge-1").Return(existingKnowledge, nil)
		mockKnowledgeRepo.On("GetLatestVersion", mock.Anything, "knowledge-1").Return(existingVersion, nil)
		mockKnowledgeRepo.On("Update", mock.Anything, mock.MatchedBy(func(k *domain.Knowledge) bool {
			return k.ID == "knowledge-1" &&
				k.Title == "Updated Title" &&
				k.Summary == "Updated summary" &&
				k.BodyMD == "# Updated Body" &&
				k.Scope == "/src/updated.go"
		})).Return(nil)
		mockKnowledgeRepo.On("CreateVersion", mock.Anything, mock.MatchedBy(func(v *domain.KnowledgeVersion) bool {
			return v.ID == "version-id-2" &&
				v.KnowledgeID == "knowledge-1" &&
				v.VersionNumber == 2 &&
				v.Title == "Updated Title" &&
				v.Summary == "Updated summary" &&
				v.BodyMD == "# Updated Body"
		})).Return(nil)
		mockEmbeddingJobRepo.On("Create", mock.Anything, mock.MatchedBy(func(job *domain.EmbeddingJob) bool {
			return job.ID == "job-id-2" &&
				job.KnowledgeID == "knowledge-1" &&
				job.Status == domain.EmbeddingJobStatusPending
		})).Return(nil)

		// Execute
		knowledge, version, err := service.Update(ctx, input)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, knowledge)
		assert.NotNil(t, version)
		assert.Equal(t, "Updated Title", knowledge.Title)
		assert.Equal(t, int64(2), version.VersionNumber)
		assert.Equal(t, "Updated Title", version.Title)
		mockKnowledgeRepo.AssertExpectations(t)
		mockEmbeddingJobRepo.AssertExpectations(t)
	})

	t.Run("returns error when knowledge not found", func(t *testing.T) {
		mockKnowledgeRepo := new(MockKnowledgeRepository)
		mockEmbeddingJobRepo := new(MockEmbeddingJobRepository)

		service := NewKnowledgeService(mockKnowledgeRepo, mockEmbeddingJobRepo)

		input := UpdateInput{
			KnowledgeID: "non-existent",
			Title:       "Updated Title",
			Summary:     "Updated summary",
			BodyMD:      "# Updated Body",
		}

		mockKnowledgeRepo.On("GetByID", mock.Anything, "non-existent").Return(nil, domain.ErrKnowledgeNotFound)

		// Execute
		knowledge, version, err := service.Update(ctx, input)

		// Assert
		require.Error(t, err)
		assert.Nil(t, knowledge)
		assert.Nil(t, version)
		assert.Equal(t, domain.ErrKnowledgeNotFound, err)
		mockKnowledgeRepo.AssertExpectations(t)
	})

	t.Run("returns error when trying to update deprecated knowledge", func(t *testing.T) {
		mockKnowledgeRepo := new(MockKnowledgeRepository)
		mockEmbeddingJobRepo := new(MockEmbeddingJobRepository)

		service := NewKnowledgeService(mockKnowledgeRepo, mockEmbeddingJobRepo)

		deprecatedKnowledge := &domain.Knowledge{
			ID:        "knowledge-1",
			OrgID:     "org-1",
			ProjectID: "project-1",
			Type:      domain.KnowledgeTypeGuideline,
			Status:    domain.KnowledgeStatusDeprecated, // Deprecated!
			Title:     "Deprecated Knowledge",
			Summary:   "Deprecated summary",
			BodyMD:    "# Deprecated Body",
			CreatedAt: time.Now().Add(-24 * time.Hour),
			UpdatedAt: time.Now().Add(-24 * time.Hour),
		}

		input := UpdateInput{
			KnowledgeID: "knowledge-1",
			Title:       "Updated Title",
			Summary:     "Updated summary",
			BodyMD:      "# Updated Body",
		}

		mockKnowledgeRepo.On("GetByID", mock.Anything, "knowledge-1").Return(deprecatedKnowledge, nil)

		// Execute
		knowledge, version, err := service.Update(ctx, input)

		// Assert
		require.Error(t, err)
		assert.Nil(t, knowledge)
		assert.Nil(t, version)
		assert.Equal(t, domain.ErrCannotModifyDeprecated, err)
		mockKnowledgeRepo.AssertExpectations(t)
	})

	t.Run("returns error on repository update failure", func(t *testing.T) {
		mockKnowledgeRepo := new(MockKnowledgeRepository)
		mockEmbeddingJobRepo := new(MockEmbeddingJobRepository)
		mockUUIDGen := NewMockUUIDGenerator("version-id-2", "job-id-2")

		service := NewKnowledgeServiceWithUUIDGen(mockKnowledgeRepo, mockEmbeddingJobRepo, mockUUIDGen)

		existingKnowledge := &domain.Knowledge{
			ID:        "knowledge-1",
			OrgID:     "org-1",
			Status:    domain.KnowledgeStatusDraft,
			Title:     "Original Title",
			Summary:   "Original summary",
			BodyMD:    "# Original Body",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		existingVersion := &domain.KnowledgeVersion{
			ID:            "version-id-1",
			KnowledgeID:   "knowledge-1",
			VersionNumber: 1,
		}

		input := UpdateInput{
			KnowledgeID: "knowledge-1",
			Title:       "Updated Title",
			Summary:     "Updated summary",
			BodyMD:      "# Updated Body",
		}

		expectedErr := errors.New("database error")
		mockKnowledgeRepo.On("GetByID", mock.Anything, "knowledge-1").Return(existingKnowledge, nil)
		mockKnowledgeRepo.On("GetLatestVersion", mock.Anything, "knowledge-1").Return(existingVersion, nil)
		mockKnowledgeRepo.On("Update", mock.Anything, mock.Anything).Return(expectedErr)

		// Execute
		knowledge, version, err := service.Update(ctx, input)

		// Assert
		require.Error(t, err)
		assert.Nil(t, knowledge)
		assert.Nil(t, version)
		assert.Equal(t, expectedErr, err)
		mockKnowledgeRepo.AssertExpectations(t)
	})
}

// TestKnowledgeService_Deprecate tests the Deprecate method
func TestKnowledgeService_Deprecate(t *testing.T) {
	ctx := context.Background()

	t.Run("sets status to deprecated", func(t *testing.T) {
		mockKnowledgeRepo := new(MockKnowledgeRepository)
		mockEmbeddingJobRepo := new(MockEmbeddingJobRepository)

		service := NewKnowledgeService(mockKnowledgeRepo, mockEmbeddingJobRepo)

		existingKnowledge := &domain.Knowledge{
			ID:        "knowledge-1",
			OrgID:     "org-1",
			ProjectID: "project-1",
			Type:      domain.KnowledgeTypeGuideline,
			Status:    domain.KnowledgeStatusApproved,
			Title:     "Test Knowledge",
			Summary:   "Test summary",
			BodyMD:    "# Test Body",
			CreatedAt: time.Now().Add(-24 * time.Hour),
			UpdatedAt: time.Now().Add(-24 * time.Hour),
		}

		mockKnowledgeRepo.On("GetByID", mock.Anything, "knowledge-1").Return(existingKnowledge, nil)
		mockKnowledgeRepo.On("Update", mock.Anything, mock.MatchedBy(func(k *domain.Knowledge) bool {
			return k.ID == "knowledge-1" && k.Status == domain.KnowledgeStatusDeprecated
		})).Return(nil)

		// Execute
		result, err := service.Deprecate(ctx, "knowledge-1")

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, domain.KnowledgeStatusDeprecated, result.Status)
		mockKnowledgeRepo.AssertExpectations(t)
	})

	t.Run("returns error when knowledge not found", func(t *testing.T) {
		mockKnowledgeRepo := new(MockKnowledgeRepository)
		mockEmbeddingJobRepo := new(MockEmbeddingJobRepository)

		service := NewKnowledgeService(mockKnowledgeRepo, mockEmbeddingJobRepo)

		mockKnowledgeRepo.On("GetByID", mock.Anything, "non-existent").Return(nil, domain.ErrKnowledgeNotFound)

		// Execute
		result, err := service.Deprecate(ctx, "non-existent")

		// Assert
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, domain.ErrKnowledgeNotFound, err)
		mockKnowledgeRepo.AssertExpectations(t)
	})

	t.Run("returns error on repository update failure", func(t *testing.T) {
		mockKnowledgeRepo := new(MockKnowledgeRepository)
		mockEmbeddingJobRepo := new(MockEmbeddingJobRepository)

		service := NewKnowledgeService(mockKnowledgeRepo, mockEmbeddingJobRepo)

		existingKnowledge := &domain.Knowledge{
			ID:        "knowledge-1",
			OrgID:     "org-1",
			Status:    domain.KnowledgeStatusDraft,
			Title:     "Test Knowledge",
			BodyMD:    "# Test Body",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		expectedErr := errors.New("database error")
		mockKnowledgeRepo.On("GetByID", mock.Anything, "knowledge-1").Return(existingKnowledge, nil)
		mockKnowledgeRepo.On("Update", mock.Anything, mock.Anything).Return(expectedErr)

		// Execute
		result, err := service.Deprecate(ctx, "knowledge-1")

		// Assert
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, expectedErr, err)
		mockKnowledgeRepo.AssertExpectations(t)
	})
}

// TestKnowledgeService_ListByOrg tests the ListByOrg method
func TestKnowledgeService_ListByOrg(t *testing.T) {
	ctx := context.Background()

	t.Run("returns all knowledge for organization", func(t *testing.T) {
		mockKnowledgeRepo := new(MockKnowledgeRepository)
		mockEmbeddingJobRepo := new(MockEmbeddingJobRepository)

		service := NewKnowledgeService(mockKnowledgeRepo, mockEmbeddingJobRepo)

		expectedKnowledge := []*domain.Knowledge{
			{
				ID:    "knowledge-1",
				OrgID: "org-1",
				Title: "Knowledge 1",
			},
			{
				ID:    "knowledge-2",
				OrgID: "org-1",
				Title: "Knowledge 2",
			},
		}

		mockKnowledgeRepo.On("ListByOrg", mock.Anything, "org-1").Return(expectedKnowledge, nil)

		// Execute
		result, err := service.ListByOrg(ctx, "org-1")

		// Assert
		require.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, "Knowledge 1", result[0].Title)
		assert.Equal(t, "Knowledge 2", result[1].Title)
		mockKnowledgeRepo.AssertExpectations(t)
	})

	t.Run("returns empty list when no knowledge found", func(t *testing.T) {
		mockKnowledgeRepo := new(MockKnowledgeRepository)
		mockEmbeddingJobRepo := new(MockEmbeddingJobRepository)

		service := NewKnowledgeService(mockKnowledgeRepo, mockEmbeddingJobRepo)

		mockKnowledgeRepo.On("ListByOrg", mock.Anything, "org-empty").Return([]*domain.Knowledge{}, nil)

		// Execute
		result, err := service.ListByOrg(ctx, "org-empty")

		// Assert
		require.NoError(t, err)
		assert.Empty(t, result)
		mockKnowledgeRepo.AssertExpectations(t)
	})

	t.Run("returns error on repository failure", func(t *testing.T) {
		mockKnowledgeRepo := new(MockKnowledgeRepository)
		mockEmbeddingJobRepo := new(MockEmbeddingJobRepository)

		service := NewKnowledgeService(mockKnowledgeRepo, mockEmbeddingJobRepo)

		expectedErr := errors.New("database error")
		mockKnowledgeRepo.On("ListByOrg", mock.Anything, "org-1").Return(nil, expectedErr)

		// Execute
		result, err := service.ListByOrg(ctx, "org-1")

		// Assert
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, expectedErr, err)
		mockKnowledgeRepo.AssertExpectations(t)
	})
}

// TestKnowledgeService_ListByProject tests the ListByProject method
func TestKnowledgeService_ListByProject(t *testing.T) {
	ctx := context.Background()

	t.Run("returns all knowledge for project", func(t *testing.T) {
		mockKnowledgeRepo := new(MockKnowledgeRepository)
		mockEmbeddingJobRepo := new(MockEmbeddingJobRepository)

		service := NewKnowledgeService(mockKnowledgeRepo, mockEmbeddingJobRepo)

		expectedKnowledge := []*domain.Knowledge{
			{
				ID:        "knowledge-1",
				OrgID:     "org-1",
				ProjectID: "project-1",
				Title:     "Project Knowledge 1",
			},
			{
				ID:        "knowledge-2",
				OrgID:     "org-1",
				ProjectID: "project-1",
				Title:     "Project Knowledge 2",
			},
		}

		mockKnowledgeRepo.On("ListByProject", mock.Anything, "project-1").Return(expectedKnowledge, nil)

		// Execute
		result, err := service.ListByProject(ctx, "project-1")

		// Assert
		require.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, "Project Knowledge 1", result[0].Title)
		assert.Equal(t, "Project Knowledge 2", result[1].Title)
		mockKnowledgeRepo.AssertExpectations(t)
	})

	t.Run("returns empty list when no knowledge found", func(t *testing.T) {
		mockKnowledgeRepo := new(MockKnowledgeRepository)
		mockEmbeddingJobRepo := new(MockEmbeddingJobRepository)

		service := NewKnowledgeService(mockKnowledgeRepo, mockEmbeddingJobRepo)

		mockKnowledgeRepo.On("ListByProject", mock.Anything, "project-empty").Return([]*domain.Knowledge{}, nil)

		// Execute
		result, err := service.ListByProject(ctx, "project-empty")

		// Assert
		require.NoError(t, err)
		assert.Empty(t, result)
		mockKnowledgeRepo.AssertExpectations(t)
	})

	t.Run("returns error on repository failure", func(t *testing.T) {
		mockKnowledgeRepo := new(MockKnowledgeRepository)
		mockEmbeddingJobRepo := new(MockEmbeddingJobRepository)

		service := NewKnowledgeService(mockKnowledgeRepo, mockEmbeddingJobRepo)

		expectedErr := errors.New("database error")
		mockKnowledgeRepo.On("ListByProject", mock.Anything, "project-1").Return(nil, expectedErr)

		// Execute
		result, err := service.ListByProject(ctx, "project-1")

		// Assert
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, expectedErr, err)
		mockKnowledgeRepo.AssertExpectations(t)
	})
}

// TestKnowledgeService_Create_QueuesEmbeddingJob specifically tests embedding job queuing
func TestKnowledgeService_Create_QueuesEmbeddingJob(t *testing.T) {
	ctx := context.Background()

	t.Run("queues embedding job with correct knowledge ID", func(t *testing.T) {
		mockKnowledgeRepo := new(MockKnowledgeRepository)
		mockEmbeddingJobRepo := new(MockEmbeddingJobRepository)
		mockUUIDGen := NewMockUUIDGenerator("knowledge-id-123", "version-id-1", "job-id-456")

		service := NewKnowledgeServiceWithUUIDGen(mockKnowledgeRepo, mockEmbeddingJobRepo, mockUUIDGen)

		input := CreateInput{
			OrgID:   "org-1",
			Type:    domain.KnowledgeTypeSnippet,
			Title:   "Code Snippet",
			Summary: "A helpful code snippet",
			BodyMD:  "```go\nfmt.Println(\"Hello\")\n```",
		}

		mockKnowledgeRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
		mockKnowledgeRepo.On("CreateVersion", mock.Anything, mock.Anything).Return(nil)

		var capturedJob *domain.EmbeddingJob
		mockEmbeddingJobRepo.On("Create", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
			capturedJob = args.Get(1).(*domain.EmbeddingJob)
		}).Return(nil)

		// Execute
		_, err := service.Create(ctx, input)

		// Assert
		require.NoError(t, err)
		require.NotNil(t, capturedJob)
		assert.Equal(t, "job-id-456", capturedJob.ID)
		assert.Equal(t, "knowledge-id-123", capturedJob.KnowledgeID)
		assert.Equal(t, domain.EmbeddingJobStatusPending, capturedJob.Status)
		assert.Equal(t, int32(0), capturedJob.Retries)
		assert.Empty(t, capturedJob.Error)
		assert.Nil(t, capturedJob.ProcessedAt)
		mockEmbeddingJobRepo.AssertExpectations(t)
	})
}

// TestKnowledgeService_Update_CreatesNewVersion specifically tests immutable versioning
func TestKnowledgeService_Update_CreatesNewVersion(t *testing.T) {
	ctx := context.Background()

	t.Run("update creates new version without modifying old version", func(t *testing.T) {
		mockKnowledgeRepo := new(MockKnowledgeRepository)
		mockEmbeddingJobRepo := new(MockEmbeddingJobRepository)
		mockUUIDGen := NewMockUUIDGenerator("version-id-5", "job-id-new")

		service := NewKnowledgeServiceWithUUIDGen(mockKnowledgeRepo, mockEmbeddingJobRepo, mockUUIDGen)

		existingKnowledge := &domain.Knowledge{
			ID:        "knowledge-1",
			OrgID:     "org-1",
			Status:    domain.KnowledgeStatusApproved,
			Title:     "Original",
			Summary:   "Original summary",
			BodyMD:    "# Original",
			CreatedAt: time.Now().Add(-72 * time.Hour),
			UpdatedAt: time.Now().Add(-48 * time.Hour),
		}

		existingVersion := &domain.KnowledgeVersion{
			ID:            "version-id-4",
			KnowledgeID:   "knowledge-1",
			VersionNumber: 4,
			Title:         "Version 4 Title",
			Summary:       "Version 4 summary",
			BodyMD:        "# Version 4 Body",
			CreatedAt:     time.Now().Add(-48 * time.Hour),
		}

		input := UpdateInput{
			KnowledgeID: "knowledge-1",
			Title:       "Version 5 Title",
			Summary:     "Version 5 summary",
			BodyMD:      "# Version 5 Body",
		}

		mockKnowledgeRepo.On("GetByID", mock.Anything, "knowledge-1").Return(existingKnowledge, nil)
		mockKnowledgeRepo.On("GetLatestVersion", mock.Anything, "knowledge-1").Return(existingVersion, nil)
		mockKnowledgeRepo.On("Update", mock.Anything, mock.Anything).Return(nil)

		var capturedVersion *domain.KnowledgeVersion
		mockKnowledgeRepo.On("CreateVersion", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
			capturedVersion = args.Get(1).(*domain.KnowledgeVersion)
		}).Return(nil)
		mockEmbeddingJobRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

		// Execute
		_, newVersion, err := service.Update(ctx, input)

		// Assert
		require.NoError(t, err)
		require.NotNil(t, capturedVersion)

		// Verify new version was created with incremented version number
		assert.Equal(t, "version-id-5", capturedVersion.ID)
		assert.Equal(t, "knowledge-1", capturedVersion.KnowledgeID)
		assert.Equal(t, int64(5), capturedVersion.VersionNumber) // 4 + 1
		assert.Equal(t, "Version 5 Title", capturedVersion.Title)
		assert.Equal(t, "Version 5 summary", capturedVersion.Summary)
		assert.Equal(t, "# Version 5 Body", capturedVersion.BodyMD)

		// Verify returned version matches
		assert.Equal(t, newVersion.VersionNumber, capturedVersion.VersionNumber)

		// Verify old version was NOT modified (we never call UpdateVersion)
		mockKnowledgeRepo.AssertNotCalled(t, "UpdateVersion", mock.Anything, mock.Anything)
	})
}
