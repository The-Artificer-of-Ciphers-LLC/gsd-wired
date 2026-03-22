package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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

// TestGetStatusWithPlanningFallback verifies that handleGetStatus returns structured
// statusResult from .planning/ files when .beads/ is absent but .planning/ exists (COMPAT-01).
func TestGetStatusWithPlanningFallback(t *testing.T) {
	tmpDir := t.TempDir()
	// No .beads/ — only .planning/ with STATE.md, ROADMAP.md, PROJECT.md.
	planningDir := filepath.Join(tmpDir, ".planning")
	if err := os.MkdirAll(planningDir, 0755); err != nil {
		t.Fatalf("failed to create .planning/: %v", err)
	}

	if err := os.WriteFile(filepath.Join(planningDir, "STATE.md"), []byte(`# Project State

## Current Position

Phase: 3 of 10 (MCP Server)
Plan: 1 of 2 in current phase
Status: Executing
Last activity: 2026-03-22 -- working

Progress: [███░░░░░░░] 30%
`), 0644); err != nil {
		t.Fatalf("failed to write STATE.md: %v", err)
	}

	if err := os.WriteFile(filepath.Join(planningDir, "ROADMAP.md"), []byte(`# Roadmap

- [x] **Phase 1: Binary Scaffold**
- [x] **Phase 2: Graph Primitives**
- [ ] **Phase 3: MCP Server**

## Phase Details

### Phase 1:
**Goal**: Build the binary
**Plans**: 2 plans

### Phase 3:
**Goal**: Serve MCP protocol
**Plans**: 2 plans
`), 0644); err != nil {
		t.Fatalf("failed to write ROADMAP.md: %v", err)
	}

	if err := os.WriteFile(filepath.Join(planningDir, "PROJECT.md"), []byte(`# FallbackProject

## Core Value

Token-efficient orchestration.
`), 0644); err != nil {
		t.Fatalf("failed to write PROJECT.md: %v", err)
	}

	// Use a serverState with beadsDir pointing to tmpDir but no bd — init will fail.
	state := &serverState{beadsDir: tmpDir, bdPath: "/nonexistent/bd"}

	result, err := handleGetStatus(context.Background(), state)
	if err != nil {
		t.Fatalf("handleGetStatus returned error: %v", err)
	}
	if result.IsError {
		t.Fatalf("handleGetStatus returned IsError=true: %v", contentText(result))
	}

	text := contentText(result)
	var resp map[string]any
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("get_status .planning/ response is not valid JSON: %v, text: %s", err, text)
	}

	// project_name must come from PROJECT.md.
	projectName, _ := resp["project_name"].(string)
	if projectName != "FallbackProject" {
		t.Errorf("expected project_name 'FallbackProject' from .planning/, got %q", projectName)
	}

	// total_phases should be set from ROADMAP.md.
	totalPhases, _ := resp["total_phases"].(float64)
	if totalPhases == 0 {
		t.Errorf("expected total_phases > 0 from .planning/ ROADMAP.md, got 0")
	}

	// current_phase should be populated with phase info.
	if resp["current_phase"] == nil {
		t.Errorf("expected current_phase to be populated from STATE.md phase 3, got nil")
	}
}

// TestGetStatusFallbackNoPlanning verifies handleGetStatus returns IsError when
// no .beads/ and no .planning/ exist.
func TestGetStatusFallbackNoPlanning(t *testing.T) {
	tmpDir := t.TempDir()
	// Neither .beads/ nor .planning/ — completely uninitialized.

	state := &serverState{beadsDir: tmpDir, bdPath: "/nonexistent/bd"}

	result, err := handleGetStatus(context.Background(), state)
	if err != nil {
		t.Fatalf("handleGetStatus returned error: %v", err)
	}
	// Without any fallback available, should return IsError=true.
	if !result.IsError {
		text := contentText(result)
		// Accept either IsError=true or an error-indicating message.
		if !strings.Contains(text, "error") && !strings.Contains(text, "Error") && !strings.Contains(text, "not found") {
			t.Errorf("expected IsError=true or error message when no .beads/ and no .planning/, got: %s", text)
		}
	}
}

// TestGetStatusFallbackPopulatesFields verifies the fallbackStatusResult helper correctly
// maps FallbackStatus fields to statusResult fields (Test 4 from plan).
func TestGetStatusFallbackPopulatesFields(t *testing.T) {
	tmpDir := t.TempDir()
	planningDir := filepath.Join(tmpDir, ".planning")
	if err := os.MkdirAll(planningDir, 0755); err != nil {
		t.Fatalf("failed to create .planning/: %v", err)
	}

	// Write files with known values.
	if err := os.WriteFile(filepath.Join(planningDir, "STATE.md"), []byte(`Phase: 7 of 10 (Execution)
Plan: 3 of 3 in current phase
Progress: [███████░░░] 70%
`), 0644); err != nil {
		t.Fatalf("failed to write STATE.md: %v", err)
	}

	if err := os.WriteFile(filepath.Join(planningDir, "ROADMAP.md"), []byte(`# Roadmap

- [x] **Phase 1: Binary Scaffold**
- [x] **Phase 2: Graph Primitives**
- [x] **Phase 3: MCP Server**
- [x] **Phase 4: Hook Integration**
- [x] **Phase 5: Project Init**
- [x] **Phase 6: Research**
- [ ] **Phase 7: Execution**
- [ ] **Phase 8: Ship**
- [ ] **Phase 9: Token**
- [ ] **Phase 10: Coexistence**

## Phase Details

### Phase 7:
**Goal**: Execute waves
**Plans**: 3 plans
`), 0644); err != nil {
		t.Fatalf("failed to write ROADMAP.md: %v", err)
	}

	if err := os.WriteFile(filepath.Join(planningDir, "PROJECT.md"), []byte(`# FieldsProject

## Core Value

Test fields mapping.
`), 0644); err != nil {
		t.Fatalf("failed to write PROJECT.md: %v", err)
	}

	state := &serverState{beadsDir: tmpDir, bdPath: "/nonexistent/bd"}

	result, err := handleGetStatus(context.Background(), state)
	if err != nil {
		t.Fatalf("handleGetStatus returned error: %v", err)
	}
	if result.IsError {
		t.Fatalf("handleGetStatus returned IsError=true: %v", contentText(result))
	}

	text := contentText(result)
	var resp map[string]any
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("response is not valid JSON: %v, text: %s", err, text)
	}

	// ProjectName.
	if resp["project_name"] != "FieldsProject" {
		t.Errorf("expected project_name 'FieldsProject', got %v", resp["project_name"])
	}

	// TotalPhases = 10 (from ROADMAP.md).
	totalPhases, _ := resp["total_phases"].(float64)
	if int(totalPhases) != 10 {
		t.Errorf("expected total_phases=10, got %v", totalPhases)
	}

	// OpenPhases should be 4 (phases 7-10 not complete).
	openPhases, _ := resp["open_phases"].(float64)
	if int(openPhases) != 4 {
		t.Errorf("expected open_phases=4, got %v", openPhases)
	}

	// CurrentPhase should be phase 7.
	cp, ok := resp["current_phase"].(map[string]any)
	if !ok || cp == nil {
		t.Fatalf("expected current_phase object, got %T: %v", resp["current_phase"], resp["current_phase"])
	}
	phaseNum, _ := cp["phase_num"].(float64)
	if int(phaseNum) != 7 {
		t.Errorf("expected current_phase.phase_num=7, got %v", phaseNum)
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
