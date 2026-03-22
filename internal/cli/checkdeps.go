package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/The-Artificer-of-Ciphers-LLC/gsd-wired/internal/deps"
)

// NewCheckDepsCmd creates the "gsdw check-deps" subcommand.
// Checks bd, dolt, Go, and container runtime, printing status with install help for missing deps.
func NewCheckDepsCmd() *cobra.Command {
	var jsonMode bool

	cmd := &cobra.Command{
		Use:   "check-deps",
		Short: "Check required dependencies (bd, dolt, Go, container runtime)",
		Long: `Check whether required dependencies are installed and reachable.

Reports [OK], [WARN], or [FAIL] for each dependency and prints actionable
install instructions for any that are missing.

Use --json for machine-readable output.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			result := deps.CheckAll()
			if jsonMode {
				return renderCheckDepsJSON(cmd.OutOrStdout(), result)
			}
			renderCheckDeps(cmd.OutOrStdout(), result)
			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonMode, "json", false, "Output structured JSON instead of human-readable text")

	return cmd
}

// renderCheckDeps renders the human-readable check-deps output to w.
// Format per plan spec:
//
//	[OK]   Go 1.22.4 (/usr/local/go/bin/go)
//	[OK]   bd 1.4.2 (/opt/homebrew/bin/bd)
//	[FAIL] dolt not found
//	       Install: brew install dolthub/tap/dolt
func renderCheckDeps(w io.Writer, result deps.CheckResult) {
	for _, d := range result.Deps {
		switch d.Status {
		case deps.StatusOK:
			fmt.Fprintf(w, "[OK]   %s %s (%s)\n", d.Name, d.Version, d.Path)
		case deps.StatusWarn:
			fmt.Fprintf(w, "[WARN] %s %s (%s)\n", d.Name, d.Version, d.Path)
		case deps.StatusFail:
			fmt.Fprintf(w, "[FAIL] %s not found\n", d.Name)
			if d.InstallHelp != "" {
				fmt.Fprintf(w, "       Install: %s\n", d.InstallHelp)
			}
		}
	}
}

// jsonCheckResult is the JSON shape for check-deps --json output.
type jsonCheckResult struct {
	AllOK bool      `json:"allOK"`
	Deps  []jsonDep `json:"deps"`
}

// jsonDep is the JSON shape for a single dependency.
type jsonDep struct {
	Name        string `json:"name"`
	Binary      string `json:"binary"`
	Status      string `json:"status"`
	Version     string `json:"version,omitempty"`
	Path        string `json:"path,omitempty"`
	InstallHelp string `json:"installHelp,omitempty"`
}

// renderCheckDepsJSON renders structured JSON output to w.
func renderCheckDepsJSON(w io.Writer, result deps.CheckResult) error {
	out := jsonCheckResult{
		AllOK: result.AllOK,
		Deps:  make([]jsonDep, 0, len(result.Deps)),
	}
	for _, d := range result.Deps {
		out.Deps = append(out.Deps, jsonDep{
			Name:        d.Name,
			Binary:      d.Binary,
			Status:      string(d.Status),
			Version:     d.Version,
			Path:        d.Path,
			InstallHelp: d.InstallHelp,
		})
	}
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return fmt.Errorf("json marshal: %w", err)
	}
	_, err = fmt.Fprintln(w, string(data))
	return err
}
