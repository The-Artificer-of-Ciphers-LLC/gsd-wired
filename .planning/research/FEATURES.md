# Feature Research

**Domain:** AI agent orchestration plugin (Claude Code + Beads/Dolt graph persistence)
**Researched:** 2026-03-21
**Confidence:** MEDIUM-HIGH (strong understanding of both GSD and Beads ecosystems; Claude Code plugin model well-documented; some features extrapolated from adjacent systems)

## Feature Landscape

### Table Stakes (Users Expect These)

Features GSD users assume exist. Missing these = product feels incomplete compared to current GSD.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Project initialization via guided questioning | GSD's `/gsd:new-project` flow is the entry point; users expect structured onboarding that captures project context | MEDIUM | Must produce beads (epic + context beads) instead of markdown files. Deep questioning flow stays similar to current GSD. |
| Phase-based workflow lifecycle (init, research, plan, execute, verify, ship) | Core GSD mental model. Users think in phases. Removing this breaks the workflow discipline that attracted them. | HIGH | Phase = epic bead. Each lifecycle stage maps to a bead state or tag. Must feel identical to current GSD from the user's perspective. |
| Subagent spawning with context isolation | Claude Code's native subagent model already does this. GSD users expect parallel research agents, execution agents, verification agents. | MEDIUM | Beads integration means subagents `bd update --claim` a bead and get only that bead's context. Core improvement over current GSD where subagents get fat markdown prompts. |
| Wave-based parallel execution | GSD's wave execution is a defining feature: dependency-aware layers where unblocked tasks run in parallel. Users expect `bd ready` to surface the current wave. | HIGH | Maps directly to Beads' dependency graph. `bd ready` returns unblocked tasks. Wave = all beads with no uncompleted dependencies at a given point. |
| Plan creation and task breakdown | `/gsd:plan-phase` decomposes phases into concrete plans with success criteria. Users expect structured planning that produces actionable tasks. | MEDIUM | Plans become task beads with parent-child relationships to phase epic beads. Success criteria stored as extensible fields on the bead. |
| Post-execution verification | GSD verifies work against success criteria stored in plans. Users expect automated verification that checks if what was built matches what was specified. | MEDIUM | Verification agent reads success criteria from bead's extensible fields, runs checks, updates bead status. Familiar GSD flow. |
| SessionStart context loading | When a user opens Claude Code in a project, they expect the plugin to load current project state automatically — where we are, what's active, what's next. | LOW | SessionStart hook queries beads graph for active/ready beads and injects lean summary into context. Replaces reading STATE.md + ROADMAP.md. |
| PreCompact state preservation | Before Claude compacts context, in-progress work must be saved. Users who've lost context mid-task will never forgive this being missing. | LOW | PreCompact hook writes current work state to beads before compaction happens. Critical for long-running sessions. |
| CLI-driven workflow (slash commands) | GSD users work via `/gsd:*` commands. The command-driven UX is table stakes. | LOW | Slash commands become plugin skills that call MCP server endpoints wrapping `bd` operations. Same UX, different backend. |
| Coexistence with .planning/ directory | Existing GSD users have projects with .planning/ state. They need a migration path, not a cliff. | MEDIUM | Read .planning/ as fallback when beads not initialized. Don't delete or modify existing .planning/ files. |
| Git integration (atomic commits, PR creation) | GSD produces atomic commits per task and creates PRs. Users expect this to keep working. | LOW | Unchanged from current GSD — git operations are independent of the persistence layer. Bead state tracks commit hashes as metadata. |

### Differentiators (Competitive Advantage)

Features that make gsd-wired superior to both current GSD and other agent orchestration systems.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Token-aware context routing | The wire decides what context to load based on token budget. Hot beads get full context, warm beads get summaries, cold beads get IDs only. No other orchestration system does budget-aware context selection. | HIGH | This is the core innovation. Current GSD loads full markdown files regardless of relevance. LangGraph/CrewAI don't address token budgets at all. Requires estimating token costs per bead and fitting within remaining budget. |
| Graph-backed state instead of markdown bloat | ROADMAP.md and STATE.md grow linearly with project complexity. Beads graph queries return only relevant state. A 50-phase project loads as fast as a 5-phase project. | MEDIUM | Replaces the #1 pain point of current GSD. Graph queries (`bd ready`, `bd show`) are O(relevant) not O(total). Dolt's SQL interface makes complex queries trivial. |
| Bead-scoped subagent context | Subagents claim a specific bead and receive only that bead's context + its dependency chain — not the entire project state. Dramatically reduces prompt size for execution agents. | MEDIUM | Current GSD subagents get full plan markdown + project context. With beads, a subagent claiming task bead `a3f2` gets: that bead's description, success criteria, parent epic summary, and relevant dependency beads. Nothing else. |
| Automatic compaction of closed work | Beads' `bd admin compact` removes old closed issues. Unlike markdown files that accumulate forever, the graph naturally shrinks as work completes. Closed beads get summarized, then eventually pruned. | LOW | Dolt's versioning means you can always go back. Compaction is safe because history is preserved in Dolt's commit log. Current GSD has no compaction story at all. |
| PreToolCall context injection | Before any tool call, the wire can inject relevant bead context. Agent about to edit `auth.ts`? Inject the bead that describes the auth feature's requirements and constraints. | HIGH | No other system intercepts tool calls to add just-in-time context. This is speculative — depends on PreToolCall hook being able to modify the context efficiently without adding latency. |
| Tiered context management (hot/warm/cold) | Active beads = full context. Recently closed = summary. Old closed = ID + one-liner. The wire manages this tiering automatically based on bead state and recency. | HIGH | This is how the system stays lean at scale. A project with 200 completed beads and 5 active beads loads context for 5, summaries for maybe 10 recent, and ignores the rest. |
| Versioned state via Dolt | Every state change is a Dolt commit. You can branch, diff, and roll back project state. Made a bad planning decision? `dolt checkout` to revert. | LOW | Comes free from Dolt. No other orchestration system offers git-for-state. LangGraph has checkpointing but not branch/merge semantics. |
| Dependency-aware task surfacing | `bd ready` returns only tasks whose dependencies are met. No manual wave tracking. No stale "next task" lists. The graph always knows what's unblocked. | LOW | Comes from Beads. GSD currently infers waves from plan ordering. Graph-based dependency resolution is strictly better — handles cross-phase dependencies, dynamic re-ordering, and parallel execution paths. |
| Research agent coordination via beads | 4 parallel research agents each claim a child bead of the research epic. Synthesizer agent queries all children when they're closed. No shared context, no collision, deterministic merging. | MEDIUM | Current GSD research spawns 4 agents with overlapping concerns. Beads-backed research has clean boundaries: each researcher owns a bead, writes findings to it, closes it. Synthesizer reads all 4. No context overlap. |

### Anti-Features (Commonly Requested, Often Problematic)

Features that seem good but would undermine the system's core value proposition.

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| Web UI / dashboard for project state | Users want visual project tracking. Sounds reasonable. | Adds massive scope. The target user is a CLI-native Claude Code user. A web UI is a separate product. Also contradicts "lean context" philosophy — UI implies API server, auth, frontend framework. | `bd list --json` piped to a lightweight TUI (terminal UI) as a v2+ add-on. Or `bd list --format table` for pretty terminal output. |
| Multi-model support (GPT, Gemini, etc.) | "Why Claude-only?" | Claude Code's hook system is the integration surface. Other agents don't have PreToolCall, PostToolCall, SessionStart, PreCompact hooks. Supporting other platforms means building N different integrations with different capabilities. | v1 is Claude Code only. If demand exists, v2 could add OpenCode (which has GSD support already via fork). |
| Real-time collaboration between human developers | "Multiple devs working on same beads graph simultaneously" | Dolt supports multi-writer, but the orchestration logic assumes single-orchestrator. Multi-human workflows need conflict resolution UI, presence indicators, locking — all massive scope. | Dolt's branch-and-merge handles async collaboration. Each developer works on their branch, merges via Dolt. No real-time needed. |
| Auto-planning / AI-generated roadmaps without human input | "Just give it a description and let it plan everything" | Removes the human-in-the-loop that makes GSD reliable. Auto-generated plans without human review produce garbage on complex projects. GSD's value is structured human-AI collaboration, not full autonomy. | Keep the current deep-questioning flow for project init. AI proposes, human approves. The guided planning flow is a feature, not a limitation. |
| Plugin marketplace / extensibility API | "Let people build plugins on top of gsd-wired" | Premature abstraction. The core system isn't built yet. Plugin APIs constrain internal architecture before it's stable. | Expose `bd` CLI as the extension point. Anyone can write scripts that call `bd`. Formalize plugin API only after v1 stabilizes. |
| Remote Dolt sync / cloud hosting | "I want my beads graph in the cloud" | Adds hosting infrastructure, auth, billing, latency. Local-only is a feature for v1 — fast, private, no network dependency. | Dolt has built-in remotes (like git). Users who want sync can `dolt push` to DoltHub. No custom infrastructure needed. |
| Automatic dependency detection | "AI should figure out task dependencies automatically" | LLMs are unreliable at inferring dependencies. Hallucinated dependencies create phantom blockers. Wrong dependency ordering breaks wave execution. | Human specifies dependencies during planning. AI can suggest, but human confirms. `bd create --blocks <id>` is explicit and reliable. |
| Full migration tooling from .planning/ to beads | "Import my existing GSD project into beads" | Every .planning/ project has different state shapes. Migration tooling is fragile, hard to test, and only used once per project. | Coexistence mode: read .planning/ as fallback. New work goes to beads. Old work stays in markdown. Natural migration over time. |

## Feature Dependencies

```
[SessionStart Hook]
    └──requires──> [Beads Graph Init (bd init)]
                       └──requires──> [Dolt + bd installed]

[Wave Execution]
    └──requires──> [Dependency Graph (bd ready)]
                       └──requires──> [Plan Creation (task beads with dependencies)]
                                          └──requires──> [Phase Init (epic beads)]

[Token-Aware Routing]
    └──requires──> [Tiered Context (hot/warm/cold)]
                       └──requires──> [Bead State Tracking]
                                          └──requires──> [PostToolCall Hook (updates bead state)]

[PreCompact State Save]
    └──requires──> [Bead State Tracking]

[Bead-Scoped Subagent Context]
    └──requires──> [Beads Graph Init]
    └──enhances──> [Wave Execution]
    └──enhances──> [Research Coordination]

[Research Coordination]
    └──requires──> [Subagent Spawning]
    └──requires──> [Epic/Child Bead Structure]

[Coexistence (.planning/ fallback)]
    └──conflicts──> [Full Migration Tooling] (pick one strategy)

[Verification]
    └──requires──> [Success Criteria in Beads]
                       └──requires──> [Plan Creation]

[Git Integration]
    └──enhances──> [Bead State Tracking] (commit hashes as metadata)
    └──independent of──> [Beads Graph]
```

### Dependency Notes

- **Wave Execution requires Dependency Graph:** Waves are derived from the beads dependency structure. Without `bd ready` returning unblocked tasks, wave execution has no data source.
- **Token-Aware Routing requires Tiered Context:** The wire can't route context without a tiering system that categorizes beads by temperature (hot/warm/cold).
- **Bead-Scoped Subagent Context enhances Wave Execution:** Subagents that claim beads naturally participate in wave execution — they pick up unblocked beads and work them.
- **Coexistence conflicts with Full Migration:** These are opposing strategies. Coexistence says "live with both." Migration says "convert everything." Pick coexistence for v1.
- **SessionStart Hook requires Beads Graph:** Can't load project state from beads if beads isn't initialized. Falls back to .planning/ in coexistence mode.

## MVP Definition

### Launch With (v1)

Minimum viable: a GSD user can init a project, plan a phase, execute tasks via waves, and verify results — all backed by beads instead of markdown.

- [ ] **MCP server wrapping bd CLI** — The foundation. Every other feature calls bd through this.
- [ ] **Project initialization flow** — Deep questioning producing epic + context beads. Entry point for new projects.
- [ ] **Phase creation as epic beads** — Phases map to epics with GSD metadata (phase number, type, status).
- [ ] **Plan creation as task beads with dependencies** — Plans decompose into task beads with parent-child + blocks relationships.
- [ ] **SessionStart hook** — Loads active/ready beads into context on session start. Replaces STATE.md reading.
- [ ] **PreCompact hook** — Saves in-progress state before compaction. Prevents context loss.
- [ ] **Wave execution via bd ready** — Surface unblocked tasks, claim beads, execute, close. Core execution loop.
- [ ] **Subagent spawning with bead-scoped context** — Subagents claim beads and get lean, relevant context only.
- [ ] **Post-execution verification** — Verify agent reads success criteria from bead, checks work, updates status.
- [ ] **Coexistence with .planning/** — Read .planning/ as fallback. Zero disruption for existing GSD users.
- [ ] **Slash commands (skills)** — /gsd:new-project, /gsd:plan-phase, /gsd:execute-phase, /gsd:verify-phase at minimum.

### Add After Validation (v1.x)

Features to add once the core loop (init-plan-execute-verify) is proven.

- [ ] **Token-aware context routing** — Trigger: when users report context bloat on large projects (10+ phases). The tiered hot/warm/cold system.
- [ ] **PreToolCall context injection** — Trigger: when execution agents make mistakes due to missing context. Just-in-time bead context before tool calls.
- [ ] **Research agent coordination via beads** — Trigger: when parallel research is needed. 4-agent research pattern backed by epic + child beads.
- [ ] **Automatic compaction scheduling** — Trigger: when bead databases grow large. Auto-compact closed beads older than N days.
- [ ] **PostToolCall bead state updates** — Trigger: when manual status updates feel tedious. Auto-close beads when their success criteria are met.
- [ ] **PR creation with bead metadata** — Trigger: when shipping. PR descriptions auto-generated from completed beads.

### Future Consideration (v2+)

Features to defer until the core is stable and adopted.

- [ ] **Multi-project bead orchestration** — Defer: cross-project dependencies add complexity; single-project must work flawlessly first.
- [ ] **TUI (terminal UI) for project visualization** — Defer: nice-to-have; CLI output is sufficient for v1 users.
- [ ] **OpenCode / non-Claude-Code support** — Defer: different hook models require separate integration work.
- [ ] **Dolt remote sync (DoltHub push/pull)** — Defer: local-only is simpler and faster; sync adds failure modes.
- [ ] **Plugin/extensibility API** — Defer: internal APIs must stabilize before external consumers depend on them.
- [ ] **Beads formula templates** — Defer: Beads' formula system for declarative workflow templates is powerful but adds learning curve.

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| MCP server wrapping bd CLI | HIGH | MEDIUM | P1 |
| Project initialization flow | HIGH | MEDIUM | P1 |
| Phase creation (epic beads) | HIGH | LOW | P1 |
| Plan creation (task beads + deps) | HIGH | MEDIUM | P1 |
| SessionStart hook | HIGH | LOW | P1 |
| PreCompact hook | HIGH | LOW | P1 |
| Wave execution (bd ready) | HIGH | MEDIUM | P1 |
| Subagent bead-scoped context | HIGH | MEDIUM | P1 |
| Verification against bead criteria | MEDIUM | MEDIUM | P1 |
| Coexistence with .planning/ | MEDIUM | LOW | P1 |
| Slash commands (skills) | HIGH | LOW | P1 |
| Token-aware context routing | HIGH | HIGH | P2 |
| PreToolCall context injection | MEDIUM | HIGH | P2 |
| Research agent coordination | MEDIUM | MEDIUM | P2 |
| Auto-compaction scheduling | LOW | LOW | P2 |
| PostToolCall auto-state updates | MEDIUM | MEDIUM | P2 |
| PR creation with bead metadata | MEDIUM | LOW | P2 |
| Multi-project orchestration | LOW | HIGH | P3 |
| TUI visualization | LOW | MEDIUM | P3 |
| Non-Claude-Code support | LOW | HIGH | P3 |
| Dolt remote sync | LOW | MEDIUM | P3 |

**Priority key:**
- P1: Must have for launch — the core init-plan-execute-verify loop
- P2: Should have, add when core is proven — token optimization and polish
- P3: Nice to have, future consideration — expansion and ecosystem

## Competitor Feature Analysis

| Feature | GSD (current) | Beads (standalone) | LangGraph | CrewAI | gsd-wired (our plan) |
|---------|---------------|-------------------|-----------|--------|---------------------|
| Structured workflow lifecycle | Full (phases, plans, waves) | None (raw issue tracker) | Custom (build your own) | Partial (sequential/goal tasks) | Full (GSD lifecycle on beads graph) |
| Context isolation for subagents | Yes (fresh 200K windows) | N/A | Yes (node-scoped state) | Partial (conversation history) | Yes + bead-scoped (only relevant context) |
| State persistence | Markdown files (.planning/) | Dolt SQL database | Checkpointing (various backends) | In-memory / conversation | Dolt graph (versioned, queryable) |
| Dependency-aware execution | Inferred from plan ordering | `bd ready` (explicit graph) | Graph edges (explicit) | Sequential or goal-based | `bd ready` (explicit, queryable) |
| Token optimization | Manual (context engineering) | Compaction (closed issues) | None built-in | None built-in | Tiered context + compaction + routing |
| Versioned state (branch/rollback) | Git commits (code only) | Dolt (full state versioning) | Checkpoints (linear) | None | Dolt (branch/merge/rollback for state) |
| Parallel execution | Wave-based (up to 5 agents) | Multi-writer via Dolt server | Async nodes | Parallel task groups | Wave-based + claim-based (bd ready) |
| Human-in-the-loop | Deep questioning, plan review | Manual issue management | Interrupt nodes | Human approval steps | Deep questioning + plan review + bead approval |
| Migration / adoption path | Fresh install only | Fresh install only | From LangChain | Standalone | Coexistence with existing GSD projects |

## Sources

- [Anthropic: Effective context engineering for AI agents](https://www.anthropic.com/engineering/effective-context-engineering-for-ai-agents)
- [Claude Code Hooks Reference](https://code.claude.com/docs/en/hooks)
- [Claude Code Subagent Documentation](https://code.claude.com/docs/en/sub-agents)
- [Beads GitHub Repository](https://github.com/steveyegge/beads)
- [Beads Documentation](https://steveyegge.github.io/beads/)
- [GSD GitHub Repository](https://github.com/gsd-build/get-shit-done)
- [LangGraph vs CrewAI Comparison (DataCamp)](https://www.datacamp.com/tutorial/crewai-vs-langgraph-vs-autogen)
- [AI Agent Orchestration Patterns (Microsoft Azure)](https://learn.microsoft.com/en-us/azure/architecture/ai-ml/guide/ai-agent-design-patterns)
- [Context Engineering for AI Agents (FlowHunt)](https://www.flowhunt.io/blog/context-engineering-ai-agents-token-optimization/)
- [Codified Context Infrastructure (arXiv)](https://arxiv.org/html/2602.20478v1)
- [Deloitte: AI Agent Orchestration 2026](https://www.deloitte.com/us/en/insights/industry/technology/technology-media-and-telecom-predictions/2026/ai-agent-orchestration.html)

---
*Feature research for: AI agent orchestration plugin (Claude Code + Beads/Dolt)*
*Researched: 2026-03-21*
