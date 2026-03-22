# Phase 1: Binary Scaffold - Context

**Gathered:** 2026-03-21
**Status:** Ready for planning

<domain>
## Phase Boundary

A single Go binary (`gsdw`) that runs as MCP server, hook dispatcher, CLI tool, and bd passthrough with correct plugin registration. Delivers INFRA-01, INFRA-04, INFRA-09. No graph operations, no real hook logic — just the skeleton that all later phases build on.

</domain>

<decisions>
## Implementation Decisions

### CLI command tree
- **D-01:** Binary name is `gsdw`
- **D-02:** Subcommands: `serve` (MCP stdio), `hook <event>` (dispatcher), `bd <args>` (passthrough to bd CLI with GSD context), `version`
- **D-03:** `hook <event>` parses and validates JSON from stdin even at stub stage — catches integration issues early
- **D-04:** Version output format: `0.1.0 (abc1234)` — semver + git commit hash

### Go module and project layout
- **D-05:** Module path: `github.com/The-Artificer-of-Ciphers-LLC/gsd-wired`
- **D-06:** Standard Go layout: `cmd/gsdw/main.go` + `internal/` packages
- **D-07:** `.claude-plugin/` directory at repo root alongside `go.mod`
- **D-08:** Installable via `go install github.com/The-Artificer-of-Ciphers-LLC/gsd-wired/cmd/gsdw@latest`

### Plugin manifest
- **D-09:** Manifest grows per phase — only register what the current phase delivers (no forward-declared stubs)
- **D-10:** Slash command prefix: `/gsd-wired:` (longer form for clarity, not `/gsdw:`)
- **D-11:** Display name: "GSD Wired" / Description: "Token-efficient development lifecycle on a versioned graph"
- **D-12:** No minimum Claude Code version pinned

### Logging
- **D-13:** Dual format: JSON and human-readable, selectable by flag
- **D-14:** `--log-level` flag with levels: error (default), info, debug
- **D-15:** Serve mode is silent unless errors occur — no chatty connection/tool-call logs
- **D-16:** Use `log/slog` from stdlib — zero external dependencies, structured, leveled, built-in JSON + text handlers

### Claude's Discretion
- Cobra command wiring details and help text
- Internal package boundaries (how to split `internal/`)
- Hook event name constants and validation logic
- Build system for embedding version/commit at compile time

</decisions>

<specifics>
## Specific Ideas

- `gsdw bd ...` passthrough should add GSD context (project database, flags) transparently — user types bd commands, gsdw enriches them
- Hook stdin parsing should validate structure even when handlers are no-ops, so protocol mismatches surface immediately during development
- "The wire threading through beads" — the binary is the wire, everything else plugs into it

</specifics>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

No external specs — requirements are fully captured in decisions above and in:

### Project context
- `.planning/PROJECT.md` — Core value, constraints, key decisions (Go language, bd hard dependency, hybrid state model)
- `.planning/REQUIREMENTS.md` — INFRA-01, INFRA-04, INFRA-09 define this phase's deliverables
- `.planning/ROADMAP.md` §Phase 1 — Success criteria (4 items that must be TRUE)

### Research (from prior session)
- `.planning/research/SUMMARY.md` — Stack decisions, architecture patterns, pitfall warnings (Dolt write amplification, PreCompact limitations)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- None — codebase is empty, this is the first phase

### Established Patterns
- None yet — this phase establishes the foundational patterns all later phases follow

### Integration Points
- `bd` CLI at `~/.local/bin/bd` — passthrough target
- `dolt` CLI at `/opt/homebrew/bin/dolt` — underlying database (not called directly by gsdw, bd wraps it)
- Go 1.26.1 at `/opt/homebrew/bin/go` — build toolchain
- Claude Code plugin system — consumes `.claude-plugin/plugin.json`

</code_context>

<deferred>
## Deferred Ideas

- MCP tool registration (Phase 3)
- Real hook logic with bead state persistence (Phase 4)
- Slash command implementations (Phase 5+)
- Token-aware context injection in hooks (Phase 9)

</deferred>

---

*Phase: 01-binary-scaffold*
*Context gathered: 2026-03-21*
