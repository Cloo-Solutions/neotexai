package client

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
)

// SearchRequest represents the search API request.
type SearchRequest struct {
	Query      string `json:"query"`
	ProjectID  string `json:"project_id,omitempty"`
	Type       string `json:"type,omitempty"`
	Status     string `json:"status,omitempty"`
	PathPrefix string `json:"path_prefix,omitempty"`
	SourceType string `json:"source_type,omitempty"`
	Mode       string `json:"mode,omitempty"`
	Exact      bool   `json:"exact,omitempty"`
	Limit      int    `json:"limit,omitempty"`
	Cursor     string `json:"cursor,omitempty"`
}

// SearchResult represents a search result.
type SearchResult struct {
	ID         string  `json:"id"`
	Title      string  `json:"title"`
	Summary    string  `json:"summary,omitempty"`
	Scope      string  `json:"scope,omitempty"`
	Snippet    string  `json:"snippet,omitempty"`
	UpdatedAt  string  `json:"updated_at,omitempty"`
	Score      float32 `json:"score"`
	SourceType string  `json:"source_type"`
	ChunkID    string  `json:"chunk_id,omitempty"`
	ChunkIndex int     `json:"chunk_index,omitempty"`
}

// SearchResponse represents the search API response.
type SearchResponse struct {
	Results  []SearchResult `json:"results"`
	Cursor   string         `json:"cursor,omitempty"`
	HasMore  bool           `json:"has_more"`
	SearchID string         `json:"search_id,omitempty"`
}

// SearchCmd creates the search command.
func SearchCmd() *cobra.Command {
	var (
		knowledgeType string
		status        string
		pathPrefix    string
		sourceType    string
		mode          string
		projectID     string
		limit         int
		cursor        string
		exact         bool
	)

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search knowledge and assets",
		Long:  "Searches the knowledge base and assets using hybrid semantic + lexical search.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			outputJSON, _ := cmd.Flags().GetBool("output")
			return runSearch(args[0], knowledgeType, status, pathPrefix, sourceType, mode, projectID, exact, limit, cursor, outputJSON)
		},
	}

	cmd.Flags().StringVarP(&knowledgeType, "type", "t", "", "Filter by knowledge type")
	cmd.Flags().StringVar(&status, "status", "", "Filter by knowledge status")
	cmd.Flags().StringVar(&pathPrefix, "path", "", "Filter by scope path prefix")
	cmd.Flags().StringVar(&sourceType, "source", "", "Filter by source type (knowledge|asset)")
	cmd.Flags().StringVar(&mode, "mode", "", "Search mode (hybrid|semantic|lexical)")
	cmd.Flags().StringVar(&projectID, "project", "", "Override project ID from config")
	cmd.Flags().BoolVar(&exact, "exact", false, "Disable query expansion")
	cmd.Flags().IntVarP(&limit, "limit", "n", 20, "Maximum number of results")
	cmd.Flags().StringVar(&cursor, "cursor", "", "Pagination cursor from previous response")

	return cmd
}

func runSearch(query, knowledgeType, status, pathPrefix, sourceType, mode, projectID string, exact bool, limit int, cursor string, outputJSON bool) error {
	// Load config to get project ID
	config, err := LoadConfig()
	if err != nil {
		return err
	}

	// Create API client
	api, err := NewAPIClient()
	if err != nil {
		return err
	}

	cleanQuery, inline := parseInlineFilters(query)
	if cleanQuery == "" {
		return fmt.Errorf("query is required (inline filters must be combined with search terms)")
	}

	if knowledgeType == "" {
		knowledgeType = inline.Type
	}
	if status == "" {
		status = inline.Status
	}
	if pathPrefix == "" {
		pathPrefix = inline.PathPrefix
	}
	if sourceType == "" {
		sourceType = inline.SourceType
	}
	if mode == "" {
		mode = inline.Mode
	}

	effectiveProjectID := config.ProjectID
	if inline.ProjectID != "" {
		effectiveProjectID = inline.ProjectID
	}
	if projectID != "" {
		effectiveProjectID = projectID
	}

	// Build search request
	req := SearchRequest{
		Query:      cleanQuery,
		ProjectID:  effectiveProjectID,
		Type:       knowledgeType,
		Status:     status,
		PathPrefix: pathPrefix,
		SourceType: sourceType,
		Mode:       mode,
		Exact:      exact,
		Limit:      limit,
		Cursor:     cursor,
	}

	// Perform search
	resp, err := api.Post("/search", req)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	// Parse results
	var searchResp SearchResponse
	if err := json.Unmarshal(resp.Data, &searchResp); err != nil {
		return fmt.Errorf("failed to parse search results: %w", err)
	}

	if outputJSON {
		output, _ := json.MarshalIndent(searchResp, "", "  ")
		fmt.Println(string(output))
	} else {
		if len(searchResp.Results) == 0 {
			fmt.Println("No results found.")
			return nil
		}

		fmt.Printf("Found %d results:\n\n", len(searchResp.Results))
		for i, result := range searchResp.Results {
			sourceType := result.SourceType
			if sourceType == "" {
				sourceType = "knowledge"
			}
			fmt.Printf("%d. %s [%s] (%.2f)\n", i+1, result.Title, sourceType, result.Score)
			if result.Snippet != "" {
				fmt.Printf("   %s\n", highlightSnippet(result.Snippet, cleanQuery))
			} else if result.Summary != "" {
				// Truncate summary to 100 chars
				summary := result.Summary
				if len(summary) > 100 {
					summary = summary[:97] + "..."
				}
				fmt.Printf("   %s\n", summary)
			}
			if result.Scope != "" {
				fmt.Printf("   Scope: %s\n", result.Scope)
			}
			if result.ChunkID != "" {
				fmt.Printf("   Chunk: %s (index %d)\n", result.ChunkID, result.ChunkIndex)
			}
			if result.UpdatedAt != "" {
				fmt.Printf("   Updated: %s\n", result.UpdatedAt)
			}
			fmt.Printf("   ID: %s\n", result.ID)
			if i < len(searchResp.Results)-1 {
				fmt.Println(strings.Repeat("-", 40))
			}
		}
		if searchResp.HasMore && searchResp.Cursor != "" {
			fmt.Printf("\n%s\n", strings.Repeat("-", 40))
			fmt.Printf("More results available. Use --cursor %s\n", searchResp.Cursor)
		}
		if searchResp.SearchID != "" {
			fmt.Printf("\nSearch ID: %s\n", searchResp.SearchID)
		}
	}

	return nil
}

type inlineSearchFilters struct {
	Type       string
	Status     string
	PathPrefix string
	SourceType string
	Mode       string
	ProjectID  string
}

func parseInlineFilters(query string) (string, inlineSearchFilters) {
	filters := inlineSearchFilters{}
	parts := strings.Fields(query)
	if len(parts) == 0 {
		return "", filters
	}
	remaining := make([]string, 0, len(parts))
	for _, part := range parts {
		key, value, ok := strings.Cut(part, ":")
		if !ok {
			remaining = append(remaining, part)
			continue
		}
		key = strings.ToLower(strings.TrimSpace(key))
		value = strings.Trim(strings.TrimSpace(value), "\"'")
		if value == "" {
			remaining = append(remaining, part)
			continue
		}
		switch key {
		case "type":
			filters.Type = value
		case "status":
			filters.Status = value
		case "path", "scope":
			filters.PathPrefix = value
		case "source", "kind":
			filters.SourceType = value
		case "mode":
			filters.Mode = value
		case "project":
			filters.ProjectID = value
		default:
			remaining = append(remaining, part)
		}
	}
	return strings.Join(remaining, " "), filters
}

func highlightSnippet(snippet, query string) string {
	if snippet == "" || query == "" {
		return snippet
	}
	terms := strings.Fields(strings.ToLower(query))
	if len(terms) == 0 {
		return snippet
	}
	out := snippet
	for _, term := range terms {
		if len(term) < 3 {
			continue
		}
		re := regexp.MustCompile(`(?i)` + regexp.QuoteMeta(term))
		out = re.ReplaceAllString(out, "[$0]")
	}
	return out
}
