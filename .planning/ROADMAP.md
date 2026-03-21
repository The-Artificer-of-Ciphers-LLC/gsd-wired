# Roadmap: gsd-wired

## Overview

gsd-wired delivers GSD's full development lifecycle as a Claude Code plugin backed by a beads graph engine. The roadmap builds from the ground up: binary scaffold and bd integration first, then hook-based Claude Code integration, then each GSD workflow phase (init, research, plan, execute, verify, ship), then the core innovation of token-aware context routing, and finally coexistence polish for existing GSD users. Each phase delivers a coherent, testable capability.

## Phases

**Phase Numbering:**
- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [x] **Phase 1: Binary Scaffold** - Go binary with Cobra subcommands, plugin manifest, stdout discipline
- [x] **Phase 2: Graph Primitives** - bd CLI wrapper and GSD-to-beads domain mapping
- [ ] **Phase 3: MCP Server** - MCP server with lazy Dolt init and tool registration
- [ ] **Phase 4: Hook Integration** - All four Claude Code hooks with state persistence
- [ ] **Phase 5: Project Initialization** - Init slash command with deep questioning flow producing beads
- [ ] **Phase 6: Research + Planning** - Research agents coordinating as beads, plan creation with dependencies
- [ ] **Phase 7: Execution + Verification** - Wave-based parallel execution and post-execution verification
- [ ] **Phase 8: Ship + Status** - PR creation, phase advancement, and project status from graph
- [ ] **Phase 9: Token-Aware Context** - Hot/warm/cold tiering, budget tracking, context injection
- [ ] **Phase 10: Coexistence** - .planning/ fallback reading and gradual adoption path

## Phase Details

### Phase 1: Binary Scaffold
**Goal**: A single Go binary that runs as MCP server, hook dispatcher, or CLI tool with correct plugin registration
**Depends on**: Nothing (first phase)
**Requirements**: INFRA-01, INFRA-04, INFRA-09
**Success Criteria** (what must be TRUE):
  1. Running `gsd-wired serve` starts a process that listens on stdio
  2. Running `gsd-wired hook <event>` dispatches to the correct handler skeleton and exits
  3. Plugin manifest (.claude-plugin/plugin.json) is valid and registers all entry points
  4. No output appears on stdout except valid MCP JSON or hook JSON responses (stderr-only logging verified)
**Plans**: 2 plans

Plans:
- [x] 01-01-PLAN.md — Go binary skeleton: Cobra CLI, slog logging, MCP serve, hook dispatcher, bd passthrough
- [x] 01-02-PLAN.md — Plugin registration files and integration test suite

### Phase 2: Graph Primitives
**Goal**: The plugin can perform all beads graph operations and map GSD concepts onto bead structures
**Depends on**: Phase 1
**Requirements**: INFRA-03, MAP-01, MAP-02, MAP-03, MAP-04, MAP-05, MAP-06
**Success Criteria** (what must be TRUE):
  1. bd CLI wrapper can create, read, update, and close beads via `bd --json` and parse responses
  2. A phase can be created as an epic bead with phase number, goal, and success criteria metadata
  3. A plan can be created as a task bead with parent-child relationship to its phase epic
  4. `bd ready` returns unblocked tasks and the wrapper surfaces them as the current wave
  5. Requirement IDs and GSD metadata are stored as bead tags/extensible fields and queryable
**Plans**: 2 plans

Plans:
- [x] 02-01-PLAN.md — internal/graph/ package: bd client, Bead types, CRUD operations, index, tests with fake bd
- [x] 02-02-PLAN.md — gsdw ready subcommand with tree format, --json, --phase filter

### Phase 3: MCP Server
**Goal**: The MCP server responds to protocol requests and exposes GSD tools with lazy database initialization
**Depends on**: Phase 2
**Requirements**: INFRA-02, INFRA-10
**Success Criteria** (what must be TRUE):
  1. MCP server responds to `initialize` request within 500ms (before Dolt is ready)
  2. First tool call triggers Dolt initialization transparently (lazy init)
  3. Tool list includes all planned GSD tools (stubs acceptable at this phase)
  4. Dolt writes are batched at operation boundaries, not per-call
**Plans**: 2 plans

Plans:
- [x] 03-01-PLAN.md — Batch write mode in graph.Client + lazy init serverState with sync.Once
- [ ] 03-02-PLAN.md — MCP tool registration (8 tools) and Serve() wiring

### Phase 4: Hook Integration
**Goal**: Claude Code lifecycle events automatically load and persist project state through beads
**Depends on**: Phase 3
**Requirements**: INFRA-05, INFRA-06, INFRA-07, INFRA-08
**Success Criteria** (what must be TRUE):
  1. SessionStart hook loads active project state (open epics, ready tasks) into Claude Code context
  2. PreCompact hook saves in-progress state to fast local buffer, then syncs to Dolt asynchronously
  3. PreToolUse hook injects relevant bead context before state-changing tool calls
  4. PostToolUse hook updates bead state after tool execution (progress, status changes)
  5. Hooks complete within latency budget (SessionStart <2s, PreCompact <200ms fast path, Pre/PostToolUse <500ms)
**Plans**: TBD

Plans:
- [ ] 04-01: TBD
- [ ] 04-02: TBD

### Phase 5: Project Initialization
**Goal**: Users can initialize a new gsd-wired project through guided questioning that produces a beads graph
**Depends on**: Phase 4
**Requirements**: INIT-01, INIT-02, INIT-03, INIT-04, INIT-05, CMD-01
**Success Criteria** (what must be TRUE):
  1. Running `/gsd-wired:init` launches a deep questioning flow (what, why, who, done criteria)
  2. Questioning produces an epic bead (project) with child context beads (decisions, constraints)
  3. PROJECT.md and config.json are created as human-readable files alongside the beads graph
  4. `bd init` creates .beads/ directory with Dolt-backed storage during initialization
  5. Running `/gsd-wired:status` after init shows the project state from the beads graph
**Plans**: TBD

Plans:
- [ ] 05-01: TBD
- [ ] 05-02: TBD

### Phase 6: Research + Planning
**Goal**: Users can run research phases and create dependency-aware plans, all coordinated through beads
**Depends on**: Phase 5
**Requirements**: RSRCH-01, RSRCH-02, RSRCH-03, RSRCH-04, PLAN-01, PLAN-02, PLAN-03, PLAN-04, CMD-03
**Success Criteria** (what must be TRUE):
  1. Research phase creates an epic bead with 4 child beads (stack, features, architecture, pitfalls)
  2. Each research agent claims its child bead via `bd update --claim` and stores results as bead content
  3. Synthesizer agent queries all 4 child beads when closed and produces a summary bead
  4. `/gsd-wired:plan` decomposes a phase epic into task beads with dependency relationships
  5. Each task bead has success criteria, estimated complexity, and file touch list
  6. Plan checker agent validates the plan achieves the phase goal before execution begins
**Plans**: TBD

Plans:
- [ ] 06-01: TBD
- [ ] 06-02: TBD
- [ ] 06-03: TBD

### Phase 7: Execution + Verification
**Goal**: Users can execute waves of parallel tasks and verify phase completion against success criteria
**Depends on**: Phase 6
**Requirements**: EXEC-01, EXEC-02, EXEC-03, EXEC-04, EXEC-05, EXEC-06, VRFY-01, VRFY-02, VRFY-03, CMD-04, CMD-05, CMD-07
**Success Criteria** (what must be TRUE):
  1. `/gsd-wired:execute` runs all unblocked tasks from `bd ready` in parallel
  2. Each execution subagent receives only its claimed bead's context chain (task + parent epic + dependency summaries)
  3. Completing a task closes its bead, triggers the next wave of unblocked tasks
  4. Each completed task produces an atomic git commit with the bead ID in the commit message
  5. Agent output is validated at the orchestrator before downstream consumption
  6. `/gsd-wired:verify` checks success criteria from the phase epic and reports pass/fail per criterion
  7. Failed verification criteria automatically produce new task beads for remediation
**Plans**: TBD

Plans:
- [ ] 07-01: TBD
- [ ] 07-02: TBD
- [ ] 07-03: TBD

### Phase 8: Ship + Status
**Goal**: Users can ship completed phases as PRs and view project state from the beads graph
**Depends on**: Phase 7
**Requirements**: SHIP-01, SHIP-02, CMD-02, CMD-06
**Success Criteria** (what must be TRUE):
  1. `/gsd-wired:ship` creates a PR with bead-sourced summary (requirements covered, phases completed)
  2. Phase completion updates bead state and triggers next phase readiness
  3. `/gsd-wired:status` shows current project state entirely from beads graph queries (no markdown parsing)
**Plans**: TBD

Plans:
- [ ] 08-01: TBD
- [ ] 08-02: TBD

### Phase 9: Token-Aware Context
**Goal**: The plugin minimizes token consumption through intelligent context routing based on bead state
**Depends on**: Phase 7
**Requirements**: TOKEN-01, TOKEN-02, TOKEN-03, TOKEN-04, TOKEN-05, TOKEN-06
**Success Criteria** (what must be TRUE):
  1. Graph queries replace full markdown file reads -- context is O(relevant) not O(total)
  2. Subagent prompts contain only claimed bead context, not full project state
  3. Closed beads are automatically compacted (summary replaces full content)
  4. Active beads get full context, recent beads get summaries, done beads get IDs only (hot/warm/cold)
  5. Token budget tracking estimates tokens per bead and fits injected context within remaining window
**Plans**: TBD

Plans:
- [ ] 09-01: TBD
- [ ] 09-02: TBD
- [ ] 09-03: TBD

### Phase 10: Coexistence
**Goal**: Existing GSD users can adopt gsd-wired gradually without abandoning their .planning/ workflow
**Depends on**: Phase 4
**Requirements**: COMPAT-01, COMPAT-02, COMPAT-03
**Success Criteria** (what must be TRUE):
  1. Plugin detects .planning/ directory and reads it as fallback when beads are not initialized
  2. Existing STATE.md and ROADMAP.md are parseable into bead-equivalent query results
  3. New work always goes to beads graph; .planning/ files are never written to by the plugin
**Plans**: TBD

Plans:
- [ ] 10-01: TBD
- [ ] 10-02: TBD

## Progress

**Execution Order:**
Phases execute in numeric order: 1 -> 2 -> 3 -> 4 -> 5 -> 6 -> 7 -> 8 -> 9 -> 10
Note: Phase 9 and Phase 10 can execute in parallel (both depend on earlier phases, not each other).

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Binary Scaffold | 2/2 | Complete | 2026-03-21 |
| 2. Graph Primitives | 2/2 | Complete | 2026-03-21 |
| 3. MCP Server | 1/2 | In progress | - |
| 4. Hook Integration | 0/TBD | Not started | - |
| 5. Project Initialization | 0/TBD | Not started | - |
| 6. Research + Planning | 0/TBD | Not started | - |
| 7. Execution + Verification | 0/TBD | Not started | - |
| 8. Ship + Status | 0/TBD | Not started | - |
| 9. Token-Aware Context | 0/TBD | Not started | - |
| 10. Coexistence | 0/TBD | Not started | - |
