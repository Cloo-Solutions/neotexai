package service

import (
	"context"
	"time"

	"github.com/cloo-solutions/neotexai/internal/domain"
	"github.com/cloo-solutions/neotexai/internal/pagination"
	"github.com/cloo-solutions/neotexai/internal/telemetry"
	"github.com/google/uuid"
)

// KnowledgeRepositoryInterface defines the repository interface for knowledge persistence
type KnowledgeRepositoryInterface interface {
	Create(ctx context.Context, k *domain.Knowledge) error
	GetByID(ctx context.Context, id string) (*domain.Knowledge, error)
	ListByOrg(ctx context.Context, orgID string) ([]*domain.Knowledge, error)
	ListByProject(ctx context.Context, projectID string) ([]*domain.Knowledge, error)
	ListByOrgWithCursor(ctx context.Context, orgID string, cursor *pagination.Cursor, limit int) (*KnowledgePageResult, error)
	ListByProjectWithCursor(ctx context.Context, projectID string, cursor *pagination.Cursor, limit int) (*KnowledgePageResult, error)
	Update(ctx context.Context, k *domain.Knowledge) error
	CreateVersion(ctx context.Context, v *domain.KnowledgeVersion) error
	GetLatestVersion(ctx context.Context, knowledgeID string) (*domain.KnowledgeVersion, error)
	GetVersions(ctx context.Context, knowledgeID string) ([]*domain.KnowledgeVersion, error)
}

type KnowledgePageResult struct {
	Items      []*domain.Knowledge
	NextCursor string
	HasMore    bool
}

// EmbeddingJobRepositoryInterface defines the repository interface for embedding job persistence
type EmbeddingJobRepositoryInterface interface {
	Create(ctx context.Context, job *domain.EmbeddingJob) error
}

// UUIDGenerator defines interface for UUID generation (for testing)
type UUIDGenerator interface {
	NewString() string
}

// DefaultUUIDGenerator is the default UUID generator using google/uuid
type DefaultUUIDGenerator struct{}

// NewString generates a new UUID string
func (g *DefaultUUIDGenerator) NewString() string {
	return uuid.NewString()
}

// KnowledgeService handles business logic for knowledge items
type KnowledgeService struct {
	knowledgeRepo    KnowledgeRepositoryInterface
	embeddingJobRepo EmbeddingJobRepositoryInterface
	uuidGen          UUIDGenerator
}

// NewKnowledgeService creates a new KnowledgeService instance
func NewKnowledgeService(
	knowledgeRepo KnowledgeRepositoryInterface,
	embeddingJobRepo EmbeddingJobRepositoryInterface,
) *KnowledgeService {
	return &KnowledgeService{
		knowledgeRepo:    knowledgeRepo,
		embeddingJobRepo: embeddingJobRepo,
		uuidGen:          &DefaultUUIDGenerator{},
	}
}

// NewKnowledgeServiceWithUUIDGen creates a new KnowledgeService with custom UUID generator (for testing)
func NewKnowledgeServiceWithUUIDGen(
	knowledgeRepo KnowledgeRepositoryInterface,
	embeddingJobRepo EmbeddingJobRepositoryInterface,
	uuidGen UUIDGenerator,
) *KnowledgeService {
	return &KnowledgeService{
		knowledgeRepo:    knowledgeRepo,
		embeddingJobRepo: embeddingJobRepo,
		uuidGen:          uuidGen,
	}
}

// CreateInput represents the input for creating a knowledge item
type CreateInput struct {
	OrgID     string
	ProjectID string
	Type      domain.KnowledgeType
	Title     string
	Summary   string
	BodyMD    string
	Scope     string
}

// UpdateInput represents the input for updating a knowledge item
type UpdateInput struct {
	KnowledgeID string
	Title       string
	Summary     string
	BodyMD      string
	Scope       string
}

type ListKnowledgeInput struct {
	OrgID     string
	ProjectID string
	Cursor    string
	Limit     int
}

type ListKnowledgeOutput struct {
	Items   []*domain.Knowledge
	Cursor  string
	HasMore bool
}

// Create creates a new knowledge item with its first version and queues an embedding job
func (s *KnowledgeService) Create(ctx context.Context, input CreateInput) (*domain.Knowledge, error) {
	ctx, span := telemetry.StartSpan(ctx, "KnowledgeService.Create", telemetry.SpanAttributes{
		OrgID:     input.OrgID,
		ProjectID: input.ProjectID,
		Operation: "create",
	})
	defer span.End()

	now := time.Now().UTC()
	knowledgeID := s.uuidGen.NewString()
	versionID := s.uuidGen.NewString()
	jobID := s.uuidGen.NewString()

	// Create knowledge record
	knowledge := &domain.Knowledge{
		ID:        knowledgeID,
		OrgID:     input.OrgID,
		ProjectID: input.ProjectID,
		Type:      input.Type,
		Status:    domain.KnowledgeStatusDraft,
		Title:     input.Title,
		Summary:   input.Summary,
		BodyMD:    input.BodyMD,
		Scope:     input.Scope,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Validate knowledge
	if err := domain.ValidateKnowledge(knowledge); err != nil {
		return nil, err
	}

	// Create knowledge in repository
	if err := s.knowledgeRepo.Create(ctx, knowledge); err != nil {
		return nil, err
	}

	// Create first version
	version := &domain.KnowledgeVersion{
		ID:            versionID,
		KnowledgeID:   knowledgeID,
		VersionNumber: 1,
		Title:         input.Title,
		Summary:       input.Summary,
		BodyMD:        input.BodyMD,
		CreatedAt:     now,
	}

	if err := s.knowledgeRepo.CreateVersion(ctx, version); err != nil {
		return nil, err
	}

	// Queue embedding job
	job := &domain.EmbeddingJob{
		ID:          jobID,
		KnowledgeID: knowledgeID,
		Status:      domain.EmbeddingJobStatusPending,
		Retries:     0,
		Error:       "",
		CreatedAt:   now,
		ProcessedAt: nil,
	}

	if err := s.embeddingJobRepo.Create(ctx, job); err != nil {
		return nil, err
	}

	return knowledge, nil
}

// GetByID retrieves a knowledge item by ID
func (s *KnowledgeService) GetByID(ctx context.Context, id string) (*domain.Knowledge, error) {
	ctx, span := telemetry.StartSpan(ctx, "KnowledgeService.GetByID", telemetry.SpanAttributes{
		KnowledgeID: id,
		Operation:   "get",
	})
	defer span.End()

	return s.knowledgeRepo.GetByID(ctx, id)
}

// GetLatestVersion retrieves the latest version for a knowledge item
func (s *KnowledgeService) GetLatestVersion(ctx context.Context, knowledgeID string) (*domain.KnowledgeVersion, error) {
	return s.knowledgeRepo.GetLatestVersion(ctx, knowledgeID)
}

// Update creates a new version of a knowledge item (immutable versioning) and queues an embedding job
func (s *KnowledgeService) Update(ctx context.Context, input UpdateInput) (*domain.Knowledge, *domain.KnowledgeVersion, error) {
	ctx, span := telemetry.StartSpan(ctx, "KnowledgeService.Update", telemetry.SpanAttributes{
		KnowledgeID: input.KnowledgeID,
		Operation:   "update",
	})
	defer span.End()

	now := time.Now().UTC()

	// Get existing knowledge
	knowledge, err := s.knowledgeRepo.GetByID(ctx, input.KnowledgeID)
	if err != nil {
		return nil, nil, err
	}

	// Cannot modify deprecated knowledge
	if knowledge.Status == domain.KnowledgeStatusDeprecated {
		return nil, nil, domain.ErrCannotModifyDeprecated
	}

	// Get the latest version to determine next version number
	latestVersion, err := s.knowledgeRepo.GetLatestVersion(ctx, input.KnowledgeID)
	if err != nil {
		return nil, nil, err
	}

	// Update knowledge record
	knowledge.Title = input.Title
	knowledge.Summary = input.Summary
	knowledge.BodyMD = input.BodyMD
	knowledge.Scope = input.Scope
	knowledge.UpdatedAt = now

	if err := s.knowledgeRepo.Update(ctx, knowledge); err != nil {
		return nil, nil, err
	}

	// Create new version (immutable)
	versionID := s.uuidGen.NewString()
	newVersion := &domain.KnowledgeVersion{
		ID:            versionID,
		KnowledgeID:   input.KnowledgeID,
		VersionNumber: latestVersion.VersionNumber + 1,
		Title:         input.Title,
		Summary:       input.Summary,
		BodyMD:        input.BodyMD,
		CreatedAt:     now,
	}

	if err := s.knowledgeRepo.CreateVersion(ctx, newVersion); err != nil {
		return nil, nil, err
	}

	// Queue embedding job
	jobID := s.uuidGen.NewString()
	job := &domain.EmbeddingJob{
		ID:          jobID,
		KnowledgeID: input.KnowledgeID,
		Status:      domain.EmbeddingJobStatusPending,
		Retries:     0,
		Error:       "",
		CreatedAt:   now,
		ProcessedAt: nil,
	}

	if err := s.embeddingJobRepo.Create(ctx, job); err != nil {
		return nil, nil, err
	}

	return knowledge, newVersion, nil
}

// Deprecate sets the status of a knowledge item to deprecated
func (s *KnowledgeService) Deprecate(ctx context.Context, knowledgeID string) (*domain.Knowledge, error) {
	ctx, span := telemetry.StartSpan(ctx, "KnowledgeService.Deprecate", telemetry.SpanAttributes{
		KnowledgeID: knowledgeID,
		Operation:   "delete",
	})
	defer span.End()

	// Get existing knowledge
	knowledge, err := s.knowledgeRepo.GetByID(ctx, knowledgeID)
	if err != nil {
		return nil, err
	}

	// Update status to deprecated
	knowledge.Status = domain.KnowledgeStatusDeprecated
	knowledge.UpdatedAt = time.Now().UTC()

	if err := s.knowledgeRepo.Update(ctx, knowledge); err != nil {
		return nil, err
	}

	return knowledge, nil
}

// ListByOrg retrieves all knowledge items for an organization
func (s *KnowledgeService) ListByOrg(ctx context.Context, orgID string) ([]*domain.Knowledge, error) {
	return s.knowledgeRepo.ListByOrg(ctx, orgID)
}

// ListByProject retrieves all knowledge items for a project
func (s *KnowledgeService) ListByProject(ctx context.Context, projectID string) ([]*domain.Knowledge, error) {
	return s.knowledgeRepo.ListByProject(ctx, projectID)
}

func (s *KnowledgeService) ListKnowledge(ctx context.Context, input ListKnowledgeInput) (*ListKnowledgeOutput, error) {
	ctx, span := telemetry.StartSpan(ctx, "KnowledgeService.ListKnowledge", telemetry.SpanAttributes{
		OrgID:     input.OrgID,
		ProjectID: input.ProjectID,
		Operation: "list",
	})
	defer span.End()

	cursor, _ := pagination.DecodeCursor(input.Cursor)
	limit := input.Limit
	if limit <= 0 {
		limit = 20
	}

	var result *KnowledgePageResult
	var err error

	if input.ProjectID != "" {
		result, err = s.knowledgeRepo.ListByProjectWithCursor(ctx, input.ProjectID, cursor, limit)
	} else {
		result, err = s.knowledgeRepo.ListByOrgWithCursor(ctx, input.OrgID, cursor, limit)
	}

	if err != nil {
		return nil, err
	}

	return &ListKnowledgeOutput{
		Items:   result.Items,
		Cursor:  result.NextCursor,
		HasMore: result.HasMore,
	}, nil
}
