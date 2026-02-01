package client

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

type DeleteRequest struct {
	ID string `json:"id"`
}

func DeleteCmd() *cobra.Command {
	var (
		file           string
		batch          bool
		atomic         bool
		idempotencyKey string
	)

	cmd := &cobra.Command{
		Use:   "delete [id]",
		Short: "Delete (deprecate) knowledge by ID",
		Long: `Delete (deprecate) knowledge items by ID.

Examples:
  # Delete single knowledge item
  neotex delete <knowledge_id>

  # Batch delete from JSON array of IDs
  echo '["id1","id2","id3"]' | neotex delete --batch

  # Batch delete from file
  neotex delete --batch --file ids.json

  # Atomic batch delete (all-or-nothing)
  neotex delete --batch --atomic --file ids.json`,
		Args: func(cmd *cobra.Command, args []string) error {
			batchFlag, _ := cmd.Flags().GetBool("batch")
			if batchFlag {
				return nil
			}
			if len(args) != 1 {
				return fmt.Errorf("requires exactly 1 argument (knowledge_id) or use --batch flag")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			outputJSON, _ := cmd.Flags().GetBool("output")
			if batch {
				return runBatchDelete(file, outputJSON, atomic, idempotencyKey)
			}
			return runDelete(args[0], outputJSON, idempotencyKey)
		},
	}

	cmd.Flags().StringVarP(&file, "file", "f", "", "Input file with JSON array of IDs")
	cmd.Flags().BoolVar(&batch, "batch", false, "Enable batch mode (expects JSON array of IDs)")
	cmd.Flags().BoolVar(&atomic, "atomic", false, "Atomic mode: all-or-nothing (only with --batch)")
	cmd.Flags().StringVar(&idempotencyKey, "idempotency-key", "", "Idempotency key for safe retries")

	return cmd
}

func runDelete(knowledgeID string, outputJSON bool, idempotencyKey string) error {
	api, err := NewAPIClient()
	if err != nil {
		return err
	}

	opts := RequestOptions{IdempotencyKey: idempotencyKey}
	resp, err := api.DeleteWithOptions(fmt.Sprintf("/knowledge/%s", knowledgeID), opts)
	if err != nil {
		return fmt.Errorf("failed to delete knowledge: %w", err)
	}

	var knowledge Knowledge
	if err := json.Unmarshal(resp.Data, &knowledge); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if outputJSON {
		output, _ := json.MarshalIndent(map[string]interface{}{
			"id":     knowledge.ID,
			"status": "deprecated",
			"title":  knowledge.Title,
		}, "", "  ")
		fmt.Println(string(output))
	} else {
		fmt.Printf("Deprecated knowledge: %s\n", knowledge.ID)
		fmt.Printf("Title: %s\n", knowledge.Title)
	}

	return nil
}

func runBatchDelete(file string, outputJSON, atomic bool, idempotencyKey string) error {
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

	var ids []string
	if err := json.Unmarshal(input, &ids); err != nil {
		return fmt.Errorf("failed to parse JSON array: %w - batch mode expects a JSON array of strings", err)
	}

	if len(ids) == 0 {
		return fmt.Errorf("empty batch: no IDs provided")
	}

	if len(ids) > maxBatchSize {
		return fmt.Errorf("batch size %d exceeds maximum of %d items", len(ids), maxBatchSize)
	}

	response := BatchResponse{
		Results: make([]BatchResult, 0, len(ids)),
		Total:   len(ids),
	}

	for i, id := range ids {
		if id == "" {
			result := BatchResult{
				Status: "failed",
				Error:  "empty ID",
			}
			if atomic {
				return reportDeleteAtomicFailure(response, result, i, outputJSON)
			}
			response.Results = append(response.Results, result)
			response.Failed++
			continue
		}

		resp, err := api.DeleteWithOptions(fmt.Sprintf("/knowledge/%s", id), opts)
		if err != nil {
			result := BatchResult{
				ID:     id,
				Status: "failed",
				Error:  err.Error(),
			}
			if atomic {
				return reportDeleteAtomicFailure(response, result, i, outputJSON)
			}
			response.Results = append(response.Results, result)
			response.Failed++
			continue
		}

		var knowledge Knowledge
		if err := json.Unmarshal(resp.Data, &knowledge); err != nil {
			result := BatchResult{
				ID:     id,
				Status: "failed",
				Error:  fmt.Sprintf("failed to parse response: %v", err),
			}
			if atomic {
				return reportDeleteAtomicFailure(response, result, i, outputJSON)
			}
			response.Results = append(response.Results, result)
			response.Failed++
			continue
		}

		response.Results = append(response.Results, BatchResult{
			ID:     knowledge.ID,
			Status: "deprecated",
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

func reportDeleteAtomicFailure(response BatchResponse, failedResult BatchResult, failedIndex int, outputJSON bool) error {
	response.Results = append(response.Results, failedResult)
	response.Failed = 1
	response.Succeeded = failedIndex

	output, _ := json.MarshalIndent(map[string]interface{}{
		"error":           "atomic batch failed",
		"failed_at":       failedIndex,
		"failed_item":     failedResult,
		"completed":       failedIndex,
		"total":           response.Total,
		"note":            "previous deletions cannot be rolled back",
		"partial_results": response.Results[:failedIndex],
	}, "", "  ")
	fmt.Println(string(output))

	return fmt.Errorf("atomic batch failed at item %d: %s", failedIndex, failedResult.Error)
}
