---
phase: 06-research-planning
plan: 02
subsystem: mcp
tags: [mcp, plan, cli, skills, go, tdd]

# Dependency graph
requires:
  - phase: 06-01
    provides: run_research + synthesize_research MCP tools, /gsd-wired:research SKILL.md pattern

provides:
  - create_plan_beads MCP tool: batch-creates task beads with topological dependency resolution
  - /gsd-wired:plan SKILL.md slash command: plan generation, wave display, inline validation loop
  - gsdw plan CLI subcommand (stub redirecting to slash command)

affects: [07-wave-execution, 08-verification]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Topological sort pattern: iterative (not recursive), remaining-list approach per Pitfall 3"
    - "CreatePlanWithMeta: extended graph.Client method for task beads with complexity + files metadata"
    - "Plan SKILL.md pattern: generate plan -> create_plan_beads -> flush_writes -> validate loop (up to 3 iterations)"

key-files:
  created:
    - internal/mcp/create_plan_beads.go
    - internal/mcp/create_plan_beads_test.go
    - skills/plan/SKILL.md
    - internal/cli/plan.go
    - internal/cli/plan_test.go
  modified:
    - internal/mcp/tools.go
    - internal/mcp/server.go
    - internal/mcp/tools_test.go
    - internal/mcp/server_test.go
    - internal/cli/root.go
    - internal/graph/create.go

key-decisions:
  - "create_plan_beads uses iterative topological sort (remaining-list pass, not recursive) — avoids stack overflow for deep chains (Pitfall 3)"
  - "CreatePlanWithMeta added to graph.Client — extends CreatePlan with complexity/files metadata fields; existing CreatePlan unchanged for backward compatibility"
  - "plan CLI stub follows same pattern as research/init: full orchestration via SKILL.md Task(), not CLI"
  - "SKILL.md validation loop capped at 3 iterations with explicit iteration tracking per D-11/D-12"

patterns-established:
  - "Planning workflow: query_by_label(gsd:phase) -> query_by_label(gsd:research) -> generate plan -> create_plan_beads -> flush_writes -> validate"
  - "Inline validation: per-requirement query_by_label check + list_ready wave verification, up to 3 rounds"

requirements-completed: [PLAN-01, PLAN-02, PLAN-03, PLAN-04, CMD-03]

# Metrics
duration: 4min
completed: 2026-03-21
---

# Phase 6 Plan 02: Planning Workflow MCP Tool and SKILL.md Summary

**create_plan_beads MCP tool batch-creating dependency-ordered task beads with create_planWithMeta, /gsd-wired:plan SKILL.md orchestrating plan generation with inline 3-iteration requirement coverage validation and GSD-familiar wave display**

## Performance

- **Duration:** 4 min
- **Started:** 2026-03-21T21:47:49Z
- **Completed:** 2026-03-21T21:52:01Z
- **Tasks:** 2
- **Files modified:** 11

## Accomplishments

- Added `create_plan_beads` MCP tool (tool 13): batch-creates task beads from a structured JSON plan with iterative topological dependency resolution — local IDs (e.g. "06-01") resolve to actual bead IDs in dependency wiring order
- Each task bead carries complexity, files, and req_ids in metadata via new `CreatePlanWithMeta` graph.Client method
- Created `/gsd-wired:plan` SKILL.md: auto-generates full plan from research + phase context, displays in GSD-familiar wave/task format, validates requirement coverage inline up to 3 iterations, auto-proceeds after 30 seconds
- Added `gsdw plan` CLI subcommand (stub redirecting to slash command)
- Tool count increased from 12 to 13; all tests pass with -race

## Task Commits

Each task was committed atomically:

1. **Task 1: create_plan_beads MCP tool with TDD** - `976eb1b` (feat)
2. **Task 2: plan CLI subcommand and SKILL.md slash command** - `a094cf5` (feat)

## Files Created/Modified

- `internal/mcp/create_plan_beads.go` - handleCreatePlanBeads handler with topological dependency resolution
- `internal/mcp/create_plan_beads_test.go` - TestCreatePlanBeads + TestCreatePlanBeadsNoDeps + TestCreatePlanBeadsBadEpic
- `internal/mcp/tools.go` - create_plan_beads tool registration (13 tools total)
- `internal/mcp/server.go` - debug log updated to "tools", 13
- `internal/mcp/tools_test.go` - TestToolsRegistered updated to expect 13 tools + create_plan_beads
- `internal/mcp/server_test.go` - TestToolsListed updated to expect 13 tools + create_plan_beads
- `internal/graph/create.go` - CreatePlanWithMeta method added for extended metadata
- `skills/plan/SKILL.md` - /gsd-wired:plan slash command with full orchestration
- `internal/cli/plan.go` - NewPlanCmd stub
- `internal/cli/plan_test.go` - TestRootCmdHasPlan + TestPlanCmdOutput
- `internal/cli/root.go` - NewPlanCmd added to AddCommand chain

## Decisions Made

- Used iterative topological sort (remaining-list approach) not recursive DFS — prevents stack overflow for deep dependency chains (Pitfall 3 from research)
- Added `CreatePlanWithMeta` to graph.Client rather than modifying `CreatePlan` — backward compatible, existing 12 tool handlers unaffected
- SKILL.md validation loop explicitly tracks iteration count ("Validation iteration N of 3") per D-11 requirement
- Plan CLI subcommand is a stub (same pattern as research/init) — parallel orchestration via Task() is a Claude Code native capability, not CLI

## Deviations from Plan

None — plan executed exactly as written.

## Known Stubs

None. The CLI stub is intentional per plan design (redirects to SKILL.md slash command). All functional behavior is implemented in the MCP tool and SKILL.md.

---

## Self-Check: PASSED

Files verified:
- internal/mcp/create_plan_beads.go: FOUND
- internal/mcp/create_plan_beads_test.go: FOUND
- skills/plan/SKILL.md: FOUND
- internal/cli/plan.go: FOUND
- internal/cli/plan_test.go: FOUND

Commits verified:
- 976eb1b: FOUND
- a094cf5: FOUND

*Phase: 06-research-planning*
*Completed: 2026-03-21*
