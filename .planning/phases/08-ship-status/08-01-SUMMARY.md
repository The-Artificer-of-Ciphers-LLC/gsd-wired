---
phase: 08-ship-status
plan: 01
subsystem: mcp
tags: [go, mcp, beads, ship, status, pr-summary, phase-advancement]

# Dependency graph
requires:
  - phase: 07-execution-verification
    provides: execute_wave, verify_phase MCP tools + get_status base implementation
  - phase: 05-project-init
    provides: get_status tool (to enrich), ClosePlan graph operation
provides:
  - create_pr_summary MCP tool (tool 16): bead-sourced PR body with requirements + phase checklist
  - advance_phase MCP tool (tool 17): close phase epic + surface next phase + unblocked beads
  - get_status enriched with CompletedPhases (ship-specific context)
affects: [09-token-routing, 10-cli-packaging, skills/ship]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "reqPattern regexp at package level in mcp package (mirrors reqLabelPattern in cli/ready.go)"
    - "phaseNumFromMeta reused from execute_wave.go for phase lookup across new tools"
    - "CompletedPhases initialized to [] not nil (same pattern as ReadyTasks)"

key-files:
  created:
    - internal/mcp/create_pr_summary.go
    - internal/mcp/create_pr_summary_test.go
    - internal/mcp/advance_phase.go
    - internal/mcp/advance_phase_test.go
    - internal/mcp/get_status_test.go
  modified:
    - internal/mcp/tools.go
    - internal/mcp/server.go
    - internal/mcp/tools_test.go
    - internal/mcp/server_test.go
    - internal/mcp/get_status.go

key-decisions:
  - "reqPattern defined locally in create_pr_summary.go — reqLabelPattern is in cli package, not mcp; local definition avoids cross-package coupling"
  - "advance_phase reuses phaseNumFromMeta (execute_wave.go private helper in same package) — no duplication"
  - "CompletedPhases appended inside existing phase bead loop — single query, no extra graph call"
  - "NextPhase in advancePhaseResult uses pre-queried phases list — avoids extra QueryByLabel after close"

patterns-established:
  - "Tool count atomically updated across 4 files: tools.go comment, server.go debug log, tools_test.go, server_test.go"
  - "TDD with FAKE_BD_QUERY_PHASE_RESPONSE for hermetic phase bead injection in mcp package tests"

requirements-completed: [SHIP-01, SHIP-02, CMD-02]

# Metrics
duration: 5min
completed: 2026-03-21
---

# Phase 8 Plan 01: Ship + Status Tools Summary

**create_pr_summary and advance_phase MCP tools added (15->17 tools), get_status enriched with completed_phases using bead graph data**

## Performance

- **Duration:** ~5 min
- **Started:** 2026-03-21T23:05:09Z
- **Completed:** 2026-03-21T23:09:43Z
- **Tasks:** 2 (both TDD)
- **Files modified:** 10

## Accomplishments
- create_pr_summary (tool 16): queries gsd:phase beads to build PR title, markdown body with requirements + phase checklist, and branch name
- advance_phase (tool 17): closes phase epic via ClosePlan, returns closed bead + unblocked + next phase info
- get_status enriched: CompletedPhases array added with phase_num, title, close_reason, closed_at for all non-open phases
- Tool count updated atomically from 15 to 17 across tools.go, server.go, tools_test.go, server_test.go
- All 161 tests pass with -race

## Task Commits

Each task was committed atomically:

1. **Task 1: RED (failing tests)** - `050029f` (test)
2. **Task 1: GREEN (implementation + tool registration)** - `f1cf392` (feat)
3. **Task 2: RED (get_status failing tests)** - (test, in get_status_test.go commit)
4. **Task 2: GREEN (get_status enrichment)** - `6b610a2` (feat)

_Note: TDD tasks have multiple commits (test RED → feat GREEN)_

## Files Created/Modified
- `internal/mcp/create_pr_summary.go` - handleCreatePrSummary, prSummaryResult, reqPattern
- `internal/mcp/create_pr_summary_test.go` - TestCreatePrSummary, TestCreatePrSummaryNoProject, TestCreatePrSummaryNotFound
- `internal/mcp/advance_phase.go` - handleAdvancePhase, advancePhaseArgs, advancePhaseResult
- `internal/mcp/advance_phase_test.go` - TestAdvancePhase, TestAdvancePhaseNotFound, TestAdvancePhaseNextPhase
- `internal/mcp/get_status.go` - completedPhaseInfo struct, CompletedPhases field, enrichment in loop
- `internal/mcp/get_status_test.go` - TestGetStatusEnriched, TestGetStatusEnrichedEmpty
- `internal/mcp/tools.go` - tool count 15->17, create_pr_summary + advance_phase registrations
- `internal/mcp/server.go` - debug log count 15->17
- `internal/mcp/tools_test.go` - TestToolsRegistered count 15->17, two new tool names
- `internal/mcp/server_test.go` - TestToolsListed count 15->17, two new tool names

## Decisions Made
- reqPattern defined locally in create_pr_summary.go — the existing reqLabelPattern is in the cli package, not mcp; local definition avoids cross-package coupling while achieving same goal
- advance_phase reuses phaseNumFromMeta from execute_wave.go (same package) — no duplication needed
- CompletedPhases populated in existing phase bead loop — single QueryByLabel query, zero extra graph I/O
- NextPhase uses pre-queried phases list after ClosePlan — avoids extra QueryByLabel call

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- 17 MCP tools registered and tested
- create_pr_summary ready for skills/ship SKILL.md integration (Phase 8 Plan 02)
- advance_phase ready for skills/ship flow
- get_status completed_phases ready for enriched status display

---
*Phase: 08-ship-status*
*Completed: 2026-03-21*
