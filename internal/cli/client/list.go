package client

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// ListAPIRequest represents the list API request.
type ListAPIRequest struct {
	ProjectID    string `json:"project_id,omitempty"`
	PathPrefix   string `json:"path_prefix,omitempty"`
	Type         string `json:"type,omitempty"`
	Status       string `json:"status,omitempty"`
	SourceType   string `json:"source_type,omitempty"`
	UpdatedSince string `json:"updated_since,omitempty"`
	Limit        int    `json:"limit,omitempty"`
	Cursor       string `json:"cursor,omitempty"`
}

// ListItemResponse represents a single item in the list response.
type ListItemResponse struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Scope      string `json:"scope,omitempty"`
	Type       string `json:"type,omitempty"`
	Status     string `json:"status,omitempty"`
	SourceType string `json:"source_type"`
	UpdatedAt  string `json:"updated_at,omitempty"`
	ChunkCount int    `json:"chunk_count,omitempty"`
	Filename   string `json:"filename,omitempty"`
	MimeType   string `json:"mime_type,omitempty"`
}

// ListAPIResponse represents the list API response.
type ListAPIResponse struct {
	Items   []ListItemResponse `json:"items"`
	Cursor  string             `json:"cursor,omitempty"`
	HasMore bool               `json:"has_more"`
}

// ListCmd creates the context list command.
func ListCmd() *cobra.Command {
	var (
		pathPrefix   string
		knowledgeType string
		status       string
		sourceType   string
		since        string
		projectID    string
		limit        int
		cursor       string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List knowledge items and assets",
		Long:  "Lists metadata for knowledge items and/or assets with filtering.",
		RunE: func(cmd *cobra.Command, args []string) error {
			outputJSON, _ := cmd.Flags().GetBool("output")
			return runList(pathPrefix, knowledgeType, status, sourceType, since, projectID, limit, cursor, outputJSON)
		},
	}

	cmd.Flags().StringVar(&pathPrefix, "path", "", "Filter by scope path prefix")
	cmd.Flags().StringVarP(&knowledgeType, "type", "t", "", "Filter by knowledge type")
	cmd.Flags().StringVar(&status, "status", "", "Filter by knowledge status")
	cmd.Flags().StringVar(&sourceType, "source", "", "Filter by source type (knowledge|asset|all)")
	cmd.Flags().StringVar(&since, "since", "", "Filter by updated_since (RFC3339 format)")
	cmd.Flags().StringVar(&projectID, "project", "", "Override project ID from config")
	cmd.Flags().IntVarP(&limit, "limit", "n", 50, "Maximum number of results")
	cmd.Flags().StringVar(&cursor, "cursor", "", "Pagination cursor from previous response")

	return cmd
}

func runList(pathPrefix, knowledgeType, status, sourceType, since, projectID string, limit int, cursor string, outputJSON bool) error {
	// Load config to get project ID
	config, err := LoadConfig()
	if err != nil {
		return err
	}

	api, err := NewAPIClient()
	if err != nil {
		return err
	}

	effectiveProjectID := config.ProjectID
	if projectID != "" {
		effectiveProjectID = projectID
	}

	req := ListAPIRequest{
		ProjectID:    effectiveProjectID,
		PathPrefix:   pathPrefix,
		Type:         knowledgeType,
		Status:       status,
		SourceType:   sourceType,
		UpdatedSince: since,
		Limit:        limit,
		Cursor:       cursor,
	}

	resp, err := api.Post("/context/list", req)
	if err != nil {
		return fmt.Errorf("list failed: %w", err)
	}

	var listResp ListAPIResponse
	if err := json.Unmarshal(resp.Data, &listResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if outputJSON {
		output, _ := json.MarshalIndent(listResp, "", "  ")
		fmt.Println(string(output))
		return nil
	}

	// Human-readable output
	if len(listResp.Items) == 0 {
		fmt.Println("No items found.")
		return nil
	}

	fmt.Printf("Found %d items:\n\n", len(listResp.Items))
	for i, item := range listResp.Items {
		fmt.Printf("%d. %s [%s]\n", i+1, item.Title, item.SourceType)
		if item.Scope != "" {
			fmt.Printf("   Scope: %s\n", item.Scope)
		}
		if item.Type != "" {
			fmt.Printf("   Type: %s, Status: %s\n", item.Type, item.Status)
		}
		if item.ChunkCount > 0 {
			fmt.Printf("   Chunks: %d\n", item.ChunkCount)
		}
		if item.MimeType != "" {
			fmt.Printf("   MIME: %s\n", item.MimeType)
		}
		if item.UpdatedAt != "" {
			fmt.Printf("   Updated: %s\n", item.UpdatedAt)
		}
		fmt.Printf("   ID: %s\n", item.ID)
		if i < len(listResp.Items)-1 {
			fmt.Println(strings.Repeat("-", 40))
		}
	}

	if listResp.HasMore && listResp.Cursor != "" {
		fmt.Printf("\n%s\n", strings.Repeat("-", 40))
		fmt.Printf("More results available. Use --cursor %s\n", listResp.Cursor)
	}

	return nil
}
