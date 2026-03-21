# Phase 5: Project Initialization - Context

**Gathered:** 2026-03-21
**Status:** Ready for planning

<domain>
## Phase Boundary

Users can initialize a new gsd-wired project through guided questioning that produces a beads graph. Three init modes: full, quick, PR/issue. Status command shows project state. Delivers INIT-01 through INIT-05, CMD-01.

</domain>

<decisions>
## Implementation Decisions

### Design principle
- **D-01:** This is the first user-facing workflow phase. The developer interacts with `/gsd-wired:init` and `/gsd-wired:status` directly. UX matters here.
- **D-02:** gsd-wired translates developer answers to beads behind the scenes. Developer never thinks about bead structure.

### Questioning flow
- **D-03:** Replicate GSD's current questioning flow — same ~12 questions, same order, same categories (what, why, who, constraints, done criteria, tech stack, prior art, risks). Familiarity for GSD users.
- **D-04:** Interactive questioning — ask one question, wait for answer, ask next based on response. Not batch.
- **D-05:** Three init modes:
  - **Full init** — 12-question deep questioning flow (new project from scratch)
  - **Quick init** — 3 essential questions (faster startup for simple projects)
  - **PR/Issue mode** — existing gsd-wired project, import a PR or issue for review/integration. gsdw asks questions about the PR, handles bead creation behind the scenes. Developer just cares about working the PR.

### Init output structure
- **D-06:** Bead granularity (category beads vs per-answer beads) at Claude's discretion — optimize for performance and reliability.
- **D-07:** PROJECT.md and config.json written independently as human-readable files. They are for the developer to review. Not derived from bead data — beads and files are parallel views.
- **D-08:** After init completes: pause and ask developer if ready to proceed. Auto-proceed after 30 seconds of silence.

### Status display (`/gsd-wired:status`)
- **D-09:** Dashboard format: project name, current phase, progress bar, ready tasks, recent activity. Like GSD's current status box.
- **D-10:** GSD-familiar terms throughout — phases, plans, waves. Never expose beads graph structure to the developer.
- **D-11:** PR/issue view in separate/debug mode, not mixed with main project status.
- **D-12:** Auto-show on session start (via SessionStart hook), but easily dismissable by the developer.

### Claude's Discretion
- Bead structure for init output (category vs per-answer, number of children)
- Quick init: which 3 questions are essential
- PR/Issue mode: questioning flow and bead structure for imported work
- Status dashboard layout and exact fields
- How "easily dismissable" is implemented (flag, config, one-time setting)
- SESSION context injection format for auto-show

</decisions>

<specifics>
## Specific Ideas

- GSD's current init questions for reference: What are you building? Why? Who is it for? What does success look like? What's the tech stack? What constraints exist? What's the prior art? What are the risks? What does done look like? etc.
- The 30-second auto-proceed after init is a "don't block the developer" pattern — they said yes by not saying no
- PR/Issue mode is unique to gsd-wired (GSD doesn't have this) — it's for developers joining mid-project
- Status auto-show leverages the SessionStart hook already built in Phase 4

</specifics>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase 1-4 foundation
- `internal/graph/create.go` — CreatePhase, CreatePlan (used to create project epic + children)
- `internal/graph/query.go` — ListReady, GetBead, QueryByLabel (used by status)
- `internal/graph/index.go` — Local index (used by status for fast lookups)
- `internal/mcp/tools.go` — 8 MCP tools (init will call these)
- `internal/mcp/init.go` — serverState with auto bd-init (handles .beads/ creation)
- `internal/hook/session_start.go` — SessionStart handler (status auto-show hooks into this)
- `internal/cli/ready.go` — gsdw ready command (status reuses patterns)
- `.claude-plugin/plugin.json` — Plugin manifest (slash commands added here)

### Project context
- `.planning/PROJECT.md` — Core value, constraints, key decisions
- `.planning/REQUIREMENTS.md` — INIT-01 through INIT-05, CMD-01 define this phase's deliverables
- `.planning/ROADMAP.md` §Phase 5 — Success criteria (5 items that must be TRUE)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/cli/ready.go` — Tree-formatted output pattern, reusable for status display
- `internal/hook/session_start.go` — `buildSessionContext` already generates project overview markdown. Status can share this logic.
- `internal/mcp/tools.go` — create_phase, create_plan tools. Init flow calls these to create beads.
- `internal/graph/client.go` — bd wrapper. Init uses this for all graph operations.

### Established Patterns
- Cobra subcommands for CLI commands (serve, hook, bd, ready, version)
- renderTree/renderJSON dual output (from ready.go)
- hookState/serverState lazy init via sync.Once
- Strict stdout discipline, slog to stderr

### Integration Points
- `.claude-plugin/plugin.json` — Must register `/gsd-wired:init` and `/gsd-wired:status` slash commands
- `hooks/hooks.json` — No changes needed (all four hooks already registered)
- `internal/cli/root.go` — New `init` and `status` subcommands added here
- `.gsdw/` directory — Status reads index.json for fast lookups

</code_context>

<deferred>
## Deferred Ideas

- Roadmap generation from init (Phase 6+ handles planning)
- Requirements definition (Phase 6+)
- Research phase from init context (Phase 6)
- Token-aware status display (Phase 9)

</deferred>

---

*Phase: 05-project-init*
*Context gathered: 2026-03-21*
