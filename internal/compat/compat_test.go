package compat_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/The-Artificer-of-Ciphers-LLC/gsd-wired/internal/compat"
)

// Sample STATE.md content matching the real .planning/STATE.md format.
const sampleState = `# Project State

## Current Position

Phase: 9 of 10 (Token-Aware Context)
Plan: 2 of 2 in current phase
Status: Executing
Last activity: 2026-03-21 -- Phase 9 Plan 02 complete

Progress: [█████████░] 87%
`

// Sample ROADMAP.md content matching the real .planning/ROADMAP.md format.
const sampleRoadmap = `# Roadmap: gsd-wired

## Phases

- [x] **Phase 1: Binary Scaffold** - Go binary with Cobra subcommands, plugin manifest, stdout discipline
- [x] **Phase 2: Graph Primitives** - bd CLI wrapper and GSD-to-beads domain mapping
- [x] **Phase 3: MCP Server** - MCP server with lazy Dolt init and tool registration
- [ ] **Phase 10: Coexistence** - .planning/ fallback reading and gradual adoption path

## Phase Details

### Phase 1: Binary Scaffold
**Goal**: A single Go binary that runs as MCP server, hook dispatcher, or CLI tool
**Plans**: 2 plans

### Phase 10: Coexistence
**Goal**: Existing GSD users can adopt gsd-wired gradually without abandoning their .planning/ workflow
**Plans**: 2 plans
`

// Sample PROJECT.md content matching the real .planning/PROJECT.md format.
const sampleProject = `# gsd-wired

## What This Is

A Claude Code plugin that fuses GSD workflow.

## Core Value

GSD's full development lifecycle (init → research → plan → execute → verify → ship) running on a beads graph engine so that orchestrator context stays lean and subagents pull only the context they need.

## Requirements
`

// ---- ParseState tests ----

func TestParseState_CurrentPhase(t *testing.T) {
	s := compat.ParseState(sampleState)
	if s.CurrentPhase != 9 {
		t.Errorf("CurrentPhase: got %d, want 9", s.CurrentPhase)
	}
}

func TestParseState_PlanProgress(t *testing.T) {
	s := compat.ParseState(sampleState)
	if s.CurrentPlan != 2 {
		t.Errorf("CurrentPlan: got %d, want 2", s.CurrentPlan)
	}
	if s.TotalPlans != 2 {
		t.Errorf("TotalPlans: got %d, want 2", s.TotalPlans)
	}
}

func TestParseState_Progress(t *testing.T) {
	s := compat.ParseState(sampleState)
	if s.Progress != "87%" {
		t.Errorf("Progress: got %q, want %q", s.Progress, "87%")
	}
}

func TestParseState_LastActivity(t *testing.T) {
	s := compat.ParseState(sampleState)
	if !strings.Contains(s.LastActivity, "2026-03-21") {
		t.Errorf("LastActivity: got %q, want it to contain '2026-03-21'", s.LastActivity)
	}
}

func TestParseState_Empty(t *testing.T) {
	s := compat.ParseState("")
	if s.CurrentPhase != 0 || s.CurrentPlan != 0 || s.TotalPlans != 0 || s.Progress != "" || s.LastActivity != "" {
		t.Errorf("Empty input: got non-zero ProjectState %+v", s)
	}
}

// ---- ParseRoadmap tests ----

func TestParseRoadmap_CompletedPhase(t *testing.T) {
	phases := compat.ParseRoadmap(sampleRoadmap)
	var phase1 *compat.PhaseEntry
	for i := range phases {
		if phases[i].Number == 1 {
			phase1 = &phases[i]
			break
		}
	}
	if phase1 == nil {
		t.Fatal("Phase 1 not found in ParseRoadmap result")
	}
	if !phase1.Complete {
		t.Errorf("Phase 1 Complete: got false, want true")
	}
	if phase1.Name != "Binary Scaffold" {
		t.Errorf("Phase 1 Name: got %q, want %q", phase1.Name, "Binary Scaffold")
	}
}

func TestParseRoadmap_IncompletePhase(t *testing.T) {
	phases := compat.ParseRoadmap(sampleRoadmap)
	var phase10 *compat.PhaseEntry
	for i := range phases {
		if phases[i].Number == 10 {
			phase10 = &phases[i]
			break
		}
	}
	if phase10 == nil {
		t.Fatal("Phase 10 not found in ParseRoadmap result")
	}
	if phase10.Complete {
		t.Errorf("Phase 10 Complete: got true, want false")
	}
	if phase10.Name != "Coexistence" {
		t.Errorf("Phase 10 Name: got %q, want %q", phase10.Name, "Coexistence")
	}
}

func TestParseRoadmap_GoalPopulated(t *testing.T) {
	phases := compat.ParseRoadmap(sampleRoadmap)
	var phase1 *compat.PhaseEntry
	for i := range phases {
		if phases[i].Number == 1 {
			phase1 = &phases[i]
			break
		}
	}
	if phase1 == nil {
		t.Fatal("Phase 1 not found")
	}
	if phase1.Goal == "" {
		t.Errorf("Phase 1 Goal: want non-empty, got empty")
	}
	if !strings.Contains(phase1.Goal, "Go binary") {
		t.Errorf("Phase 1 Goal: got %q, want it to contain 'Go binary'", phase1.Goal)
	}
}

func TestParseRoadmap_Empty(t *testing.T) {
	phases := compat.ParseRoadmap("")
	if len(phases) != 0 {
		t.Errorf("Empty input: got %d phases, want 0", len(phases))
	}
}

// ---- ParseProject tests ----

func TestParseProject_Name(t *testing.T) {
	name, _ := compat.ParseProject(sampleProject)
	if name != "gsd-wired" {
		t.Errorf("ParseProject name: got %q, want %q", name, "gsd-wired")
	}
}

func TestParseProject_CoreValue(t *testing.T) {
	_, coreValue := compat.ParseProject(sampleProject)
	if coreValue == "" {
		t.Error("ParseProject coreValue: got empty, want non-empty")
	}
	if !strings.Contains(coreValue, "GSD") {
		t.Errorf("ParseProject coreValue: got %q, want it to contain 'GSD'", coreValue)
	}
}

func TestParseProject_Empty(t *testing.T) {
	name, coreValue := compat.ParseProject("")
	if name != "" || coreValue != "" {
		t.Errorf("Empty input: got name=%q coreValue=%q, want both empty", name, coreValue)
	}
}

// ---- DetectPlanning tests ----

func TestDetectPlanning_ExistingDir(t *testing.T) {
	tmp := t.TempDir()
	if err := os.Mkdir(filepath.Join(tmp, ".planning"), 0o755); err != nil {
		t.Fatal(err)
	}
	if !compat.DetectPlanning(tmp) {
		t.Error("DetectPlanning: got false for existing .planning/, want true")
	}
}

func TestDetectPlanning_MissingDir(t *testing.T) {
	tmp := t.TempDir()
	if compat.DetectPlanning(tmp) {
		t.Error("DetectPlanning: got true for missing .planning/, want false")
	}
}

// ---- BuildFallbackStatus tests ----

func TestBuildFallbackStatus_ReadsFiles(t *testing.T) {
	tmp := t.TempDir()
	planningDir := filepath.Join(tmp, ".planning")
	if err := os.Mkdir(planningDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(planningDir, "STATE.md"), []byte(sampleState), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(planningDir, "ROADMAP.md"), []byte(sampleRoadmap), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(planningDir, "PROJECT.md"), []byte(sampleProject), 0o644); err != nil {
		t.Fatal(err)
	}

	status, err := compat.BuildFallbackStatus(tmp)
	if err != nil {
		t.Fatalf("BuildFallbackStatus: unexpected error: %v", err)
	}
	if status.ProjectName != "gsd-wired" {
		t.Errorf("ProjectName: got %q, want %q", status.ProjectName, "gsd-wired")
	}
	if status.State.CurrentPhase != 9 {
		t.Errorf("State.CurrentPhase: got %d, want 9", status.State.CurrentPhase)
	}
	if len(status.Phases) == 0 {
		t.Error("Phases: got empty slice, want non-empty")
	}
}

func TestBuildFallbackStatus_MissingFilesNonFatal(t *testing.T) {
	tmp := t.TempDir()
	planningDir := filepath.Join(tmp, ".planning")
	if err := os.Mkdir(planningDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Only write STATE.md; others are absent — should still succeed
	if err := os.WriteFile(filepath.Join(planningDir, "STATE.md"), []byte(sampleState), 0o644); err != nil {
		t.Fatal(err)
	}

	status, err := compat.BuildFallbackStatus(tmp)
	if err != nil {
		t.Fatalf("BuildFallbackStatus with missing files: unexpected error: %v", err)
	}
	if status.State.CurrentPhase != 9 {
		t.Errorf("State.CurrentPhase: got %d, want 9", status.State.CurrentPhase)
	}
	if status.ProjectName != "" {
		t.Errorf("ProjectName: got %q, want empty (file missing)", status.ProjectName)
	}
}

// ---- Integration test using real .planning/ files ----

func TestParseRoadmap_RealFile(t *testing.T) {
	// Use the actual ROADMAP.md from the project repo as a fixture.
	// Walk up from the test file's location to find .planning/ROADMAP.md.
	content, err := os.ReadFile("../../.planning/ROADMAP.md")
	if err != nil {
		t.Skip("Skipping real-file test: .planning/ROADMAP.md not found:", err)
	}
	phases := compat.ParseRoadmap(string(content))
	// ROADMAP.md evolves as milestones are added. Phases 1-10 may be in a
	// <details> block (collapsed after milestone completion) and new phases
	// (11+) appear in the active section. Accept any non-zero count.
	if len(phases) == 0 {
		t.Errorf("ParseRoadmap real file: got 0 phases, want at least 1")
	}
}
