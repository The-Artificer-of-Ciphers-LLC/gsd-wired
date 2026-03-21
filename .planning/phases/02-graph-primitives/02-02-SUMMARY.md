---
phase: 02-graph-primitives
plan: "02"
subsystem: internal/cli
tags: [cli, cobra, ready-command, tree-output, json, phase-filter, tdd]
dependency_graph:
  requires: [02-01-SUMMARY]
  provides: [gsdw ready subcommand — tree/JSON output of unblocked tasks]
  affects: [internal/cli (root command), cmd/gsdw binary]
tech_stack:
  added: []
  patterns:
    - Pure function rendering (renderReadyTree/renderReadyJSON) for testability without live bd
    - regexp.MustCompile for label filtering at package init time
    - ASCII tree chars (|-- / +--) for universal terminal compatibility
    - phaseNumFromBead type-switch handles float64/int/int64 from JSON unmarshaling
key_files:
  created:
    - internal/cli/ready.go
    - internal/cli/ready_test.go
  modified:
    - internal/cli/root.go
decisions:
  - "renderReadyTree/renderReadyJSON extracted as pure functions — testable with constructed Bead slices, no fake bd needed"
  - "reqLabelPattern compiled once at package level — avoids per-call regexp compilation cost"
  - "phaseNumFromBead uses type switch — JSON unmarshal produces float64 for numbers; int variants handle direct construction in tests"
  - "ASCII tree chars (|-- / +--) per plan spec — safer than unicode box-drawing for all terminals"
metrics:
  duration: "2 min"
  completed: "2026-03-21"
  tasks_completed: 1
  files_created: 2
requirements_satisfied: [MAP-03, MAP-06]
---

# Phase 2 Plan 02: gsdw ready Subcommand Summary

**One-liner:** Cobra subcommand rendering unblocked beads as ASCII-tree grouped by GSD phase with req-label brackets, plus --json for machine consumption and --phase N for single-phase filtering.

## What Was Built

The `gsdw ready` subcommand is the primary user-facing output of Phase 2. It surfaces the wave computation from Plan 01's graph primitives in a human-friendly format.

### Files Created

**internal/cli/ready.go** — `NewReadyCmd()` cobra command with `--json` and `--phase` flags. `findBeadsDir()` walks up the directory tree for `.beads/` or falls back to `BEADS_DIR` env. `renderReadyTree()` groups ready beads by `gsd_phase` metadata, sorts phases and plans, applies `--phase` filter, renders ASCII tree with `|--`/`+--` connectors, GSD names (`Phase N:`, `Plan XX-YY:`), and requirement labels in brackets. Footer shows `N ready | M queued | P remaining` (per D-16). `renderReadyJSON()` marshals the bead array to indented JSON. `reqLabelPattern` compiled from `^[A-Z]+-[0-9]+$` filters internal `gsd:` labels from user output (per D-19).

**internal/cli/ready_test.go** — 6 tests covering all acceptance criteria. Tests use `testBead()` helper constructing realistic Bead slices with `Metadata: map[string]any{"gsd_phase": float64(2), "gsd_plan": "02-01"}` and label arrays. All tests call the pure rendering functions directly with `bytes.Buffer` — no fake bd needed for rendering tests.

### Files Modified

**internal/cli/root.go** — `NewReadyCmd()` added to `AddCommand(...)` call on line 31.

## Test Results

```
go test ./internal/cli/... -run TestReady -race -count=1 -v
6/6 tests PASS
```

```
go test ./... -race -count=1
47/47 tests PASS (Phase 1 + Phase 2 regression: PASS)
```

```
go vet ./...
No warnings
```

```
go build ./cmd/gsdw
Binary compiles successfully
```

```
./gsdw ready --help
Shows usage with --json and --phase flags
```

## Commits

1. `test(02-02)` — TDD RED: failing tests for all 6 ready behaviors (RED commit)
2. `feat(02-02)` — GREEN: implement ready.go + update root.go (f36dc6d)

## Deviations from Plan

None — plan executed exactly as written.

## Known Stubs

None — all rendering logic is fully implemented and tested.

## Self-Check: PASSED

Files exist:
- internal/cli/ready.go: FOUND
- internal/cli/ready_test.go: FOUND
- internal/cli/root.go: FOUND (modified, contains NewReadyCmd())

Commits exist:
- RED test commit: FOUND
- f36dc6d (GREEN implementation): FOUND
