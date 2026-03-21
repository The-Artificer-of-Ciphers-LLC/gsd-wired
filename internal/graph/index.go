package graph

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Index maps GSD identifiers to bd bead IDs for fast local lookups.
// It is a cache — the source of truth is always the beads graph.
// Save/Load operations are atomic via temp+rename.
type Index struct {
	PhaseToID map[string]string `json:"phase_to_id"` // "phase-2" -> "bd-proj-c4l"
	PlanToID  map[string]string `json:"plan_to_id"`  // "02-01" -> "bd-proj-c4l.1"
}

// NewIndex returns an empty Index with initialized maps.
func NewIndex() *Index {
	return &Index{
		PhaseToID: make(map[string]string),
		PlanToID:  make(map[string]string),
	}
}

// Save writes the index to <dir>/index.json atomically via temp+rename.
// The directory must already exist.
func (idx *Index) Save(dir string) error {
	path := filepath.Join(dir, "index.json")
	tmp := path + ".tmp"

	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return fmt.Errorf("index marshal: %w", err)
	}

	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return fmt.Errorf("index write temp: %w", err)
	}

	// Atomic rename — appears as complete or not at all on same filesystem.
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("index rename: %w", err)
	}

	return nil
}

// LoadIndex reads the index from <dir>/index.json.
// Returns an error if the file does not exist or cannot be parsed.
func LoadIndex(dir string) (*Index, error) {
	path := filepath.Join(dir, "index.json")

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("index read: %w", err)
	}

	var idx Index
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, fmt.Errorf("index unmarshal: %w", err)
	}

	// Ensure maps are non-nil even if file had empty/null values.
	if idx.PhaseToID == nil {
		idx.PhaseToID = make(map[string]string)
	}
	if idx.PlanToID == nil {
		idx.PlanToID = make(map[string]string)
	}

	return &idx, nil
}

// RebuildIndex rebuilds the index by querying bd for all gsd:phase and gsd:plan beads.
// This is the recovery path when the index is stale or missing.
func (c *Client) RebuildIndex(ctx context.Context) (*Index, error) {
	idx := NewIndex()

	// Query all phase (epic) beads.
	phaseOut, err := c.run(ctx, "list", "--all", "--label", "gsd:phase", "--limit", "0")
	if err != nil {
		return nil, fmt.Errorf("rebuild index (phases): %w", err)
	}
	var phaseBeads []Bead
	if err := json.Unmarshal(phaseOut, &phaseBeads); err != nil {
		return nil, fmt.Errorf("rebuild index (phases unmarshal): %w", err)
	}
	for _, b := range phaseBeads {
		if phaseNum, ok := b.Metadata["gsd_phase"]; ok {
			key := fmt.Sprintf("phase-%v", phaseNum)
			idx.PhaseToID[key] = b.ID
		}
	}

	// Query all plan (task) beads.
	planOut, err := c.run(ctx, "list", "--all", "--label", "gsd:plan", "--limit", "0")
	if err != nil {
		return nil, fmt.Errorf("rebuild index (plans): %w", err)
	}
	var planBeads []Bead
	if err := json.Unmarshal(planOut, &planBeads); err != nil {
		return nil, fmt.Errorf("rebuild index (plans unmarshal): %w", err)
	}
	for _, b := range planBeads {
		if planID, ok := b.Metadata["gsd_plan"]; ok {
			idx.PlanToID[fmt.Sprint(planID)] = b.ID
		}
	}

	return idx, nil
}
