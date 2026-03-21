---
phase: 05-project-init
plan: 01
subsystem: mcp
tags: [mcp, go-sdk, beads, bead-creation, project-init]

# Dependency graph
requires:
  - phase: 03-mcp-server
    provides: "serverState, registerTools pattern, toolError/toolResult helpers, connectInProcess test helper"
  - phase: 02-graph-primitives
    provides: "graph.Client.CreatePhase, graph.Client.CreatePlan, graph.Client.QueryByLabel, graph.Client.ListReady"

provides:
  - "init_project MCP tool: creates project epic bead (phaseNum=0) + up to 4 context child beads, writes PROJECT.md and .gsdw/config.json"
  - "get_status MCP tool: queries gsd:project and gsd:phase labels plus ListReady, returns structured statusResult JSON"
  - "10 total MCP tools registered (was 8)"

affects: [06-planning-phase, skills-init, skills-status]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "phaseNum=0 convention for project-level epic beads (not a real phase)"
    - "handleInitProject and handleGetStatus as standalone functions called from anonymous tool handlers"
    - "graceful degradation in get_status: slog.Warn on query failure, never IsError=true"
    - "buildProjectMD helper with conditional section rendering (skip empty fields)"

key-files:
  created:
    - internal/mcp/init_project.go
    - internal/mcp/get_status.go
  modified:
    - internal/mcp/tools.go
    - internal/mcp/server.go
    - internal/mcp/tools_test.go
    - internal/mcp/server_test.go

key-decisions:
  - "phaseNum=0 used as project-level epic convention — CreatePhase accepts any int so 0 works without schema changes"
  - "Single init_project tool handles both bead creation and file writing — simpler SKILL.md, eventual consistency acceptable if file writing fails after bead creation"
  - "Context child bead failures are non-fatal in handleInitProject — partial context better than init failure"
  - "get_status replicates buildSessionContext query pattern directly rather than calling the hook function — avoids hook package coupling"
  - "contains() helper implemented inline in test file instead of importing strings package to keep test file self-contained"

patterns-established:
  - "handleXxx(ctx, state, args) pattern for complex tool handlers — separable from anonymous closure for testability"
  - "ReadyTasks initialized as []taskInfo{} (not nil) for clean JSON serialization ([] vs null)"

requirements-completed: [INIT-03, INIT-04, INIT-05]

# Metrics
duration: 8min
completed: 2026-03-21
---

# Phase 5 Plan 01: init_project and get_status MCP Tools Summary

**init_project MCP tool creates project epic bead (phaseNum=0) + context child beads, writes PROJECT.md and .gsdw/config.json; get_status returns structured JSON dashboard from gsd:project/gsd:phase/ListReady queries**

## Performance

- **Duration:** ~8 min
- **Started:** 2026-03-21T21:00:00Z
- **Completed:** 2026-03-21T21:07:53Z
- **Tasks:** 2 (TDD: test → feat for each)
- **Files modified:** 6

## Accomplishments

- init_project MCP tool creates project epic bead with phaseNum=0 convention, plus context child beads for done-criteria, constraints, tech-stack, and risks (non-empty only)
- init_project writes PROJECT.md with conditional sections (quick mode skips empty fields) and .gsdw/config.json with project_name, initialized timestamp, and mode
- get_status MCP tool queries gsd:project, gsd:phase, and ListReady to return structured statusResult JSON with graceful degradation (never IsError=true on query failures)
- Tool count increased from 8 to 10; server.go tools debug log and server_test.go TestToolsListed both updated

## Task Commits

Each task was committed atomically using TDD:

1. **Task 1: RED — failing tests for init_project and get_status** - `78808fa` (test)
2. **Task 1+2: GREEN — init_project, get_status, tools.go, server.go** - `1e6f92f` (feat)

_Note: Both tasks implemented in a single GREEN commit since get_status tests were written in Task 1's RED phase and both implementations required tools.go registration._

## Files Created/Modified

- `/Users/trekkie/projects/gsd-wired/internal/mcp/init_project.go` - handleInitProject, initProjectArgs, initProjectResult, buildProjectMD
- `/Users/trekkie/projects/gsd-wired/internal/mcp/get_status.go` - handleGetStatus, statusResult, phaseInfo, taskInfo, phaseNumFromMetadata
- `/Users/trekkie/projects/gsd-wired/internal/mcp/tools.go` - registerTools updated with init_project and get_status tool registrations (8 -> 10)
- `/Users/trekkie/projects/gsd-wired/internal/mcp/server.go` - slog.Debug tools count 8 -> 10
- `/Users/trekkie/projects/gsd-wired/internal/mcp/tools_test.go` - 6 new tests added
- `/Users/trekkie/projects/gsd-wired/internal/mcp/server_test.go` - TestToolsListed updated to expect 10 tools

## Decisions Made

- phaseNum=0 for project epic: CreatePhase accepts any int, so 0 serves as a project-level convention without schema changes
- Single init_project tool for bead creation + file writing: simpler SKILL.md interface, eventual consistency acceptable
- Context child bead failures are non-fatal: partial context is better than init failure
- get_status replicates buildSessionContext query pattern directly rather than importing from hook package — avoids coupling

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Updated TestToolsListed in server_test.go (not mentioned in plan)**
- **Found during:** GREEN phase (full mcp package test run)
- **Issue:** server_test.go TestToolsListed had its own hardcoded `len == 8` check and `wantNames` list — not the same test as tools_test.go TestToolsRegistered
- **Fix:** Updated len check to 10, added "init_project" and "get_status" to wantNames, updated comment
- **Files modified:** internal/mcp/server_test.go
- **Verification:** `go test ./internal/mcp/... -v -race` all 20 tests pass
- **Committed in:** 1e6f92f (part of feat commit)

---

**Total deviations:** 1 auto-fixed (Rule 1 - existing test in wrong file)
**Impact on plan:** Required to make full package green. No scope creep.

## Issues Encountered

None beyond the server_test.go deviation above.

## Next Phase Readiness

- 10 MCP tools registered and tested; SKILL.md files can call init_project and get_status
- Full project regression: 7 packages, all green with -race
- Phase 05 Plan 02 (skills + CLI) ran in parallel — both plans now complete

## Self-Check: PASSED

All files created, commits verified, acceptance criteria met:
- FOUND: internal/mcp/init_project.go (func handleInitProject)
- FOUND: internal/mcp/get_status.go (func handleGetStatus, statusResult)
- FOUND: tools.go registrations for init_project and get_status
- FOUND: tools_test.go with "init_project" test entries
- FOUND: commit 78808fa (RED tests) and 1e6f92f (GREEN implementation)
- FOUND: .planning/phases/05-project-init/05-01-SUMMARY.md

---
*Phase: 05-project-init*
*Completed: 2026-03-21*
