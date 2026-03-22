# Phase 12: Setup UX - Context

**Gathered:** 2026-03-22
**Status:** Ready for planning
**Source:** Auto-mode (Claude selected recommended defaults)

<domain>
## Phase Boundary

Developer can verify their environment and get guided to a working state in one command. `gsdw check-deps`, `gsdw setup` wizard, `gsdw doctor` health check. Delivers SETUP-01 through SETUP-05.

</domain>

<decisions>
## Implementation Decisions

### Design principle
- **D-01:** gsdw is the developer interface. Setup commands are user-facing — clear, actionable output matters.
- **D-02:** Follow the `brew doctor` / `rustup check` pattern: detect → report → guide. Never silently install.

### check-deps
- **D-03:** `gsdw check-deps` reports `[OK]`/`[WARN]`/`[FAIL]` for: bd, dolt, Go, container runtime (Docker/Podman/Apple Container)
- **D-04:** Checks `$(go env GOPATH)/bin` in addition to PATH for bd and dolt (avoids false negatives after `go install`)
- **D-05:** For each missing dependency, prints the install command (brew, go install, or binary download URL)

### setup wizard
- **D-06:** `gsdw setup` is interactive: runs check-deps first, then for each missing dep offers install methods
- **D-07:** Install methods offered: brew (if available), `go install`, manual binary download
- **D-08:** After deps are satisfied, setup proceeds to container runtime selection and connection config (Phases 13-14 commands)

### doctor
- **D-09:** `gsdw doctor` is read-only — checks everything, modifies nothing. Same pattern as `brew doctor`.
- **D-10:** Phase 12 doctor reports: dependencies, .beads/ directory, .gsdw/ config. Container status added in Phase 13, Dolt server reachability added in Phase 14.
- **D-11:** Output is structured with `[OK]`/`[WARN]`/`[FAIL]` markers, scannable at a glance

### Claude's Discretion
- Exact dependency version requirements and how to check them
- Output formatting and color usage
- Whether setup auto-runs doctor at the end
- Error message wording for each failure scenario

</decisions>

<specifics>
## Specific Ideas

- check-deps is the foundation — both setup and doctor reuse its detection logic
- Build check-deps first, then doctor (wraps check-deps + more), then setup (wraps check-deps + interactive install)
- The `exec.LookPath` + version command pattern is established from Phase 1 (bd detection in cli/bd.go)

</specifics>

<canonical_refs>
## Canonical References

### Existing code
- `internal/cli/root.go` — Cobra command registration
- `internal/cli/bd.go` — exec.LookPath pattern for bd detection
- `internal/graph/client.go` — NewClient with LookPath (reusable detection)
- `internal/cli/ready.go` — Tree-formatted output pattern

### Requirements
- `.planning/REQUIREMENTS.md` — SETUP-01 through SETUP-05
- `.planning/ROADMAP.md` §Phase 12 — Success criteria

### Research
- `.planning/research/FEATURES.md` — CLI install UX patterns
- `.planning/research/SUMMARY.md` — Dependency detection approach

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `exec.LookPath` pattern from `internal/graph/client.go`
- Cobra subcommand pattern from all existing CLI commands
- Tree/table output from `internal/cli/ready.go`

### Integration Points
- `internal/cli/root.go` — New: check-deps, setup, doctor subcommands
- New package: `internal/deps/` — Dependency detection logic (shared by all three commands)

</code_context>

<deferred>
## Deferred Ideas

- Container runtime detection (Phase 13)
- Connection configuration (Phase 14)
- Auto-update checking

</deferred>

---

*Phase: 12-setup-ux*
*Context gathered: 2026-03-22 via auto-mode*
