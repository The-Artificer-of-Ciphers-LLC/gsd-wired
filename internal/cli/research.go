package cli

import (
	"errors"

	"github.com/spf13/cobra"
)

// NewResearchCmd creates the "gsdw research" subcommand.
// Research orchestration requires Claude Code's Task() tool for parallel subagents.
// The /gsd-wired:research slash command (skills/research/SKILL.md) handles the full flow.
func NewResearchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "research",
		Short: "Run research phase",
		Long: `Run the research phase for the current project.

Research orchestration requires Claude Code's Task() tool to spawn parallel
subagents (stack, features, architecture, pitfalls). Use the slash command instead:

    /gsd-wired:research [phase_number]

This command is a stub that redirects to the SKILL.md slash command.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.New("research phase must be run through /gsd-wired:research slash command (requires Claude Code)")
		},
	}
	return cmd
}
