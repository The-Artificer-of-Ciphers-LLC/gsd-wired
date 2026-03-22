# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-03-22)

**Core value:** GSD's full development lifecycle running on a beads graph engine so that orchestrator context stays lean and subagents pull only the context they need.
**Current focus:** v1.0 Installation Toolkit — Phase 12: Setup UX

## Current Position

Phase: 12 — Setup UX
Plan: 2 of 2 in current phase
Status: Complete
Last activity: 2026-03-22 -- Phase 12 Plan 02 complete (gsdw setup interactive wizard)

Progress: [████░░░░░░] 40% (Installation Toolkit milestone)

## Performance Metrics

**Velocity (v1.0 core):**
- Total plans completed: 22
- Average duration: 4.3 min
- Total execution time: ~1.6 hours

**By Phase (v1.0 core):**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 1. Binary Scaffold | 2 | 14 min | 7 min |
| 2. Graph Primitives | 2 | 6 min | 3 min |
| 3. MCP Server | 2 | 11 min | 5.5 min |
| 4. Hook Integration | 3 | 14 min | 4.7 min |
| 5. Project Init | 2 | 10 min | 5 min |
| 6. Research + Planning | 2 | 9 min | 4.5 min |
| 7. Execution + Verification | 3 | 12 min | 4 min |
| 8. Ship + Status | 2 | 9 min | 4.5 min |
| 9. Token-Aware Context | 2 | 10 min | 5 min |
| 10. Coexistence | 2 | 9 min | 4.5 min |

**Recent Trend:**
- Last 5 plans: 2 min, 4 min, 8 min, 2 min, 5 min
- Trend: stable

*Updated after each plan completion*

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- [Roadmap v1.0 Install]: 4 phases (11-14) derived from 24 requirements at fine granularity
- [Roadmap v1.0 Install]: Phase 11 (Distribution) has no source dependencies — GoReleaser config only
- [Roadmap v1.0 Install]: Phase 12 before Phase 13 — container subcommand reuses dependency detection infrastructure
- [Roadmap v1.0 Install]: Phase 13 combines Docker, Podman, and Apple Container into one phase (runtime abstraction unifies all three)
- [Roadmap v1.0 Install]: CNTR-05/CNTR-06 (Apple Container) in Phase 13, not a separate phase — same container subcommand, additive runtime branch
- [Research]: beads bug #2073 confirmed — must inject BEADS_DOLT_SERVER_* as env vars directly, never rely on config.yaml port
- [Research]: Use `homebrew_casks` (not `brews`) in GoReleaser — `brews` deprecated since v2.10, removed in v3
- [Research]: Apple Container requires macOS 26 + Apple Silicon — gate with sw_vers check before any container run attempt
- [Research]: `gsdw.compose.yaml` fragment (not compose.override.yml) — never modify user's existing compose files
- [Research]: volume mount before bd init pitfall — setup must check .beads/dolt/ exists and run bd init if needed
- [Research]: dockers_v2 is alpha — pin GoReleaser to v2.14.3 in GitHub Actions, do not use latest
- [11-01]: Used stable dockers section (not dockers_v2 alpha) — goreleaser not installed locally, stable approach safer
- [11-01]: ldflags vars are package-level (not unexported) — allows TestLdflagsOverride to set directly without build flags
- [11-01]: signs section retained with GPG config — can be removed if GPG not available at release time
- [11-02]: Token name is GSDWHOMEBREW in release.yml — matches .goreleaser.yaml .Env.GSDWHOMEBREW reference exactly (plan template had wrong name)

Prior v1.0 decisions (preserved for continuity):
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
- [06-01]: run_research uses CreatePhase for epic (gsd:research label) + CreatePlan for each of 4 fixed topics (gsd:research-child label)
- [06-01]: synthesize_research falls back to first research epic if phase num not found in metadata — test hermetic with fake_bd
- [06-01]: fake_bd updated to return canned research epic for label=gsd:research queries — keeps test infrastructure hermetic
- [06-01]: research CLI stub follows init pattern — full orchestration belongs in SKILL.md, not CLI (requires Claude Code Task() tool)
- [06-01]: SKILL.md subagent prompts are minimal (bead ID + topic + 4 instructions) — avoids context bloat per Pitfall 2
- [06-02]: create_plan_beads uses iterative topological sort (remaining-list pass, not recursive) — avoids stack overflow for deep chains (Pitfall 3)
- [06-02]: CreatePlanWithMeta added to graph.Client — extends CreatePlan with complexity/files metadata; existing CreatePlan unchanged for backward compatibility
- [06-02]: plan CLI stub follows same pattern as research/init — full orchestration via SKILL.md Task(), not CLI
- [06-02]: SKILL.md validation loop capped at 3 iterations with explicit iteration tracking per D-11/D-12
- [07-01]: execute_wave reports wave=1 always in v1 — dynamic wave computation deferred to v2
- [07-01]: verify_phase "failed" array contains raw criterion text for SKILL.md remediation per D-10; does NOT call create_plan_beads
- [07-01]: go_test method uses exec.CommandContext with 60-second timeout (Pitfall 5)
- [07-01]: FAKE_BD_QUERY_PHASE_RESPONSE env var added to fake_bd query subcommand for hermetic phase bead injection in tests
- [07-01]: verify_phase.go written fully in Task 1 (not as stub) since tools.go required verifyPhaseArgs/handleVerifyPhase to compile
- [07-02]: execute and verify CLI stubs follow identical pattern to plan.go — no new patterns introduced
- [07-02]: Both commands wired into root.go AddCommand chain in same line as all other commands
- [07-03]: execute SKILL.md uses plan_id (not bead_id) in commit messages and developer output per D-06/D-01
- [07-03]: verify SKILL.md uses detail field (not raw criterion) in remediation task titles per Pitfall 4
- [07-03]: ready has disable-model-invocation: false — lightweight informational, no agent spawning unlike execute/verify/plan/research
- [07-03]: Remediation task id pattern is {phase}-fix-{N} (e.g., 07-fix-1) for uniqueness and traceability
- [08-01]: reqPattern defined locally in create_pr_summary.go — reqLabelPattern is in cli package, avoid cross-package coupling
- [08-01]: advance_phase reuses phaseNumFromMeta from execute_wave.go (same mcp package) — no duplication needed
- [08-01]: CompletedPhases populated in existing phase bead loop in get_status — single QueryByLabel, zero extra graph I/O
- [08-01]: NextPhase uses pre-queried phases list after ClosePlan — avoids extra QueryByLabel call post-close
- [08-02]: ship.go follows exact execute.go/verify.go stub pattern — consistency with existing CLI commands
- [08-02]: SKILL.md no-changes-to-ship path still calls advance_phase — phase state always advances even without commits
- [08-02]: Error handling stops at gh failure before advance_phase — PR creation and phase advancement atomic from user perspective
- [09-01]: count-based warm (last N closed by ClosedAt desc) over time-based threshold — deterministic, project-lifecycle independent
- [09-01]: classifyTier takes warmIDs map[string]bool (not time.Time) — avoids non-deterministic tests
- [09-01]: tier types, pure functions, CompactBead, QueryTiered all in new tier.go — co-located with Bead type in graph package
- [09-01]: CompactBead only called inside ClosePlan post-close — structural guard against compacting open beads
- [09-01]: FAKE_BD_QUERY_TIERED_RESPONSE env var added to fake_bd query subcommand for QueryTiered test isolation
- [09-02]: buildSessionContext becomes thin wrapper (calls buildBudgetContext with 2000 token default) — existing call site unchanged
- [09-02]: Exported EstimateTokens/FormatHot/FormatWarm/FormatCold added to tier.go as wrappers — hook package calls graph package without duplicating logic
- [09-02]: extractCompact in execute_wave.go reads Metadata['gsd:compact'] first, falls back to CloseReason — backward compatible
- [09-02]: get_tiered_context (tool 18) defaults budget_tokens to 2000 when 0/omitted — consistent with SessionStart default
- [10-01]: Package-level compiled regexp patterns in compat package — same convention as reqLabelPattern from 02-02
- [10-01]: Pure parse functions (string in, struct out) with all file I/O in BuildFallbackStatus — makes parsers trivially testable
- [10-01]: ParseRoadmap two-pass design: first pass extracts phase rows, second pass enriches with goals from Phase Details
- [10-01]: All parsers return partial results on malformed/empty input — non-fatal by design (fallback path must be resilient)
- [10-01]: Zero write operations in compat package enforced structurally — D-09 compliance verified
- [10-02]: formatFallbackContext in session_start.go emits compatibility mode prefix, project name, core value, phase/plan counters, phase checkbox list
- [10-02]: fallbackStatusResult maps FallbackStatus to statusResult — OpenPhases counts !Complete phases, ReadyTasks stays empty (no graph)
- [10-02]: state.beadsDir set during sync.Once even on failure — DetectPlanning correctly receives project root for .planning/ check
- [12-01]: Tests use isolated temp-dir-only PATH to prevent real system binaries (bd at ~/.local/bin, docker) from interfering with hermetic tests
- [12-01]: lookInGoPath uses exec.LookPath("go") first so tests can inject a fake go binary via PATH, not exec.Command("go") directly
- [12-01]: renderDoctor delegates dep rendering inline — avoids coupling indentation style between check-deps and doctor commands
- [12-01]: checkContainerRuntime iterates docker-then-podman slice — clean extension point for Apple Container in Phase 13
- [12-02]: depInstallOptions map keyed on binary name — clean lookup for per-dep install methods without switch statements
- [12-02]: brewAvailable passed as bool parameter to runSetup — testable without PATH manipulation
- [12-02]: numbered menu options built dynamically (brew first if available, then go install, then download) — order deterministic
- [12-02]: printNextSteps extracted as separate function — both all-OK and missing-dep paths share same guidance block

### Pending Todos

None.

### Blockers/Concerns

- dockers_v2 is alpha in GoReleaser v2.14.3 — pin version in GitHub Actions, do not use latest tag
- Apple Container requires macOS 26 + Apple Silicon — narrow audience, must gate explicitly with sw_vers check
- beads bug #2073 — BEADS_DOLT_SERVER_* env var injection is the required workaround, not config.yaml

## Session Continuity

Last session: 2026-03-22 (Phase 12 Plan 02 execution — completed)
Stopped at: Completed 12-02-PLAN.md. gsdw setup interactive wizard implemented and tested. Phase 12 complete.
Resume file: None
