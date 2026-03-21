package hook

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// fakeBdPath builds the fake_bd binary and returns its path.
// The binary is built once per test binary via TestMain or lazily here.
func buildFakeBd(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	fakeBdPath := filepath.Join(tmpDir, "fake_bd")
	cmd := exec.Command("go", "build", "-o", fakeBdPath, "./internal/graph/testdata/fake_bd")
	cmd.Dir = filepath.Join(os.Getenv("GOPATH"), "src/github.com/The-Artificer-of-Ciphers-LLC/gsd-wired")
	// Use the module root instead
	repoRoot, err := findRepoRoot()
	if err != nil {
		t.Fatalf("could not find repo root: %v", err)
	}
	cmd = exec.Command("go", "build", "-o", fakeBdPath, "./internal/graph/testdata/fake_bd")
	cmd.Dir = repoRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build fake_bd: %v\n%s", err, out)
	}
	return fakeBdPath
}

// findRepoRoot walks up from the current file's directory to find the module root
// (directory containing go.mod).
func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", os.ErrNotExist
}

// TestHookInputDecode verifies that all per-event input structs decode
// event-specific JSON fields correctly.
func TestHookInputDecode(t *testing.T) {
	t.Run("SessionStartInput", func(t *testing.T) {
		raw := `{"session_id":"s1","transcript_path":"/tmp/t.jsonl","cwd":"/tmp","hook_event_name":"SessionStart","source":"startup","model":"claude-opus","agent_type":"main"}`
		var in SessionStartInput
		if err := json.Unmarshal([]byte(raw), &in); err != nil {
			t.Fatalf("unmarshal failed: %v", err)
		}
		if in.SessionID != "s1" {
			t.Errorf("expected session_id=s1, got %q", in.SessionID)
		}
		if in.Source != "startup" {
			t.Errorf("expected source=startup, got %q", in.Source)
		}
		if in.Model != "claude-opus" {
			t.Errorf("expected model=claude-opus, got %q", in.Model)
		}
		if in.AgentType != "main" {
			t.Errorf("expected agent_type=main, got %q", in.AgentType)
		}
	})

	t.Run("PreCompactInput", func(t *testing.T) {
		raw := `{"session_id":"s2","transcript_path":"/tmp/t.jsonl","cwd":"/tmp","hook_event_name":"PreCompact","trigger":"manual","custom_instructions":"save state"}`
		var in PreCompactInput
		if err := json.Unmarshal([]byte(raw), &in); err != nil {
			t.Fatalf("unmarshal failed: %v", err)
		}
		if in.Trigger != "manual" {
			t.Errorf("expected trigger=manual, got %q", in.Trigger)
		}
		if in.CustomInstructions != "save state" {
			t.Errorf("expected custom_instructions='save state', got %q", in.CustomInstructions)
		}
	})

	t.Run("PreToolUseInput", func(t *testing.T) {
		raw := `{"session_id":"s3","transcript_path":"/tmp/t.jsonl","cwd":"/tmp","hook_event_name":"PreToolUse","permission_mode":"auto","tool_name":"Bash","tool_input":{"command":"ls"},"tool_use_id":"tu-123"}`
		var in PreToolUseInput
		if err := json.Unmarshal([]byte(raw), &in); err != nil {
			t.Fatalf("unmarshal failed: %v", err)
		}
		if in.ToolName != "Bash" {
			t.Errorf("expected tool_name=Bash, got %q", in.ToolName)
		}
		if in.ToolUseID != "tu-123" {
			t.Errorf("expected tool_use_id=tu-123, got %q", in.ToolUseID)
		}
		if in.PermissionMode != "auto" {
			t.Errorf("expected permission_mode=auto, got %q", in.PermissionMode)
		}
		if len(in.ToolInput) == 0 {
			t.Error("expected non-empty tool_input")
		}
	})

	t.Run("PostToolUseInput", func(t *testing.T) {
		raw := `{"session_id":"s4","transcript_path":"/tmp/t.jsonl","cwd":"/tmp","hook_event_name":"PostToolUse","permission_mode":"auto","tool_name":"Read","tool_input":{"file_path":"/tmp/f"},"tool_response":{"output":"content"},"tool_use_id":"tu-456"}`
		var in PostToolUseInput
		if err := json.Unmarshal([]byte(raw), &in); err != nil {
			t.Fatalf("unmarshal failed: %v", err)
		}
		if in.ToolName != "Read" {
			t.Errorf("expected tool_name=Read, got %q", in.ToolName)
		}
		if in.ToolUseID != "tu-456" {
			t.Errorf("expected tool_use_id=tu-456, got %q", in.ToolUseID)
		}
		if len(in.ToolResponse) == 0 {
			t.Error("expected non-empty tool_response")
		}
	})
}

// TestHookOutputMarshal verifies HookOutput marshals additionalContext correctly.
func TestHookOutputMarshal(t *testing.T) {
	out := HookOutput{AdditionalContext: "some context"}
	data, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if string(data) == "" {
		t.Fatal("expected non-empty JSON")
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if v, ok := m["additionalContext"]; !ok || v != "some context" {
		t.Errorf("expected additionalContext='some context', got %v", m["additionalContext"])
	}
}

// TestPreToolUseHookOutputMarshal verifies nested hookSpecificOutput marshals correctly.
func TestPreToolUseHookOutputMarshal(t *testing.T) {
	specific := PreToolUseHookOutput{
		HookEventName:      "PreToolUse",
		PermissionDecision: "allow",
		AdditionalContext:  "tool context",
	}
	out := HookOutput{
		HookSpecificOutput: specific,
	}
	data, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	hso, ok := m["hookSpecificOutput"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected hookSpecificOutput to be object, got %T", m["hookSpecificOutput"])
	}
	if hso["permissionDecision"] != "allow" {
		t.Errorf("expected permissionDecision=allow, got %v", hso["permissionDecision"])
	}
}

// TestHookStateInit verifies hookState.init creates a non-nil graph.Client.
func TestHookStateInit(t *testing.T) {
	fakeBd := buildFakeBd(t)
	beadsDir := t.TempDir()
	hs := &hookState{
		bdPath:   fakeBd,
		beadsDir: beadsDir,
	}
	ctx := context.Background()
	if err := hs.init(ctx); err != nil {
		t.Fatalf("init returned error: %v", err)
	}
	if hs.client == nil {
		t.Error("expected non-nil client after successful init")
	}
}

// TestHookStateInitError verifies hookState.init returns error for nonexistent bdPath.
func TestHookStateInitError(t *testing.T) {
	hs := &hookState{
		bdPath:   "/nonexistent/bd",
		beadsDir: t.TempDir(),
	}
	ctx := context.Background()
	err := hs.init(ctx)
	if err == nil {
		t.Fatal("expected error for nonexistent bdPath, got nil")
	}
}

// TestHookStateOnce verifies init runs exactly once (sync.Once semantics).
func TestHookStateOnce(t *testing.T) {
	fakeBd := buildFakeBd(t)
	beadsDir := t.TempDir()
	hs := &hookState{
		bdPath:   fakeBd,
		beadsDir: beadsDir,
	}
	ctx := context.Background()

	// First call
	err1 := hs.init(ctx)
	client1 := hs.client

	// Second call — must return same result
	err2 := hs.init(ctx)
	client2 := hs.client

	if err1 != err2 {
		t.Errorf("expected same error from both calls: first=%v, second=%v", err1, err2)
	}
	if client1 != client2 {
		t.Error("expected same client pointer from both init calls")
	}
}

// TestWriteOutput verifies writeOutput encodes valid JSON to the writer.
func TestWriteOutput(t *testing.T) {
	var buf bytes.Buffer
	out := HookOutput{AdditionalContext: "test context"}
	if err := writeOutput(&buf, out); err != nil {
		t.Fatalf("writeOutput returned error: %v", err)
	}
	data := buf.Bytes()
	if len(data) == 0 {
		t.Fatal("expected non-empty output")
	}
	var m map[string]interface{}
	if err := json.Unmarshal(bytes.TrimSpace(data), &m); err != nil {
		t.Errorf("output is not valid JSON: %v, output was: %q", err, string(data))
	}
	if m["additionalContext"] != "test context" {
		t.Errorf("expected additionalContext='test context', got %v", m["additionalContext"])
	}
}
