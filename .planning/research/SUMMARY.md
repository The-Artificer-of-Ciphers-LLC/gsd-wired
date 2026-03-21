# Project Research Summary

**Project:** gsd-wired (Claude Code Plugin: GSD + Beads/Dolt Agent Orchestration)
**Domain:** AI agent orchestration plugin with graph-backed persistence
**Researched:** 2026-03-21
**Confidence:** HIGH

## Executive Summary

gsd-wired is a Claude Code plugin that replaces the current GSD system's markdown-based state management (.planning/ directory) with a graph-backed persistence layer using Beads (issue tracker) on Dolt (versioned SQL database). The plugin operates as a single Go binary serving three roles: MCP server (long-lived, exposes workflow tools via stdio), hook dispatcher (short-lived, handles Claude Code lifecycle events), and CLI (debugging/manual operations). Research strongly supports Go as the implementation language due to direct Beads Go package compatibility, single-binary distribution, and the availability of an official MCP Go SDK (v1.4.1). The architecture maps GSD concepts onto Beads primitives -- phases become epic beads, plans become task beads with dependencies, and waves are computed dynamically from the dependency graph via `bd ready`.

The recommended approach is to build in layers that mirror the architecture's dependency chain: start with the bd CLI wrapper and MCP server scaffold (Layer 0-1), then add hook integration and basic slash commands (Layer 2), then multi-agent orchestration with validation gates (Layer 3), and finally token-aware context routing and optimization (Layer 4-5). This ordering respects both technical dependencies and risk mitigation -- the hardest engineering problems (token budget management, context tiering) come after the foundation is proven. The coexistence strategy with .planning/ files is the correct migration path; do not build full migration tooling.

The primary risks are: (1) hook latency degrading the agentic loop if hooks do too much work or trigger too many Dolt writes, (2) Dolt write amplification from per-tool-call commits causing database bloat and CPU burn, and (3) multi-agent error amplification if subagent outputs are not validated before downstream consumption. All three are mitigable with specific patterns identified in research: batched writes at wave boundaries, lazy MCP initialization, strict stdout/stderr discipline in hooks, and explicit agent output contracts with validation gates.

## Key Findings

### Recommended Stack

Go 1.25.x is the clear choice -- same language as Beads enables direct package import (no CLI shelling for hot paths), single-binary distribution eliminates runtime dependencies, and the official MCP Go SDK handles protocol compliance. The embedded Dolt driver provides in-process database access without a running server.

**Core technologies:**
- **Go 1.25.x**: Implementation language -- matches Beads ecosystem, single binary, no runtime deps
- **modelcontextprotocol/go-sdk v1.4.1**: MCP server -- official SDK with Google co-maintenance, auto JSON schema from Go structs
- **github.com/steveyegge/beads v0.61.0**: Graph persistence -- Go package with Storage/Transaction interfaces, direct import
- **github.com/dolthub/driver v1.83.8**: Embedded Dolt access -- database/sql compatible, no server process
- **spf13/cobra v1.10.2**: CLI framework -- handles dual-mode binary (serve vs hook vs CLI subcommands)

**Critical version note:** Go 1.25.x must match Beads' Go version for import compatibility. MCP SDK requires spec 2025-11-25.

### Expected Features

**Must have (table stakes):**
- Project initialization via guided questioning (produces epic + context beads)
- Phase-based workflow lifecycle (init, research, plan, execute, verify, ship)
- Wave-based parallel execution via `bd ready` dependency resolution
- Subagent spawning with bead-scoped context isolation
- SessionStart hook for automatic project state loading
- PreCompact hook for state preservation before compaction
- Coexistence with .planning/ directory (read as fallback)
- Slash commands: /gsd-wired:init, :phase, :plan, :execute, :verify, :status, :ship

**Should have (differentiators):**
- Token-aware context routing (hot/warm/cold tiering) -- the core innovation
- Bead-scoped subagent context (only relevant beads, not full project)
- Graph-backed state replacing markdown bloat (O(relevant) not O(total) queries)
- Versioned state via Dolt (branch, diff, rollback project decisions)
- Automatic compaction of closed work

**Defer (v2+):**
- Web UI / dashboard (separate product, wrong audience for v1)
- Multi-model support (hook model is Claude Code-specific)
- Multi-project orchestration (single-project must work first)
- Plugin/extensibility API (premature abstraction)
- TUI visualization, Dolt remote sync, non-Claude-Code support

### Architecture Approach

The system is a single Go binary with three execution modes selected by Cobra subcommand. The MCP server runs long-lived over stdio, exposing tools (gsd_init, gsd_phase, gsd_plan, gsd_wave, gsd_execute, gsd_verify, gsd_status, gsd_compact). The hook dispatcher runs short-lived per Claude Code lifecycle event (SessionStart, PreToolUse, PostToolUse, PreCompact, SubagentStart, SubagentStop). All graph operations go through a bd CLI wrapper layer that shells out to `bd --json` for v1, with a clear migration path to direct Go package import for performance in v2.

**Major components:**
1. **MCP Server** (internal/mcp/) -- tool registration and handler dispatch, long-lived stdio process
2. **Hook Dispatcher** (internal/hooks/) -- routes Claude Code lifecycle events to handlers, short-lived per invocation
3. **bd Wrapper** (internal/beads/) -- shells out to `bd --json`, parses responses, maps GSD domain model
4. **Domain Model** (internal/domain/) -- project, phase, plan, wave abstractions independent of persistence
5. **Context Manager** (internal/context/) -- token budget tracking, hot/warm/cold tiering, context selection
6. **Fallback Reader** (internal/fallback/) -- reads .planning/ files for coexistence mode
7. **Plugin Assets** (commands/, agents/, skills/, hooks/) -- markdown/JSON files loaded by Claude Code

### Critical Pitfalls

1. **Hook stdout pollution** -- Any non-JSON output to stdout from hook handlers breaks Claude Code's JSON parsing. All logging must go to stderr. Establish this discipline in the first commit with CI validation.
2. **Dolt write amplification** -- Per-tool-call Dolt commits cause database bloat and CPU burn from auto-stats. Batch writes at wave boundaries, not per operation. Set auto-stats interval to 300s+.
3. **PreCompact race condition** -- PreCompact is non-blocking; compaction does not wait for the hook to finish. Write state to a fast local buffer first, sync to Dolt asynchronously.
4. **Multi-agent error amplification** -- Without validation gates between agents, errors compound 17x in unstructured multi-agent systems. Define explicit bead schemas per agent type, validate before passing downstream.
5. **MCP startup latency** -- Dolt initialization can take 3-10s. Respond to MCP `initialize` immediately, defer database connection to first tool call (lazy init).

## Implications for Roadmap

Based on research, the architecture's dependency chain (Layers 0-5) maps cleanly to implementation phases. Here is the suggested structure:

### Phase 1: Foundation -- Binary Scaffold + bd Wrapper
**Rationale:** Everything depends on the Go binary being able to talk to bd and respond to MCP/hook requests. This is Layer 0-1 from the architecture research.
**Delivers:** Working Go binary with Cobra subcommands (serve, hook), bd CLI wrapper with JSON parsing, MCP server that responds to `initialize` with lazy Dolt connection, hook dispatcher skeleton that reads stdin JSON and routes events.
**Addresses:** MCP server wrapping bd CLI, plugin manifest, .mcp.json, hooks.json structure
**Avoids:** Hook stdout pollution (establish stderr-only logging from day one), MCP startup latency (lazy init), hooks.json duplication (correct plugin structure from first commit)

### Phase 2: Hook Integration + State Persistence
**Rationale:** Hooks are the integration surface between Claude Code and the plugin. Without SessionStart loading context and PreCompact saving state, the plugin is inert. This is Layer 2 from the architecture.
**Delivers:** SessionStart hook loading active/ready beads as context, PreCompact hook with two-stage save (fast local buffer + async Dolt sync), PostToolUse hook with selective matchers (Write/Edit/Bash only), basic slash commands (init, status).
**Addresses:** SessionStart context loading, PreCompact state preservation, coexistence with .planning/ (fallback reading in SessionStart)
**Avoids:** PreCompact race condition (two-stage save pattern), Dolt write amplification (batched writes, selective hook matchers)

### Phase 3: GSD Workflow -- Init, Plan, Execute, Verify
**Rationale:** With hooks and MCP working, build the actual GSD workflow. This is where the product becomes usable. Layers 2-3 from architecture.
**Delivers:** Project initialization flow producing epic + context beads, phase creation as epic beads, plan creation as task beads with dependencies, wave execution via `bd ready`, post-execution verification against bead success criteria, full slash command suite.
**Addresses:** Project init, phase creation, plan creation, wave execution, verification, slash commands
**Avoids:** Bag-of-agents error amplification (define agent contracts and validation gates before enabling multi-agent workflows)

### Phase 4: Multi-Agent Orchestration
**Rationale:** Subagent spawning with bead-scoped context is the key differentiator but depends on the workflow being solid. Layer 3 from architecture.
**Delivers:** SubagentStart/SubagentStop hooks injecting bead-scoped context, parallel research agents (4x) each claiming child beads, wave-based execution with concurrent agents claiming unblocked tasks, claim timeout/expiry for crashed agents.
**Addresses:** Subagent bead-scoped context, research agent coordination, wave execution parallelism
**Avoids:** Agent error amplification (validation gates on every agent output), subprocess overhead (begin evaluating Go library import vs CLI wrapping)

### Phase 5: Token-Aware Context Routing
**Rationale:** This is the hardest engineering problem and the core innovation. It must come after the foundation is proven because tier boundary decisions require real usage data. Layer 4 from architecture.
**Delivers:** Hot/warm/cold bead classification, token budget tracking with injection limits, PreToolUse context injection for state-changing tools, automatic compaction scheduling, budget visibility command (/gsd-wired:budget).
**Addresses:** Token-aware context routing, tiered context management, PreToolUse context injection, automatic compaction
**Avoids:** Context budget starvation (hard limits per injection, orchestrator stays lean)

### Phase 6: Polish + Migration
**Rationale:** Optimization and adoption paths. Layer 5 from architecture. Only after core is proven.
**Delivers:** .planning/ fallback reader improvements, performance optimization (potential bd Go library import replacing CLI wrapping), cross-platform binary distribution via goreleaser, installation script that checks dependencies.
**Addresses:** Coexistence improvements, bd subprocess overhead optimization, distribution
**Avoids:** Premature optimization (only optimize after profiling real usage)

### Phase Ordering Rationale

- **Phases 1-2 first** because every subsequent feature depends on the binary scaffold, bd wrapper, and hook integration working correctly. Getting stdout discipline and lazy init right from the start prevents costly retrofits.
- **Phase 3 before Phase 4** because single-agent workflow must work before multi-agent. You cannot debug agent coordination if the underlying init-plan-execute-verify loop is broken.
- **Phase 5 after Phase 4** because token budget decisions require real multi-agent usage data. Hard-coding budgets from theory will be wrong; measure first, then optimize.
- **Phase 6 last** because migration tooling and distribution polish are adoption concerns, not functionality concerns. Ship the core, then smooth the edges.

### Research Flags

Phases likely needing deeper research during planning:
- **Phase 2 (Hook Integration):** PreCompact's non-blocking behavior and the two-stage save pattern need prototyping. The exact hook input/output JSON schemas should be verified against Claude Code's current implementation.
- **Phase 4 (Multi-Agent):** SubagentStart/SubagentStop hook behavior with bead claims needs empirical testing. Claim timeout/expiry patterns are not documented in Beads.
- **Phase 5 (Token Routing):** No established patterns for token budget estimation in MCP plugins. The 1-token-per-4-chars heuristic needs validation. Tier boundary decisions are novel engineering.

Phases with standard patterns (skip research-phase):
- **Phase 1 (Foundation):** MCP Go SDK has clear examples. Cobra CLI is well-documented. bd CLI wrapping is straightforward exec.Command.
- **Phase 3 (GSD Workflow):** The init-plan-execute-verify lifecycle is well-understood from current GSD. Mapping to beads is the only novel part.
- **Phase 6 (Polish):** Goreleaser, installation scripts, and fallback readers are standard Go patterns.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | All technologies verified against official docs and pkg.go.dev. Version compatibility confirmed. |
| Features | MEDIUM-HIGH | GSD and Beads ecosystems well-understood. Token-aware routing is novel -- no prior art to validate against. |
| Architecture | HIGH | Three integration surfaces (MCP, hooks, bd CLI) all verified against official documentation. Build order is clear. |
| Pitfalls | HIGH | Critical pitfalls sourced from official docs (PreCompact non-blocking) and production reports (Dolt auto-stats). |

**Overall confidence:** HIGH

### Gaps to Address

- **Token budget estimation:** No API to query remaining context window size. The heuristic approach (count chars, estimate tokens) needs empirical validation during Phase 5 planning.
- **bd batch operations:** Research suggests batching writes, but bd's current CLI may not support multi-operation batches. Verify `bd` supports transaction-like batching or implement at the Go wrapper level.
- **SubagentStart hook payload:** The exact JSON schema for SubagentStart hook input needs verification. Research assumes it includes a mechanism to inject `additionalContext`, but the subagent-specific fields should be confirmed.
- **Claim expiry in Beads:** Beads supports `--claim` on updates, but automatic claim expiry (for crashed agents) is not a built-in feature. This will need to be implemented as a custom watchdog in the hook layer.
- **Embedded Dolt driver vs bd CLI:** STACK.md recommends the embedded driver for direct SQL; ARCHITECTURE.md recommends bd CLI wrapping for decoupling. Resolution: use bd CLI wrapping for v1 (simpler, decoupled), embedded driver is the Phase 6 optimization path if profiling shows subprocess overhead is the bottleneck.

## Sources

### Primary (HIGH confidence)
- [Official MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk) -- API patterns, tool registration, stdio transport
- [Claude Code Hooks Reference](https://code.claude.com/docs/en/hooks) -- all hook events, blocking behavior, JSON formats
- [Claude Code Plugin Docs](https://code.claude.com/docs/en/plugins) -- plugin structure, manifest schema, hooks.json convention
- [Beads GitHub + Go Package](https://github.com/steveyegge/beads) -- Storage/Transaction API, bd CLI, dependency graph
- [Dolt Embedded Driver](https://github.com/dolthub/driver) -- database/sql compatible access

### Secondary (MEDIUM confidence)
- [Anthropic: Effective Context Engineering](https://www.anthropic.com/engineering/effective-context-engineering-for-ai-agents) -- token optimization patterns
- [17x Error Trap in Multi-Agent Systems](https://towardsdatascience.com/why-your-multi-agent-system-is-failing-escaping-the-17x-error-trap-of-the-bag-of-agents/) -- coordination topology research
- [Dolt Production Investigation](https://gist.github.com/l0g1x/ef6dc1a971fa124e8d5939f3115b4e7d) -- auto-stats CPU burn report

### Tertiary (LOW confidence)
- Token budget heuristic (1 token per 4 chars) -- commonly cited but not formally validated for Claude's tokenizer
- Claim timeout patterns for crashed agents -- extrapolated from distributed systems patterns, not Beads-specific

---
*Research completed: 2026-03-21*
*Ready for roadmap: yes*
