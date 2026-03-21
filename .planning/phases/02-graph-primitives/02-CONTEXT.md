# Phase 2: Graph Primitives - Context

**Gathered:** 2026-03-21
**Status:** Ready for planning

<domain>
## Phase Boundary

bd CLI wrapper layer and GSD-to-beads domain mapping. The plugin can perform all beads graph operations (create, read, update, close) and map GSD concepts (phase, plan, wave, requirements) onto bead structures. Delivers INFRA-03, MAP-01 through MAP-06.

</domain>

<decisions>
## Implementation Decisions

### Design principle
- **D-01:** bd is an implementation detail — gsdw is the interface. Users never need to think about bd query syntax, label conventions, or metadata keys. All design decisions optimize for gsdw's needs.
- **D-02:** Developers interact with gsdw commands only. bd mechanics (claiming, wave computation, bead creation timing) happen behind the scenes.

### GSD-to-beads field mapping
- **D-03:** Phases use `--type epic`, plans use `--type task` with `--parent <phase-bead-id>`
- **D-04:** Success criteria stored in `--acceptance` (bd native field)
- **D-05:** Requirement IDs stored as labels — `--add-label INFRA-01` for queryability via `bd query label=INFRA-01`
- **D-06:** GSD role markers as labels — `--add-label gsd:phase` and `--add-label gsd:plan`
- **D-07:** Inter-plan dependencies via `--deps <bead-id>` (bd native)
- **D-08:** Human-readable context in `--context` (visible in `bd show`), structured machine data in `--metadata` JSON (phase number, goal, wave number, files_modified)
- **D-09:** Native bd fields for everything bd supports (type, parent, deps, acceptance, context). Metadata JSON only for GSD-specific extras (gsd_phase, gsd_plan, gsd_wave, gsd_files_modified, gsd_req_ids)

### Wave computation
- **D-10:** Hybrid model — wave numbers stored in metadata (`"gsd_wave": 1`) for reporting, but `bd ready` used for actual execution ordering. Wave numbers are informational, not structural.
- **D-11:** Wave cache invalidated after every `bd close` (task completion triggers recompute)
- **D-12:** `gsdw ready` shows both tree format (human) and `--json` (machine). Tree format groups by phase.
- **D-13:** On task close, gsdw notifies what became ready: "Closed bd-a3f → 2 new tasks ready: bd-c9d, bd-e1f"
- **D-14:** Own simpler interface for `gsdw ready` (`--phase N`, `--json`). `gsdw bd ready` passthrough exists as escape hatch for full bd power.
- **D-15:** "Queued" terminology for dependency-waiting tasks (not "blocked" — nothing is wrong, they're just waiting)
- **D-16:** Progress shown as "3 ready │ 4 queued │ 7 remaining" — no upfront wave numbering needed

### Bead discoverability
- **D-17:** Metadata is canonical lookup — `gsd_phase: 3`, `gsd_plan: "02-01"` in metadata JSON. gsdw controls all reads/writes.
- **D-18:** Local index at `.gsdw/index.json` for fast lookups, rebuildable from `bd list --json` if stale
- **D-19:** GSD names only in output (Phase 2, Plan 02-01). bd IDs are implementation details, not shown to users.
- **D-20:** gsdw enforces ownership — warn if bd mutates gsd-labeled beads directly (detect via label check on read)

### Claude's Discretion
- bd wrapper Go architecture (client struct, exec.Command patterns, error handling)
- Bead creation timing (batch at plan-phase vs incremental at each step)
- Claiming mechanics (gsdw claim vs auto-claim in gsdw ready)
- Index file schema and rebuild strategy
- JSON parsing approach for bd --json output
- Error messages when bd is not installed or database not initialized

</decisions>

<specifics>
## Specific Ideas

- `gsdw ready` tree mockup (from discussion):
  ```
  Ready Work (3 tasks, 7 remaining)

    Phase 2: Graph Primitives
    ├─ Plan 02-01: bd CLI wrapper      [INFRA-03]
    └─ Plan 02-02: Domain mapping      [MAP-01, MAP-02]

    Phase 10: Coexistence
    └─ Plan 10-01: .planning/ reader   [COMPAT-01]

  Total: 3 ready │ 4 queued │ 7 remaining
  ```
- Surface all phases by default in ready output, `--phase N` to filter
- Close notification: "Closed bd-a3f → 2 new tasks ready" — keeps orchestrator informed without polling

</specifics>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase 1 foundation
- `internal/cli/bd.go` — Existing bd passthrough subcommand (integration point for wrapper)
- `internal/cli/root.go` — Cobra root command structure
- `.planning/phases/01-binary-scaffold/01-01-SUMMARY.md` — What Phase 1 built, interfaces available

### Project context
- `.planning/PROJECT.md` — Core value, constraints, key decisions
- `.planning/REQUIREMENTS.md` — INFRA-03, MAP-01 through MAP-06 define this phase's deliverables
- `.planning/ROADMAP.md` §Phase 2 — Success criteria (5 items that must be TRUE)

### bd CLI reference
- `bd create --help` — All creation flags (type, parent, metadata, acceptance, deps, labels, context)
- `bd update --help` — All update flags (claim, status, metadata, labels)
- `bd ready --help` — Unblocked work query semantics
- `bd query --help` — Query language syntax for filtering

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/cli/bd.go` — bd passthrough with `DisableFlagParsing`. The wrapper will be a separate package (`internal/graph/`) that calls bd programmatically, not through the passthrough.
- `internal/logging/logging.go` — slog dual-format logging, reuse for wrapper debug output
- `internal/hook/dispatcher.go` — Pattern for JSON stdin/stdout handling, reusable for bd --json parsing

### Established Patterns
- Injected io.Reader/io.Writer for testability (from hook dispatcher)
- stderr-only logging, stdout reserved for protocol data
- `exec.Command` + `exec.LookPath` pattern (bd passthrough uses this)

### Integration Points
- `bd` CLI at `~/.local/bin/bd` — wrapper shells out to this
- `.gsdw/` directory (new) — local index and state
- `internal/cli/root.go` — new `ready` subcommand will be added here

</code_context>

<deferred>
## Deferred Ideas

- Direct Go import of beads library instead of CLI wrapper — Phase 6/optimization path
- MCP tool registration for graph operations — Phase 3
- Hook-triggered automatic bead state updates — Phase 4
- Token-aware context injection from bead data — Phase 9

</deferred>

---

*Phase: 02-graph-primitives*
*Context gathered: 2026-03-21*
