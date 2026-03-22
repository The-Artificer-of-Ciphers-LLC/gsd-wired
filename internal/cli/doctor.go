package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/The-Artificer-of-Ciphers-LLC/gsd-wired/internal/deps"
)

// NewDoctorCmd creates the "gsdw doctor" subcommand.
// Runs dependency checks plus project health checks (.beads/, .gsdw/).
// Per D-09: doctor is strictly read-only — no file writes, no network calls.
func NewDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check environment and project health",
		Long: `Run a comprehensive health check of your environment and project setup.

Checks dependencies (bd, dolt, Go, container runtime) plus project
initialization state (.beads/ directory and .gsdw/ config).

Doctor is strictly read-only — it will not modify any files.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			result := deps.CheckAll()

			// Locate .beads/ directory (reuse findBeadsDir logic).
			beadsDir, _ := findBeadsDir() // ignore error — we report as WARN

			// Locate .gsdw/ directory by checking cwd and walking up.
			gsdwDir := findGsdwDir()

			renderDoctor(cmd.OutOrStdout(), result, beadsDir, gsdwDir)
			return nil
		},
	}
}

// findGsdwDir locates the .gsdw/ directory by checking cwd and walking up.
// Returns the path if found, or empty string if not found.
func findGsdwDir() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}

	dir := cwd
	for {
		candidate := filepath.Join(dir, ".gsdw")
		if info, statErr := os.Stat(candidate); statErr == nil && info.IsDir() {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

// renderDoctor renders the human-readable doctor output to w.
// beadsDir is the resolved path to .beads/ (empty string if not found).
// gsdwDir is the resolved path to .gsdw/ (empty string if not found).
//
// Output format:
//
//	Dependencies:
//	  [OK]   Go 1.22.4
//	  [FAIL] dolt not found
//	         Install: brew install dolthub/tap/dolt
//
//	Project:
//	  [OK]   .beads/ found at /path/to/.beads
//	  [WARN] .gsdw/ not found — run gsdw init first
func renderDoctor(w io.Writer, result deps.CheckResult, beadsDir, gsdwDir string) {
	fmt.Fprintln(w, "Dependencies:")
	for _, d := range result.Deps {
		switch d.Status {
		case deps.StatusOK:
			fmt.Fprintf(w, "  [OK]   %s %s\n", d.Name, d.Version)
		case deps.StatusWarn:
			fmt.Fprintf(w, "  [WARN] %s %s\n", d.Name, d.Version)
		case deps.StatusFail:
			fmt.Fprintf(w, "  [FAIL] %s not found\n", d.Name)
			if d.InstallHelp != "" {
				fmt.Fprintf(w, "         Install: %s\n", d.InstallHelp)
			}
		}
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "Project:")

	// .beads/ directory check
	if beadsDir != "" {
		fmt.Fprintf(w, "  [OK]   .beads/ found at %s\n", beadsDir)
	} else {
		fmt.Fprintf(w, "  [WARN] .beads/ not found — run gsdw init first\n")
	}

	// .gsdw/ config check
	if gsdwDir != "" {
		fmt.Fprintf(w, "  [OK]   .gsdw/ found at %s\n", gsdwDir)
	} else {
		fmt.Fprintf(w, "  [WARN] .gsdw/ not found — run gsdw init first\n")
	}
}
