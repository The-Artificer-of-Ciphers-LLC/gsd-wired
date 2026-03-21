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

// TestSessionStartSyncsSnapshot verifies that handleSessionStart detects and syncs
// a pending precompact-snapshot.json to Dolt, removing the file on success.
func TestSessionStartSyncsSnapshot(t *testing.T) {
	fakeBd := buildFakeBd(t)
	tmpDir := t.TempDir()

	// Create .beads/ so the handler proceeds past the fast path.
	if err := os.MkdirAll(filepath.Join(tmpDir, ".beads"), 0755); err != nil {
		t.Fatalf("failed to create .beads/: %v", err)
	}

	// Create .gsdw/precompact-snapshot.json with valid snapshot data.
	gsdwDir := filepath.Join(tmpDir, ".gsdw")
	if err := os.MkdirAll(gsdwDir, 0755); err != nil {
		t.Fatalf("failed to create .gsdw/: %v", err)
	}
	snapshotPath := filepath.Join(gsdwDir, "precompact-snapshot.json")
	snapshotData := `{"session_id":"snap-session-1","transcript_path":"/tmp/t.jsonl","trigger":"auto","timestamp":"2026-03-21T19:00:00Z","cwd":"` + tmpDir + `"}`
	if err := os.WriteFile(snapshotPath, []byte(snapshotData), 0644); err != nil {
		t.Fatalf("failed to write snapshot: %v", err)
	}

	hs := &hookState{
		bdPath:   fakeBd,
		beadsDir: tmpDir,
	}

	raw, err := json.Marshal(SessionStartInput{
		HookInputBase: HookInputBase{
			SessionID:     "new-session",
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

	// Primary verification: snapshot file must be removed after successful sync.
	// File removal only happens inside syncPendingSnapshot after UpdateBeadMetadata succeeds,
	// proving the full sync path (QueryByLabel -> UpdateBeadMetadata -> os.Remove) ran.
	if _, err := os.Stat(snapshotPath); !os.IsNotExist(err) {
		t.Error("expected precompact-snapshot.json to be removed after sync, but it still exists")
	}

	// Output must still be valid JSON (session continued normally after sync).
	output := strings.TrimSpace(buf.String())
	var out HookOutput
	if err := json.Unmarshal([]byte(output), &out); err != nil {
		t.Fatalf("output is not valid JSON: %v, output was: %q", err, output)
	}
}

// TestSessionStartNoSnapshot verifies handleSessionStart works normally when
// no precompact-snapshot.json exists (regression guard).
func TestSessionStartNoSnapshot(t *testing.T) {
	fakeBd := buildFakeBd(t)
	tmpDir := t.TempDir()

	// Create .beads/ but no .gsdw/precompact-snapshot.json.
	if err := os.MkdirAll(filepath.Join(tmpDir, ".beads"), 0755); err != nil {
		t.Fatalf("failed to create .beads/: %v", err)
	}

	hs := &hookState{
		bdPath:   fakeBd,
		beadsDir: tmpDir,
	}

	raw, err := json.Marshal(SessionStartInput{
		HookInputBase: HookInputBase{
			SessionID:     "no-snap-session",
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
	// Must complete without error.
	if err := handleSessionStart(ctx, raw, hs, &buf); err != nil {
		t.Fatalf("handleSessionStart returned error: %v", err)
	}

	// Output must be valid JSON.
	output := strings.TrimSpace(buf.String())
	var out HookOutput
	if err := json.Unmarshal([]byte(output), &out); err != nil {
		t.Fatalf("output is not valid JSON: %v, output was: %q", err, output)
	}
	// AdditionalContext must be non-empty (normal behavior without snapshot).
	if out.AdditionalContext == "" {
		t.Error("expected non-empty additionalContext when no snapshot present")
	}
}

// TestSessionStartSnapshotSyncError verifies handleSessionStart completes in degraded
// mode when bead sync fails (broken bdPath). The snapshot file must remain on disk
// for retry on next session.
func TestSessionStartSnapshotSyncError(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .beads/ so handler bypasses the no-beads fast path.
	if err := os.MkdirAll(filepath.Join(tmpDir, ".beads"), 0755); err != nil {
		t.Fatalf("failed to create .beads/: %v", err)
	}

	// Create snapshot file.
	gsdwDir := filepath.Join(tmpDir, ".gsdw")
	if err := os.MkdirAll(gsdwDir, 0755); err != nil {
		t.Fatalf("failed to create .gsdw/: %v", err)
	}
	snapshotPath := filepath.Join(gsdwDir, "precompact-snapshot.json")
	snapshotData := `{"session_id":"err-session","transcript_path":"/tmp/t.jsonl","trigger":"auto","timestamp":"2026-03-21T19:30:00Z","cwd":"` + tmpDir + `"}`
	if err := os.WriteFile(snapshotPath, []byte(snapshotData), 0644); err != nil {
		t.Fatalf("failed to write snapshot: %v", err)
	}

	// Use a broken bdPath — graph operations will fail.
	hs := &hookState{
		bdPath:   "/nonexistent/bd",
		beadsDir: tmpDir,
	}

	raw, err := json.Marshal(SessionStartInput{
		HookInputBase: HookInputBase{
			SessionID:     "err-session",
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
	// Must NOT return error — SessionStart degrades gracefully.
	if err := handleSessionStart(ctx, raw, hs, &buf); err != nil {
		t.Fatalf("expected no error in degraded mode, got: %v", err)
	}

	// Output must be valid JSON.
	output := strings.TrimSpace(buf.String())
	if len(output) == 0 {
		t.Fatal("expected non-empty output even in degraded mode")
	}
	var out HookOutput
	if err := json.Unmarshal([]byte(output), &out); err != nil {
		t.Fatalf("output is not valid JSON in degraded mode: %v, output was: %q", err, output)
	}

	// Snapshot file must remain on disk (not removed) so it can be retried next session.
	if _, err := os.Stat(snapshotPath); os.IsNotExist(err) {
		t.Error("expected precompact-snapshot.json to remain on disk when sync fails, but it was removed")
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
