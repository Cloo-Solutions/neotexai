package main

import (
	"fmt"
	"os"

	"github.com/cloo-solutions/neotexai/internal/cli"
	"github.com/cloo-solutions/neotexai/internal/cli/admin"
	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "neotexd",
		Short: "Neotex daemon and CLI",
		Long:  "Neotex daemon for running the API server and managing organizations and API keys",
	}

	cli.AddHelpJSONFlag(rootCmd)
	rootCmd.AddCommand(admin.ServeCmd())
	rootCmd.AddCommand(admin.OrgCmd())
	rootCmd.AddCommand(admin.APIKeyCmd())

	if len(os.Args) == 1 {
		os.Args = append(os.Args, "serve")
	}

	cli.CheckHelpJSON(rootCmd)
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
