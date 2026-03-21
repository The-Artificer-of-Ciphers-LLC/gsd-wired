package cli

import (
	"errors"

	"github.com/spf13/cobra"
)

// NewExecuteCmd creates the "gsdw execute" subcommand.
// Wave execution requires Claude Code's Task() tool for parallel subagent orchestration.
// The /gsd-wired:execute slash command (skills/execute/SKILL.md) handles the full flow.
func NewExecuteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "execute",
		Short: "Execute the current wave of unblocked tasks",
		Long: `Execute tasks from the current wave in parallel.

Wave execution requires Claude Code to orchestrate parallel agents via Task(). Use the slash command instead:

    /gsd-wired:execute [phase_number]

This command is a stub that redirects to the SKILL.md slash command.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.New("execution must be run through /gsd-wired:execute slash command (requires Claude Code)")
		},
	}
	return cmd
}
