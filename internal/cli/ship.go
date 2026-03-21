package cli

import (
	"errors"

	"github.com/spf13/cobra"
)

// NewShipCmd creates the "gsdw ship" subcommand.
// Shipping requires Claude Code for PR creation orchestration.
// The /gsd-wired:ship slash command (skills/ship/SKILL.md) handles the full flow.
func NewShipCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ship",
		Short: "Create PR and advance to next phase",
		Long: `Ship the current phase: create a PR with bead-sourced summary and advance phase state.

Shipping requires Claude Code for PR creation orchestration. Use the slash command instead:

    /gsd-wired:ship [phase_number]

This command is a stub that redirects to the SKILL.md slash command.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.New("shipping must be run through /gsd-wired:ship slash command (requires Claude Code)")
		},
	}
	return cmd
}
