package cli

import (
	"io"

	"github.com/spf13/cobra"

	"github.com/The-Artificer-of-Ciphers-LLC/gsd-wired/internal/deps"
)

// NewCheckDepsCmd creates the "gsdw check-deps" subcommand.
func NewCheckDepsCmd() *cobra.Command {
	panic("not implemented")
}

// renderCheckDeps renders the human-readable check-deps output to w.
func renderCheckDeps(w io.Writer, result deps.CheckResult) {
	panic("not implemented")
}

// renderCheckDepsJSON renders the JSON check-deps output to w.
func renderCheckDepsJSON(w io.Writer, result deps.CheckResult) error {
	panic("not implemented")
}
