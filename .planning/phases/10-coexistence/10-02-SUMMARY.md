---
phase: 10-coexistence
plan: 02
subsystem: hook+mcp
tags: [go, compat, tdd, session-start, mcp, fallback, planning]

requires:
  - phase: 10-coexistence
    plan: 01
    provides: internal/compat package with DetectPlanning and BuildFallbackStatus

provides:
  - .planning/ fallback path in handleSessionStart (formatFallbackContext)
  - .planning/ fallback path in handleGetStatus (fallbackStatusResult)
  - compatibility mode indicator in SessionStart additionalContext

affects:
  - hook/session_start.go: new fallback branch before "Run init" hint
  - mcp/get_status.go: new fallback branch after state.init() failure

tech-stack:
  added: []
  patterns:
    - ".beads/ checked first, .planning/ fallback only on absence (D-10 beads-first rule)"
    - "Read-only compat calls: DetectPlanning + BuildFallbackStatus, no writes (D-09, COMPAT-03)"
    - "Graceful degradation: fallback errors logged to slog.Warn, not surfaced to caller"
    - "TDD: RED test commit before GREEN implementation commit (4 tests per task)"

key-files:
  created: []
  modified:
    - internal/hook/session_start.go
    - internal/hook/session_start_test.go
    - internal/mcp/get_status.go
    - internal/mcp/get_status_test.go

key-decisions:
  - "formatFallbackContext in session_start.go: markdown string with compatibility mode prefix, project name, core value, phase/plan counters, and phase checkbox list — concise for context window"
  - "fallbackStatusResult in get_status.go: maps FallbackStatus fields to statusResult struct — OpenPhases counts !Complete phases, CurrentPhase found by matching State.CurrentPhase to Phases list, ReadyTasks stays empty (no graph)"
  - "state.beadsDir is set during sync.Once even on failure — DetectPlanning uses state.beadsDir as the dir to check for .planning/"
  - "compat_test.go WriteFile calls are fixture setup in t.TempDir() — not writes to real .planning/ dirs (D-09 not violated)"

requirements-completed: [COMPAT-01, COMPAT-03]

duration: 5min
completed: 2026-03-22
---

# Phase 10 Plan 02: Coexistence Summary

**Wire compat parsers into SessionStart hook and get_status MCP tool as .planning/ fallback paths — TDD, 8 new tests, -race clean, zero writes to .planning/**

## Performance

- **Duration:** 5 min
- **Started:** 2026-03-22T00:13:49Z
- **Completed:** 2026-03-22T00:18:27Z
- **Tasks:** 2 (TDD: RED commit + GREEN commit each)
- **Files modified:** 4

## Accomplishments

- Added `formatFallbackContext` helper to `session_start.go` that formats `FallbackStatus` into a markdown additionalContext string with compatibility mode indicator, project name, core value, current phase/plan counters, and phase checkbox list
- Modified `handleSessionStart` to check `compat.DetectPlanning` when `.beads/` is absent, then call `compat.BuildFallbackStatus` and return formatted context — keeping `.beads/` check first per D-10
- Added `fallbackStatusResult` helper to `get_status.go` that maps `FallbackStatus` to `statusResult` with `ProjectName`, `TotalPhases`, `OpenPhases`, `CompletedPhases`, `CurrentPhase`, and empty `ReadyTasks`
- Modified `handleGetStatus` to check `compat.DetectPlanning(state.beadsDir)` when `state.init()` fails, then call `compat.BuildFallbackStatus` and return structured result — or fall through to existing `toolError` if no `.planning/`
- Written 4 failing tests per task (TDD RED) then implemented (GREEN), all 220 project tests pass with -race
- Verified zero writes to `.planning/` in all modified files — D-09 and COMPAT-03 enforced

## Task Commits

Each task was committed atomically:

1. **Task 1 RED: Failing tests for SessionStart .planning/ fallback** — `c0ee314` (test)
2. **Task 1 GREEN: Add .planning/ fallback to SessionStart hook** — `caf02d1` (feat)
3. **Task 2 RED: Failing tests for get_status .planning/ fallback** — `2c1ac1a` (test)
4. **Task 2 GREEN: Add .planning/ fallback to get_status MCP tool** — `177bddf` (feat)

## Files Created/Modified

- `/Users/trekkie/projects/gsd-wired/internal/hook/session_start.go` — added `compat` import, `.planning/` fallback in `handleSessionStart`, new `formatFallbackContext` helper
- `/Users/trekkie/projects/gsd-wired/internal/hook/session_start_test.go` — 4 new tests: `TestSessionStartWithPlanningDir`, `TestSessionStartNoPlanningDir`, `TestSessionStartPlanningCompatibilityModeIndicator`, `TestSessionStartBeadsPriorityOverPlanning`
- `/Users/trekkie/projects/gsd-wired/internal/mcp/get_status.go` — added `compat` import, `.planning/` fallback after `state.init()` failure, new `fallbackStatusResult` helper
- `/Users/trekkie/projects/gsd-wired/internal/mcp/get_status_test.go` — 3 new tests: `TestGetStatusWithPlanningFallback`, `TestGetStatusFallbackNoPlanning`, `TestGetStatusFallbackPopulatesFields`

## Decisions Made

- `state.beadsDir` is populated during `sync.Once` initialization even when `state.init()` returns an error (because the dir is set before `runBdInit` is called). This means `compat.DetectPlanning(state.beadsDir)` correctly receives the project root directory even in the failure path.
- `ReadyTasks` is always an empty slice in fallback mode — there is no bead graph to surface unblocked tasks from. This is correct behavior per plan spec.
- `compat_test.go` writes are test fixture setup to `t.TempDir()` (not to real `.planning/` dirs) — D-09 compliance is not violated.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## Known Stubs

None — all fallback paths return real data from parsed .planning/ files.

## Verification Results

```
go test ./... -race -count=1
220 tests pass in 8 packages

compat.go is read-only: PASS
session_start.go and get_status.go are read-only: PASS
beadsPath check (line 104) before DetectPlanning (line 107): PASS (D-10 priority maintained)
```

## Self-Check: PASSED

- FOUND: internal/hook/session_start.go (modified with compat import + formatFallbackContext)
- FOUND: internal/hook/session_start_test.go (4 new tests added)
- FOUND: internal/mcp/get_status.go (modified with compat import + fallbackStatusResult)
- FOUND: internal/mcp/get_status_test.go (3 new tests added)
- FOUND commit: c0ee314 (test: failing tests for SessionStart)
- FOUND commit: caf02d1 (feat: SessionStart .planning/ fallback)
- FOUND commit: 2c1ac1a (test: failing tests for get_status)
- FOUND commit: 177bddf (feat: get_status .planning/ fallback)
- All 220 tests pass with -race -count=1
- Zero write operations in compat.go, session_start.go, get_status.go confirmed

---
*Phase: 10-coexistence*
*Completed: 2026-03-22*
