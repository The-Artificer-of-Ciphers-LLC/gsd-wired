package graph

import (
	"context"
	"encoding/json"
	"strings"
)

// CreatePhase creates a phase as an epic bead in the beads graph.
// The bead gets the "gsd:phase" label plus any requirement IDs, and metadata
// containing the phase number for later lookups.
func (c *Client) CreatePhase(ctx context.Context, phaseNum int, title, goal, acceptance string, reqIDs []string) (*Bead, error) {
	// Build labels: "gsd:phase" + comma-separated reqIDs (per D-05, D-06).
	labels := "gsd:phase"
	for _, rid := range reqIDs {
		labels += "," + rid
	}

	// Build metadata: {"gsd_phase": phaseNum} (per D-09, D-17).
	meta := map[string]any{"gsd_phase": phaseNum}
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return nil, err
	}

	out, err := c.run(ctx,
		"create", title,
		"--type", "epic",
		"--acceptance", acceptance,
		"--context", goal,
		"--metadata", string(metaJSON),
		"--labels", labels,
	)
	if err != nil {
		return nil, err
	}

	var bead Bead
	if err := json.Unmarshal(out, &bead); err != nil {
		return nil, err
	}
	return &bead, nil
}

// CreatePlan creates a plan as a task bead with a parent phase epic bead.
// The bead gets the "gsd:phase" label plus any requirement IDs, and metadata
// containing the phase number and plan ID. If depBeadIDs is non-empty,
// --deps is passed to establish inter-plan dependencies (per D-07).
func (c *Client) CreatePlan(ctx context.Context, planID string, phaseNum int, parentBeadID, title, acceptance, planContext string, reqIDs []string, depBeadIDs []string) (*Bead, error) {
	// Build labels: "gsd:plan" + comma-separated reqIDs (per D-05, D-06).
	labels := "gsd:plan"
	for _, rid := range reqIDs {
		labels += "," + rid
	}

	// Build metadata: {"gsd_phase": phaseNum, "gsd_plan": planID} (per D-09, D-17).
	meta := map[string]any{"gsd_phase": phaseNum, "gsd_plan": planID}
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return nil, err
	}

	args := []string{
		"create", title,
		"--type", "task",
		"--parent", parentBeadID,
		"--no-inherit-labels",
		"--acceptance", acceptance,
		"--context", planContext,
		"--metadata", string(metaJSON),
		"--labels", labels,
	}

	// Append deps only when provided (per D-07).
	if len(depBeadIDs) > 0 {
		args = append(args, "--deps", strings.Join(depBeadIDs, ","))
	}

	out, err := c.run(ctx, args...)
	if err != nil {
		return nil, err
	}

	var bead Bead
	if err := json.Unmarshal(out, &bead); err != nil {
		return nil, err
	}
	return &bead, nil
}
