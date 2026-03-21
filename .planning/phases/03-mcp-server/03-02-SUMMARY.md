---
phase: 03-mcp-server
plan: 02
subsystem: infra
tags: [go, mcp, graph, tools, json-schema, lazy-init, batch-writes]

requires:
  - phase: 03-mcp-server
    plan: 01
    provides: serverState with sync.Once lazy init, graph.Client batch mode, FlushWrites()

provides:
  - "registerTools function: 8 MCP tools backed by graph.Client methods"
  - "create_phase tool: creates GSD phase as epic bead"
  - "create_plan tool: creates GSD plan as task bead under phase epic"
  - "get_bead tool: retrieves a single bead by ID"
  - "list_ready tool: lists all unblocked beads"
  - "query_by_label tool: queries beads matching a label"
  - "claim_bead tool: atomically claims a bead for the current agent"
  - "close_plan tool: closes a plan bead, returns newly unblocked beads"
  - "flush_writes tool: flushes batch writes to Dolt (INFRA-10)"
  - "Serve() wired with serverState and registerTools — initialize responds before any Dolt work"

affects: [04-hooks]

tech-stack:
  added: []
  patterns:
    - "toolError/toolResult helpers reduce per-handler boilerplate: error path returns IsError=true, success path marshals data to TextContent"
    - "All tool handlers: state.init(ctx) first, then unmarshal args with json.Unmarshal, then call state.client method"
    - "JSON Schema as json.RawMessage with additionalProperties:false on all 8 tools (D-05)"
    - "closeResult struct captures {closed, unblocked} for close_plan wave-awareness response"
    - "NewInMemoryTransports() for in-process server/client testing — faster than subprocess for unit tests"
    - "Subprocess integration test (TestToolsListed) validates full JSON-RPC tools/list flow including notifications/initialized handshake"

key-files:
  created:
    - internal/mcp/tools.go
    - internal/mcp/tools_test.go
  modified:
    - internal/mcp/server.go
    - internal/mcp/server_test.go

key-decisions:
  - "NewInMemoryTransports for tool handler tests: faster than subprocess, avoids bd dependency for protocol-level tests"
  - "toolError/toolResult helpers: eliminates 5-line boilerplate per handler, consistent Content slice construction"
  - "closeResult struct: explicit JSON shape for close_plan response (closed + unblocked) matches wave-based workflow needs"
  - "server.go adds slog.Debug tools=8: self-documenting that tools are registered, visible in debug logs"

requirements-completed: [INFRA-02, INFRA-10]

duration: 7min
completed: 2026-03-21
---

# Phase 3 Plan 02: MCP Tool Registration Summary

**8 MCP tools with strict JSON Schema registered in Serve(), each calling state.init(ctx) for lazy graph initialization before delegating to graph.Client methods**

## Performance

- **Duration:** 7 min
- **Started:** 2026-03-21T19:15:31Z
- **Completed:** 2026-03-21T19:22:01Z
- **Tasks:** 2 (Task 1 TDD: RED + GREEN commits; Task 2: single commit)
- **Files modified:** 4

## Accomplishments

- 8 MCP tools registered via registerTools(server, state) with explicit JSON Schema (additionalProperties:false on all)
- Each handler: state.init(ctx) → unmarshal args → delegate to graph.Client → return JSON in TextContent
- toolError/toolResult helpers reduce per-handler boilerplate; errors use IsError=true (not Go error)
- close_plan returns closeResult{closed, unblocked} for wave-based workflow awareness
- flush_writes returns {status:"flushed"} enabling INFRA-10 batch boundary control
- Serve() wires serverState{} (lazy — no init) + registerTools before Run, so initialize responds before any Dolt work
- 14 tests total in mcp package: 6 unit (tools), 6 lazy-init, 2 integration (subprocess); all pass with -race
- Full regression green across all 7 packages; go vet clean; binary compiles

## Task Commits

1. **Task 1 RED: Failing tool tests** - `cb137d3` (test)
2. **Task 1 GREEN: Tool implementations** - `b3ec2c7` (feat)
3. **Task 2: Serve() wiring + TestToolsListed** - `fd79127` (feat)

## Files Created/Modified

- `internal/mcp/tools.go` - registerTools function with all 8 tool definitions and handlers
- `internal/mcp/tools_test.go` - 6 tests: TestToolsRegistered, TestToolCallCreatePhase, TestToolCallGetBead, TestToolCallBadArgs, TestToolCallInitError, TestToolCallFlushWrites
- `internal/mcp/server.go` - Serve() creates serverState{} and calls registerTools before Run()
- `internal/mcp/server_test.go` - Added TestToolsListed: subprocess test verifying 8 tools in tools/list response

## Decisions Made

- NewInMemoryTransports for tool handler tests: avoids subprocess overhead for unit-level testing, uses the SDK's own in-process transport
- toolError/toolResult helpers: consistent IsError=true pattern, eliminates repeated Content slice construction
- closeResult struct: explicit JSON shape for wave-awareness (who gets unblocked when a plan closes)
- slog.Debug with tools=8 in Serve(): self-documenting registration count visible in debug logs

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None - all tests passed on first implementation attempt.

## User Setup Required

None.

## Next Phase Readiness

- MCP server exposes all 8 graph tools: Claude Code can now create, query, claim, and close beads via MCP
- flush_writes gives Claude Code explicit batch boundary control (INFRA-10)
- Serve() signature unchanged — serve.go compatibility maintained
- Phase 3 complete: MCP server fully operational (Plan 01: infra + Plan 02: tools)
- Ready for Phase 4: Hook Dispatcher

## Self-Check: PASSED

- internal/mcp/tools.go: FOUND
- internal/mcp/tools_test.go: FOUND
- internal/mcp/server.go: FOUND (registerTools wired)
- internal/mcp/server_test.go: FOUND (TestToolsListed added)
- commit cb137d3 (test: RED tool tests): FOUND
- commit b3ec2c7 (feat: tool implementations): FOUND
- commit fd79127 (feat: Serve() wiring): FOUND

---
*Phase: 03-mcp-server*
*Completed: 2026-03-21*
