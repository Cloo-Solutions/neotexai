package service

import (
	"context"
	"math"
	"sort"
	"strings"
	"time"
)

const (
	defaultCandidateMultiplier = 4
	defaultMinCandidates       = 20
	defaultMaxCandidates       = 200
	defaultSnippetMaxChars     = 220

	rrfK             = 60
	semanticWeight   = 1.0
	lexicalWeight    = 0.85
	recencyWindowDays = 30
	recencyMaxBoost   = 0.10
	pathExactBoost   = 0.12
	pathPrefixBoost  = 0.06
)

func normalizeSearchMode(mode SearchMode) SearchMode {
	switch strings.ToLower(strings.TrimSpace(string(mode))) {
	case string(SearchModeSemantic):
		return SearchModeSemantic
	case string(SearchModeLexical):
		return SearchModeLexical
	default:
		return SearchModeHybrid
	}
}

func normalizeSourceTypeFilter(source string) string {
	value := strings.ToLower(strings.TrimSpace(source))
	if value == "" || value == "all" {
		return ""
	}
	if value == "knowledge" || value == "asset" {
		return value
	}
	return ""
}

func (s *ContextService) searchOnce(ctx context.Context, input SearchInput, limit int) ([]*SearchResult, error) {
	query := strings.TrimSpace(input.Query)
	if query == "" {
		return []*SearchResult{}, nil
	}

	mode := normalizeSearchMode(input.Mode)
	includeKnowledge := input.Filters.SourceType == "" || input.Filters.SourceType == "knowledge"
	includeAssets := input.Filters.SourceType == "" || input.Filters.SourceType == "asset"

	candidateLimit := limit * defaultCandidateMultiplier
	if candidateLimit < defaultMinCandidates {
		candidateLimit = defaultMinCandidates
	}
	if candidateLimit > defaultMaxCandidates {
		candidateLimit = defaultMaxCandidates
	}

	lexicalOK := strings.TrimSpace(keywordQuery(query)) != ""

	var embedding []float32
	var err error
	if mode != SearchModeLexical {
		embedding, err = s.embedding.GenerateEmbedding(ctx, query)
		if err != nil {
			return nil, err
		}
	}

	var semanticKnowledgeChunks []*ChunkSearchResult
	var lexicalKnowledgeChunks []*ChunkSearchResult
	var semanticKnowledgeDocs []*SearchResult
	var lexicalKnowledgeDocs []*SearchResult

	if includeKnowledge {
		if mode != SearchModeLexical {
			semanticKnowledgeChunks, err = s.repo.SearchKnowledgeChunksSemantic(ctx, embedding, input.Filters, candidateLimit)
			if err != nil {
				return nil, err
			}
		}
		if mode != SearchModeSemantic && lexicalOK {
			lexicalKnowledgeChunks, err = s.repo.SearchKnowledgeChunksLexical(ctx, query, input.Filters, candidateLimit)
			if err != nil {
				return nil, err
			}
		}

		if len(semanticKnowledgeChunks)+len(lexicalKnowledgeChunks) == 0 {
			if mode != SearchModeLexical {
				semanticKnowledgeDocs, err = s.repo.SearchKnowledgeSemantic(ctx, embedding, input.Filters, candidateLimit)
				if err != nil {
					return nil, err
				}
			}
			if mode != SearchModeSemantic && lexicalOK {
				lexicalKnowledgeDocs, err = s.repo.SearchKnowledgeLexical(ctx, query, input.Filters, candidateLimit)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	var semanticAssets []*SearchResult
	var lexicalAssets []*SearchResult
	if includeAssets {
		if mode != SearchModeLexical {
			semanticAssets, err = s.repo.SearchAssetsSemantic(ctx, embedding, input.Filters, candidateLimit)
			if err != nil {
				return nil, err
			}
		}
		if mode != SearchModeSemantic && lexicalOK {
			lexicalAssets, err = s.repo.SearchAssetsLexical(ctx, query, input.Filters, candidateLimit)
			if err != nil {
				return nil, err
			}
		}
	}

	semanticKnowledge := aggregateChunkResults(semanticKnowledgeChunks)
	if len(semanticKnowledge) == 0 {
		semanticKnowledge = semanticKnowledgeDocs
	}
	lexicalKnowledge := aggregateChunkResults(lexicalKnowledgeChunks)
	if len(lexicalKnowledge) == 0 {
		lexicalKnowledge = lexicalKnowledgeDocs
	}

	prepareResults(semanticKnowledge)
	prepareResults(lexicalKnowledge)
	prepareResults(semanticAssets)
	prepareResults(lexicalAssets)

	if mode == SearchModeSemantic {
		return mergeByScore(input.Filters, semanticKnowledge, semanticAssets), nil
	}
	if mode == SearchModeLexical {
		return mergeByScore(input.Filters, lexicalKnowledge, lexicalAssets), nil
	}

	return mergeHybridResults(input.Filters, semanticKnowledge, lexicalKnowledge, semanticAssets, lexicalAssets), nil
}

func (s *ContextService) shouldAgentic(input SearchInput, results []*SearchResult, limit int) bool {
	if input.Exact {
		return false
	}
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
		variantInput := input
		variantInput.Query = variant
		results, err := s.searchOnce(ctx, variantInput, limit)
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

func aggregateChunkResults(chunks []*ChunkSearchResult) []*SearchResult {
	if len(chunks) == 0 {
		return nil
	}
	// Track best chunk per knowledge ID (highest score), preserving first-seen order
	best := make(map[string]*ChunkSearchResult, len(chunks))
	order := make([]string, 0, len(chunks))
	for _, c := range chunks {
		if c == nil {
			continue
		}
		existing, ok := best[c.KnowledgeID]
		if !ok {
			order = append(order, c.KnowledgeID)
			best[c.KnowledgeID] = c
		} else if c.Score > existing.Score {
			best[c.KnowledgeID] = c
		}
	}
	results := make([]*SearchResult, 0, len(best))
	for _, knowledgeID := range order {
		c := best[knowledgeID]
		results = append(results, &SearchResult{
			ID:         c.KnowledgeID,
			Title:      c.Title,
			Summary:    c.Summary,
			Scope:      c.Scope,
			Snippet:    makeSnippet(c.Content),
			UpdatedAt:  c.UpdatedAt,
			Score:      c.Score,
			SourceType: "knowledge",
			ChunkID:    c.ChunkID,
			ChunkIndex: c.ChunkIndex,
		})
	}
	return results
}

func prepareResults(results []*SearchResult) {
	for _, r := range results {
		if r == nil {
			continue
		}
		r.SourceType = normalizeSourceType(r.SourceType)
		if r.Snippet == "" {
			r.Snippet = makeSnippet(r.Summary)
		} else {
			r.Snippet = makeSnippet(r.Snippet)
		}
		// Set ChunkIndex to -1 for non-chunk results (empty ChunkID)
		if r.ChunkID == "" {
			r.ChunkIndex = -1
		}
	}
}

func mergeByScore(filters SearchFilters, lists ...[]*SearchResult) []*SearchResult {
	merged := make(map[string]*SearchResult)
	for _, list := range lists {
		mergeResults(merged, list)
	}
	out := sortResultsByScore(merged)
	applySearchBoosts(out, filters)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		return out[i].UpdatedAt.After(out[j].UpdatedAt)
	})
	return out
}

type fusionCandidate struct {
	result       *SearchResult
	rrfScore     float32
	semanticScore float32
	lexicalScore  float32
}

func mergeHybridResults(filters SearchFilters, semanticKnowledge, lexicalKnowledge, semanticAssets, lexicalAssets []*SearchResult) []*SearchResult {
	candidates := make(map[string]*fusionCandidate)
	addList := func(list []*SearchResult, weight float32, semantic bool) {
		for i, r := range list {
			if r == nil {
				continue
			}
			key := normalizeSourceType(r.SourceType) + ":" + r.ID
			cand, ok := candidates[key]
			if !ok {
				cloned := *r
				cand = &fusionCandidate{result: &cloned}
				candidates[key] = cand
			}
			cand.rrfScore += weight / float32(rrfK+i+1)
			if semantic {
				cand.semanticScore = float32(math.Max(float64(cand.semanticScore), float64(r.Score)))
			} else {
				cand.lexicalScore = float32(math.Max(float64(cand.lexicalScore), float64(r.Score)))
			}
			if cand.result.Snippet == "" && r.Snippet != "" {
				cand.result.Snippet = r.Snippet
			}
			if cand.result.UpdatedAt.IsZero() && !r.UpdatedAt.IsZero() {
				cand.result.UpdatedAt = r.UpdatedAt
			}
			if cand.result.Title == "" && r.Title != "" {
				cand.result.Title = r.Title
			}
			if cand.result.Summary == "" && r.Summary != "" {
				cand.result.Summary = r.Summary
			}
			if cand.result.Scope == "" && r.Scope != "" {
				cand.result.Scope = r.Scope
			}
		}
	}

	addList(semanticKnowledge, semanticWeight, true)
	addList(semanticAssets, semanticWeight, true)
	addList(lexicalKnowledge, lexicalWeight, false)
	addList(lexicalAssets, lexicalWeight, false)

	out := make([]*SearchResult, 0, len(candidates))
	for _, cand := range candidates {
		cand.result.Score = cand.rrfScore
		out = append(out, cand.result)
	}

	applySearchBoosts(out, filters)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		return out[i].UpdatedAt.After(out[j].UpdatedAt)
	})
	return out
}

func applySearchBoosts(results []*SearchResult, filters SearchFilters) {
	if len(results) == 0 {
		return
	}
	for _, r := range results {
		if r == nil {
			continue
		}
		boost := float32(0)
		boost += pathBoost(r.Scope, filters.PathPrefix)
		boost += recencyBoost(r.UpdatedAt)
		r.Score += boost
	}
}

func pathBoost(scope, prefix string) float32 {
	if scope == "" || prefix == "" {
		return 0
	}
	cleanScope := strings.TrimRight(scope, "/")
	cleanPrefix := strings.TrimRight(prefix, "/")
	if cleanScope == cleanPrefix {
		return pathExactBoost
	}
	if isPathPrefix(cleanPrefix, cleanScope) {
		return pathPrefixBoost
	}
	return 0
}

func recencyBoost(updatedAt time.Time) float32 {
	if updatedAt.IsZero() {
		return 0
	}
	ageDays := time.Since(updatedAt).Hours() / 24
	if ageDays < 0 {
		ageDays = 0
	}
	if ageDays > recencyWindowDays {
		return 0
	}
	scale := 1 - (ageDays / recencyWindowDays)
	return float32(scale) * recencyMaxBoost
}

func makeSnippet(content string) string {
	if content == "" {
		return ""
	}
	clean := strings.Join(strings.Fields(content), " ")
	if len(clean) <= defaultSnippetMaxChars {
		return clean
	}
	return clean[:defaultSnippetMaxChars-3] + "..."
}
