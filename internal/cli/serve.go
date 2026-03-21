package cli

import (
	"github.com/spf13/cobra"

	"github.com/The-Artificer-of-Ciphers-LLC/gsd-wired/internal/mcp"
)

// NewServeCmd creates the serve subcommand which starts the MCP stdio server.
func NewServeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Start MCP stdio server",
		// CRITICAL: No output to stdout before or after mcp.Serve —
		// the SDK owns stdout entirely in this mode (D-15).
		RunE: func(cmd *cobra.Command, args []string) error {
			return mcp.Serve(cmd.Context())
		},
	}
}
