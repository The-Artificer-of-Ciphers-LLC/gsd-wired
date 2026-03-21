# Phase 8: Ship + Status - Context

**Gathered:** 2026-03-21
**Status:** Ready for planning
**Source:** Auto-mode (Claude selected recommended defaults)

<domain>
## Phase Boundary

PR creation with bead-sourced summaries, phase completion with state advancement, and enriched project status. Delivers SHIP-01, SHIP-02, CMD-02, CMD-06.

</domain>

<decisions>
## Implementation Decisions

### Design principle
- **D-01:** gsdw is the developer interface. Ship and status are developer-facing commands — clear output matters.
- **D-02:** 30-second auto-proceed pattern continues throughout.

### PR creation (/gsd-wired:ship)
- **D-03:** SKILL.md instructs Claude to use `gh pr create` (GitHub CLI) for PR creation. No custom git integration — use the standard tool.
- **D-04:** New `create_pr_summary` MCP tool generates bead-sourced PR summary: requirements covered, phases completed, key changes. Returns structured data for SKILL.md to format.
- **D-05:** PR summary uses GSD-familiar terms. Developer sees requirements and phases, not bead IDs.

### Phase advancement
- **D-06:** Phase completion closes the phase epic bead, updates metadata with completion date. Reuses existing `close_plan` pattern (closing an epic is the same operation).
- **D-07:** After phase completion, `list_ready` surfaces next phase's unblocked work automatically. SKILL.md auto-proceeds.

### Status enrichment
- **D-08:** `get_status` tool (Phase 5) already returns project state. Phase 8 enriches it with ship-specific data: PR links, completion dates, phase history. Backward compatible — same tool, richer output.
- **D-09:** CMD-02 (`/gsd-wired:status`) already works from Phase 5. No new skill needed — just richer data in the existing tool response.

### Claude's Discretion
- PR summary format and content depth
- Phase advancement ceremony (what gets logged, what gets displayed)
- Status enrichment fields (which ship-specific data to add)
- `gh` CLI integration details (flags, branch naming)
- How to handle ship when no changes to ship (empty PR)

</decisions>

<specifics>
## Specific Ideas

- This is the smallest phase by requirement count (4 requirements: SHIP-01, SHIP-02, CMD-02, CMD-06)
- CMD-02 (/gsd-wired:status) is already partially implemented from Phase 5 — just needs enrichment
- The ship flow is: verify passes → create PR → close phase epic → advance state → auto-proceed
- `gh pr create` is the developer's familiar tool — no reason to build custom git integration

</specifics>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase 5-7 foundation
- `internal/mcp/get_status.go` — Existing get_status tool (enrich, don't replace)
- `internal/mcp/verify_phase.go` — verify_phase returns pass/fail (ship happens after pass)
- `internal/mcp/tools.go` — 15 MCP tools (ship adds 1-2 more)
- `internal/graph/update.go` — ClosePlan (reuse for closing phase epics)
- `internal/graph/query.go` — QueryByLabel, ListReady
- `skills/status/SKILL.md` — Existing status skill (may not need changes)
- `skills/verify/SKILL.md` — Verify flow that feeds into ship

### Project context
- `.planning/PROJECT.md` — Core value, constraints
- `.planning/REQUIREMENTS.md` — SHIP-01, SHIP-02, CMD-02, CMD-06
- `.planning/ROADMAP.md` §Phase 8 — Success criteria (3 items)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/mcp/get_status.go` — Status tool to enrich
- `internal/graph/update.go` — ClosePlan works for epics too
- `skills/verify/SKILL.md` — Verify → ship flow pattern

### Established Patterns
- SKILL.md → MCP tool → 30s auto-proceed
- GSD-familiar output, no bead terminology
- Tool count increments atomically across 4 files

### Integration Points
- `internal/mcp/tools.go` — New tool(s): create_pr_summary, advance_phase
- `skills/` directory — New: skills/ship/SKILL.md
- `internal/cli/root.go` — New: ship subcommand

</code_context>

<deferred>
## Deferred Ideas

- Token-aware PR summaries (Phase 9)
- Milestone completion ceremony (future — not v1)
- Automated PR review integration (future)

</deferred>

---

*Phase: 08-ship-status*
*Context gathered: 2026-03-21 via auto-mode*
