---
phase: 04-hook-integration
plan: 03
subsystem: infra
tags: [hooks, gap-closure, infra-06, infra-08, dolt-sync, bead-state, go]

# Dependency graph
requires:
  - phase: 04-hook-integration
    plan: 02
    provides: handleSessionStart, handlePostToolUse, hookState, compactSnapshot, precompact-snapshot.json, tool-events.jsonl
  - phase: 02-graph-primitives
    provides: graph.Client, AddLabel, LoadIndex, QueryByLabel, Bead types
provides:
  - UpdateBeadMetadata method on graph.Client (bd update --metadata JSON merge patch)
  - syncPendingSnapshot helper in session_start.go (INFRA-06 Stage 2)
  - updateBeadOnToolUse helper in post_tool_use.go (INFRA-08 bead state)
  - 6 new tests covering sync happy path, sync error, no-snapshot, bead update, no-beads, bead update error
affects:
  - INFRA-06: fully satisfied (two-stage: PreCompact local write + SessionStart Dolt sync)
  - INFRA-08: fully satisfied (PostToolUse JSONL + AddLabel on active plan bead)

# Tech tracking
tech-stack:
  added: []
  patterns:
    - UpdateBeadMetadata via bd update --metadata: JSON merge patch, same runWrite() pattern as AddLabel
    - syncPendingSnapshot: os.Stat fast path, unmarshal, QueryByLabel for phase bead, UpdateBeadMetadata, os.Remove on success
    - updateBeadOnToolUse: os.Stat fast path for .beads/, graph.LoadIndex for active bead, AddLabel best-effort
    - Graceful degradation: slog.Warn + return on every error path, never block hook

key-files:
  created: []
  modified:
    - internal/graph/update.go (added UpdateBeadMetadata method)
    - internal/graph/graph_test.go (added TestUpdateBeadMetadata)
    - internal/graph/testdata/fake_bd/main.go (fixed query case to return phase bead for label=gsd:phase)
    - internal/hook/session_start.go (added syncPendingSnapshot helper + call in handleSessionStart)
    - internal/hook/session_start_test.go (added TestSessionStartSyncsSnapshot, TestSessionStartNoSnapshot, TestSessionStartSnapshotSyncError)
    - internal/hook/post_tool_use.go (added updateBeadOnToolUse helper + call after JSONL write)
    - internal/hook/post_tool_use_test.go (added TestPostToolUseBeadUpdate, TestPostToolUseBeadUpdateNoBeads, TestPostToolUseBeadUpdateError)

key-decisions:
  - "syncPendingSnapshot uses QueryByLabel not LoadIndex to find phase bead — LoadIndex maps phase numbers, QueryByLabel finds the active open phase directly from graph"
  - "Snapshot deletion is proof of sync — os.Remove only runs after UpdateBeadMetadata succeeds, making file presence/absence the reliable sync indicator"
  - "updateBeadOnToolUse uses AddLabel(gsd:tool-use) not UpdateBeadMetadata — minimal change surface, AddLabel already tested, satisfies INFRA-08 without new metadata schema"
  - "fake_bd query case updated to return cannedPhaseBead for label=gsd:phase — enables sync tests without changing test architecture"
  - "TestSessionStartSyncsSnapshot verifies file removal not capture args — multiple bd calls in one handler overwrite capture file, file deletion is idempotent proof"

requirements: [INFRA-06, INFRA-08]

# Metrics
duration: 5min
completed: 2026-03-21
---

# Phase 4 Plan 03: Gap Closure (INFRA-06 Stage 2 + INFRA-08 Bead Updates) Summary

**UpdateBeadMetadata added to graph.Client; SessionStart syncs pending precompact-snapshot.json to Dolt; PostToolUse adds gsd:tool-use label to active bead — both INFRA-06 and INFRA-08 now fully satisfied**

## Performance

- **Duration:** ~5 min
- **Started:** 2026-03-21T20:29:50Z
- **Completed:** 2026-03-21T20:34:26Z
- **Tasks:** 2
- **Files modified:** 7

## Accomplishments

- Added `UpdateBeadMetadata(ctx, beadID, meta)` method to `graph.Client` wrapping `bd update --metadata JSON_STRING` (JSON merge patch)
- Added `syncPendingSnapshot` to `session_start.go`: detects `.gsdw/precompact-snapshot.json`, queries active phase bead via `QueryByLabel("gsd:phase")`, calls `UpdateBeadMetadata` with `last_precompact` and `last_session_id`, removes file on success, degrades gracefully on any error
- Wired `syncPendingSnapshot` into `handleSessionStart` after `hs.init()` succeeds, using the same 1500ms `queryCtx`
- Added `updateBeadOnToolUse` to `post_tool_use.go`: checks `.beads/` existence (fast path), loads `.gsdw/index.json` via `graph.LoadIndex`, calls `AddLabel("gsd:tool-use")` on active plan bead with 400ms timeout
- Fixed `fake_bd` `query` case to return `cannedPhaseBead` for `label=gsd:phase` queries (needed for sync test)
- 6 new tests all pass: snapshot sync (happy + error + no-snapshot), bead update (happy + no-beads + error)
- 121 total tests pass across 7 packages with `-race`

## Task Commits

Each task was committed atomically:

1. **Task 1: UpdateBeadMetadata + syncPendingSnapshot in SessionStart** - `e10a53b` (feat)
2. **Task 2: PostToolUse bead state update via AddLabel** - `18bea3a` (feat)

## Files Created/Modified

- `internal/graph/update.go` — Added `UpdateBeadMetadata` method
- `internal/graph/graph_test.go` — Added `TestUpdateBeadMetadata`
- `internal/graph/testdata/fake_bd/main.go` — Fixed `query` case for `label=gsd:phase`
- `internal/hook/session_start.go` — Added `syncPendingSnapshot` helper + call in `handleSessionStart`
- `internal/hook/session_start_test.go` — Added 3 new snapshot sync tests
- `internal/hook/post_tool_use.go` — Added `updateBeadOnToolUse` helper + call after JSONL write; import graph package
- `internal/hook/post_tool_use_test.go` — Added 3 new bead update tests; import graph package

## Decisions Made

- `syncPendingSnapshot` uses `QueryByLabel` not `LoadIndex` to find the phase bead — `LoadIndex` maps phase numbers to IDs but doesn't indicate which is active; `QueryByLabel` surfaces open phase beads directly
- Snapshot file deletion is the reliable sync proof in tests — multiple bd calls in one handler overwrite the capture file, so file presence/absence is the idempotent indicator
- `updateBeadOnToolUse` uses `AddLabel("gsd:tool-use")` rather than `UpdateBeadMetadata` — minimal change surface, satisfies INFRA-08 requirement for bead state update, AddLabel already well-tested
- `fake_bd` `query` case updated to return `cannedPhaseBead` for `label=gsd:phase` — Rule 3 fix (blocking issue: test couldn't verify sync without a phase bead response)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] fake_bd query case returned empty array for all label queries**
- **Found during:** Task 1 (TestSessionStartSyncsSnapshot failing — "no active phase bead found")
- **Issue:** `fake_bd`'s `query` subcommand returned `[]` for all inputs including `label=gsd:phase`. `syncPendingSnapshot` uses `QueryByLabel("gsd:phase")` which uses `bd query label=gsd:phase`. No phase bead was returned, so no `UpdateBeadMetadata` call was made, so the snapshot file was not deleted.
- **Fix:** Updated `fake_bd/main.go` to check if `label=gsd:phase` appears in query args and return `cannedPhaseBead` in that case. Added `strings` import.
- **Files modified:** `internal/graph/testdata/fake_bd/main.go`
- **Commit:** e10a53b (Task 1 commit)

**2. [Rule 1 - Bug] TestSessionStartSyncsSnapshot used capture file to verify update args**
- **Found during:** Task 1 (test still failing after fake_bd fix — capture file showed last command was `ready --limit 0`)
- **Issue:** `handleSessionStart` makes multiple bd calls in sequence (QueryByLabel, UpdateBeadMetadata, QueryByLabel again for buildSessionContext, ListReady). Each call overwrites `FAKE_BD_CAPTURE_FILE`. The capture file only shows the last call, which is `ready --limit 0`, not the `update --metadata` call.
- **Fix:** Changed test to verify snapshot file removal (os.Remove only runs after UpdateBeadMetadata succeeds — file deletion is the reliable proof of the full sync path running).
- **Files modified:** `internal/hook/session_start_test.go`
- **Commit:** e10a53b (Task 1 commit, same)

## Known Stubs

None — all sync paths produce real behavior. Both gaps are fully closed.

## Self-Check: PASSED

- FOUND: internal/graph/update.go (UpdateBeadMetadata method present)
- FOUND: internal/hook/session_start.go (syncPendingSnapshot + precompact-snapshot.json present)
- FOUND: internal/hook/post_tool_use.go (AddLabel + gsd:tool-use present)
- FOUND: internal/hook/session_start_test.go (TestSessionStartSyncsSnapshot present)
- FOUND: internal/graph/graph_test.go (TestUpdateBeadMetadata present)
- FOUND: internal/hook/post_tool_use_test.go (TestPostToolUseBeadUpdate present)
- FOUND: internal/hook/post_tool_use_test.go (TestPostToolUseBeadUpdateNoBeads present)
- FOUND commit e10a53b (Task 1)
- FOUND commit 18bea3a (Task 2)
- Full test suite: 121/121 tests pass with -race across 7 packages

---
*Phase: 04-hook-integration*
*Completed: 2026-03-21*
