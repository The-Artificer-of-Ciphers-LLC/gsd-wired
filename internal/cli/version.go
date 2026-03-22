package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/The-Artificer-of-Ciphers-LLC/gsd-wired/internal/version"
)

// NewVersionCmd creates the version subcommand with optional --json flag.
func NewVersionCmd() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		RunE: func(cmd *cobra.Command, args []string) error {
			info := version.GetInfo()
			if jsonOutput {
				out, err := info.JSON()
				if err != nil {
					return fmt.Errorf("version --json: %w", err)
				}
				fmt.Fprintln(cmd.OutOrStdout(), out)
			} else {
				// NOTE: version output to stdout is intentional — it's not MCP mode
				fmt.Fprintln(cmd.OutOrStdout(), info.String())
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output version information as JSON")
	return cmd
}
