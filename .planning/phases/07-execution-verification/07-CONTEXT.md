# Phase 7: Execution + Verification - Context

**Gathered:** 2026-03-21
**Status:** Ready for planning
**Source:** Auto-mode (Claude selected recommended defaults)

<domain>
## Phase Boundary

Wave-based parallel execution where agents claim unblocked beads, work, and close them. Post-execution verification against success criteria with automatic remediation task creation. Delivers EXEC-01 through EXEC-06, VRFY-01 through VRFY-03, CMD-04, CMD-05, CMD-07.

</domain>

<decisions>
## Implementation Decisions

### Design principle
- **D-01:** Same as all phases — gsdw is the developer interface. Execution orchestration, wave management, and verification happen behind the scenes. Developer says "execute" and work gets done.
- **D-02:** 30-second auto-proceed pattern continues. After verification passes, auto-proceed to next phase.

### Wave execution
- **D-03:** SKILL.md orchestrates execution. Calls `list_ready` for current wave, spawns parallel Task() agents per ready task, each agent claims its bead, works, closes. `list_ready` again for next wave. Repeat until empty. Same pattern as research orchestration.
- **D-04:** Each execution agent receives only its claimed bead's context chain: task description, success criteria, parent epic summary, dependency bead summaries. Minimal prompt per EXEC-02/EXEC-03.
- **D-05:** On task completion, agent closes bead with results. `list_ready` surfaces newly unblocked tasks, triggering next wave automatically per EXEC-04.
- **D-06:** Atomic git commits per completed task. Commit message uses GSD-friendly plan ID (e.g., "feat(07-01): description"), not bd bead ID. Developer never sees bd IDs.

### Agent output validation
- **D-07:** SKILL.md validates results inline between waves — checks must-haves from task bead's acceptance criteria before proceeding. Best-effort validation, same pattern as plan checker. No separate validation agent.
- **D-08:** Validation errors are surfaced to developer for decision (retry, skip, abort). Same escalation pattern as plan checker.

### Post-execution verification
- **D-09:** New `verify_phase` MCP tool reads phase epic's success criteria, checks against codebase state, returns structured pass/fail per criterion.
- **D-10:** Failed verification criteria automatically produce new remediation task beads via existing `create_plan_beads` tool (VRFY-03).
- **D-11:** `/gsd-wired:verify` SKILL.md presents verification results in GSD-familiar format (pass/fail table, not bead data).
- **D-12:** `/gsd-wired:ready` SKILL.md shows unblocked tasks (CMD-07). Reuses existing `gsdw ready` CLI command + `list_ready` MCP tool.

### Claude's Discretion
- Exact subagent prompt structure for execution agents
- How to extract task context chain (parent epic + dependency summaries) efficiently
- Agent output validation depth (acceptance criteria check vs full code review)
- Verification implementation (code checks, test execution, file existence)
- Remediation task granularity (one per failed criterion vs grouped)

</decisions>

<specifics>
## Specific Ideas

- Execution is the largest requirement set (EXEC-01 through EXEC-06 + VRFY-01 through VRFY-03 + 3 commands = 12 requirements)
- The wave execution pattern is identical to research orchestration from Phase 6 — just applied to execution tasks instead of research topics
- `verify_phase` is the first tool that inspects actual codebase state (file existence, test results) — all prior tools only inspect bead state
- The remediation loop (verify → find gaps → create tasks → execute → verify again) is GSD's gap closure cycle implemented in beads

</specifics>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase 2-6 foundation
- `internal/mcp/tools.go` — 13 MCP tools (claim_bead, close_plan, list_ready, create_plan_beads, etc.)
- `internal/mcp/run_research.go` — Pattern for orchestrating parallel agents via epic + child beads
- `internal/mcp/create_plan_beads.go` — Topological dependency resolution (reusable for remediation tasks)
- `internal/graph/query.go` — ListReady, GetBead, QueryByLabel
- `internal/graph/update.go` — ClaimBead, ClosePlan
- `skills/research/SKILL.md` — Pattern for parallel Task() agent spawning
- `skills/plan/SKILL.md` — Pattern for inline validation with iteration limit
- `internal/cli/ready.go` — gsdw ready command (CMD-07 already partially implemented)

### Project context
- `.planning/PROJECT.md` — Core value, constraints, key decisions
- `.planning/REQUIREMENTS.md` — EXEC-01 through EXEC-06, VRFY-01 through VRFY-03, CMD-04, CMD-05, CMD-07
- `.planning/ROADMAP.md` §Phase 7 — Success criteria (7 items that must be TRUE)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `skills/research/SKILL.md` — Parallel Task() spawning pattern. Execution uses identical pattern with different bead types.
- `skills/plan/SKILL.md` — Inline validation loop with 3-iteration limit. Verification reuses this pattern.
- `internal/mcp/create_plan_beads.go` — Topological dependency resolution. Remediation tasks use same tool.
- `internal/cli/ready.go` — `gsdw ready` already shows unblocked tasks. CMD-07 is partially done.

### Established Patterns
- SKILL.md orchestrates → MCP tools execute → 30s auto-proceed
- `claim_bead` + work + `close_plan` lifecycle
- `list_ready` for wave boundary detection
- GSD-familiar output (wave tables, progress bars, plan IDs)
- Best-effort validation with developer escalation

### Integration Points
- `internal/mcp/tools.go` — New tools: verify_phase, execute_task (or reuse claim_bead + close_plan)
- `skills/` directory — New: execute/SKILL.md, verify/SKILL.md, ready/SKILL.md
- `internal/cli/root.go` — New: execute, verify subcommands

</code_context>

<deferred>
## Deferred Ideas

- PR creation from execution results (Phase 8)
- Token-aware context loading for execution agents (Phase 9)
- Cross-phase regression testing (future — not in v1 requirements)

</deferred>

---

*Phase: 07-execution-verification*
*Context gathered: 2026-03-21 via auto-mode*
