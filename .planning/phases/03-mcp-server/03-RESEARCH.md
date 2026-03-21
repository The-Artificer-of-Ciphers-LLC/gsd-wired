# Phase 3: MCP Server - Research

**Researched:** 2026-03-21
**Domain:** MCP Go SDK tool registration, lazy initialization, Dolt batched writes
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- D-01: gsdw is the developer interface. MCP tools are the machine interface (called by Claude Code, not humans). All internal mechanics are invisible to the developer.
- D-02: Developer never needs to manage Dolt, bd, or .beads/ directly. gsdw handles all infrastructure setup, recovery, and lifecycle.
- D-03: Tools optimized for machine consumption — Claude Code is the caller, not humans. Granularity at Claude's discretion.
- D-04: Only register tools with real graph backing from `internal/graph/`. No stubs. Tool surface grows per phase.
- D-05: Strict JSON Schema input schemas on tools — guides Claude Code to generate correct calls.
- D-06: MCP `initialize` response is immediate (sub-500ms). Dolt/beads init happens on first tool call.
- D-07: First tool call blocks until init completes — optimize for performance, developer doesn't see this.
- D-08: gsdw auto-creates `.beads/` directory via `bd init` when needed. Developer never runs `bd init` manually.
- D-09: Connection/session management at Claude's discretion — optimize for reliability and simplicity.
- D-10: On Dolt init failure: attempt recovery (retry, repair). If unrecoverable, shut down the MCP server process with a clear, actionable error message telling the developer what to fix.
- D-11: Write batching strategy at Claude's discretion. Optimize for the performance vs data safety tradeoff. Developer doesn't care about commit timing — only that work isn't lost.

### Claude's Discretion
- Tool list and granularity (1:1 with graph ops, or higher-level abstractions)
- Batching implementation (per-call commit, deferred flush, session boundary, or hybrid)
- Dolt connection lifecycle (keep-alive vs reconnect)
- Lazy init blocking vs async pattern
- Recovery strategy for Dolt failures (retry count, repair attempts)
- Error message format for unrecoverable failures

### Deferred Ideas (OUT OF SCOPE)
- Slash command registration in plugin manifest — Phase 5+
- Hook-triggered tool calls — Phase 4
- Token-aware context in tool responses — Phase 9
- Tool for project initialization flow — Phase 5
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| INFRA-02 | MCP server exposes tools via official Go SDK (v1.4.1) with lazy Dolt initialization | AddTool API verified directly from go-sdk source; lazy init pattern via sync.Once documented |
| INFRA-10 | Batched Dolt writes at wave boundaries to prevent write amplification | `--dolt-auto-commit=batch` flag + `bd dolt commit` verified from live bd CLI |
</phase_requirements>

## Summary

Phase 3 wires the existing MCP server skeleton (`internal/mcp/server.go`) and graph layer (`internal/graph/`) together by registering real tools. The go-sdk v1.4.1 provides two tool registration paths: `server.AddTool()` (raw, with `json.RawMessage` schema) and the generic `mcp.AddTool[In, Out]()` (typed, with automatic schema inference). For this project, raw `server.AddTool()` with explicit `json.RawMessage` input schemas is the right choice — it avoids the generic's schema inference overhead and keeps schemas explicit and machine-readable per D-05.

Lazy initialization is straightforward with `sync.Once`: the `Serve()` function creates a `sync.Once` and a shared `*graph.Client`, and each tool handler calls `once.Do(initFunc)` before any graph operation. The first caller blocks until init completes (D-07). On failure, `sync.Once` stores the error and all subsequent callers get it immediately.

Batched Dolt writes are controlled via bd's `--dolt-auto-commit=batch` global flag. In batch mode, writes accumulate in Dolt's working set across multiple bd calls. A single `bd dolt commit` flushes them all. This maps directly to INFRA-10 — gsdw passes `--dolt-auto-commit=batch` on every bd invocation and calls `bd dolt commit` at wave boundaries.

**Primary recommendation:** Use `server.AddTool()` with raw JSON schemas, `sync.Once` for lazy init, and `--dolt-auto-commit=batch` + `bd dolt commit` for write batching.

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| github.com/modelcontextprotocol/go-sdk/mcp | v1.4.1 | Tool registration, StdioTransport, CallToolResult | Already in go.mod; official SDK; used in Phase 1 |
| sync (stdlib) | Go 1.26.1 | sync.Once for lazy init | Zero-cost after first call; goroutine-safe by definition |
| encoding/json (stdlib) | Go 1.26.1 | json.RawMessage for raw input schemas, tool arg unmarshaling | Already used throughout project |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| log/slog (stdlib) | Go 1.26.1 | Structured debug logging of tool calls | All tool handlers; already the project standard |
| context (stdlib) | Go 1.26.1 | Propagate cancellation into graph.Client.run() | Every tool handler receives ctx |
| os/exec (stdlib) | Go 1.26.1 | bd init subprocess | Already used in graph.Client.run() |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| server.AddTool() raw | mcp.AddTool[In, Out]() generic | Generic auto-infers schema from Go structs — less explicit, harder to control exactly what Claude Code sees. Raw is better when D-05 (strict, explicit schemas) is the goal. |
| sync.Once | channel-based init | sync.Once is simpler, well-understood, zero allocation after init. Channel approach adds complexity with no benefit here. |
| --dolt-auto-commit=batch per call | per-call commit (default "off") | Per-call is safer but causes write amplification. Batch mode accumulates writes and flushes on explicit `bd dolt commit`. |

**Installation:** No new packages required. go-sdk v1.4.1 already in go.mod.

## Architecture Patterns

### Recommended Project Structure
```
internal/mcp/
├── server.go        # Serve() — creates server, registers tools, runs stdio transport
├── tools.go         # Tool definitions and handlers (new file)
├── init.go          # Lazy init logic — sync.Once, graph.Client lifecycle (new file)
└── server_test.go   # Existing subprocess integration test; extend with tool call tests
```

### Pattern 1: Raw Tool Registration with json.RawMessage Schema
**What:** `server.AddTool()` takes `*mcp.Tool` with `InputSchema json.RawMessage` and a `ToolHandler func(context.Context, *CallToolRequest) (*CallToolResult, error)`.
**When to use:** Always — the raw form gives explicit control over the JSON Schema sent to Claude Code (D-05 compliance).
**Example:**
```go
// Source: /Users/trekkie/go/pkg/mod/github.com/modelcontextprotocol/go-sdk@v1.4.1/mcp/tool_example_test.go
server.AddTool(&mcp.Tool{
    Name:        "bead_get",
    Description: "Get a bead by ID from the beads graph",
    InputSchema: json.RawMessage(`{
        "type": "object",
        "properties": {
            "id": {"type": "string", "description": "Bead ID"}
        },
        "required": ["id"],
        "additionalProperties": false
    }`),
}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    var args struct{ ID string `json:"id"` }
    if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
        return nil, err
    }
    // ... call graph.Client
    return &mcp.CallToolResult{
        Content: []mcp.Content{&mcp.TextContent{Text: resultJSON}},
    }, nil
})
```

**CRITICAL:** `server.AddTool()` panics if `InputSchema` is nil. Always provide a schema. The schema must be type "object" or the server panics.

### Pattern 2: Lazy Initialization with sync.Once
**What:** A `sync.Once` wraps the `bd init` check and `graph.NewClient()` creation. All tool handlers call `once.Do(initFunc)` before using the client.
**When to use:** Mandatory — D-06 requires `initialize` to return immediately, D-07 requires first tool call to block until init is done.
**Example:**
```go
// Source: stdlib sync package + project patterns
type lazyInit struct {
    once   sync.Once
    client *graph.Client
    err    error
}

func (l *lazyInit) do(ctx context.Context) error {
    l.once.Do(func() {
        beadsDir := findOrCreateBeadsDir()
        if err := runBdInit(ctx, beadsDir); err != nil {
            l.err = err
            return
        }
        c, err := graph.NewClient(beadsDir)
        if err != nil {
            l.err = err
            return
        }
        l.client = c
    })
    return l.err
}
```

**Note:** `sync.Once` does NOT retry on failure. If `initFunc` errors, subsequent `once.Do` calls are no-ops and the stored error is returned. This is correct behavior for D-10 (unrecoverable = shut down).

### Pattern 3: Tool Error Reporting — IsError vs Protocol Error
**What:** MCP distinguishes between tool errors (packed into the result) and protocol errors (returned as Go error from the handler).
- Return `(*mcp.CallToolResult, error)` where `error != nil` → protocol error (JSON-RPC error to client)
- Return `&mcp.CallToolResult{IsError: true, Content: ...}` → tool-level error (Claude Code sees error in tool result, can handle it)
**When to use:** Return tool-level errors for expected failures (bead not found, graph op failed). Return protocol errors only for programmer bugs (nil schema, marshal failure).
**Example:**
```go
// Source: go-sdk mcp/server.go lines 340-354
if graphErr != nil {
    return &mcp.CallToolResult{
        IsError: true,
        Content: []mcp.Content{&mcp.TextContent{Text: graphErr.Error()}},
    }, nil
}
```

### Pattern 4: Batched Dolt Writes via --dolt-auto-commit=batch
**What:** Pass `--dolt-auto-commit=batch` to every bd invocation. Writes accumulate in the working set. Call `bd dolt commit` to flush. SIGTERM/SIGHUP also flush.
**When to use:** All write operations during a wave. Flush at the wave boundary or session end.
**Implementation:** In `graph.Client.run()`, append `--dolt-auto-commit=batch` to the env or args for write operations. Expose a `FlushWrites(ctx)` method that shells out to `bd dolt commit --json`.
**Example:**
```go
// bd dolt commit flushes accumulated batch writes
func (c *Client) FlushWrites(ctx context.Context) error {
    _, err := c.run(ctx, "dolt", "commit", "--message", "gsdw: wave boundary flush")
    return err
}
```

**Note:** `bd init` takes `--quiet` flag to suppress output (non-interactive by default once given explicit flags). The key flags for non-interactive init: `--quiet`, `--skip-hooks` (optional), `--skip-agents` (optional if AGENTS.md not needed here).

### Pattern 5: bd init Automation
**What:** gsdw auto-runs `bd init` when `.beads/` does not exist. `bd init` is non-interactive when run with explicit flags.
**Key flags:**
- `--quiet` / `-q` — suppress output to stdout (only errors on stderr)
- `--prefix <name>` — set issue prefix to project name (avoids interactive prompt)
- `--skip-hooks` — skip git hook installation (gsdw manages its own hooks)
- `--skip-agents` — skip AGENTS.md generation (gsdw manages its own context files)
**Detection:** Check for `.beads/` directory existence with `os.Stat()`.

### Anti-Patterns to Avoid
- **Registering tools before `server.Run()`:** Tools can be added before or after `Run()` — the server notifies clients of list changes. But for the lazy-init pattern, tools must be registered before `Run()` so Claude Code discovers them in `tools/list`.
- **Returning nil from `server.AddTool` handler with no content:** The SDK handles nil `CallToolResult` only in the typed `AddTool[In,Out]` variant. With raw `server.AddTool`, always return a non-nil `*CallToolResult`.
- **Using `--dolt-auto-commit=on` for high-frequency writes:** Each write triggers a Dolt commit — causes write amplification. Use `batch` mode.
- **Ignoring `sync.Once` error:** `once.Do` runs exactly once. If init errors, the client is nil. All tool handlers must check `l.err` after calling `l.do()`.
- **Writing to stdout in tool handlers:** MCP stdio transport owns stdout. Any stray write breaks the JSON-RPC framing. All tool handler logging goes to `slog` (which writes to stderr).

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| JSON Schema validation of tool inputs | Custom validator | sdk auto-validates with raw AddTool (panics on bad schema at register time) | SDK validates at registration; structural validation of args is caller's responsibility with raw form |
| Tool list caching | Manual list management | server.AddTool() — server tracks registered tools | go-sdk featureSet handles thread-safe storage and list-changed notifications |
| Thread-safe init gate | Mutex + bool flag | sync.Once | sync.Once is designed for exactly this; handles concurrent callers blocking on the first |
| Dolt commit accumulation | Write log / buffer | --dolt-auto-commit=batch | bd's built-in batch mode with working set accumulation |
| MCP JSON-RPC framing | Custom newline-delimited parser | mcp.StdioTransport | Already used in Phase 1; handles all framing details |

**Key insight:** The go-sdk handles all MCP protocol details. The project code only needs to: define tool schemas, unmarshal args, call graph.Client, marshal results.

## Common Pitfalls

### Pitfall 1: AddTool Panics on Missing or Non-Object Schema
**What goes wrong:** Calling `server.AddTool()` with `InputSchema: nil` or a schema whose `type` is not `"object"` causes an immediate panic.
**Why it happens:** The SDK enforces MCP spec requirement that tool input must be a JSON object.
**How to avoid:** Every tool MUST have `InputSchema: json.RawMessage(`{"type":"object",...}`)`. Verify at compile time by writing a test that calls `Serve()` (which registers tools) in a test binary.
**Warning signs:** Runtime panic in `server.AddTool` during startup.

### Pitfall 2: sync.Once Does Not Retry
**What goes wrong:** If bd init fails on first call (bd not on PATH, Dolt server not running), `sync.Once` marks initialization done. All subsequent tool calls get the stored error immediately — they never retry init.
**Why it happens:** By design. `sync.Once.Do` runs the function exactly once.
**How to avoid:** Treat a failed init as fatal per D-10. Log a clear, actionable error message, then call `os.Exit(1)` or cancel the server context. Don't silently swallow the init error.
**Warning signs:** Tool calls return errors after first failure even after developer fixes the underlying issue (server restart required).

### Pitfall 3: bd init Requires Dolt Server Running
**What goes wrong:** `bd init` connects to Dolt on port 3307 (default). If no Dolt server is running, `bd init` fails with a connection error.
**Why it happens:** bd v0.61+ uses server-mode Dolt exclusively. The legacy embedded mode is removed.
**How to avoid:** The error message from `bd init` is actionable: "start dolt sql-server first." Pass it through to the developer (D-10). Do NOT attempt to auto-start dolt — that's out of scope per D-02 constraint intent.
**Warning signs:** `bd init` exits non-zero with "connection refused" or "no server detected".

### Pitfall 4: BEADS_DIR Must Point to Parent of .beads/, Not .beads/ Itself
**What goes wrong:** `bd` expects `BEADS_DIR` to be the directory *containing* `.beads/` (i.e., the project root), not the `.beads/` directory itself.
**Why it happens:** bd auto-discovers `.beads/` within `BEADS_DIR`.
**How to avoid:** Set `beadsDir` in `graph.Client` to the project root. The existing `graph.NewClient(beadsDir string)` already handles this correctly — document the contract clearly in init logic.
**Warning signs:** `bd: no .beads/ directory found` despite directory existing.

### Pitfall 5: Tool Handler Must Not Block the MCP goroutine Indefinitely
**What goes wrong:** If `once.Do(initFunc)` blocks indefinitely (e.g., bd init hangs waiting for Dolt), Claude Code's tool call times out and the MCP session may be torn down.
**Why it happens:** `once.Do` is synchronous; no timeout is built in.
**How to avoid:** Wrap `bd init` subprocess call with `exec.CommandContext(ctx, ...)` using a reasonable timeout (e.g., 30s). If context is cancelled, `once.Do` returns and the error propagates to Claude Code as a tool error.
**Warning signs:** Tool call hangs with no response; Claude Code shows timeout.

### Pitfall 6: Batch Mode Working Set Lost on SIGKILL
**What goes wrong:** `--dolt-auto-commit=batch` defers commits. If the process is killed with SIGKILL (not SIGTERM), uncommitted writes are lost.
**Why it happens:** SIGTERM/SIGHUP flush the batch; SIGKILL does not.
**How to avoid:** For critical operations (create bead, close bead), consider calling `FlushWrites()` immediately after. For non-critical reads, batch mode is fine. For this phase's tool surface, call flush after any write tool completes.
**Warning signs:** Beads disappear after unexpected gsdw termination.

## Code Examples

Verified patterns from official sources and live codebase:

### Tool Registration (raw form)
```go
// Source: go-sdk@v1.4.1/mcp/tool_example_test.go ExampleServer_AddTool_rawSchema
server.AddTool(&mcp.Tool{
    Name:        "bead_create_phase",
    Description: "Create a GSD phase as an epic bead",
    InputSchema: json.RawMessage(`{
        "type": "object",
        "properties": {
            "phase_num":   {"type": "integer", "description": "Phase number (1-10)"},
            "title":       {"type": "string", "description": "Phase title"},
            "goal":        {"type": "string", "description": "Phase goal"},
            "acceptance":  {"type": "string", "description": "Acceptance criteria"},
            "req_ids":     {"type": "array", "items": {"type": "string"}, "description": "Requirement IDs"}
        },
        "required": ["phase_num", "title", "goal", "acceptance"],
        "additionalProperties": false
    }`),
}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    if err := li.do(ctx); err != nil {  // lazy init check
        return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}}}, nil
    }
    var args struct {
        PhaseNum   int      `json:"phase_num"`
        Title      string   `json:"title"`
        Goal       string   `json:"goal"`
        Acceptance string   `json:"acceptance"`
        ReqIDs     []string `json:"req_ids"`
    }
    if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
        return nil, err // protocol error — malformed args
    }
    bead, err := li.client.CreatePhase(ctx, args.PhaseNum, args.Title, args.Goal, args.Acceptance, args.ReqIDs)
    if err != nil {
        return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}}}, nil
    }
    result, _ := json.Marshal(bead)
    return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(result)}}}, nil
})
```

### Lazy Init Structure
```go
// Source: stdlib sync.Once + project patterns
type serverState struct {
    once   sync.Once
    client *graph.Client
    err    error
}

func (s *serverState) init(ctx context.Context, beadsDir string) error {
    s.once.Do(func() {
        // Check if .beads/ exists; if not, run bd init
        if _, statErr := os.Stat(filepath.Join(beadsDir, ".beads")); os.IsNotExist(statErr) {
            initCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
            defer cancel()
            cmd := exec.CommandContext(initCtx, bdPath, "init", "--quiet", "--skip-hooks", "--skip-agents", "--prefix", projectName)
            cmd.Dir = beadsDir
            if out, err := cmd.CombinedOutput(); err != nil {
                s.err = fmt.Errorf("bd init failed: %w\n%s\nFix: ensure dolt sql-server is running on port 3307", err, out)
                return
            }
        }
        c, err := graph.NewClient(beadsDir)
        if err != nil {
            s.err = err
            return
        }
        s.client = c
    })
    return s.err
}
```

### Serve() with tools registered
```go
// Source: go-sdk@v1.4.1/mcp/server.go + Phase 1 server.go
func Serve(ctx context.Context) error {
    server := mcp.NewServer(&mcp.Implementation{
        Name:    "gsd-wired",
        Version: version.String(),
    }, nil)

    state := &serverState{}
    registerTools(server, state)

    slog.Debug("mcp server starting on stdio")
    return server.Run(ctx, &mcp.StdioTransport{})
}
```

### Batched Write Pattern (bd dolt commit)
```go
// Source: live bd CLI -- bd dolt commit --help verified 2026-03-21
// Pass --dolt-auto-commit=batch as a global flag on every write bd call
// Then call FlushWrites() at wave boundary

// In graph.Client.run(), for write operations:
args = append([]string{"--dolt-auto-commit=batch"}, args...)

// FlushWrites flushes all accumulated batch writes.
func (c *Client) FlushWrites(ctx context.Context) error {
    _, err := c.run(ctx, "dolt", "commit", "--message", "gsdw: batch flush")
    return err
}
```

### Calling TextContent result
```go
// Source: go-sdk@v1.4.1/mcp/tool_example_test.go
return &mcp.CallToolResult{
    Content: []mcp.Content{&mcp.TextContent{Text: jsonStr}},
}, nil
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| mark3labs/mcp-go (community) | modelcontextprotocol/go-sdk (official) | 2025 | go-sdk is co-maintained by Google; authoritative; already in go.mod |
| AddTool[In,Out] generic | server.AddTool() raw for explicit schema control | go-sdk v1.4.1 | Both exist; choose based on schema explicitness requirement |
| bd embedded SQLite | bd Dolt server mode (port 3307) | bd v0.61+ | SQLite backend removed; must have Dolt server running |
| --dolt-auto-commit=off (default) | --dolt-auto-commit=batch for write efficiency | bd current | Batch mode defers commits; explicit flush via `bd dolt commit` |

**Deprecated/outdated:**
- `bd --backend=sqlite`: Removed in bd current. Prints deprecation/migration instructions if used.
- `mcp.Server.Run()` with blocking pattern: Phase 1 uses `server.Run(ctx, &mcp.StdioTransport{})` which is still correct for v1.4.1.

## Open Questions

1. **beadsDir location — project root vs. gsdw-specific subdirectory**
   - What we know: `BEADS_DIR` is set to the project root in graph.Client. `bd init` creates `.beads/` there.
   - What's unclear: For Phase 3, what is "the project root"? The user's working directory when `gsdw serve` starts? Or a fixed location?
   - Recommendation: Use the user's working directory (os.Getwd() at serve time), consistent with how bd resolves `.beads/` normally. Document this contract.

2. **Tool granularity for Phase 3 initial surface**
   - What we know: D-04 says only register tools with real graph backing. D-03 says granularity at Claude's discretion.
   - What's unclear: Which subset of graph.Client methods to expose in Phase 3 vs. later phases?
   - Recommendation: Expose the minimal set needed for Phase 4 hooks to function: CreatePhase, CreatePlan, GetBead, ListReady, ClaimBead, ClosePlan. Omit FlushWrites exposure as a tool (it's internal plumbing).

3. **bd init --prefix value**
   - What we know: `--prefix` sets the issue prefix (e.g., "gsd-wired"). Without it, bd uses the current directory name.
   - What's unclear: Should gsdw use a fixed prefix or derive from project name?
   - Recommendation: Default to current directory name (omit --prefix; let bd default). Claude's discretion per D-09.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing stdlib |
| Config file | none (go test ./...) |
| Quick run command | `go test ./internal/mcp/... -race -count=1` |
| Full suite command | `go test ./... -race -count=1` |

### Phase Requirements to Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| INFRA-02 | tools/list returns registered tools | integration (subprocess) | `go test ./internal/mcp/... -run TestToolsListed -race` | No — Wave 0 |
| INFRA-02 | tool call with valid args reaches graph.Client | unit (fake bd) | `go test ./internal/mcp/... -run TestToolCall -race` | No — Wave 0 |
| INFRA-02 | initialize returns immediately (sub-500ms) | integration (subprocess) | `go test ./internal/mcp/... -run TestInitializeLatency -race` | No — Wave 0 |
| INFRA-02 | first tool call triggers lazy init | unit (fake bd) | `go test ./internal/mcp/... -run TestLazyInit -race` | No — Wave 0 |
| INFRA-02 | tool call with invalid args returns IsError=true | unit | `go test ./internal/mcp/... -run TestToolCallBadArgs -race` | No — Wave 0 |
| INFRA-10 | write calls use --dolt-auto-commit=batch | unit (arg capture) | `go test ./internal/graph/... -run TestBatchFlag -race` | No — Wave 0 |
| INFRA-10 | FlushWrites calls bd dolt commit | unit (arg capture) | `go test ./internal/graph/... -run TestFlushWrites -race` | No — Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./internal/mcp/... -race -count=1`
- **Per wave merge:** `go test ./... -race -count=1`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `internal/mcp/tools_test.go` — unit tests for tool registration, arg parsing, lazy init, error paths
- [ ] `internal/mcp/init_test.go` — unit tests for serverState.init() with fake bd
- [ ] `internal/graph/batch_test.go` — tests for --dolt-auto-commit=batch arg and FlushWrites

*(Existing `internal/mcp/server_test.go` covers INFRA-02 basic initialize — extend it for tool call tests)*

## Sources

### Primary (HIGH confidence)
- `/Users/trekkie/go/pkg/mod/github.com/modelcontextprotocol/go-sdk@v1.4.1/mcp/tool.go` — ToolHandler, ToolHandlerFor signatures
- `/Users/trekkie/go/pkg/mod/github.com/modelcontextprotocol/go-sdk@v1.4.1/mcp/server.go` — AddTool() implementation, panic conditions, tool validation
- `/Users/trekkie/go/pkg/mod/github.com/modelcontextprotocol/go-sdk@v1.4.1/mcp/tool_example_test.go` — canonical usage patterns for raw and typed AddTool
- `~/.local/bin/bd dolt commit --help` — `--dolt-auto-commit` flag, batch mode behavior, SIGTERM flush guarantee
- `~/.local/bin/bd init --help` — non-interactive flags: --quiet, --skip-hooks, --skip-agents, --prefix
- `/Users/trekkie/projects/gsd-wired/internal/mcp/server.go` — existing Serve() structure (Phase 1)
- `/Users/trekkie/projects/gsd-wired/internal/graph/client.go` — run() pattern, BEADS_DIR, two-tier error handling

### Secondary (MEDIUM confidence)
- `/Users/trekkie/go/pkg/mod/github.com/modelcontextprotocol/go-sdk@v1.4.1/mcp/mcp_example_test.go` — lifecycle, cancellation, progress patterns
- `/Users/trekkie/go/pkg/mod/github.com/modelcontextprotocol/go-sdk@v1.4.1/mcp/server_example_test.go` — prompt/resource registration patterns (analogous to tool registration)
- `.planning/phases/01-binary-scaffold/01-01-SUMMARY.md` — established patterns for MCP serve, subprocess test approach
- `.planning/phases/02-graph-primitives/02-01-SUMMARY.md` — graph.Client API, fake_bd test pattern

### Tertiary (LOW confidence)
- None — all critical claims verified from source files directly.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — verified from go-sdk source and go.mod
- Architecture: HIGH — patterns derived from live SDK source code and working Phase 1/2 code
- Pitfalls: HIGH — AddTool panic conditions verified from server.go source; bd init behavior from live CLI
- Batched writes: HIGH — `bd dolt commit --help` verified live on this machine

**Research date:** 2026-03-21
**Valid until:** 2026-06-21 (go-sdk is stable; bd batch mode is established behavior)
