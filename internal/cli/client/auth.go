package client

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// AuthCmd creates the auth parent command
func AuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage authentication credentials",
		Long:  "Login, logout, and check authentication status for neotex CLI",
	}

	cmd.AddCommand(AuthLoginCmd())
	cmd.AddCommand(AuthLogoutCmd())
	cmd.AddCommand(AuthStatusCmd())

	return cmd
}

// AuthLoginCmd creates the auth login command
func AuthLoginCmd() *cobra.Command {
	var apiKey string
	var apiURL string

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Login with API key",
		Long:  "Store API key and URL in global config (~/.config/neotex/config.json)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAuthLogin(apiKey, apiURL)
		},
	}

	cmd.Flags().StringVar(&apiKey, "api-key", "", "API key (ntx_...)")
	cmd.Flags().StringVar(&apiURL, "url", "http://localhost:8080", "API URL")

	return cmd
}

// AuthLogoutCmd creates the auth logout command
func AuthLogoutCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Logout and clear credentials",
		Long:  "Remove stored credentials from global config",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAuthLogout()
		},
	}

	return cmd
}

// AuthStatusCmd creates the auth status command
func AuthStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show authentication status",
		Long:  "Display current authentication source and credentials",
		RunE: func(cmd *cobra.Command, args []string) error {
			outputJSON, _ := cmd.Flags().GetBool("output")
			return runAuthStatus(outputJSON)
		},
	}

	cmd.Flags().Bool("output", false, "Output as JSON")

	return cmd
}

func runAuthLogin(apiKey, apiURL string) error {
	if apiKey == "" {
		fmt.Print("Enter API key: ")
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read API key: %w", err)
		}
		apiKey = strings.TrimSpace(input)
	}

	if !IsValidAPIKey(apiKey) {
		return fmt.Errorf("invalid API key format (expected: ntx_ + 64 hex characters)")
	}

	config := &GlobalConfig{
		APIKey: apiKey,
		APIURL: apiURL,
	}

	if err := SaveGlobalConfig(config); err != nil {
		return fmt.Errorf("failed to save credentials: %w", err)
	}

	fmt.Println("Successfully logged in")
	return nil
}

func runAuthLogout() error {
	if err := DeleteGlobalConfig(); err != nil {
		return fmt.Errorf("failed to logout: %w", err)
	}

	fmt.Println("Successfully logged out")
	return nil
}

func runAuthStatus(outputJSON bool) error {
	source, apiKey, apiURL := GetCredentialSource("", "")

	if outputJSON {
		return outputStatusJSON(source, apiKey, apiURL)
	}

	return outputStatusText(source, apiKey, apiURL)
}

func outputStatusJSON(source CredentialSource, apiKey, apiURL string) error {
	status := map[string]interface{}{
		"authenticated": source != SourceNone,
		"source":        string(source),
	}

	if source != SourceNone {
		status["api_key"] = maskAPIKey(apiKey)
		status["api_url"] = apiURL
	}

	data, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal status: %w", err)
	}

	fmt.Println(string(data))
	return nil
}

func outputStatusText(source CredentialSource, apiKey, apiURL string) error {
	if source == SourceNone {
		fmt.Println("Not authenticated")
		fmt.Println("Run 'neotex auth login' to authenticate")
		return nil
	}

	fmt.Printf("Authenticated: yes\n")
	fmt.Printf("Source: %s\n", source)
	fmt.Printf("API Key: %s\n", maskAPIKey(apiKey))
	fmt.Printf("API URL: %s\n", apiURL)

	return nil
}

func maskAPIKey(key string) string {
	if len(key) < 8 {
		return "***"
	}
	return key[:7] + "..." + key[len(key)-4:]
}
