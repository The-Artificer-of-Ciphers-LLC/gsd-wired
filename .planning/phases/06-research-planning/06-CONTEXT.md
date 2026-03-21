# Phase 6: Research + Planning - Context

**Gathered:** 2026-03-21
**Status:** Ready for planning

<domain>
## Phase Boundary

Users can run research phases and create dependency-aware plans, all coordinated through beads. Research agents claim child beads, synthesizer produces summary. Plan decomposes phase into task beads with dependencies. Plan checker validates coverage. Delivers RSRCH-01 through RSRCH-04, PLAN-01 through PLAN-04, CMD-03.

</domain>

<decisions>
## Implementation Decisions

### Design principle
- **D-01:** Developer speaks plainly. They say "research this" or "plan this" and gsdw handles all orchestration. Agent coordination, bead claiming, synthesizer triggering — all invisible.
- **D-02:** 30-second auto-proceed is the standard pattern. After plan approval, pause 30s for developer reaction, then flow into execution. Developer just wants it done — we're GSD'ing.

### Research agent coordination
- **D-03:** SKILL.md orchestrates research. Natural language instructions tell Claude to spawn 4 agents, each claims a bead, writes results. No programmatic Go orchestration.
- **D-04:** Synthesizer coordination at Claude's discretion — optimize for performance and reliability. Developer doesn't care how it knows all 4 are done.
- **D-05:** Result storage at Claude's discretion — beads, files, or both. Developer just wants the research result available for planning.
- **D-06:** Fixed 4 research topics matching GSD: stack, features, architecture, pitfalls. Not configurable.

### Plan decomposition
- **D-07:** Auto-generate full plan from research + context. No interactive task-by-task questioning. Same as GSD.
- **D-08:** Developer sees familiar GSD-style plan output — wave structure, task list with objectives, dependency graph. Not raw bead data.
- **D-09:** 30-second auto-proceed after plan display. Developer can interrupt to review or modify.

### Plan checker
- **D-10:** Plan checker implementation at Claude's discretion — SKILL.md instruction, separate agent, or MCP tool. Optimize for performance/reliability.
- **D-11:** Same escalation as GSD: iterate up to 3 times. After 3 failures, ask developer (force proceed, provide guidance, abandon).
- **D-12:** Requirement coverage gate required — every phase requirement ID must appear in at least one task bead. Same as GSD's coverage check.
- **D-13:** After plan approved, 30-second auto-proceed then auto-flow into execution.

### Claude's Discretion
- Research agent spawning mechanics (Task tool calls from SKILL.md)
- Synthesizer trigger mechanism (poll, event, or instruction-based)
- Result storage format (bead content, bead metadata, files, or combination)
- Plan checker implementation approach
- Plan display format (exact layout of wave/task/dependency output)
- How to show GSD-familiar output while data lives in beads

</decisions>

<specifics>
## Specific Ideas

- This is the core innovation: AI agents coordinating through a graph. Research agents don't pass context to each other directly — they write to beads, and the synthesizer reads from beads. Minimal prompt overhead.
- The 30-second auto-proceed pattern is now established across init (D-08 Phase 5), plan review (D-09), and plan-to-execution (D-13). Consistent UX.
- Plan checker validates the same things GSD's gsd-plan-checker does: goal alignment, requirement coverage, task quality, wave ordering, research alignment.
- `/gsd-wired:plan` is the second user-facing workflow command after `/gsd-wired:init`.

</specifics>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase 1-5 foundation
- `internal/mcp/tools.go` — 10 MCP tools (create_phase, create_plan, claim_bead, close_plan, etc.)
- `internal/mcp/init_project.go` — Pattern for creating epic + child beads (reusable for research epic)
- `internal/graph/create.go` — CreatePhase, CreatePlan with labels and metadata
- `internal/graph/update.go` — ClaimBead, ClosePlan, AddLabel, UpdateBeadMetadata
- `internal/graph/query.go` — ListReady, GetBead, QueryByLabel
- `skills/init/SKILL.md` — Pattern for slash command skills with MCP tool calls
- `skills/status/SKILL.md` — Pattern for display-oriented skills

### Project context
- `.planning/PROJECT.md` — Core value, constraints, key decisions
- `.planning/REQUIREMENTS.md` — RSRCH-01 through RSRCH-04, PLAN-01 through PLAN-04, CMD-03
- `.planning/ROADMAP.md` §Phase 6 — Success criteria (6 items that must be TRUE)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/mcp/init_project.go` — `handleInitProject` creates epic + children pattern. Research uses the same structure (epic + 4 researcher children + synthesizer).
- `internal/mcp/tools.go` — Tool registration with `server.AddTool`. New research/plan tools added here.
- `skills/init/SKILL.md` — 12-question flow with 3 modes. Research/plan skills follow same pattern.
- `internal/cli/ready.go` — Tree-formatted output. Plan display can reuse this for wave visualization.

### Established Patterns
- SKILL.md orchestrates, MCP tools execute (Phase 5 pattern)
- 30-second auto-proceed (Phase 5 init, now standard)
- Category beads as children of epic (init_project pattern)
- `claim_bead` + work + `close_plan` lifecycle (Phase 2)
- `bd ready` for wave computation (Phase 2)

### Integration Points
- `internal/mcp/tools.go` — New tools: run_research, create_plan_beads, validate_plan, get_plan_status
- `skills/` directory — New: `skills/research/SKILL.md`, `skills/plan/SKILL.md`
- `.claude-plugin/plugin.json` — No changes needed (skills auto-discover)

</code_context>

<deferred>
## Deferred Ideas

- Execution of plans (Phase 7)
- PR creation from plan results (Phase 8)
- Token-aware research context loading (Phase 9)
- Direct Go import of beads library for performance (Phase 6 optimization path — deferred to v2)

</deferred>

---

*Phase: 06-research-planning*
*Context gathered: 2026-03-21*
