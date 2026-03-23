# Milestones

## v1.1 — Installation Toolkit

**Shipped:** 2026-03-22
**Phases:** 4 (11-14) | **Plans:** 8 | **Tests:** 342 total (across 11 packages)
**LOC added:** ~3,069 | **Total project:** 16,922 Go LOC

### Key Accomplishments

1. **GoReleaser release pipeline** — cross-platform binaries, GPG-signed checksums, multi-arch Docker images on ghcr.io, Homebrew cask
2. **Dependency detection and setup wizard** — `gsdw check-deps`, `gsdw setup`, `gsdw doctor` with install guidance
3. **Container runtime abstraction** — Docker, Podman, Apple Container with `gsdw container start/stop` and compose fragment
4. **Connection wizard and health check** — `gsdw connect` with auto-detection, remote fallback, two-phase TCP+SQL health check
5. **Env var injection** — BEADS_DOLT_SERVER_HOST/PORT into every bd subprocess from connection.json

### Archive

- `milestones/v1.1-ROADMAP.md` — Full roadmap with 4 phase details
- `milestones/v1.1-REQUIREMENTS.md` — 24 requirements with traceability

---

## v1.0 — gsd-wired

**Shipped:** 2026-03-22
**Phases:** 10 | **Plans:** 22 | **Tests:** 220
**Go files:** 77 | **LOC:** 12,373 | **Commits:** 129

### Key Accomplishments

1. **Single Go binary** (`gsdw`) serving as MCP server, hook dispatcher, and CLI tool with strict stdout discipline
2. **bd CLI wrapper** with full CRUD, wave computation, and local index — bd is an invisible implementation detail
3. **18 MCP tools** from basic graph operations to budget-aware tiered context, all with lazy Dolt initialization
4. **All 4 Claude Code hooks** — SessionStart (context loading), PreCompact (state buffer), PreToolUse (context injection), PostToolUse (state updates)
5. **8 slash commands** covering the full GSD lifecycle: init → research → plan → execute → verify → ship → status → ready
6. **Token-aware context routing** — hot/warm/cold bead tiering, budget tracking, automatic compaction on close
7. **.planning/ coexistence** — existing GSD users can adopt gsd-wired gradually with read-only fallback parsing

### Archive

- `milestones/v1.0-ROADMAP.md` — Full roadmap with 10 phase details
- `milestones/v1.0-REQUIREMENTS.md` — 56 requirements with traceability
- `milestones/v1.0-MILESTONE-AUDIT.md` — Final audit (passed, 56/56)
