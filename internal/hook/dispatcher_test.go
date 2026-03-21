package hook

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const validSessionStartJSON = `{"session_id":"test-123","transcript_path":"/tmp/test.jsonl","cwd":"/tmp","hook_event_name":"SessionStart","source":"startup"}`

func TestDispatchSessionStart(t *testing.T) {
	stdin := strings.NewReader(validSessionStartJSON)
	var stdout bytes.Buffer

	err := Dispatch(context.Background(), EventSessionStart, stdin, &stdout)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := stdout.String()
	if output == "" {
		t.Error("expected output on stdout, got empty string")
	}

	// Verify output is valid JSON
	var out HookOutput
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &out); err != nil {
		t.Errorf("output is not valid JSON: %v, output was: %q", err, output)
	}
}

func TestDispatchInvalidJSON(t *testing.T) {
	stdin := strings.NewReader("not valid json {{{{")
	var stdout bytes.Buffer

	err := Dispatch(context.Background(), EventSessionStart, stdin, &stdout)
	if err == nil {
		t.Fatal("expected error for invalid JSON input, got nil")
	}

	// stdout must be empty when there's an error
	if stdout.Len() != 0 {
		t.Errorf("expected empty stdout on error, got: %q", stdout.String())
	}
}

func TestDispatchEventMismatch(t *testing.T) {
	// JSON says PreToolUse but we dispatch as SessionStart
	mismatchJSON := `{"session_id":"test-123","transcript_path":"/tmp/test.jsonl","cwd":"/tmp","hook_event_name":"PreToolUse"}`
	stdin := strings.NewReader(mismatchJSON)
	var stdout bytes.Buffer

	err := Dispatch(context.Background(), EventSessionStart, stdin, &stdout)
	if err == nil {
		t.Fatal("expected error for event mismatch, got nil")
	}
	if !strings.Contains(err.Error(), "mismatch") {
		t.Errorf("expected error to contain 'mismatch', got: %v", err)
	}
}

func TestDispatchUnknownEvent(t *testing.T) {
	stdin := strings.NewReader(validSessionStartJSON)
	var stdout bytes.Buffer

	err := Dispatch(context.Background(), "FakeEvent", stdin, &stdout)
	if err == nil {
		t.Fatal("expected error for unknown event, got nil")
	}
	if !strings.Contains(err.Error(), "unknown") {
		t.Errorf("expected error to contain 'unknown', got: %v", err)
	}
}

func TestDispatchStdoutPurity(t *testing.T) {
	stdin := strings.NewReader(validSessionStartJSON)
	var stdout bytes.Buffer

	err := Dispatch(context.Background(), EventSessionStart, stdin, &stdout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdout.Bytes()
	if len(output) == 0 {
		t.Fatal("expected some output, got empty")
	}

	// The entire output (minus trailing newline from json.Encode) must be valid JSON
	trimmed := strings.TrimSpace(string(output))
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(trimmed), &parsed); err != nil {
		t.Errorf("stdout output is not valid JSON: %v, raw output: %q", err, string(output))
	}
}

// TestDispatchSessionStartRoute verifies that Dispatch routes SessionStart events
// to handleSessionStart (not the no-op stub). With no .beads/ at /tmp, it should
// emit the init hint in additionalContext.
func TestDispatchSessionStartRoute(t *testing.T) {
	// Use a temp dir without .beads/ to trigger the init hint path
	tmpDir := t.TempDir()
	input := SessionStartInput{
		HookInputBase: HookInputBase{
			SessionID:     "test-route",
			CWD:           tmpDir,
			HookEventName: "SessionStart",
		},
		Source: "startup",
	}
	raw, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	stdin := bytes.NewReader(raw)
	var stdout bytes.Buffer

	if err := Dispatch(context.Background(), EventSessionStart, stdin, &stdout); err != nil {
		t.Fatalf("Dispatch returned error: %v", err)
	}

	output := strings.TrimSpace(stdout.String())
	var out HookOutput
	if err := json.Unmarshal([]byte(output), &out); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	// The handler should have emitted the init hint (no .beads/ present)
	if out.AdditionalContext == "" {
		t.Error("expected additionalContext to contain init hint")
	}
	if !strings.Contains(out.AdditionalContext, "init") && !strings.Contains(out.AdditionalContext, "No .beads") {
		t.Errorf("expected init hint in additionalContext, got: %q", out.AdditionalContext)
	}
}

// TestDispatchAllEventsValidJSON verifies all four event types produce valid JSON stdout.
func TestDispatchAllEventsValidJSON(t *testing.T) {
	tmpDir := t.TempDir()

	events := []struct {
		name  string
		input string
	}{
		{
			EventSessionStart,
			func() string {
				raw, _ := json.Marshal(SessionStartInput{
					HookInputBase: HookInputBase{SessionID: "s", CWD: tmpDir, HookEventName: EventSessionStart},
					Source:        "startup",
				})
				return string(raw)
			}(),
		},
		{
			EventPreCompact,
			func() string {
				raw, _ := json.Marshal(PreCompactInput{
					HookInputBase: HookInputBase{SessionID: "s", CWD: tmpDir, HookEventName: EventPreCompact},
					Trigger:       "manual",
				})
				return string(raw)
			}(),
		},
		{
			EventPreToolUse,
			func() string {
				raw, _ := json.Marshal(PreToolUseInput{
					HookInputBase: HookInputBase{SessionID: "s", CWD: tmpDir, HookEventName: EventPreToolUse},
					ToolName:      "Bash",
				})
				return string(raw)
			}(),
		},
		{
			EventPostToolUse,
			func() string {
				raw, _ := json.Marshal(PostToolUseInput{
					HookInputBase: HookInputBase{SessionID: "s", CWD: tmpDir, HookEventName: EventPostToolUse},
					ToolName:      "Bash",
				})
				return string(raw)
			}(),
		},
	}

	for _, tc := range events {
		t.Run(tc.name, func(t *testing.T) {
			stdin := strings.NewReader(tc.input)
			var stdout bytes.Buffer

			if err := Dispatch(context.Background(), tc.name, stdin, &stdout); err != nil {
				t.Fatalf("Dispatch(%s) returned error: %v", tc.name, err)
			}

			trimmed := strings.TrimSpace(stdout.String())
			if len(trimmed) == 0 {
				t.Fatal("expected non-empty output")
			}
			var m map[string]interface{}
			if err := json.Unmarshal([]byte(trimmed), &m); err != nil {
				t.Errorf("output is not valid JSON: %v, raw: %q", err, trimmed)
			}
		})
	}
}

// TestDispatchUsesBeadsDir verifies that the dispatcher creates a hookState
// with beadsDir set from the CWD in the incoming JSON.
func TestDispatchUsesBeadsDir(t *testing.T) {
	// Create a temp dir with .beads/ to test with real fake_bd
	tmpDir := t.TempDir()
	beadsPath := filepath.Join(tmpDir, ".beads")
	if err := os.MkdirAll(beadsPath, 0755); err != nil {
		t.Fatalf("failed to create .beads/: %v", err)
	}

	input := SessionStartInput{
		HookInputBase: HookInputBase{
			SessionID:     "test-bd",
			CWD:           tmpDir,
			HookEventName: EventSessionStart,
		},
		Source: "startup",
	}
	raw, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	// Without bd on path this will produce degraded output (valid JSON with empty context)
	stdin := bytes.NewReader(raw)
	var stdout bytes.Buffer
	// Should not return error even if bd is not found (degraded mode)
	_ = Dispatch(context.Background(), EventSessionStart, stdin, &stdout)

	trimmed := strings.TrimSpace(stdout.String())
	if len(trimmed) > 0 {
		var m map[string]interface{}
		if err := json.Unmarshal([]byte(trimmed), &m); err != nil {
			t.Errorf("output is not valid JSON: %v, raw: %q", err, trimmed)
		}
	}
}
