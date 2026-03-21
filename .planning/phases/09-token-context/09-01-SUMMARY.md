---
phase: 09-token-context
plan: "01"
subsystem: graph
tags: [tiering, token-estimation, compaction, context-routing]
dependency_graph:
  requires: []
  provides: [graph.Tier, graph.TierHot, graph.TierWarm, graph.TierCold, graph.TieredBead, graph.classifyTier, graph.estimateTokens, graph.compactSummary, graph.CompactBead, graph.QueryTiered]
  affects: [internal/graph/update.go, internal/mcp/execute_wave.go, internal/hook/session_start.go]
tech_stack:
  added: []
  patterns: [count-based-warm-classification, write-time-compaction, pure-function-tiering]
key_files:
  created:
    - internal/graph/tier.go
    - internal/graph/tier_test.go
  modified:
    - internal/graph/update.go
    - internal/graph/testdata/fake_bd/main.go
decisions:
  - "[09-01]: count-based warm (last N closed by ClosedAt desc) over time-based threshold — deterministic, project-lifecycle independent, immune to inactive project failure mode"
  - "[09-01]: classifyTier takes warmIDs map[string]bool (not time.Time) — avoids non-deterministic tests (Pitfall 4), caller controls warm set"
  - "[09-01]: tier types, pure functions, CompactBead, QueryTiered all in new tier.go — co-located with Bead type in graph package (Research Open Question 2)"
  - "[09-01]: CompactBead only called inside ClosePlan post-close — structural guard against compacting open beads (Pitfall 3)"
  - "[09-01]: extractCompactFromBead prefers Metadata['gsd:compact'] over compactSummary — uses write-time compacted value when available"
metrics:
  duration_seconds: 322
  completed: "2026-03-21"
  tasks_completed: 2
  files_changed: 4
  tests_added: 25
---

# Phase 9 Plan 01: Graph-Layer Tiering Summary

**One-liner:** Count-based hot/warm/cold tiering with write-time compaction on ClosePlan, implemented as pure functions in graph package.

## What Was Built

### Task 1: tier.go — Tier Types and Pure Functions

Created `internal/graph/tier.go` with the complete graph-layer tiering foundation:

- **Tier type** (string constant): `TierHot = "hot"`, `TierWarm = "warm"`, `TierCold = "cold"`
- **TieredBead struct**: embeds `Bead`, carries `Tier Tier` and `Compact string` (JSON: `tier`, `compact,omitempty`)
- **classifyTier(b Bead, warmIDs map[string]bool) Tier**: pure function, no I/O. open/in_progress=hot; warmIDs member=warm; otherwise cold
- **estimateTokens(s string) int**: `len(s)/4` with min 1 for non-empty strings. Single source of truth for budget math (D-07)
- **compactSummary(b *Bead) string**: `"title: close_reason"` or `"title"` if no CloseReason
- **formatHot/Warm/Cold**: tier-appropriate rendering functions for context inclusion
- **CompactBead(ctx, beadID, summary)**: writes `gsd:compact` to `Metadata` via UpdateBeadMetadata (Research Pattern 5)
- **QueryTiered(ctx, label, warmCount)**: queries label, sorts closed by ClosedAt desc, takes warmCount as warm set, classifies all, returns (hot, warm, cold []TieredBead, error)

### Task 2: ClosePlan Compaction

Modified `internal/graph/update.go`:
- Added `log/slog` import
- After bd close succeeds: calls `compactSummary(&closed[0])` then `c.CompactBead(ctx, beadID, summary)`
- Best-effort: compaction failure logs `slog.Warn` and returns close result anyway
- Structural guard: compaction only runs inside ClosePlan after close — never on open beads

### Fake BD Update

Updated `internal/graph/testdata/fake_bd/main.go` to support `FAKE_BD_QUERY_TIERED_RESPONSE` env var for QueryTiered tests — follows existing FAKE_BD_READY_RESPONSE pattern.

## Verification

- **Task 1 tests**: 25 new tests in `tier_test.go` — all pass
- **Full graph package**: 51 tests pass (was 26 pre-task)
- **Full suite**: `go test -race ./...` — 189 tests pass across 7 packages (was 164)
- **No regressions**: All existing ClosePlan, QueryByLabel, UpdateBeadMetadata tests still pass

## Deviations from Plan

None — plan executed exactly as written.

The plan specified count-based warm classification (Research Open Question 1 recommendation). Implementation used `warmIDs map[string]bool` parameter for classifyTier (not `time.Time` threshold) matching the research recommendation precisely.

## Known Stubs

None. All functions are fully implemented with real behavior.

## Self-Check: PASSED

Files exist:
- FOUND: internal/graph/tier.go
- FOUND: internal/graph/tier_test.go
- FOUND: internal/graph/update.go (modified)

Commits exist:
- d342bb5: test(09-01): add failing tests (RED)
- 2d23cc1: feat(09-01): create tier.go (GREEN)
- 3208d0f: feat(09-01): add compaction to ClosePlan
