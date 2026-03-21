# Phase 9: Token-Aware Context - Context

**Gathered:** 2026-03-21
**Status:** Ready for planning
**Source:** Auto-mode (Claude selected recommended defaults)

<domain>
## Phase Boundary

Minimize token consumption through intelligent context routing based on bead state. Hot/warm/cold tiering, budget tracking, context injection. Delivers TOKEN-01 through TOKEN-06.

</domain>

<decisions>
## Implementation Decisions

### Design principle
- **D-01:** gsdw is the developer interface. Token optimization is invisible — developer gets the right context without knowing about budgets or tiers.
- **D-02:** This is the core innovation: graph queries replace full file reads, context is O(relevant) not O(total).

### Hot/warm/cold tiering
- **D-03:** Active beads (open, in_progress) = hot → full context (description, acceptance, notes, metadata).
- **D-04:** Recently closed beads = warm → summary only (title + close reason, ~50 tokens).
- **D-05:** Old closed beads = cold → ID + title only (~10 tokens).
- **D-06:** "Recently closed" threshold at Claude's discretion (e.g., last 24 hours, last N beads).

### Token budget tracking
- **D-07:** Simple byte-count heuristic: 1 token ≈ 4 bytes. No external tokenizer dependency.
- **D-08:** Budget-aware context fitting: measure total tiered content, trim warm→cold or omit cold beads to fit within budget.

### Context injection
- **D-09:** SessionStart's `additionalContext` becomes budget-aware. Check available budget, inject tiered content that fits.
- **D-10:** New `get_tiered_context` MCP tool returns context at requested tier level with budget constraint. Used by hooks and skills.

### Subagent context optimization
- **D-11:** `execute_wave` context chains use compacted summaries for closed dependency beads (warm tier) instead of full content.
- **D-12:** Closed beads are automatically compacted (summary replaces full content in query results). This is a graph-layer optimization, not a tool-layer one.

### Claude's Discretion
- Exact tiering thresholds (hot/warm/cold boundaries)
- Budget estimation accuracy (byte heuristic vs more sophisticated)
- How to handle budget overflow (progressive degradation strategy)
- Compaction trigger (on close, on query, on timer)
- Whether to modify existing tools or create new ones

</decisions>

<specifics>
## Specific Ideas

- SessionStart already builds context via `buildSessionContext` (Phase 4) — Phase 9 makes it budget-aware
- `execute_wave` already pre-computes context chains (Phase 7) — Phase 9 adds compaction to those chains
- The byte heuristic (1 token ≈ 4 bytes) is deliberately simple — it's a budget guide, not a tokenizer
- TOKEN-06 (tiered injection in SessionStart) is the most impactful: it determines cold-start quality

</specifics>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase 4, 7 foundation
- `internal/hook/session_start.go` — buildSessionContext (make budget-aware)
- `internal/mcp/execute_wave.go` — Context chain pre-computation (add compaction)
- `internal/graph/query.go` — GetBead, QueryByLabel, ListReady (add tiered returns)
- `internal/graph/bead.go` — Bead struct (add summary/compacted fields?)

### Project context
- `.planning/PROJECT.md` — Core value, constraints
- `.planning/REQUIREMENTS.md` — TOKEN-01 through TOKEN-06
- `.planning/ROADMAP.md` §Phase 9 — Success criteria (5 items)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/hook/session_start.go` — Context injection via additionalContext
- `internal/mcp/execute_wave.go` — Pre-computed context chains for execution agents
- `internal/graph/query.go` — All graph queries (add tiering here)

### Established Patterns
- MCP tools for machine operations, SKILL.md for orchestration
- Best-effort with graceful degradation
- No external dependencies (stdlib only)

### Integration Points
- `internal/hook/session_start.go` — Budget-aware injection
- `internal/mcp/execute_wave.go` — Compacted dependency summaries
- `internal/graph/query.go` — Tiered query results
- `internal/mcp/tools.go` — New tool: get_tiered_context

</code_context>

<deferred>
## Deferred Ideas

- PreToolUse file-aware context injection (TOKEN-A01, v2)
- Automatic dependency detection suggestions (TOKEN-A02, v2)
- TUI visualization of token usage (PLAT-01, v2)

</deferred>

---

*Phase: 09-token-context*
*Context gathered: 2026-03-21 via auto-mode*
