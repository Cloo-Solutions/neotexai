package main

import (
	"fmt"
	"os"

	"github.com/cloo-solutions/neotexai/internal/cli"
	"github.com/cloo-solutions/neotexai/internal/cli/client"
	"github.com/spf13/cobra"
)

var version = "dev"

func main() {
	rootCmd := &cobra.Command{
		Use:   "neotex",
		Short: "Neotex CLI - Knowledge management for AI agents",
		Long: `Neotex CLI provides commands to manage knowledge for AI agents.

Environment variables:
  NEOTEX_API_KEY   API key for authentication (required)
  NEOTEX_API_URL   API base URL (default: http://localhost:8080)`,
		Version: version,
	}

	rootCmd.PersistentFlags().Bool("output", false, "Output as JSON")
	rootCmd.PersistentFlags().String("api-key", "", "API key for authentication (overrides env and config)")
	rootCmd.PersistentFlags().String("api-url", "", "API base URL (overrides env and config)")
	cli.AddHelpJSONFlag(rootCmd)

	rootCmd.AddCommand(client.InitCmd())
	rootCmd.AddCommand(client.PullCmd())
	rootCmd.AddCommand(client.SearchCmd())
	rootCmd.AddCommand(client.GetCmd())
	rootCmd.AddCommand(client.AddCmd())
	rootCmd.AddCommand(client.DeleteCmd())
	rootCmd.AddCommand(client.AssetCmd())
	rootCmd.AddCommand(client.EvalCmd())
	rootCmd.AddCommand(client.AuthCmd())

	cli.CheckHelpJSON(rootCmd)
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
