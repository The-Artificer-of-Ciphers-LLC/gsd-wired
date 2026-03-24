---
name: add-mcp-tool
description: Creates a new MCP tool following the handle*() pattern in internal/mcp/. Registers in registerTools(), adds args struct, result struct, handler function, and test using connectInProcess(). Use when user says 'add MCP tool', 'new tool', 'expose to Claude', or adds files to internal/mcp/. Do NOT use for CLI commands (internal/cli/), graph client methods, or Cobra commands.
---
# Add MCP Tool

## Critical

- **Error contract (D-09):** Tool/business errors return `toolError(msg)` with `IsError=true` and `nil` Go error. Go errors are ONLY for protocol failures. Never return both.
- **Lazy init required (D-06, D-07):** Every handler MUST call `state.init(ctx)` before any graph operation. Return `toolError(err.Error()), nil` on init failure.
- **Tool count:** After adding, update the tool count in `registerTools()` comment AND `TestToolsRegistered` assertion. Currently 19 — your tool makes it N+1.
- **No `nil` slices in results:** Initialize empty slices as `[]Type{}` not `nil` to ensure clean JSON (`[]` not `null`).

## Instructions

1. **Define the args and result structs.** For simple tools (≤3 fields), use an inline anonymous struct in `tools.go`. For complex tools (4+ fields or reusable logic), create `internal/mcp/<tool_name>.go`.

   Naming: `<toolName>Args`, `<toolName>Result`. JSON tags use `snake_case` with `omitempty` for optional fields.
   ```go
   type myToolArgs struct {
       PhaseNum int    `json:"phase_num"`
       Title    string `json:"title"`
       ReqIDs   []string `json:"req_ids,omitempty"`
   }
   type myToolResult struct {
       BeadID string `json:"bead_id"`
   }
   ```
   Verify: All required fields have no `omitempty`. JSON tags match the InputSchema property names.

2. **Write the handler function** (if using a dedicated file). Signature:
   ```go
   func handleMyTool(ctx context.Context, state *serverState, args myToolArgs) (*mcpsdk.CallToolResult, error) {
       if err := state.init(ctx); err != nil {
           return toolError(err.Error()), nil
       }
       // ... call state.client methods ...
       return toolResult(&myToolResult{BeadID: bead.ID})
   }
   ```
   Imports: `mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"` and `"github.com/The-Artificer-of-Ciphers-LLC/gsd-wired/internal/graph"` as needed.
   Verify: `state.init(ctx)` is the first call. All error paths return `toolError(...)`, nil.

3. **Register the tool in `registerTools()`** in `internal/mcp/tools.go`. Add a new `server.AddTool()` block following the existing pattern:
   ```go
   // my_tool — Short description of what it does.
   server.AddTool(&mcpsdk.Tool{
       Name:        "my_tool",
       Description: "Full description for Claude to read.",
       InputSchema: json.RawMessage(`{"type":"object","properties":{"phase_num":{"type":"integer","description":"Phase number"},"title":{"type":"string","description":"Title"}},"required":["phase_num","title"],"additionalProperties":false}`),
   }, func(ctx context.Context, req *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
       var args myToolArgs
       if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
           return toolError("invalid arguments: " + err.Error()), nil
       }
       return handleMyTool(ctx, state, args)
   })
   ```
   For inline handlers (simple tools), put `state.init(ctx)` inside the closure directly instead of delegating.
   Verify: InputSchema JSON is valid. `"additionalProperties":false` is set. `"required"` lists all mandatory fields.

4. **Update the tool count.** In `tools.go`, update the `registerTools` doc comment (`// registerTools registers all N GSD MCP tools`). In `tools_test.go`, update `TestToolsRegistered`:
   - Change the count assertion: `if len(result.Tools) != N {`
   - Add your tool name to the `wantNames` slice.
   Verify: `go test ./internal/mcp/ -run TestToolsRegistered` passes.

5. **Write the test** in `internal/mcp/tools_test.go` (simple) or `internal/mcp/<tool_name>_test.go` (complex). Follow this pattern:
   ```go
   func TestToolCallMyTool(t *testing.T) {
       tmpDir := t.TempDir()
       if err := os.MkdirAll(filepath.Join(tmpDir, ".beads"), 0755); err != nil {
           t.Fatal(err)
       }
       state := &serverState{beadsDir: tmpDir, bdPath: fakeBdPathMCP}
       if err := state.init(context.Background()); err != nil {
           t.Fatalf("state.init() failed: %v", err)
       }
       cs := connectInProcess(t, state)

       result, err := cs.CallTool(context.Background(), &mcpsdk.CallToolParams{
           Name: "my_tool",
           Arguments: map[string]any{
               "phase_num": 1,
               "title":     "Test",
           },
       })
       if err != nil {
           t.Fatalf("CallTool(my_tool) returned error: %v", err)
       }
       if result.IsError {
           t.Fatalf("CallTool(my_tool) returned IsError=true: %v", contentText(result))
       }
       var resp myToolResult
       if err := json.Unmarshal([]byte(contentText(result)), &resp); err != nil {
           t.Fatalf("response is not valid JSON: %v", err)
       }
       if resp.BeadID == "" {
           t.Error("expected non-empty bead_id")
       }
   }
   ```
   Also add a bad-args subtest if the tool has required fields with specific types.
   Verify: `go test ./internal/mcp/ -run TestToolCallMyTool -v` passes.

6. **Run full test suite.** `go test ./internal/mcp/` — all tests must pass including the updated tool count.

## Examples

**User says:** "Add an MCP tool that lets Claude update a bead's close reason"

**Actions:**
1. Create `internal/mcp/update_close_reason.go` with `updateCloseReasonArgs` (ID, CloseReason string) and handler `handleUpdateCloseReason`.
2. Handler calls `state.init(ctx)`, then `state.client.GetBead()` + `state.client.ClosePlan()` (or appropriate graph method).
3. Register `update_close_reason` in `registerTools()` with InputSchema requiring `id` and `close_reason`.
4. Update count from 19→20 in comment and test assertion. Add `"update_close_reason"` to `wantNames`.
5. Write `TestToolCallUpdateCloseReason` using `connectInProcess` pattern.
6. Run `go test ./internal/mcp/` — all 20 tools registered, handler test passes.

**Result:** New tool appears in Claude's tool list via MCP stdio transport.

## Common Issues

- **`TestToolsRegistered` fails with "expected N tools, got M"**: You forgot to update the count assertion in `tools_test.go:45` or the `wantNames` slice. Both must match.
- **`invalid arguments` error at runtime**: InputSchema JSON and Go struct tags are mismatched. Verify property names in schema exactly match `json:"..."` tags. Common: schema has `phase_num` but struct tag says `phaseNum`.
- **`toolResult` returns `null` for slices**: Initialize slices as `make([]T, 0)` or `[]T{}` in result structs before marshalling.
- **Handler panics with nil client**: Missing `state.init(ctx)` call at the top of the handler. Every handler must init first.
- **`fakeBdPathMCP` undefined in new test file**: Add `var fakeBdPathMCP string` is only in `init_test.go` via `TestMain`. New `_test.go` files in the same package (`package mcp`) can access it. If you get undefined, ensure your test file uses `package mcp` not `package mcp_test`.
- **JSON schema parse error**: Ensure the `json.RawMessage` string uses escaped quotes correctly. Validate with `echo '{...}' | jq .` before embedding.