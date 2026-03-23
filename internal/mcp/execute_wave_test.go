package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestExecuteWave verifies that execute_wave with phase_num=1 returns executeWaveResult
// with a tasks array containing ready task context chains.
func TestExecuteWave(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, ".beads"), 0755); err != nil {
		t.Fatal(err)
	}

	// Create a ready bead JSON with phase 1 metadata and acceptance criteria.
	readyBead := map[string]any{
		"id":                  "bd-task-001",
		"title":               "Implement execute_wave",
		"status":              "open",
		"description":         "Context description for the task",
		"acceptance_criteria": "execute_wave tool implemented and tested",
		"metadata": map[string]any{
			"gsd_phase": 1,
			"gsd_plan":  "07-01",
		},
		"labels": []string{"gsd:plan"},
	}
	readyData, err := json.Marshal([]any{readyBead})
	if err != nil {
		t.Fatal(err)
	}

	readyFile := filepath.Join(tmpDir, "ready.json")
	if err := os.WriteFile(readyFile, readyData, 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FAKE_BD_READY_RESPONSE", readyFile)

	state := &serverState{beadsDir: tmpDir, bdPath: fakeBdPathMCP}
	if err := state.init(context.Background()); err != nil {
		t.Fatalf("state.init() failed: %v", err)
	}
	cs := connectInProcess(t, state)

	result, err := cs.CallTool(context.Background(), &mcpsdk.CallToolParams{
		Name:      "execute_wave",
		Arguments: map[string]any{"phase_num": 1},
	})
	if err != nil {
		t.Fatalf("CallTool(execute_wave) returned error: %v", err)
	}
	if result.IsError {
		t.Fatalf("CallTool(execute_wave) returned IsError=true: %v", contentText(result))
	}

	text := contentText(result)
	var resp executeWaveResult
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("execute_wave response is not valid JSON: %v, text: %s", err, text)
	}

	if len(resp.Tasks) == 0 {
		t.Fatalf("execute_wave expected non-empty tasks array, got 0 tasks")
	}

	task := resp.Tasks[0]
	if task.BeadID == "" {
		t.Errorf("task.BeadID is empty")
	}
	if task.PlanID == "" {
		t.Errorf("task.PlanID is empty")
	}
	if task.Title == "" {
		t.Errorf("task.Title is empty")
	}
	if task.AcceptanceCriteria == "" {
		t.Errorf("task.AcceptanceCriteria is empty")
	}
}

// TestExecuteWaveContextChain verifies that dep_summaries contains CloseReason from
// each closed dependency bead when a ready task has dependencies.
func TestExecuteWaveContextChain(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, ".beads"), 0755); err != nil {
		t.Fatal(err)
	}

	// Create a dependency bead JSON (closed with a close_reason).
	depBead := map[string]any{
		"id":           "bd-dep-001",
		"title":        "Closed dependency",
		"status":       "closed",
		"close_reason": "Dependency completed: graph primitives done",
	}
	depData, err := json.Marshal(depBead)
	if err != nil {
		t.Fatal(err)
	}
	showFile := filepath.Join(tmpDir, "show.json")
	if err := os.WriteFile(showFile, depData, 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FAKE_BD_SHOW_RESPONSE", showFile)

	// Create a ready bead with dependencies.
	readyBead := map[string]any{
		"id":                  "bd-task-002",
		"title":               "Task with dependency",
		"status":              "open",
		"acceptance_criteria": "works with deps",
		"metadata": map[string]any{
			"gsd_phase": 1,
			"gsd_plan":  "07-01",
		},
		"dependencies": []map[string]any{
			{"issue_id": "bd-task-002", "depends_on_id": "bd-dep-001"},
		},
	}
	readyData, err := json.Marshal([]any{readyBead})
	if err != nil {
		t.Fatal(err)
	}
	readyFile := filepath.Join(tmpDir, "ready.json")
	if err := os.WriteFile(readyFile, readyData, 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FAKE_BD_READY_RESPONSE", readyFile)

	state := &serverState{beadsDir: tmpDir, bdPath: fakeBdPathMCP}
	if err := state.init(context.Background()); err != nil {
		t.Fatalf("state.init() failed: %v", err)
	}
	cs := connectInProcess(t, state)

	result, err := cs.CallTool(context.Background(), &mcpsdk.CallToolParams{
		Name:      "execute_wave",
		Arguments: map[string]any{"phase_num": 1},
	})
	if err != nil {
		t.Fatalf("CallTool(execute_wave, deps) returned error: %v", err)
	}
	if result.IsError {
		t.Fatalf("CallTool(execute_wave, deps) returned IsError=true: %v", contentText(result))
	}

	text := contentText(result)
	var resp executeWaveResult
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("execute_wave response is not valid JSON: %v, text: %s", err, text)
	}

	if len(resp.Tasks) == 0 {
		t.Fatalf("expected tasks, got empty array")
	}

	task := resp.Tasks[0]
	if len(task.DepSummaries) == 0 {
		t.Errorf("expected dep_summaries to contain close_reason from dependency, got empty")
	}
	if len(task.DepSummaries) > 0 && task.DepSummaries[0] == "" {
		t.Errorf("dep_summaries[0] is empty, want close_reason text")
	}
}

// TestExecuteWaveEmpty verifies that execute_wave for a phase with no ready tasks
// returns an empty tasks array (not an error).
func TestExecuteWaveEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, ".beads"), 0755); err != nil {
		t.Fatal(err)
	}

	readyFile := filepath.Join(tmpDir, "empty.json")
	if err := os.WriteFile(readyFile, []byte(`[]`), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FAKE_BD_READY_RESPONSE", readyFile)

	state := &serverState{beadsDir: tmpDir, bdPath: fakeBdPathMCP}
	if err := state.init(context.Background()); err != nil {
		t.Fatalf("state.init() failed: %v", err)
	}
	cs := connectInProcess(t, state)

	result, err := cs.CallTool(context.Background(), &mcpsdk.CallToolParams{
		Name:      "execute_wave",
		Arguments: map[string]any{"phase_num": 1},
	})
	if err != nil {
		t.Fatalf("CallTool(execute_wave, empty) returned error: %v", err)
	}
	if result.IsError {
		t.Fatalf("CallTool(execute_wave, empty) returned IsError=true: %v", contentText(result))
	}

	text := contentText(result)
	var resp executeWaveResult
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("execute_wave empty response is not valid JSON: %v, text: %s", err, text)
	}
	if len(resp.Tasks) != 0 {
		t.Errorf("expected empty tasks array, got %d tasks", len(resp.Tasks))
	}
}

// TestExecuteWaveCompaction verifies that execute_wave prefers gsd:compact metadata
// over CloseReason for dep summaries when the compact field is present.
func TestExecuteWaveCompaction(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, ".beads"), 0755); err != nil {
		t.Fatal(err)
	}

	// Create a dep bead with both gsd:compact metadata and CloseReason.
	// The test verifies compact value is preferred over close_reason.
	depBead := map[string]any{
		"id":           "bd-dep-compact",
		"title":        "Closed dep with compact",
		"status":       "closed",
		"close_reason": "full close reason — should NOT be used",
		"metadata": map[string]any{
			"gsd:compact": "compact summary",
		},
	}
	depData, err := json.Marshal(depBead)
	if err != nil {
		t.Fatal(err)
	}
	showFile := filepath.Join(tmpDir, "show-compact.json")
	if err := os.WriteFile(showFile, depData, 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FAKE_BD_SHOW_RESPONSE", showFile)

	// Create a ready bead that depends on the compacted dep.
	readyBead := map[string]any{
		"id":                  "bd-task-compact",
		"title":               "Task needing compact dep",
		"status":              "open",
		"acceptance_criteria": "works with compact",
		"metadata": map[string]any{
			"gsd_phase": 1,
			"gsd_plan":  "09-02",
		},
		"dependencies": []map[string]any{
			{"issue_id": "bd-task-compact", "depends_on_id": "bd-dep-compact"},
		},
	}
	readyData, err := json.Marshal([]any{readyBead})
	if err != nil {
		t.Fatal(err)
	}
	readyFile := filepath.Join(tmpDir, "ready-compact.json")
	if err := os.WriteFile(readyFile, readyData, 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FAKE_BD_READY_RESPONSE", readyFile)

	state := &serverState{beadsDir: tmpDir, bdPath: fakeBdPathMCP}
	if err := state.init(context.Background()); err != nil {
		t.Fatalf("state.init() failed: %v", err)
	}
	cs := connectInProcess(t, state)

	result, err := cs.CallTool(context.Background(), &mcpsdk.CallToolParams{
		Name:      "execute_wave",
		Arguments: map[string]any{"phase_num": 1},
	})
	if err != nil {
		t.Fatalf("CallTool(execute_wave, compaction) returned error: %v", err)
	}
	if result.IsError {
		t.Fatalf("CallTool(execute_wave, compaction) returned IsError=true: %v", contentText(result))
	}

	text := contentText(result)
	var resp executeWaveResult
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("execute_wave compaction response is not valid JSON: %v, text: %s", err, text)
	}

	if len(resp.Tasks) == 0 {
		t.Fatalf("expected tasks, got empty array")
	}

	task := resp.Tasks[0]
	if len(task.DepSummaries) == 0 {
		t.Errorf("expected dep_summaries to contain compact summary, got empty")
	}
	if len(task.DepSummaries) > 0 {
		got := task.DepSummaries[0]
		if got != "compact summary" {
			t.Errorf("dep_summaries[0] = %q, want %q (gsd:compact preferred over close_reason)", got, "compact summary")
		}
	}
}

// TestPhaseNumFromMeta_NilMeta verifies phaseNumFromMeta returns 0 for nil metadata.
func TestPhaseNumFromMeta_NilMeta(t *testing.T) {
	if got := phaseNumFromMeta(nil); got != 0 {
		t.Errorf("phaseNumFromMeta(nil) = %d, want 0", got)
	}
}

// TestPhaseNumFromMeta_MissingKey verifies phaseNumFromMeta returns 0 when key is absent.
func TestPhaseNumFromMeta_MissingKey(t *testing.T) {
	meta := map[string]any{"other_key": 42}
	if got := phaseNumFromMeta(meta); got != 0 {
		t.Errorf("phaseNumFromMeta(missing key) = %d, want 0", got)
	}
}

// TestPhaseNumFromMeta_Float64 verifies JSON-unmarshaled float64 is correctly converted.
func TestPhaseNumFromMeta_Float64(t *testing.T) {
	meta := map[string]any{"gsd_phase": float64(7)}
	if got := phaseNumFromMeta(meta); got != 7 {
		t.Errorf("phaseNumFromMeta(float64(7)) = %d, want 7", got)
	}
}

// TestPhaseNumFromMeta_Int verifies direct int construction works.
func TestPhaseNumFromMeta_Int(t *testing.T) {
	meta := map[string]any{"gsd_phase": 3}
	if got := phaseNumFromMeta(meta); got != 3 {
		t.Errorf("phaseNumFromMeta(int 3) = %d, want 3", got)
	}
}

// TestPhaseNumFromMeta_Int64 verifies int64 type is handled.
func TestPhaseNumFromMeta_Int64(t *testing.T) {
	meta := map[string]any{"gsd_phase": int64(14)}
	if got := phaseNumFromMeta(meta); got != 14 {
		t.Errorf("phaseNumFromMeta(int64(14)) = %d, want 14", got)
	}
}

// TestPhaseNumFromMeta_StringType verifies wrong type returns 0.
func TestPhaseNumFromMeta_StringType(t *testing.T) {
	meta := map[string]any{"gsd_phase": "five"}
	if got := phaseNumFromMeta(meta); got != 0 {
		t.Errorf("phaseNumFromMeta(string) = %d, want 0", got)
	}
}

// TestPhaseNumFromMeta_EmptyMap verifies empty map returns 0.
func TestPhaseNumFromMeta_EmptyMap(t *testing.T) {
	meta := map[string]any{}
	if got := phaseNumFromMeta(meta); got != 0 {
		t.Errorf("phaseNumFromMeta(empty) = %d, want 0", got)
	}
}

// TestExecuteWaveNoPhase verifies that execute_wave with a non-existent phase_num
// returns toolError "no phase epic found".
func TestExecuteWaveNoPhase(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, ".beads"), 0755); err != nil {
		t.Fatal(err)
	}

	state := &serverState{beadsDir: tmpDir, bdPath: fakeBdPathMCP}
	if err := state.init(context.Background()); err != nil {
		t.Fatalf("state.init() failed: %v", err)
	}
	cs := connectInProcess(t, state)

	// phase_num=99 does not exist in fake_bd (cannedPhaseBead has gsd_phase:1).
	result, err := cs.CallTool(context.Background(), &mcpsdk.CallToolParams{
		Name:      "execute_wave",
		Arguments: map[string]any{"phase_num": 99},
	})
	if err != nil {
		t.Fatalf("CallTool(execute_wave, no-phase) returned protocol error: %v", err)
	}
	if !result.IsError {
		t.Fatalf("execute_wave with non-existent phase_num should return IsError=true, got: %s", contentText(result))
	}
	text := contentText(result)
	if !contains(text, "no phase epic found") {
		t.Errorf("execute_wave error message should contain 'no phase epic found', got: %s", text)
	}
}
