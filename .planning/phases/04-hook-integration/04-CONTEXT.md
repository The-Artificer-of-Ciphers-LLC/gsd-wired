# Phase 4: Hook Integration - Context

**Gathered:** 2026-03-21
**Status:** Ready for planning

<domain>
## Phase Boundary

All four Claude Code hooks (SessionStart, PreCompact, PreToolUse, PostToolUse) load and persist project state through beads. Hooks complete within latency budgets. Delivers INFRA-05, INFRA-06, INFRA-07, INFRA-08.

</domain>

<decisions>
## Implementation Decisions

### Design principle
- **D-01:** gsdw is the developer interface. All hook behavior is invisible infrastructure — the developer interacts with gsdw commands, not hooks directly.
- **D-02:** Developer never sees hook internals, PreCompact saves, or tool-use context injection. These happen behind the scenes.

### SessionStart hook
- **D-03:** Full context loading: project name, current phase, ready tasks with objectives, recent decisions, blockers, last session summary
- **D-04:** Both output formats: structured JSON for machine consumption + human-readable markdown for context injection
- **D-05:** gsdw handles uninitialized state (no .beads/) — Claude's discretion on approach (hint, auto-init, silent)
- **D-06:** gsdw handles slow Dolt gracefully — Claude's discretion (cache fallback, partial results, timeout handling)

### PreCompact hook
- **D-07:** PreCompact cannot block compaction — observability only. Two-stage: fast local write, then async Dolt sync.
- **D-08:** What to save, where to buffer, when to sync — all at Claude's discretion. Optimize for performance + reliability.
- **D-09:** Developer never sees PreCompact behavior. It's crash-recovery infrastructure.

### PreToolUse/PostToolUse hooks
- **D-10:** Scope, filtering, injection level, auto-detection — all at Claude's discretion. Developer interacts with gsdw commands, not individual file edits.
- **D-11:** gsdw handles latency budget (<500ms) gracefully — Claude's discretion on degradation strategy.

### Claude's Discretion
- SessionStart: what to show when uninitialized, cache vs live query strategy, output structure
- PreCompact: state snapshot contents, local buffer location, async sync timing, skip-if-unchanged logic
- PreToolUse: which tools trigger injection, what context to inject, relevance filtering
- PostToolUse: which tools trigger updates, auto-detection of related beads, progress tracking granularity
- All hooks: timeout/degradation behavior within latency budgets
- Hook handler architecture: shared state across hooks vs independent, connection reuse

</decisions>

<specifics>
## Specific Ideas

- SessionStart is the most impactful hook — it's the "cold start" that gives Claude Code full project awareness
- PreCompact is the most constrained — can't block, must be fast, async sync is critical
- PreToolUse/PostToolUse are the highest frequency — every tool call fires them, performance is critical
- The existing hook dispatcher (`internal/hook/dispatcher.go`) already parses JSON and validates events — Phase 4 adds real logic to the stub handlers
- Latency budgets from ROADMAP: SessionStart <2s, PreCompact <200ms fast path, Pre/PostToolUse <500ms

</specifics>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase 1-3 foundation
- `internal/hook/dispatcher.go` — Existing hook dispatcher (parses JSON stdin, validates event, writes response)
- `internal/hook/events.go` — Event name constants (SessionStart, PreCompact, PreToolUse, PostToolUse)
- `internal/mcp/init.go` — serverState with sync.Once lazy init pattern (reusable for hooks)
- `internal/graph/client.go` — bd wrapper with batch mode
- `internal/graph/query.go` — ListReady, GetBead, QueryByLabel
- `internal/graph/index.go` — Local index at .gsdw/index.json
- `hooks/hooks.json` — All four events registered pointing to `gsdw hook <event>`

### Project context
- `.planning/PROJECT.md` — Core value, constraints, key decisions
- `.planning/REQUIREMENTS.md` — INFRA-05 through INFRA-08 define this phase's deliverables
- `.planning/ROADMAP.md` §Phase 4 — Success criteria (5 items including latency budgets)

### Prior research
- `.planning/research/SUMMARY.md` — PreCompact limitations, two-stage save pattern

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/hook/dispatcher.go` — JSON stdin parsing, event validation, stdout response. Phase 4 replaces no-op handlers with real logic.
- `internal/mcp/init.go` — sync.Once lazy init pattern. Hooks need similar Dolt initialization.
- `internal/graph/` — Full CRUD and query layer. Hooks call these for state loading/persistence.
- `.gsdw/index.json` — Local index for fast lookups without Dolt queries.

### Established Patterns
- Injected io.Reader/io.Writer in hook.Dispatch() for testability
- Two-tier error handling in graph.Client
- Batch mode for write operations
- fake_bd test binary for unit tests

### Integration Points
- `internal/hook/dispatcher.go` — Hook entry point, needs real handler implementations
- `internal/graph/client.go` — Hooks call graph operations for state
- `.gsdw/` directory — Local state buffer for PreCompact fast path
- `hooks/hooks.json` — Already registered, no changes needed

</code_context>

<deferred>
## Deferred Ideas

- Token-aware context injection (deciding HOW MUCH to inject based on budget) — Phase 9
- File-aware PreToolUse (inject context specific to the file being edited) — v2 (TOKEN-A01)
- Slash command integration with hooks — Phase 5+

</deferred>

---

*Phase: 04-hook-integration*
*Context gathered: 2026-03-21*
