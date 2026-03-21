package cli

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

// NewBdCmd creates the bd subcommand which passes all arguments through to the bd CLI.
func NewBdCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "bd",
		Short: "Passthrough to bd CLI with GSD context",
		// ArbitraryArgs allows any number of arguments to be passed through
		Args: cobra.ArbitraryArgs,
		// DisableFlagParsing passes all args/flags through to bd unmodified
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			bdPath, err := exec.LookPath("bd")
			if err != nil {
				return fmt.Errorf("bd not found on PATH — install beads first")
			}

			bdCmd := exec.CommandContext(cmd.Context(), bdPath, args...)
			bdCmd.Stdout = os.Stdout
			bdCmd.Stderr = os.Stderr
			bdCmd.Stdin = os.Stdin

			return bdCmd.Run()
		},
	}
}
