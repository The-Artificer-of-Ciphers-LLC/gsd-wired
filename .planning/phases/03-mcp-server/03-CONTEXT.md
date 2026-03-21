# Phase 3: MCP Server - Context

**Gathered:** 2026-03-21
**Status:** Ready for planning

<domain>
## Phase Boundary

The MCP server responds to protocol requests and exposes GSD tools with lazy database initialization. Tools wire through to `internal/graph/` for real operations. Delivers INFRA-02, INFRA-10.

</domain>

<decisions>
## Implementation Decisions

### Design principle
- **D-01:** gsdw is the developer interface. MCP tools are the machine interface (called by Claude Code, not humans). All internal mechanics are invisible to the developer.
- **D-02:** Developer never needs to manage Dolt, bd, or .beads/ directly. gsdw handles all infrastructure setup, recovery, and lifecycle.

### Tool surface
- **D-03:** Tools optimized for machine consumption — Claude Code is the caller, not humans. Granularity at Claude's discretion.
- **D-04:** Only register tools with real graph backing from `internal/graph/`. No stubs. Tool surface grows per phase (consistent with Phase 1 D-09 manifest-grows-per-phase).
- **D-05:** Strict JSON Schema input schemas on tools — guides Claude Code to generate correct calls.

### Lazy init behavior
- **D-06:** MCP `initialize` response is immediate (sub-500ms). Dolt/beads init happens on first tool call.
- **D-07:** First tool call blocks until init completes — optimize for performance, developer doesn't see this.
- **D-08:** gsdw auto-creates `.beads/` directory via `bd init` when needed. Developer never runs `bd init` manually.
- **D-09:** Connection/session management at Claude's discretion — optimize for reliability and simplicity.
- **D-10:** On Dolt init failure: attempt recovery (retry, repair). If unrecoverable, shut down the MCP server process with a clear, actionable error message telling the developer what to fix.

### Batched write strategy
- **D-11:** Write batching strategy at Claude's discretion. Optimize for the performance vs data safety tradeoff. Developer doesn't care about commit timing — only that work isn't lost.

### Claude's Discretion
- Tool list and granularity (1:1 with graph ops, or higher-level abstractions)
- Batching implementation (per-call commit, deferred flush, session boundary, or hybrid)
- Dolt connection lifecycle (keep-alive vs reconnect)
- Lazy init blocking vs async pattern
- Recovery strategy for Dolt failures (retry count, repair attempts)
- Error message format for unrecoverable failures

</decisions>

<specifics>
## Specific Ideas

- The existing `internal/mcp/server.go` already has a working MCP server that responds to `initialize` — Phase 3 adds tools to it and wires in the graph package
- `internal/graph/` already has all CRUD and query operations — tools are thin wrappers
- The 500ms `initialize` budget means no Dolt work during handshake — lazy init is mandatory
- Dolt write amplification (from research) is the reason batching matters — each `dolt commit` is expensive

</specifics>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase 1 and 2 foundation
- `internal/mcp/server.go` — Existing MCP server skeleton (stdio transport, initialize handler)
- `internal/graph/client.go` — bd CLI wrapper client (NewClient, run(), two-tier errors)
- `internal/graph/create.go` — CreatePhase, CreatePlan operations
- `internal/graph/query.go` — ListReady, ReadyForPhase, QueryByLabel, GetBead
- `internal/graph/update.go` — ClaimBead, ClosePlan, AddLabel
- `internal/graph/index.go` — Local index at .gsdw/index.json

### Project context
- `.planning/PROJECT.md` — Core value, constraints, key decisions
- `.planning/REQUIREMENTS.md` — INFRA-02, INFRA-10 define this phase's deliverables
- `.planning/ROADMAP.md` §Phase 3 — Success criteria (4 items that must be TRUE)

### Prior research
- `.planning/research/SUMMARY.md` — MCP Go SDK patterns, Dolt write amplification warnings

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/mcp/server.go` — MCP server with stdio transport, responds to initialize. Phase 3 adds tools here.
- `internal/graph/` — Full CRUD and query layer. Tools are thin wrappers around these functions.
- `internal/logging/logging.go` — slog dual-format to stderr. MCP server can log without stdout pollution.

### Established Patterns
- `mcp.NewServer()` + `server.Run(ctx, &mcp.StdioTransport{})` from Phase 1
- `graph.NewClient(beadsDir)` for bd operations
- Two-tier error handling (JSON on stdout, text on stderr) in graph client
- Injected io.Reader/io.Writer for testability

### Integration Points
- `internal/cli/serve.go` — `gsdw serve` calls `mcp.Serve(ctx)`. Phase 3 enriches what `Serve` sets up.
- `internal/graph/client.go` — MCP tools call graph.Client methods
- `.gsdw/index.json` — Local index used by tools for fast lookups
- `bd` CLI at `~/.local/bin/bd` — underlying graph operations

</code_context>

<deferred>
## Deferred Ideas

- Slash command registration in plugin manifest — Phase 5+
- Hook-triggered tool calls — Phase 4
- Token-aware context in tool responses — Phase 9
- Tool for project initialization flow — Phase 5

</deferred>

---

*Phase: 03-mcp-server*
*Context gathered: 2026-03-21*
