package cli

import (
	"errors"

	"github.com/spf13/cobra"
)

// NewVerifyCmd creates the "gsdw verify" subcommand.
// Phase verification requires Claude Code to run codebase checks and evaluate acceptance criteria.
// The /gsd-wired:verify slash command (skills/verify/SKILL.md) handles the full flow.
func NewVerifyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify acceptance criteria for a completed phase",
		Long: `Verify that acceptance criteria stored in beads are satisfied by the current codebase.

Verification requires Claude Code to run codebase checks (file existence, go test, manual review)
and evaluate results against success criteria. Use the slash command instead:

    /gsd-wired:verify [phase_number]

This command is a stub that redirects to the SKILL.md slash command.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.New("verification must be run through /gsd-wired:verify slash command (requires Claude Code)")
		},
	}
	return cmd
}
