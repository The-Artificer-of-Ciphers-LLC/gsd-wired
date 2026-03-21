# Phase 10: Coexistence - Context

**Gathered:** 2026-03-21
**Status:** Ready for planning
**Source:** Auto-mode (Claude selected recommended defaults)

<domain>
## Phase Boundary

Existing GSD users can adopt gsd-wired gradually without abandoning their .planning/ workflow. Plugin detects and reads .planning/ as fallback. Delivers COMPAT-01, COMPAT-02, COMPAT-03.

</domain>

<decisions>
## Implementation Decisions

### Design principle
- **D-01:** gsdw is the developer interface. Coexistence is transparent — if .planning/ exists but .beads/ doesn't, gsdw reads .planning/ seamlessly.
- **D-02:** New work always goes to beads graph. .planning/ files are read-only fallback, never written by the plugin.

### .planning/ detection and fallback
- **D-03:** On SessionStart and status commands, check for .planning/ directory. If present and .beads/ absent, parse .planning/ files for project state.
- **D-04:** Parse STATE.md for current phase/plan/progress. Parse ROADMAP.md for phase list and completion status. Parse PROJECT.md for project context.
- **D-05:** Parsed .planning/ data returned in the same format as bead-sourced data. The SKILL.md and developer see identical output regardless of source.

### What gets parsed
- **D-06:** STATE.md → current phase number, plan progress, last activity
- **D-07:** ROADMAP.md → phase list with completion checkboxes, phase goals
- **D-08:** PROJECT.md → project name, core value, requirements summary

### Read-only guarantee
- **D-09:** Plugin NEVER writes to .planning/ files. All new state goes to beads. This is a hard constraint, not a preference.
- **D-10:** If both .planning/ and .beads/ exist, beads take priority. .planning/ is only consulted when beads are absent.

### Claude's Discretion
- Markdown parsing approach (regex vs structured parser)
- How much of STATE.md/ROADMAP.md to parse (full vs essential fields)
- Graceful degradation when .planning/ files have unexpected format
- Whether to surface "running in .planning/ compatibility mode" to developer

</decisions>

<specifics>
## Specific Ideas

- This is the adoption bridge for existing GSD users — they install gsd-wired plugin and it immediately understands their project
- The .planning/ parser doesn't need to be perfect — it's a compatibility layer, not a production parser
- Once the user runs /gsd-wired:init, beads take over and .planning/ is never consulted again
- COMPAT-03 (new work to beads, .planning/ read-only) is already enforced by design — all MCP tools write to beads, none write to .planning/

</specifics>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase 4-5 foundation
- `internal/hook/session_start.go` — SessionStart handler (add .planning/ fallback path)
- `internal/mcp/get_status.go` — get_status tool (add .planning/ fallback)
- `internal/mcp/init.go` — serverState.init checks for .beads/ (add .planning/ detection)

### Project context
- `.planning/PROJECT.md` — Example of what needs to be parsed
- `.planning/STATE.md` — Example of what needs to be parsed
- `.planning/ROADMAP.md` — Example of what needs to be parsed
- `.planning/REQUIREMENTS.md` — COMPAT-01, COMPAT-02, COMPAT-03

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/hook/session_start.go` — Already checks for .beads/ existence. Add .planning/ fallback.
- `internal/mcp/get_status.go` — Returns structured status. Add .planning/ source.

### Established Patterns
- Graceful degradation (SessionStart degrades to hint when no .beads/)
- Best-effort parsing (partial results better than failure)
- No external dependencies

### Integration Points
- `internal/hook/session_start.go` — .planning/ fallback in buildSessionContext
- `internal/mcp/get_status.go` — .planning/ fallback in handleGetStatus
- New package: `internal/compat/` — .planning/ parser (keeps fallback logic isolated)

</code_context>

<deferred>
## Deferred Ideas

- Formal .planning/ → beads migration tooling (MIGR-01, v2)
- Bidirectional sync between .planning/ and beads (MIGR-02, v2)

</deferred>

---

*Phase: 10-coexistence*
*Context gathered: 2026-03-21 via auto-mode*
