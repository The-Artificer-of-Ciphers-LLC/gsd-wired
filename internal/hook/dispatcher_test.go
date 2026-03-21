package hook

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

const validSessionStartJSON = `{"session_id":"test-123","transcript_path":"/tmp/test.jsonl","cwd":"/tmp","hook_event_name":"SessionStart"}`

func TestDispatchSessionStart(t *testing.T) {
	stdin := strings.NewReader(validSessionStartJSON)
	var stdout bytes.Buffer

	err := Dispatch(EventSessionStart, stdin, &stdout)
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

	err := Dispatch(EventSessionStart, stdin, &stdout)
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

	err := Dispatch(EventSessionStart, stdin, &stdout)
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

	err := Dispatch("FakeEvent", stdin, &stdout)
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

	err := Dispatch(EventSessionStart, stdin, &stdout)
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
