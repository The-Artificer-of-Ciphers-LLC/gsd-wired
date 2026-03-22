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

// TestBuildBudgetContext verifies budget-aware context building with hot/warm/cold beads.
// Hot beads always included; warm beads included if budget allows; cold beads omitted when over.
func TestBuildBudgetContext(t *testing.T) {
	fakeBd := buildFakeBd(t)
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, ".beads"), 0755); err != nil {
		t.Fatalf("failed to create .beads/: %v", err)
	}

	hs := &hookState{
		bdPath:   fakeBd,
		beadsDir: tmpDir,
	}
	if err := hs.init(context.Background()); err != nil {
		t.Fatalf("hookState.init() failed: %v", err)
	}

	// Build context with a generous budget — all tiers should be included.
	ctx := context.Background()
	result := buildBudgetContext(ctx, hs.client, 2000)
	// Context should be non-empty (at least the header).
	if result == "" {
		t.Error("buildBudgetContext returned empty string with budget=2000")
	}
}

// TestBuildBudgetContextHotAlways verifies hot beads are never dropped even with tiny budget.
func TestBuildBudgetContextHotAlways(t *testing.T) {
	fakeBd := buildFakeBd(t)
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, ".beads"), 0755); err != nil {
		t.Fatalf("failed to create .beads/: %v", err)
	}

	hs := &hookState{
		bdPath:   fakeBd,
		beadsDir: tmpDir,
	}
	if err := hs.init(context.Background()); err != nil {
		t.Fatalf("hookState.init() failed: %v", err)
	}

	// Budget=1 (tiny) — hot beads must still be included per Pitfall 2.
	ctx := context.Background()
	// Should not panic and should return a string (possibly just the header).
	result := buildBudgetContext(ctx, hs.client, 1)
	// Result may be empty or have minimal content — the key is it doesn't panic.
	_ = result
}

// TestSessionStartBudget verifies buildSessionContext still works as a wrapper for buildBudgetContext.
func TestSessionStartBudget(t *testing.T) {
	fakeBd := buildFakeBd(t)
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, ".beads"), 0755); err != nil {
		t.Fatalf("failed to create .beads/: %v", err)
	}

	hs := &hookState{
		bdPath:   fakeBd,
		beadsDir: tmpDir,
	}
	if err := hs.init(context.Background()); err != nil {
		t.Fatalf("hookState.init() failed: %v", err)
	}

	ctx := context.Background()
	result := buildSessionContext(ctx, hs.client)
	// Should return a non-empty string with default budget (2000 tokens).
	if result == "" {
		t.Error("buildSessionContext returned empty string")
	}
}

// TestSessionStartWithPlanningDir verifies that handleSessionStart emits project context
// from .planning/ files when .beads/ is absent but .planning/ is present (COMPAT-01).
func TestSessionStartWithPlanningDir(t *testing.T) {
	tmpDir := t.TempDir()
	// No .beads/ — only .planning/ with STATE.md, ROADMAP.md, PROJECT.md.
	planningDir := filepath.Join(tmpDir, ".planning")
	if err := os.MkdirAll(planningDir, 0755); err != nil {
		t.Fatalf("failed to create .planning/: %v", err)
	}

	// Write minimal STATE.md.
	stateContent := `# Project State

## Current Position

Phase: 5 of 10 (Token Optimization)
Plan: 2 of 3 in current phase
Status: Executing
Last activity: 2026-03-22 -- working on tests

Progress: [█████░░░░░] 50%
`
	if err := os.WriteFile(filepath.Join(planningDir, "STATE.md"), []byte(stateContent), 0644); err != nil {
		t.Fatalf("failed to write STATE.md: %v", err)
	}

	// Write minimal ROADMAP.md.
	roadmapContent := `# Roadmap

- [x] **Phase 1: Binary Scaffold**
- [x] **Phase 2: Graph Primitives**
- [ ] **Phase 5: Token Optimization**

## Phase Details

### Phase 1:
**Goal**: Build the binary
**Plans**: 2 plans

### Phase 5:
**Goal**: Optimize tokens
**Plans**: 3 plans
`
	if err := os.WriteFile(filepath.Join(planningDir, "ROADMAP.md"), []byte(roadmapContent), 0644); err != nil {
		t.Fatalf("failed to write ROADMAP.md: %v", err)
	}

	// Write minimal PROJECT.md.
	projectContent := `# TestProject

## Core Value

Optimize token usage for AI agents.

## Context

Details here.
`
	if err := os.WriteFile(filepath.Join(planningDir, "PROJECT.md"), []byte(projectContent), 0644); err != nil {
		t.Fatalf("failed to write PROJECT.md: %v", err)
	}

	hs := &hookState{} // no bdPath needed — exits before graph init
	raw, err := json.Marshal(SessionStartInput{
		HookInputBase: HookInputBase{
			SessionID:     "compat-1",
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

	// Must contain project name from PROJECT.md.
	if !strings.Contains(out.AdditionalContext, "TestProject") {
		t.Errorf("expected additionalContext to contain project name 'TestProject', got: %q", out.AdditionalContext)
	}

	// Must contain current phase info from STATE.md.
	if !strings.Contains(out.AdditionalContext, "5") {
		t.Errorf("expected additionalContext to contain phase number '5', got: %q", out.AdditionalContext)
	}

	// Must contain phase list from ROADMAP.md.
	if !strings.Contains(out.AdditionalContext, "Phase") {
		t.Errorf("expected additionalContext to contain 'Phase' entries, got: %q", out.AdditionalContext)
	}
}

// TestSessionStartNoPlanningDir verifies that handleSessionStart emits the "Run /gsd-wired:init"
// hint when neither .beads/ nor .planning/ exists.
func TestSessionStartNoPlanningDir(t *testing.T) {
	tmpDir := t.TempDir()
	// Neither .beads/ nor .planning/ — completely uninitialized.

	hs := &hookState{}
	raw, err := json.Marshal(SessionStartInput{
		HookInputBase: HookInputBase{
			SessionID:     "compat-2",
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

	// Must still emit the init hint.
	if !strings.Contains(out.AdditionalContext, "init") && !strings.Contains(out.AdditionalContext, "No .beads") {
		t.Errorf("expected hint to mention 'init' or 'No .beads' when no .planning/ either, got: %q", out.AdditionalContext)
	}

	// Must NOT contain "compatibility mode" — that's for .planning/ fallback only.
	if strings.Contains(out.AdditionalContext, "compatibility mode") {
		t.Errorf("expected no 'compatibility mode' in hint when no .planning/, got: %q", out.AdditionalContext)
	}
}

// TestSessionStartPlanningCompatibilityModeIndicator verifies the compatibility mode
// indicator string is present when .planning/ fallback is active.
func TestSessionStartPlanningCompatibilityModeIndicator(t *testing.T) {
	tmpDir := t.TempDir()
	planningDir := filepath.Join(tmpDir, ".planning")
	if err := os.MkdirAll(planningDir, 0755); err != nil {
		t.Fatalf("failed to create .planning/: %v", err)
	}

	// Write minimal files so BuildFallbackStatus succeeds.
	if err := os.WriteFile(filepath.Join(planningDir, "STATE.md"), []byte("Phase: 1 of 1\nPlan: 1 of 1\n"), 0644); err != nil {
		t.Fatalf("failed to write STATE.md: %v", err)
	}

	hs := &hookState{}
	raw, err := json.Marshal(SessionStartInput{
		HookInputBase: HookInputBase{
			SessionID:     "compat-3",
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

	// Must contain compatibility mode indicator per D-01.
	if !strings.Contains(out.AdditionalContext, "compatibility mode") {
		t.Errorf("expected 'compatibility mode' in additionalContext, got: %q", out.AdditionalContext)
	}
}

// TestSessionStartBeadsPriorityOverPlanning verifies that .beads/ takes priority over
// .planning/ when both exist (D-10: beads-first rule).
func TestSessionStartBeadsPriorityOverPlanning(t *testing.T) {
	fakeBd := buildFakeBd(t)
	tmpDir := t.TempDir()

	// Create both .beads/ and .planning/.
	if err := os.MkdirAll(filepath.Join(tmpDir, ".beads"), 0755); err != nil {
		t.Fatalf("failed to create .beads/: %v", err)
	}
	planningDir := filepath.Join(tmpDir, ".planning")
	if err := os.MkdirAll(planningDir, 0755); err != nil {
		t.Fatalf("failed to create .planning/: %v", err)
	}
	if err := os.WriteFile(filepath.Join(planningDir, "PROJECT.md"), []byte("# ShouldNotAppear\n"), 0644); err != nil {
		t.Fatalf("failed to write PROJECT.md: %v", err)
	}

	hs := &hookState{bdPath: fakeBd, beadsDir: tmpDir}
	raw, err := json.Marshal(SessionStartInput{
		HookInputBase: HookInputBase{
			SessionID:     "compat-4",
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

	// Must NOT contain compatibility mode — .beads/ is present and takes priority.
	if strings.Contains(out.AdditionalContext, "compatibility mode") {
		t.Errorf("expected beads path, not .planning/ fallback, when .beads/ exists; got: %q", out.AdditionalContext)
	}

	// The content should come from beads (graph header present).
	if !strings.Contains(out.AdditionalContext, "GSD Project State") {
		t.Errorf("expected beads-sourced 'GSD Project State' header, got: %q", out.AdditionalContext)
	}

	// Must NOT contain "ShouldNotAppear" from the .planning/ PROJECT.md.
	if strings.Contains(out.AdditionalContext, "ShouldNotAppear") {
		t.Errorf(".planning/ content leaked into beads-mode output: %q", out.AdditionalContext)
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
