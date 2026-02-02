package client

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

// Knowledge represents a knowledge item from the API.
type Knowledge struct {
	ID        string `json:"id"`
	OrgID     string `json:"org_id"`
	ProjectID string `json:"project_id"`
	Type      string `json:"type"`
	Status    string `json:"status"`
	Title     string `json:"title"`
	Summary   string `json:"summary"`
	BodyMD    string `json:"body_md"`
	Scope     string `json:"scope"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// GetCmd creates the get command.
func GetCmd() *cobra.Command {
	var searchID string

	cmd := &cobra.Command{
		Use:     "get <knowledge_id>",
		Short:   "Get a knowledge item by ID",
		Long:    "Retrieves a knowledge item by its ID and displays the full content.",
		Aliases: []string{"view"},
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			outputJSON, _ := cmd.Flags().GetBool("output")
			return runGet(args[0], outputJSON, searchID)
		},
	}

	cmd.Flags().StringVar(&searchID, "search-id", "", "Associate this selection with a search ID")

	return cmd
}

func runGet(knowledgeID string, outputJSON bool, searchID string) error {
	// Create API client
	api, err := NewAPIClient()
	if err != nil {
		return err
	}

	// Fetch knowledge
	resp, err := api.Get(fmt.Sprintf("/knowledge/%s", knowledgeID))
	if err != nil {
		return fmt.Errorf("failed to get knowledge: %w", err)
	}

	// Parse knowledge
	var knowledge Knowledge
	if err := json.Unmarshal(resp.Data, &knowledge); err != nil {
		return fmt.Errorf("failed to parse knowledge: %w", err)
	}

	_ = sendSearchFeedback(api, searchID, knowledgeID, "knowledge")

	if outputJSON {
		output, _ := json.MarshalIndent(knowledge, "", "  ")
		fmt.Println(string(output))
	} else {
		fmt.Printf("Title: %s\n", knowledge.Title)
		fmt.Printf("Type: %s\n", knowledge.Type)
		fmt.Printf("Status: %s\n", knowledge.Status)
		if knowledge.Scope != "" {
			fmt.Printf("Scope: %s\n", knowledge.Scope)
		}
		if knowledge.Summary != "" {
			fmt.Printf("Summary: %s\n", knowledge.Summary)
		}
		fmt.Printf("Created: %s\n", knowledge.CreatedAt)
		fmt.Printf("Updated: %s\n", knowledge.UpdatedAt)
		fmt.Println()
		fmt.Println("--- Content ---")
		fmt.Println(knowledge.BodyMD)
	}

	return nil
}
