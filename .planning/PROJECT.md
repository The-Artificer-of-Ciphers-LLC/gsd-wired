# gsd-wired

## What This Is

A Claude Code plugin that fuses GSD's workflow orchestration (phases, research, wave-based execution, verification) with Beads' token-efficient graph persistence (Dolt-backed, hash-based IDs, selective loading, compaction). GSD and Beads are the beads — Claude Code is the wire threading through them. Built in Go as a Claude Code plugin (MCP server + hooks + skills) targeting GSD users who want the same workflow discipline with dramatically lower token consumption.

## Core Value

GSD's full development lifecycle (init → research → plan → execute → verify → ship) running on a beads graph engine so that orchestrator context stays lean and subagents pull only the context they need.

## Requirements

### Validated

- [x] Single Go binary serves as MCP server, hook dispatcher, and CLI tool (INFRA-01) — Phase 1
- [x] Plugin manifest registers MCP server and hooks (INFRA-04) — Phase 1
- [x] Strict stdout discipline — no stray output breaking MCP stdio protocol (INFRA-09) — Phase 1
- [x] bd CLI wrapper layer shells out to `bd --json` for all graph operations (INFRA-03) — Phase 2
- [x] Phase maps to epic bead with metadata (MAP-01) — Phase 2
- [x] Plan maps to task bead with parent-child relationship (MAP-02) — Phase 2
- [x] Wave computed dynamically via `bd ready` (MAP-03) — Phase 2
- [x] Success criteria stored as extensible fields on beads (MAP-04) — Phase 2
- [x] Requirement IDs stored as bead labels for traceability (MAP-05) — Phase 2
- [x] GSD-specific metadata via bd extensible fields (MAP-06) — Phase 2

### Active

- [ ] Claude Code plugin with MCP server wrapping bd CLI for graph operations
- [ ] SessionStart hook loads project context from beads graph on session start
- [ ] PreCompact hook saves in-progress state to beads before context compaction
- [ ] PreToolCall hook intercepts calls to inject bead context or route through bd
- [ ] PostToolCall hook updates bead state after execution (auto-close, progress)
- [ ] Project initialization via deep questioning → context stored as beads
- [ ] Phase = epic bead, Plan = task bead, Wave = dependency layer in the graph
- [ ] Research agents (4 parallel) coordinate as epic + child beads — each researcher claims a child, synthesizer queries all when done
- [ ] Subagents claim beads (bd update --claim), get that bead's context, work, close — minimal prompt overhead
- [ ] Wave-based parallel execution where agents claim unblocked beads via bd ready
- [ ] Post-execution verification against success criteria stored in beads
- [ ] PR creation and milestone tracking via beads state
- [ ] ROADMAP.md and STATE.md replaced by beads graph queries — no markdown bloat
- [ ] PROJECT.md and config.json remain as human-readable files (hybrid state model)
- [ ] GSD-specific metadata stored via bd's extensible fields (phase tags, requirement IDs, success criteria)
- [ ] Coexistence: can read .planning/ as fallback if beads not initialized (migration path for existing GSD projects)
- [ ] Token-aware routing: wire decides what context to load/compact based on budget
- [ ] Tiered context in practice: hot beads get full context, closed beads get compacted

### Out of Scope

- Non-Claude Code platforms (Codex, Gemini, etc.) — v1 is Claude Code only
- Building our own graph persistence — hard dependency on bd/Dolt
- Forking bd — use as-is, extend via wrapper and extensible fields
- Migration tooling from .planning/ to beads — coexistence handles this, formal migration is v2
- Mobile or web UI — CLI/plugin only
- Dolt hosting or remote sync — local-only for v1

## Context

**GSD (get-shit-done-cc):** A meta-prompting system for Claude Code that solves context rot through structured orchestration. Uses markdown files (.planning/) for state, spawns subagents for research/planning/execution, and maintains workflow discipline through phases and waves. Pain point: markdown files bloat context windows as projects grow.

**Beads:** A distributed graph issue tracker for AI agents built on Dolt (Git-for-data SQL database). Hash-based IDs prevent merge collisions. `bd ready` surfaces only unblocked tasks. Compaction summarizes closed tasks to reduce context. Pain point: great persistence layer but no orchestration intelligence on top.

**The fusion:** GSD's orchestration logic becomes the "wire" — it knows phases, manages token budgets, routes work to subagents. Beads' graph becomes the storage layer — tasks, dependencies, state, and context all live in the versioned graph instead of markdown files. Subagents claim beads instead of receiving fat prompts.

**Target users:** Existing GSD users who want better token efficiency with the same workflow UX.

**Prior art on this machine:** User has GSD installed (`~/.claude/get-shit-done/`), has beads-planning directory (`~/.beads-planning/`), has Dolt config (`~/.dolt/`), but neither `bd` nor `dolt` binaries are currently on PATH.

## Constraints

- **Runtime**: Go — matches bd's language for tight integration and potential upstream contributions
- **Hard dependency**: bd CLI and Dolt must be installed — no graceful degradation without them
- **Platform**: Claude Code plugin (MCP server + hooks) — no other agent platforms in v1
- **Hooks**: All four Claude Code hook points (SessionStart, PreCompact, PreToolCall, PostToolCall) required
- **Compatibility**: Must coexist with .planning/ directory for gradual adoption by GSD users

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Hard dependency on bd/Dolt | Leverage existing graph persistence + versioning instead of building our own | — Pending |
| Go as implementation language | Same as bd — enables tight integration, potential upstream PRs, single binary distribution | — Pending |
| Hybrid state: files + beads | PROJECT.md and config.json stay human-readable; dynamic state (roadmap, tasks, progress) moves to beads graph | — Pending |
| Natural mapping: Phase=epic, Plan=task, Wave=dep layer | Preserves GSD mental model while leveraging beads' graph structure | — Pending |
| All four hooks | Deep integration required for token-aware routing and automatic state management | — Pending |
| Coexistence with .planning/ | Lowers adoption barrier for existing GSD users — read .planning/ as fallback | — Pending |

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition** (via `/gsd:transition`):
1. Requirements invalidated? → Move to Out of Scope with reason
2. Requirements validated? → Move to Validated with phase reference
3. New requirements emerged? → Add to Active
4. Decisions to log? → Add to Key Decisions
5. "What This Is" still accurate? → Update if drifted

**After each milestone** (via `/gsd:complete-milestone`):
1. Full review of all sections
2. Core Value check — still the right priority?
3. Audit Out of Scope — reasons still valid?
4. Update Context with current state

---
*Last updated: 2026-03-21 after Phase 2 completion*
