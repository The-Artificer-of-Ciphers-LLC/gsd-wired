package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// setupAdvancePhaseState sets up a serverState with fake_bd returning the given phase beads.
func setupAdvancePhaseState(t *testing.T, phaseBeads []byte) (*serverState, string) {
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
	return state, tmpDir
}

// TestAdvancePhase verifies that advance_phase closes a phase epic bead via ClosePlan
// and returns advancePhaseResult with a closed bead and unblocked list.
func TestAdvancePhase(t *testing.T) {
	// Create phase beads: phase 1 (open — to be closed) and phase 2 (open — next).
	phaseBeads := []map[string]any{
		{
			"id":     "bd-phase-1",
			"title":  "Phase 1",
			"status": "open",
			"metadata": map[string]any{
				"gsd_phase": 1,
			},
			"labels": []string{"gsd:phase"},
		},
		{
			"id":     "bd-phase-2",
			"title":  "Phase 2",
			"status": "open",
			"metadata": map[string]any{
				"gsd_phase": 2,
			},
			"labels": []string{"gsd:phase"},
		},
	}
	beadsData, err := json.Marshal(phaseBeads)
	if err != nil {
		t.Fatal(err)
	}

	state, _ := setupAdvancePhaseState(t, beadsData)
	cs := connectInProcess(t, state)

	result, err := cs.CallTool(context.Background(), &mcpsdk.CallToolParams{
		Name: "advance_phase",
		Arguments: map[string]any{
			"phase_num": 1,
			"reason":    "Phase 1 complete — all acceptance criteria verified",
		},
	})
	if err != nil {
		t.Fatalf("CallTool(advance_phase) returned error: %v", err)
	}
	if result.IsError {
		t.Fatalf("CallTool(advance_phase) returned IsError=true: %v", contentText(result))
	}

	text := contentText(result)
	var resp advancePhaseResult
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("advance_phase response is not valid JSON: %v, text: %s", err, text)
	}

	if resp.Closed == nil {
		t.Fatal("advance_phase closed bead is nil")
	}
	if resp.Unblocked == nil {
		t.Error("advance_phase unblocked array is nil (should be empty slice at minimum)")
	}
}

// TestAdvancePhaseNotFound verifies that a non-existent phase_num returns toolError.
func TestAdvancePhaseNotFound(t *testing.T) {
	// Use a single phase bead (phase 1) so phase 99 won't be found.
	phaseBeads := []map[string]any{
		{
			"id":     "bd-phase-1",
			"title":  "Phase 1",
			"status": "open",
			"metadata": map[string]any{
				"gsd_phase": 1,
			},
			"labels": []string{"gsd:phase"},
		},
	}
	beadsData, err := json.Marshal(phaseBeads)
	if err != nil {
		t.Fatal(err)
	}

	state, _ := setupAdvancePhaseState(t, beadsData)
	cs := connectInProcess(t, state)

	result, err := cs.CallTool(context.Background(), &mcpsdk.CallToolParams{
		Name: "advance_phase",
		Arguments: map[string]any{
			"phase_num": 99,
			"reason":    "test",
		},
	})
	if err != nil {
		t.Fatalf("CallTool(advance_phase, not-found) returned protocol error: %v", err)
	}
	if !result.IsError {
		t.Fatalf("advance_phase with non-existent phase_num should return IsError=true, got: %s", contentText(result))
	}
	text := contentText(result)
	if !contains(text, "no phase epic found") {
		t.Errorf("advance_phase error should contain 'no phase epic found', got: %s", text)
	}
}

// TestAdvancePhaseNextPhase verifies that advance_phase surfaces the next open phase info.
func TestAdvancePhaseNextPhase(t *testing.T) {
	phaseBeads := []map[string]any{
		{
			"id":     "bd-phase-1",
			"title":  "Phase 1",
			"status": "open",
			"metadata": map[string]any{
				"gsd_phase": 1,
			},
			"labels": []string{"gsd:phase"},
		},
		{
			"id":     "bd-phase-2",
			"title":  "Phase 2",
			"status": "open",
			"metadata": map[string]any{
				"gsd_phase": 2,
			},
			"labels": []string{"gsd:phase"},
		},
	}
	beadsData, err := json.Marshal(phaseBeads)
	if err != nil {
		t.Fatal(err)
	}

	state, _ := setupAdvancePhaseState(t, beadsData)
	cs := connectInProcess(t, state)

	result, err := cs.CallTool(context.Background(), &mcpsdk.CallToolParams{
		Name: "advance_phase",
		Arguments: map[string]any{
			"phase_num": 1,
			"reason":    "Done",
		},
	})
	if err != nil {
		t.Fatalf("CallTool(advance_phase, next-phase) returned error: %v", err)
	}
	if result.IsError {
		t.Fatalf("CallTool(advance_phase, next-phase) returned IsError=true: %v", contentText(result))
	}

	text := contentText(result)
	var resp advancePhaseResult
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("advance_phase next-phase response is not valid JSON: %v, text: %s", err, text)
	}

	// With phase 2 open and having phaseNum > 1, NextPhase should be set.
	if resp.NextPhase == nil {
		t.Error("advance_phase NextPhase is nil, want phaseInfo for phase 2")
	} else if resp.NextPhase.PhaseNum != 2 {
		t.Errorf("advance_phase NextPhase.PhaseNum = %d, want 2", resp.NextPhase.PhaseNum)
	}
}
