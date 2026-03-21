package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// connectInProcess creates an in-process server/client pair for testing.
// The server has the given state and all tools registered.
// Returns the client session — caller must cancel ctx to shut down.
func connectInProcess(t *testing.T, state *serverState) *mcpsdk.ClientSession {
	t.Helper()
	server := mcpsdk.NewServer(&mcpsdk.Implementation{Name: "test", Version: "0"}, nil)
	registerTools(server, state)

	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "test-client", Version: "0"}, nil)
	t1, t2 := mcpsdk.NewInMemoryTransports()

	ctx := context.Background()
	if _, err := server.Connect(ctx, t1, nil); err != nil {
		t.Fatalf("server.Connect() failed: %v", err)
	}
	cs, err := client.Connect(ctx, t2, nil)
	if err != nil {
		t.Fatalf("client.Connect() failed: %v", err)
	}
	t.Cleanup(func() { cs.Close() })
	return cs
}

// TestToolsRegistered verifies that registerTools adds exactly 10 tools to the server.
func TestToolsRegistered(t *testing.T) {
	state := &serverState{}
	cs := connectInProcess(t, state)

	result, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools() returned error: %v", err)
	}
	if len(result.Tools) != 10 {
		names := make([]string, len(result.Tools))
		for i, tool := range result.Tools {
			names[i] = tool.Name
		}
		t.Errorf("expected 10 tools, got %d: %v", len(result.Tools), names)
	}

	wantNames := []string{
		"create_phase",
		"create_plan",
		"get_bead",
		"list_ready",
		"query_by_label",
		"claim_bead",
		"close_plan",
		"flush_writes",
		"init_project",
		"get_status",
	}
	toolMap := make(map[string]bool)
	for _, tool := range result.Tools {
		toolMap[tool.Name] = true
	}
	for _, name := range wantNames {
		if !toolMap[name] {
			t.Errorf("expected tool %q not found in registered tools", name)
		}
	}
}

// TestToolCallCreatePhase verifies the create_phase handler unmarshals args and returns a JSON bead.
func TestToolCallCreatePhase(t *testing.T) {
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
		Name: "create_phase",
		Arguments: map[string]any{
			"phase_num":  1,
			"title":      "Test Phase",
			"goal":       "do stuff",
			"acceptance": "it works",
			"req_ids":    []string{"INFRA-01"},
		},
	})
	if err != nil {
		t.Fatalf("CallTool(create_phase) returned error: %v", err)
	}
	if result.IsError {
		t.Fatalf("CallTool(create_phase) returned IsError=true: %v", contentText(result))
	}
	if len(result.Content) == 0 {
		t.Fatal("CallTool(create_phase) returned empty Content")
	}
	text := contentText(result)
	var bead map[string]any
	if err := json.Unmarshal([]byte(text), &bead); err != nil {
		t.Fatalf("create_phase response is not valid JSON bead: %v, text: %s", err, text)
	}
	if bead["id"] == nil {
		t.Errorf("create_phase response missing 'id' field: %v", bead)
	}
}

// TestToolCallGetBead verifies the get_bead handler returns a JSON bead.
func TestToolCallGetBead(t *testing.T) {
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
		Name:      "get_bead",
		Arguments: map[string]any{"id": "bd-test-abc"},
	})
	if err != nil {
		t.Fatalf("CallTool(get_bead) returned error: %v", err)
	}
	if result.IsError {
		t.Fatalf("CallTool(get_bead) returned IsError=true: %v", contentText(result))
	}
	text := contentText(result)
	var bead map[string]any
	if err := json.Unmarshal([]byte(text), &bead); err != nil {
		t.Fatalf("get_bead response is not valid JSON: %v, text: %s", err, text)
	}
	if bead["id"] == nil {
		t.Errorf("get_bead response missing 'id' field: %v", bead)
	}
}

// TestToolCallBadArgs verifies that a handler returns IsError=true with descriptive message on malformed args.
func TestToolCallBadArgs(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, ".beads"), 0755); err != nil {
		t.Fatal(err)
	}

	state := &serverState{beadsDir: tmpDir, bdPath: fakeBdPathMCP}
	if err := state.init(context.Background()); err != nil {
		t.Fatalf("state.init() failed: %v", err)
	}

	cs := connectInProcess(t, state)

	// Pass phase_num as a string to cause unmarshal error (expects integer).
	result, err := cs.CallTool(context.Background(), &mcpsdk.CallToolParams{
		Name: "create_phase",
		Arguments: map[string]any{
			"phase_num":  "not-an-int", // wrong type — string instead of integer
			"title":      "Test",
			"goal":       "goal",
			"acceptance": "ok",
		},
	})
	if err != nil {
		t.Fatalf("CallTool(create_phase, bad args) unexpected protocol error: %v", err)
	}
	if !result.IsError {
		t.Fatalf("expected IsError=true for bad args, got IsError=false, text: %s", contentText(result))
	}
}

// TestToolCallInitError verifies that a handler returns IsError=true when state.init() fails.
func TestToolCallInitError(t *testing.T) {
	// No .beads/ dir so bd init will be triggered; non-existent bd so it fails.
	state := &serverState{
		beadsDir:    t.TempDir(),
		bdPath:      "/nonexistent/fake_bd",
		initTimeout: 100,
	}

	cs := connectInProcess(t, state)

	result, err := cs.CallTool(context.Background(), &mcpsdk.CallToolParams{
		Name:      "list_ready",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("CallTool(list_ready, init-error) unexpected protocol error: %v", err)
	}
	if !result.IsError {
		t.Fatalf("expected IsError=true when init fails, got IsError=false, text: %s", contentText(result))
	}
}

// TestToolCallFlushWrites verifies flush_writes calls FlushWrites and returns {status:flushed}.
func TestToolCallFlushWrites(t *testing.T) {
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
		Name:      "flush_writes",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("CallTool(flush_writes) returned error: %v", err)
	}
	if result.IsError {
		t.Fatalf("CallTool(flush_writes) returned IsError=true: %v", contentText(result))
	}
	text := contentText(result)
	var resp map[string]any
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("flush_writes response is not valid JSON: %v, text: %s", err, text)
	}
	if resp["status"] != "flushed" {
		t.Errorf("flush_writes expected {\"status\":\"flushed\"}, got: %v", resp)
	}
}

// TestToolCallInitProject verifies init_project tool creates a project bead and reports files_written.
func TestToolCallInitProject(t *testing.T) {
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
		Name: "init_project",
		Arguments: map[string]any{
			"project_name":  "My Test Project",
			"what":          "A test project for testing",
			"why":           "To verify init_project works",
			"who":           "Developers",
			"done_criteria": "All tests pass",
			"tech_stack":    "Go",
			"constraints":   "Must be fast",
			"risks":         "Complexity",
			"mode":          "full",
		},
	})
	if err != nil {
		t.Fatalf("CallTool(init_project) returned error: %v", err)
	}
	if result.IsError {
		t.Fatalf("CallTool(init_project) returned IsError=true: %v", contentText(result))
	}

	text := contentText(result)
	var resp map[string]any
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("init_project response is not valid JSON: %v, text: %s", err, text)
	}
	if resp["project_bead_id"] == nil {
		t.Errorf("init_project response missing 'project_bead_id' field: %v", resp)
	}
	filesWritten, ok := resp["files_written"].([]any)
	if !ok || len(filesWritten) < 2 {
		t.Errorf("init_project response 'files_written' should have at least 2 entries, got: %v", resp["files_written"])
	}
}

// TestToolCallInitProjectWritesFiles verifies init_project writes PROJECT.md and .gsdw/config.json.
func TestToolCallInitProjectWritesFiles(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, ".beads"), 0755); err != nil {
		t.Fatal(err)
	}

	state := &serverState{beadsDir: tmpDir, bdPath: fakeBdPathMCP}
	if err := state.init(context.Background()); err != nil {
		t.Fatalf("state.init() failed: %v", err)
	}

	cs := connectInProcess(t, state)

	const wantWhat = "A project for verifying file writing"
	const wantName = "FileWriteProject"

	result, err := cs.CallTool(context.Background(), &mcpsdk.CallToolParams{
		Name: "init_project",
		Arguments: map[string]any{
			"project_name":  wantName,
			"what":          wantWhat,
			"why":           "To test file output",
			"done_criteria": "Files exist",
			"mode":          "quick",
		},
	})
	if err != nil {
		t.Fatalf("CallTool(init_project) returned error: %v", err)
	}
	if result.IsError {
		t.Fatalf("CallTool(init_project) returned IsError=true: %v", contentText(result))
	}

	// Check PROJECT.md exists and contains expected sections.
	projectMD := filepath.Join(tmpDir, "PROJECT.md")
	data, err := os.ReadFile(projectMD)
	if err != nil {
		t.Fatalf("PROJECT.md not written: %v", err)
	}
	if !contains(string(data), "## What") {
		t.Errorf("PROJECT.md missing '## What' section, content: %s", data)
	}
	if !contains(string(data), wantWhat) {
		t.Errorf("PROJECT.md missing 'what' value %q, content: %s", wantWhat, data)
	}

	// Check .gsdw/config.json exists with project_name.
	configJSON := filepath.Join(tmpDir, ".gsdw", "config.json")
	configData, err := os.ReadFile(configJSON)
	if err != nil {
		t.Fatalf(".gsdw/config.json not written: %v", err)
	}
	var config map[string]any
	if err := json.Unmarshal(configData, &config); err != nil {
		t.Fatalf(".gsdw/config.json is not valid JSON: %v", err)
	}
	if config["project_name"] != wantName {
		t.Errorf(".gsdw/config.json project_name = %v, want %q", config["project_name"], wantName)
	}
}

// TestToolCallInitProjectQuickMode verifies init_project works with only required fields (quick mode).
func TestToolCallInitProjectQuickMode(t *testing.T) {
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
		Name: "init_project",
		Arguments: map[string]any{
			"project_name":  "Quick Project",
			"what":          "Quick init test",
			"why":           "Speed",
			"done_criteria": "Works fast",
			"mode":          "quick",
		},
	})
	if err != nil {
		t.Fatalf("CallTool(init_project, quick) returned error: %v", err)
	}
	if result.IsError {
		t.Fatalf("CallTool(init_project, quick) returned IsError=true: %v", contentText(result))
	}

	text := contentText(result)
	var resp map[string]any
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("init_project quick response is not valid JSON: %v, text: %s", err, text)
	}
	if resp["project_bead_id"] == nil {
		t.Errorf("init_project quick response missing 'project_bead_id': %v", resp)
	}
}

// TestToolCallGetStatus verifies get_status returns expected JSON structure.
func TestToolCallGetStatus(t *testing.T) {
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
	var resp map[string]any
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("get_status response is not valid JSON: %v, text: %s", err, text)
	}
	// Must have project_name and ready_tasks fields (may be empty).
	if _, ok := resp["project_name"]; !ok {
		t.Errorf("get_status response missing 'project_name' field: %v", resp)
	}
	if _, ok := resp["ready_tasks"]; !ok {
		t.Errorf("get_status response missing 'ready_tasks' field: %v", resp)
	}
}

// TestToolCallGetStatusNoBeads verifies get_status degrades gracefully when no beads exist.
func TestToolCallGetStatusNoBeads(t *testing.T) {
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
		Name:      "get_status",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("CallTool(get_status, no beads) returned error: %v", err)
	}
	// Must never return IsError=true on query failures (graceful degradation).
	if result.IsError {
		t.Fatalf("get_status should not return IsError=true when no beads exist: %v", contentText(result))
	}
	text := contentText(result)
	var resp map[string]any
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("get_status empty response is not valid JSON: %v, text: %s", err, text)
	}
}

// contains is a simple substring check helper.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		}())
}

// contentText extracts the text from the first TextContent in a CallToolResult.
func contentText(result *mcpsdk.CallToolResult) string {
	if result == nil || len(result.Content) == 0 {
		return ""
	}
	if tc, ok := result.Content[0].(*mcpsdk.TextContent); ok {
		return tc.Text
	}
	return ""
}
