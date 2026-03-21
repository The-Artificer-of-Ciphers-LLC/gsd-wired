package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/The-Artificer-of-Ciphers-LLC/gsd-wired/internal/version"
)

// NewVersionCmd creates the version subcommand.
func NewVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		RunE: func(cmd *cobra.Command, args []string) error {
			// NOTE: version output to stdout is intentional — it's not MCP mode
			fmt.Fprintln(cmd.OutOrStdout(), version.String())
			return nil
		},
	}
}
