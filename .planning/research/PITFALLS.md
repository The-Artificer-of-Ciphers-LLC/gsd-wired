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
*Pitfalls research for: AI agent orchestration plugin (Claude Code + Beads/Dolt)*
*Researched: 2026-03-21*
