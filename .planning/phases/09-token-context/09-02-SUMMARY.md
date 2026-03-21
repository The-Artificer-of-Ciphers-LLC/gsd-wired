---
phase: 09-token-context
plan: "02"
subsystem: hook, mcp
tags: [token-budget, session-start, tiered-context, mcp-tool, compaction]
dependency_graph:
  requires: [graph.QueryTiered, graph.TieredBead, graph.CompactBead, graph.EstimateTokens, graph.FormatHot, graph.FormatWarm, graph.FormatCold]
  provides: [buildBudgetContext, sessionStartDefaultBudget, get_tiered_context, handleGetTieredContext, extractCompact]
  affects: [internal/hook/session_start.go, internal/mcp/execute_wave.go, internal/mcp/tools.go, internal/mcp/server.go]
tech_stack:
  added: []
  patterns: [progressive-budget-degradation, metadata-preferred-fallback, thin-wrapper-pattern]
key_files:
  created:
    - internal/mcp/get_tiered_context.go
    - internal/mcp/get_tiered_context_test.go
  modified:
    - internal/hook/session_start.go
    - internal/hook/session_start_test.go
    - internal/graph/tier.go
    - internal/mcp/execute_wave.go
    - internal/mcp/execute_wave_test.go
    - internal/mcp/tools.go
    - internal/mcp/server.go
    - internal/mcp/tools_test.go
    - internal/mcp/server_test.go
decisions:
  - "[09-02]: buildSessionContext becomes thin wrapper (calls buildBudgetContext with 2000 token default) — existing call site in handleSessionStart unchanged"
  - "[09-02]: Exported EstimateTokens/FormatHot/FormatWarm/FormatCold added to tier.go as wrappers — hook package calls graph package without duplicating logic"
  - "[09-02]: filterByPhaseNum returns empty slice when no phase match found — caller handles empty gracefully, never silently returns all beads"
  - "[09-02]: extractCompact in execute_wave.go reads Metadata['gsd:compact'] first, falls back to CloseReason — backward compatible, existing beads still work"
  - "[09-02]: get_tiered_context defaults budget_tokens to 2000 when 0/omitted — same default as SessionStart for consistency"
metrics:
  duration_seconds: 314
  completed: "2026-03-21"
  tasks_completed: 2
  files_changed: 10
  tests_added: 7
---

# Phase 9 Plan 02: Budget-Aware Context + get_tiered_context + execute_wave Compaction Summary

**One-liner:** Budget-aware SessionStart with progressive degradation, new get_tiered_context MCP tool (tool 18), and execute_wave compacted dep summaries via gsd:compact metadata preference.

## What Was Built

### Task 1: Budget-Aware SessionStart + get_tiered_context MCP Tool

**Part A: Refactored `internal/hook/session_start.go`**

- Added `const sessionStartDefaultBudget = 2000` (per D-09)
- Refactored `buildSessionContext` as thin wrapper: calls `buildBudgetContext(ctx, c, 2000)`
- Created `buildBudgetContext(ctx, c *graph.Client, budget int) string`:
  - Calls `c.QueryTiered(ctx, "gsd:phase", 5)` to get hot/warm/cold phase beads
  - Progressive degradation loop per Research Pattern 4:
    - Hot beads: always included (never dropped per Pitfall 2)
    - Warm beads: included if budget allows, degrade to cold format if tight
    - Cold beads: included only if budget allows, omitted if over budget
  - Uses exported `graph.EstimateTokens`, `graph.FormatHot`, `graph.FormatWarm`, `graph.FormatCold`
  - Appends ready tasks section unconditionally (active work)

**Part B: New `internal/graph/tier.go` exported wrappers**

- Added `EstimateTokens`, `FormatHot`, `FormatWarm`, `FormatCold` — exported wrappers enabling hook and MCP packages to call graph functions without duplicating logic

**Part C: New `internal/mcp/get_tiered_context.go`**

- `tieredContextArgs { PhaseNum int, BudgetTokens int }`
- `tieredContextResult { Hot, Warm, Cold []graph.TieredBead, ContextString string, EstimatedTokens int }`
- `handleGetTieredContext`: initializes state, defaults budget to 2000 if 0, calls `QueryTiered`, filters by phase_num, builds budget-fitted context_string using same progressive degradation pattern

**Part D: Tool count updated to 18 atomically across all 4 files**

- `internal/mcp/tools.go`: registered `get_tiered_context` as tool 18, updated comment "17" → "18"
- `internal/mcp/server.go`: updated debug log count 17 → 18
- `internal/mcp/tools_test.go`: added `get_tiered_context` to wantNames, count 17 → 18
- `internal/mcp/server_test.go`: added `get_tiered_context` to wantNames, count 17 → 18

**New tests (7 total):**
- `TestBuildBudgetContext`: budget=2000, verifies non-empty context returned
- `TestBuildBudgetContextHotAlways`: budget=1 (tiny), verifies no panic — hot never dropped
- `TestSessionStartBudget`: buildSessionContext wrapper still works
- `TestGetTieredContext`: tool returns hot/warm/cold + context_string + estimated_tokens
- `TestGetTieredContextDefaultBudget`: omitting budget_tokens defaults to 2000
- `TestToolCountIs18`: exactly 18 tools registered
- `TestExecuteWaveCompaction`: compact value preferred over close_reason

### Task 2: execute_wave Compacted Dep Summaries

Modified `internal/mcp/execute_wave.go`:

- Added `graph` import
- Added `extractCompact(b *graph.Bead) string`:
  - Reads `b.Metadata["gsd:compact"]` first (write-time compacted value)
  - Falls back to `b.CloseReason` for beads not yet compacted
- Replaced dep resolution block: now calls `extractCompact(depBead)` instead of direct `depBead.CloseReason`

Backward compatible: existing closed beads without `gsd:compact` metadata continue to use CloseReason.

## Verification

- All hook and MCP tests pass: `go test ./internal/hook/... ./internal/mcp/... -v -count=1`
- Full suite green with race detector: `go test -race ./...` — all 7 packages pass
- Tool count is 18: `grep -c 'server.AddTool' internal/mcp/tools.go` returns 18
- All 8 existing SessionStart tests pass without modification
- All 4 existing execute_wave tests pass without modification

## Deviations from Plan

None — plan executed exactly as written.

## Known Stubs

None. All functions are fully implemented with real behavior.

## Self-Check: PASSED

Files exist:
- FOUND: internal/hook/session_start.go (modified)
- FOUND: internal/mcp/get_tiered_context.go (created)
- FOUND: internal/mcp/get_tiered_context_test.go (created)
- FOUND: internal/mcp/execute_wave.go (modified)

Commits exist:
- cb150a5: feat(09-02): budget-aware SessionStart + get_tiered_context MCP tool (tool 18)
- f6d3511: feat(09-02): execute_wave uses compacted dep summaries (gsd:compact preferred)
