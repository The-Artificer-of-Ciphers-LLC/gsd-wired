package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestGetTieredContext verifies the get_tiered_context tool returns hot/warm/cold arrays
// plus a context_string and estimated_tokens field.
func TestGetTieredContext(t *testing.T) {
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
		Name: "get_tiered_context",
		Arguments: map[string]any{
			"phase_num":     1,
			"budget_tokens": 2000,
		},
	})
	if err != nil {
		t.Fatalf("CallTool(get_tiered_context) returned error: %v", err)
	}
	if result.IsError {
		t.Fatalf("CallTool(get_tiered_context) returned IsError=true: %v", contentText(result))
	}

	text := contentText(result)
	var resp map[string]any
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("get_tiered_context response is not valid JSON: %v, text: %s", err, text)
	}

	// Must have hot, warm, cold arrays.
	if _, ok := resp["hot"]; !ok {
		t.Errorf("get_tiered_context response missing 'hot' field: %v", resp)
	}
	if _, ok := resp["warm"]; !ok {
		t.Errorf("get_tiered_context response missing 'warm' field: %v", resp)
	}
	if _, ok := resp["cold"]; !ok {
		t.Errorf("get_tiered_context response missing 'cold' field: %v", resp)
	}
	// Must have context_string and estimated_tokens.
	if _, ok := resp["context_string"]; !ok {
		t.Errorf("get_tiered_context response missing 'context_string' field: %v", resp)
	}
	if _, ok := resp["estimated_tokens"]; !ok {
		t.Errorf("get_tiered_context response missing 'estimated_tokens' field: %v", resp)
	}
}

// TestGetTieredContextDefaultBudget verifies budget_tokens defaults to 2000 when omitted.
func TestGetTieredContextDefaultBudget(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, ".beads"), 0755); err != nil {
		t.Fatal(err)
	}

	state := &serverState{beadsDir: tmpDir, bdPath: fakeBdPathMCP}
	if err := state.init(context.Background()); err != nil {
		t.Fatalf("state.init() failed: %v", err)
	}

	cs := connectInProcess(t, state)

	// Omit budget_tokens — should default to 2000.
	result, err := cs.CallTool(context.Background(), &mcpsdk.CallToolParams{
		Name: "get_tiered_context",
		Arguments: map[string]any{
			"phase_num": 1,
		},
	})
	if err != nil {
		t.Fatalf("CallTool(get_tiered_context, no budget) returned error: %v", err)
	}
	if result.IsError {
		t.Fatalf("CallTool(get_tiered_context, no budget) returned IsError=true: %v", contentText(result))
	}

	text := contentText(result)
	var resp map[string]any
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("get_tiered_context response is not valid JSON: %v, text: %s", err, text)
	}
	if _, ok := resp["context_string"]; !ok {
		t.Errorf("get_tiered_context default budget response missing 'context_string': %v", resp)
	}
}

// TestToolCountIs18 verifies that exactly 18 tools are registered.
func TestToolCountIs18(t *testing.T) {
	state := &serverState{}
	cs := connectInProcess(t, state)

	result, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools() returned error: %v", err)
	}
	if len(result.Tools) != 18 {
		names := make([]string, len(result.Tools))
		for i, tool := range result.Tools {
			names[i] = tool.Name
		}
		t.Errorf("expected 18 tools, got %d: %v", len(result.Tools), names)
	}
}
