package cli

import (
	"io"

	"github.com/spf13/cobra"

	"github.com/The-Artificer-of-Ciphers-LLC/gsd-wired/internal/deps"
)

// NewSetupCmd creates the "gsdw setup" subcommand.
func NewSetupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Interactive setup wizard — install missing dependencies",
	}
	return cmd
}

// runSetup is the testable core of the setup wizard.
// Stub: not yet implemented.
func runSetup(in io.Reader, out io.Writer, checkFn func() deps.CheckResult, brewAvailable bool) error {
	return nil
}
