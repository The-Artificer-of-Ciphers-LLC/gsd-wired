# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-03-21)

**Core value:** GSD's full development lifecycle running on a beads graph engine so that orchestrator context stays lean and subagents pull only the context they need.
**Current focus:** Phase 5: Project Initialization

## Current Position

Phase: 5 of 10 (Project Initialization)
Plan: 2 of 2 in current phase
Status: Executing
Last activity: 2026-03-21 -- Phase 5 Plan 02 complete (SKILL.md slash commands + CLI subcommands)

Progress: [████████░░] 44%

## Performance Metrics

**Velocity:**
- Total plans completed: 10
- Average duration: 4.4 min
- Total execution time: 0.65 hours

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 1. Binary Scaffold | 2 | 14 min | 7 min |
| 2. Graph Primitives | 2 | 6 min | 3 min |
| 3. MCP Server | 2 | 11 min | 5.5 min |
| 4. Hook Integration | 3 | 14 min | 4.7 min |
| 5. Project Init | 2 | 10 min | 5 min |

**Recent Trend:**
- Last 5 plans: 4 min, 7 min, 5 min, 4 min, 2 min
- Trend: stable

*Updated after each plan completion*

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- [Roadmap]: 10 phases derived from 56 requirements at fine granularity
- [Roadmap]: Phases 9 and 10 can run in parallel (independent dependencies)
- [Roadmap]: bd CLI wrapping for v1, direct Go import is Phase 6/optimization path
- [01-01]: go-sdk v1.4.1 used as official MCP library — authoritative, Google co-maintained
- [01-01]: runtime/debug.ReadBuildInfo() for version hash — works with go install, no ldflags required
- [01-01]: Injected io.Reader/io.Writer in hook.Dispatch() — testable without os pipe mocking
- [01-01]: Pre-logger slog default in main() before Execute() — prevents stdout pollution
- [01-02]: No minAppVersion in plugin.json — per D-12, no minimum Claude Code version pinned
- [01-02]: StdoutPipe + bufio.Scanner in goroutine over bytes.Buffer — race-free subprocess stdout reading pattern
- [01-02]: .gitignore must use /gsdw (root-anchored) not gsdw — unanchored pattern matches cmd/gsdw directory
- [02-01]: NewClientWithPath() added as test injection point — bypasses LookPath for unit tests without live bd
- [02-01]: fake_bd uses FAKE_BD_CAPTURE_FILE env to write received args for exact arg verification in tests
- [02-01]: ListBlocked/QueryByLabel pass --limit 0 per research recommendation to prevent silent truncation
- [02-01]: ClosePlan post-close ListReady failure is best-effort — close succeeded, notification optional (D-13)
- [02-02]: renderReadyTree/renderReadyJSON extracted as pure functions — testable with constructed Bead slices, no fake bd needed
- [02-02]: reqLabelPattern compiled once at package level — avoids per-call regexp compilation cost
- [02-02]: phaseNumFromBead uses type switch — JSON unmarshal produces float64 for numbers; int variants handle direct construction in tests
- [02-02]: ASCII tree chars (|-- / +--) per plan spec — safer than unicode box-drawing for all terminals
- [03-01]: runWrite() as separate method from run() — write ops get batch flag, reads never do, clean separation
- [03-01]: FlushWrites uses run() not runWrite() — the dolt commit itself is not a batched operation
- [03-01]: initTimeout int field (ms) in serverState — allows test-configurable timeout without changing 30s default
- [03-01]: fake_bd strips leading --flag args before dispatch — enables testing global bd flags without special-casing
- [03-02]: NewInMemoryTransports for tool handler tests — faster than subprocess, avoids bd dependency for protocol-level tests
- [03-02]: toolError/toolResult helpers — consistent IsError=true pattern, eliminates repeated Content slice construction
- [03-02]: closeResult struct — explicit JSON shape for wave-awareness (who gets unblocked when a plan closes)
- [04-01]: hookState validates bdPath existence on init (os.Stat) so init() returns error immediately rather than deferring to first graph call
- [04-01]: buildSessionContext always returns string, never error — partial context better than nothing, errors logged to slog.Warn
- [04-01]: handleSessionStart sets hs.beadsDir from input.CWD before init() — CWD drives the beads directory for hook context
- [04-01]: Dispatcher creates fresh hookState per invocation — hooks are short-lived, no state reuse needed
- [04-02]: PreCompact writes to .gsdw/precompact-snapshot.json atomically via temp+rename — no goroutines (research Pitfall 2)
- [04-02]: PreToolUse fast path for read-class tools exits before any graph or file I/O — zero overhead for Read/Glob/Grep
- [04-02]: PreToolUse loads .gsdw/index.json as cheap context source (<1ms) before attempting 400ms graph query
- [04-02]: PostToolUse records Write/Edit/Bash to JSONL (Agent excluded) — no additionalContext injection (deferred to v2/TOKEN-A01)
- [04-03]: syncPendingSnapshot uses QueryByLabel not LoadIndex to find phase bead — active open phase surfaced directly from graph
- [04-03]: Snapshot file deletion is proof of sync — os.Remove only runs after UpdateBeadMetadata succeeds
- [04-03]: updateBeadOnToolUse uses AddLabel(gsd:tool-use) not UpdateBeadMetadata — minimal change surface, satisfies INFRA-08
- [05-01]: phaseNum=0 convention for project-level epic bead — CreatePhase accepts any int so 0 works without schema changes
- [05-01]: Single init_project tool for bead creation + file writing — simpler SKILL.md, eventual consistency acceptable
- [05-01]: Context child bead failures non-fatal in handleInitProject — partial context better than init failure
- [05-01]: get_status replicates buildSessionContext query pattern directly — avoids hook package coupling in MCP server
- [05-02]: SKILL.md files placed at plugin root skills/ (not inside .claude-plugin/) — auto-discovered as /gsd-wired:name slash commands
- [05-02]: disable-model-invocation: true on init SKILL.md — user must explicitly invoke /gsd-wired:init
- [05-02]: renderStatus extracted as pure function (io.Writer, phases, ready) — testable without graph client, same pattern as renderReadyTree

### Pending Todos

None yet.

### Blockers/Concerns

- bd is confirmed on PATH at ~/.local/bin/bd (blocker resolved in practice)
- Go 1.26.1 installed at /opt/homebrew/bin/go (satisfies go-sdk v1.4.1 requirement of Go 1.25+)

## Session Continuity

Last session: 2026-03-21 (Phase 5 Plan 02 — slash commands + CLI)
Stopped at: Phase 5 Plan 02 complete — SKILL.md slash commands (/gsd-wired:init, /gsd-wired:status) and gsdw init/status CLI subcommands created. 7 packages pass go test ./... -race.
Resume file: None
