package client

import (
	"github.com/spf13/cobra"
)

// ContextCmd creates the context command with subcommands.
func ContextCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "context",
		Short: "Context retrieval commands",
		Long:  "Commands for retrieving knowledge and asset content (virtual filesystem operations).",
	}

	cmd.AddCommand(OpenCmd())
	cmd.AddCommand(ListCmd())

	return cmd
}
