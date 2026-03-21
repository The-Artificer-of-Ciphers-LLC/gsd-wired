---
phase: 04-hook-integration
plan: 02
subsystem: infra
tags: [hooks, claude-code, pre-compact, pre-tool-use, post-tool-use, go, tdd]

# Dependency graph
requires:
  - phase: 04-hook-integration
    plan: 01
    provides: hookState, writeOutput, HookOutput types, PreCompactInput, PreToolUseInput, PostToolUseInput, dispatcher stubs
  - phase: 02-graph-primitives
    provides: graph.Client, LoadIndex, ListReady, Bead types
provides:
  - handlePreCompact with atomic local snapshot write (no goroutines)
  - handlePreToolUse with contextInjectTools fast path and local index lookup
  - handlePostToolUse with beadUpdateTools JSONL event recording
  - All four Claude Code hooks fully wired in dispatcher
  - 55 hook package tests pass with -race
affects:
  - Phase 5+ (precompact-snapshot.json and tool-events.jsonl ready for sync)

# Tech tracking
tech-stack:
  added: []
  patterns:
    - Atomic write via temp+rename for precompact snapshot (same pattern as graph/index.go)
    - contextInjectTools map: write-class tools get context, read-class tools fast path
    - beadUpdateTools map: write-class tools get JSONL event appended
    - Best-effort writes: any file system error logged but never returned (hooks must not block)
    - Local index as cheap context source (<1ms) before expensive graph query

key-files:
  created:
    - internal/hook/pre_compact.go (handlePreCompact — atomic snapshot, no goroutines)
    - internal/hook/pre_compact_test.go (6 tests: local write, output, atomic, dir, error, purity)
    - internal/hook/pre_tool_use.go (handlePreToolUse — contextInjectTools, index lookup, 400ms graph timeout)
    - internal/hook/pre_tool_use_test.go (6 tests: write tool, read tool, glob, latency, no beads, purity)
    - internal/hook/post_tool_use.go (handlePostToolUse — beadUpdateTools, JSONL appender)
    - internal/hook/post_tool_use_test.go (5 tests: write tool, read tool, multiple writes, output, purity)
  modified:
    - internal/hook/dispatcher.go (replaced 3 stubs with handlePreCompact/handlePreToolUse/handlePostToolUse)
    - internal/hook/dispatcher_test.go (added routing tests for all three new handlers)

key-decisions:
  - "PreCompact writes to .gsdw/precompact-snapshot.json atomically via temp+rename — no goroutines (research Pitfall 2)"
  - "PreToolUse fast path for read-class tools exits before any graph or file I/O — zero overhead for Read/Glob/Grep"
  - "PreToolUse loads .gsdw/index.json as cheap context source (<1ms) before attempting 400ms graph query"
  - "PostToolUse records Write/Edit/Bash to JSONL (Agent excluded) — minimal set, no additionalContext injection (deferred to v2/TOKEN-A01)"

requirements-completed: [INFRA-06, INFRA-07, INFRA-08]

# Metrics
duration: 4min
completed: 2026-03-21
---

# Phase 4 Plan 02: PreCompact, PreToolUse, and PostToolUse Handlers Summary

**PreCompact writes atomic local snapshot, PreToolUse fast-paths reads and injects index context for writes, PostToolUse records JSONL events for write-class tools — all four hooks fully wired in dispatcher**

## Performance

- **Duration:** 4 min
- **Started:** 2026-03-21T20:05:32Z
- **Completed:** 2026-03-21T20:09:50Z
- **Tasks:** 2
- **Files modified:** 8 (6 created, 2 modified)

## Accomplishments

- Implemented handlePreCompact: writes compactSnapshot to `.gsdw/precompact-snapshot.json` atomically via temp+rename, returns empty HookOutput immediately, no goroutines
- Implemented handlePreToolUse: contextInjectTools fast path for read-class tools (Read/Glob/Grep/WebFetch), local `.gsdw/index.json` lookup for write-class tools, optional 400ms graph query when `.beads/` present
- Implemented handlePostToolUse: beadUpdateTools recording of Write/Edit/Bash events to `.gsdw/tool-events.jsonl` (JSON Lines append), no-op for read-class tools
- Wired all three handlers into dispatcher (replaced Plan 01 stubs)
- Added dispatcher routing tests for PreCompact, PreToolUse, PostToolUse
- 55 hook tests pass with -race; full 7-package suite green

## Task Commits

Each task was committed atomically:

1. **Task 1: PreCompact handler with atomic local buffer** - `5f087b7` (feat)
2. **Task 2: PreToolUse and PostToolUse handlers with dispatcher wiring** - `efd4142` (feat)

## Files Created/Modified

- `internal/hook/pre_compact.go` — New: handlePreCompact, compactSnapshot struct, atomic write via temp+rename
- `internal/hook/pre_compact_test.go` — New: 6 tests (local write, output, atomic, dir creation, write error, purity)
- `internal/hook/pre_tool_use.go` — New: handlePreToolUse, contextInjectTools map, buildPreToolUseContext, preToolUseAllow
- `internal/hook/pre_tool_use_test.go` — New: 6 tests (write tool, read tool, glob fast path, latency, no beads, purity)
- `internal/hook/post_tool_use.go` — New: handlePostToolUse, beadUpdateTools map, toolEvent struct, JSONL appender
- `internal/hook/post_tool_use_test.go` — New: 5 tests (write tool, read tool, multiple writes, output, purity)
- `internal/hook/dispatcher.go` — Modified: replaced 3 stubs with real handlers
- `internal/hook/dispatcher_test.go` — Modified: added TestDispatchPreCompactRoute, TestDispatchPreToolUseRoute, TestDispatchPostToolUseRoute

## Decisions Made

- PreCompact uses no goroutines — process exits immediately after compaction, goroutines are killed before completing async Dolt sync (research Pitfall 2)
- PreToolUse fast path returns before any file I/O for read-class tools — zero overhead
- PreToolUse loads local index as the primary context source (near-zero latency) before attempting graph query
- PostToolUse does NOT inject additionalContext — deferred to v2/TOKEN-A01 per research Open Question 3

## Deviations from Plan

None — plan executed exactly as written.

## Known Stubs

None — all handlers produce real behavior. PostToolUse defers additionalContext injection to v2 (TOKEN-A01) by design; this is an intentional scope boundary, not a stub.

## Self-Check: PASSED

- FOUND: internal/hook/pre_compact.go
- FOUND: internal/hook/pre_compact_test.go
- FOUND: internal/hook/pre_tool_use.go
- FOUND: internal/hook/pre_tool_use_test.go
- FOUND: internal/hook/post_tool_use.go
- FOUND: internal/hook/post_tool_use_test.go
- FOUND: internal/hook/dispatcher.go (updated)
- FOUND commit 5f087b7 (Task 1)
- FOUND commit efd4142 (Task 2)
- Full test suite: 7/7 packages pass with -race (55 hook tests)

---
*Phase: 04-hook-integration*
*Completed: 2026-03-21*
