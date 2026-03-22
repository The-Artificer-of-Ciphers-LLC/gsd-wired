package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/The-Artificer-of-Ciphers-LLC/gsd-wired/internal/connection"
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

			// Load connection config for doctor display.
			var connCfg *connection.Config
			var connHealthErr error
			if gsdwDir != "" {
				connCfg, _ = connection.LoadConnection(gsdwDir)
				if connCfg != nil {
					host, port := connCfg.ActiveHostPort()
					user := connCfg.Remote.User
					if connCfg.ActiveMode == "local" && user == "" {
						user = "root" // Dolt container default
					}
					connHealthErr = connection.CheckConnectivity(host, port, user, os.Getenv("GSDW_DB_PASSWORD"), 2*time.Second)
				}
			}

			renderDoctor(cmd.OutOrStdout(), result, beadsDir, gsdwDir, connCfg, connHealthErr)
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
// connCfg is the loaded connection config (nil if not configured).
// connHealthErr is the result of CheckConnectivity (nil if healthy, error if unreachable).
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
//
//	Connection:
//	  Mode:    local
//	  Address: 127.0.0.1:3307
//	  [OK]   Dolt server responding
func renderDoctor(w io.Writer, result deps.CheckResult, beadsDir, gsdwDir string, connCfg *connection.Config, connHealthErr error) {
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

	// Connection section (D-08).
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Connection:")
	if connCfg == nil {
		fmt.Fprintln(w, "  [WARN] Not configured — run gsdw connect")
	} else {
		host, port := connCfg.ActiveHostPort()
		fmt.Fprintf(w, "  Mode:    %s\n", connCfg.ActiveMode)
		fmt.Fprintf(w, "  Address: %s:%s\n", host, port)
		if connHealthErr != nil {
			fmt.Fprintf(w, "  [FAIL] Dolt unreachable: %v\n", connHealthErr)
		} else {
			fmt.Fprintln(w, "  [OK]   Dolt server responding")
		}
	}
}
