package cli

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"github.com/The-Artificer-of-Ciphers-LLC/gsd-wired/internal/hook"
)

// NewHookCmd creates the hook subcommand which dispatches a hook event.
func NewHookCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "hook",
		Short: "Dispatch a hook event",
		// Requires exactly one arg: the hook event name (e.g., SessionStart)
		// This prevents panic on empty input (Pitfall 6 in RESEARCH.md)
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			err := hook.Dispatch(args[0], os.Stdin, os.Stdout)
			if err != nil {
				slog.Error("hook dispatch failed", "event", args[0], "err", err)
				os.Exit(2) // exit 2 = show stderr to user per hook protocol
			}
			return nil
		},
	}
}
