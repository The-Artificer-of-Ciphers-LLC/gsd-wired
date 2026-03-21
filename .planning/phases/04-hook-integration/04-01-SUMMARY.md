---
phase: 04-hook-integration
plan: 01
subsystem: infra
tags: [hooks, claude-code, session-start, graph-client, go]

# Dependency graph
requires:
  - phase: 03-mcp-server
    provides: serverState pattern and lazy graph.Client init via sync.Once
  - phase: 02-graph-primitives
    provides: graph.Client, Bead types, QueryByLabel, ListReady
provides:
  - Extended per-event hook input structs (SessionStartInput, PreCompactInput, PreToolUseInput, PostToolUseInput)
  - Extended HookOutput with AdditionalContext and HookSpecificOutput
  - hookState with sync.Once lazy non-batch graph.Client init
  - writeOutput helper as single exit path for all hook handlers
  - handleSessionStart emitting beads-sourced project context on session start
  - buildSessionContext querying gsd:phase beads and ListReady for markdown context
  - Dispatcher routing SessionStart to real handler; PreCompact/PreToolUse/PostToolUse to stubs
  - Dispatcher now accepts context.Context (passed from cmd.Context())
affects:
  - 04-02 (Plan 02 reuses hookState, HookOutput, writeOutput for PreCompact/PreToolUse/PostToolUse)

# Tech tracking
tech-stack:
  added: []
  patterns:
    - hookState pattern (mirrors serverState but non-batch, no bd init — hooks are read-only)
    - writeOutput as single exit path — all handler return paths funnel through one JSON encoder
    - Fast path for .beads/ absence — check os.Stat before any graph init
    - Degraded mode — slog.Warn + writeOutput(empty) on any init/query failure, never crash

key-files:
  created:
    - internal/hook/events.go (extended — per-event input structs + richer HookOutput)
    - internal/hook/hook_state.go (new — hookState with sync.Once, writeOutput helper)
    - internal/hook/hook_state_test.go (new — type decode tests, hookState init tests)
    - internal/hook/session_start.go (new — handleSessionStart, buildSessionContext)
    - internal/hook/session_start_test.go (new — handler behavior, latency, purity tests)
  modified:
    - internal/hook/dispatcher.go (add context.Context, route SessionStart, stub others)
    - internal/hook/dispatcher_test.go (update Dispatch calls, add routing/all-events tests)
    - internal/cli/hook.go (pass cmd.Context() to Dispatch)

key-decisions:
  - "hookState validates bdPath existence on init (os.Stat) so init() returns error immediately rather than deferring to first graph call"
  - "buildSessionContext always returns string, never error — partial context better than nothing, errors logged to slog.Warn"
  - "handleSessionStart sets hs.beadsDir from input.CWD before calling init() — CWD drives the beads directory for hook context"
  - "formatSessionContext uses []byte append instead of strings.Builder — consistent with Go stdlib patterns, avoids fmt.Sprintf overhead"

patterns-established:
  - "hookState pattern: non-batch graph.Client for hooks (read-only), vs serverState batch for MCP (read-write)"
  - "Fast path before expensive init: check .beads/ existence with os.Stat before initializing graph client"
  - "Single writeOutput exit: all handlers return via writeOutput(w, HookOutput{...}) for stdout purity"

requirements-completed: [INFRA-05]

# Metrics
duration: 5min
completed: 2026-03-21
---

# Phase 4 Plan 01: Hook Type System and SessionStart Handler Summary

**Extended hook type system with per-event structs, hookState lazy graph init, and SessionStart handler that emits beads-sourced project context as additionalContext**

## Performance

- **Duration:** 5 min
- **Started:** 2026-03-21T19:54:53Z
- **Completed:** 2026-03-21T20:00:00Z
- **Tasks:** 2
- **Files modified:** 8

## Accomplishments
- Extended events.go: per-event input structs (SessionStartInput/PreCompactInput/PreToolUseInput/PostToolUseInput) and richer HookOutput with AdditionalContext and HookSpecificOutput
- Created hookState with sync.Once lazy non-batch graph.Client init and writeOutput single-exit helper
- Implemented handleSessionStart with fast path (no .beads/ -> init hint), degraded mode (graph error -> empty JSON), and 1500ms timeout within 2s budget
- Updated dispatcher to accept context.Context and route SessionStart to real handler, other three events to stubs
- 28 hook tests pass with -race; full 7-package suite green

## Task Commits

Each task was committed atomically:

1. **Task 1: Extend hook types and create hookState** - `1933423` (feat)
2. **Task 2: SessionStart handler and dispatcher routing** - `eb3fc35` (feat)

## Files Created/Modified
- `internal/hook/events.go` - Extended with per-event input structs and richer HookOutput (replaced HookInput stub)
- `internal/hook/hook_state.go` - New: hookState struct with sync.Once lazy init and writeOutput helper
- `internal/hook/hook_state_test.go` - New: type decode tests, hookState init/error/once tests, writeOutput test
- `internal/hook/session_start.go` - New: handleSessionStart, buildSessionContext, formatSessionContext
- `internal/hook/session_start_test.go` - New: handler behavior, latency, stdout purity tests
- `internal/hook/dispatcher.go` - Rewrote: add context.Context, route events, use json.RawMessage
- `internal/hook/dispatcher_test.go` - Updated: pass ctx to Dispatch, add routing/all-events tests
- `internal/cli/hook.go` - Pass cmd.Context() to Dispatch

## Decisions Made
- hookState validates bdPath existence via os.Stat on init() so the error surfaces immediately rather than at first graph call
- buildSessionContext always returns string (never error) — partial results better than nothing, errors logged to slog.Warn
- handleSessionStart sets hs.beadsDir from input.CWD before init() — CWD is the authoritative beads directory for hooks
- Dispatcher creates a fresh hookState per invocation — hooks are short-lived, no state reuse needed

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] hookState.init() did not validate bdPath existence before NewClientWithPath**
- **Found during:** Task 1 (TestHookStateInitError was failing)
- **Issue:** NewClientWithPath does not validate that the binary exists — it only stores the path. The test expected init() to return an error for a nonexistent binary, but init() succeeded.
- **Fix:** Added os.Stat check in hookState.init() before calling NewClientWithPath when bdPath is set
- **Files modified:** internal/hook/hook_state.go
- **Verification:** TestHookStateInitError passes
- **Committed in:** 1933423 (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (Rule 1 - bug)
**Impact on plan:** Necessary fix — the test specified the intended behavior and the implementation was incorrect without it. No scope creep.

## Issues Encountered
None beyond the auto-fixed deviation above.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- hookState and writeOutput ready for Plan 02 (PreCompact, PreToolUse, PostToolUse handlers)
- Extended type system (PreToolUseInput, PostToolUseInput, PreCompactInput) ready for Plan 02
- Dispatcher stubs in place; Plan 02 replaces with real handlers
- INFRA-05 satisfied: SessionStart emits additionalContext when .beads/ present

## Self-Check: PASSED

- FOUND: internal/hook/events.go
- FOUND: internal/hook/hook_state.go
- FOUND: internal/hook/session_start.go
- FOUND: internal/hook/dispatcher.go
- FOUND: .planning/phases/04-hook-integration/04-01-SUMMARY.md
- FOUND commit 1933423 (Task 1)
- FOUND commit eb3fc35 (Task 2)
- Full test suite: 7/7 packages pass with -race

---
*Phase: 04-hook-integration*
*Completed: 2026-03-21*
