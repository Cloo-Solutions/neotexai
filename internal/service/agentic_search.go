package service

import (
	"sort"
	"strings"
	"unicode"
)

var stopwords = map[string]struct{}{
	"a": {}, "an": {}, "the": {}, "and": {}, "or": {}, "of": {}, "to": {}, "for": {}, "with": {}, "by": {},
	"in": {}, "on": {}, "at": {}, "from": {}, "as": {}, "is": {}, "are": {}, "was": {}, "were": {}, "be": {},
	"been": {}, "it": {}, "this": {}, "that": {}, "these": {}, "those": {}, "we": {}, "our": {}, "you": {},
	"your": {}, "i": {}, "me": {}, "my": {}, "us": {}, "them": {}, "they": {}, "their": {}, "do": {},
	"does": {}, "did": {}, "what": {}, "how": {}, "why": {}, "when": {}, "where": {}, "which": {}, "can": {},
	"could": {}, "should": {}, "would": {}, "may": {}, "might": {}, "will": {}, "shall": {},
}

func mergeResults(dst map[string]*SearchResult, results []*SearchResult) {
	for _, r := range results {
		if r == nil {
			continue
		}
		sourceType := normalizeSourceType(r.SourceType)
		key := sourceType + ":" + r.ID
		if r.SourceType == "" {
			r.SourceType = sourceType
		}
		existing, ok := dst[key]
		if !ok || r.Score > existing.Score {
			dst[key] = r
			continue
		}
		if existing.Title == "" && r.Title != "" {
			existing.Title = r.Title
		}
		if existing.Summary == "" && r.Summary != "" {
			existing.Summary = r.Summary
		}
		if existing.Scope == "" && r.Scope != "" {
			existing.Scope = r.Scope
		}
		if existing.Snippet == "" && r.Snippet != "" {
			existing.Snippet = r.Snippet
		}
		if existing.UpdatedAt.IsZero() && !r.UpdatedAt.IsZero() {
			existing.UpdatedAt = r.UpdatedAt
		}
	}
}

func sortResultsByScore(results map[string]*SearchResult) []*SearchResult {
	out := make([]*SearchResult, 0, len(results))
	for _, r := range results {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		return out[i].UpdatedAt.After(out[j].UpdatedAt)
	})
	return out
}

func generateQueryVariants(query string, max int) []string {
	if max <= 0 {
		return nil
	}
	clean := strings.TrimSpace(query)
	if clean == "" {
		return nil
	}

	seen := make(map[string]struct{})
	var variants []string

	add := func(candidate string) {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			return
		}
		key := strings.ToLower(candidate)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		variants = append(variants, candidate)
	}

	for _, part := range splitQueryParts(clean) {
		add(part)
		if len(variants) >= max {
			return variants[:max]
		}
	}

	keyword := keywordQuery(clean)
	add(keyword)

	if len(variants) > max {
		return variants[:max]
	}
	return variants
}

func splitQueryParts(query string) []string {
	parts := []string{}
	chunks := strings.FieldsFunc(query, func(r rune) bool {
		switch r {
		case ',', ';', '/', '|', ':', '?', '!', '(', ')', '[', ']', '{', '}':
			return true
		default:
			return false
		}
	})

	for _, chunk := range chunks {
		subParts := strings.Split(chunk, " and ")
		for _, sub := range subParts {
			sub = strings.TrimSpace(sub)
			if sub != "" {
				parts = append(parts, sub)
			}
		}
	}

	return parts
}

func keywordQuery(query string) string {
	var tokens []string
	for _, token := range strings.FieldsFunc(query, func(r rune) bool {
		return unicode.IsSpace(r)
	}) {
		clean := strings.ToLower(strings.TrimSpace(token))
		if clean == "" {
			continue
		}
		if _, ok := stopwords[clean]; ok {
			continue
		}
		tokens = append(tokens, token)
	}
	return strings.Join(tokens, " ")
}
