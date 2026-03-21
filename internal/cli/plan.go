package cli

import (
	"errors"

	"github.com/spf13/cobra"
)

// NewPlanCmd creates the "gsdw plan" subcommand.
// Planning orchestration requires Claude Code's Task() tool for parallel subagents.
// The /gsd-wired:plan slash command (skills/plan/SKILL.md) handles the full flow.
func NewPlanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plan",
		Short: "Create phase plan with dependencies",
		Long: `Create a dependency-aware phase plan from research results.

Planning requires Claude Code to orchestrate the full flow including plan generation,
dependency wiring, and inline requirement coverage validation. Use the slash command instead:

    /gsd-wired:plan [phase_number]

This command is a stub that redirects to the SKILL.md slash command.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.New("planning must be run through /gsd-wired:plan slash command (requires Claude Code)")
		},
	}
	return cmd
}
