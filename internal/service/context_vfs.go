package service

import (
	"context"
	"strings"
	"time"

	"github.com/cloo-solutions/neotexai/internal/domain"
	"github.com/cloo-solutions/neotexai/internal/telemetry"
)

// OpenInput represents input for the Open operation
type OpenInput struct {
	ID         string
	SourceType string // "knowledge", "asset", or "chunk"
	ChunkID    string // optional: specific chunk to retrieve
	Range      *ContentRange
	IncludeURL bool // for assets: include presigned download URL
}

// ContentRange specifies a portion of content to retrieve
type ContentRange struct {
	StartLine int
	EndLine   int
	MaxChars  int
}

// OpenResult represents the result of an Open operation
type OpenResult struct {
	ID         string
	SourceType string
	Title      string
	Content    string // sliced content for knowledge/chunk
	TotalLines int
	TotalChars int
	ChunkID    string
	ChunkIndex int
	ChunkCount int
	UpdatedAt  time.Time
	// Asset-specific fields
	Filename    string
	MimeType    string
	SizeBytes   int64
	Description string
	Keywords    []string
	DownloadURL string // only if IncludeURL was true
}

// ListInput represents input for the List operation
type ListInput struct {
	OrgID        string
	ProjectID    string
	PathPrefix   string
	Type         domain.KnowledgeType
	Status       domain.KnowledgeStatus
	SourceType   string // "knowledge", "asset", or "all"
	UpdatedSince *time.Time
	Limit        int
	Cursor       string
}

// ListItem represents a single item in the list response
type ListItem struct {
	ID         string
	Title      string
	Scope      string
	Type       domain.KnowledgeType
	Status     domain.KnowledgeStatus
	SourceType string
	UpdatedAt  time.Time
	ChunkCount int
	// Asset-specific fields
	Filename string
	MimeType string
}

// ListOutput represents the output of a List operation
type ListOutput struct {
	Items   []*ListItem
	Cursor  string
	HasMore bool
}

// VFSKnowledgeRepo provides knowledge item access for the VFS service
type VFSKnowledgeRepo interface {
	GetByID(ctx context.Context, id string) (*domain.Knowledge, error)
}

// VFSChunkRepo provides chunk access for the VFS service
type VFSChunkRepo interface {
	GetByID(ctx context.Context, chunkID string) (*domain.KnowledgeChunk, error)
	GetByKnowledgeID(ctx context.Context, knowledgeID string) ([]*domain.KnowledgeChunk, error)
	CountByKnowledgeID(ctx context.Context, knowledgeID string) (int, error)
}

// VFSAssetRepo provides asset access for the VFS service
type VFSAssetRepo interface {
	GetByID(ctx context.Context, id string) (*domain.Asset, error)
}

// VFSStorage provides presigned URL generation for the VFS service
type VFSStorage interface {
	GenerateDownloadURL(ctx context.Context, key string) (string, error)
}

// VFSListRepo provides listing capabilities for the VFS service
type VFSListRepo interface {
	ListKnowledge(ctx context.Context, input ListInput) ([]*ListItem, error)
	ListAssets(ctx context.Context, input ListInput) ([]*ListItem, error)
}

// VFSService provides virtual filesystem-like operations for knowledge/assets
type VFSService struct {
	knowledgeRepo VFSKnowledgeRepo
	chunkRepo     VFSChunkRepo
	assetRepo     VFSAssetRepo
	storage       VFSStorage
	listRepo      VFSListRepo
}

// NewVFSService creates a new VFSService
func NewVFSService(
	knowledgeRepo VFSKnowledgeRepo,
	chunkRepo VFSChunkRepo,
	assetRepo VFSAssetRepo,
	storage VFSStorage,
	listRepo VFSListRepo,
) *VFSService {
	return &VFSService{
		knowledgeRepo: knowledgeRepo,
		chunkRepo:     chunkRepo,
		assetRepo:     assetRepo,
		storage:       storage,
		listRepo:      listRepo,
	}
}

const (
	defaultMaxChars = 4000
	maxMaxChars     = 16000
)

// Open retrieves content for a knowledge item, chunk, or asset
func (s *VFSService) Open(ctx context.Context, input OpenInput) (*OpenResult, error) {
	ctx, span := telemetry.StartSpan(ctx, "VFSService.Open", telemetry.SpanAttributes{
		Operation: "open",
	})
	defer span.End()

	sourceType := normalizeSourceType(input.SourceType)

	switch sourceType {
	case "asset":
		return s.openAsset(ctx, input)
	case "chunk":
		return s.openChunk(ctx, input)
	default:
		return s.openKnowledge(ctx, input)
	}
}

func (s *VFSService) openKnowledge(ctx context.Context, input OpenInput) (*OpenResult, error) {
	// If chunk_id is provided, open that specific chunk
	if input.ChunkID != "" {
		return s.openChunk(ctx, OpenInput{
			ID:         input.ChunkID,
			SourceType: "chunk",
			Range:      input.Range,
		})
	}

	knowledge, err := s.knowledgeRepo.GetByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	content := knowledge.BodyMD
	totalLines := countLines(content)
	totalChars := len(content)

	// Apply range if specified
	if input.Range != nil {
		content = applyRange(content, input.Range)
	}

	// Get chunk count
	chunkCount, _ := s.chunkRepo.CountByKnowledgeID(ctx, knowledge.ID)

	return &OpenResult{
		ID:         knowledge.ID,
		SourceType: "knowledge",
		Title:      knowledge.Title,
		Content:    content,
		TotalLines: totalLines,
		TotalChars: totalChars,
		ChunkCount: chunkCount,
		ChunkIndex: -1,
		UpdatedAt:  knowledge.UpdatedAt,
	}, nil
}

func (s *VFSService) openChunk(ctx context.Context, input OpenInput) (*OpenResult, error) {
	chunk, err := s.chunkRepo.GetByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	content := chunk.Content
	totalLines := countLines(content)
	totalChars := len(content)

	// Apply range if specified
	if input.Range != nil {
		content = applyRange(content, input.Range)
	}

	// Get chunk count for the parent knowledge
	chunkCount, _ := s.chunkRepo.CountByKnowledgeID(ctx, chunk.KnowledgeID)

	return &OpenResult{
		ID:         chunk.KnowledgeID,
		SourceType: "knowledge",
		Title:      chunk.Title,
		Content:    content,
		TotalLines: totalLines,
		TotalChars: totalChars,
		ChunkID:    chunk.ID,
		ChunkIndex: chunk.ChunkIndex,
		ChunkCount: chunkCount,
		UpdatedAt:  chunk.UpdatedAt,
	}, nil
}

func (s *VFSService) openAsset(ctx context.Context, input OpenInput) (*OpenResult, error) {
	asset, err := s.assetRepo.GetByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	result := &OpenResult{
		ID:          asset.ID,
		SourceType:  "asset",
		Title:       asset.Filename,
		Filename:    asset.Filename,
		MimeType:    asset.MimeType,
		Description: asset.Description,
		Keywords:    asset.Keywords,
		UpdatedAt:   asset.CreatedAt,
		ChunkIndex:  -1,
	}

	// Generate presigned URL only if requested
	if input.IncludeURL && s.storage != nil {
		url, err := s.storage.GenerateDownloadURL(ctx, asset.StorageKey)
		if err != nil {
			return nil, err
		}
		result.DownloadURL = url
	}

	return result, nil
}

// List retrieves metadata for knowledge items and/or assets
func (s *VFSService) List(ctx context.Context, input ListInput) (*ListOutput, error) {
	ctx, span := telemetry.StartSpan(ctx, "VFSService.List", telemetry.SpanAttributes{
		OrgID:     input.OrgID,
		ProjectID: input.ProjectID,
		Operation: "list",
	})
	defer span.End()

	if input.Limit <= 0 {
		input.Limit = 50
	}

	sourceType := normalizeSourceTypeFilter(input.SourceType)

	var items []*ListItem
	var err error

	switch sourceType {
	case "asset":
		items, err = s.listRepo.ListAssets(ctx, input)
	case "knowledge":
		items, err = s.listRepo.ListKnowledge(ctx, input)
	default:
		// List both knowledge and assets
		knowledgeItems, kerr := s.listRepo.ListKnowledge(ctx, input)
		if kerr != nil {
			return nil, kerr
		}
		assetItems, aerr := s.listRepo.ListAssets(ctx, input)
		if aerr != nil {
			return nil, aerr
		}
		items = append(knowledgeItems, assetItems...)
	}

	if err != nil {
		return nil, err
	}

	// Apply pagination
	hasMore := len(items) > input.Limit
	if hasMore {
		items = items[:input.Limit]
	}

	var cursor string
	if hasMore && len(items) > 0 {
		cursor = encodeListCursor(items[len(items)-1])
	}

	return &ListOutput{
		Items:   items,
		Cursor:  cursor,
		HasMore: hasMore,
	}, nil
}

func countLines(content string) int {
	if content == "" {
		return 0
	}
	return strings.Count(content, "\n") + 1
}

func applyRange(content string, r *ContentRange) string {
	if content == "" {
		return ""
	}

	maxChars := r.MaxChars
	if maxChars <= 0 {
		maxChars = defaultMaxChars
	}
	if maxChars > maxMaxChars {
		maxChars = maxMaxChars
	}

	lines := strings.Split(content, "\n")

	startLine := r.StartLine
	if startLine < 0 {
		startLine = 0
	}
	if startLine >= len(lines) {
		return ""
	}

	endLine := r.EndLine
	if endLine <= 0 || endLine > len(lines) {
		endLine = len(lines)
	}
	if endLine <= startLine {
		return ""
	}

	sliced := strings.Join(lines[startLine:endLine], "\n")

	// Enforce max chars
	if len(sliced) > maxChars {
		sliced = sliced[:maxChars]
	}

	return sliced
}

func encodeListCursor(item *ListItem) string {
	if item == nil {
		return ""
	}
	// Simple cursor: updated_at|id
	return item.UpdatedAt.Format(time.RFC3339Nano) + "|" + item.ID
}
