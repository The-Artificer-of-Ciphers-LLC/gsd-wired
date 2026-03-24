---
name: add-hook-handler
description: Creates a new hook event handler in internal/hook/ following the handle*() dispatcher pattern. Adds event constant to events.go, handler function, dispatcher case, and tests using fake bd binary. Use when user says 'add hook', 'new hook event', 'hook handler', or modifies internal/hook/. Do NOT use for MCP tools, CLI commands, or graph queries.
---
# Add Hook Handler

## Critical

- Every handler MUST emit valid JSON to stdout via `writeOutput()` — even on error paths. Never write partial output or mix logs with stdout.
- Handlers MUST degrade gracefully: log errors with `slog.Warn`/`slog.Error`, never return them to the caller unless the input JSON itself is invalid.
- The `hook_event_name` field in the JSON input MUST match the `event` parameter passed to `Dispatch()`. The dispatcher already validates this — do not duplicate.
- Hook processes are short-lived. Never spawn goroutines. Never block longer than 500ms for tool-use hooks or 2s for session hooks.

## Instructions

### Step 1 — Add Event Constant

Edit `internal/hook/events.go`:

1. Add a new constant following the naming pattern:
   ```go
   const (
       EventSessionStart = "SessionStart"
       EventPreToolUse   = "PreToolUse"
       EventPostToolUse  = "PostToolUse"
       EventPreCompact   = "PreCompact"
       EventYourEvent    = "YourEvent"  // <-- add here
   )
   ```
2. Append to `ValidEvents` slice:
   ```go
   var ValidEvents = []string{EventSessionStart, EventPreToolUse, EventPostToolUse, EventPreCompact, EventYourEvent}
   ```
3. If the event has unique input fields beyond `HookInputBase`, add a typed struct:
   ```go
   type YourEventInput struct {
       HookInputBase
       CustomField string `json:"custom_field"`
   }
   ```
4. If the event needs hook-specific output beyond `HookOutput`, add an output struct:
   ```go
   type YourEventHookOutput struct {
       AdditionalContext string `json:"additionalContext,omitempty"`
   }
   ```

**Verify:** `go vet ./internal/hook/` passes. `IsValidEvent("YourEvent")` returns true.

### Step 2 — Create Handler File

Create `internal/hook/your_event.go` (snake_case matching event name):

```go
package hook

import (
    "context"
    "encoding/json"
    "io"
    "log/slog"
)

func handleYourEvent(ctx context.Context, raw json.RawMessage, hs *hookState, w io.Writer) error {
    var in YourEventInput
    if err := json.Unmarshal(raw, &in); err != nil {
        return err  // only return errors for bad input JSON
    }

    // Fast path: check if .beads/ exists, bail early if nothing to do
    // ... handler logic ...

    // Best-effort graph operations with timeout
    ctx, cancel := context.WithTimeout(ctx, 400*time.Millisecond)
    defer cancel()

    // Always return valid output
    return writeOutput(w, HookOutput{
        AdditionalContext: "...",
    })
}
```

Key rules for the handler body:
- Signature is always `func handle<Event>(ctx context.Context, raw json.RawMessage, hs *hookState, w io.Writer) error`
- Unmarshal into the event-specific input type first
- Use `hs.initGraph(cwd)` to lazily get graph client (called via `hs.once`)
- Use `writeOutput(w, out)` as the single exit path for stdout
- Log failures with `slog.Warn("your_event: ...", "err", err)` — never crash

**Verify:** `go build ./internal/hook/` compiles.

### Step 3 — Wire Into Dispatcher

Edit `internal/hook/dispatcher.go`, add a case to the switch:

```go
switch event {
case EventSessionStart:
    return handleSessionStart(ctx, raw, hs, stdout)
case EventPreCompact:
    return handlePreCompact(ctx, raw, hs, stdout)
case EventPreToolUse:
    return handlePreToolUse(ctx, raw, hs, stdout)
case EventPostToolUse:
    return handlePostToolUse(ctx, raw, hs, stdout)
case EventYourEvent:                              // <-- add
    return handleYourEvent(ctx, raw, hs, stdout)   // <-- add
default:
    return fmt.Errorf("unhandled hook event: %s", event)
}
```

**Verify:** `go vet ./internal/hook/` passes. The default case is unreachable for your event.

### Step 4 — Write Tests

Create `internal/hook/your_event_test.go`:

```go
package hook

import (
    "bytes"
    "context"
    "encoding/json"
    "testing"
)

func makeYourEventInput(cwd string) []byte {
    in := YourEventInput{
        HookInputBase: HookInputBase{
            SessionID:     "test-session",
            CWD:           cwd,
            HookEventName: EventYourEvent,
        },
        CustomField: "value",
    }
    b, _ := json.Marshal(in)
    return b
}

func TestHandleYourEvent(t *testing.T) {
    dir := t.TempDir()
    raw := makeYourEventInput(dir)
    var buf bytes.Buffer
    hs := &hookState{}

    err := handleYourEvent(context.Background(), raw, hs, &buf)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    var out HookOutput
    if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
        t.Fatalf("stdout not valid JSON: %v", err)
    }
}

func TestHandleYourEvent_NoBeads(t *testing.T) {
    // Verify degraded mode: no .beads/ dir → still emits valid JSON, no error
    dir := t.TempDir()
    raw := makeYourEventInput(dir)
    var buf bytes.Buffer
    hs := &hookState{}

    err := handleYourEvent(context.Background(), raw, hs, &buf)
    if err != nil {
        t.Fatalf("should not error without .beads/: %v", err)
    }
    if buf.Len() == 0 {
        t.Fatal("expected JSON output even without .beads/")
    }
}
```

Test categories to cover:
- **Happy path** with `.beads/` directory present
- **Degraded mode** without `.beads/` — must still return valid JSON
- **Stdout purity** — output is always parseable `HookOutput` JSON
- **Latency** — handler completes within budget (use `time.Now()` checks)
- For graph interactions, use `buildFakeBd()` from `hook_state_test.go` and `FAKE_BD_CAPTURE_FILE`

**Verify:** `go test ./internal/hook/ -run YourEvent -v` passes all tests.

### Step 5 — Add Dispatcher Routing Test

Edit `internal/hook/dispatcher_test.go`, add a test case to the routing table:

```go
{name: "routes YourEvent", event: EventYourEvent, input: makeYourEventInput(dir)},
```

**Verify:** `go test ./internal/hook/ -run TestDispatch -v` passes.

## Examples

**User says:** "Add a PostCompact hook that records compaction metadata"

**Actions taken:**
1. Add `EventPostCompact = "PostCompact"` to `events.go`, append to `ValidEvents`
2. Add `PostCompactInput` struct with `HookInputBase` + `CompactedTokens int`
3. Create `internal/hook/post_compact.go` with `handlePostCompact()` that writes metadata to `.gsdw/compact-log.jsonl`
4. Add `case EventPostCompact:` to dispatcher switch
5. Create `internal/hook/post_compact_test.go` with happy path, no-beads, and stdout purity tests
6. Add routing case to `dispatcher_test.go`
7. Run `go test ./internal/hook/ -v` — all pass

**Result:** New hook handler following identical patterns to existing `handlePostToolUse()`, with degraded mode, valid JSON output, and full test coverage.

## Common Issues

- **`unhandled hook event: YourEvent`** — You added the constant but forgot to add the `case` in `dispatcher.go`. Check the switch statement.
- **`hook_event_name mismatch`** — The `HookEventName` in your test input doesn't match the event string. Use the constant: `HookEventName: EventYourEvent`.
- **Test fails with `stdout not valid JSON`** — Your handler wrote to stdout outside of `writeOutput()`, or logged to stdout instead of stderr. Use `slog` (writes to stderr) for logging.
- **`undefined: YourEventInput`** — You defined the struct in `events.go` but the test file can't see it. Both are in package `hook` — check for typos or build errors: `go build ./internal/hook/`.
- **Fake bd binary not found in tests** — Use `buildFakeBd(t)` from `hook_state_test.go`. It compiles `internal/graph/testdata/fake_bd/`. Ensure that directory exists and the Go file inside builds.