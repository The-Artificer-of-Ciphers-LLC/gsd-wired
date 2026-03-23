package hook

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/The-Artificer-of-Ciphers-LLC/gsd-wired/internal/graph"
)

// makePostToolUseInput returns a marshaled PostToolUseInput for the given tool and CWD.
func makePostToolUseInput(t *testing.T, cwd, toolName, toolUseID string) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(PostToolUseInput{
		HookInputBase: HookInputBase{
			SessionID:      "test-session",
			TranscriptPath: "/tmp/test.jsonl",
			CWD:            cwd,
			HookEventName:  "PostToolUse",
		},
		PermissionMode: "default",
		ToolName:       toolName,
		ToolInput:      json.RawMessage(`{}`),
		ToolResponse:   json.RawMessage(`{}`),
		ToolUseID:      toolUseID,
	})
	if err != nil {
		t.Fatalf("marshal PostToolUseInput: %v", err)
	}
	return json.RawMessage(raw)
}

// TestPostToolUseWriteTool verifies that a write-class tool appends to tool-events.jsonl.
func TestPostToolUseWriteTool(t *testing.T) {
	tmpDir := t.TempDir()
	hs := &hookState{}
	var buf bytes.Buffer

	raw := makePostToolUseInput(t, tmpDir, "Write", "tu-write-1")
	if err := handlePostToolUse(context.Background(), raw, hs, &buf); err != nil {
		t.Fatalf("handlePostToolUse returned error: %v", err)
	}

	eventsPath := filepath.Join(tmpDir, ".gsdw", "tool-events.jsonl")
	data, err := os.ReadFile(eventsPath)
	if err != nil {
		t.Fatalf("tool-events.jsonl not found at %s: %v", eventsPath, err)
	}

	// File must contain at least one JSON line
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) == 0 || lines[0] == "" {
		t.Fatal("expected at least one line in tool-events.jsonl")
	}

	var event map[string]interface{}
	if err := json.Unmarshal([]byte(lines[0]), &event); err != nil {
		t.Fatalf("tool-events.jsonl line is not valid JSON: %v", err)
	}
	if event["tool_name"] != "Write" {
		t.Errorf("expected tool_name=Write, got %v", event["tool_name"])
	}
	if event["tool_use_id"] != "tu-write-1" {
		t.Errorf("expected tool_use_id=tu-write-1, got %v", event["tool_use_id"])
	}
	if event["session_id"] != "test-session" {
		t.Errorf("expected session_id=test-session, got %v", event["session_id"])
	}
	ts, ok := event["timestamp"].(string)
	if !ok || ts == "" {
		t.Errorf("expected non-empty timestamp, got %v", event["timestamp"])
	}
}

// TestPostToolUseReadTool verifies that a read-class tool does NOT write to tool-events.jsonl.
func TestPostToolUseReadTool(t *testing.T) {
	tmpDir := t.TempDir()
	hs := &hookState{}
	var buf bytes.Buffer

	raw := makePostToolUseInput(t, tmpDir, "Read", "tu-read-1")
	if err := handlePostToolUse(context.Background(), raw, hs, &buf); err != nil {
		t.Fatalf("handlePostToolUse returned error: %v", err)
	}

	// tool-events.jsonl must NOT exist for read-class tools
	eventsPath := filepath.Join(tmpDir, ".gsdw", "tool-events.jsonl")
	if _, err := os.Stat(eventsPath); !os.IsNotExist(err) {
		t.Errorf("expected tool-events.jsonl to NOT exist for Read tool, but it was created")
	}
}

// TestPostToolUseMultipleWrites verifies that multiple write calls append separate lines.
func TestPostToolUseMultipleWrites(t *testing.T) {
	tmpDir := t.TempDir()
	hs := &hookState{}

	for _, id := range []string{"tu-a", "tu-b"} {
		var buf bytes.Buffer
		raw := makePostToolUseInput(t, tmpDir, "Write", id)
		if err := handlePostToolUse(context.Background(), raw, hs, &buf); err != nil {
			t.Fatalf("handlePostToolUse returned error for %s: %v", id, err)
		}
	}

	eventsPath := filepath.Join(tmpDir, ".gsdw", "tool-events.jsonl")
	f, err := os.Open(eventsPath)
	if err != nil {
		t.Fatalf("tool-events.jsonl not found: %v", err)
	}
	defer f.Close()

	var lineCount int
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		lineCount++
	}

	if lineCount != 2 {
		t.Errorf("expected 2 lines in tool-events.jsonl, got %d", lineCount)
	}
}

// TestPostToolUseOutput verifies that stdout is {} (empty HookOutput) for both
// read and write tools.
func TestPostToolUseOutput(t *testing.T) {
	tmpDir := t.TempDir()

	for _, toolName := range []string{"Read", "Write"} {
		t.Run(toolName, func(t *testing.T) {
			hs := &hookState{}
			var buf bytes.Buffer
			raw := makePostToolUseInput(t, tmpDir, toolName, "tu-out")
			if err := handlePostToolUse(context.Background(), raw, hs, &buf); err != nil {
				t.Fatalf("handlePostToolUse returned error: %v", err)
			}
			output := strings.TrimSpace(buf.String())
			var out HookOutput
			if err := json.Unmarshal([]byte(output), &out); err != nil {
				t.Fatalf("output is not valid JSON: %v, output was: %q", err, output)
			}
			if out.AdditionalContext != "" {
				t.Errorf("expected empty additionalContext, got %q", out.AdditionalContext)
			}
			if out.HookSpecificOutput != nil {
				t.Errorf("expected nil hookSpecificOutput, got %v", out.HookSpecificOutput)
			}
		})
	}
}

// TestPostToolUseStdoutPurity verifies valid JSON output for both tools.
func TestPostToolUseStdoutPurity(t *testing.T) {
	tmpDir := t.TempDir()

	for _, toolName := range []string{"Read", "Edit", "Bash"} {
		t.Run(toolName, func(t *testing.T) {
			hs := &hookState{}
			var buf bytes.Buffer
			raw := makePostToolUseInput(t, tmpDir, toolName, "tu-purity")
			if err := handlePostToolUse(context.Background(), raw, hs, &buf); err != nil {
				t.Fatalf("handlePostToolUse returned error: %v", err)
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

// TestPostToolUseBeadUpdate verifies that a write-class tool triggers both JSONL write
// and bead state update (AddLabel gsd:tool-use) when .beads/ and index are present.
func TestPostToolUseBeadUpdate(t *testing.T) {
	fakeBd := buildFakeBd(t)
	tmpDir := t.TempDir()

	// Create .beads/ to trigger bead update path.
	if err := os.MkdirAll(filepath.Join(tmpDir, ".beads"), 0755); err != nil {
		t.Fatalf("failed to create .beads/: %v", err)
	}

	// Create .gsdw/ with index.json containing an active plan bead.
	gsdwDir := filepath.Join(tmpDir, ".gsdw")
	if err := os.MkdirAll(gsdwDir, 0755); err != nil {
		t.Fatalf("failed to create .gsdw/: %v", err)
	}
	idx := graph.NewIndex()
	idx.PlanToID["04-03"] = "bd-active-plan"
	if err := idx.Save(gsdwDir); err != nil {
		t.Fatalf("failed to save index: %v", err)
	}

	// Set up capture file to verify bd add-label was called.
	captureDir := t.TempDir()
	captureFile := filepath.Join(captureDir, "capture.json")
	t.Setenv("FAKE_BD_CAPTURE_FILE", captureFile)

	hs := &hookState{bdPath: fakeBd, beadUpdateTimeout: 5000} // 5s timeout for test (fake bd needs time)
	var buf bytes.Buffer
	raw := makePostToolUseInput(t, tmpDir, "Write", "tu-bead-update")
	if err := handlePostToolUse(context.Background(), raw, hs, &buf); err != nil {
		t.Fatalf("handlePostToolUse returned error: %v", err)
	}

	// Verify JSONL was written (primary path still works).
	eventsPath := filepath.Join(gsdwDir, "tool-events.jsonl")
	if _, err := os.Stat(eventsPath); os.IsNotExist(err) {
		t.Error("expected tool-events.jsonl to exist after Write tool")
	}

	// Verify bd update --add-label was called.
	data, err := os.ReadFile(captureFile)
	if err != nil {
		t.Fatalf("capture file not found — bd may not have been called: %v", err)
	}
	var capturedArgs []string
	if err := json.Unmarshal(data, &capturedArgs); err != nil {
		t.Fatalf("capture file is not valid JSON: %v", err)
	}
	hasUpdate := false
	hasAddLabel := false
	hasToolUseLabel := false
	for _, a := range capturedArgs {
		if a == "update" {
			hasUpdate = true
		}
		if a == "--add-label" {
			hasAddLabel = true
		}
		if a == "gsd:tool-use" {
			hasToolUseLabel = true
		}
	}
	if !hasUpdate {
		t.Errorf("expected 'update' in captured args, got: %v", capturedArgs)
	}
	if !hasAddLabel {
		t.Errorf("expected '--add-label' in captured args, got: %v", capturedArgs)
	}
	if !hasToolUseLabel {
		t.Errorf("expected 'gsd:tool-use' in captured args, got: %v", capturedArgs)
	}
}

// TestPostToolUseBeadUpdateNoBeads verifies that handlePostToolUse works when .beads/
// does not exist — only JSONL is written, no graph call attempted (regression guard).
func TestPostToolUseBeadUpdateNoBeads(t *testing.T) {
	tmpDir := t.TempDir()
	// No .beads/ created — project is uninitialized.

	// Set up capture file to verify bd was NOT called.
	captureDir := t.TempDir()
	captureFile := filepath.Join(captureDir, "capture.json")
	t.Setenv("FAKE_BD_CAPTURE_FILE", captureFile)

	hs := &hookState{} // no bdPath — graph client must not be initialized
	var buf bytes.Buffer
	raw := makePostToolUseInput(t, tmpDir, "Write", "tu-no-beads")
	if err := handlePostToolUse(context.Background(), raw, hs, &buf); err != nil {
		t.Fatalf("handlePostToolUse returned error: %v", err)
	}

	// JSONL must still be written.
	eventsPath := filepath.Join(tmpDir, ".gsdw", "tool-events.jsonl")
	if _, err := os.Stat(eventsPath); os.IsNotExist(err) {
		t.Error("expected tool-events.jsonl to exist even without .beads/")
	}

	// Capture file must NOT exist (no bd command was run).
	if _, err := os.Stat(captureFile); !os.IsNotExist(err) {
		t.Error("expected capture file to NOT exist when .beads/ is absent — bd should not have been called")
	}

	// Output must be valid JSON.
	output := strings.TrimSpace(buf.String())
	var out HookOutput
	if err := json.Unmarshal([]byte(output), &out); err != nil {
		t.Fatalf("output is not valid JSON: %v, output was: %q", err, output)
	}
}

// TestPostToolUseBeadUpdateError verifies that JSONL is still written and hook returns
// valid JSON when bead update fails (broken bdPath — degraded mode for bead update only).
func TestPostToolUseBeadUpdateError(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .beads/ to trigger bead update path.
	if err := os.MkdirAll(filepath.Join(tmpDir, ".beads"), 0755); err != nil {
		t.Fatalf("failed to create .beads/: %v", err)
	}

	// Create .gsdw/ with index.json containing an active plan bead.
	gsdwDir := filepath.Join(tmpDir, ".gsdw")
	if err := os.MkdirAll(gsdwDir, 0755); err != nil {
		t.Fatalf("failed to create .gsdw/: %v", err)
	}
	idx := graph.NewIndex()
	idx.PlanToID["04-03"] = "bd-active-plan"
	if err := idx.Save(gsdwDir); err != nil {
		t.Fatalf("failed to save index: %v", err)
	}

	// Use broken bdPath — bead update will fail.
	hs := &hookState{bdPath: "/nonexistent/bd"}
	var buf bytes.Buffer
	raw := makePostToolUseInput(t, tmpDir, "Write", "tu-bead-error")
	// Must NOT return error — bead update failure is best-effort.
	if err := handlePostToolUse(context.Background(), raw, hs, &buf); err != nil {
		t.Fatalf("handlePostToolUse returned error: %v", err)
	}

	// JSONL must still be written despite bead update failure.
	eventsPath := filepath.Join(gsdwDir, "tool-events.jsonl")
	if _, err := os.Stat(eventsPath); os.IsNotExist(err) {
		t.Error("expected tool-events.jsonl to exist even when bead update fails")
	}

	// Hook output must be valid JSON.
	output := strings.TrimSpace(buf.String())
	if len(output) == 0 {
		t.Fatal("expected non-empty output even in degraded mode")
	}
	var out HookOutput
	if err := json.Unmarshal([]byte(output), &out); err != nil {
		t.Fatalf("output is not valid JSON in degraded mode: %v, output was: %q", err, output)
	}
}
