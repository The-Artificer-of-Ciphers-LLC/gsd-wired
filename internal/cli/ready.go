package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/The-Artificer-of-Ciphers-LLC/gsd-wired/internal/graph"
)

// reqLabelPattern matches GSD requirement ID labels like INFRA-03, MAP-01, AUTH-02.
var reqLabelPattern = regexp.MustCompile(`^[A-Z]+-[0-9]+$`)

// NewReadyCmd creates the "gsdw ready" subcommand.
// Displays unblocked tasks in tree format grouped by phase, with --json for machine output
// and --phase N for filtering to a single phase.
func NewReadyCmd() *cobra.Command {
	var jsonMode bool
	var phaseFilter int

	cmd := &cobra.Command{
		Use:   "ready",
		Short: "Show unblocked tasks ready to work on",
		Long: `Display all unblocked tasks grouped by phase in tree format.

Use --json for machine-readable output or --phase N to filter to one phase.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			// Locate .beads/ directory by walking up from cwd or using env var.
			beadsDir, err := findBeadsDir()
			if err != nil {
				return err
			}

			client, err := graph.NewClient(beadsDir)
			if err != nil {
				return err
			}

			if jsonMode {
				// JSON mode: emit raw Bead array.
				var beads []graph.Bead
				if phaseFilter != 0 {
					// Load index to resolve phase number -> bead ID.
					idx, idxErr := graph.LoadIndex(beadsDir)
					if idxErr == nil {
						phaseKey := fmt.Sprintf("phase-%d", phaseFilter)
						if phaseBeadID, ok := idx.PhaseToID[phaseKey]; ok {
							beads, err = client.ReadyForPhase(ctx, phaseBeadID)
						} else {
							// Unknown phase — return empty array.
							beads = []graph.Bead{}
						}
					} else {
						// Index unavailable: fall back to full list + client-side filter.
						all, listErr := client.ListReady(ctx)
						if listErr != nil {
							return listErr
						}
						for _, b := range all {
							if phaseNumFromBead(b) == phaseFilter {
								beads = append(beads, b)
							}
						}
					}
				} else {
					beads, err = client.ListReady(ctx)
				}
				if err != nil {
					return err
				}
				return renderReadyJSON(cmd.OutOrStdout(), beads)
			}

			// Tree mode (default).
			ready, err := client.ListReady(ctx)
			if err != nil {
				return err
			}
			blocked, err := client.ListBlocked(ctx)
			if err != nil {
				return err
			}

			return renderReadyTree(cmd.OutOrStdout(), ready, blocked, phaseFilter)
		},
	}

	cmd.Flags().BoolVar(&jsonMode, "json", false, "Output raw JSON array instead of tree")
	cmd.Flags().IntVar(&phaseFilter, "phase", 0, "Filter to specific phase number (0 = all phases)")

	return cmd
}

// findBeadsDir locates the .beads/ directory by walking up from the current working directory,
// or returns the BEADS_DIR environment variable if set.
func findBeadsDir() (string, error) {
	if dir := os.Getenv("BEADS_DIR"); dir != "" {
		return dir, nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("cannot determine working directory: %w", err)
	}

	dir := cwd
	for {
		candidate := filepath.Join(dir, ".beads")
		if info, statErr := os.Stat(candidate); statErr == nil && info.IsDir() {
			return candidate, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("no beads database found — run gsdw init first")
}

// phaseNumFromBead extracts the integer gsd_phase from a bead's metadata.
// Returns 0 if metadata is absent or not a numeric type.
func phaseNumFromBead(b graph.Bead) int {
	if b.Metadata == nil {
		return 0
	}
	switch v := b.Metadata["gsd_phase"].(type) {
	case float64:
		return int(v)
	case int:
		return v
	case int64:
		return int(v)
	}
	return 0
}

// planIDFromBead extracts the gsd_plan string from a bead's metadata (e.g. "02-01").
func planIDFromBead(b graph.Bead) string {
	if b.Metadata == nil {
		return ""
	}
	if v, ok := b.Metadata["gsd_plan"].(string); ok {
		return v
	}
	return ""
}

// reqLabels filters a bead's Labels to only those matching the [A-Z]+-[0-9]+ requirement pattern.
// Internal labels (gsd:plan, gsd:phase, etc.) are excluded.
func reqLabels(labels []string) []string {
	var out []string
	for _, l := range labels {
		if reqLabelPattern.MatchString(l) {
			out = append(out, l)
		}
	}
	return out
}

// renderReadyJSON marshals the ready bead slice to an indented JSON array on w.
func renderReadyJSON(w io.Writer, ready []graph.Bead) error {
	if ready == nil {
		ready = []graph.Bead{}
	}
	data, err := json.MarshalIndent(ready, "", "  ")
	if err != nil {
		return fmt.Errorf("json marshal: %w", err)
	}
	_, err = fmt.Fprintln(w, string(data))
	return err
}

// phaseGroup holds all ready beads for a single GSD phase number.
type phaseGroup struct {
	phaseNum int
	beads    []graph.Bead
}

// renderReadyTree renders the human-readable tree to w.
// ready are the unblocked beads, blocked are queued beads for count purposes.
// phaseFilter == 0 means show all phases; non-zero means restrict to that phase number.
func renderReadyTree(w io.Writer, ready []graph.Bead, blocked []graph.Bead, phaseFilter int) error {
	// Group ready beads by phase number.
	groupMap := make(map[int][]graph.Bead)
	for _, b := range ready {
		pn := phaseNumFromBead(b)
		groupMap[pn] = append(groupMap[pn], b)
	}

	// Apply phase filter.
	if phaseFilter != 0 {
		filtered := make(map[int][]graph.Bead)
		if beads, ok := groupMap[phaseFilter]; ok {
			filtered[phaseFilter] = beads
		}
		groupMap = filtered
	}

	// Sort phase numbers for deterministic output.
	phaseNums := make([]int, 0, len(groupMap))
	for pn := range groupMap {
		phaseNums = append(phaseNums, pn)
	}
	sort.Ints(phaseNums)

	// Count blocked beads that have a gsd:plan label (queued plans).
	queuedCount := 0
	for _, b := range blocked {
		for _, l := range b.Labels {
			if l == "gsd:plan" {
				queuedCount++
				break
			}
		}
	}

	readyCount := len(ready)
	if phaseFilter != 0 {
		// When filtering, readyCount is just the filtered count.
		readyCount = 0
		for _, beads := range groupMap {
			readyCount += len(beads)
		}
	}
	remaining := readyCount + queuedCount

	if len(groupMap) == 0 {
		fmt.Fprintf(w, "No ready work.\n")
		fmt.Fprintf(w, "\nTotal: 0 ready | %d queued | %d remaining\n", queuedCount, remaining)
		return nil
	}

	fmt.Fprintf(w, "Ready Work (%d tasks, %d remaining)\n", readyCount, remaining)

	for i, pn := range phaseNums {
		beads := groupMap[pn]

		// Phase header — use "Phase N:" format (per D-19, no bd IDs).
		fmt.Fprintf(w, "\n  Phase %d:\n", pn)

		// Sort beads by plan ID for deterministic output.
		sort.Slice(beads, func(a, b int) bool {
			return planIDFromBead(beads[a]) < planIDFromBead(beads[b])
		})

		_ = i // suppress unused warning
		for j, b := range beads {
			connector := "|--"
			if j == len(beads)-1 {
				connector = "+--"
			}

			planID := planIDFromBead(b)
			reqLbls := reqLabels(b.Labels)

			var line string
			if planID != "" && b.Title != "" {
				line = fmt.Sprintf("  %s Plan %s: %s", connector, planID, b.Title)
			} else if planID != "" {
				line = fmt.Sprintf("  %s Plan %s:", connector, planID)
			} else {
				line = fmt.Sprintf("  %s %s", connector, b.Title)
			}

			if len(reqLbls) > 0 {
				line += "      [" + strings.Join(reqLbls, ", ") + "]"
			}

			fmt.Fprintln(w, line)
		}
	}

	fmt.Fprintf(w, "\nTotal: %d ready | %d queued | %d remaining\n", readyCount, queuedCount, remaining)
	return nil
}
