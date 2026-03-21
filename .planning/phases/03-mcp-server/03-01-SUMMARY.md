---
phase: 03-mcp-server
plan: 01
subsystem: infra
tags: [go, graph, batch-writes, dolt, sync-once, mcp, lazy-init]

requires:
  - phase: 02-graph-primitives
    provides: graph.Client with NewClientWithPath, CreatePhase, ListReady, ClaimBead, ClosePlan, AddLabel, fake_bd test infrastructure

provides:
  - "graph.Client batch write mode: runWrite() prepends --dolt-auto-commit=batch for all mutating ops"
  - "graph.Client FlushWrites() method: commits batched writes via bd dolt commit"
  - "NewClientBatch / NewClientWithPathBatch constructors for batch mode"
  - "serverState type with sync.Once lazy initialization of batch-mode graph.Client"
  - "Automatic bd init when .beads/ directory does not exist (30s timeout)"
  - "Permanent error storage on init failure (sync.Once does not retry)"

affects: [03-mcp-server/02, 04-hooks]

tech-stack:
  added: []
  patterns:
    - "runWrite() delegates to run() with --dolt-auto-commit=batch prepended as global bd flag before subcommand"
    - "sync.Once lazy init pattern: serverState.init() blocks on first call, returns stored result on all subsequent calls"
    - "initTimeout field in serverState for test-configurable timeout without changing prod behavior"
    - "fake_bd: global flags (--dolt-auto-commit=batch) stripped before subcommand dispatch"

key-files:
  created:
    - internal/graph/batch_test.go
    - internal/mcp/init.go
    - internal/mcp/init_test.go
  modified:
    - internal/graph/client.go
    - internal/graph/create.go
    - internal/graph/update.go
    - internal/graph/testdata/fake_bd/main.go

key-decisions:
  - "runWrite() as separate method from run(): write ops get batch flag, reads never do — clean separation without conditional noise at each call site"
  - "FlushWrites uses run() not runWrite(): the dolt commit itself is not a batched operation"
  - "initTimeout int field (milliseconds) in serverState: allows test-configurable timeout without changing the 30s production default"
  - "fake_bd strips leading --flag args before dispatch: enables testing global bd flags without special-casing every test"

patterns-established:
  - "Pattern: write ops use runWrite(), read ops use run() — enforced at call site in create.go and update.go"
  - "Pattern: serverState{beadsDir, bdPath} for test construction — mirrors graph.NewClientWithPath injection pattern"
  - "Pattern: FAKE_BD_CAPTURE_FILE captures args for structural verification across both graph and mcp packages"

requirements-completed: [INFRA-02, INFRA-10]

duration: 4min
completed: 2026-03-21
---

# Phase 3 Plan 01: MCP Server Infrastructure Summary

**Batch write mode added to graph.Client via runWrite()/FlushWrites(), and serverState with sync.Once lazy init auto-creates a batch-mode client and runs bd init when .beads/ is missing**

## Performance

- **Duration:** 4 min
- **Started:** 2026-03-21T19:05:59Z
- **Completed:** 2026-03-21T19:09:55Z
- **Tasks:** 2 (each TDD: RED commit + GREEN commit)
- **Files modified:** 7

## Accomplishments

- graph.Client batch write mode: write ops prepend `--dolt-auto-commit=batch` as a global bd flag before the subcommand; reads do not
- FlushWrites() method calls `bd dolt commit --message "gsdw: batch flush"` to persist accumulated batch writes
- serverState type with sync.Once: first call to init() blocks and creates batch-mode graph.Client, subsequent calls return stored result immediately
- Automatic bd init with 30-second timeout when .beads/ directory does not exist
- 10 new tests (4 batch/flush + 6 lazy-init), all 7 packages pass with -race

## Task Commits

Each task was committed atomically:

1. **Task 1 RED: Batch write mode tests** - `341c539` (test)
2. **Task 1 GREEN: Batch write mode implementation** - `4e30738` (feat)
3. **Task 2 RED: Lazy init serverState tests** - `47f3111` (test)
4. **Task 2 GREEN: serverState with sync.Once** - `1ded7e7` (feat)

_Note: TDD tasks have separate test (RED) and implementation (GREEN) commits_

## Files Created/Modified

- `internal/graph/client.go` - Added batchMode field, NewClientBatch, NewClientWithPathBatch, runWrite(), FlushWrites()
- `internal/graph/create.go` - CreatePhase and CreatePlan now use runWrite() instead of run()
- `internal/graph/update.go` - ClaimBead, ClosePlan, AddLabel now use runWrite() instead of run()
- `internal/graph/testdata/fake_bd/main.go` - Added global flag stripping, dolt and init command handlers
- `internal/graph/batch_test.go` - 4 tests: TestBatchFlagOnWrite, TestBatchFlagNotOnRead, TestFlushWrites, TestFlushWritesError
- `internal/mcp/init.go` - serverState type with sync.Once lazy init and runBdInit()
- `internal/mcp/init_test.go` - 6 tests: TestLazyInit{CreatesClient, RunsBdInit, OnlyOnce, ErrorStored, Timeout, BatchMode}

## Decisions Made

- runWrite() as separate method rather than modifying run(): write ops get batch flag, reads never do — clean separation without conditional logic at each call site
- FlushWrites uses run() not runWrite(): the dolt commit itself is not a batched operation
- initTimeout int field (milliseconds) in serverState: allows test-configurable timeout without changing the 30s production default
- fake_bd strips leading --flag args before dispatch: enables testing global bd flags without special-casing every test scenario

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None - all tests passed on first implementation attempt.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- batch-mode graph.Client and serverState lazy init are ready for MCP tool handlers (Plan 02)
- serverState is an unexported type in package mcp — Plan 02 MCP tools will call s.init(ctx) as their first step
- All 7 packages pass -race with full test suite (25 graph tests + 7 mcp tests + cmd/cli/hook/logging/version)

## Self-Check: PASSED

- internal/graph/client.go: FOUND
- internal/graph/batch_test.go: FOUND
- internal/mcp/init.go: FOUND
- internal/mcp/init_test.go: FOUND
- .planning/phases/03-mcp-server/03-01-SUMMARY.md: FOUND
- commit 341c539 (test: batch mode RED): FOUND
- commit 4e30738 (feat: batch mode GREEN): FOUND
- commit 47f3111 (test: lazy init RED): FOUND
- commit 1ded7e7 (feat: lazy init GREEN): FOUND

---
*Phase: 03-mcp-server*
*Completed: 2026-03-21*
