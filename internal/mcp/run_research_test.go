package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestRunResearch verifies that run_research creates an epic bead + 4 child beads
// and returns structured JSON with epic_bead_id and child_bead_ids map.
func TestRunResearch(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, ".beads"), 0755); err != nil {
		t.Fatal(err)
	}

	state := &serverState{beadsDir: tmpDir, bdPath: fakeBdPathMCP}
	if err := state.init(context.Background()); err != nil {
		t.Fatalf("state.init() failed: %v", err)
	}

	cs := connectInProcess(t, state)

	result, err := cs.CallTool(context.Background(), &mcpsdk.CallToolParams{
		Name: "run_research",
		Arguments: map[string]any{
			"phase_num": 6,
			"title":     "Test Research",
			"req_ids":   []string{"RSRCH-01"},
		},
	})
	if err != nil {
		t.Fatalf("CallTool(run_research) returned error: %v", err)
	}
	if result.IsError {
		t.Fatalf("CallTool(run_research) returned IsError=true: %v", contentText(result))
	}

	text := contentText(result)
	var resp map[string]any
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("run_research response is not valid JSON: %v, text: %s", err, text)
	}

	// epic_bead_id must be non-empty.
	epicID, ok := resp["epic_bead_id"].(string)
	if !ok || epicID == "" {
		t.Errorf("run_research response missing or empty 'epic_bead_id': %v", resp)
	}

	// child_bead_ids must be a map with exactly 4 keys: stack, features, architecture, pitfalls.
	childIDs, ok := resp["child_bead_ids"].(map[string]any)
	if !ok {
		t.Fatalf("run_research response 'child_bead_ids' is not a map: %v", resp["child_bead_ids"])
	}
	if len(childIDs) != 4 {
		t.Errorf("run_research expected 4 child_bead_ids, got %d: %v", len(childIDs), childIDs)
	}

	wantTopics := []string{"stack", "features", "architecture", "pitfalls"}
	for _, topic := range wantTopics {
		id, ok := childIDs[topic].(string)
		if !ok || id == "" {
			t.Errorf("run_research child_bead_ids[%q] is missing or empty: %v", topic, childIDs)
		}
	}
}

// TestSynthesizeResearch verifies that synthesize_research creates a summary bead
// and returns JSON with summary_bead_id.
func TestSynthesizeResearch(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, ".beads"), 0755); err != nil {
		t.Fatal(err)
	}

	state := &serverState{beadsDir: tmpDir, bdPath: fakeBdPathMCP}
	if err := state.init(context.Background()); err != nil {
		t.Fatalf("state.init() failed: %v", err)
	}

	cs := connectInProcess(t, state)

	// First create a research epic so synthesize can find it.
	setupResult, err := cs.CallTool(context.Background(), &mcpsdk.CallToolParams{
		Name: "run_research",
		Arguments: map[string]any{
			"phase_num": 6,
			"title":     "Test Research for Synthesis",
			"req_ids":   []string{"RSRCH-01"},
		},
	})
	if err != nil {
		t.Fatalf("CallTool(run_research) setup returned error: %v", err)
	}
	if setupResult.IsError {
		t.Fatalf("CallTool(run_research) setup returned IsError=true: %v", contentText(setupResult))
	}

	result, err := cs.CallTool(context.Background(), &mcpsdk.CallToolParams{
		Name: "synthesize_research",
		Arguments: map[string]any{
			"phase_num": 6,
			"summary":   "Research complete: Go stack recommended, feature set defined",
		},
	})
	if err != nil {
		t.Fatalf("CallTool(synthesize_research) returned error: %v", err)
	}
	if result.IsError {
		t.Fatalf("CallTool(synthesize_research) returned IsError=true: %v", contentText(result))
	}

	text := contentText(result)
	var resp map[string]any
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("synthesize_research response is not valid JSON: %v, text: %s", err, text)
	}

	summaryID, ok := resp["summary_bead_id"].(string)
	if !ok || summaryID == "" {
		t.Errorf("synthesize_research response missing or empty 'summary_bead_id': %v", resp)
	}
}
