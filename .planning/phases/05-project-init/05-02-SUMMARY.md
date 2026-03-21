---
phase: 05-project-init
plan: "02"
subsystem: cli
tags: [cobra, skills, slash-commands, claude-code-plugin, gsdw-init, gsdw-status]

requires:
  - phase: 04-hook-integration
    provides: hookState, buildSessionContext, graph.Client patterns
  - phase: 02-graph-primitives
    provides: graph.NewClient, QueryByLabel, ListReady, findBeadsDir, phaseNumFromBead

provides:
  - "skills/init/SKILL.md: /gsd-wired:init slash command with 12-question full, 3-question quick, and PR/issue modes"
  - "skills/status/SKILL.md: /gsd-wired:status slash command calling get_status MCP tool"
  - "internal/cli/init.go: gsdw init CLI subcommand (bd init + PROJECT.md + .gsdw/config.json)"
  - "internal/cli/status.go: gsdw status CLI subcommand with renderStatus pure function"
  - "internal/cli/root.go: NewInitCmd and NewStatusCmd registered"

affects: [06-planning-workflow, 07-research-agents, 09-token-routing]

tech-stack:
  added: []
  patterns:
    - "renderStatus as pure function: testable with constructed Bead slices, no graph dependency in tests"
    - "SKILL.md at plugin root skills/ dir: auto-discovered as /gsd-wired:name slash commands"
    - "disable-model-invocation: true in SKILL.md: user must explicitly invoke, prevents auto-triggering"

key-files:
  created:
    - skills/init/SKILL.md
    - skills/status/SKILL.md
    - internal/cli/init.go
    - internal/cli/init_test.go
    - internal/cli/status.go
    - internal/cli/status_test.go
  modified:
    - internal/cli/root.go

key-decisions:
  - "SKILL.md files placed at plugin root skills/ (not inside .claude-plugin/) for auto-discovery per Pitfall 1"
  - "disable-model-invocation: true on init SKILL.md — user controls when to run /gsd-wired:init"
  - "renderStatus extracted as pure function for testability (same pattern as renderReadyTree in ready.go)"
  - "gsdw init skips bd init if bd not on PATH in test environments (graceful failure path)"
  - "TestInitCmdWritesFiles uses real temp dir via t.TempDir() — verifies actual file creation behavior"

patterns-established:
  - "renderStatus pattern: pure function (io.Writer, phases, ready) — testable without graph client"
  - "NewInitCmd/NewStatusCmd follow NewReadyCmd cobra pattern exactly"

requirements-completed: [INIT-01, INIT-02, CMD-01]

duration: 2min
completed: 2026-03-21
---

# Phase 5 Plan 02: Project Initialization User Interface Summary

**SKILL.md slash commands (/gsd-wired:init with 12-question flow and /gsd-wired:status) plus gsdw init/status CLI subcommands with cobra, all registered in root**

## Performance

- **Duration:** 2 min
- **Started:** 2026-03-21T00:25:02Z
- **Completed:** 2026-03-21T00:27:02Z
- **Tasks:** 2 (Task 1: SKILL.md files, Task 2: CLI subcommands with TDD)
- **Files modified:** 7

## Accomplishments

- Two Claude Code slash commands created via skills/init/SKILL.md and skills/status/SKILL.md at plugin root — auto-discovered as /gsd-wired:init and /gsd-wired:status
- skills/init/SKILL.md implements full 12-question flow, 3-question quick mode, and PR/issue mode with one-question-at-a-time discipline per D-03/D-04/D-05
- gsdw init CLI subcommand (internal/cli/init.go) runs bd init, writes PROJECT.md template, and creates .gsdw/config.json
- gsdw status CLI subcommand (internal/cli/status.go) queries beads graph and renders GSD-familiar dashboard (phases/plans/waves — never bead IDs)
- All 7 packages pass go test ./... -race

## Task Commits

Each task committed atomically:

1. **Task 1: SKILL.md files for /gsd-wired:init and /gsd-wired:status** - `2aba3ea` (feat)
2. **Task 2 RED: Failing tests for gsdw init and gsdw status** - `5de4178` (test)
3. **Task 2 GREEN: gsdw init and gsdw status implementation** - `a9205e9` (feat)

**Plan metadata:** (docs commit follows)

## Files Created/Modified

- `skills/init/SKILL.md` - /gsd-wired:init slash command: 12-question full, 3-question quick, PR/issue modes; calls init_project MCP tool
- `skills/status/SKILL.md` - /gsd-wired:status slash command: calls get_status MCP tool, renders GSD-familiar dashboard
- `internal/cli/init.go` - NewInitCmd: bd init + PROJECT.md template + .gsdw/config.json creation
- `internal/cli/init_test.go` - TestRootCmdHasInit, TestInitCmdWritesFiles
- `internal/cli/status.go` - NewStatusCmd + renderStatus pure function for testability
- `internal/cli/status_test.go` - TestRootCmdHasStatus, TestStatusCmdOutput, TestStatusCmdNoProject
- `internal/cli/root.go` - Added NewInitCmd() and NewStatusCmd() to AddCommand chain

## Decisions Made

- SKILL.md files placed at plugin root `skills/` directory (not inside `.claude-plugin/`) — auto-discovery per Pitfall 1 from research
- `disable-model-invocation: true` in init SKILL.md — user must explicitly invoke `/gsd-wired:init`, prevents accidental trigger
- `renderStatus` extracted as pure function (io.Writer, phases, ready beads) — same testability pattern as `renderReadyTree`
- `gsdw init` returns error if bd not on PATH (clean failure) but still writes PROJECT.md and config.json as long as it progresses past bd step
- `TestInitCmdWritesFiles` uses real temp dir via t.TempDir() for authentic file creation verification

## Deviations from Plan

None - plan executed exactly as written. All acceptance criteria met, all tests pass with -race.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- /gsd-wired:init and /gsd-wired:status slash commands ready for use after plugin installation
- init_project and get_status MCP tools referenced by SKILL.md files need to be implemented in Phase 5 Plan 01 (parallel wave — may already be complete)
- gsdw init and gsdw status CLI subcommands available immediately via gsdw binary
- Phase 6 (Planning Workflow) can build on the init foundation — project beads created by init_project become the parent for planning phase/plan beads

---
*Phase: 05-project-init*
*Completed: 2026-03-21*
