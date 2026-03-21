# Architecture Research

**Domain:** Claude Code plugin -- MCP server + hooks orchestrator wrapping Beads/Dolt
**Researched:** 2026-03-21
**Confidence:** HIGH (all three integration surfaces verified against official docs)

## Standard Architecture

### System Overview

```
+---------------------------------------------------------------------+
|                     Claude Code Runtime                              |
|                                                                      |
|  +---------------------------------------------------------------+  |
|  |                  gsd-wired Plugin                              |  |
|  |                                                                |  |
|  |  +-----------+  +----------+  +-----------------+              |  |
|  |  |  Skills   |  |  Agents  |  | Slash Commands  |              |  |
|  |  | (SKILL.md)|  | (agent.md)|  | (commands/)    |              |  |
|  |  +-----+-----+  +-----+----+  +-------+---------+             |  |
|  |        |               |               |                       |  |
|  |  +-----+---------------+---------------+---------+             |  |
|  |  |              Hooks Layer (hooks.json)          |            |  |
|  |  | SessionStart | PreToolUse | PostToolUse        |            |  |
|  |  | PreCompact | SubagentStart | SubagentStop      |            |  |
|  |  +----------------------------+-------------------+            |  |
|  |                               | stdin/stdout JSON              |  |
|  |  +----------------------------+-------------------+            |  |
|  |  |         Hook Dispatcher (Go binary)            |            |  |
|  |  | Receives hook events, routes to handler        |            |  |
|  |  +----------------------------+-------------------+            |  |
|  |                               |                                |  |
|  |  +----------------------------+-------------------+            |  |
|  |  |        MCP Server (Go, stdio transport)        |            |  |
|  |  | Tools: gsd_init, gsd_phase, gsd_plan,          |            |  |
|  |  |   gsd_wave, gsd_execute, gsd_verify,           |            |  |
|  |  |   gsd_status, gsd_compact, gsd_context         |            |  |
|  |  +----------------------------+-------------------+            |  |
|  |                               |                                |  |
|  +-------------------------------+--------------------------------+  |
|                                  |                                   |
|  +-------------------------------+--------------------------------+  |
|  |             bd CLI Wrapper (Go library)                        |  |
|  | Calls bd commands, parses JSON, maps GSD concepts              |  |
|  | Phase=epic, Plan=task, Wave=dependency layer                   |  |
|  +-------------------------------+--------------------------------+  |
|                                  | exec bd --json                    |
|  +-------------------------------+--------------------------------+  |
|  |             bd CLI (external binary)                           |  |
|  +-------------------------------+--------------------------------+  |
|                                  |                                   |
|  +-------------------------------+--------------------------------+  |
|  |             Dolt Database (.beads/)                            |  |
|  | Version-controlled SQL, hash-based IDs, cell-level merge      |  |
|  +----------------------------------------------------------------+  |
+----------------------------------------------------------------------+
```

### Component Responsibilities

| Component | Responsibility | Typical Implementation |
|-----------|----------------|------------------------|
| Plugin Manifest | Identity, version, namespace for `gsd-wired:*` commands | `.claude-plugin/plugin.json` |
| Slash Commands | User-facing entry points (`/gsd-wired:init`, `/gsd-wired:phase`, etc.) | Markdown files in `commands/` with `$ARGUMENTS` |
| Skills | Model-invoked capabilities (auto-detected by Claude based on task) | `skills/*/SKILL.md` with frontmatter |
| Agents | Specialized subagents (researcher, planner, executor, verifier) | `agents/*.md` with tool restrictions |
| Hooks Layer | Lifecycle integration -- context injection, state persistence, routing | `hooks/hooks.json` pointing to Go binary |
| Hook Dispatcher | Single Go binary handling all hook events via subcommand routing | `gsd-wired hook <event-name>` reads stdin JSON |
| MCP Server | Tool exposure for graph operations and workflow orchestration | Go binary using `modelcontextprotocol/go-sdk`, stdio transport |
| bd Wrapper | Go library that shells out to `bd` with `--json`, parses responses, maps GSD domain model | Internal Go package, not a separate binary |
| bd CLI | External dependency -- graph operations on Dolt database | Installed separately (`go install`, `brew`, or `npm`) |
| Dolt Database | Persistent storage -- issues, dependencies, metadata, audit trail | `.beads/` directory in project root |
| Fallback Reader | Reads `.planning/` markdown files when beads not initialized | Go package for migration/coexistence path |

## Recommended Project Structure

```
gsd-wired/
  .claude-plugin/
    plugin.json              # Plugin manifest (name, version, author)
  commands/                  # Slash commands (user-invoked)
    init.md                  # /gsd-wired:init -- project initialization
    phase.md                 # /gsd-wired:phase -- create/transition phases
    plan.md                  # /gsd-wired:plan -- create execution plans
    execute.md               # /gsd-wired:execute -- run wave-based execution
    verify.md                # /gsd-wired:verify -- post-execution verification
    status.md                # /gsd-wired:status -- project status from graph
    ship.md                  # /gsd-wired:ship -- PR creation and milestone close
  agents/                    # Subagent definitions
    researcher.md            # Research agent (parallel, 4x)
    planner.md               # Plan creation agent
    executor.md              # Wave execution agent
    verifier.md              # Verification agent
    synthesizer.md           # Research synthesis agent
  skills/                    # Model-invoked skills
    context-loading/
      SKILL.md               # Auto-load bead context for current work
    wave-detection/
      SKILL.md               # Detect next ready wave from graph
    token-budget/
      SKILL.md               # Token-aware context selection
  hooks/
    hooks.json               # Hook configuration pointing to Go binary
  .mcp.json                  # MCP server configuration (points to Go binary)
  cmd/                       # Go binary entry points
    gsd-wired/
      main.go                # MCP server entry point
    gsd-hook/
      main.go                # Hook dispatcher entry point
  internal/                  # Internal Go packages
    mcp/                     # MCP server implementation
      server.go              # Server setup, tool registration
      tools/                 # Individual tool handlers
        init.go
        phase.go
        plan.go
        wave.go
        status.go
        compact.go
    hooks/                   # Hook event handlers
      dispatcher.go          # Route hook events to handlers
      session_start.go       # Load project context from beads
      pre_tool_use.go        # Intercept/route tool calls
      post_tool_use.go       # Update bead state after execution
      pre_compact.go         # Save state before compaction
      subagent_start.go      # Inject bead context into subagents
      subagent_stop.go       # Collect results, close beads
      stop.go                # Final state persistence
    beads/                   # bd CLI wrapper
      client.go              # Execute bd commands, parse JSON
      models.go              # Go types for beads entities
      phase.go               # Phase=epic mapping
      plan.go                # Plan=task mapping
      wave.go                # Wave=dependency layer logic
      context.go             # Context loading/tiering
    domain/                  # GSD domain model
      project.go             # Project state
      phase.go               # Phase lifecycle
      plan.go                # Plan/task model
      wave.go                # Wave computation
    context/                 # Token-aware context management
      budget.go              # Token budget tracking
      tiering.go             # Hot/warm/cold bead classification
      selector.go            # Select context within budget
    fallback/                # .planning/ compatibility
      reader.go              # Read PROJECT.md, ROADMAP.md, STATE.md
      migrator.go            # Future: migrate .planning/ to beads
  go.mod
  go.sum
  Makefile                   # Build, install, test targets
```

### Structure Rationale

- **`cmd/` split into two binaries:** The MCP server (`gsd-wired`) runs as a long-lived stdio process. The hook dispatcher (`gsd-hook`) runs as a short-lived command per hook event. Separating them keeps startup fast for hooks (critical -- hooks block the agentic loop) while the MCP server can maintain state in memory.
- **`internal/` for all Go packages:** Standard Go convention. Nothing is exported as a library; this is an application.
- **`internal/beads/` wraps bd CLI:** Does not import beads Go packages directly. Shells out to `bd --json` so we stay decoupled from bd's internal API and can track any bd version. If bd's Go API stabilizes, this becomes the single swap point.
- **`internal/hooks/` mirrors Claude Code hook events:** One file per hook event type for clarity. The dispatcher routes based on the `hook_event_name` field from stdin JSON.
- **Plugin assets (`commands/`, `agents/`, `skills/`, `hooks/`) at root:** Required by Claude Code's plugin directory structure. These are markdown/JSON files, not Go code.

## Architectural Patterns

### Pattern 1: Single Binary, Dual Mode

**What:** One Go binary serves both as MCP server (long-lived) and hook dispatcher (short-lived), selected by subcommand.
**When to use:** When you control the binary but need two execution modes. Claude Code's `.mcp.json` and `hooks.json` both point to the same binary with different arguments.
**Trade-offs:** Simpler distribution (one binary to install) vs. slightly larger binary. Hook startup must be fast (<100ms) or it blocks the agentic loop.

**Example:**
```go
func main() {
    if len(os.Args) > 1 && os.Args[1] == "hook" {
        // Short-lived: read stdin JSON, dispatch, write stdout JSON, exit
        hooks.Dispatch(os.Args[2]) // "SessionStart", "PreToolUse", etc.
    } else {
        // Long-lived: start MCP server on stdio
        server := mcp.NewServer(&mcp.Implementation{
            Name:    "gsd-wired",
            Version: "0.1.0",
        }, nil)
        registerTools(server)
        server.Run(context.Background(), &mcp.StdioTransport{})
    }
}
```

**Alternative considered:** Two separate binaries. Rejected because it doubles build/install complexity for minimal benefit. The hook path is a thin dispatcher that exits immediately.

### Pattern 2: bd CLI as Data Access Layer

**What:** All Dolt/beads operations go through `bd` CLI with `--json` flag, never through direct Dolt SQL or bd Go imports.
**When to use:** When the upstream tool has a stable CLI but unstable Go API, or when you want version decoupling.
**Trade-offs:** Subprocess overhead per operation (~10-50ms) vs. full API decoupling. Acceptable because hook events are infrequent (not in a hot loop).

**Example:**
```go
func (c *Client) Ready() ([]Issue, error) {
    out, err := c.exec("ready", "--json")
    if err != nil {
        return nil, fmt.Errorf("bd ready: %w", err)
    }
    var issues []Issue
    if err := json.Unmarshal(out, &issues); err != nil {
        return nil, fmt.Errorf("parse bd ready: %w", err)
    }
    return issues, nil
}
```

### Pattern 3: Hook Event Routing via stdin JSON

**What:** Claude Code passes hook context as JSON on stdin. The Go binary reads it, deserializes based on `hook_event_name`, dispatches to the appropriate handler, and writes JSON response to stdout.
**When to use:** Always -- this is how Claude Code hooks work. The binary is invoked fresh each time.
**Trade-offs:** No persistent state between hook invocations (must read from beads each time) vs. simplicity and crash safety.

**Example:**
```go
func Dispatch(eventName string) {
    input, _ := io.ReadAll(os.Stdin)

    var response any
    switch eventName {
    case "SessionStart":
        response = handleSessionStart(input)
    case "PreToolUse":
        response = handlePreToolUse(input)
    case "PostToolUse":
        response = handlePostToolUse(input)
    case "PreCompact":
        response = handlePreCompact(input)
    }

    json.NewEncoder(os.Stdout).Encode(response)
}
```

### Pattern 4: GSD Domain Mapping onto Beads Graph

**What:** GSD concepts map to beads entities: Phase=epic bead, Plan=task bead, Wave=dependency layer (computed from `bd ready` at each step). Research=epic with 4 child tasks (one per researcher). Verification=task with success criteria in bead metadata.
**When to use:** For all workflow orchestration. The mapping is the core intellectual property of this plugin.
**Trade-offs:** Leverages beads' existing dependency graph vs. limited to what beads can express (no custom node types, only extensible fields on issues).

### Pattern 5: Tiered Context Loading

**What:** Beads are classified as hot (currently executing), warm (same phase, open), or cold (closed/other phase). Hot beads get full context injected. Warm beads get summary. Cold beads get nothing (queryable on demand). Classification computed at SessionStart and updated at each PostToolUse.
**When to use:** Always -- this is how token budgets stay manageable.
**Trade-offs:** Complexity of tier management vs. dramatic token savings. The tier boundary decisions are the hardest engineering problem in this project.

## Data Flow

### Session Lifecycle Flow

```
Session Start
    |
    v
[SessionStart Hook]
    | Read PROJECT.md + bd stats --json
    | Classify beads into hot/warm/cold tiers
    | Return additionalContext with project state + hot bead details
    v
Claude receives project context automatically
    |
    v
User invokes /gsd-wired:execute (or Claude auto-selects skill)
    |
    v
[PreToolUse Hook] <-- fires for each tool Claude wants to use
    | If MCP tool call to gsd-wired: inject current bead context
    | If Bash/Write/Edit: check if within claimed bead scope
    | Return: allow/deny + additionalContext
    v
[MCP Tool Executes] <-- e.g., gsd_wave tool
    | Calls bd ready --json to find unblocked tasks
    | Computes wave (set of tasks with no open deps)
    | Returns wave tasks to Claude
    v
[PostToolUse Hook]
    | If file was written: check if it satisfies a bead's criteria
    | If bead completed: bd close <id> "completion message"
    | Update tier classification
    | Return additionalContext with updated state
    v
[SubagentStart Hook] <-- when Claude spawns researcher/executor
    | Inject only the claimed bead's context (not full project)
    | Return additionalContext scoped to that bead
    v
[SubagentStop Hook]
    | Collect subagent results
    | Update bead with results via bd update
    | If all sibling beads closed: notify parent
    v
[PreCompact Hook] <-- before context window compaction
    | Save in-progress state to beads via bd update
    | Ensure nothing is lost when context is compressed
    | Return: no decision control (observability only per docs)
    v
[Stop Hook]
    | Persist any unsaved state
    | Update session metadata in beads
    v
Session End
```

### Wave Execution Flow (Multi-Agent)

```
Orchestrator (main thread)
    |
    +-- bd ready --json --> Returns: [task-a, task-b, task-c] (Wave 1)
    |
    +-- Spawn Agent 1 --> bd update task-a --claim
    |   |                  [SubagentStart: inject task-a context]
    |   |                  Agent works on task-a
    |   |                  [SubagentStop: bd close task-a "done"]
    |
    +-- Spawn Agent 2 --> bd update task-b --claim
    |   |                  [SubagentStart: inject task-b context]
    |   |                  Agent works on task-b
    |   |                  [SubagentStop: bd close task-b "done"]
    |
    +-- Spawn Agent 3 --> bd update task-c --claim
    |   |                  [SubagentStart: inject task-c context]
    |   |                  Agent works on task-c
    |   |                  [SubagentStop: bd close task-c "done"]
    |
    +-- All Wave 1 complete
    |
    +-- bd ready --json --> Returns: [task-d, task-e] (Wave 2)
    |
    +-- ... repeat ...
```

### Key Data Flows

1. **Context injection:** SessionStart hook reads beads graph, computes tiers, returns `additionalContext` string that Claude receives as session context. This is the primary mechanism for Claude to "know" project state without reading markdown files.

2. **State persistence:** PostToolUse and PreCompact hooks write state back to beads via `bd update`. This ensures progress is captured even if the session crashes or compacts. The beads graph is the source of truth, not the conversation context.

3. **Subagent scoping:** SubagentStart hook reads the specific bead the subagent will work on and injects only that bead's context. This is how token consumption stays lean -- subagents never see the full project graph.

4. **Wave computation:** The MCP `gsd_wave` tool queries `bd ready --json` to get unblocked tasks, groups them by dependency depth, and returns the current wave. This is a computed view, not stored state.

## Scaling Considerations

| Scale | Architecture Adjustments |
|-------|--------------------------|
| 1-5 phases, <50 beads | Single project, single Dolt DB, no optimization needed |
| 5-20 phases, 50-500 beads | `bd compact` becomes essential -- run at phase transitions to summarize closed beads. Tier classification critical for context budget. |
| 20+ phases, 500+ beads | Consider Dolt branching per phase to isolate query scope. May need `bd query` with custom SQL instead of `bd ready` for performance. |

### Scaling Priorities

1. **First bottleneck: Hook latency.** Every hook invocation shells out to the Go binary, which may shell out to `bd`. If hook processing exceeds ~200ms, the agentic loop feels sluggish. Mitigation: keep hook handlers minimal; cache bead state in a temp file between hook invocations within a session.
2. **Second bottleneck: Context window pressure.** As projects grow, even tiered loading may exceed budgets. Mitigation: aggressive compaction, summary-only for warm beads, and `bd compact` at every phase transition.

## Anti-Patterns

### Anti-Pattern 1: Fat Hooks

**What people do:** Put complex orchestration logic (multi-step bd queries, wave computation, subagent spawning) inside hook handlers.
**Why it's wrong:** Hooks block the agentic loop. They run synchronously. A 2-second hook means Claude waits 2 seconds before every tool use. Hooks also cannot spawn subagents -- only the MCP server and skills can trigger those.
**Do this instead:** Hooks should do minimal state read/write. Push orchestration logic into MCP tools and skills that Claude invokes when ready.

### Anti-Pattern 2: Direct Dolt SQL from Hooks

**What people do:** Import Dolt Go client and query the database directly, bypassing `bd`.
**Why it's wrong:** Couples to bd's internal schema (which is not stable). Risks schema version conflicts. Loses bd's built-in conflict resolution and compaction logic.
**Do this instead:** Always go through `bd` CLI with `--json`. Accept the subprocess overhead. It keeps you decoupled.

### Anti-Pattern 3: Stateful Hook Binary

**What people do:** Try to maintain in-memory state across hook invocations (e.g., daemon mode for the hook binary).
**Why it's wrong:** Claude Code invokes hooks as fresh processes each time. There is no persistent process for command-type hooks. The binary starts, reads stdin, writes stdout, exits.
**Do this instead:** Use the beads graph as persistent state. For hot-path caching, write a session state file to a temp directory (keyed by `session_id` from hook input) and read it back on next invocation.

### Anti-Pattern 4: Duplicating bd's Data Model

**What people do:** Define custom SQL tables in Dolt alongside beads' schema, or store GSD state in a parallel system.
**Why it's wrong:** Two sources of truth. Beads' compaction and dependency tracking will not know about your custom tables.
**Do this instead:** Use bd's extensible fields for GSD-specific metadata (phase tags, requirement IDs, success criteria, token budgets). Beads issues support arbitrary metadata -- use it.

### Anti-Pattern 5: Monolithic Slash Commands

**What people do:** Put the entire workflow (init + research + plan + execute + verify) behind a single slash command.
**Why it's wrong:** Claude Code is already an agent. Give it tools and context, and let it orchestrate. A single command that tries to do everything fights against Claude's natural agentic flow.
**Do this instead:** Each slash command maps to one workflow phase. Skills and agents handle the orchestration within that phase. Claude decides when to transition.

## Integration Points

### External Services

| Service | Integration Pattern | Notes |
|---------|---------------------|-------|
| bd CLI | Subprocess exec with `--json` flag | Must be on PATH or specified via `BEADS_PATH` env var |
| Dolt | Accessed only through bd (never directly) | `.beads/` directory must exist (via `bd init`) |
| Claude Code | Plugin directory loaded via `--plugin-dir` or marketplace install | Hooks via command type, MCP via stdio transport |
| GitHub (optional) | `gh` CLI for PR creation from within executor agent | Not a hard dependency; PR workflow is optional |

### Internal Boundaries

| Boundary | Communication | Notes |
|----------|---------------|-------|
| Hooks layer to Hook dispatcher | stdin/stdout JSON (Claude Code spawns process) | One process per event, must exit quickly |
| MCP server to Claude Code | JSON-RPC 2.0 over stdio (long-lived connection) | Uses `modelcontextprotocol/go-sdk` |
| Hook dispatcher to bd wrapper | Go function call (same binary) | No IPC overhead |
| MCP server to bd wrapper | Go function call (same binary) | No IPC overhead |
| bd wrapper to bd CLI | `exec.Command("bd", ...)` subprocess | ~10-50ms per invocation |
| Skills/Agents to MCP tools | Claude invokes MCP tools based on skill instructions | Skills guide Claude; MCP tools do the work |
| Slash commands to Skills/Agents | Commands set context; Claude uses skills to fulfill | Commands are entry points, skills are capabilities |

## Build Order (Dependency Chain)

The components have clear dependency ordering that maps to implementation phases:

```
Layer 0 (no deps):     bd wrapper (internal/beads/)
                        domain model (internal/domain/)

Layer 1 (needs L0):     MCP server with basic tools (internal/mcp/)
                        Hook dispatcher skeleton (internal/hooks/)

Layer 2 (needs L1):     SessionStart hook (context loading)
                        Plugin manifest + .mcp.json
                        Basic slash commands

Layer 3 (needs L2):     PreToolUse / PostToolUse hooks (state management)
                        SubagentStart / SubagentStop hooks (subagent scoping)
                        Agent definitions (researcher, executor, etc.)

Layer 4 (needs L3):     Context tiering (internal/context/)
                        PreCompact hook (state preservation)
                        Wave execution orchestration
                        Skills (auto-invoked capabilities)

Layer 5 (needs L4):     Fallback reader (internal/fallback/)
                        Token budget optimization
                        Full workflow (init to ship)
```

**Implication for roadmap:** Each layer is a natural phase boundary. Layer 0-1 is foundation. Layer 2 is "it works at all." Layer 3 is "it works with subagents." Layer 4 is "it works efficiently." Layer 5 is "it works for migration users."

## Sources

- [Claude Code Plugin Documentation](https://code.claude.com/docs/en/plugins) -- official plugin structure, manifest format, directory layout
- [Claude Code Hooks Reference](https://code.claude.com/docs/en/hooks) -- all 16 hook events, input/output formats, lifecycle
- [MCP Go SDK (official)](https://github.com/modelcontextprotocol/go-sdk) -- server creation, tool registration, stdio transport
- [Beads README](https://github.com/steveyegge/beads/blob/main/README.md) -- bd CLI commands, data model, hash-based IDs
- [Beads Plugin Documentation](https://github.com/steveyegge/beads/blob/main/docs/PLUGIN.md) -- existing Claude Code plugin integration pattern
- [mcp-go community SDK](https://github.com/mark3labs/mcp-go) -- alternative Go MCP implementation (fallback if official SDK has gaps)

---
*Architecture research for: gsd-wired (Claude Code plugin with MCP + hooks + Beads/Dolt)*
*Researched: 2026-03-21*
