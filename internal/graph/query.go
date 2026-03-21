package graph

import (
	"context"
	"encoding/json"
)

// ListReady returns all unblocked (ready) beads with no limit.
// Uses bd ready --limit 0 to prevent silent truncation (default limit is 10).
func (c *Client) ListReady(ctx context.Context) ([]Bead, error) {
	out, err := c.run(ctx, "ready", "--limit", "0")
	if err != nil {
		return nil, err
	}
	var beads []Bead
	if err := json.Unmarshal(out, &beads); err != nil {
		return nil, err
	}
	return beads, nil
}

// ReadyForPhase returns all unblocked beads that are children of the given phase epic bead.
// Uses --parent filter for server-side scoping.
func (c *Client) ReadyForPhase(ctx context.Context, phaseBeadID string) ([]Bead, error) {
	out, err := c.run(ctx, "ready", "--parent", phaseBeadID, "--limit", "0")
	if err != nil {
		return nil, err
	}
	var beads []Bead
	if err := json.Unmarshal(out, &beads); err != nil {
		return nil, err
	}
	return beads, nil
}

// ListBlocked returns all beads that are blocked by unresolved dependencies.
func (c *Client) ListBlocked(ctx context.Context) ([]Bead, error) {
	out, err := c.run(ctx, "blocked", "--limit", "0")
	if err != nil {
		return nil, err
	}
	var beads []Bead
	if err := json.Unmarshal(out, &beads); err != nil {
		return nil, err
	}
	return beads, nil
}

// GetBead retrieves a single bead by ID.
func (c *Client) GetBead(ctx context.Context, beadID string) (*Bead, error) {
	out, err := c.run(ctx, "show", beadID)
	if err != nil {
		return nil, err
	}
	var bead Bead
	if err := json.Unmarshal(out, &bead); err != nil {
		return nil, err
	}
	return &bead, nil
}

// QueryByLabel returns all beads matching the given label.
// Uses bd's server-side label index for efficiency.
func (c *Client) QueryByLabel(ctx context.Context, label string) ([]Bead, error) {
	out, err := c.run(ctx, "query", "label="+label, "--limit", "0")
	if err != nil {
		return nil, err
	}
	var beads []Bead
	if err := json.Unmarshal(out, &beads); err != nil {
		return nil, err
	}
	return beads, nil
}
