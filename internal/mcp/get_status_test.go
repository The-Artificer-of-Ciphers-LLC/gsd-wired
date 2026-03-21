package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// phaseBeadsWithClosedPhase returns JSON for two phases: one closed (phase 1) and one open (phase 2).
// Used to verify completed_phases enrichment in get_status.
func phaseBeadsWithClosedPhase() []byte {
	closedAt := time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC)
	beads := []map[string]any{
		{
			"id":           "bd-phase-1-closed",
			"title":        "Binary Scaffold",
			"status":       "closed",
			"close_reason": "Phase 1 complete — binary scaffold verified",
			"closed_at":    closedAt.Format(time.RFC3339),
			"metadata": map[string]any{
				"gsd_phase": 1,
			},
			"labels": []string{"gsd:phase"},
		},
		{
			"id":     "bd-phase-2-open",
			"title":  "Graph Primitives",
			"status": "open",
			"metadata": map[string]any{
				"gsd_phase": 2,
			},
			"labels": []string{"gsd:phase"},
		},
	}
	data, _ := json.Marshal(beads)
	return data
}

// setupGetStatusEnrichedState sets up a serverState with fake_bd returning the given phase beads.
func setupGetStatusEnrichedState(t *testing.T, phaseBeads []byte) *serverState {
	t.Helper()
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, ".beads"), 0755); err != nil {
		t.Fatal(err)
	}
	phaseFile := filepath.Join(tmpDir, "phases.json")
	if err := os.WriteFile(phaseFile, phaseBeads, 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FAKE_BD_QUERY_PHASE_RESPONSE", phaseFile)

	state := &serverState{beadsDir: tmpDir, bdPath: fakeBdPathMCP}
	if err := state.init(context.Background()); err != nil {
		t.Fatalf("state.init() failed: %v", err)
	}
	return state
}

// TestGetStatusEnriched verifies that get_status returns completed_phases with phase num,
// title, close reason, and closed_at for closed phase beads.
func TestGetStatusEnriched(t *testing.T) {
	state := setupGetStatusEnrichedState(t, phaseBeadsWithClosedPhase())
	cs := connectInProcess(t, state)

	result, err := cs.CallTool(context.Background(), &mcpsdk.CallToolParams{
		Name:      "get_status",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("CallTool(get_status) returned error: %v", err)
	}
	if result.IsError {
		t.Fatalf("CallTool(get_status) returned IsError=true: %v", contentText(result))
	}

	text := contentText(result)

	// Parse as raw map to check the JSON structure without strict type assertions.
	var resp map[string]any
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("get_status response is not valid JSON: %v, text: %s", err, text)
	}

	// completed_phases must be present.
	completedRaw, ok := resp["completed_phases"]
	if !ok {
		t.Fatalf("get_status response missing 'completed_phases' field: %v", resp)
	}

	completed, ok := completedRaw.([]any)
	if !ok {
		t.Fatalf("completed_phases is not an array: %T, val: %v", completedRaw, completedRaw)
	}

	if len(completed) == 0 {
		t.Fatal("completed_phases should contain at least one entry (phase 1 is closed)")
	}

	entry, ok := completed[0].(map[string]any)
	if !ok {
		t.Fatalf("completed_phases[0] is not an object: %T", completed[0])
	}

	// Verify phase_num field.
	if entry["phase_num"] == nil {
		t.Errorf("completed_phases[0] missing 'phase_num' field: %v", entry)
	}

	// Verify title field.
	if entry["title"] == nil {
		t.Errorf("completed_phases[0] missing 'title' field: %v", entry)
	}

	// Verify close_reason field.
	closeReason, hasReason := entry["close_reason"].(string)
	if !hasReason || closeReason == "" {
		t.Errorf("completed_phases[0] missing or empty 'close_reason': %v", entry)
	}
}

// TestGetStatusEnrichedEmpty verifies that completed_phases is an empty array (not nil)
// when no phases are closed.
func TestGetStatusEnrichedEmpty(t *testing.T) {
	// Use only an open phase — no closed phases.
	openOnly := []map[string]any{
		{
			"id":     "bd-phase-1-open",
			"title":  "Phase 1",
			"status": "open",
			"metadata": map[string]any{
				"gsd_phase": 1,
			},
			"labels": []string{"gsd:phase"},
		},
	}
	data, _ := json.Marshal(openOnly)

	state := setupGetStatusEnrichedState(t, data)
	cs := connectInProcess(t, state)

	result, err := cs.CallTool(context.Background(), &mcpsdk.CallToolParams{
		Name:      "get_status",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("CallTool(get_status, no-closed) returned error: %v", err)
	}
	if result.IsError {
		t.Fatalf("CallTool(get_status, no-closed) returned IsError=true: %v", contentText(result))
	}

	text := contentText(result)
	var resp map[string]any
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("get_status response is not valid JSON: %v, text: %s", err, text)
	}

	// completed_phases must be present and be an empty array (not null/nil).
	completedRaw, ok := resp["completed_phases"]
	if !ok {
		t.Fatalf("get_status response missing 'completed_phases' field: %v", resp)
	}

	completed, ok := completedRaw.([]any)
	if !ok {
		t.Fatalf("completed_phases is not an array (got %T) — should be [] not null: %v", completedRaw, completedRaw)
	}

	if len(completed) != 0 {
		t.Errorf("completed_phases should be empty when no phases are closed, got %d entries", len(completed))
	}
}
