package hook

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// makePreToolUseInput returns a marshaled PreToolUseInput for the given tool and CWD.
func makePreToolUseInput(t *testing.T, cwd, toolName, toolUseID string) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(PreToolUseInput{
		HookInputBase: HookInputBase{
			SessionID:      "test-session",
			TranscriptPath: "/tmp/test.jsonl",
			CWD:            cwd,
			HookEventName:  "PreToolUse",
		},
		PermissionMode: "default",
		ToolName:       toolName,
		ToolInput:      json.RawMessage(`{}`),
		ToolUseID:      toolUseID,
	})
	if err != nil {
		t.Fatalf("marshal PreToolUseInput: %v", err)
	}
	return json.RawMessage(raw)
}

// TestPreToolUseWriteTool verifies that a write-class tool gets permissionDecision="allow"
// and non-empty additionalContext from the local index.
func TestPreToolUseWriteTool(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .gsdw/ with a minimal index.json so context can be loaded
	gsdwDir := filepath.Join(tmpDir, ".gsdw")
	if err := os.MkdirAll(gsdwDir, 0755); err != nil {
		t.Fatalf("mkdir .gsdw: %v", err)
	}
	indexData := `{"phase_to_id":{"phase-4":"bd-phase-4"},"plan_to_id":{"04-01":"bd-plan-01"}}`
	if err := os.WriteFile(filepath.Join(gsdwDir, "index.json"), []byte(indexData), 0644); err != nil {
		t.Fatalf("write index.json: %v", err)
	}

	hs := &hookState{}
	var buf bytes.Buffer
	raw := makePreToolUseInput(t, tmpDir, "Write", "tu-1")
	if err := handlePreToolUse(context.Background(), raw, hs, &buf); err != nil {
		t.Fatalf("handlePreToolUse returned error: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	var out HookOutput
	if err := json.Unmarshal([]byte(output), &out); err != nil {
		t.Fatalf("output is not valid JSON: %v, output was: %q", err, output)
	}

	// Verify hookSpecificOutput contains permissionDecision="allow"
	hso, ok := out.HookSpecificOutput.(map[string]interface{})
	if !ok {
		// Try re-marshaling to get the specific output
		hsoData, _ := json.Marshal(out.HookSpecificOutput)
		var parsed PreToolUseHookOutput
		if err := json.Unmarshal(hsoData, &parsed); err != nil {
			t.Fatalf("hookSpecificOutput is not PreToolUseHookOutput: %v", err)
		}
		if parsed.PermissionDecision != "allow" {
			t.Errorf("expected permissionDecision=allow, got %q", parsed.PermissionDecision)
		}
		if parsed.HookEventName != "PreToolUse" {
			t.Errorf("expected hookEventName=PreToolUse, got %q", parsed.HookEventName)
		}
		return
	}
	if hso["permissionDecision"] != "allow" {
		t.Errorf("expected permissionDecision=allow, got %v", hso["permissionDecision"])
	}
	if hso["hookEventName"] != "PreToolUse" {
		t.Errorf("expected hookEventName=PreToolUse, got %v", hso["hookEventName"])
	}
	// Non-empty additionalContext from index
	if hso["additionalContext"] == "" {
		t.Error("expected non-empty additionalContext for write-class tool with index")
	}
}

// TestPreToolUseReadTool verifies that a read-class tool gets permissionDecision="allow"
// and empty additionalContext (fast path, no graph queries).
func TestPreToolUseReadTool(t *testing.T) {
	tmpDir := t.TempDir()
	hs := &hookState{}
	var buf bytes.Buffer

	raw := makePreToolUseInput(t, tmpDir, "Read", "tu-2")
	if err := handlePreToolUse(context.Background(), raw, hs, &buf); err != nil {
		t.Fatalf("handlePreToolUse returned error: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	var out HookOutput
	if err := json.Unmarshal([]byte(output), &out); err != nil {
		t.Fatalf("output is not valid JSON: %v, output was: %q", err, output)
	}

	// Fast path: hookSpecificOutput.permissionDecision should be "allow"
	hsoData, _ := json.Marshal(out.HookSpecificOutput)
	var parsed PreToolUseHookOutput
	if err := json.Unmarshal(hsoData, &parsed); err != nil {
		t.Fatalf("hookSpecificOutput is not PreToolUseHookOutput: %v", err)
	}
	if parsed.PermissionDecision != "allow" {
		t.Errorf("expected permissionDecision=allow, got %q", parsed.PermissionDecision)
	}
	// Fast path: no graph query → empty additionalContext
	if parsed.AdditionalContext != "" {
		t.Errorf("expected empty additionalContext for read-class tool, got %q", parsed.AdditionalContext)
	}
}

// TestPreToolUseGlobFastPath verifies Glob tool uses the fast path (same as Read).
func TestPreToolUseGlobFastPath(t *testing.T) {
	tmpDir := t.TempDir()
	hs := &hookState{}
	var buf bytes.Buffer

	raw := makePreToolUseInput(t, tmpDir, "Glob", "tu-3")
	if err := handlePreToolUse(context.Background(), raw, hs, &buf); err != nil {
		t.Fatalf("handlePreToolUse returned error: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	var out HookOutput
	if err := json.Unmarshal([]byte(output), &out); err != nil {
		t.Fatalf("output is not valid JSON: %v, output was: %q", err, output)
	}

	hsoData, _ := json.Marshal(out.HookSpecificOutput)
	var parsed PreToolUseHookOutput
	if err := json.Unmarshal(hsoData, &parsed); err != nil {
		t.Fatalf("hookSpecificOutput is not PreToolUseHookOutput: %v", err)
	}
	if parsed.PermissionDecision != "allow" {
		t.Errorf("expected permissionDecision=allow for Glob, got %q", parsed.PermissionDecision)
	}
	if parsed.AdditionalContext != "" {
		t.Errorf("expected empty additionalContext for Glob fast path, got %q", parsed.AdditionalContext)
	}
}

// TestPreToolUseLatency verifies handlePreToolUse completes within 500ms.
func TestPreToolUseLatency(t *testing.T) {
	tmpDir := t.TempDir()
	hs := &hookState{}
	var buf bytes.Buffer

	raw := makePreToolUseInput(t, tmpDir, "Write", "tu-latency")

	start := time.Now()
	if err := handlePreToolUse(context.Background(), raw, hs, &buf); err != nil {
		t.Fatalf("handlePreToolUse returned error: %v", err)
	}
	elapsed := time.Since(start)

	if elapsed > 500*time.Millisecond {
		t.Errorf("handlePreToolUse took %v, want < 500ms", elapsed)
	}
}

// TestPreToolUseNoBeads verifies graceful degradation when .beads/ does not exist.
func TestPreToolUseNoBeads(t *testing.T) {
	tmpDir := t.TempDir()
	// No .beads/ and no .gsdw/ -- handler must degrade gracefully
	hs := &hookState{}
	var buf bytes.Buffer

	raw := makePreToolUseInput(t, tmpDir, "Write", "tu-nobeads")
	if err := handlePreToolUse(context.Background(), raw, hs, &buf); err != nil {
		t.Fatalf("handlePreToolUse returned error: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	var out HookOutput
	if err := json.Unmarshal([]byte(output), &out); err != nil {
		t.Fatalf("output is not valid JSON: %v, output was: %q", err, output)
	}

	// Still returns allow even without .beads/
	hsoData, _ := json.Marshal(out.HookSpecificOutput)
	var parsed PreToolUseHookOutput
	if err := json.Unmarshal(hsoData, &parsed); err != nil {
		t.Fatalf("hookSpecificOutput is not PreToolUseHookOutput: %v", err)
	}
	if parsed.PermissionDecision != "allow" {
		t.Errorf("expected permissionDecision=allow even without .beads/, got %q", parsed.PermissionDecision)
	}
}

// TestPreToolUseStdoutPurity verifies valid JSON output for both read and write tools.
func TestPreToolUseStdoutPurity(t *testing.T) {
	tmpDir := t.TempDir()

	for _, toolName := range []string{"Read", "Write"} {
		t.Run(toolName, func(t *testing.T) {
			hs := &hookState{}
			var buf bytes.Buffer
			raw := makePreToolUseInput(t, tmpDir, toolName, "tu-purity")
			if err := handlePreToolUse(context.Background(), raw, hs, &buf); err != nil {
				t.Fatalf("handlePreToolUse returned error: %v", err)
			}
			output := buf.Bytes()
			if len(output) == 0 {
				t.Fatal("expected non-empty output")
			}
			trimmed := bytes.TrimSpace(output)
			if trimmed[0] != '{' {
				t.Errorf("expected output to start with '{', got %q", string(trimmed[:1]))
			}
			var m map[string]interface{}
			if err := json.Unmarshal(trimmed, &m); err != nil {
				t.Errorf("stdout output is not valid JSON: %v, raw: %q", err, string(output))
			}
		})
	}
}
