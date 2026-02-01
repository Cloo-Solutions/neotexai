package client

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// SearchRequest represents the search API request.
type SearchRequest struct {
	Query     string `json:"query"`
	ProjectID string `json:"project_id,omitempty"`
	Type      string `json:"type,omitempty"`
	Limit     int    `json:"limit,omitempty"`
	Cursor    string `json:"cursor,omitempty"`
}

// SearchResult represents a search result.
type SearchResult struct {
	ID      string  `json:"id"`
	Title   string  `json:"title"`
	Summary string  `json:"summary,omitempty"`
	Scope   string  `json:"scope,omitempty"`
	Score   float32 `json:"score"`
}

// SearchResponse represents the search API response.
type SearchResponse struct {
	Results []SearchResult `json:"results"`
	Cursor  string         `json:"cursor,omitempty"`
	HasMore bool           `json:"has_more"`
}

// SearchCmd creates the search command.
func SearchCmd() *cobra.Command {
	var (
		knowledgeType string
		limit         int
		cursor        string
	)

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search knowledge",
		Long:  "Searches the knowledge base using semantic search.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			outputJSON, _ := cmd.Flags().GetBool("output")
			return runSearch(args[0], knowledgeType, limit, cursor, outputJSON)
		},
	}

	cmd.Flags().StringVarP(&knowledgeType, "type", "t", "", "Filter by knowledge type")
	cmd.Flags().IntVarP(&limit, "limit", "n", 20, "Maximum number of results")
	cmd.Flags().StringVar(&cursor, "cursor", "", "Pagination cursor from previous response")

	return cmd
}

func runSearch(query, knowledgeType string, limit int, cursor string, outputJSON bool) error {
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

	// Build search request
	req := SearchRequest{
		Query:     query,
		ProjectID: config.ProjectID,
		Type:      knowledgeType,
		Limit:     limit,
		Cursor:    cursor,
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
			fmt.Printf("%d. %s (%.2f)\n", i+1, result.Title, result.Score)
			if result.Summary != "" {
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
			fmt.Printf("   ID: %s\n", result.ID)
			if i < len(searchResp.Results)-1 {
				fmt.Println(strings.Repeat("-", 40))
			}
		}
		if searchResp.HasMore && searchResp.Cursor != "" {
			fmt.Printf("\n%s\n", strings.Repeat("-", 40))
			fmt.Printf("More results available. Use --cursor %s\n", searchResp.Cursor)
		}
	}

	return nil
}
