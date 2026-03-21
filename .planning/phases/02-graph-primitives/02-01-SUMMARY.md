---
phase: 02-graph-primitives
plan: "01"
subsystem: internal/graph
tags: [graph, bd-wrapper, exec-command, json-parsing, index, tdd]
dependency_graph:
  requires: [01-01-SUMMARY]
  provides: [internal/graph package — Client/Bead/Index for all bd operations]
  affects: [internal/cli (ready subcommand — Phase 2 Plan 02), internal/mcp (Phase 3), internal/hook (Phase 4)]
tech_stack:
  added: []
  patterns:
    - Client struct with injected bdPath/beadsDir for testability
    - exec.CommandContext with BEADS_DIR env injection
    - Two-tier bd error handling (stdout JSON error vs stderr text)
    - Fake bd binary in testdata/ for unit tests without live dolt
    - Atomic index write via temp+rename
key_files:
  created:
    - internal/graph/bead.go
    - internal/graph/client.go
    - internal/graph/create.go
    - internal/graph/query.go
    - internal/graph/update.go
    - internal/graph/index.go
    - internal/graph/testdata/fake_bd/main.go
    - internal/graph/graph_test.go
  modified:
    - .gitignore
decisions:
  - "NewClientWithPath() added as test injection point — bypasses LookPath for unit tests"
  - "fake_bd uses FAKE_BD_CAPTURE_FILE env to write received args — allows tests to verify exact args passed without parsing output"
  - "fake_bd uses FAKE_BD_READY_RESPONSE env for ClosePlan before/after diff testing"
  - "ListBlocked passes --limit 0 for consistency even though blocked default may differ from ready"
  - "QueryByLabel passes --limit 0 per research recommendation to prevent silent truncation"
metrics:
  duration: "4 min"
  completed: "2026-03-21"
  tasks_completed: 2
  files_created: 9
requirements_satisfied: [INFRA-03, MAP-01, MAP-02, MAP-04, MAP-05, MAP-06]
---

# Phase 2 Plan 01: Graph Primitives — bd CLI Wrapper Summary

**One-liner:** Go Client struct wrapping bd CLI via exec.Command with typed Bead structs, GSD domain mapping (phases=epics, plans=tasks), atomic local index, and full test suite using a fake bd binary.

## What Was Built

The `internal/graph/` package is the foundation layer for all beads graph operations. No other code shells out to bd directly — everything goes through this package.

### Files Created

**internal/graph/bead.go** — Typed structs matching bd v0.61.0 JSON schema. `Bead` contains all fields from live bd output including `AcceptanceCriteria`, `Metadata map[string]any`, `Labels []string`, `Parent string`. `Dependency` has `Metadata string` (not a map — bd stores it as JSON string). `BeadSummary` for the dependents array.

**internal/graph/client.go** — `Client` struct with `bdPath` and `beadsDir`. `NewClient()` uses `exec.LookPath("bd")`. `NewClientWithPath()` injects path directly for testing. `run()` appends `--json` to all calls, sets `BEADS_DIR` env, captures stdout/stderr, logs via `slog.Debug`, and handles two-tier errors: JSON `{"error":"..."}` on stdout takes precedence over stderr text.

**internal/graph/create.go** — `CreatePhase()` creates epic beads with `--type epic`, `--labels gsd:phase,REQ-IDs`, `--acceptance`, `--context`, `--metadata {"gsd_phase":N}`. `CreatePlan()` creates task beads with `--type task`, `--parent`, `--no-inherit-labels`, `--labels gsd:plan,REQ-IDs`, `--metadata {"gsd_phase":N,"gsd_plan":"XX-YY"}`, conditional `--deps` when depBeadIDs non-empty.

**internal/graph/query.go** — Five query methods: `ListReady` (`bd ready --limit 0`), `ReadyForPhase` (`bd ready --parent <id> --limit 0`), `ListBlocked` (`bd blocked --limit 0`), `GetBead` (`bd show <id>`), `QueryByLabel` (`bd query label=<label> --limit 0`). All use `--limit 0` to prevent silent truncation.

**internal/graph/update.go** — `ClaimBead` (`bd update --claim`), `ClosePlan` (pre/post ready snapshot diff for D-13 unblocked notifications, post-close failure is best-effort), `AddLabel` (`bd update --add-label`).

**internal/graph/index.go** — `Index` struct with `PhaseToID`/`PlanToID` maps. `Save()` uses atomic temp+rename pattern. `LoadIndex()` reads and unmarshals. `RebuildIndex()` queries `bd list --all --label gsd:phase` and `bd list --all --label gsd:plan` to reconstruct from live graph.

**internal/graph/testdata/fake_bd/main.go** — Standalone test binary dispatching on first arg. Supports: create/list/ready/show/close/blocked/update/query (canned responses), error-json (stdout JSON error, exit 1), error-stderr (stderr text, exit 1), echo-args (full args as JSON), echo-env (BEADS_DIR value). Supports `FAKE_BD_CAPTURE_FILE` to write received args for test verification, and `FAKE_BD_READY_RESPONSE` to inject custom ready responses.

**internal/graph/graph_test.go** — 21 tests covering all acceptance criteria. `TestMain` builds fake_bd binary at test startup. Tests use `NewClientWithPath(fakeBdPath, t.TempDir())`.

## Test Results

```
go test ./internal/graph/... -race -count=1 -v
21/21 tests PASS
```

```
go test ./... -race -count=1
All packages PASS (including Phase 1 tests)
```

```
go vet ./internal/graph/...
No warnings
```

## Commits

1. `test(02-01)` — TDD RED: failing tests for all graph operations (1f638ae)
2. `feat(02-01)` — GREEN: implement all 6 source files + .gitignore update (c52e526)

## Deviations from Plan

None — plan executed exactly as written.

## Known Stubs

None — all operations have real implementations calling fake_bd in tests.

## Self-Check: PASSED

Files exist:
- internal/graph/bead.go: FOUND
- internal/graph/client.go: FOUND
- internal/graph/create.go: FOUND
- internal/graph/query.go: FOUND
- internal/graph/update.go: FOUND
- internal/graph/index.go: FOUND
- internal/graph/testdata/fake_bd/main.go: FOUND
- internal/graph/graph_test.go: FOUND

Commits exist:
- 1f638ae: FOUND (TDD RED)
- c52e526: FOUND (GREEN implementation)
