// Package cli provides shared CLI utilities for neotex and neotexd.
package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// FlagSchema represents the JSON schema for a command flag.
type FlagSchema struct {
	Name        string `json:"name"`
	Shorthand   string `json:"shorthand,omitempty"`
	Type        string `json:"type"`
	Default     string `json:"default,omitempty"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required"`
}

// CommandSchema represents the JSON schema for a command.
type CommandSchema struct {
	Name        string          `json:"name"`
	Use         string          `json:"use,omitempty"`
	Description string          `json:"description,omitempty"`
	Long        string          `json:"long,omitempty"`
	Flags       []FlagSchema    `json:"flags,omitempty"`
	Subcommands []CommandSchema `json:"subcommands,omitempty"`
}

// GenerateSchema generates a JSON schema for a cobra command.
func GenerateSchema(cmd *cobra.Command) CommandSchema {
	schema := CommandSchema{
		Name:        cmd.Name(),
		Use:         cmd.Use,
		Description: cmd.Short,
		Long:        cmd.Long,
		Flags:       extractFlags(cmd),
	}

	for _, sub := range cmd.Commands() {
		if sub.Name() == "help" || sub.Hidden {
			continue
		}
		schema.Subcommands = append(schema.Subcommands, GenerateSchema(sub))
	}

	return schema
}

func extractFlags(cmd *cobra.Command) []FlagSchema {
	var flags []FlagSchema

	cmd.LocalFlags().VisitAll(func(f *pflag.Flag) {
		if f.Name == "help-json" || f.Name == "help" {
			return
		}
		flags = append(flags, flagToSchema(f, cmd))
	})

	return flags
}

func flagToSchema(f *pflag.Flag, cmd *cobra.Command) FlagSchema {
	schema := FlagSchema{
		Name:        f.Name,
		Shorthand:   f.Shorthand,
		Type:        f.Value.Type(),
		Default:     f.DefValue,
		Description: f.Usage,
		Required:    false,
	}

	if ann := cmd.Annotations; ann != nil {
		if _, ok := ann[cobra.BashCompOneRequiredFlag]; ok {
			schema.Required = true
		}
	}

	return schema
}

// PrintSchema outputs the command schema as JSON and exits.
func PrintSchema(cmd *cobra.Command) {
	schema := GenerateSchema(cmd)
	output, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating schema: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(output))
	os.Exit(0)
}

// AddHelpJSONFlag adds the --help-json flag to a command.
func AddHelpJSONFlag(cmd *cobra.Command) {
	cmd.PersistentFlags().Bool("help-json", false, "Output command schema as JSON")
}

// CheckHelpJSON checks os.Args for --help-json and outputs schema if found.
// Call this before cmd.Execute() to handle the flag before arg validation.
func CheckHelpJSON(rootCmd *cobra.Command) {
	for i, arg := range os.Args {
		if arg == "--help-json" {
			targetCmd := findTargetCommand(rootCmd, os.Args[1:i])
			PrintSchema(targetCmd)
		}
	}
}

func findTargetCommand(cmd *cobra.Command, args []string) *cobra.Command {
	if len(args) == 0 {
		return cmd
	}

	for _, sub := range cmd.Commands() {
		if sub.Name() == args[0] || sub.HasAlias(args[0]) {
			return findTargetCommand(sub, args[1:])
		}
	}

	return cmd
}
