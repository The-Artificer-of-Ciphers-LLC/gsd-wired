# Requirements: gsd-wired

**Defined:** 2026-03-21
**Core Value:** GSD's full development lifecycle running on a beads graph engine so that orchestrator context stays lean and subagents pull only the context they need.

## v1 Requirements

Requirements for initial release. Each maps to roadmap phases.

### Plugin Infrastructure

- [x] **INFRA-01**: Single Go binary serves as MCP server (stdio), hook dispatcher (subcommand), and CLI tool
- [ ] **INFRA-02**: MCP server exposes tools via official Go SDK (v1.4.1) with lazy Dolt initialization
- [ ] **INFRA-03**: bd CLI wrapper layer shells out to `bd --json` for all graph operations
- [x] **INFRA-04**: Plugin manifest (.claude-plugin/plugin.json) registers MCP server, hooks, and slash commands
- [ ] **INFRA-05**: SessionStart hook loads active project state from beads graph into context
- [ ] **INFRA-06**: PreCompact hook saves in-progress work state to beads (two-stage: fast local write, async Dolt commit)
- [ ] **INFRA-07**: PreToolUse hook injects relevant bead context before tool execution
- [ ] **INFRA-08**: PostToolUse hook updates bead state after tool execution (progress, status changes)
- [x] **INFRA-09**: Strict stdout discipline — no stray output that could break MCP stdio protocol
- [ ] **INFRA-10**: Batched Dolt writes at wave boundaries to prevent write amplification

### GSD Domain Mapping

- [ ] **MAP-01**: Phase maps to epic bead with metadata (phase number, goal, success criteria)
- [ ] **MAP-02**: Plan maps to task bead with parent-child relationship to phase epic
- [ ] **MAP-03**: Wave computed dynamically from dependency graph via `bd ready`
- [ ] **MAP-04**: Success criteria stored as extensible fields on task beads
- [ ] **MAP-05**: Requirement IDs (REQ-IDs) stored as bead tags for traceability
- [ ] **MAP-06**: GSD-specific metadata stored via bd's extensible fields (phase tags, status, wave assignment)

### Project Initialization

- [ ] **INIT-01**: User can initialize a new project via `/gsd-wired:init` slash command
- [ ] **INIT-02**: Deep questioning flow captures project context (what, why, who, done criteria)
- [ ] **INIT-03**: Questioning produces epic bead (project) + context beads (decisions, constraints)
- [ ] **INIT-04**: PROJECT.md and config.json remain as human-readable files (hybrid state model)
- [ ] **INIT-05**: `bd init` creates .beads/ directory with Dolt-backed storage

### Research Phase

- [ ] **RSRCH-01**: Research phase creates epic bead with 4 child beads (stack, features, architecture, pitfalls)
- [ ] **RSRCH-02**: Each research agent claims its child bead via `bd update --claim`
- [ ] **RSRCH-03**: Research results stored as bead content/metadata, not separate markdown files
- [ ] **RSRCH-04**: Synthesizer agent queries all 4 child beads when they close, produces summary bead

### Planning Phase

- [ ] **PLAN-01**: User can create a phase plan via `/gsd-wired:plan` slash command
- [ ] **PLAN-02**: Plan decomposes phase epic into task beads with dependencies
- [ ] **PLAN-03**: Each task bead has success criteria, estimated complexity, and file touch list
- [ ] **PLAN-04**: Plan checker agent validates plan achieves phase goal before execution

### Execution Phase

- [ ] **EXEC-01**: Wave execution runs unblocked tasks in parallel (tasks from `bd ready`)
- [ ] **EXEC-02**: Each execution subagent claims a task bead and receives only that bead's context chain
- [ ] **EXEC-03**: Subagent context includes: task description, success criteria, parent epic summary, dependency bead summaries
- [ ] **EXEC-04**: On task completion, subagent closes bead with results, triggering next wave
- [ ] **EXEC-05**: Atomic git commits per completed task with bead ID in commit message
- [ ] **EXEC-06**: Agent output validated at orchestrator before downstream consumption (error amplification prevention)

### Verification Phase

- [ ] **VRFY-01**: Verification agent reads success criteria from phase epic's extensible fields
- [ ] **VRFY-02**: Verification runs checks against codebase and reports pass/fail per criterion
- [ ] **VRFY-03**: Failed criteria produce new task beads for remediation

### Ship Phase

- [ ] **SHIP-01**: PR creation with bead-sourced summary (requirements covered, phases completed)
- [ ] **SHIP-02**: Phase completion updates bead state and triggers next phase readiness

### Token Optimization

- [ ] **TOKEN-01**: Graph queries replace full markdown file reads (O(relevant) not O(total))
- [ ] **TOKEN-02**: Subagent prompts contain only claimed bead context, not full project state
- [ ] **TOKEN-03**: Closed beads automatically compacted (summary replaces full content)
- [ ] **TOKEN-04**: Token-aware context routing: hot beads (active) get full context, warm (recent) get summaries, cold (done) get IDs only
- [ ] **TOKEN-05**: Context budget tracking estimates tokens per bead and fits within remaining window
- [ ] **TOKEN-06**: Tiered context injection in SessionStart based on available token budget

### Slash Commands

- [ ] **CMD-01**: `/gsd-wired:init` — Initialize new project with deep questioning
- [ ] **CMD-02**: `/gsd-wired:status` — Show project state from beads graph
- [ ] **CMD-03**: `/gsd-wired:plan` — Create phase plan (task beads with dependencies)
- [ ] **CMD-04**: `/gsd-wired:execute` — Execute current wave of unblocked tasks
- [ ] **CMD-05**: `/gsd-wired:verify` — Verify phase against success criteria
- [ ] **CMD-06**: `/gsd-wired:ship` — Create PR and advance to next phase
- [ ] **CMD-07**: `/gsd-wired:ready` — Show unblocked tasks (next wave)

### Coexistence

- [ ] **COMPAT-01**: Plugin detects .planning/ directory and reads it as fallback when beads not initialized
- [ ] **COMPAT-02**: Existing GSD STATE.md/ROADMAP.md parseable into bead-equivalent queries
- [ ] **COMPAT-03**: New work always goes to beads graph; .planning/ files are read-only fallback

## v2 Requirements

Deferred to future release. Tracked but not in current roadmap.

### Token Advanced

- **TOKEN-A01**: PreToolUse context injection with file-aware relevance (agent editing auth.ts gets auth bead context)
- **TOKEN-A02**: Automatic dependency detection suggestions (AI proposes, human confirms)

### Platform

- **PLAT-01**: TUI visualization of beads graph and project state
- **PLAT-02**: Dolt remote sync to DoltHub for backup/sharing
- **PLAT-03**: Non-Claude Code agent support (OpenCode, Codex)

### Migration

- **MIGR-01**: Formal .planning/ → beads migration tooling
- **MIGR-02**: Bidirectional sync between .planning/ and beads during transition

## Out of Scope

| Feature | Reason |
|---------|--------|
| Web UI / dashboard | Massive scope, wrong audience for CLI-native users, contradicts lean philosophy |
| Multi-model support | Hook system is Claude Code-specific; other platforms have different integration surfaces |
| Real-time multi-developer collaboration | Requires conflict resolution UI, presence, locking — use Dolt branch-and-merge instead |
| Auto-planning without human review | Removes human-in-the-loop that makes GSD reliable |
| Plugin/extensibility API | Premature abstraction before v1 stabilizes; use bd CLI as extension point |
| Remote hosting / cloud infrastructure | Local-only is a feature for v1 — fast, private, no network dependency |
| Full migration tooling | Coexistence handles this; formal migration is fragile and only used once per project |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| INFRA-01 | Phase 1 | Pending |
| INFRA-02 | Phase 3 | Pending |
| INFRA-03 | Phase 2 | Pending |
| INFRA-04 | Phase 1 | Pending |
| INFRA-05 | Phase 4 | Pending |
| INFRA-06 | Phase 4 | Pending |
| INFRA-07 | Phase 4 | Pending |
| INFRA-08 | Phase 4 | Pending |
| INFRA-09 | Phase 1 | Pending |
| INFRA-10 | Phase 3 | Pending |
| MAP-01 | Phase 2 | Pending |
| MAP-02 | Phase 2 | Pending |
| MAP-03 | Phase 2 | Pending |
| MAP-04 | Phase 2 | Pending |
| MAP-05 | Phase 2 | Pending |
| MAP-06 | Phase 2 | Pending |
| INIT-01 | Phase 5 | Pending |
| INIT-02 | Phase 5 | Pending |
| INIT-03 | Phase 5 | Pending |
| INIT-04 | Phase 5 | Pending |
| INIT-05 | Phase 5 | Pending |
| RSRCH-01 | Phase 6 | Pending |
| RSRCH-02 | Phase 6 | Pending |
| RSRCH-03 | Phase 6 | Pending |
| RSRCH-04 | Phase 6 | Pending |
| PLAN-01 | Phase 6 | Pending |
| PLAN-02 | Phase 6 | Pending |
| PLAN-03 | Phase 6 | Pending |
| PLAN-04 | Phase 6 | Pending |
| EXEC-01 | Phase 7 | Pending |
| EXEC-02 | Phase 7 | Pending |
| EXEC-03 | Phase 7 | Pending |
| EXEC-04 | Phase 7 | Pending |
| EXEC-05 | Phase 7 | Pending |
| EXEC-06 | Phase 7 | Pending |
| VRFY-01 | Phase 7 | Pending |
| VRFY-02 | Phase 7 | Pending |
| VRFY-03 | Phase 7 | Pending |
| SHIP-01 | Phase 8 | Pending |
| SHIP-02 | Phase 8 | Pending |
| TOKEN-01 | Phase 9 | Pending |
| TOKEN-02 | Phase 9 | Pending |
| TOKEN-03 | Phase 9 | Pending |
| TOKEN-04 | Phase 9 | Pending |
| TOKEN-05 | Phase 9 | Pending |
| TOKEN-06 | Phase 9 | Pending |
| CMD-01 | Phase 5 | Pending |
| CMD-02 | Phase 8 | Pending |
| CMD-03 | Phase 6 | Pending |
| CMD-04 | Phase 7 | Pending |
| CMD-05 | Phase 7 | Pending |
| CMD-06 | Phase 8 | Pending |
| CMD-07 | Phase 7 | Pending |
| COMPAT-01 | Phase 10 | Pending |
| COMPAT-02 | Phase 10 | Pending |
| COMPAT-03 | Phase 10 | Pending |

**Coverage:**
- v1 requirements: 56 total
- Mapped to phases: 56
- Unmapped: 0

---
*Requirements defined: 2026-03-21*
*Last updated: 2026-03-21 after roadmap creation*
