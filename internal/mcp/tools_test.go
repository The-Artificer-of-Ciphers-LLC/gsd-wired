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

// TestToolsRegistered verifies that registerTools adds exactly 8 tools to the server.
func TestToolsRegistered(t *testing.T) {
	state := &serverState{}
	cs := connectInProcess(t, state)

	result, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools() returned error: %v", err)
	}
	if len(result.Tools) != 8 {
		names := make([]string, len(result.Tools))
		for i, tool := range result.Tools {
			names[i] = tool.Name
		}
		t.Errorf("expected 8 tools, got %d: %v", len(result.Tools), names)
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
