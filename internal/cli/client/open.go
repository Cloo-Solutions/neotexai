package client

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// OpenRequest represents the open API request.
type OpenRequest struct {
	ID         string        `json:"id"`
	SourceType string        `json:"source_type,omitempty"`
	ChunkID    string        `json:"chunk_id,omitempty"`
	Range      *ContentRange `json:"range,omitempty"`
	IncludeURL bool          `json:"include_url,omitempty"`
}

// ContentRange specifies a portion of content to retrieve.
type ContentRange struct {
	StartLine int `json:"start_line,omitempty"`
	EndLine   int `json:"end_line,omitempty"`
	MaxChars  int `json:"max_chars,omitempty"`
}

// OpenResponse represents the open API response.
type OpenResponse struct {
	ID          string   `json:"id"`
	SourceType  string   `json:"source_type"`
	Title       string   `json:"title"`
	Content     string   `json:"content,omitempty"`
	TotalLines  int      `json:"total_lines,omitempty"`
	TotalChars  int      `json:"total_chars,omitempty"`
	ChunkID     string   `json:"chunk_id,omitempty"`
	ChunkIndex  int      `json:"chunk_index,omitempty"`
	ChunkCount  int      `json:"chunk_count,omitempty"`
	UpdatedAt   string   `json:"updated_at,omitempty"`
	Filename    string   `json:"filename,omitempty"`
	MimeType    string   `json:"mime_type,omitempty"`
	SizeBytes   int64    `json:"size_bytes,omitempty"`
	Description string   `json:"description,omitempty"`
	Keywords    []string `json:"keywords,omitempty"`
	DownloadURL string   `json:"download_url,omitempty"`
}

// OpenCmd creates the context open command.
func OpenCmd() *cobra.Command {
	var (
		sourceType string
		chunkID    string
		lines      string
		maxChars   int
		includeURL bool
	)

	cmd := &cobra.Command{
		Use:   "open <id>",
		Short: "Open a knowledge item, chunk, or asset",
		Long:  "Retrieves content for a knowledge item, chunk, or asset metadata.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			outputJSON, _ := cmd.Flags().GetBool("output")
			return runOpen(args[0], sourceType, chunkID, lines, maxChars, includeURL, outputJSON)
		},
	}

	cmd.Flags().StringVar(&sourceType, "source", "", "Source type (knowledge|asset|chunk)")
	cmd.Flags().StringVar(&chunkID, "chunk", "", "Specific chunk ID to retrieve")
	cmd.Flags().StringVar(&lines, "lines", "", "Line range (e.g., 0:100)")
	cmd.Flags().IntVar(&maxChars, "max-chars", 4000, "Maximum characters to return")
	cmd.Flags().BoolVar(&includeURL, "include-url", false, "Include presigned download URL for assets")

	return cmd
}

func runOpen(id, sourceType, chunkID, lines string, maxChars int, includeURL, outputJSON bool) error {
	api, err := NewAPIClient()
	if err != nil {
		return err
	}

	req := OpenRequest{
		ID:         id,
		SourceType: sourceType,
		ChunkID:    chunkID,
		IncludeURL: includeURL,
	}

	// Parse line range
	if lines != "" {
		startLine, endLine, err := parseLineRange(lines)
		if err != nil {
			return fmt.Errorf("invalid line range: %w", err)
		}
		req.Range = &ContentRange{
			StartLine: startLine,
			EndLine:   endLine,
			MaxChars:  maxChars,
		}
	} else if maxChars > 0 {
		req.Range = &ContentRange{
			MaxChars: maxChars,
		}
	}

	resp, err := api.Post("/context/open", req)
	if err != nil {
		return fmt.Errorf("open failed: %w", err)
	}

	var openResp OpenResponse
	if err := json.Unmarshal(resp.Data, &openResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if outputJSON {
		output, _ := json.MarshalIndent(openResp, "", "  ")
		fmt.Println(string(output))
		return nil
	}

	// Human-readable output
	fmt.Printf("ID: %s\n", openResp.ID)
	fmt.Printf("Title: %s\n", openResp.Title)
	fmt.Printf("Type: %s\n", openResp.SourceType)

	if openResp.SourceType == "asset" {
		if openResp.Filename != "" {
			fmt.Printf("Filename: %s\n", openResp.Filename)
		}
		if openResp.MimeType != "" {
			fmt.Printf("MIME Type: %s\n", openResp.MimeType)
		}
		if openResp.Description != "" {
			fmt.Printf("Description: %s\n", openResp.Description)
		}
		if len(openResp.Keywords) > 0 {
			fmt.Printf("Keywords: %s\n", strings.Join(openResp.Keywords, ", "))
		}
		if openResp.DownloadURL != "" {
			fmt.Printf("Download URL: %s\n", openResp.DownloadURL)
		}
	} else {
		if openResp.ChunkID != "" {
			fmt.Printf("Chunk ID: %s\n", openResp.ChunkID)
			fmt.Printf("Chunk Index: %d of %d\n", openResp.ChunkIndex, openResp.ChunkCount)
		} else if openResp.ChunkCount > 0 {
			fmt.Printf("Chunk Count: %d\n", openResp.ChunkCount)
		}
		if openResp.TotalChars > 0 {
			fmt.Printf("Total: %d lines, %d chars\n", openResp.TotalLines, openResp.TotalChars)
		}
		if openResp.UpdatedAt != "" {
			fmt.Printf("Updated: %s\n", openResp.UpdatedAt)
		}
		if openResp.Content != "" {
			fmt.Println(strings.Repeat("-", 40))
			fmt.Println(openResp.Content)
		}
	}

	return nil
}

func parseLineRange(lines string) (int, int, error) {
	parts := strings.SplitN(lines, ":", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("expected format: start:end")
	}
	var start, end int
	if _, err := fmt.Sscanf(parts[0], "%d", &start); err != nil {
		return 0, 0, err
	}
	if _, err := fmt.Sscanf(parts[1], "%d", &end); err != nil {
		return 0, 0, err
	}
	return start, end, nil
}
