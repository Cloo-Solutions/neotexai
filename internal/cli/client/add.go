package client

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// CreateKnowledgeRequest represents the create knowledge API request.
type CreateKnowledgeRequest struct {
	Type      string `json:"type"`
	Title     string `json:"title"`
	Summary   string `json:"summary,omitempty"`
	BodyMD    string `json:"body_md"`
	ProjectID string `json:"project_id,omitempty"`
	Scope     string `json:"scope,omitempty"`
}

// BatchResult represents a single result in a batch operation.
type BatchResult struct {
	ID     string `json:"id,omitempty"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
	Title  string `json:"title,omitempty"`
}

// BatchResponse represents the response for a batch operation.
type BatchResponse struct {
	Results   []BatchResult `json:"results"`
	Total     int           `json:"total"`
	Succeeded int           `json:"succeeded"`
	Failed    int           `json:"failed"`
}

const maxBatchSize = 100

// AddCmd creates the add command.
func AddCmd() *cobra.Command {
	var (
		file           string
		knowledgeType  string
		title          string
		summary        string
		scope          string
		batch          bool
		atomic         bool
		idempotencyKey string
		format         string
		stream         bool
	)

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add knowledge from stdin or file",
		Long: `Add knowledge from JSON input (stdin or file) or markdown with flags.

Examples:
  # Add from JSON on stdin
  echo '{"type":"guideline","title":"Test","body_md":"# Test"}' | neotex add

  # Add from JSON file
  neotex add --file knowledge.json

  # Add from markdown file with flags
  neotex add --file guide.md --type guideline --title "My Guide"

  # Batch add from JSON array
  echo '[{"type":"guideline","title":"Test1","body_md":"# Test1"},{"type":"guideline","title":"Test2","body_md":"# Test2"}]' | neotex add --batch

  # Atomic batch add (all-or-nothing)
  neotex add --batch --atomic --file batch.json

  # Streaming batch add from JSONL (one JSON object per line)
  cat batch.jsonl | neotex add --batch --format jsonl --stream`,
		RunE: func(cmd *cobra.Command, args []string) error {
			outputJSON, _ := cmd.Flags().GetBool("output")
			if batch {
				if format == "jsonl" || stream {
					return runStreamingBatchAdd(file, outputJSON, idempotencyKey)
				}
				return runBatchAdd(file, outputJSON, atomic, idempotencyKey)
			}
			return runAdd(file, knowledgeType, title, summary, scope, outputJSON, idempotencyKey)
		},
	}

	cmd.Flags().StringVarP(&file, "file", "f", "", "Input file (JSON or markdown)")
	cmd.Flags().StringVarP(&knowledgeType, "type", "t", "", "Knowledge type (guideline, learning, decision, template, checklist, snippet)")
	cmd.Flags().StringVar(&title, "title", "", "Title (required with --file for markdown)")
	cmd.Flags().StringVar(&summary, "summary", "", "Summary (optional)")
	cmd.Flags().StringVar(&scope, "scope", "", "Scope (file path pattern)")
	cmd.Flags().BoolVar(&batch, "batch", false, "Enable batch mode (expects JSON array input)")
	cmd.Flags().BoolVar(&atomic, "atomic", false, "Atomic mode: all-or-nothing (only with --batch)")
	cmd.Flags().StringVar(&idempotencyKey, "idempotency-key", "", "Idempotency key for request deduplication")
	cmd.Flags().StringVar(&format, "format", "json", "Input format: json (array) or jsonl (line-delimited)")
	cmd.Flags().BoolVar(&stream, "stream", false, "Enable streaming mode for memory-efficient batch processing")

	return cmd
}

func runAdd(file, knowledgeType, title, summary, scope string, outputJSON bool, idempotencyKey string) error {
	config, err := LoadConfig()
	if err != nil {
		return err
	}

	api, err := NewAPIClient()
	if err != nil {
		return err
	}

	var req CreateKnowledgeRequest
	req.ProjectID = config.ProjectID
	req.Scope = scope

	// Read input
	var input []byte
	if file != "" {
		input, err = os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}
	} else {
		input, err = io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("failed to read stdin: %w", err)
		}
	}

	if len(input) == 0 {
		return fmt.Errorf("no input provided")
	}

	// Try to parse as JSON first
	if isJSONInput(input) {
		var jsonReq CreateKnowledgeRequest
		if err := json.Unmarshal(input, &jsonReq); err != nil {
			return fmt.Errorf("failed to parse JSON input: %w", err)
		}
		req.Type = jsonReq.Type
		req.Title = jsonReq.Title
		req.Summary = jsonReq.Summary
		req.BodyMD = jsonReq.BodyMD
		if jsonReq.Scope != "" {
			req.Scope = jsonReq.Scope
		}
	} else {
		// Treat as markdown
		if title == "" {
			return fmt.Errorf("--title is required when adding markdown content")
		}
		if knowledgeType == "" {
			return fmt.Errorf("--type is required when adding markdown content")
		}
		req.Type = knowledgeType
		req.Title = title
		req.Summary = summary
		req.BodyMD = string(input)
	}

	// Override with flags if provided
	if knowledgeType != "" {
		req.Type = knowledgeType
	}
	if title != "" {
		req.Title = title
	}
	if summary != "" {
		req.Summary = summary
	}

	// Validate
	if req.Type == "" {
		return fmt.Errorf("type is required")
	}
	if req.Title == "" {
		return fmt.Errorf("title is required")
	}
	if req.BodyMD == "" {
		return fmt.Errorf("body is required")
	}

	opts := RequestOptions{IdempotencyKey: idempotencyKey}
	resp, err := api.PostWithOptions("/knowledge", req, opts)
	if err != nil {
		return fmt.Errorf("failed to create knowledge: %w", err)
	}

	// Parse response
	var knowledge Knowledge
	if err := json.Unmarshal(resp.Data, &knowledge); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if outputJSON {
		output, _ := json.MarshalIndent(knowledge, "", "  ")
		fmt.Println(string(output))
	} else {
		fmt.Printf("Created knowledge: %s\n", knowledge.ID)
		fmt.Printf("Title: %s\n", knowledge.Title)
		fmt.Printf("Type: %s\n", knowledge.Type)
	}

	return nil
}

func isJSONInput(input []byte) bool {
	s := strings.TrimSpace(string(input))
	return len(s) > 0 && (s[0] == '{' || s[0] == '[')
}

func runBatchAdd(file string, outputJSON, atomic bool, idempotencyKey string) error {
	config, err := LoadConfig()
	if err != nil {
		return err
	}

	api, err := NewAPIClient()
	if err != nil {
		return err
	}

	opts := RequestOptions{IdempotencyKey: idempotencyKey}

	var input []byte
	if file != "" {
		input, err = os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}
	} else {
		input, err = io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("failed to read stdin: %w", err)
		}
	}

	if len(input) == 0 {
		return fmt.Errorf("no input provided")
	}

	var items []CreateKnowledgeRequest
	if err := json.Unmarshal(input, &items); err != nil {
		return fmt.Errorf("failed to parse JSON array: %w - batch mode expects a JSON array", err)
	}

	if len(items) == 0 {
		return fmt.Errorf("empty batch: no items provided")
	}

	if len(items) > maxBatchSize {
		return fmt.Errorf("batch size %d exceeds maximum of %d items", len(items), maxBatchSize)
	}

	response := BatchResponse{
		Results: make([]BatchResult, 0, len(items)),
		Total:   len(items),
	}

	for i, item := range items {
		item.ProjectID = config.ProjectID

		if item.Type == "" {
			result := BatchResult{
				Status: "failed",
				Error:  "type is required",
				Title:  item.Title,
			}
			if atomic {
				return reportAtomicFailure(response, result, i, outputJSON)
			}
			response.Results = append(response.Results, result)
			response.Failed++
			continue
		}
		if item.Title == "" {
			result := BatchResult{
				Status: "failed",
				Error:  "title is required",
			}
			if atomic {
				return reportAtomicFailure(response, result, i, outputJSON)
			}
			response.Results = append(response.Results, result)
			response.Failed++
			continue
		}
		if item.BodyMD == "" {
			result := BatchResult{
				Status: "failed",
				Error:  "body_md is required",
				Title:  item.Title,
			}
			if atomic {
				return reportAtomicFailure(response, result, i, outputJSON)
			}
			response.Results = append(response.Results, result)
			response.Failed++
			continue
		}

		resp, err := api.PostWithOptions("/knowledge", item, opts)
		if err != nil {
			result := BatchResult{
				Status: "failed",
				Error:  err.Error(),
				Title:  item.Title,
			}
			if atomic {
				return reportAtomicFailure(response, result, i, outputJSON)
			}
			response.Results = append(response.Results, result)
			response.Failed++
			continue
		}

		var knowledge Knowledge
		if err := json.Unmarshal(resp.Data, &knowledge); err != nil {
			result := BatchResult{
				Status: "failed",
				Error:  fmt.Sprintf("failed to parse response: %v", err),
				Title:  item.Title,
			}
			if atomic {
				return reportAtomicFailure(response, result, i, outputJSON)
			}
			response.Results = append(response.Results, result)
			response.Failed++
			continue
		}

		response.Results = append(response.Results, BatchResult{
			ID:     knowledge.ID,
			Status: "created",
			Title:  knowledge.Title,
		})
		response.Succeeded++
	}

	output, _ := json.MarshalIndent(response, "", "  ")
	fmt.Println(string(output))

	if response.Failed > 0 && !outputJSON {
		return fmt.Errorf("batch completed with %d failures", response.Failed)
	}

	return nil
}

func reportAtomicFailure(response BatchResponse, failedResult BatchResult, failedIndex int, outputJSON bool) error {
	response.Results = append(response.Results, failedResult)
	response.Failed = 1
	response.Succeeded = failedIndex

	output, _ := json.MarshalIndent(map[string]interface{}{
		"error":           "atomic batch failed",
		"failed_at":       failedIndex,
		"failed_item":     failedResult,
		"completed":       failedIndex,
		"total":           response.Total,
		"rolled_back":     failedIndex,
		"partial_results": response.Results[:failedIndex],
	}, "", "  ")
	fmt.Println(string(output))

	return fmt.Errorf("atomic batch failed at item %d: %s", failedIndex, failedResult.Error)
}

// runStreamingBatchAdd processes JSONL input line by line for memory efficiency.
func runStreamingBatchAdd(file string, outputJSON bool, idempotencyKey string) error {
	config, err := LoadConfig()
	if err != nil {
		return err
	}

	api, err := NewAPIClient()
	if err != nil {
		return err
	}

	opts := RequestOptions{IdempotencyKey: idempotencyKey}

	// Get input reader
	var reader io.Reader
	if file != "" {
		f, err := os.Open(file)
		if err != nil {
			return fmt.Errorf("failed to open file: %w", err)
		}
		defer f.Close()
		reader = f
	} else {
		reader = os.Stdin
	}

	scanner := bufio.NewScanner(reader)
	// Increase buffer size for large lines (up to 5MB per line)
	const maxScanTokenSize = 5 * 1024 * 1024
	buf := make([]byte, maxScanTokenSize)
	scanner.Buffer(buf, maxScanTokenSize)

	response := BatchResponse{
		Results: make([]BatchResult, 0),
	}

	lineNum := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue // Skip empty lines
		}

		lineNum++
		response.Total++

		var item CreateKnowledgeRequest
		if err := json.Unmarshal([]byte(line), &item); err != nil {
			result := BatchResult{
				Status: "failed",
				Error:  fmt.Sprintf("line %d: failed to parse JSON: %v", lineNum, err),
			}
			response.Results = append(response.Results, result)
			response.Failed++
			if !outputJSON {
				fmt.Fprintf(os.Stderr, "Line %d: parse error: %v\n", lineNum, err)
			}
			continue
		}

		item.ProjectID = config.ProjectID

		// Validate
		if item.Type == "" {
			result := BatchResult{
				Status: "failed",
				Error:  "type is required",
				Title:  item.Title,
			}
			response.Results = append(response.Results, result)
			response.Failed++
			if !outputJSON {
				fmt.Fprintf(os.Stderr, "Line %d: type is required\n", lineNum)
			}
			continue
		}
		if item.Title == "" {
			result := BatchResult{
				Status: "failed",
				Error:  "title is required",
			}
			response.Results = append(response.Results, result)
			response.Failed++
			if !outputJSON {
				fmt.Fprintf(os.Stderr, "Line %d: title is required\n", lineNum)
			}
			continue
		}
		if item.BodyMD == "" {
			result := BatchResult{
				Status: "failed",
				Error:  "body_md is required",
				Title:  item.Title,
			}
			response.Results = append(response.Results, result)
			response.Failed++
			if !outputJSON {
				fmt.Fprintf(os.Stderr, "Line %d: body_md is required\n", lineNum)
			}
			continue
		}

		resp, err := api.PostWithOptions("/knowledge", item, opts)
		if err != nil {
			result := BatchResult{
				Status: "failed",
				Error:  err.Error(),
				Title:  item.Title,
			}
			response.Results = append(response.Results, result)
			response.Failed++
			if !outputJSON {
				fmt.Fprintf(os.Stderr, "Line %d: %v\n", lineNum, err)
			}
			continue
		}

		var knowledge Knowledge
		if err := json.Unmarshal(resp.Data, &knowledge); err != nil {
			result := BatchResult{
				Status: "failed",
				Error:  fmt.Sprintf("failed to parse response: %v", err),
				Title:  item.Title,
			}
			response.Results = append(response.Results, result)
			response.Failed++
			continue
		}

		response.Results = append(response.Results, BatchResult{
			ID:     knowledge.ID,
			Status: "created",
			Title:  knowledge.Title,
		})
		response.Succeeded++

		if !outputJSON {
			fmt.Printf("Created: %s - %s\n", knowledge.ID, knowledge.Title)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading input: %w", err)
	}

	if response.Total == 0 {
		return fmt.Errorf("no items provided")
	}

	// Output final summary
	if outputJSON {
		output, _ := json.MarshalIndent(response, "", "  ")
		fmt.Println(string(output))
	} else {
		fmt.Printf("\nBatch complete: %d succeeded, %d failed out of %d total\n",
			response.Succeeded, response.Failed, response.Total)
	}

	if response.Failed > 0 {
		return fmt.Errorf("batch completed with %d failures", response.Failed)
	}

	return nil
}
