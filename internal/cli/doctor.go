package cli

import (
	"io"

	"github.com/spf13/cobra"

	"github.com/The-Artificer-of-Ciphers-LLC/gsd-wired/internal/deps"
)

// NewDoctorCmd creates the "gsdw doctor" subcommand.
func NewDoctorCmd() *cobra.Command {
	panic("not implemented")
}

// renderDoctor renders the human-readable doctor output to w.
// beadsDir is the path to .beads/ (empty string if not found).
// gsdwDir is the path to .gsdw/ (empty string if not found).
func renderDoctor(w io.Writer, result deps.CheckResult, beadsDir, gsdwDir string) {
	panic("not implemented")
}
