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
}

// SearchResult represents a search result with relevance score
type SearchResult struct {
	ID      string
	Title   string
	Summary string
	Scope   string
	Score   float32
}

// SearchInput represents input for search operation
type SearchInput struct {
	Query   string
	Filters SearchFilters
	Limit   int
	Cursor  string
}

// SearchOutput represents output from search operation
type SearchOutput struct {
	Results []*SearchResult
	Cursor  string
	HasMore bool
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
	SearchByEmbedding(ctx context.Context, embedding []float32, filters SearchFilters, limit int) ([]*SearchResult, error)
	GetByIDs(ctx context.Context, ids []string) ([]*domain.Knowledge, error)
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
	results, err := s.searchOnce(ctx, input.Query, input.Filters, fetchLimit)
	if err != nil {
		return nil, err
	}

	if !s.shouldAgentic(results, fetchLimit) {
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

func (s *ContextService) searchOnce(ctx context.Context, query string, filters SearchFilters, limit int) ([]*SearchResult, error) {
	embedding, err := s.embedding.GenerateEmbedding(ctx, query)
	if err != nil {
		return nil, err
	}
	return s.repo.SearchByEmbedding(ctx, embedding, filters, limit)
}

func (s *ContextService) shouldAgentic(results []*SearchResult, limit int) bool {
	if !s.cfg.AgenticSearch.Enabled {
		return false
	}
	minResults := s.cfg.AgenticSearch.MinResults
	if minResults <= 0 {
		return false
	}
	if minResults > limit {
		minResults = limit
	}
	return len(results) < minResults
}

func (s *ContextService) agenticSearch(ctx context.Context, input SearchInput, initial []*SearchResult, limit int) ([]*SearchResult, error) {
	merged := make(map[string]*SearchResult)
	mergeResults(merged, initial)

	variants := generateQueryVariants(input.Query, s.cfg.AgenticSearch.MaxVariants)
	maxIterations := s.cfg.AgenticSearch.MaxIterations
	if maxIterations <= 0 {
		return initial, nil
	}

	iterations := 0
	for _, variant := range variants {
		if iterations >= maxIterations {
			break
		}
		if variant == "" || strings.EqualFold(strings.TrimSpace(variant), strings.TrimSpace(input.Query)) {
			continue
		}
		results, err := s.searchOnce(ctx, variant, input.Filters, limit)
		if err != nil {
			return nil, err
		}
		mergeResults(merged, results)
		iterations++
		if len(merged) >= limit && limit > 0 {
			break
		}
	}

	return sortResultsByScore(merged), nil
}

// GetRelevantKnowledge auto-fetches up to 3 relevant items based on context
// Relevance ranking: exact file match > path prefix match > semantic similarity
func (s *ContextService) GetRelevantKnowledge(ctx context.Context, input RelevantKnowledgeInput) ([]*domain.Knowledge, error) {
	// Generate embedding for query
	embedding, err := s.embedding.GenerateEmbedding(ctx, input.Query)
	if err != nil {
		return nil, err
	}

	// Search with org/project filter
	filters := SearchFilters{
		OrgID:     input.OrgID,
		ProjectID: input.ProjectID,
	}

	// Fetch more results than needed to allow for re-ranking
	results, err := s.repo.SearchByEmbedding(ctx, embedding, filters, 10)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return []*domain.Knowledge{}, nil
	}

	// Re-rank results by relevance
	rankedResults := rankByRelevance(results, input.FilePath)

	// Take top 3
	maxResults := 3
	if len(rankedResults) < maxResults {
		maxResults = len(rankedResults)
	}
	topResults := rankedResults[:maxResults]

	// Extract IDs
	ids := make([]string, len(topResults))
	for i, r := range topResults {
		ids[i] = r.ID
	}

	// Fetch full knowledge items
	return s.repo.GetByIDs(ctx, ids)
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

	ranked := make([]rankedSearchResult, len(results))
	for i, r := range results {
		score := 1 // Default: semantic match only

		if r.Scope != "" {
			// Exact file match
			if r.Scope == filePath {
				score = 3
			} else if strings.HasPrefix(filePath, r.Scope) || strings.HasPrefix(r.Scope+"/", filePath[:min(len(filePath), len(r.Scope)+1)]) {
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
