# Milestones

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
