# Phase 1: Binary Scaffold - Research

**Researched:** 2026-03-21
**Domain:** Go binary scaffold, Claude Code plugin system, MCP stdio server
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**CLI command tree**
- D-01: Binary name is `gsdw`
- D-02: Subcommands: `serve` (MCP stdio), `hook <event>` (dispatcher), `bd <args>` (passthrough to bd CLI with GSD context), `version`
- D-03: `hook <event>` parses and validates JSON from stdin even at stub stage — catches integration issues early
- D-04: Version output format: `0.1.0 (abc1234)` — semver + git commit hash

**Go module and project layout**
- D-05: Module path: `github.com/The-Artificer-of-Ciphers-LLC/gsd-wired`
- D-06: Standard Go layout: `cmd/gsdw/main.go` + `internal/` packages
- D-07: `.claude-plugin/` directory at repo root alongside `go.mod`
- D-08: Installable via `go install github.com/The-Artificer-of-Ciphers-LLC/gsd-wired/cmd/gsdw@latest`

**Plugin manifest**
- D-09: Manifest grows per phase — only register what the current phase delivers (no forward-declared stubs)
- D-10: Slash command prefix: `/gsd-wired:` (longer form for clarity, not `/gsdw:`)
- D-11: Display name: "GSD Wired" / Description: "Token-efficient development lifecycle on a versioned graph"
- D-12: No minimum Claude Code version pinned

**Logging**
- D-13: Dual format: JSON and human-readable, selectable by flag
- D-14: `--log-level` flag with levels: error (default), info, debug
- D-15: Serve mode is silent unless errors occur — no chatty connection/tool-call logs
- D-16: Use `log/slog` from stdlib — zero external dependencies, structured, leveled, built-in JSON + text handlers

### Claude's Discretion
- Cobra command wiring details and help text
- Internal package boundaries (how to split `internal/`)
- Hook event name constants and validation logic
- Build system for embedding version/commit at compile time

### Deferred Ideas (OUT OF SCOPE)
- MCP tool registration (Phase 3)
- Real hook logic with bead state persistence (Phase 4)
- Slash command implementations (Phase 5+)
- Token-aware context injection in hooks (Phase 9)
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| INFRA-01 | Single Go binary serves as MCP server (stdio), hook dispatcher (subcommand), and CLI tool | Cobra multi-mode pattern, MCP SDK stdio transport |
| INFRA-04 | Plugin manifest (.claude-plugin/plugin.json) registers MCP server, hooks, and slash commands | Verified plugin.json schema, .mcp.json format, hooks/hooks.json format |
| INFRA-09 | Strict stdout discipline — no stray output that could break MCP stdio protocol | slog stderr-only pattern, Cobra SilenceUsage, stderr redirect |
</phase_requirements>

## Summary

Phase 1 builds the skeleton that all later phases plug into. The binary runs in three modes selected by Cobra subcommand: `serve` (long-lived MCP stdio server), `hook <event>` (short-lived hook dispatcher reading JSON from stdin), and `bd <args>` (bd CLI passthrough). The plugin manifest in `.claude-plugin/plugin.json` plus a `.mcp.json` and `hooks/hooks.json` registers the binary with Claude Code.

The official MCP Go SDK (`github.com/modelcontextprotocol/go-sdk` v1.4.1, published 2026-03-13) is the correct library. It requires Go 1.25+, which matches the installed Go 1.26.1. The `server.Run(ctx, &mcp.StdioTransport{})` call blocks until the client disconnects, responding to `initialize` automatically. No tool registration is needed at this phase — a server with zero tools still satisfies the protocol.

The most critical discipline in Phase 1 is stdout purity: any non-JSON byte on stdout while `serve` is running corrupts the MCP framing. All logging MUST go to stderr. This means `log/slog` must be configured with `os.Stderr` as the output writer, Cobra must be configured with `SilenceUsage: true` and `SilenceErrors: true` on the root command, and the hook subcommand must only emit valid JSON on stdout (or nothing).

**Primary recommendation:** Build the Cobra command tree first, wire slog to stderr immediately, then add the MCP server stub, then the hook dispatcher with stdin JSON parsing, then the plugin manifest. Test stdout purity explicitly in CI.

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/modelcontextprotocol/go-sdk` | v1.4.1 | MCP stdio server | Official SDK, Google co-maintained, auto JSON schema from Go structs, spec 2025-11-25 |
| `github.com/spf13/cobra` | v1.10.2 | CLI framework | Industry standard, persistent flags, subcommand isolation, help generation |
| `log/slog` | stdlib (Go 1.21+) | Structured logging | Zero deps, built-in JSON + text handlers, leveled, writes to any `io.Writer` |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `runtime/debug` | stdlib | Build info (VCS metadata) | Read `debug.ReadBuildInfo()` for git hash at runtime — no ldflags needed |
| `encoding/json` | stdlib | Hook stdin/stdout JSON | Decode hook event JSON from stdin, encode response JSON to stdout |
| `os/exec` | stdlib | bd CLI passthrough | Shell out to `bd` for graph operations (Phase 2+) |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `modelcontextprotocol/go-sdk` | `mark3labs/mcp-go` v0.45.0 | mark3labs is unofficial (v0.x), no Google co-maintenance; official SDK is authoritative |
| `log/slog` | `zerolog`, `zap` | External deps — locked decision is stdlib |
| `runtime/debug` for version | `-ldflags -X main.version=...` | ldflags requires build-time scripting; `debug.ReadBuildInfo()` works with `go install` without Makefile magic |

**Installation:**
```bash
go get github.com/modelcontextprotocol/go-sdk@v1.4.1
go get github.com/spf13/cobra@v1.10.2
```

**Version verification (confirmed 2026-03-21):**
- `github.com/modelcontextprotocol/go-sdk` → v1.4.1 (2026-03-13)
- `github.com/spf13/cobra` → v1.10.2 (2025-12-03)
- `github.com/modelcontextprotocol/go-sdk` requires Go 1.25.0 — satisfied by system Go 1.26.1

## Architecture Patterns

### Recommended Project Structure
```
cmd/gsdw/
├── main.go              # Entry: Cobra root, persistent flags, version
internal/
├── hook/
│   ├── dispatcher.go    # Reads JSON from stdin, routes to handler
│   └── events.go        # Hook event type constants and structs
├── mcp/
│   └── server.go        # MCP server constructor, stub serve loop
└── version/
    └── version.go       # Build info extraction via runtime/debug
.claude-plugin/
└── plugin.json          # Plugin manifest (name, description, version)
.mcp.json                # MCP server registration (points to gsdw serve)
hooks/
└── hooks.json           # Hook event registrations (points to gsdw hook)
go.mod
go.sum
```

### Pattern 1: Cobra Multi-Mode Binary with Stdout Discipline
**What:** Root command carries persistent logging flags. Each subcommand is a separate execution mode. Error and usage output explicitly silenced.
**When to use:** Any binary that serves double duty as a long-lived server and a short-lived CLI.

```go
// Source: Cobra docs + stdout discipline requirement
package main

import (
    "log/slog"
    "os"

    "github.com/spf13/cobra"
)

var (
    logLevel  string
    logFormat string
    logger    *slog.Logger
)

func main() {
    root := &cobra.Command{
        Use:          "gsdw",
        Short:        "GSD Wired — token-efficient development lifecycle",
        SilenceUsage: true,   // CRITICAL: prevents usage dump to stdout on error
        SilenceErrors: true,  // CRITICAL: prevents error text to stdout on error
        PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
            return initLogger(logLevel, logFormat)
        },
    }
    root.PersistentFlags().StringVar(&logLevel, "log-level", "error", "Log level (error|info|debug)")
    root.PersistentFlags().StringVar(&logFormat, "log-format", "text", "Log format (text|json)")

    root.AddCommand(serveCmd())
    root.AddCommand(hookCmd())
    root.AddCommand(bdCmd())
    root.AddCommand(versionCmd())

    if err := root.Execute(); err != nil {
        // Write errors to stderr, never stdout
        slog.Error("command failed", "err", err)
        os.Exit(1)
    }
}
```

### Pattern 2: slog Dual Handler (JSON + Text, stderr-only)
**What:** Selectable log format via flag. ALL output to stderr. JSON for machine consumption, text for human debugging.
**When to use:** Any subcommand — but critical for `serve` which shares stdout with MCP protocol frames.

```go
// Source: log/slog stdlib docs (Go 1.21+)
func initLogger(level, format string) error {
    var lvl slog.Level
    switch level {
    case "debug":
        lvl = slog.LevelDebug
    case "info":
        lvl = slog.LevelInfo
    default:
        lvl = slog.LevelError
    }

    opts := &slog.HandlerOptions{Level: lvl}
    var handler slog.Handler

    if format == "json" {
        handler = slog.NewJSONHandler(os.Stderr, opts)  // ALWAYS os.Stderr
    } else {
        handler = slog.NewTextHandler(os.Stderr, opts)  // ALWAYS os.Stderr
    }

    slog.SetDefault(slog.New(handler))
    return nil
}
```

### Pattern 3: MCP Stdio Server Stub
**What:** Minimal MCP server that responds to `initialize` immediately with no tools registered. Phase 3 adds tools.
**When to use:** Phase 1 scaffold — proves the binary can speak MCP protocol before adding domain logic.

```go
// Source: github.com/modelcontextprotocol/go-sdk@v1.4.1/examples/server/hello/main.go
package mcp

import (
    "context"
    "log/slog"

    "github.com/modelcontextprotocol/go-sdk/mcp"
)

func Serve(ctx context.Context) error {
    server := mcp.NewServer(&mcp.Implementation{
        Name:    "gsd-wired",
        Version: version.String(), // from internal/version
    }, nil)
    // No tools registered at Phase 1 — added in Phase 3
    slog.Debug("mcp server starting on stdio")
    return server.Run(ctx, &mcp.StdioTransport{})
}
```

### Pattern 4: Hook Dispatcher with stdin JSON Parsing
**What:** Reads hook event JSON from stdin, validates structure, routes to handler stub, writes response JSON to stdout.
**When to use:** `gsdw hook <event>` subcommand — validates the wire protocol even before handlers do real work.

```go
// Source: Claude Code hooks docs (verified 2026-03-21)
type HookInput struct {
    SessionID       string          `json:"session_id"`
    TranscriptPath  string          `json:"transcript_path"`
    CWD             string          `json:"cwd"`
    HookEventName   string          `json:"hook_event_name"`
    // Event-specific fields populated via json.RawMessage or typed structs
}

type HookOutput struct {
    HookSpecificOutput *HookSpecificOutput `json:"hookSpecificOutput,omitempty"`
    Continue           *bool               `json:"continue,omitempty"`
    SuppressOutput     bool                `json:"suppressOutput,omitempty"`
}

type HookSpecificOutput struct {
    HookEventName     string `json:"hookEventName"`
    AdditionalContext string `json:"additionalContext,omitempty"`
}

func Dispatch(event string) error {
    var input HookInput
    if err := json.NewDecoder(os.Stdin).Decode(&input); err != nil {
        // Write error to stderr (stdout must remain clean for hook protocol)
        slog.Error("failed to decode hook input", "err", err)
        os.Exit(2) // exit 2 = show stderr to user, non-fatal
    }
    // Validate event name matches subcommand arg
    if input.HookEventName != event {
        slog.Error("hook event mismatch", "expected", event, "got", input.HookEventName)
        os.Exit(2)
    }
    // Phase 1: emit no-op response (empty JSON is valid for most hooks)
    return json.NewEncoder(os.Stdout).Encode(HookOutput{})
}
```

### Pattern 5: Version via runtime/debug (no ldflags required)
**What:** Read VCS metadata embedded by `go build` from the binary itself at runtime.
**When to use:** When `go install` is the distribution mechanism — ldflags require custom build scripts.

```go
// Source: Go stdlib runtime/debug docs
package version

import (
    "fmt"
    "runtime/debug"
)

const fallbackVersion = "0.1.0"

func String() string {
    info, ok := debug.ReadBuildInfo()
    if !ok {
        return fallbackVersion + " (unknown)"
    }
    hash := "unknown"
    for _, s := range info.Settings {
        if s.Key == "vcs.revision" {
            if len(s.Value) >= 7 {
                hash = s.Value[:7]
            }
            break
        }
    }
    return fmt.Sprintf("%s (%s)", fallbackVersion, hash)
}
```

**Note:** If ldflags approach is preferred (for CI/goreleaser), set `var Version = "dev"` in version.go and inject at build time with:
```bash
go build -ldflags "-X github.com/The-Artificer-of-Ciphers-LLC/gsd-wired/internal/version.Version=0.1.0" ./cmd/gsdw
```
Both approaches are valid. `runtime/debug` works out-of-the-box with `go install`.

### Pattern 6: Plugin Manifest Structure (verified 2026-03-21)

**`.claude-plugin/plugin.json`** — metadata only:
```json
{
  "name": "gsd-wired",
  "version": "0.1.0",
  "description": "Token-efficient development lifecycle on a versioned graph",
  "author": {
    "name": "The Artificer of Ciphers LLC"
  }
}
```

**`.mcp.json`** — MCP server registration (at repo root, not inside .claude-plugin/):
```json
{
  "mcpServers": {
    "gsd-wired": {
      "command": "gsdw",
      "args": ["serve"]
    }
  }
}
```

**`hooks/hooks.json`** — hook event registrations (at repo root, not inside .claude-plugin/):
```json
{
  "hooks": {
    "SessionStart": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "gsdw hook SessionStart"
          }
        ]
      }
    ],
    "PreCompact": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "gsdw hook PreCompact"
          }
        ]
      }
    ],
    "PreToolUse": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "gsdw hook PreToolUse"
          }
        ]
      }
    ],
    "PostToolUse": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "gsdw hook PostToolUse"
          }
        ]
      }
    ]
  }
}
```

**CRITICAL structural rule:** Only `plugin.json` lives inside `.claude-plugin/`. All other directories (`hooks/`, `commands/`, `agents/`, `skills/`) MUST be at the plugin root (repo root), NOT inside `.claude-plugin/`. Common mistake that causes components to silently not load.

### Anti-Patterns to Avoid
- **Putting hooks/ inside .claude-plugin/:** Components must be at plugin root. Only plugin.json goes in .claude-plugin/.
- **Writing to stdout in hook handlers:** Any non-JSON byte corrupts the protocol. Always `os.Stderr` for logs.
- **Registering future stubs in plugin.json:** Decision D-09 says manifest grows per phase — don't forward-declare Phase 3 tools.
- **Using log.Println (default logger):** Default slog writes to stdout before `initLogger` runs. Set stderr handler before any logging.
- **Cobra printing errors to stdout:** `SilenceUsage: true` + `SilenceErrors: true` required. Handle errors in main() writing to stderr.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| MCP protocol framing | Custom JSON-RPC over stdio | `mcp.NewServer` + `mcp.StdioTransport{}` | JSON-RPC 2.0 framing, capabilities negotiation, lifecycle — 300+ lines of fiddly protocol |
| CLI flag parsing | Custom args parsing | Cobra | Persistent flags, subcommand isolation, help generation, completion |
| Structured logging | `fmt.Fprintf(os.Stderr, ...)` | `log/slog` | Leveled, structured, multi-format, zero deps |
| Version embedding | Custom file parsing | `runtime/debug.ReadBuildInfo()` | Reads VCS metadata embedded by `go build` automatically |

**Key insight:** The MCP protocol has enough edge cases (capability negotiation, batch requests, error codes) that hand-rolling it would cost a week and still miss edge cases. The SDK handles all of this.

## Hook Protocol Reference (verified 2026-03-21)

### Hook blocking behavior — critical for Phase 4+ implementation

| Hook | Blocking | Can Block Execution | Exit 2 Behavior |
|------|----------|--------------------|----|
| `SessionStart` | NO | Cannot block session | Shows stderr to user |
| `PreToolUse` | YES | Can deny tool call | Blocks tool call, stderr fed to Claude |
| `PostToolUse` | NO | Cannot block (tool ran) | Stderr shown to Claude |
| `PreCompact` | NO | Cannot block compaction | Shows stderr to user only |

**PreCompact is NON-BLOCKING.** Verified against official docs 2026-03-21. The prior research SUMMARY.md correctly identified this. Phase 1 stubs must not assume PreCompact can prevent compaction.

### Hook input fields (common across all hooks)
```json
{
  "session_id": "string",
  "transcript_path": "string — path to conversation .jsonl",
  "cwd": "string — working directory",
  "hook_event_name": "string — matches the event name",
  "permission_mode": "string — present on PreToolUse/PostToolUse"
}
```

### SessionStart-specific input fields
```json
{
  "source": "startup|resume|clear|compact",
  "model": "claude-sonnet-4-6"
}
```

### PreToolUse-specific input fields
```json
{
  "tool_name": "Bash|Write|Edit|Read|...",
  "tool_input": {},
  "tool_use_id": "toolu_01..."
}
```

### PostToolUse-specific input fields
```json
{
  "tool_name": "string",
  "tool_input": {},
  "tool_response": {},
  "tool_use_id": "string"
}
```

### PreCompact-specific input fields
```json
{
  "trigger": "manual|auto",
  "custom_instructions": "string"
}
```

### SessionStart response (adds context)
```json
{
  "hookSpecificOutput": {
    "hookEventName": "SessionStart",
    "additionalContext": "string injected into Claude's context"
  }
}
```

### PreToolUse response (can allow/deny)
```json
{
  "hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "permissionDecision": "allow|deny|ask",
    "permissionDecisionReason": "string",
    "additionalContext": "string"
  }
}
```

## Common Pitfalls

### Pitfall 1: Stdout Pollution from Cobra Error Handling
**What goes wrong:** Cobra by default writes usage and error messages to stdout when a command fails. In MCP serve mode, any non-JSON byte on stdout breaks the JSON-RPC framing silently — the client just disconnects.
**Why it happens:** Cobra's default `SetOut` is os.Stdout. Developers test with `hook` subcommand and don't notice stdout pollution in serve mode.
**How to avoid:** Set `SilenceUsage: true` and `SilenceErrors: true` on the root cobra.Command. Handle errors in `main()` by writing to `slog.Error()` then `os.Exit(1)`. Never use `fmt.Print*` anywhere.
**Warning signs:** MCP client disconnects immediately after `initialize` succeeds. Hook response parsing fails intermittently.

### Pitfall 2: slog Default Logger Writing to Stdout
**What goes wrong:** `slog.SetDefault` is not called before the first log statement. The default handler writes to stdout.
**Why it happens:** `PersistentPreRunE` runs after argument parsing, which may log. Any `init()` functions that log will use the default handler.
**How to avoid:** Set a stderr-only slog handler in `main()` before `root.Execute()`, not in `PersistentPreRunE`. Use a conservative default (error level, text format).
**Warning signs:** MCP initialize succeeds then breaks on subsequent messages; debug shows extra lines before valid JSON.

### Pitfall 3: Plugin Components Inside .claude-plugin/
**What goes wrong:** Placing `hooks/`, `commands/`, or other directories inside `.claude-plugin/` instead of at the repo root. Claude Code silently ignores them.
**Why it happens:** The `.claude-plugin/` name implies it's the config directory for all plugin contents.
**How to avoid:** Only `plugin.json` goes in `.claude-plugin/`. All other directories at repo root.
**Warning signs:** `claude --debug` shows plugin loading but hooks never fire; `/gsd-wired:` commands not available.

### Pitfall 4: Hook Binary Not on PATH
**What goes wrong:** `hooks/hooks.json` references `gsdw hook SessionStart` but `gsdw` is not in PATH when Claude Code executes the hook.
**Why it happens:** `go install` puts the binary in `~/go/bin/`, which may not be in the shell environment Claude Code inherits.
**How to avoid:** Use absolute path in hooks.json: `"command": "/Users/trekkie/go/bin/gsdw hook SessionStart"`. Or use `${CLAUDE_PLUGIN_ROOT}` if distributing as a plugin. For local development, verify PATH inheritance.
**Warning signs:** Hooks registered but never fire; no output in Claude Code's hook debug log.

### Pitfall 5: MCP Server Initialization Latency
**What goes wrong:** Heavy initialization (DB connections, file parsing) before `server.Run()` causes the MCP client to time out waiting for the server to be ready.
**Why it happens:** Dolt initialization in Phase 3 can take 3-10 seconds.
**How to avoid:** In Phase 1, `serve` subcommand must call `server.Run()` immediately with zero initialization. No Dolt in Phase 1. Establish the pattern: heavy init happens lazily on first tool call, not at startup.
**Warning signs:** MCP tool panel shows "server connecting..." indefinitely; client logs show timeout.

### Pitfall 6: Hook Subcommand Not Accepting Event as Argument
**What goes wrong:** `gsdw hook` requires the event name as an argument (e.g., `gsdw hook SessionStart`), but the Cobra subcommand accepts it as `Args: cobra.ExactArgs(1)`. Forgetting `Args` validation causes panic on empty input.
**Why it happens:** hooks.json specifies the full command string; if it's wrong, the hook binary receives no args.
**How to avoid:** Set `Args: cobra.ExactArgs(1)` on the hook subcommand. Validate the event name against a known set of constants. Return exit code 2 with an error on stderr for unknown events.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go stdlib `testing` — no external test framework needed |
| Config file | None — `go test ./...` discovers tests automatically |
| Quick run command | `go test ./...` |
| Full suite command | `go test ./... -race -count=1` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| INFRA-01 | Binary compiles and `gsdw version` exits 0 | smoke | `go build ./cmd/gsdw && ./gsdw version` | ❌ Wave 0 |
| INFRA-01 | `gsdw serve` starts MCP server responding on stdio | integration | `go test ./internal/mcp/... -run TestServeRespondsToInitialize` | ❌ Wave 0 |
| INFRA-01 | `gsdw hook SessionStart` reads stdin JSON and exits 0 | unit | `go test ./internal/hook/... -run TestDispatchSessionStart` | ❌ Wave 0 |
| INFRA-04 | `.claude-plugin/plugin.json` is valid JSON with required fields | unit | `go test ./... -run TestPluginManifestValid` | ❌ Wave 0 |
| INFRA-04 | `hooks/hooks.json` is valid JSON with correct event names | unit | `go test ./... -run TestHooksJsonValid` | ❌ Wave 0 |
| INFRA-09 | `gsdw hook SessionStart` produces no non-JSON bytes on stdout | unit | `go test ./internal/hook/... -run TestHookStdoutPurity` | ❌ Wave 0 |
| INFRA-09 | Error logging goes to stderr not stdout | unit | `go test ./... -run TestLoggerWritesToStderr` | ❌ Wave 0 |

**Note:** INFRA-01's MCP integration test requires sending `{"jsonrpc":"2.0","id":1,"method":"initialize",...}` to the process stdin and reading back a valid `initialize` response. Use `os/exec` to start the binary as a subprocess.

### Sampling Rate
- **Per task commit:** `go test ./... -count=1`
- **Per wave merge:** `go test ./... -race -count=1`
- **Phase gate:** Full suite green + manual smoke test: `gsdw version`, `gsdw serve` (MCP init), `gsdw hook SessionStart` (JSON in → JSON out) before `/gsd:verify-work`

### Wave 0 Gaps
All test files need to be created — no existing test infrastructure:
- [ ] `internal/hook/dispatcher_test.go` — covers INFRA-01 (hook dispatch), INFRA-09 (stdout purity)
- [ ] `internal/mcp/server_test.go` — covers INFRA-01 (MCP serve)
- [ ] `internal/version/version_test.go` — covers version output format
- [ ] `cmd/gsdw/main_test.go` — integration: binary builds and runs

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `mark3labs/mcp-go` (unofficial) | `modelcontextprotocol/go-sdk` (official) | 2025 | Use official SDK; mark3labs still maintained but v0.x unofficial |
| `log` stdlib | `log/slog` stdlib | Go 1.21 (2023) | slog is now the standard; log package is legacy |
| Manual ldflags for version | `runtime/debug.ReadBuildInfo()` | Go 1.12 (2019) | Works with `go install`, no build script required |
| hooks in settings.json | `hooks/hooks.json` in plugin | Claude Code plugin system | Plugin-scoped hooks in dedicated file, same format as settings hooks |

**Deprecated/outdated:**
- `mark3labs/mcp-go`: Not the official SDK. Use `modelcontextprotocol/go-sdk` instead.
- `log.Println` / `log.Printf`: Use `slog.Info()`, `slog.Error()` etc. Default slog can be stderr-configured.

## Open Questions

1. **PATH inheritance for hooks**
   - What we know: Claude Code executes hook commands via shell; `gsdw` must be on the PATH that shell inherits.
   - What's unclear: Does Claude Code inherit the user's full shell PATH including `~/go/bin/`?
   - Recommendation: During Phase 1 testing, verify hook execution by checking Claude Code logs. If not on PATH, use absolute path in hooks.json initially. Plan for this in installation docs (Phase 10).

2. **hooks.json location for local development (non-installed plugin)**
   - What we know: When using `--plugin-dir` flag, hooks at `hooks/hooks.json` are loaded. When installed via marketplace, same structure applies.
   - What's unclear: For development without `--plugin-dir` (using local project scope), hooks go in `.claude/settings.json` or `.claude/settings.local.json`.
   - Recommendation: Phase 1 should support both: `hooks/hooks.json` for plugin mode, and document local dev setup for project scope.

3. **MCP server registration for local development**
   - What we know: `.mcp.json` at repo root registers servers in project scope.
   - What's unclear: Whether `gsdw serve` is auto-started or must be pre-running when `.mcp.json` is present.
   - Recommendation: Claude Code auto-starts MCP servers from `.mcp.json`. Test by adding `.mcp.json` and observing tool panel.

## Sources

### Primary (HIGH confidence)
- `github.com/modelcontextprotocol/go-sdk@v1.4.1` — locally cached module, examined source and examples directly
- [Claude Code Hooks Reference](https://code.claude.com/docs/en/hooks) — verified hook input/output JSON schemas 2026-03-21
- [Claude Code Plugins Reference](https://code.claude.com/docs/en/plugins-reference) — verified plugin.json schema, .mcp.json format, hooks/hooks.json structure 2026-03-21
- [Claude Code Plugins Guide](https://code.claude.com/docs/en/plugins) — plugin directory structure, common mistakes 2026-03-21
- Go stdlib documentation — slog, runtime/debug verified against Go 1.26.1

### Secondary (MEDIUM confidence)
- `go list -m -json` package registry — confirmed cobra v1.10.2 (2025-12-03), mcp-go-sdk v1.4.1 (2026-03-13)
- Prior project research summary (`.planning/research/SUMMARY.md`) — stack decisions and pitfall warnings cross-referenced

### Tertiary (LOW confidence)
- None — all claims verified against primary sources for this phase's scope

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — versions confirmed via `go list -m -json` against live registry
- Architecture: HIGH — plugin manifest structure verified against official docs 2026-03-21
- Hook protocol: HIGH — input/output schemas verified against official docs; blocking behavior confirmed (PreCompact is NON-blocking)
- Pitfalls: HIGH — stdout pollution and component location pitfalls confirmed in official docs troubleshooting section
- MCP SDK: HIGH — examined actual source at v1.4.1, ran hello example pattern

**Research date:** 2026-03-21
**Valid until:** 2026-06-21 (stable ecosystem — plugin manifest schema unlikely to change; MCP SDK minor versions acceptable)
