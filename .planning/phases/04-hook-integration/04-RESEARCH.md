# Phase 4: Hook Integration - Research

**Researched:** 2026-03-21
**Domain:** Claude Code hook protocol, Go concurrency, async state persistence
**Confidence:** HIGH

## Summary

Phase 4 wires real logic into the four stub hook handlers (SessionStart, PreCompact, PreToolUse, PostToolUse). The existing dispatcher skeleton already parses JSON stdin, validates events, and writes `{}` to stdout. Phase 4 replaces the no-op handlers with handlers that load project state from the beads graph (SessionStart), persist in-progress work before compaction (PreCompact), inject bead context before tool calls (PreToolUse), and update bead state after tool calls (PostToolUse).

The Claude Code hook protocol is fully documented and verified. Key finding: the existing `HookInput` struct in `internal/hook/events.go` is missing several fields that Claude Code actually sends (notably `source` for SessionStart, `trigger` for PreCompact, `tool_name`/`tool_input`/`tool_use_id`/`permission_mode` for Pre/PostToolUse). These structs must be extended before implementing handlers. The `HookOutput` struct also needs expansion to support `additionalContext` injection (the primary mechanism for loading context into Claude).

PreCompact's constraint is confirmed: it cannot block compaction — it is purely informational. The two-stage save pattern (fast local write + async Dolt sync) is the correct approach, but the implementation detail is that PreCompact runs synchronously before compaction begins and has no special latency budget from the protocol side. The 200ms fast-path budget is self-imposed for UX.

**Primary recommendation:** Extend HookInput/HookOutput structs first (Task 0), then implement handlers in dependency order: SessionStart (highest value, most complex), PreToolUse/PostToolUse (high frequency, must be fast), PreCompact (crash recovery, async pattern).

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** gsdw is the developer interface. All hook behavior is invisible infrastructure.
- **D-02:** Developer never sees hook internals, PreCompact saves, or tool-use context injection.
- **D-03:** SessionStart loads: project name, current phase, ready tasks with objectives, recent decisions, blockers, last session summary.
- **D-04:** SessionStart outputs both structured JSON for machine consumption and human-readable markdown for context injection.
- **D-05:** gsdw handles uninitialized state (no .beads/) — Claude's discretion on approach.
- **D-06:** gsdw handles slow Dolt gracefully — Claude's discretion (cache fallback, partial results, timeout handling).
- **D-07:** PreCompact cannot block compaction — observability only. Two-stage: fast local write, then async Dolt sync.
- **D-08:** What to save in PreCompact, where to buffer, when to sync — all at Claude's discretion. Optimize for performance + reliability.
- **D-09:** Developer never sees PreCompact behavior. Crash-recovery infrastructure.
- **D-10:** Pre/PostToolUse scope, filtering, injection level, auto-detection — all at Claude's discretion.
- **D-11:** gsdw handles latency budget (<500ms) gracefully — Claude's discretion on degradation strategy.

### Claude's Discretion
- SessionStart: what to show when uninitialized, cache vs live query strategy, output structure
- PreCompact: state snapshot contents, local buffer location, async sync timing, skip-if-unchanged logic
- PreToolUse: which tools trigger injection, what context to inject, relevance filtering
- PostToolUse: which tools trigger updates, auto-detection of related beads, progress tracking granularity
- All hooks: timeout/degradation behavior within latency budgets
- Hook handler architecture: shared state across hooks vs independent, connection reuse

### Deferred Ideas (OUT OF SCOPE)
- Token-aware context injection (deciding HOW MUCH to inject based on budget) — Phase 9
- File-aware PreToolUse (inject context specific to the file being edited) — v2 (TOKEN-A01)
- Slash command integration with hooks — Phase 5+
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| INFRA-05 | SessionStart hook loads active project state from beads graph into context | `additionalContext` field in HookOutput is the injection mechanism; must extend HookInput for `source` field |
| INFRA-06 | PreCompact hook saves in-progress work state to beads (two-stage: fast local write, async Dolt commit) | PreCompact is non-blocking; runs synchronously before compaction; two-stage save with goroutine for Dolt sync |
| INFRA-07 | PreToolUse hook injects relevant bead context before tool execution | `tool_name`/`tool_input` available in input; `additionalContext` in hookSpecificOutput for injection; `updatedInput` can modify tool parameters |
| INFRA-08 | PostToolUse hook updates bead state after tool execution (progress, status changes) | `tool_response` available; `additionalContext` for feedback to Claude; `tool_response` is tool-specific |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| encoding/json (stdlib) | Go 1.26.1 | Hook stdin/stdout JSON encode/decode | Already used in dispatcher; zero deps |
| sync (stdlib) | Go 1.26.1 | Mutex for shared hookState; Once for lazy init | Already established pattern in mcp/init.go |
| context (stdlib) | Go 1.26.1 | Timeout propagation to graph.Client calls | Established throughout graph package |
| os (stdlib) | Go 1.26.1 | File I/O for PreCompact local buffer | Already used in graph/index.go |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| time (stdlib) | Go 1.26.1 | Timeout budgets, async sync delay | PreToolUse <500ms budget enforcement |
| path/filepath (stdlib) | Go 1.26.1 | .gsdw/ buffer path construction | PreCompact fast-path local write |

**No new external dependencies needed for Phase 4.** The graph.Client already handles all bd operations. The MCP SDK is not used by hooks (hooks are a separate binary invocation, not long-lived).

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Subprocess graph.Client per hook invocation | Shared persistent process | Hook is short-lived; subprocess overhead is 5-20ms per bd call, acceptable for SessionStart but borderline for PreToolUse |
| JSON additionalContext | Plain stdout text | JSON is more reliable and explicit; plain text stdout also works for SessionStart per docs but JSON is preferred |

**Installation:** No new packages needed. `go mod tidy` only.

## Architecture Patterns

### Recommended Project Structure
```
internal/hook/
├── dispatcher.go       # Existing: routes events to handlers; extend for real logic
├── events.go           # Existing: extend HookInput, new PreToolUseInput/PostToolUseInput structs
├── session_start.go    # New: SessionStart handler
├── pre_compact.go      # New: PreCompact handler + async Dolt sync
├── pre_tool_use.go     # New: PreToolUse handler + context injection
├── post_tool_use.go    # New: PostToolUse handler + bead state updates
├── hook_state.go       # New: shared hookState (lazy graph.Client init, .gsdw buffer path)
├── session_start_test.go
├── pre_compact_test.go
├── pre_tool_use_test.go
└── post_tool_use_test.go
```

### Pattern 1: Extended HookInput Structs per Event

The current `HookInput` struct is a minimal common base. Each hook event sends different fields. The correct pattern is a base struct plus event-specific extension:

```go
// Source: https://code.claude.com/docs/en/hooks (verified 2026-03-21)

// HookInputBase contains fields common to all hook events.
type HookInputBase struct {
    SessionID      string `json:"session_id"`
    TranscriptPath string `json:"transcript_path"`
    CWD            string `json:"cwd"`
    HookEventName  string `json:"hook_event_name"`
}

// SessionStartInput is the JSON payload for SessionStart events.
type SessionStartInput struct {
    HookInputBase
    Source    string `json:"source"`    // "startup" | "resume" | "clear" | "compact"
    Model     string `json:"model"`     // e.g., "claude-sonnet-4-6"
    AgentType string `json:"agent_type,omitempty"` // only if --agent flag used
}

// PreCompactInput is the JSON payload for PreCompact events.
type PreCompactInput struct {
    HookInputBase
    Trigger            string `json:"trigger"`            // "manual" | "auto"
    CustomInstructions string `json:"custom_instructions"` // empty for auto
}

// PreToolUseInput is the JSON payload for PreToolUse events.
type PreToolUseInput struct {
    HookInputBase
    PermissionMode string          `json:"permission_mode"` // "default"|"plan"|"acceptEdits"|"dontAsk"|"bypassPermissions"
    ToolName       string          `json:"tool_name"`
    ToolInput      json.RawMessage `json:"tool_input"` // tool-specific, decode separately
    ToolUseID      string          `json:"tool_use_id"`
}

// PostToolUseInput is the JSON payload for PostToolUse events.
type PostToolUseInput struct {
    HookInputBase
    PermissionMode string          `json:"permission_mode"`
    ToolName       string          `json:"tool_name"`
    ToolInput      json.RawMessage `json:"tool_input"`
    ToolResponse   json.RawMessage `json:"tool_response"` // tool-specific
    ToolUseID      string          `json:"tool_use_id"`
}
```

Use `json.RawMessage` for `tool_input` and `tool_response` because the shape is tool-specific. Decode into tool-specific structs only when the handler needs the content.

### Pattern 2: HookOutput with additionalContext

`additionalContext` is the injection mechanism. For SessionStart, it populates Claude's context before the session begins:

```go
// Source: https://code.claude.com/docs/en/hooks (verified 2026-03-21)

// HookOutput is the JSON response written to stdout by any hook handler.
type HookOutput struct {
    Continue         *bool          `json:"continue,omitempty"`
    StopReason       string         `json:"stopReason,omitempty"`
    SuppressOutput   bool           `json:"suppressOutput,omitempty"`
    SystemMessage    string         `json:"systemMessage,omitempty"`
    AdditionalContext string        `json:"additionalContext,omitempty"` // injected into Claude's context
    HookSpecificOutput any          `json:"hookSpecificOutput,omitempty"` // for PreToolUse/PostToolUse
}

// PreToolUseHookOutput is the hookSpecificOutput payload for PreToolUse.
type PreToolUseHookOutput struct {
    HookEventName           string          `json:"hookEventName"` // must be "PreToolUse"
    PermissionDecision      string          `json:"permissionDecision,omitempty"` // "allow"|"deny"|"ask"
    PermissionDecisionReason string         `json:"permissionDecisionReason,omitempty"`
    UpdatedInput            json.RawMessage `json:"updatedInput,omitempty"` // modified tool parameters
    AdditionalContext       string          `json:"additionalContext,omitempty"`
}

// PostToolUseHookOutput is the hookSpecificOutput payload for PostToolUse.
type PostToolUseHookOutput struct {
    HookEventName     string `json:"hookEventName"` // must be "PostToolUse"
    AdditionalContext string `json:"additionalContext,omitempty"`
    // UpdatedMCPToolOutput omitted — only for MCP tools, not built-in tools
}
```

### Pattern 3: hookState — Shared Lazy Init Across Hook Invocations

Hooks are short-lived subprocess invocations. Each invocation starts fresh. Do NOT attempt to share state across invocations via memory. The `hookState` type mirrors `serverState` in `internal/mcp/init.go` but for hook context:

```go
// internal/hook/hook_state.go

// hookState initializes a read-only graph.Client (no batch mode needed for hooks).
// Hooks do reads (SessionStart queries beads) and writes (PostToolUse updates beads).
// Writes go through runWrite() which is already part of graph.Client.
type hookState struct {
    once     sync.Once
    client   *graph.Client
    err      error
    beadsDir string // from HookInputBase.CWD or os.Getwd()
    bdPath   string // optional override for testing
}

func (h *hookState) init(ctx context.Context) error {
    h.once.Do(func() {
        dir := h.beadsDir
        if dir == "" {
            var err error
            dir, err = os.Getwd()
            if err != nil {
                h.err = fmt.Errorf("hookState: getwd: %w", err)
                return
            }
        }
        // Use non-batch mode for hooks: each hook invocation is its own
        // transaction boundary, not part of a wave batch.
        if h.bdPath != "" {
            h.client = graph.NewClientWithPath(h.bdPath, dir)
        } else {
            c, err := graph.NewClient(dir)
            if err != nil {
                h.err = err
                return
            }
            h.client = c
        }
    })
    return h.err
}
```

**Key difference from mcp/init.go:** Hooks use non-batch graph.Client. The MCP server batches writes at wave boundaries. Hooks are individual operations that should commit immediately.

### Pattern 4: SessionStart Handler

```go
// internal/hook/session_start.go
// Source: verified against https://code.claude.com/docs/en/hooks

func handleSessionStart(ctx context.Context, input SessionStartInput, hs *hookState, w io.Writer) error {
    // Fast path: if .beads/ does not exist, emit hint and return immediately.
    beadsPath := filepath.Join(input.CWD, ".beads")
    if _, err := os.Stat(beadsPath); os.IsNotExist(err) {
        return writeOutput(w, HookOutput{
            AdditionalContext: "gsd-wired: No .beads/ directory found. Run /gsd-wired:init to initialize.",
        })
    }

    // Initialize graph client (uses CWD from hook input).
    hs.beadsDir = input.CWD
    if err := hs.init(ctx); err != nil {
        // Degraded mode: log to stderr, emit partial context.
        slog.Warn("hookState init failed, emitting empty context", "err", err)
        return writeOutput(w, HookOutput{})
    }

    // Query beads for session context (with timeout).
    queryCtx, cancel := context.WithTimeout(ctx, 1500*time.Millisecond)
    defer cancel()

    // Load ready tasks, current phase, recent decisions.
    // Build human-readable markdown for additionalContext.
    context := buildSessionContext(queryCtx, hs.client)

    return writeOutput(w, HookOutput{AdditionalContext: context})
}
```

### Pattern 5: PreCompact Two-Stage Save

```go
// internal/hook/pre_compact.go
// Source: verified against https://code.claude.com/docs/en/hooks

func handlePreCompact(ctx context.Context, input PreCompactInput, hs *hookState, w io.Writer) error {
    // Stage 1: Fast local snapshot write (must complete before compaction begins).
    // PreCompact is synchronous — compaction waits for this handler to exit.
    snapshot := captureSessionSnapshot(ctx, input)
    bufferPath := filepath.Join(input.CWD, ".gsdw", "precompact-snapshot.json")
    if err := writeSnapshotAtomic(bufferPath, snapshot); err != nil {
        slog.Error("precompact: local write failed", "err", err)
        // Do not fail — compaction must proceed.
    }

    // Stage 2: Async Dolt sync (fire and forget — hook exits, goroutine continues).
    // IMPORTANT: The goroutine must NOT use the hook's context (already cancelled at exit).
    go syncSnapshotToDolt(context.Background(), input.CWD, bufferPath)

    // PreCompact cannot inject context or block — return minimal output immediately.
    return writeOutput(w, HookOutput{})
}
```

**Critical insight:** The goroutine for Dolt sync must use `context.Background()`, not the hook's context parameter. When the hook binary exits, the context is cancelled. The goroutine needs its own context.

**Also critical:** The goroutine will run in a process that is about to exit. On Go, goroutines are terminated when main() exits. The async sync must either:
1. Complete before main() exits (requires a WaitGroup/channel in main), OR
2. Write to the local buffer only and rely on a later hook invocation (PostToolUse or next SessionStart) to sync to Dolt.

Option 2 is simpler and more reliable. The local buffer IS the fast path. Dolt sync happens during PostToolUse or next SessionStart when the binary is fully alive.

### Pattern 6: PreToolUse Injection

```go
// internal/hook/pre_tool_use.go

// Matchers: only tools that write state should trigger context injection.
// Read-only tools (Read, Glob, Grep, WebFetch, WebSearch) do not need bead context.
var contextInjectTools = map[string]bool{
    "Write": true,
    "Edit":  true,
    "Bash":  true,
    "Agent": true,
}

func handlePreToolUse(ctx context.Context, input PreToolUseInput, hs *hookState, w io.Writer) error {
    if !contextInjectTools[input.ToolName] {
        // Fast path: tool doesn't need context injection. Return allow immediately.
        return writeOutput(w, preToolUseAllow(""))
    }

    // Inject relevant bead context with strict timeout (must stay <500ms total).
    queryCtx, cancel := context.WithTimeout(ctx, 400*time.Millisecond)
    defer cancel()

    context := buildToolContext(queryCtx, hs.client, input.ToolName)

    return writeOutput(w, preToolUseAllow(context))
}

func preToolUseAllow(additionalContext string) HookOutput {
    return HookOutput{
        HookSpecificOutput: PreToolUseHookOutput{
            HookEventName:      "PreToolUse",
            PermissionDecision: "allow",
            AdditionalContext:  additionalContext,
        },
    }
}
```

### Pattern 7: PostToolUse State Updates

```go
// internal/hook/post_tool_use.go

// Only state-mutating tools trigger bead updates.
var beadUpdateTools = map[string]bool{
    "Write": true,
    "Edit":  true,
    "Bash":  true,
}

func handlePostToolUse(ctx context.Context, input PostToolUseInput, hs *hookState, w io.Writer) error {
    if !beadUpdateTools[input.ToolName] {
        return writeOutput(w, HookOutput{})
    }

    // Best-effort bead update — if it fails, log to stderr but don't block Claude.
    updateCtx, cancel := context.WithTimeout(ctx, 400*time.Millisecond)
    defer cancel()

    if err := updateBeadProgress(updateCtx, hs.client, input); err != nil {
        slog.Warn("postToolUse: bead update failed", "tool", input.ToolName, "err", err)
    }

    return writeOutput(w, HookOutput{})
}
```

### Anti-Patterns to Avoid

- **Anti-pattern: Writing to stdout before JSON.** Any non-JSON byte on stdout breaks Claude Code's hook response parsing. The first byte written must be `{`. Never use `fmt.Println` in hook handlers.
- **Anti-pattern: Blocking on slow Dolt in PreToolUse/PostToolUse.** These fire on every tool call. If Dolt is cold, a 3s init blocks the entire agentic loop. Use the hookState init timeout; emit empty response on timeout rather than blocking.
- **Anti-pattern: Goroutine in main process for async work.** Go process exits when main() returns. Goroutines are killed. For PreCompact async work, use the local file buffer as the reliable path; do not rely on goroutine completion.
- **Anti-pattern: Returning `continue: false`.** The docs show this as an option, but for gsdw's invisible-infrastructure model (D-01/D-02), hooks should never stop Claude. Only return `continue: false` for unrecoverable internal gsdw errors, and only in SessionStart.
- **Anti-pattern: Using the shared `HookInput` struct for all events.** Events have different fields. Decode into event-specific structs so handlers have typed access to `tool_name`, `trigger`, `source`, etc.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Atomic file writes | Custom write-then-rename | `os.WriteFile` to tmp + `os.Rename` | Already proven in graph/index.go — same OS-level atomicity |
| Timeout enforcement | Manual timer goroutines | `context.WithTimeout` | Already used throughout graph package; cancel is automatic |
| JSON schema validation | Custom validator | Decode into typed structs, check required fields | json.Unmarshal already rejects invalid types |
| Graph state queries | New bd CLI calls | `graph.Client.ListReady`, `GetBead`, `QueryByLabel` | Already implemented and tested in Phase 2/3 |
| Local index lookups | Fresh bd queries every hook | `graph.LoadIndex` from .gsdw/index.json | Already implemented — fast, no subprocess overhead |

**Key insight:** The graph package (Phase 2/3) already has all query and update primitives needed by hooks. Phase 4 is about wiring them into hook handlers, not building new persistence logic.

## Common Pitfalls

### Pitfall 1: Stdout Pollution in Hook Handlers
**What goes wrong:** Any byte written to stdout that is not valid JSON causes Claude Code to fail to parse the hook response. The hook is silently ignored or causes an error visible to the user.
**Why it happens:** Forgetting that hooks run as subprocesses. Go's `fmt.Print`, stdlib log output, or error messages printed to stdout all corrupt the protocol.
**How to avoid:** All slog handlers must write to `os.Stderr` (established in Phase 1). Use the `writeOutput` helper that wraps `json.NewEncoder(stdout).Encode()` for all responses.
**Warning signs:** Hook exits 0 but Claude Code doesn't inject context. Test with `echo '...' | gsdw hook SessionStart` and verify stdout is valid JSON.

### Pitfall 2: Goroutine Exit Race in PreCompact
**What goes wrong:** PreCompact spawns a goroutine to sync to Dolt. The hook binary exits immediately after. The goroutine is killed mid-write, leaving a corrupt or incomplete Dolt commit.
**Why it happens:** Go runtime terminates all goroutines when `os.Exit()` or `main()` returns. Background goroutines do not complete.
**How to avoid:** Do not rely on goroutines for the Dolt write in PreCompact. Write only to the local buffer (`.gsdw/precompact-snapshot.json`) in the PreCompact handler. Let the next hook invocation (SessionStart or PostToolUse) detect and sync the pending snapshot.
**Warning signs:** Dolt state doesn't reflect saved snapshots after compaction events.

### Pitfall 3: hookState Init on Every Tool Call
**What goes wrong:** Each PreToolUse/PostToolUse invocation starts a fresh process and calls `bd` (via LookPath + subprocess) during hookState init. For a typical session with 50 tool calls, this means 50 bd process starts, 50 PATH lookups, etc.
**Why it happens:** Hooks are short-lived subprocesses. There's no shared memory between invocations.
**How to avoid:** Use the local index (`.gsdw/index.json`) as the fast path. For Pre/PostToolUse, read the index (file read, <1ms) instead of querying bd (subprocess, 20-100ms). Reserve bd queries for SessionStart where latency budget (2s) is generous.
**Warning signs:** Pre/PostToolUse latency exceeds 200ms consistently. Profile with `time echo '...' | gsdw hook PreToolUse`.

### Pitfall 4: Missing tool_input Fields in PreToolUse
**What goes wrong:** Handler tries to decode `tool_input` as a fixed struct, but different tools have different shapes. Write sends `file_path`+`content`, Bash sends `command`, etc.
**Why it happens:** Claude Code sends tool-specific JSON in the `tool_input` field.
**How to avoid:** Use `json.RawMessage` for `tool_input` and `tool_response`. Decode only when the handler specifically needs tool-specific data. For bead context injection, the tool NAME is sufficient to decide what to inject — the tool input body is only needed for PostToolUse file path extraction.
**Warning signs:** json.Unmarshal error on PreToolUse input when tool is not Write/Edit.

### Pitfall 5: PreCompact `trigger` Field Not Checked
**What goes wrong:** PreCompact fires for both `manual` (user ran `/compact`) and `auto` (automatic compaction). Treating them identically can cause over-saving.
**Why it happens:** The trigger field is present but easy to overlook.
**How to avoid:** For v1, treat both triggers identically (D-08: what to save is Claude's discretion). For optimization, auto-compaction saves are typically less urgent than manual.
**Warning signs:** No behavioral difference needed for v1.

### Pitfall 6: hooks.json Already Registered Without Matchers
**What goes wrong:** The existing `hooks/hooks.json` registers all four events without `matcher` fields. This means PreToolUse and PostToolUse fire for EVERY tool call. The hook binary has to do its own filtering.
**Why it happens:** The plugin.json / hooks.json format supports matcher-level filtering, but it was not set up in Phase 1.
**How to avoid:** Either (a) add `matcher` filters to hooks.json to restrict which tool names trigger Pre/PostToolUse, or (b) do filtering inside the Go handler with early-return for non-matching tools. Option (b) is simpler for v1 since hooks.json changes require plugin reinstall.
**Warning signs:** Every tool call spawns gsdw hook PostToolUse even for Read operations — visible in stderr logs or via time measurement.

## Code Examples

Verified patterns from official sources:

### SessionStart: additionalContext Injection
```go
// Source: https://code.claude.com/docs/en/hooks (verified 2026-03-21)
// Exit 0, stdout is valid JSON with additionalContext field.
// Claude sees this text before responding to the first user message.
output := HookOutput{
    AdditionalContext: "## GSD Project State\n\n**Phase 4:** Hook Integration\n**Ready tasks:** 3\n...",
}
json.NewEncoder(stdout).Encode(output)
```

### PreToolUse: Allow with Context
```go
// Source: https://code.claude.com/docs/en/hooks (verified 2026-03-21)
// hookSpecificOutput.permissionDecision = "allow" skips permission prompt.
output := HookOutput{
    HookSpecificOutput: PreToolUseHookOutput{
        HookEventName:      "PreToolUse",
        PermissionDecision: "allow",
        AdditionalContext:  "Current phase task: implement session_start handler",
    },
}
json.NewEncoder(stdout).Encode(output)
```

### PreCompact: Fast Local Buffer Write
```go
// Source: pattern derived from graph/index.go atomic write (verified)
// Must complete in <200ms to not feel sluggish before compaction.
snapshot, _ := json.MarshalIndent(state, "", "  ")
tmp := bufferPath + ".tmp"
os.WriteFile(tmp, snapshot, 0644)
os.Rename(tmp, bufferPath) // atomic on same filesystem
```

### Testing Hook Handler with Injected I/O
```go
// Source: established pattern from dispatcher_test.go (Phase 1)
// No process spawn needed — inject bytes.Buffer as stdin/stdout.
func TestSessionStartEmitsContext(t *testing.T) {
    input := SessionStartInput{
        HookInputBase: HookInputBase{
            SessionID:     "test-123",
            CWD:           t.TempDir(),
            HookEventName: "SessionStart",
        },
        Source: "startup",
    }
    data, _ := json.Marshal(input)
    stdin := bytes.NewReader(data)
    var stdout bytes.Buffer

    hs := &hookState{bdPath: fakeBdPath, beadsDir: input.CWD}
    err := handleSessionStart(context.Background(), input, hs, &stdout)
    require.NoError(t, err)

    var out HookOutput
    json.Unmarshal(bytes.TrimSpace(stdout.Bytes()), &out)
    assert.NotEmpty(t, out.AdditionalContext)
}
```

### Go Profiling for Latency Measurement
```go
// Source: Go stdlib time package
// For hook latency measurement in tests and manual verification:
start := time.Now()
// ... hook logic ...
elapsed := time.Since(start)
slog.Debug("hook latency", "event", "PreToolUse", "ms", elapsed.Milliseconds())
```

```bash
# Manual latency test via CLI:
time echo '{"session_id":"t","transcript_path":"/tmp/t.jsonl","cwd":"/tmp","hook_event_name":"PreToolUse","tool_name":"Read","tool_input":{"file_path":"/tmp/t"},"tool_use_id":"tu-1"}' | ./gsdw hook PreToolUse
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Exit code 2 for blocking in PreToolUse | `permissionDecision: "deny"` in JSON | Recent Claude Code versions | Exit 2 still works but JSON is preferred and more explicit |
| Plain stdout text for SessionStart context | `additionalContext` JSON field | Documented in current hooks spec | Both work; JSON is more structured and avoids encoding edge cases |
| Blocking hooks for state saves | Non-blocking with async pattern | Design principle from start | PreCompact never blocks; saves are best-effort fast path |

**Deprecated/outdated:**
- Exit code 2 as primary blocking mechanism: still supported but `permissionDecision: "deny"` in JSON is the current approach for PreToolUse
- The current `HookInput` struct (Phase 1 stub) is missing `source`, `trigger`, `tool_name`, `tool_input`, `permission_mode` — these are real protocol fields that must be added

## Validation Architecture

> nyquist_validation is true — section included.

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib) |
| Config file | none — `go test ./...` from repo root |
| Quick run command | `go test ./internal/hook/... -race -count=1` |
| Full suite command | `go test ./... -race -count=1` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| INFRA-05 | SessionStart emits additionalContext with project state | unit | `go test ./internal/hook/... -run TestSessionStart -race` | ❌ Wave 0 |
| INFRA-05 | SessionStart degrades gracefully when .beads/ missing | unit | `go test ./internal/hook/... -run TestSessionStartNoBEads -race` | ❌ Wave 0 |
| INFRA-05 | SessionStart respects 1.5s total latency budget | unit | `go test ./internal/hook/... -run TestSessionStartLatency -race` | ❌ Wave 0 |
| INFRA-06 | PreCompact writes local buffer atomically | unit | `go test ./internal/hook/... -run TestPreCompactLocalWrite -race` | ❌ Wave 0 |
| INFRA-06 | PreCompact returns empty output (cannot block) | unit | `go test ./internal/hook/... -run TestPreCompactOutput -race` | ❌ Wave 0 |
| INFRA-07 | PreToolUse allows write-class tools with context | unit | `go test ./internal/hook/... -run TestPreToolUseWrite -race` | ❌ Wave 0 |
| INFRA-07 | PreToolUse allows read-class tools with no injection | unit | `go test ./internal/hook/... -run TestPreToolUseRead -race` | ❌ Wave 0 |
| INFRA-07 | PreToolUse completes in <500ms on cold start | unit | `go test ./internal/hook/... -run TestPreToolUseLatency -race` | ❌ Wave 0 |
| INFRA-08 | PostToolUse triggers bead update for write tools | unit | `go test ./internal/hook/... -run TestPostToolUseWrite -race` | ❌ Wave 0 |
| INFRA-08 | PostToolUse is no-op for read tools | unit | `go test ./internal/hook/... -run TestPostToolUseRead -race` | ❌ Wave 0 |
| INFRA-05-08 | All hook handlers produce valid JSON stdout | unit | `go test ./internal/hook/... -run TestStdoutPurity -race` | existing (dispatcher_test.go extends) |
| INFRA-05-08 | Dispatcher routes to correct handler per event | unit | `go test ./internal/hook/... -run TestDispatch -race` | existing (dispatcher_test.go) |

### Sampling Rate
- **Per task commit:** `go test ./internal/hook/... -race -count=1`
- **Per wave merge:** `go test ./... -race -count=1`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `internal/hook/session_start_test.go` — covers INFRA-05
- [ ] `internal/hook/pre_compact_test.go` — covers INFRA-06
- [ ] `internal/hook/pre_tool_use_test.go` — covers INFRA-07
- [ ] `internal/hook/post_tool_use_test.go` — covers INFRA-08
- [ ] `internal/hook/hook_state_test.go` — covers hookState init behavior (mirrors mcp/init_test.go pattern)

## Open Questions

1. **Should hookState be in a separate package or the hook package?**
   - What we know: mcp/init.go puts serverState in the mcp package. Same pattern makes sense.
   - What's unclear: If hooks ever need to call MCP tools or vice versa, shared state across packages becomes relevant.
   - Recommendation: Put hookState in the hook package for v1. Cross-package state sharing is a Phase 9 optimization concern.

2. **PreCompact snapshot: what exact state to capture?**
   - What we know: D-08 defers this to Claude's discretion.
   - What's unclear: Whether to snapshot all open beads, just the active task, or a configurable subset.
   - Recommendation: Snapshot the active session context: current phase bead ID, in-progress task bead IDs, and the last N characters of the transcript path reference. Keep it small (<4KB) for the local buffer.

3. **PostToolUse: how to detect which bead a file edit relates to?**
   - What we know: D-10 defers auto-detection to Claude's discretion. TOKEN-A01 (file-aware injection) is v2.
   - What's unclear: For v1, should PostToolUse attempt any bead correlation, or just mark "tool called" as a progress signal?
   - Recommendation: For v1, do not attempt file-to-bead correlation. PostToolUse records the tool call event to the session snapshot (local buffer only). Bead status updates happen via explicit MCP tool calls from Claude, not from automatic PostToolUse detection.

## Sources

### Primary (HIGH confidence)
- https://code.claude.com/docs/en/hooks — Complete hook event reference, all JSON schemas verified 2026-03-21
- `internal/hook/events.go` — Existing HookInput/HookOutput (stub level, needs extension)
- `internal/hook/dispatcher.go` — Existing Dispatch() function structure
- `internal/mcp/init.go` — serverState lazy-init pattern (directly reusable)
- `internal/graph/index.go` — Atomic file write pattern with temp+rename
- `internal/graph/query.go` — Available query operations for SessionStart context loading

### Secondary (MEDIUM confidence)
- `.planning/research/SUMMARY.md` — PreCompact non-blocking constraint, two-stage save pattern recommendation
- `internal/hook/dispatcher_test.go` — Existing test patterns (injected I/O, stdout purity checks)

### Tertiary (LOW confidence)
- None — all critical claims verified against official docs or existing code.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new external deps; all stdlib and existing graph package
- Hook protocol schemas: HIGH — verified against official Claude Code hooks docs 2026-03-21
- Architecture: HIGH — patterns derived directly from existing Phase 1-3 code and verified hook protocol
- Pitfalls: HIGH — stdout pollution, goroutine exit race, and hookState init cost all verified against Go runtime behavior and Claude Code protocol constraints
- PreCompact async pattern: HIGH — goroutine-cannot-survive-exit confirmed; local buffer as reliable path is the correct design

**Research date:** 2026-03-21
**Valid until:** 2026-04-21 (hook protocol is stable; Claude Code releases infrequently)
