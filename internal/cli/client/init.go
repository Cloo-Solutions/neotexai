package client

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

const (
	neotexDir    = ".neotex"
	configFile   = "config.yaml"
	manifestFile = "index.json"
	envFile      = ".env"
)

type Config struct {
	ProjectID string `json:"project_id" yaml:"project_id"`
}

func InitCmd() *cobra.Command {
	var projectName string
	var apiKey string
	var apiURL string

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a neotex project",
		Long:  "Creates the .neotex/ directory, config.yaml, and .env with API key.",
		RunE: func(cmd *cobra.Command, args []string) error {
			outputJSON, _ := cmd.Flags().GetBool("output")
			return runInit(projectName, apiKey, apiURL, outputJSON)
		},
	}

	cmd.Flags().StringVar(&projectName, "project", "", "Project name (auto-generated from directory name if not provided)")
	cmd.Flags().StringVar(&apiKey, "api-key", "", "API key for authentication")
	cmd.Flags().StringVar(&apiURL, "api-url", "", "API base URL (default: http://localhost:8080)")

	return cmd
}

func runInit(projectName, apiKey, apiURL string, outputJSON bool) error {
	if _, err := os.Stat(neotexDir); err == nil {
		return fmt.Errorf(".neotex directory already exists")
	}

	_ = godotenv.Load()
	if apiKey == "" {
		apiKey = os.Getenv(envAPIKey)
	}
	if apiKey == "" {
		fmt.Print("Enter API key: ")
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read API key: %w", err)
		}
		apiKey = strings.TrimSpace(input)
		if apiKey == "" {
			return fmt.Errorf("API key is required")
		}
	}

	if apiURL == "" {
		apiURL = os.Getenv(envAPIURL)
	}
	if apiURL == "" {
		apiURL = defaultAPIURL
	}

	if projectName == "" {
		cwd, _ := os.Getwd()
		projectName = filepath.Base(cwd)
	}

	envData := fmt.Sprintf("NEOTEX_API_KEY=%s\nNEOTEX_API_URL=%s\n", apiKey, apiURL)
	if err := os.WriteFile(envFile, []byte(envData), 0600); err != nil {
		return fmt.Errorf("failed to create .env: %w", err)
	}

	api, err := NewAPIClientWithConfig(apiKey, apiURL)
	if err != nil {
		os.Remove(envFile)
		return fmt.Errorf("failed to create API client: %w", err)
	}

	resp, err := api.Post("/projects", map[string]string{"name": projectName})
	if err != nil {
		os.Remove(envFile)
		return fmt.Errorf("failed to create project: %w", err)
	}

	var project struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(resp.Data, &project); err != nil {
		os.Remove(envFile)
		return fmt.Errorf("failed to parse project response: %w", err)
	}

	if err := os.MkdirAll(neotexDir, 0755); err != nil {
		return fmt.Errorf("failed to create .neotex directory: %w", err)
	}

	configPath := filepath.Join(neotexDir, configFile)
	configData := fmt.Sprintf("project_id: %s\nproject_name: %s\n", project.ID, project.Name)
	if err := os.WriteFile(configPath, []byte(configData), 0644); err != nil {
		return fmt.Errorf("failed to create config.yaml: %w", err)
	}

	if outputJSON {
		result := map[string]interface{}{
			"success":      true,
			"project_id":   project.ID,
			"project_name": project.Name,
			"config":       configPath,
			"env":          envFile,
		}
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
	} else {
		fmt.Printf("Initialized neotex project '%s'\n", project.Name)
		fmt.Printf("Project ID: %s\n", project.ID)
		fmt.Printf("Config saved to %s\n", configPath)
	}

	return nil
}

// LoadConfig reads the config from .neotex/config.yaml.
func LoadConfig() (*Config, error) {
	configPath := filepath.Join(neotexDir, configFile)
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("not a neotex project (run 'neotex init' first)")
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	// Simple YAML parsing for single field
	var config Config
	for _, line := range splitLines(string(data)) {
		if len(line) > 12 && line[:12] == "project_id: " {
			config.ProjectID = line[12:]
			break
		}
	}

	if config.ProjectID == "" {
		return nil, fmt.Errorf("invalid config: project_id not found")
	}

	return &config, nil
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
