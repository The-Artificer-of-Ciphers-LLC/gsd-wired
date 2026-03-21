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

// TestSessionStartEmitsContext verifies that handleSessionStart emits additionalContext
// when .beads/ directory is present and graph client initializes.
func TestSessionStartEmitsContext(t *testing.T) {
	fakeBd := buildFakeBd(t)
	tmpDir := t.TempDir()

	// Create .beads/ to trigger normal context loading
	beadsDir := filepath.Join(tmpDir, ".beads")
	if err := os.MkdirAll(beadsDir, 0755); err != nil {
		t.Fatalf("failed to create .beads/: %v", err)
	}

	hs := &hookState{
		bdPath:   fakeBd,
		beadsDir: tmpDir,
	}

	raw, err := json.Marshal(SessionStartInput{
		HookInputBase: HookInputBase{
			SessionID:     "t1",
			CWD:           tmpDir,
			HookEventName: "SessionStart",
		},
		Source: "startup",
	})
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var buf bytes.Buffer
	ctx := context.Background()
	if err := handleSessionStart(ctx, raw, hs, &buf); err != nil {
		t.Fatalf("handleSessionStart returned error: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	var out HookOutput
	if err := json.Unmarshal([]byte(output), &out); err != nil {
		t.Fatalf("output is not valid JSON: %v, output was: %q", err, output)
	}
	// fake_bd returns empty ready list and empty query list, so context may be minimal
	// but AdditionalContext should be non-empty (at least the header)
	if out.AdditionalContext == "" {
		t.Error("expected non-empty additionalContext")
	}
}

// TestSessionStartNoBeads verifies that handleSessionStart emits a hint
// when .beads/ directory does not exist (uninitialized project).
func TestSessionStartNoBeads(t *testing.T) {
	tmpDir := t.TempDir()
	// No .beads/ created

	hs := &hookState{} // no bdPath needed — fast path exits before init

	raw, err := json.Marshal(SessionStartInput{
		HookInputBase: HookInputBase{
			SessionID:     "t2",
			CWD:           tmpDir,
			HookEventName: "SessionStart",
		},
		Source: "startup",
	})
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var buf bytes.Buffer
	ctx := context.Background()
	if err := handleSessionStart(ctx, raw, hs, &buf); err != nil {
		t.Fatalf("handleSessionStart returned error: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	var out HookOutput
	if err := json.Unmarshal([]byte(output), &out); err != nil {
		t.Fatalf("output is not valid JSON: %v, output was: %q", err, output)
	}

	if out.AdditionalContext == "" {
		t.Error("expected non-empty additionalContext hint for uninitialized project")
	}
	// Must contain "init" or "No .beads" (per D-05)
	if !strings.Contains(out.AdditionalContext, "init") && !strings.Contains(out.AdditionalContext, "No .beads") {
		t.Errorf("expected hint to mention 'init' or 'No .beads', got: %q", out.AdditionalContext)
	}
}

// TestSessionStartInitError verifies handleSessionStart degrades gracefully
// when graph client init fails (nonexistent bd binary).
func TestSessionStartInitError(t *testing.T) {
	tmpDir := t.TempDir()
	// Create .beads/ to bypass fast path
	if err := os.MkdirAll(filepath.Join(tmpDir, ".beads"), 0755); err != nil {
		t.Fatalf("failed to create .beads/: %v", err)
	}

	hs := &hookState{
		bdPath:   "/nonexistent/bd",
		beadsDir: tmpDir,
	}

	raw, err := json.Marshal(SessionStartInput{
		HookInputBase: HookInputBase{
			SessionID:     "t3",
			CWD:           tmpDir,
			HookEventName: "SessionStart",
		},
		Source: "startup",
	})
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var buf bytes.Buffer
	ctx := context.Background()
	// Must NOT return error — degrades gracefully (D-06)
	if err := handleSessionStart(ctx, raw, hs, &buf); err != nil {
		t.Fatalf("expected no error in degraded mode, got: %v", err)
	}

	// Output must be valid JSON
	output := strings.TrimSpace(buf.String())
	if len(output) == 0 {
		t.Fatal("expected non-empty output even in degraded mode")
	}
	var out HookOutput
	if err := json.Unmarshal([]byte(output), &out); err != nil {
		t.Fatalf("output is not valid JSON in degraded mode: %v, output was: %q", err, output)
	}
}

// TestSessionStartLatency verifies handleSessionStart completes within the 2s budget.
func TestSessionStartLatency(t *testing.T) {
	fakeBd := buildFakeBd(t)
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, ".beads"), 0755); err != nil {
		t.Fatalf("failed to create .beads/: %v", err)
	}

	hs := &hookState{
		bdPath:   fakeBd,
		beadsDir: tmpDir,
	}

	raw, err := json.Marshal(SessionStartInput{
		HookInputBase: HookInputBase{
			SessionID:     "t4",
			CWD:           tmpDir,
			HookEventName: "SessionStart",
		},
		Source: "startup",
	})
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var buf bytes.Buffer
	ctx := context.Background()

	start := time.Now()
	if err := handleSessionStart(ctx, raw, hs, &buf); err != nil {
		t.Fatalf("handleSessionStart returned error: %v", err)
	}
	elapsed := time.Since(start)

	if elapsed > 2*time.Second {
		t.Errorf("handleSessionStart took %v, want < 2s", elapsed)
	}
}

// TestSessionStartStdoutPurity verifies output starts with '{' and is valid JSON.
func TestSessionStartStdoutPurity(t *testing.T) {
	fakeBd := buildFakeBd(t)
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, ".beads"), 0755); err != nil {
		t.Fatalf("failed to create .beads/: %v", err)
	}

	hs := &hookState{
		bdPath:   fakeBd,
		beadsDir: tmpDir,
	}

	raw, err := json.Marshal(SessionStartInput{
		HookInputBase: HookInputBase{
			SessionID:     "t5",
			CWD:           tmpDir,
			HookEventName: "SessionStart",
		},
		Source: "startup",
	})
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var buf bytes.Buffer
	ctx := context.Background()
	if err := handleSessionStart(ctx, raw, hs, &buf); err != nil {
		t.Fatalf("handleSessionStart returned error: %v", err)
	}

	output := buf.Bytes()
	if len(output) == 0 {
		t.Fatal("expected non-empty output")
	}
	// First byte must be '{'
	trimmed := bytes.TrimSpace(output)
	if trimmed[0] != '{' {
		t.Errorf("expected output to start with '{', got %q", string(trimmed[:1]))
	}
	// Entire output must be valid JSON
	var m map[string]interface{}
	if err := json.Unmarshal(trimmed, &m); err != nil {
		t.Errorf("stdout output is not valid JSON: %v, raw output: %q", err, string(output))
	}
}
