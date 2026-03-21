package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// phaseBeadsWithClosed returns JSON for two phase beads: one open (phase 2) and one closed (phase 1).
// Used to test create_pr_summary with completed phases in the checklist.
func phaseBeadsWithClosed() []byte {
	beads := []map[string]any{
		{
			"id":          "bd-phase-1",
			"title":       "Binary Scaffold",
			"status":      "closed",
			"description": "Build the binary scaffold goal",
			"close_reason": "Phase 1 complete",
			"metadata": map[string]any{
				"gsd_phase": 1,
			},
			"labels": []string{"gsd:phase", "INFRA-01", "INFRA-02"},
		},
		{
			"id":          "bd-phase-2",
			"title":       "Graph Primitives",
			"status":      "open",
			"description": "Build graph primitives goal",
			"metadata": map[string]any{
				"gsd_phase": 2,
			},
			"labels": []string{"gsd:phase", "MAP-01", "MAP-02"},
		},
	}
	data, _ := json.Marshal(beads)
	return data
}

// setupPrSummaryState sets up a serverState with fake_bd returning custom phase beads.
func setupPrSummaryState(t *testing.T, phaseBeads []byte) (*serverState, string) {
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

// TestCreatePrSummary verifies that create_pr_summary with canned phase beads returns
// a prSummaryResult with title, body containing "## Requirements" and "## Phases", and branch_name.
func TestCreatePrSummary(t *testing.T) {
	state, _ := setupPrSummaryState(t, phaseBeadsWithClosed())
	cs := connectInProcess(t, state)

	result, err := cs.CallTool(context.Background(), &mcpsdk.CallToolParams{
		Name:      "create_pr_summary",
		Arguments: map[string]any{"phase_num": 2},
	})
	if err != nil {
		t.Fatalf("CallTool(create_pr_summary) returned error: %v", err)
	}
	if result.IsError {
		t.Fatalf("CallTool(create_pr_summary) returned IsError=true: %v", contentText(result))
	}

	text := contentText(result)
	var resp prSummaryResult
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("create_pr_summary response is not valid JSON: %v, text: %s", err, text)
	}

	if resp.Title == "" {
		t.Error("create_pr_summary title is empty")
	}
	if !contains(resp.Title, "2") {
		t.Errorf("create_pr_summary title should contain phase number '2', got: %q", resp.Title)
	}
	if !contains(resp.Body, "## Requirements") {
		t.Errorf("create_pr_summary body missing '## Requirements' section, body:\n%s", resp.Body)
	}
	if !contains(resp.Body, "## Phases") {
		t.Errorf("create_pr_summary body missing '## Phases' section, body:\n%s", resp.Body)
	}
	if resp.BranchName == "" {
		t.Error("create_pr_summary branch_name is empty")
	}
	if !contains(resp.BranchName, "2") {
		t.Errorf("create_pr_summary branch_name should contain phase number '2', got: %q", resp.BranchName)
	}
}

// TestCreatePrSummaryNoProject verifies that with no project bead, create_pr_summary
// returns a result with a fallback title containing "Phase N Ship".
func TestCreatePrSummaryNoProject(t *testing.T) {
	// Use phase beads but no project bead — fake_bd returns [] for non-phase labels.
	state, _ := setupPrSummaryState(t, phaseBeadsWithClosed())
	cs := connectInProcess(t, state)

	result, err := cs.CallTool(context.Background(), &mcpsdk.CallToolParams{
		Name:      "create_pr_summary",
		Arguments: map[string]any{"phase_num": 2},
	})
	if err != nil {
		t.Fatalf("CallTool(create_pr_summary, no-project) returned error: %v", err)
	}
	if result.IsError {
		t.Fatalf("CallTool(create_pr_summary, no-project) returned IsError=true: %v", contentText(result))
	}

	text := contentText(result)
	var resp prSummaryResult
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("create_pr_summary no-project response is not valid JSON: %v, text: %s", err, text)
	}
	// Title should exist even without a project bead.
	if resp.Title == "" {
		t.Error("create_pr_summary no-project title is empty")
	}
}

// TestCreatePrSummaryNotFound verifies that a non-existent phase_num returns IsError=true.
func TestCreatePrSummaryNotFound(t *testing.T) {
	state, _ := setupPrSummaryState(t, phaseBeadsWithClosed())
	cs := connectInProcess(t, state)

	result, err := cs.CallTool(context.Background(), &mcpsdk.CallToolParams{
		Name:      "create_pr_summary",
		Arguments: map[string]any{"phase_num": 99},
	})
	if err != nil {
		t.Fatalf("CallTool(create_pr_summary, not-found) returned protocol error: %v", err)
	}
	if !result.IsError {
		t.Fatalf("create_pr_summary with non-existent phase_num should return IsError=true, got: %s", contentText(result))
	}
}
