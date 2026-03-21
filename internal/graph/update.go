package graph

import (
	"context"
	"encoding/json"
	"fmt"
)

// ClaimBead atomically claims a bead for the current user.
// Uses bd's native atomic claim semantics (fails if already claimed).
func (c *Client) ClaimBead(ctx context.Context, beadID string) (*Bead, error) {
	out, err := c.run(ctx, "update", beadID, "--claim")
	if err != nil {
		return nil, err
	}
	var bead Bead
	if err := json.Unmarshal(out, &bead); err != nil {
		return nil, err
	}
	return &bead, nil
}

// ClosePlan closes a plan bead and computes newly unblocked tasks via before/after ready diff.
// Returns: (closed bead, newly unblocked beads, error).
// If post-close ready query fails, returns (closed, nil, nil) — notification is best-effort (per D-13).
func (c *Client) ClosePlan(ctx context.Context, beadID, reason string) (*Bead, []Bead, error) {
	// Step 1: Snapshot ready beads before close.
	prevReady, err := c.ListReady(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("pre-close ready snapshot: %w", err)
	}
	prevReadyIDs := make(map[string]bool, len(prevReady))
	for _, b := range prevReady {
		prevReadyIDs[b.ID] = true
	}

	// Step 2: Close the bead.
	args := []string{"close", beadID}
	if reason != "" {
		args = append(args, "--reason", reason)
	}
	out, err := c.run(ctx, args...)
	if err != nil {
		return nil, nil, err
	}

	// bd close --json returns an array containing the closed bead.
	var closed []Bead
	if err := json.Unmarshal(out, &closed); err != nil {
		return nil, nil, err
	}
	if len(closed) == 0 {
		return nil, nil, fmt.Errorf("bd close returned empty response")
	}

	// Step 3: Snapshot ready beads after close; diff gives newly unblocked.
	newReady, err := c.ListReady(ctx)
	if err != nil {
		// Close succeeded; notification is best-effort.
		return &closed[0], nil, nil
	}

	var newlyUnblocked []Bead
	for _, b := range newReady {
		if !prevReadyIDs[b.ID] {
			newlyUnblocked = append(newlyUnblocked, b)
		}
	}

	return &closed[0], newlyUnblocked, nil
}

// AddLabel adds a label to an existing bead using bd update --add-label.
func (c *Client) AddLabel(ctx context.Context, beadID, label string) (*Bead, error) {
	out, err := c.run(ctx, "update", beadID, "--add-label", label)
	if err != nil {
		return nil, err
	}
	var bead Bead
	if err := json.Unmarshal(out, &bead); err != nil {
		return nil, err
	}
	return &bead, nil
}
