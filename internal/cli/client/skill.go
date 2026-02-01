package client

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// Embedded skill files from packages/skill/ at build time.
// This keeps the npm package and CLI in sync.
//
//go:embed skill_embed.md
var skillContent string

//go:embed skill_init_embed.md
var skillInitContent string

// skills defines the skills to install
var skills = []struct {
	name    string
	content *string
}{
	{"neotex", &skillContent},
	{"neotex-init", &skillInitContent},
}

// SkillCmd creates the skill command group.
func SkillCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skill",
		Short: "Manage Claude Code skill integration",
		Long:  "Commands for managing the neotex skill for Claude Code.",
	}

	cmd.AddCommand(skillInitCmd())

	return cmd
}

func skillInitCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Install neotex skills for Claude Code",
		Long:  "Installs neotex skills to ~/.claude/skills/ for Claude Code integration.",
		RunE: func(cmd *cobra.Command, args []string) error {
			outputJSON, _ := cmd.Flags().GetBool("output")
			return runSkillInit(force, outputJSON)
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Overwrite existing skills if present")

	return cmd
}

func runSkillInit(force, outputJSON bool) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	skillsBase := filepath.Join(home, ".claude", "skills")
	var installed []string

	for _, skill := range skills {
		skillDir := filepath.Join(skillsBase, skill.name)
		skillPath := filepath.Join(skillDir, "SKILL.md")

		// Check if skill already exists
		if _, err := os.Stat(skillPath); err == nil && !force {
			if !outputJSON {
				fmt.Printf("Skill %s already exists (use --force to overwrite)\n", skill.name)
			}
			continue
		}

		// Create skill directory
		if err := os.MkdirAll(skillDir, 0755); err != nil {
			return fmt.Errorf("failed to create skill directory %s: %w", skill.name, err)
		}

		// Write skill file
		if err := os.WriteFile(skillPath, []byte(*skill.content), 0644); err != nil {
			return fmt.Errorf("failed to write skill file %s: %w", skill.name, err)
		}

		installed = append(installed, skill.name)
	}

	if outputJSON {
		result := map[string]any{
			"success":   true,
			"installed": installed,
			"message":   "Neotex skills installed for Claude Code",
		}
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
	} else {
		if len(installed) > 0 {
			fmt.Printf("Installed skills: %v\n", installed)
			fmt.Println("Skills are now available in Claude Code sessions.")
		} else {
			fmt.Println("All skills already installed (use --force to overwrite).")
		}
	}

	return nil
}
