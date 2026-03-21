package cli

import (
	"fmt"
	"io"
	"sort"

	"github.com/spf13/cobra"

	"github.com/The-Artificer-of-Ciphers-LLC/gsd-wired/internal/graph"
)

// NewStatusCmd creates the "gsdw status" subcommand.
// Shows current project status by querying the beads graph for phases and ready tasks.
func NewStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show project status from beads graph",
		Long: `Display the current project status including the active phase and tasks
ready to work on next. Uses GSD terminology (phases, plans, waves) — never
exposes bead IDs or graph internals.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			beadsDir, err := findBeadsDir()
			if err != nil {
				return err
			}

			client, err := graph.NewClient(beadsDir)
			if err != nil {
				return err
			}

			phases, err := client.QueryByLabel(ctx, "gsd:phase")
			if err != nil {
				return fmt.Errorf("cannot query phases: %w", err)
			}

			ready, err := client.ListReady(ctx)
			if err != nil {
				return fmt.Errorf("cannot list ready tasks: %w", err)
			}

			renderStatus(cmd.OutOrStdout(), phases, ready)
			return nil
		},
	}
	return cmd
}

// renderStatus renders the GSD project status dashboard to w.
// phases are all beads labeled gsd:phase; ready are all unblocked task beads.
// Uses GSD-familiar terms (Phase N, Plan XX-YY) — never exposes bead IDs.
func renderStatus(w io.Writer, phases []graph.Bead, ready []graph.Bead) {
	fmt.Fprintln(w, "GSD Project Status")
	fmt.Fprintln(w, "==================")

	if len(phases) == 0 {
		fmt.Fprintln(w, "No project initialized. Run gsdw init first.")
		return
	}

	// Find the current (highest-numbered open) phase.
	// Sort phases by phase number descending to find the active one.
	openPhases := make([]graph.Bead, 0, len(phases))
	for _, p := range phases {
		if p.Status == "open" || p.Status == "" {
			openPhases = append(openPhases, p)
		}
	}

	var currentPhase *graph.Bead
	if len(openPhases) > 0 {
		sort.Slice(openPhases, func(i, j int) bool {
			return phaseNumFromBead(openPhases[i]) > phaseNumFromBead(openPhases[j])
		})
		currentPhase = &openPhases[0]
	} else {
		// All phases closed — use the highest numbered one.
		sorted := make([]graph.Bead, len(phases))
		copy(sorted, phases)
		sort.Slice(sorted, func(i, j int) bool {
			return phaseNumFromBead(sorted[i]) > phaseNumFromBead(sorted[j])
		})
		currentPhase = &sorted[0]
	}

	phaseNum := phaseNumFromBead(*currentPhase)
	if phaseNum != 0 {
		fmt.Fprintf(w, "\nCurrent Phase: %s (Phase %d)\n", currentPhase.Title, phaseNum)
	} else {
		fmt.Fprintf(w, "\nCurrent Phase: %s\n", currentPhase.Title)
	}

	// List ready tasks.
	if len(ready) == 0 {
		fmt.Fprintln(w, "\nReady tasks: none — all work may be queued or complete.")
	} else {
		fmt.Fprintln(w, "\nReady tasks (next wave):")
		for _, b := range ready {
			planID := planIDFromBead(b)
			if planID != "" {
				fmt.Fprintf(w, "  - Plan %s: %s\n", planID, b.Title)
			} else {
				fmt.Fprintf(w, "  - %s\n", b.Title)
			}
		}
	}
}
