---
phase: 07-execution-verification
plan: 01
subsystem: mcp
tags: [mcp, execute_wave, verify_phase, tdd, fake_bd, context-chain, acceptance-criteria]

# Dependency graph
requires:
  - phase: 06-research-planning
    provides: create_plan_beads MCP tool (tool 13), graph client, fake_bd test infrastructure
provides:
  - execute_wave MCP tool (tool 14) returning full context chains per task
  - verify_phase MCP tool (tool 15) checking acceptance criteria against codebase state
  - fake_bd FAKE_BD_SHOW_RESPONSE support for parameterized show responses
  - fake_bd FAKE_BD_QUERY_PHASE_RESPONSE support for custom phase query injection
  - phaseNumFromMeta / planIDFromMeta private helpers in mcp package
affects: [08-skills-slash-commands, execute-slash-command, verify-slash-command]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "execute_wave pre-computes full context chain (bead + parent + deps) per D-04 minimal prompt"
    - "verify_phase uses method dispatch: file_exists (os.Stat), go_test (exec with 60s timeout), manual fallback"
    - "checkCriterion dispatches on criterion text: path separator / extension → file_exists, 'test' keyword → go_test"
    - "FAKE_BD_*_RESPONSE env vars for per-subcommand fake_bd test parameterization"
    - "fake_bd FAKE_BD_QUERY_PHASE_RESPONSE checked before cannedPhaseBead for query subcommand"

key-files:
  created:
    - internal/mcp/execute_wave.go
    - internal/mcp/execute_wave_test.go
    - internal/mcp/verify_phase.go
    - internal/mcp/verify_phase_test.go
  modified:
    - internal/mcp/tools.go
    - internal/mcp/server.go
    - internal/mcp/tools_test.go
    - internal/mcp/server_test.go
    - internal/graph/testdata/fake_bd/main.go

key-decisions:
  - "execute_wave reports wave=1 always in v1 (dynamic wave computation deferred to v2)"
  - "verify_phase 'failed' array contains raw criterion text for SKILL.md remediation per D-10"
  - "verify_phase does NOT call create_plan_beads — per D-10, SKILL.md handles remediation"
  - "go_test uses exec.CommandContext with 60-second timeout (Pitfall 5 from research doc)"
  - "hasUppercaseIdentifier identifier pattern falls through to manual method in v1"
  - "FAKE_BD_QUERY_PHASE_RESPONSE added to fake_bd query subcommand — checked before canned fallback"
  - "verify_phase.go written in full alongside tests (not as a stub) since both tasks are in same plan"

patterns-established:
  - "Tool 14+15 pattern: register in tools.go, implement in named file, test in named _test.go"
  - "Private meta helpers (phaseNumFromMeta, planIDFromMeta) duplicated in mcp package — DO NOT import from cli"

requirements-completed: [EXEC-01, EXEC-02, EXEC-03, EXEC-04, VRFY-01, VRFY-02, VRFY-03]

# Metrics
duration: 8min
completed: 2026-03-21
---

# Phase 7 Plan 01: Execution + Verification Summary

**execute_wave (tool 14) pre-computes task context chains from phase epic + dep CloseReasons; verify_phase (tool 15) checks acceptance criteria against codebase via file_exists, go_test, and manual methods — 15 MCP tools total, 149 tests pass**

## Performance

- **Duration:** 8 min
- **Started:** 2026-03-21T21:03:21Z
- **Completed:** 2026-03-21T21:11:23Z
- **Tasks:** 2
- **Files modified:** 9

## Accomplishments
- execute_wave MCP tool pre-computes full context chain (bead + parent summary + dep CloseReasons) for all ready tasks in a phase, enabling D-04 minimal prompt for execution agents
- verify_phase MCP tool parses acceptance criteria and dispatches to file_exists (os.Stat), go_test (exec with 60s timeout), or manual check methods — Failed array feeds SKILL.md remediation per D-10
- fake_bd extended with FAKE_BD_SHOW_RESPONSE (show subcommand) and FAKE_BD_QUERY_PHASE_RESPONSE (query subcommand) for hermetic test parameterization
- Tool count updated from 13 to 15 across all 4 required files atomically

## Task Commits

Each task was committed atomically:

1. **Task 1: execute_wave MCP tool with TDD** - `fc3b21d` (feat)
2. **Task 2: verify_phase MCP tool with TDD** - `57f5f25` (feat)

**Plan metadata:** (docs commit after SUMMARY.md)

_Note: TDD tasks — implementation completed in same commit as tests since verify_phase.go was needed for tools.go compilation in Task 1_

## Files Created/Modified
- `internal/mcp/execute_wave.go` - handleExecuteWave handler with taskContext/executeWaveResult types and phaseNumFromMeta/planIDFromMeta helpers
- `internal/mcp/execute_wave_test.go` - 4 tests: TestExecuteWave, TestExecuteWaveContextChain, TestExecuteWaveEmpty, TestExecuteWaveNoPhase
- `internal/mcp/verify_phase.go` - handleVerifyPhase handler with criterionResult/verifyPhaseResult types, checkCriterion dispatcher, extractFilePath, hasUppercaseIdentifier
- `internal/mcp/verify_phase_test.go` - 5 tests: TestVerifyPhase, TestVerifyPhaseFileCheck, TestVerifyPhaseGoTest, TestVerifyPhaseFailures, TestVerifyPhaseNoPhase
- `internal/mcp/tools.go` - count updated to 15, execute_wave and verify_phase registered as tools 14 and 15
- `internal/mcp/server.go` - debug log count updated to 15
- `internal/mcp/tools_test.go` - count updated to 15, "execute_wave" and "verify_phase" added to wantNames
- `internal/mcp/server_test.go` - count updated to 15, both new tools added to wantNames
- `internal/graph/testdata/fake_bd/main.go` - FAKE_BD_SHOW_RESPONSE on show subcommand, FAKE_BD_QUERY_PHASE_RESPONSE on query subcommand

## Decisions Made
- execute_wave reports wave=1 always in v1; dynamic computation via dependency depth analysis deferred to v2
- verify_phase "failed" array contains raw criterion text (not IDs) — SKILL.md receives this for remediation prompts per D-10
- verify_phase does NOT call create_plan_beads directly — remediation is SKILL.md's job per design decision D-10
- go_test method uses exec.CommandContext with 60-second context timeout as specified in Pitfall 5
- Uppercase Go identifier pattern (e.g. HandleExecuteWave) defaults to "manual" method in v1 — grep-based scan deferred
- FAKE_BD_QUERY_PHASE_RESPONSE checked before cannedPhaseBead fallback, preserving backward compatibility for all existing tests

## Deviations from Plan

None — plan executed exactly as written. verify_phase.go was implemented fully in Task 1 (not as a stub) because tools.go required the `verifyPhaseArgs` and `handleVerifyPhase` types to compile — this is the "Update count to 15 atomically" instruction from the plan's Task 1 note.

## Issues Encountered
None — all tests passed on first run after implementation.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- execute_wave and verify_phase handlers are ready for /gsd-wired:execute and /gsd-wired:verify SKILL.md slash commands
- Phase 7 Plan 02 (execute-slash-command) can begin immediately
- All 15 MCP tools registered and tested; 149 tests pass with -race

## Self-Check: PASSED
- All 5 created/modified files confirmed present on disk
- Commits fc3b21d and 57f5f25 confirmed in git log

---
*Phase: 07-execution-verification*
*Completed: 2026-03-21*
