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
