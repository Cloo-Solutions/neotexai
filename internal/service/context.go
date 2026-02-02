package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/cloo-solutions/neotexai/internal/domain"
	"github.com/cloo-solutions/neotexai/internal/telemetry"
)

// KnowledgeManifestItem represents a lightweight knowledge index entry
type KnowledgeManifestItem struct {
	ID      string
	Title   string
	Summary string
	Type    domain.KnowledgeType
	Scope   string
}

// SearchFilters represents filters for knowledge search
type SearchFilters struct {
	OrgID      string
	ProjectID  string
	Type       domain.KnowledgeType
	Status     domain.KnowledgeStatus
	PathPrefix string
	// SourceType filters results to "knowledge" or "asset"
	SourceType string
}

// SearchMode controls retrieval strategy.
type SearchMode string

const (
	SearchModeHybrid   SearchMode = "hybrid"
	SearchModeSemantic SearchMode = "semantic"
	SearchModeLexical  SearchMode = "lexical"
)

// SearchResult represents a search result with relevance score
type SearchResult struct {
	ID        string
	Title     string
	Summary   string
	Scope     string
	Snippet   string
	UpdatedAt time.Time
	Score     float32
	// SourceType is "knowledge" or "asset"
	SourceType string
}

// ChunkSearchResult represents a chunk-level knowledge hit.
type ChunkSearchResult struct {
	KnowledgeID string
	Title       string
	Summary     string
	Scope       string
	Content     string
	UpdatedAt   time.Time
	Score       float32
}

// SearchInput represents input for search operation
type SearchInput struct {
	Query   string
	Filters SearchFilters
	Mode    SearchMode
	Exact   bool
	Limit   int
	Cursor  string
}

// SearchOutput represents output from search operation
type SearchOutput struct {
	Results  []*SearchResult
	Cursor   string
	HasMore  bool
	SearchID string
}

// RelevantItem represents a top-ranked knowledge or asset item.
type RelevantItem struct {
	ID         string
	SourceType string
	Score      float32
	Scope      string
	Knowledge  *domain.Knowledge
	Asset      *domain.Asset
}

// RelevantKnowledgeInput represents input for GetRelevantKnowledge
type RelevantKnowledgeInput struct {
	OrgID     string
	ProjectID string
	FilePath  string
	Query     string
}

// ContextRepositoryInterface defines the repository interface for context operations
type ContextRepositoryInterface interface {
	GetManifest(ctx context.Context, orgID, projectID string) ([]*KnowledgeManifestItem, error)
	SearchKnowledgeChunksSemantic(ctx context.Context, embedding []float32, filters SearchFilters, limit int) ([]*ChunkSearchResult, error)
	SearchKnowledgeChunksLexical(ctx context.Context, query string, filters SearchFilters, limit int) ([]*ChunkSearchResult, error)
	SearchKnowledgeSemantic(ctx context.Context, embedding []float32, filters SearchFilters, limit int) ([]*SearchResult, error)
	SearchKnowledgeLexical(ctx context.Context, query string, filters SearchFilters, limit int) ([]*SearchResult, error)
	SearchAssetsSemantic(ctx context.Context, embedding []float32, filters SearchFilters, limit int) ([]*SearchResult, error)
	SearchAssetsLexical(ctx context.Context, query string, filters SearchFilters, limit int) ([]*SearchResult, error)
	GetByIDs(ctx context.Context, ids []string) ([]*domain.Knowledge, error)
	GetAssetsByIDs(ctx context.Context, ids []string) ([]*domain.Asset, error)
}

// EmbeddingServiceInterface defines the interface for embedding generation
type EmbeddingServiceInterface interface {
	GenerateEmbedding(ctx context.Context, text string) ([]float32, error)
}

// ContextService handles context retrieval for knowledge items
type ContextService struct {
	repo      ContextRepositoryInterface
	embedding EmbeddingServiceInterface
	cfg       ContextServiceConfig
}

// AgenticSearchConfig controls iterative search behavior.
type AgenticSearchConfig struct {
	Enabled       bool
	MaxIterations int
	MinResults    int
	MaxVariants   int
}

// ContextServiceConfig controls context service behavior.
type ContextServiceConfig struct {
	AgenticSearch AgenticSearchConfig
}

// DefaultContextServiceConfig returns the default service configuration.
func DefaultContextServiceConfig() ContextServiceConfig {
	return ContextServiceConfig{
		AgenticSearch: AgenticSearchConfig{
			Enabled:       true,
			MaxIterations: 2,
			MinResults:    3,
			MaxVariants:   6,
		},
	}
}

// NewContextService creates a new ContextService instance
func NewContextService(
	repo ContextRepositoryInterface,
	embedding EmbeddingServiceInterface,
) *ContextService {
	return NewContextServiceWithConfig(repo, embedding, DefaultContextServiceConfig())
}

// NewContextServiceWithConfig creates a new ContextService with explicit configuration.
func NewContextServiceWithConfig(
	repo ContextRepositoryInterface,
	embedding EmbeddingServiceInterface,
	cfg ContextServiceConfig,
) *ContextService {
	return &ContextService{
		repo:      repo,
		embedding: embedding,
		cfg:       cfg,
	}
}

// GetManifest returns a lightweight index of knowledge items for org/project
func (s *ContextService) GetManifest(ctx context.Context, orgID, projectID string) ([]*KnowledgeManifestItem, error) {
	ctx, span := telemetry.StartSpan(ctx, "ContextService.GetManifest", telemetry.SpanAttributes{
		OrgID:     orgID,
		ProjectID: projectID,
		Operation: "manifest",
	})
	defer span.End()

	return s.repo.GetManifest(ctx, orgID, projectID)
}

// Search performs hybrid search with metadata filtering and pgvector similarity
func (s *ContextService) Search(ctx context.Context, input SearchInput) (*SearchOutput, error) {
	ctx, span := telemetry.StartSpan(ctx, "ContextService.Search", telemetry.SpanAttributes{
		OrgID:     input.Filters.OrgID,
		ProjectID: input.Filters.ProjectID,
		Operation: "search",
	})
	defer span.End()

	input.Mode = normalizeSearchMode(input.Mode)
	input.Filters.SourceType = normalizeSourceTypeFilter(input.Filters.SourceType)

	limit := input.Limit
	if limit <= 0 {
		limit = 20
	}

	offset := 0
	if input.Cursor != "" {
		var err error
		offset, err = decodeSearchCursor(input.Cursor)
		if err != nil {
			offset = 0
		}
	}

	fetchLimit := limit + offset + 1
	results, err := s.searchOnce(ctx, input, fetchLimit)
	if err != nil {
		return nil, err
	}

	if !s.shouldAgentic(input, results, fetchLimit) {
		return s.buildSearchOutput(results, offset, limit), nil
	}

	allResults, err := s.agenticSearch(ctx, input, results, fetchLimit)
	if err != nil {
		return nil, err
	}

	return s.buildSearchOutput(allResults, offset, limit), nil
}

func (s *ContextService) buildSearchOutput(results []*SearchResult, offset, limit int) *SearchOutput {
	if offset >= len(results) {
		return &SearchOutput{
			Results: []*SearchResult{},
			HasMore: false,
		}
	}

	results = results[offset:]

	hasMore := len(results) > limit
	if hasMore {
		results = results[:limit]
	}

	var cursor string
	if hasMore && len(results) > 0 {
		cursor = encodeSearchCursor(offset + limit)
	}

	return &SearchOutput{
		Results: results,
		Cursor:  cursor,
		HasMore: hasMore,
	}
}

func encodeSearchCursor(offset int) string {
	raw := fmt.Sprintf("%d|%s", offset, time.Now().UTC().Format(time.RFC3339Nano))
	return base64.StdEncoding.EncodeToString([]byte(raw))
}

func decodeSearchCursor(cursor string) (int, error) {
	decoded, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return 0, err
	}
	parts := strings.SplitN(string(decoded), "|", 2)
	if len(parts) < 1 {
		return 0, fmt.Errorf("invalid cursor")
	}
	return strconv.Atoi(parts[0])
}

// GetRelevantKnowledge auto-fetches up to 3 relevant knowledge or asset items based on context
// Relevance ranking: exact file match > path prefix match > semantic similarity
func (s *ContextService) GetRelevantKnowledge(ctx context.Context, input RelevantKnowledgeInput) ([]*RelevantItem, error) {
	// Search with org/project filter
	filters := SearchFilters{
		OrgID:     input.OrgID,
		ProjectID: input.ProjectID,
	}

	// Fetch more results than needed to allow for re-ranking
	results, err := s.searchOnce(ctx, SearchInput{
		Query:   input.Query,
		Filters: filters,
		Mode:    SearchModeSemantic,
		Exact:   true,
	}, 10)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return []*RelevantItem{}, nil
	}

	// Re-rank results by relevance
	rankedResults := rankByRelevance(results, input.FilePath)

	// Take top 3
	maxResults := 3
	if len(rankedResults) < maxResults {
		maxResults = len(rankedResults)
	}
	topResults := rankedResults[:maxResults]

	knowledgeIDs := make([]string, 0, len(topResults))
	assetIDs := make([]string, 0, len(topResults))
	for _, r := range topResults {
		sourceType := normalizeSourceType(r.SourceType)
		if sourceType == "asset" {
			assetIDs = append(assetIDs, r.ID)
		} else {
			knowledgeIDs = append(knowledgeIDs, r.ID)
		}
	}

	var knowledgeItems []*domain.Knowledge
	var assetItems []*domain.Asset
	if len(knowledgeIDs) > 0 {
		var err error
		knowledgeItems, err = s.repo.GetByIDs(ctx, knowledgeIDs)
		if err != nil {
			return nil, err
		}
	}
	if len(assetIDs) > 0 {
		var err error
		assetItems, err = s.repo.GetAssetsByIDs(ctx, assetIDs)
		if err != nil {
			return nil, err
		}
	}

	knowledgeByID := make(map[string]*domain.Knowledge, len(knowledgeItems))
	for _, item := range knowledgeItems {
		if item != nil {
			knowledgeByID[item.ID] = item
		}
	}
	assetByID := make(map[string]*domain.Asset, len(assetItems))
	for _, item := range assetItems {
		if item != nil {
			assetByID[item.ID] = item
		}
	}

	relevantItems := make([]*RelevantItem, 0, len(topResults))
	for _, r := range topResults {
		sourceType := normalizeSourceType(r.SourceType)
		item := &RelevantItem{
			ID:         r.ID,
			SourceType: sourceType,
			Score:      r.Score,
			Scope:      r.Scope,
		}
		if sourceType == "asset" {
			if asset, ok := assetByID[r.ID]; ok {
				item.Asset = asset
				relevantItems = append(relevantItems, item)
			}
			continue
		}
		if knowledge, ok := knowledgeByID[r.ID]; ok {
			item.Knowledge = knowledge
			relevantItems = append(relevantItems, item)
		}
	}

	return relevantItems, nil
}

// rankedSearchResult extends SearchResult with relevance score for ranking
type rankedSearchResult struct {
	*SearchResult
	relevanceScore int
}

// rankByRelevance re-ranks search results based on path relevance
// Priority: exact file match (3) > path prefix match (2) > semantic only (1)
func rankByRelevance(results []*SearchResult, filePath string) []*SearchResult {
	if filePath == "" {
		return results
	}

	cleanFilePath := strings.TrimRight(filePath, "/")

	ranked := make([]rankedSearchResult, len(results))
	for i, r := range results {
		score := 1 // Default: semantic match only

		if r.Scope != "" {
			scope := strings.TrimRight(r.Scope, "/")
			// Exact file match
			if scope == cleanFilePath {
				score = 3
			} else if isPathPrefix(scope, cleanFilePath) {
				// Path prefix match: scope is a parent directory of filePath
				score = 2
			}
		}

		ranked[i] = rankedSearchResult{
			SearchResult:   r,
			relevanceScore: score,
		}
	}

	// Sort by relevance score (descending), then by semantic score (descending)
	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].relevanceScore != ranked[j].relevanceScore {
			return ranked[i].relevanceScore > ranked[j].relevanceScore
		}
		return ranked[i].Score > ranked[j].Score
	})

	// Convert back to SearchResult slice
	result := make([]*SearchResult, len(ranked))
	for i, r := range ranked {
		result[i] = r.SearchResult
	}

	return result
}

func isPathPrefix(scope, filePath string) bool {
	if scope == "" || filePath == "" {
		return false
	}
	if scope == "/" {
		return strings.HasPrefix(filePath, "/")
	}
	if !strings.HasPrefix(filePath, scope) {
		return false
	}
	if len(filePath) == len(scope) {
		return true
	}
	return filePath[len(scope)] == '/'
}

func normalizeSourceType(sourceType string) string {
	sourceType = strings.TrimSpace(strings.ToLower(sourceType))
	if sourceType == "" {
		return "knowledge"
	}
	return sourceType
}
