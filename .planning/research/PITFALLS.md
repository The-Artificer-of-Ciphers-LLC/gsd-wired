# Pitfalls Research

**Domain:** AI agent orchestration plugin (Claude Code + Beads/Dolt graph persistence)
**Researched:** 2026-03-21
**Confidence:** HIGH (official docs verified most claims)

## Critical Pitfalls

### Pitfall 1: Hook Output Pollution Kills JSON Parsing

**What goes wrong:**
Claude Code hook handlers (command type) parse stdout as JSON. If a Go binary, shell profile, or bd subprocess prints *anything* to stdout besides the JSON response object, the hook fails with "JSON validation failed." This is especially insidious because Go's `log` package defaults to stderr (safe), but `fmt.Println` goes to stdout (fatal). A single debug print statement or Dolt startup banner breaks the entire hook chain.

**Why it happens:**
Developers test hooks in isolation where stdout goes to a terminal. In production, Claude Code consumes stdout as structured data. The Go binary wrapping bd may capture bd's own stdout output (progress messages, warnings) and accidentally forward it.

**How to avoid:**
- All Go hook handlers MUST write JSON to stdout and everything else to stderr
- Wrap bd CLI calls with stdout capture/suppression -- only parse bd's JSON output, never forward raw
- Use `os.Stderr` explicitly for all logging in hook binaries
- Add a CI test that runs each hook handler and validates stdout is parseable JSON

**Warning signs:**
- Hooks work in manual testing but fail when Claude Code invokes them
- "JSON validation failed" errors in Claude Code verbose mode
- Intermittent hook failures (only when bd emits warnings)

**Phase to address:**
Phase 1 (foundation) -- establish the hook binary scaffold with stdout discipline from day one

---

### Pitfall 2: PreCompact Hook Race Condition Loses State

**What goes wrong:**
PreCompact fires before context compaction (both manual `/compact` and automatic when context fills). The hook must save in-progress state to the beads graph. But if the bd CLI write is slow (Dolt commit takes 200-500ms on larger repos) and the compaction timeout fires first, state is lost. Worse: PreCompact hooks **cannot block** compaction -- they are non-blocking events (exit code 2 is ignored). The hook runs but compaction does not wait for it to finish.

**Why it happens:**
Developers assume PreCompact is a gate that blocks until the hook completes. The [official docs](https://code.claude.com/docs/en/hooks) explicitly list PreCompact as non-blocking. If the hook is slow, compaction proceeds and the saved state may be incomplete or not committed.

**How to avoid:**
- Make PreCompact hooks async-safe: write state to a local buffer/WAL file first (fast), then sync to Dolt asynchronously
- Keep PreCompact handler under 100ms by writing to a staging table or temp file, not doing a full Dolt commit
- Use PostCompact (also non-blocking but fires after) to verify the save completed and retry if needed
- Consider using SessionStart to reconcile any incomplete PreCompact saves from the previous session

**Warning signs:**
- State occasionally missing after compaction events
- Dolt commit logs show gaps during long sessions
- PreCompact handler times reported > 200ms in logs

**Phase to address:**
Phase 2 (hook integration) -- design the two-stage save pattern before building any hook logic

---

### Pitfall 3: Bag-of-Agents Error Amplification

**What goes wrong:**
GSD spawns 4 parallel research agents, wave-based execution agents, and a verification agent. Without strict coordination topology, errors compound across agents -- [research shows](https://towardsdatascience.com/why-your-multi-agent-system-is-failing-escaping-the-17x-error-trap-of-the-bag-of-agents/) a 17x error amplification in unstructured multi-agent systems. An agent creates a malformed bead, a downstream agent reads it and makes a bad decision, the orchestrator aggregates garbage.

**Why it happens:**
The "just spawn more subagents" pattern feels productive but without defined handoff contracts (what data format each agent produces, what the next agent expects), you get nondeterministic results. Claude Code subagents cannot spawn sub-subagents, which helps prevent infinite chains, but the orchestrator still must validate each agent's output before feeding it to the next.

**How to avoid:**
- Define explicit bead schemas for each agent type (research agent output bead, execution agent output bead)
- Orchestrator validates bead content after each agent closes its work -- never pass unchecked beads downstream
- Use bd's `relates_to` links to create explicit handoff edges in the graph, not implicit ordering
- Limit parallel agents to the minimum needed (4 research agents is fine because they are independent; wave execution needs sequential validation gates)

**Warning signs:**
- Downstream agents producing obviously wrong output
- Bead graph has orphaned or contradictory entries
- Verification phase consistently catches errors that should have been prevented earlier

**Phase to address:**
Phase 3 (orchestration) -- define agent contracts and validation gates before enabling multi-agent workflows

---

### Pitfall 4: Dolt Write Amplification on Every Hook

**What goes wrong:**
Every PostToolUse hook fires a bd update, which triggers a Dolt write and potentially a commit. In an active session, Claude might execute 50-100 tool calls. That is 50-100 Dolt writes, each with version control overhead. The database grows rapidly, hook latency increases, and auto-stats (Dolt's background indexer, refreshing every 30s) burns CPU scanning an ever-growing data directory.

**Why it happens:**
The natural mapping of "every tool call updates state" seems correct but ignores Dolt's write cost. Unlike MySQL, every Dolt write creates a new chunk in the content-addressed store. [Production reports](https://gist.github.com/l0g1x/ef6dc1a971fa124e8d5939f3115b4e7d) show Dolt auto-stats at 30s intervals causing perpetual CPU burn on repos > 2GB.

**How to avoid:**
- Batch bead updates: accumulate state changes in memory during a wave, commit once at wave boundaries
- Disable Dolt auto-stats or set interval to 300s+ for local-only usage (`dolt config --local --add sqlserver.global.dolt_stats_auto_refresh_interval 300`)
- Use PostToolUse hooks selectively via matchers -- only update beads for significant tool calls (Write, Edit, Bash), not reads (Read, Glob, Grep)
- Implement a write coalescing layer in Go between the hooks and bd CLI

**Warning signs:**
- Session latency increases over time (hooks taking longer)
- Dolt data directory growing > 500MB for a single project
- CPU usage spikes every 30s from dolt stats process
- Hook timeouts in later parts of long sessions

**Phase to address:**
Phase 2 (persistence layer) -- implement batched writes before connecting hooks to bd

---

### Pitfall 5: MCP Server Startup Latency Blocks Session

**What goes wrong:**
The MCP server (Go binary) starts as a subprocess when Claude Code loads the plugin. If the server needs to initialize a Dolt database, check schema migrations, or warm caches, it blocks Claude Code's session startup. Users see a hanging cursor for 3-10 seconds. If the MCP server crashes on startup (missing Dolt binary, corrupted database), the entire plugin fails silently or with a cryptic error.

**Why it happens:**
Go binaries start fast (~50ms), but Dolt database initialization is slow. Opening a Dolt database, running schema checks, and verifying the beads graph integrity can take seconds. The stdio transport means Claude Code is waiting for the MCP `initialize` response.

**How to avoid:**
- Lazy initialization: respond to MCP `initialize` immediately, defer Dolt connection to first tool call
- Validate Dolt/bd availability in SessionStart hook (which runs in parallel with session setup), not in MCP server startup
- Provide clear error messages when dependencies are missing -- do not just exit(1)
- Cache schema migration status in a local file to skip validation on subsequent starts

**Warning signs:**
- Session start takes > 2 seconds
- Users report "Claude Code hangs when starting" in projects with beads enabled
- MCP server logs show Dolt initialization dominating startup time

**Phase to address:**
Phase 1 (MCP server scaffold) -- implement lazy init from the start, never blocking on initialize

---

### Pitfall 6: Context Budget Miscalculation Starves Orchestrator

**What goes wrong:**
The wire (orchestrator) loads project context from beads at SessionStart, injects bead context via PreToolUse, and accumulates PostToolUse state. Each injection consumes context window tokens. If the orchestrator is too generous with context injection (loading full bead descriptions, history, dependencies for every tool call), it fills its own 200K context window and triggers auto-compaction, which wipes the orchestration state it needs to coordinate agents.

**Why it happens:**
Token budgeting is invisible -- there is no API to query "how much context remains." Developers test with small projects (10 beads) where everything fits. At 50+ beads with rich descriptions, the math breaks. Claude Code's context editing feature (2026) helps by clearing stale tool outputs, but the orchestrator must still be conservative about what it injects.

**How to avoid:**
- Implement tiered context loading: hot beads get full context, warm beads get summary, cold beads get ID + title only
- Track approximate token count of injected context (rough heuristic: 1 token per 4 chars)
- Set a hard budget per injection (e.g., PreToolUse injects max 2000 tokens of bead context)
- Use bd's compaction feature to keep closed bead summaries small
- Rely on subagents for deep bead context -- orchestrator stays lean, subagent gets full bead content via bd show

**Warning signs:**
- Auto-compaction triggering during orchestration (not during execution)
- Orchestrator "forgetting" earlier decisions after compaction
- Session transcripts showing massive PreToolUse context injections

**Phase to address:**
Phase 3 (token-aware routing) -- this is the core feature; must be designed before any context injection logic

---

### Pitfall 7: hooks.json Duplication Detection Breaks Plugin Loading

**What goes wrong:**
Claude Code automatically loads `hooks/hooks.json` from the plugin directory. If the same hooks are also declared in `plugin.json` manifest or in the project's `.claude/settings.json`, Claude Code's deduplication logic either silently drops hooks or throws duplicate detection errors. The plugin loads but hooks do not fire.

**Why it happens:**
The [official docs](https://code.claude.com/docs/en/plugins) explicitly warn: "The manifest does NOT include a hooks field. Claude Code CLI v2.1+ automatically loads hooks/hooks.json by convention. Explicitly declaring it causes duplicate detection errors." Developers coming from other plugin systems expect to declare all components in the manifest.

**How to avoid:**
- Never declare hooks in plugin.json -- only in hooks/hooks.json
- Document this constraint in the plugin's developer guide
- Add a CI check that validates plugin.json does not contain a "hooks" key
- Test plugin loading with `claude --plugin-dir ./plugin` and verify hooks via `/hooks` browser

**Warning signs:**
- Hooks appear in configuration but never fire
- `/hooks` browser shows duplicate entries or missing hooks
- Plugin works in standalone mode but breaks as a packaged plugin

**Phase to address:**
Phase 1 (plugin scaffold) -- get the directory structure right from the first commit

---

### Pitfall 8: bd CLI Subprocess Overhead on Every MCP Tool Call

**What goes wrong:**
The MCP server wraps bd CLI by shelling out to `bd` for each tool invocation (bd create, bd update, bd ready, etc.). Each subprocess spawn has overhead: ~20-50ms for Go binary startup + Dolt connection. For a research phase spawning 4 agents that each make 10+ bd calls, that is 40+ subprocess invocations adding 1-2 seconds of pure overhead.

**Why it happens:**
Wrapping a CLI is the fastest path to integration. Calling `exec.Command("bd", "create", ...)` is simple Go code. But bd was designed for human-speed CLI usage, not machine-speed MCP tool invocation.

**How to avoid:**
- Phase 1: Start with CLI wrapping (it works, it is simple, ship it)
- Phase 2+: Import beads as a Go library (`github.com/steveyegge/beads`) instead of shelling out -- the Go package is [published on pkg.go.dev](https://pkg.go.dev/github.com/steveyegge/beads)
- If staying with CLI wrapping, implement a connection pool pattern: keep a long-running bd process with stdin/stdout IPC instead of spawning per-call
- Batch related operations (create + add dependency + update status = one bd call if possible)

**Warning signs:**
- MCP tool calls consistently taking > 100ms for simple operations
- Profiling shows majority of time in process spawn, not bd logic
- Agent sessions feel sluggish despite simple graph operations

**Phase to address:**
Phase 1 starts with CLI wrapping (acceptable), Phase 4+ migrates to library import (performance)

---

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| Shell out to bd CLI instead of importing Go package | Fast to build, no coupling to bd internals | 20-50ms overhead per call, fragile argument escaping | MVP only -- migrate to library by Phase 4 |
| Store all bead context as unstructured text | No schema to maintain, flexible | Cannot query or filter efficiently, token waste | Never -- define schemas for at least title, status, type, summary |
| Single Dolt commit per hook invocation | Simple, correct state after each operation | Write amplification, database bloat, CPU overhead | Never -- batch from the start |
| Hard-code token budgets | Works for known project sizes | Breaks on large projects, no adaptability | Prototype only -- add heuristic sizing in Phase 3 |
| Skip bd compaction of closed beads | Simpler implementation | Context windows fill with stale data, token waste | Never -- compaction is core to the value proposition |
| Synchronous MCP server initialization | Simpler startup code | 3-10s session start delay, poor UX | Never -- lazy init from Phase 1 |

## Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| Claude Code hooks + MCP server in same plugin | Hooks and MCP server competing for the same bd state, causing read-after-write inconsistency | Hooks write to a staging area; MCP server reads from committed state. Clear ownership of who writes what |
| bd CLI + Dolt database | Assuming bd operations are atomic -- a `bd create` + `bd dep add` is two separate Dolt commits with a gap between them | Use bd's batch operations where available, or wrap sequences in explicit Dolt transactions |
| Claude Code subagents + beads claims | Subagent claims a bead but crashes before closing it, leaving it permanently claimed | Implement claim timeout/expiry -- if a bead is claimed for > N minutes without update, auto-unclaim |
| SessionStart hook + MCP server startup | Hook fires before MCP server is ready, hook tries to call bd via MCP and gets connection refused | SessionStart hook should use direct bd CLI call (not MCP), since the hook is a command handler independent of MCP |
| PreToolUse context injection + tool matchers | Injecting bead context for every tool call including Read/Grep/Glob -- wastes tokens on pure read operations | Use matcher regex to only inject for Write, Edit, Bash -- operations that might change state |
| Dolt auto-sync + local-only v1 | Enabling Dolt remote sync accidentally, causing network calls on every commit | Explicitly disable remotes in v1: `dolt config --local --add core.denyRemotePush true` |

## Performance Traps

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| Dolt auto-stats on large data dirs | Perpetual 30s CPU spikes, 100% core usage | Set `dolt_stats_auto_refresh_interval` to 300+ seconds | Data dir > 500MB |
| Loading full bead graph at SessionStart | Session start > 5s, initial context 50%+ consumed | Load only open/ready beads at startup, lazy-load others | > 30 beads in project |
| PreToolUse hook on every tool call | Cumulative 500ms+ latency per session turn | Match only state-changing tools (Write, Edit, Bash) | > 20 tool calls per turn |
| Dolt commit per operation | Database grows 10-50MB/day during active development | Batch commits at wave boundaries | After 1 week of active use |
| Subprocess spawn per bd call | 1-2s overhead per research phase (4 agents x 10 calls) | Import beads Go package or use long-running process | > 5 concurrent agents |
| Uncompacted closed beads | Token budget consumed by irrelevant history | Run bd compaction after each phase completion | > 20 closed beads |

## Security Mistakes

| Mistake | Risk | Prevention |
|---------|------|------------|
| Storing secrets in bead descriptions | Dolt is version-controlled -- secrets persist in history even after deletion | Never store secrets in beads; use environment variables referenced by name only |
| MCP server listening on network interface | Other processes or network actors can invoke bd operations | Use stdio transport only (not HTTP) for local plugin; Dolt is local-only in v1 |
| Hook handlers with unrestricted file access | A malicious bead description could craft a path traversal in bd show output | Sanitize all bd output before using in file operations; never use bead content as file paths |
| bd CLI invoked with unsanitized arguments | Shell injection via bead titles or descriptions containing backticks/semicolons | Use `exec.Command` with argument arrays, never `exec.Command("sh", "-c", ...)` with string interpolation |

## UX Pitfalls

| Pitfall | User Impact | Better Approach |
|---------|-------------|-----------------|
| Silent hook failures | User thinks state is being saved but hooks are failing silently | Show status messages via hook `statusMessage` field; log errors visibly |
| Requiring Dolt + bd pre-installed | Installation friction kills adoption -- user must install 3 things (plugin + dolt + bd) | Plugin install script that checks for and offers to install dependencies |
| No fallback when beads not initialized | Plugin errors on every hook in non-beads projects | Check for beads init status; gracefully skip hooks in non-beads projects (coexistence mode) |
| Opaque token budget decisions | User cannot tell why orchestrator loaded some context but not other | Provide a `/gsd-wired:budget` command showing current token allocation and what was loaded/skipped |
| Forcing beads-only workflow | Existing GSD users lose access to .planning/ files they know | Hybrid mode: read .planning/ as fallback, gradually suggest migration to beads |
| Long MCP server startup | 3-10s hang when opening project | Lazy initialization; show "Connecting to beads..." status message |

## "Looks Done But Isn't" Checklist

- [ ] **SessionStart hook:** Often missing resume vs. startup distinction -- verify hook handles both `startup` and `resume` matchers correctly (resume should load incremental state, not full reload)
- [ ] **PreCompact save:** Often missing verification that the save actually completed -- verify PostCompact hook confirms state was persisted
- [ ] **Subagent bead claims:** Often missing unclaim-on-failure -- verify that a crashed subagent's bead is released (SubagentStop hook with failure detection)
- [ ] **Wave execution:** Often missing dependency cycle detection -- verify `bd ready` correctly handles circular deps (bd should handle this, but verify)
- [ ] **Token budget:** Often missing compaction-awareness -- verify that after auto-compaction, the orchestrator can reconstruct enough state from beads to continue (not just from context)
- [ ] **Plugin manifest:** Often missing version pinning for bd/Dolt compatibility -- verify plugin documents minimum bd and Dolt versions
- [ ] **Coexistence mode:** Often missing write-back -- verify that changes made via beads are reflected in .planning/ for users who check both (or explicitly document they are not)
- [ ] **Error messages:** Often missing actionable guidance -- verify that "Dolt not found" errors include installation instructions, not just "command not found"

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| Hook output pollution | LOW | Fix stdout/stderr routing in Go binary; no data loss |
| PreCompact state loss | MEDIUM | Reconcile from Dolt history (`dolt log` + `dolt diff`) to find last good state; replay missing operations |
| Agent error amplification | HIGH | Must re-run affected phase; add validation gates to prevent recurrence |
| Dolt write amplification / bloat | MEDIUM | Run `dolt gc` to garbage collect; implement batching; may need to re-clone if database too large |
| MCP startup latency | LOW | Refactor to lazy init; no data impact |
| Context budget exhaustion | MEDIUM | Session must be restarted; orchestrator loses accumulated state; implement budget tracking to prevent |
| hooks.json duplication | LOW | Remove duplicate declarations; reload plugins |
| Subprocess overhead | LOW | Performance improvement only; no data impact; migrate to library import |

## Pitfall-to-Phase Mapping

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| Hook output pollution | Phase 1 (foundation) | CI test: all hook handlers produce valid JSON on stdout |
| PreCompact race condition | Phase 2 (hook integration) | Integration test: trigger compaction during bd write, verify state survives |
| Bag-of-agents error amplification | Phase 3 (orchestration) | Run 4 parallel research agents on a test project; verify synthesizer gets valid data |
| Dolt write amplification | Phase 2 (persistence) | Benchmark: measure Dolt data dir growth over 100 simulated tool calls |
| MCP startup latency | Phase 1 (MCP scaffold) | Measure: session start to first tool availability < 500ms |
| Context budget starvation | Phase 3 (token routing) | Load test: 50-bead project, verify orchestrator context usage < 50% after SessionStart |
| hooks.json duplication | Phase 1 (plugin scaffold) | CI check: plugin.json does not contain "hooks" key; /hooks browser shows correct count |
| bd subprocess overhead | Phase 1 (ship it) -> Phase 4 (optimize) | Benchmark: measure MCP tool call latency; target < 50ms for simple operations by Phase 4 |

## Sources

- [Claude Code Hooks Reference](https://code.claude.com/docs/en/hooks) -- official hook event types, blocking behavior, JSON output format
- [Claude Code Plugin Docs](https://code.claude.com/docs/en/plugins) -- plugin structure, hooks.json convention, common mistakes
- [Beads Plugin Documentation](https://github.com/steveyegge/beads/blob/main/docs/PLUGIN.md) -- MCP server architecture, bd CLI integration
- [Beads Go Package](https://pkg.go.dev/github.com/steveyegge/beads) -- library import alternative to CLI wrapping
- [Dolt Production Investigation (2026)](https://gist.github.com/l0g1x/ef6dc1a971fa124e8d5939f3115b4e7d) -- auto-stats CPU burn, dropped database accumulation
- [17x Error Trap in Multi-Agent Systems](https://towardsdatascience.com/why-your-multi-agent-system-is-failing-escaping-the-17x-error-trap-of-the-bag-of-agents/) -- coordination topology vs. bag-of-agents
- [Claude Code Subagent Cost Analysis](https://www.aicosts.ai/blog/claude-code-subagent-cost-explosion-887k-tokens-minute-crisis) -- token economics of multi-agent workflows
- [Multi-Agent Anti-Patterns (Enterprise)](https://medium.com/@armankamran/anti-patterns-in-multi-agent-gen-ai-solutions-enterprise-pitfalls-and-best-practices-ea39118f3b70) -- uncontrolled agent interactions, chaotic memory management

---
*Pitfalls research for: AI agent orchestration plugin (Claude Code + Beads/Dolt graph persistence)*
*Researched: 2026-03-21*

---

# Distribution Milestone Pitfalls

**Milestone:** Installation toolkit — Homebrew tap, GoReleaser packaging, container support, dependency detection wizard, remote Dolt connectivity
**Researched:** 2026-03-21
**Confidence:** HIGH (GoReleaser official docs, Docker official docs, Homebrew docs verified)

---

## Critical Pitfalls

### Pitfall D1: GoReleaser `brews` Is Deprecated — Use `homebrew_casks` (Breaking)

**What goes wrong:**
GoReleaser v2.10+ deprecated the `brews` configuration key in favor of `homebrew_casks`. The `brews` key generated Homebrew Formulas, which are semantically wrong for distributing precompiled binaries (Formulas are meant for building from source). Starting a new project with `brews` means shipping via the wrong Homebrew mechanism, and migration later breaks existing users: when the formula is disabled in favor of the cask, `brew upgrade` throws an error for all existing installs.

**Why it happens:**
Every tutorial and StackOverflow answer written before v2.10 uses `brews`. GoReleaser's own examples in older README versions use `brews`. A developer bootstrapping from any doc older than mid-2024 will write the wrong key and it will work — until they try to migrate.

**Consequences:**
- Existing `brew install user/tap/gsdw` users get a broken upgrade path when you switch to casks
- Must add tap migration instructions and `tap_migrations.json` to the tap repo
- The Homebrew ecosystem only recently added formula-to-cask migration in the same tap (still in progress as of 2025)

**Prevention:**
- Use `homebrew_casks:` in `.goreleaser.yml` from day one — do not touch `brews:` at all
- Tap repo needs a `Casks/` directory (not `Formula/`)
- After first release: if any old Formula exists, add `conflicts_with formula: "user/tap/gsdw"` to the cask

**Detection:**
- `goreleaser check` warns about deprecated keys
- GoReleaser deprecation page lists exact removal schedule

**Phase to address:** Distribution Phase 1 (GoReleaser setup)

---

### Pitfall D2: GitHub Actions Token Cannot Push to a Separate Tap Repo

**What goes wrong:**
GoReleaser needs to push the updated Homebrew cask file to the tap repository (e.g., `user/homebrew-tap`). The default `GITHUB_TOKEN` auto-generated by GitHub Actions only has write access to the repository where the workflow runs — not to the tap repo. The publish step silently succeeds in GoReleaser's output but the cask file never updates in the tap.

**Why it happens:**
GoReleaser does not fail hard on cask publish errors — "if PR creation fails, the pipeline continues without halting — errors only appear in release logs." This makes it look like the release succeeded. Users who install from the tap get stale or missing cask files.

**Consequences:**
- Release appears successful in CI but tap is stale
- Users who `brew install user/tap/gsdw` get old binary or "formula not found"
- Without a proper bot token, every release requires manual tap update

**Prevention:**
- Create a dedicated GitHub bot account (or use a machine account) with push access to the tap repo
- Generate a PAT scoped to `contents: write` for the tap repo only (not the broad repo scope)
- Store as `HOMEBREW_TAP_GITHUB_TOKEN` secret and reference in `.goreleaser.yml` under `homebrew_casks.repository.token`
- Do not use a personal PAT with full repo scope — it grants access to all your repos

**Detection:**
- GoReleaser release log shows "failed to publish artifacts" but exit code is 0
- `git log` on the tap repo shows no new commits after a release

**Phase to address:** Distribution Phase 1 (CI/CD setup) — must be configured before first release

---

### Pitfall D3: macOS Gatekeeper Blocks Unsigned Go Binary Even From Homebrew

**What goes wrong:**
macOS Gatekeeper quarantines downloaded binaries that are not signed and notarized by an Apple Developer account. A Go binary distributed via Homebrew without code signing produces "gsdw is damaged and can't be opened. You should move it to the Trash." This is not a bug — Gatekeeper is doing its job. Users who do not know the `xattr` workaround are permanently blocked from using the tool.

**Why it happens:**
Gatekeeper attaches a `com.apple.quarantine` extended attribute to all binaries downloaded from the internet. If the binary lacks a valid Apple Developer signature, Gatekeeper refuses to run it. The "damaged" message is misleading — the binary is fine, it is just unsigned.

**Consequences:**
- First-run experience is broken for most macOS users unless they know the workaround
- Requires either Apple Developer subscription ($99/year) or documented manual override
- The error message does not hint at the solution; users assume the binary is corrupt

**Prevention (choose one):**
1. **Sign + notarize (recommended for public release):** Use GoReleaser's `signs:` block with `gon` or `rcodesign` to sign and notarize in CI. Requires Apple Developer account. Adds ~2-3 minutes to CI build.
2. **Post-install xattr hook in cask:** Add `postinstall do ... system "xattr", "-d", "com.apple.quarantine", "#{bin}/gsdw" end` to the cask definition. This is a known workaround used by many open source tools. Homebrew applies it as root so it works without user intervention.
3. **Document the workaround:** For internal/developer tools, document `xattr -d com.apple.quarantine $(which gsdw)`. Acceptable for developer tools, not for general distribution.

**Detection:**
- "damaged and can't be opened" error on fresh install on any macOS 13+ machine
- `spctl -a -vvv gsdw` output shows "rejected" with "no usable signature"

**Phase to address:** Distribution Phase 1 (Homebrew cask definition) — the xattr hook approach has zero cost and is the minimum viable prevention

---

### Pitfall D4: GoReleaser Config Missing `version: 2` Header Fails Silently in CI

**What goes wrong:**
GoReleaser v2 requires `version: 2` at the top of `.goreleaser.yml`. Without it, v2 treats the file as invalid and errors out. The error message is "only version: 2 configuration files are supported" but this often appears buried in CI logs. Old blog posts and GitHub starter templates do not include this header, so copy-paste from any pre-v2 resource produces a broken config.

**Why it happens:**
GoReleaser v2 was released mid-2024. Most tutorials, GitHub Actions marketplace examples, and community templates predate it and use the v1 config format (no `version:` key at all). The v1 format silently worked for years, making the version requirement feel like new friction.

**Additional breaking changes in v2 to know:**
- `--rm-dist` flag renamed to `--clean` — old GitHub Actions workflows using `--rm-dist` fail silently
- `archives.builds` renamed to `archives.ids`
- `archives.format` is now `archives.formats: []` (list)
- `dockers` config requires migration to `dockers_v2` for multi-platform builds

**Prevention:**
- Start `.goreleaser.yml` with `version: 2` as the first line
- Run `goreleaser check` locally before pushing; it validates the schema and reports deprecated keys
- Pin GoReleaser version in GitHub Actions (`goreleaser/goreleaser-action@v6`) to avoid unexpected v3 upgrades in the future

**Detection:**
- `goreleaser check` exits non-zero with "configuration is invalid"
- CI fails on release workflow with version-related error in first few lines of goreleaser output

**Phase to address:** Distribution Phase 1 (initial GoReleaser setup)

---

### Pitfall D5: CGO_ENABLED=1 Breaks Cross-Compilation (Keep CGO Disabled)

**What goes wrong:**
Go's `net` and `os/user` packages use CGO by default on macOS when CGO is available. If the build environment enables CGO, the resulting binary dynamically links against system libraries (libc, libresolv). This binary will not run on Linux (different libc), and cross-compiling for arm64/amd64 requires a C cross-compiler toolchain that is painful to configure in CI.

**Why it happens:**
GoReleaser builds for multiple platforms. `CGO_ENABLED=1` is the default when a C compiler is present. Most macOS CI runners have Xcode toolchain available, accidentally enabling CGO. The build succeeds but produces a dynamically linked binary that fails on other platforms with "exec format error" or missing library errors.

**Prevention:**
- Explicitly set `CGO_ENABLED: '0'` in the `builds` section of `.goreleaser.yml`
- Use `go build -tags netgo` or `-ldflags="-extldflags -static"` to force static builds if any CGO is unavoidable
- gsdw does not need CGO — it is a CLI tool with no FFI dependencies

**Detection:**
- `file gsdw` shows "dynamically linked" instead of "statically linked" or "executable"
- Linux users get "error while loading shared libraries" on a binary that worked on macOS

**Phase to address:** Distribution Phase 1 (GoReleaser builds config)

---

### Pitfall D6: `go install` Path Not in User's PATH — Silent Detection Failure

**What goes wrong:**
The `go install github.com/user/gsd-wired/cmd/gsdw@latest` path installs the binary to `$GOPATH/bin` (default `~/go/bin`). This directory is not in PATH by default on a fresh macOS or Linux system. The binary installs successfully, `go install` exits 0, but running `gsdw` returns "command not found." Users assume the install failed.

**Why it happens:**
Go's installer does not modify shell profile files. The `~/go/bin` directory is only in PATH if the user explicitly added it to `~/.bashrc`, `~/.zshrc`, or equivalent. This is a well-known Go ecosystem issue.

**Additional complication for dependency detection:**
When the wizard checks for `bd` and `dolt` via `exec.LookPath()`, it only searches PATH. If a user installed them via `go install` without the PATH fix, the wizard incorrectly reports them as missing and triggers the install flow again — false negative detection that can cause double-install attempts.

**Prevention:**
- In the installation wizard, check both `exec.LookPath(binary)` AND common install locations explicitly:
  - `~/go/bin/binary`
  - `$GOPATH/bin/binary` (from `go env GOPATH`)
  - `/usr/local/go/bin/binary`
- Print actionable PATH fix instructions when `go install` path is found but not in PATH
- For Homebrew installs, this is not an issue — brew manages symlinks into `/usr/local/bin` or `/opt/homebrew/bin`

**Detection:**
- `which gsdw` returns nothing but `ls ~/go/bin/gsdw` finds the binary
- Dependency wizard says "dolt not found" when `ls ~/go/bin/dolt` exists

**Phase to address:** Distribution Phase (dependency detection wizard)

---

### Pitfall D7: Dependency Version Detection — Version String Format Varies

**What goes wrong:**
The dependency wizard checks for `bd` and `dolt` by running `bd --version` or `dolt version` and parsing the output. Both tools may change their version output format across releases. Parsing the version string to enforce minimum version requirements is fragile — a format change in bd or dolt breaks the wizard's version check, causing false "dependency too old" errors or silent acceptance of actually-old versions.

**Why it happens:**
Version output formats are not stable contracts. `dolt version` outputs `dolt version X.Y.Z` on some releases and `dolt X.Y.Z` on others. `bd --version` may output JSON in one release and plain text in another. A regex that matches current format breaks silently when the format changes.

**Prevention:**
- Parse version output conservatively: extract the first semver-shaped string (`\d+\.\d+\.\d+`) anywhere in output using a broad regex, not a format-specific one
- Set minimum version floor only for capabilities actually needed (e.g., bd must support `--json` flag) — test the flag directly rather than parsing version number
- For `dolt`: use `dolt version` output as informational only; test actual connectivity (`dolt sql -q "SELECT 1"`) as the real health check
- Log the raw version string when a check fails so users can report it

**Detection:**
- Wizard rejects a valid install claiming "version too old"
- Wizard accepts an install that does not have required features

**Phase to address:** Distribution Phase (dependency detection wizard)

---

### Pitfall D8: Docker Compose `include:` Network Name Conflicts

**What goes wrong:**
When gsdw provides a drop-in `docker-compose.dolt.yml` that users include in their existing compose file, both files may define a network with the same name (e.g., `default`, `app-network`). Docker Compose's `include:` directive detects this and throws "imported compose file defines conflicting network." The user's existing `docker-compose.yml` stops working.

**Why it happens:**
`include:` evaluates the included file as an isolated sub-project. If the included file declares any network that shares a name with the parent file's networks, Compose reports a conflict — even if they would resolve to the same network. The `default` network name is a particularly common collision since both files implicitly create it.

**Consequences:**
- The drop-in fragment breaks the user's existing Docker setup entirely
- The error message points to the included file, which users did not write, creating confusion about what to fix

**Prevention:**
- Use a unique network name with a gsdw prefix: `gsdw-dolt-network` (not `default`, not `dolt`)
- In the fragment, attach the Dolt service only to the gsdw-specific network
- Document that users who want their app containers to talk to Dolt must explicitly add `gsdw-dolt-network` to their service's networks list
- Test the fragment by including it in a minimal compose file that also has a `default` network before shipping

**Detection:**
- `docker compose config` on the merged config shows network conflict warning
- `docker compose up` fails with "conflicting network" error immediately

**Phase to address:** Distribution Phase (docker-compose fragment design)

---

### Pitfall D9: `--net=host` Does Not Work on macOS — Container Cannot Reach Host

**What goes wrong:**
On Linux, `--net=host` shares the host network stack. A container running Dolt on `--net=host` is accessible at `localhost:3306` from processes on the host. On macOS, Docker runs inside a Linux VM — `--net=host` attaches the container to the VM's network, not the Mac host's network. `localhost:3306` from a macOS process does not reach the container. The connection wizard cannot detect or connect to the containerized Dolt server.

**Why it happens:**
Docker Desktop on macOS uses a HyperKit or Apple Hypervisor VM. `--net=host` means "host of the VM," not "the Mac." This is explicitly documented but almost universally misunderstood.

**Consequences:**
- Container starts with no errors but host process cannot connect
- `Connection refused` or timeout on `localhost:3306`
- Apple Containers (new native tool) has similar VM isolation on macOS 15 but different behavior on macOS 26

**Prevention:**
- Never use `--net=host` in the docker-compose fragment — always use explicit port mapping: `ports: ["3306:3306"]`
- Use `host.docker.internal` when a container needs to reach the Mac host (not the reverse)
- In the connection wizard, test connectivity with an explicit host:port, not via host networking assumptions
- Document that macOS does not support `--net=host` and the port mapping approach is required

**Detection:**
- Container shows "Up" in `docker ps` but `mysql -h 127.0.0.1 -P 3306` returns connection refused
- `docker inspect` shows `NetworkMode: host` instead of port bindings

**Phase to address:** Distribution Phase (container networking design)

---

### Pitfall D10: Apple Containers Tool Requires macOS 26 for Full Networking

**What goes wrong:**
Apple's native `container` tool (open-sourced June 2025) is OCI-compatible and targets Apple Silicon. On macOS 15 (Sequoia, current stable), it has "significant networking limitations including no container-to-container communication." The Dolt server container and any sidecar containers cannot communicate with each other on macOS 15 with Apple Containers. Container-to-container networking only works properly on macOS 26 (Tahoe, in beta as of March 2026, expected public release ~September 2026).

**Why it happens:**
Apple Containers uses per-container micro-VMs. The inter-VM networking stack that enables container-to-container communication was implemented as part of macOS 26's virtualization framework improvements.

**Consequences:**
- A user on macOS 15 using Apple Containers cannot run a Dolt server + client container pair that communicate
- The feature flag "Apple Container support" will have a silent capability gap for most users until macOS 26 is broadly adopted

**Prevention:**
- Detect runtime: use `container version` vs `docker version` vs `podman version` to identify the runtime
- If Apple Containers detected, check macOS version: `sw_vers -productVersion`
- On macOS 15 + Apple Containers: warn user and suggest Docker Desktop or Podman as alternatives, or use local Dolt binary instead of containerized Dolt
- On macOS 26 + Apple Containers: full support, document as primary path
- Document macOS version requirement prominently in Apple Container support section

**Detection:**
- Container starts, IP assigned, but `dolt sql -q "SELECT 1" --host <container-ip>` times out
- No error — just timeout, making diagnosis difficult

**Phase to address:** Distribution Phase (Apple Container support detection)

---

### Pitfall D11: Remote Dolt Connection Fails Silently — SSL Mode Mismatch

**What goes wrong:**
Dolt's SQL server supports TLS. If the server is configured with `require_secure_transport: true` but the client connects without `--ssl-mode` (or vice versa — server has no TLS, client demands it), the connection fails with a generic "connection refused" or TLS handshake error. The wizard's health check reports "cannot reach Dolt at host:port" when the real issue is SSL mode mismatch, which requires a different fix than connectivity.

**Why it happens:**
MySQL client SSL negotiation has multiple modes (`DISABLED`, `PREFERRED`, `REQUIRED`, `VERIFY_CA`, `VERIFY_IDENTITY`). The default varies by client version. Dolt's TLS support was added in late 2024 with `require_client_cert` added in 2025. Users who set up their Dolt server with TLS and then connect from a client using defaults get silent TLS failures that look like network failures.

**Prevention:**
- Connection wizard should test in sequence: (1) no TLS, (2) TLS without cert verification, (3) report which succeeded
- Show the actual error from the MySQL protocol layer, not just "connection failed"
- For remote Dolt: document the three connection modes and which `--ssl-mode` to use for each
- Default to `--ssl-mode=PREFERRED` in the wizard (tries TLS, falls back to plain)

**Detection:**
- Wizard says "cannot connect" but `mysql -h host -P 3306 --ssl-mode=DISABLED` succeeds
- TLS handshake error in Dolt server logs while wizard shows generic connectivity failure

**Phase to address:** Distribution Phase (connection wizard + remote connectivity)

---

### Pitfall D12: Port 3306 Already in Use — Local MySQL Conflicts Dolt Container

**What goes wrong:**
Many developer machines run a local MySQL or MariaDB instance on port 3306. The Dolt container also defaults to port 3306. When `docker-compose up` starts the Dolt container, it fails to bind port 3306 with "address already in use" — but this error appears in the container logs, not at the compose level. The wizard's subsequent connectivity check times out, giving a misleading "cannot reach Dolt" error rather than "port conflict."

**Why it happens:**
The docker-compose fragment exposes port 3306:3306 by default. Port conflicts on the host side appear as container startup failure, not as a compose configuration error. The user sees the container enter a restart loop.

**Prevention:**
- Default to port 3307:3306 in the drop-in fragment (3307 is not typically in use)
- Before starting containers, check if the host port is in use: `lsof -i :3306` (or Go equivalent using `net.Listen`)
- If port conflict detected, suggest port 3307 automatically in the wizard
- Make the port configurable via environment variable in the compose fragment: `${GSDW_DOLT_PORT:-3307}:3306`

**Detection:**
- `docker compose up` shows container restarting in a loop
- `docker logs <dolt-container>` shows "bind: address already in use"
- `curl localhost:3306` from the host returns connection but not Dolt response

**Phase to address:** Distribution Phase (docker-compose fragment + connection wizard)

---

### Pitfall D13: Homebrew Tap Naming Collision With Core Formula

**What goes wrong:**
If Homebrew's core formulae ever adds a `gsdw` formula (unlikely, but `gsd` might exist), then `brew install gsdw` installs from core, not from the tap. Users must use the fully qualified form `brew install user/tap/gsdw`. The `brew install gsdw` form in documentation becomes wrong without any warning to the user.

**More common real scenario:** A user has another tap that provides a binary with the same name. Both taps are active, and `brew install gsdw` picks one non-deterministically or errors with "multiple formulae found."

**Prevention:**
- Choose a unique name: `gsd-wired` or `gsdw` — verify against `brew search gsdw` before committing to the name
- Document the fully qualified install command (`brew install user/homebrew-tap/gsdw`) as the canonical form
- The cask approach (from Pitfall D1) reduces collision risk since casks live in a separate namespace from formulae

**Detection:**
- `brew info gsdw` shows a different package than expected
- Users report the wrong binary being installed

**Phase to address:** Distribution Phase (tap repository naming, before publishing)

---

## Distribution-Specific Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| GoReleaser + GitHub Actions | Using default `GITHUB_TOKEN` for tap push | Dedicated bot PAT with `contents: write` on tap repo only |
| GoReleaser + macOS binary | CGO enabled by default, dynamic linking | Explicitly set `CGO_ENABLED: '0'` in builds config |
| GoReleaser v2 config | Copying v1 config without `version: 2` header | Always start config with `version: 2`; run `goreleaser check` |
| GoReleaser + Homebrew | Using `brews:` key | Use `homebrew_casks:` — `brews` is deprecated since v2.10 |
| Docker Compose fragment + user's compose | Default network name collision | Use unique network name `gsdw-dolt-network` in fragment |
| macOS Docker + host networking | `--net=host` in compose fragment | Always use explicit port mapping; test with `host.docker.internal` |
| Dolt container + local MySQL | Port 3306 collision | Default to 3307 in fragment; make port configurable via env var |
| Apple Containers + macOS 15 | Assuming container-to-container networking works | Detect macOS version; warn on 15, document 26 requirement |
| `go install` + dependency wizard | `exec.LookPath()` misses `~/go/bin` | Also check `$(go env GOPATH)/bin` explicitly |
| Remote Dolt + TLS | Generic connectivity error on SSL mismatch | Test with and without TLS; show specific error from protocol layer |

---

## Phase-Specific Warnings (Distribution Milestone)

| Phase Topic | Likely Pitfall | Mitigation |
|-------------|---------------|------------|
| GoReleaser initial setup | Using `brews:` instead of `homebrew_casks:` | Start with cask — no migration debt |
| GoReleaser initial setup | Missing `version: 2` | Run `goreleaser check` in CI as pre-release gate |
| First release | GitHub token lacks tap push permission | Configure bot token before any release attempt |
| macOS binary | Gatekeeper blocking unsigned binary | Add `postinstall` xattr hook to cask definition |
| Dependency wizard | PATH misses `~/go/bin` | Check `$(go env GOPATH)/bin` directly |
| Dependency wizard | Version string parsing breaks | Parse first semver string broadly; test capability not version |
| Docker compose fragment | Network name collision | Use `gsdw-dolt-network` prefix; test against minimal compose |
| Container networking | macOS `--net=host` | Always use port mapping; never use host networking in compose |
| Dolt container | Port 3306 conflict with local MySQL | Default to 3307; pre-check port availability |
| Apple Container support | macOS 15 networking limitations | Version-gate feature; suggest Docker Desktop as fallback |
| Connection wizard | SSL/TLS mode mismatch looks like connectivity failure | Test both plain and TLS paths; surface protocol-layer errors |
| Remote Dolt fallback | Fallback silently fails | Test fallback path explicitly; log the trigger reason |

---

## Sources (Distribution Milestone)

- [GoReleaser Homebrew Casks Documentation](https://goreleaser.com/customization/homebrew_casks/) -- cask configuration, token requirements, signing
- [GoReleaser Deprecation Notices](https://goreleaser.com/deprecations/) -- brews deprecation timeline, breaking changes in v2
- [GoReleaser v2 Announcement](https://goreleaser.com/blog/goreleaser-v2/) -- version 2 breaking changes, upgrade guide
- [GoReleaser Version Errors](https://goreleaser.com/errors/version/) -- version header requirement
- [GoReleaser CGO Documentation](https://goreleaser.com/limitations/cgo/) -- cross-compilation constraints
- [Homebrew Taps Documentation](https://docs.brew.sh/Taps) -- naming, tap structure, formula vs cask
- [Docker Compose Include Directive](https://docs.docker.com/compose/how-tos/multiple-compose-files/include/) -- include behavior, conflict detection
- [Docker Compose Merge Behavior](https://docs.docker.com/compose/how-tos/multiple-compose-files/merge/) -- list accumulation, map override rules
- [Docker Desktop macOS Networking](https://docs.docker.com/desktop/features/networking/networking-how-tos/) -- host networking limitations on macOS
- [Dolt Docker Hub](https://hub.docker.com/r/dolthub/dolt-sql-server) -- official container, port mapping, env vars
- [Dolt SSL Mode Documentation](https://www.dolthub.com/blog/2024-12-03-ssl-mode/) -- TLS configuration, require_secure_transport
- [Apple Container GitHub](https://github.com/apple/container) -- macOS version requirements, networking limitations
- [Golang go install PATH issue (actions/setup-go#27)](https://github.com/actions/setup-go/issues/27) -- GOPATH/bin not in PATH by default
- [GoReleaser Issue #1146: one tap can handle only 1 linux and 1 macos archive](https://github.com/goreleaser/goreleaser/issues/1146) -- archive constraint for tap publication

---
*Distribution milestone pitfalls added: 2026-03-21*
