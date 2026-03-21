package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestCreatePlanBeads verifies that create_plan_beads creates 2 task beads
// (one depends on the other) and returns a task_bead_ids map with both keys.
func TestCreatePlanBeads(t *testing.T) {
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
		Name: "create_plan_beads",
		Arguments: map[string]any{
			"phase_num":    6,
			"epic_bead_id": "epic-123",
			"tasks": []any{
				map[string]any{
					"id":         "06-01",
					"title":      "First Task",
					"acceptance": "Task 1 done",
					"context":    "Do the first thing",
					"req_ids":    []string{"PLAN-01"},
					"depends_on": []string{},
					"complexity": "M",
					"files":      []string{"internal/mcp/first.go"},
				},
				map[string]any{
					"id":         "06-02",
					"title":      "Second Task",
					"acceptance": "Task 2 done",
					"context":    "Do the second thing",
					"req_ids":    []string{"PLAN-02"},
					"depends_on": []string{"06-01"},
					"complexity": "L",
					"files":      []string{"internal/mcp/second.go"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("CallTool(create_plan_beads) returned error: %v", err)
	}
	if result.IsError {
		t.Fatalf("CallTool(create_plan_beads) returned IsError=true: %v", contentText(result))
	}

	text := contentText(result)
	var resp map[string]any
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("create_plan_beads response is not valid JSON: %v, text: %s", err, text)
	}

	taskBeadIDs, ok := resp["task_bead_ids"].(map[string]any)
	if !ok {
		t.Fatalf("create_plan_beads response missing 'task_bead_ids' map: %v", resp)
	}
	if len(taskBeadIDs) != 2 {
		t.Errorf("expected 2 task bead IDs, got %d: %v", len(taskBeadIDs), taskBeadIDs)
	}

	for _, key := range []string{"06-01", "06-02"} {
		id, ok := taskBeadIDs[key].(string)
		if !ok || id == "" {
			t.Errorf("task_bead_ids[%q] is missing or empty: %v", key, taskBeadIDs)
		}
	}
}

// TestCreatePlanBeadsNoDeps verifies that create_plan_beads works with a single task
// having no dependencies.
func TestCreatePlanBeadsNoDeps(t *testing.T) {
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
		Name: "create_plan_beads",
		Arguments: map[string]any{
			"phase_num":    6,
			"epic_bead_id": "epic-456",
			"tasks": []any{
				map[string]any{
					"id":         "06-01",
					"title":      "Solo Task",
					"acceptance": "Solo task done",
					"context":    "Do solo thing",
					"complexity": "S",
					"files":      []string{"internal/mcp/solo.go"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("CallTool(create_plan_beads) no-deps returned error: %v", err)
	}
	if result.IsError {
		t.Fatalf("CallTool(create_plan_beads) no-deps returned IsError=true: %v", contentText(result))
	}

	text := contentText(result)
	var resp map[string]any
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("create_plan_beads no-deps response is not valid JSON: %v, text: %s", err, text)
	}

	taskBeadIDs, ok := resp["task_bead_ids"].(map[string]any)
	if !ok {
		t.Fatalf("create_plan_beads no-deps response missing 'task_bead_ids' map: %v", resp)
	}
	if len(taskBeadIDs) != 1 {
		t.Errorf("expected 1 task bead ID, got %d: %v", len(taskBeadIDs), taskBeadIDs)
	}
}

// TestCreatePlanBeadsBadEpic verifies that create_plan_beads returns IsError=true
// when epic_bead_id is empty.
func TestCreatePlanBeadsBadEpic(t *testing.T) {
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
		Name: "create_plan_beads",
		Arguments: map[string]any{
			"phase_num":    6,
			"epic_bead_id": "", // empty — should fail
			"tasks": []any{
				map[string]any{
					"id":         "06-01",
					"title":      "Task",
					"acceptance": "Done",
					"context":    "Do it",
					"complexity": "S",
					"files":      []string{},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("CallTool(create_plan_beads, bad epic) returned protocol error: %v", err)
	}
	if !result.IsError {
		t.Fatalf("expected IsError=true for empty epic_bead_id, got IsError=false, text: %s", contentText(result))
	}
}
