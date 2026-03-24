---
name: graph-operations
description: Implements beads graph operations in internal/graph/ using the bd CLI wrapper pattern. Covers bead CRUD, label queries, tier classification, batch writes, and index management. Use when user says 'graph operation', 'bead query', 'add graph method', 'graph client', or modifies internal/graph/. Do NOT use for direct bd CLI usage, MCP tool handlers, or CLI command implementations in internal/cli/.
---
# Graph Operations

## Critical

1. **All graph operations go through `Client.run()` or `Client.runWrite()`** — never call `exec.Command("bd", ...)` directly. The `run()` method handles `--json` appending, `BEADS_DIR` env injection, connection config env vars, and two-tier error parsing.
2. **Read operations use `run()`. Write operations use `runWrite()`.** `runWrite()` conditionally prepends `--dolt-auto-commit=batch` when `batchMode` is true. Mixing these up breaks batch commit semantics.
3. **`FlushWrites()` uses `run()`, not `runWrite()`** — the commit itself is not a batched operation.
4. **Labels are comma-separated strings, not arrays.** When passing multiple labels to bd, join with `,` (e.g., `"gsd:phase," + strings.Join(reqIDs, ",")`).
5. **Always pass `--limit`, `0`** on list/query operations to prevent silent truncation (bd defaults to 10).

## Instructions

### Step 1: Define the Method Signature

Add the method to the appropriate file based on operation type:
- **Read operations** → `internal/graph/query.go`
- **Create operations** → `internal/graph/create.go`
- **Update/close operations** → `internal/graph/update.go`
- **Tier/classification** → `internal/graph/tier.go`
- **Index management** → `internal/graph/index.go`

All methods are on `*Client` and take `context.Context` as the first parameter. Return the appropriate type + `error`.

**Verify**: Method is in the correct file for its operation type before proceeding.

### Step 2: Implement the bd CLI Call

Follow the established pattern exactly:

```go
// Read operation pattern:
func (c *Client) MyQuery(ctx context.Context, param string) ([]Bead, error) {
    out, err := c.run(ctx, "subcommand", "--flag", param, "--limit", "0")
    if err != nil {
        return nil, err
    }
    var beads []Bead
    if err := json.Unmarshal(out, &beads); err != nil {
        return nil, err
    }
    return beads, nil
}

// Write operation pattern:
func (c *Client) MyMutation(ctx context.Context, beadID string) (*Bead, error) {
    out, err := c.runWrite(ctx, "subcommand", beadID)
    if err != nil {
        return nil, err
    }
    var bead Bead
    if err := json.Unmarshal(out, &bead); err != nil {
        return nil, err
    }
    return &bead, nil
}
```

Key rules:
- `run()` for reads, `runWrite()` for writes — never the reverse
- Do NOT append `--json` — `run()` does this automatically
- Unmarshal into `Bead`, `[]Bead`, or a purpose-built struct from `bead.go`
- Return `*Bead` for single results, `[]Bead` for lists

**Verify**: Method uses `run()` for reads or `runWrite()` for writes. No manual `--json` flag.

### Step 3: Handle Metadata and Labels

When creating beads with metadata:
```go
meta := map[string]any{"gsd_phase": phaseNum, "custom_key": value}
metaJSON, err := json.Marshal(meta)
if err != nil {
    return nil, fmt.Errorf("marshal metadata: %w", err)
}
args = append(args, "--metadata", string(metaJSON))
```

When adding labels:
```go
// Single label
args = append(args, "--label", "gsd:mylabel")
// Multiple labels: comma-separated, single --label flag
labelStr := strings.Join(labels, ",")
args = append(args, "--label", labelStr)
```

Use `--no-inherit-labels` when creating child beads that should not inherit parent labels (see `CreatePlan` pattern).

**Verify**: Metadata is `map[string]any` marshaled to JSON string. Labels are comma-joined.

### Step 4: Handle Compaction for Close Operations

When closing beads, follow the `ClosePlan` pattern:
1. Snapshot ready beads before close
2. Perform the close via `runWrite()`
3. Compact the closed bead with `gsd:compact` metadata (best-effort, don't fail on error)
4. Snapshot ready beads after close
5. Return the diff as newly unblocked beads

```go
// Best-effort compaction (D-12 pattern)
compact := bead.Title
if bead.CloseReason != "" {
    compact += ": " + bead.CloseReason
}
_ = c.CompactBead(ctx, beadID, compact) // best-effort
```

**Verify**: Compaction errors are swallowed (best-effort). Ready-diff logic is present for close ops.

### Step 5: Add New Types to bead.go

If the operation needs a new response type, add it to `internal/graph/bead.go`. Follow existing conventions:
- Use `json` struct tags matching bd's JSON output
- Use `omitempty` for optional fields
- Use `*time.Time` for nullable timestamps
- Use `map[string]any` for metadata (not `map[string]string`)

**Verify**: New types are in `bead.go`, not scattered across operation files.

### Step 6: Write Tests Using fake_bd

Add tests to the appropriate `*_test.go` file. Follow the established pattern:

```go
func TestMyOperation(t *testing.T) {
    // Use the pre-built fake_bd binary from TestMain
    captureFile := filepath.Join(t.TempDir(), "args.json")
    beadsDir := t.TempDir()

    client := NewClientWithPath(fakeBdPath, beadsDir)
    t.Setenv("FAKE_BD_CAPTURE_FILE", captureFile)

    result, err := client.MyOperation(context.Background(), "arg1")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    // Verify args passed to bd
    raw, _ := os.ReadFile(captureFile)
    var args []string
    json.Unmarshal(raw, &args)
    mustContain(t, args, "subcommand")
    mustContain(t, args, "--json")
}
```

For batch mode tests, use `NewClientWithPathBatch` and verify `--dolt-auto-commit=batch` appears before the subcommand.

To test with custom bd responses, set `FAKE_BD_*_RESPONSE` env vars pointing to JSON fixture files.

**Verify**: Test uses `NewClientWithPath(fakeBdPath, ...)`, not `NewClient()`. No real bd/dolt dependency.

### Step 7: Update fake_bd if Needed

If your new operation uses a bd subcommand not yet handled by `testdata/fake_bd/main.go`, add a case to its switch statement:

```go
case "mysubcommand":
    fmt.Print(cannedBead) // or a new canned response
```

Keep canned responses minimal — just enough JSON to unmarshal into the expected type.

**Verify**: `go build ./internal/graph/testdata/fake_bd/` succeeds. Tests pass with `go test ./internal/graph/`.

## Examples

**User says**: "Add a method to query beads by status"

**Actions**:
1. Add `QueryByStatus` to `internal/graph/query.go`:
```go
func (c *Client) QueryByStatus(ctx context.Context, status string) ([]Bead, error) {
    out, err := c.run(ctx, "query", "status="+status, "--limit", "0")
    if err != nil {
        return nil, err
    }
    var beads []Bead
    if err := json.Unmarshal(out, &beads); err != nil {
        return nil, err
    }
    return beads, nil
}
```
2. Add test in `graph_test.go` using `NewClientWithPath` and `FAKE_BD_CAPTURE_FILE`
3. Verify args contain `query`, `status=open`, `--limit`, `0`, `--json`

**Result**: New read method follows `run()` + unmarshal pattern, tested via fake_bd arg capture.

## Common Issues

- **`bd not found on PATH`**: Tests must use `NewClientWithPath(fakeBdPath, beadsDir)`, not `NewClient()`. Real `bd` is not available in CI.
- **`json: cannot unmarshal object into Go value of type []graph.Bead`**: bd returned a single object but you're unmarshaling into a slice. Use `*Bead` for `show`-style commands, `[]Bead` for list commands.
- **Batch flag appears after subcommand**: `runWrite()` prepends `--dolt-auto-commit=batch` before all args. If you manually build args with the subcommand first, the flag order is correct. Do not re-prepend.
- **Silent truncation in list results**: If results seem incomplete, ensure `--limit`, `0` is passed. bd defaults to returning only 10 results.
- **`BEADS_DIR` not set error in tests**: Pass a valid temp directory as `beadsDir` to the client constructor: `NewClientWithPath(fakeBdPath, t.TempDir())`.
- **Metadata not appearing on created bead**: Metadata must be `json.Marshal`'d to a string and passed via `--metadata`. Do not pass a Go map directly as a CLI arg.