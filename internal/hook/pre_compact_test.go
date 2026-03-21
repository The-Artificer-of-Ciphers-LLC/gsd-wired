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

// makePreCompactInput returns a marshaled PreCompactInput for the given CWD.
func makePreCompactInput(t *testing.T, cwd, trigger string) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(PreCompactInput{
		HookInputBase: HookInputBase{
			SessionID:      "test-session",
			TranscriptPath: "/tmp/test.jsonl",
			CWD:            cwd,
			HookEventName:  "PreCompact",
		},
		Trigger:            trigger,
		CustomInstructions: "",
	})
	if err != nil {
		t.Fatalf("marshal PreCompactInput: %v", err)
	}
	return json.RawMessage(raw)
}

// TestPreCompactLocalWrite verifies that handlePreCompact creates the snapshot file
// with the correct fields (session_id, trigger, timestamp).
func TestPreCompactLocalWrite(t *testing.T) {
	tmpDir := t.TempDir()
	hs := &hookState{}
	var buf bytes.Buffer

	raw := makePreCompactInput(t, tmpDir, "auto")
	if err := handlePreCompact(context.Background(), raw, hs, &buf); err != nil {
		t.Fatalf("handlePreCompact returned error: %v", err)
	}

	snapshotPath := filepath.Join(tmpDir, ".gsdw", "precompact-snapshot.json")
	data, err := os.ReadFile(snapshotPath)
	if err != nil {
		t.Fatalf("snapshot file not found at %s: %v", snapshotPath, err)
	}

	var snapshot map[string]interface{}
	if err := json.Unmarshal(data, &snapshot); err != nil {
		t.Fatalf("snapshot is not valid JSON: %v", err)
	}

	if snapshot["session_id"] != "test-session" {
		t.Errorf("expected session_id=test-session, got %v", snapshot["session_id"])
	}
	if snapshot["trigger"] != "auto" {
		t.Errorf("expected trigger=auto, got %v", snapshot["trigger"])
	}
	ts, ok := snapshot["timestamp"].(string)
	if !ok || ts == "" {
		t.Errorf("expected non-empty timestamp, got %v", snapshot["timestamp"])
	}
}

// TestPreCompactOutput verifies that handlePreCompact writes empty HookOutput ({}).
func TestPreCompactOutput(t *testing.T) {
	tmpDir := t.TempDir()
	hs := &hookState{}
	var buf bytes.Buffer

	raw := makePreCompactInput(t, tmpDir, "manual")
	if err := handlePreCompact(context.Background(), raw, hs, &buf); err != nil {
		t.Fatalf("handlePreCompact returned error: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	var out HookOutput
	if err := json.Unmarshal([]byte(output), &out); err != nil {
		t.Fatalf("output is not valid JSON: %v, output was: %q", err, output)
	}

	// Empty HookOutput: no additionalContext, no hookSpecificOutput, no stopReason
	if out.AdditionalContext != "" {
		t.Errorf("expected empty additionalContext, got: %q", out.AdditionalContext)
	}
	if out.HookSpecificOutput != nil {
		t.Errorf("expected nil hookSpecificOutput, got: %v", out.HookSpecificOutput)
	}
	if out.StopReason != "" {
		t.Errorf("expected empty stopReason, got: %q", out.StopReason)
	}
}

// TestPreCompactAtomicWrite verifies that .tmp file does not exist after completion
// (the rename succeeded atomically).
func TestPreCompactAtomicWrite(t *testing.T) {
	tmpDir := t.TempDir()
	hs := &hookState{}
	var buf bytes.Buffer

	raw := makePreCompactInput(t, tmpDir, "auto")
	if err := handlePreCompact(context.Background(), raw, hs, &buf); err != nil {
		t.Fatalf("handlePreCompact returned error: %v", err)
	}

	tmpFile := filepath.Join(tmpDir, ".gsdw", "precompact-snapshot.json.tmp")
	if _, err := os.Stat(tmpFile); !os.IsNotExist(err) {
		t.Errorf("expected .tmp file to not exist after successful rename, but found it")
	}
}

// TestPreCompactCreatesGsdwDir verifies that .gsdw/ is created if it does not exist.
func TestPreCompactCreatesGsdwDir(t *testing.T) {
	tmpDir := t.TempDir()
	// No .gsdw/ created — handler must create it
	gsdwDir := filepath.Join(tmpDir, ".gsdw")
	if _, err := os.Stat(gsdwDir); !os.IsNotExist(err) {
		t.Fatal("precondition failed: .gsdw/ should not exist yet")
	}

	hs := &hookState{}
	var buf bytes.Buffer
	raw := makePreCompactInput(t, tmpDir, "auto")
	if err := handlePreCompact(context.Background(), raw, hs, &buf); err != nil {
		t.Fatalf("handlePreCompact returned error: %v", err)
	}

	if _, err := os.Stat(gsdwDir); os.IsNotExist(err) {
		t.Error("expected .gsdw/ to be created by handlePreCompact")
	}
}

// TestPreCompactWriteError verifies that handlePreCompact does NOT return an error
// when the CWD is nonexistent (best-effort write, best-effort write failure tolerance).
func TestPreCompactWriteError(t *testing.T) {
	hs := &hookState{}
	var buf bytes.Buffer

	// Use a nonexistent path — MkdirAll will fail
	raw := makePreCompactInput(t, "/nonexistent/path/that/cannot/be/created/ever", "auto")

	// Must NOT return an error — PreCompact is best-effort
	err := handlePreCompact(context.Background(), raw, hs, &buf)
	if err != nil {
		t.Fatalf("expected no error for best-effort write failure, got: %v", err)
	}

	// Output must still be valid JSON
	output := strings.TrimSpace(buf.String())
	if len(output) == 0 {
		t.Fatal("expected non-empty output even when write fails")
	}
	var out HookOutput
	if err := json.Unmarshal([]byte(output), &out); err != nil {
		t.Fatalf("output is not valid JSON after write error: %v, output was: %q", err, output)
	}
}

// TestPreCompactStdoutPurity verifies the first byte of stdout is '{'.
func TestPreCompactStdoutPurity(t *testing.T) {
	tmpDir := t.TempDir()
	hs := &hookState{}
	var buf bytes.Buffer

	raw := makePreCompactInput(t, tmpDir, "auto")
	if err := handlePreCompact(context.Background(), raw, hs, &buf); err != nil {
		t.Fatalf("handlePreCompact returned error: %v", err)
	}

	output := buf.Bytes()
	if len(output) == 0 {
		t.Fatal("expected non-empty output")
	}
	trimmed := bytes.TrimSpace(output)
	if trimmed[0] != '{' {
		t.Errorf("expected output to start with '{', got %q", string(trimmed[:1]))
	}
	// Full output must be valid JSON
	var m map[string]interface{}
	if err := json.Unmarshal(trimmed, &m); err != nil {
		t.Errorf("stdout output is not valid JSON: %v, raw: %q", err, string(output))
	}
}
