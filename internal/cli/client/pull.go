package client

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// ManifestItem represents a knowledge item in the manifest.
type ManifestItem struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Summary string `json:"summary"`
	Type    string `json:"type"`
	Scope   string `json:"scope,omitempty"`
}

// Manifest represents the full manifest response.
type Manifest struct {
	Manifest []ManifestItem `json:"manifest"`
}

// PullCmd creates the pull command.
func PullCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pull",
		Short: "Download the knowledge manifest",
		Long:  "Downloads the knowledge manifest from the API and saves it to .neotex/index.json.",
		RunE: func(cmd *cobra.Command, args []string) error {
			outputJSON, _ := cmd.Flags().GetBool("output")
			return runPull(outputJSON)
		},
	}

	return cmd
}

func runPull(outputJSON bool) error {
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

	// Fetch manifest
	path := "/context"
	if config.ProjectID != "" {
		path = fmt.Sprintf("/context?project_id=%s", config.ProjectID)
	}

	resp, err := api.Get(path)
	if err != nil {
		return fmt.Errorf("failed to fetch manifest: %w", err)
	}

	// Parse manifest
	var manifest Manifest
	if err := json.Unmarshal(resp.Data, &manifest); err != nil {
		return fmt.Errorf("failed to parse manifest: %w", err)
	}

	// Save to .neotex/index.json
	manifestPath := filepath.Join(neotexDir, manifestFile)
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	if err := os.WriteFile(manifestPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	if outputJSON {
		result := map[string]interface{}{
			"success": true,
			"items":   len(manifest.Manifest),
			"path":    manifestPath,
		}
		output, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(output))
	} else {
		fmt.Printf("Downloaded %d knowledge items to %s\n", len(manifest.Manifest), manifestPath)
	}

	return nil
}

// LoadManifest reads the manifest from .neotex/index.json.
func LoadManifest() (*Manifest, error) {
	manifestPath := filepath.Join(neotexDir, manifestFile)
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("manifest not found (run 'neotex pull' first)")
		}
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	return &manifest, nil
}
