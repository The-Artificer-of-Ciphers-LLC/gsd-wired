# Technology Stack

**Project:** gsd-wired (Claude Code Plugin: GSD + Beads/Dolt Agent Orchestration)
**Researched:** 2026-03-21

## Recommended Stack

### Core Runtime & Language

| Technology | Version | Purpose | Why | Confidence |
|------------|---------|---------|-----|------------|
| Go | 1.25.x | Implementation language | Matches beads (Go 1.25.8) for tight integration, single binary distribution, potential upstream contributions to bd. No runtime dependencies for end users. | HIGH |

### MCP Server SDK

| Technology | Version | Purpose | Why | Confidence |
|------------|---------|---------|-----|------------|
| modelcontextprotocol/go-sdk | v1.4.1 | MCP server implementation | Official Go SDK maintained in collaboration with Google. Typed tool handlers with automatic JSON schema inference from Go structs. StdioTransport for Claude Code integration. Supports MCP spec 2025-11-25. | HIGH |

The official SDK (`modelcontextprotocol/go-sdk`) is the correct choice over the community `mark3labs/mcp-go`. The official SDK reached v1.4.1 (March 2026) and has Google co-maintenance. The community library was the go-to before the official SDK existed but is now secondary.

**Key API patterns this project will use:**

```go
// Server creation
server := mcp.NewServer(&mcp.Implementation{
    Name:    "gsd-wired",
    Version: "v0.1.0",
}, nil)

// Typed tool registration (auto-generates JSON schema from struct)
type CreatePhaseArgs struct {
    Name        string `json:"name" jsonschema:"phase name"`
    Description string `json:"description" jsonschema:"phase description"`
}
mcp.AddTool(server, &mcp.Tool{
    Name:        "gsd_create_phase",
    Description: "Create a new GSD phase as an epic bead",
}, createPhaseHandler)

// Run over stdio (Claude Code transport)
server.Run(ctx, &mcp.StdioTransport{})
```

### Beads Integration

| Technology | Version | Purpose | Why | Confidence |
|------------|---------|---------|-----|------------|
| github.com/steveyegge/beads | v0.61.0 | Graph persistence API | Go package with `Storage` interface and `Transaction` support. Provides `Open()`, `FindBeadsDir()`, `FindDatabasePath()` for programmatic access. Same language = direct import, no CLI shelling. | HIGH |
| bd CLI | v0.61.0 | Fallback / user-facing commands | Users need bd installed anyway. Use programmatic API for hot path, CLI for edge cases or user-initiated operations. | HIGH |

**Critical insight:** Beads exports a minimal Go API (`Storage`, `Transaction`, issue types, dependency types) but recommends "most extensions should use direct SQL queries against bd's database." This means we should use the Go API for discovery/setup and direct SQL for performance-critical graph queries.

**Integration strategy (two layers):**
1. **Go API layer** -- `beads.Open()` / `beads.OpenFromConfig()` for database access, `Storage.RunInTransaction()` for atomic operations
2. **Direct SQL layer** -- Custom queries against Dolt tables for GSD-specific operations (wave computation, bulk status updates, context loading)

### Database Access

| Technology | Version | Purpose | Why | Confidence |
|------------|---------|---------|-----|------------|
| github.com/dolthub/driver | v1.83.8 | Embedded Dolt database access | Go `database/sql` compatible driver. Enables in-process DB access without running a Dolt server. Beads already uses this as a dependency. | HIGH |

**Connection pattern:**
```go
cfg, _ := embedded.ParseDSN("file:///path/to/.beads/dolt?commitname=gsd-wired&commitemail=gsd@local&database=beads")
connector, _ := embedded.NewConnector(cfg)
db := sql.OpenDB(connector)
```

**Do NOT use:** `go-sql-driver/mysql` with a running Dolt SQL server. The embedded driver avoids server process management and matches beads' own approach. Server mode is for multi-user scenarios we don't need in v1.

### CLI Framework

| Technology | Version | Purpose | Why | Confidence |
|------------|---------|---------|-----|------------|
| spf13/cobra | v1.10.2 | CLI subcommands | Beads uses it (consistency), industry standard for Go CLIs, handles the dual-mode binary pattern (CLI commands + MCP server mode). | HIGH |

The binary needs to serve as both an MCP stdio server (when invoked by Claude Code) AND a standalone CLI (for debugging, manual operations). Cobra handles this cleanly with subcommands:

```
gsd-wired serve    # MCP stdio mode (Claude Code invokes this)
gsd-wired status   # Manual CLI usage
gsd-wired migrate  # Import from .planning/
```

### Configuration

| Technology | Version | Purpose | Why | Confidence |
|------------|---------|---------|-----|------------|
| spf13/viper | v1.21.0 | Configuration management | Beads uses it. Handles config file discovery, env var overrides, defaults. Reads PROJECT.md companion config.json. | MEDIUM |

Viper may be overkill for this project's config needs (config.json + env vars). Consider dropping it if config stays simple. But matching beads' dependency reduces cognitive overhead.

### Testing

| Technology | Version | Purpose | Why | Confidence |
|------------|---------|---------|-----|------------|
| Go standard testing | 1.25.x | Unit tests | Go convention. Table-driven tests cover the majority of cases. | HIGH |
| stretchr/testify | v1.8.3+ | Assertions & mocks | Cleaner assertions for complex struct comparisons (bead states, graph queries). Beads uses it. | HIGH |
| testcontainers-go | latest | Integration tests | Beads uses it for Dolt integration tests. Spin up isolated Dolt environments per test. | MEDIUM |

### Build & Distribution

| Technology | Version | Purpose | Why | Confidence |
|------------|---------|---------|-----|------------|
| goreleaser | latest | Cross-platform binary builds | Standard Go binary distribution. Produces single binaries for macOS/Linux. Homebrew tap integration. | MEDIUM |
| go install | 1.25.x | Developer install path | `go install github.com/user/gsd-wired/cmd/gsd-wired@latest` for Go developers. | HIGH |

## Claude Code Plugin Structure

The plugin bundles the Go binary with skills, hooks, and MCP server configuration.

```
gsd-wired/
├── .claude-plugin/
│   └── plugin.json              # Plugin manifest
├── skills/
│   ├── new-project/
│   │   └── SKILL.md             # /gsd:new-project
│   ├── new-milestone/
│   │   └── SKILL.md             # /gsd:new-milestone
│   ├── transition/
│   │   └── SKILL.md             # /gsd:transition
│   └── ...
├── agents/
│   ├── researcher.md            # Research subagent
│   ├── planner.md               # Planning subagent
│   └── executor.md              # Execution subagent
├── hooks/
│   └── hooks.json               # SessionStart, PreCompact, etc.
├── .mcp.json                    # MCP server config
├── scripts/
│   └── install.sh               # Build + install Go binary
├── cmd/
│   └── gsd-wired/
│       └── main.go              # Binary entry point
├── internal/
│   ├── mcp/                     # MCP tool handlers
│   ├── orchestrator/            # GSD workflow logic
│   ├── beads/                   # Beads integration layer
│   └── hooks/                   # Hook handler logic
└── go.mod
```

**Plugin MCP configuration** (`.mcp.json`):
```json
{
  "mcpServers": {
    "gsd-wired": {
      "command": "${CLAUDE_PLUGIN_ROOT}/bin/gsd-wired",
      "args": ["serve"],
      "env": {
        "GSD_PLUGIN_ROOT": "${CLAUDE_PLUGIN_ROOT}",
        "GSD_PLUGIN_DATA": "${CLAUDE_PLUGIN_DATA}"
      }
    }
  }
}
```

**Hook configuration** (`hooks/hooks.json`):
```json
{
  "hooks": {
    "SessionStart": [
      {
        "hooks": [{
          "type": "command",
          "command": "${CLAUDE_PLUGIN_ROOT}/bin/gsd-wired hook session-start"
        }]
      }
    ],
    "PreCompact": [
      {
        "hooks": [{
          "type": "command",
          "command": "${CLAUDE_PLUGIN_ROOT}/bin/gsd-wired hook pre-compact"
        }]
      }
    ],
    "PreToolUse": [
      {
        "matcher": "mcp__gsd-wired__.*",
        "hooks": [{
          "type": "command",
          "command": "${CLAUDE_PLUGIN_ROOT}/bin/gsd-wired hook pre-tool-use"
        }]
      }
    ],
    "PostToolUse": [
      {
        "matcher": "mcp__gsd-wired__.*",
        "hooks": [{
          "type": "command",
          "command": "${CLAUDE_PLUGIN_ROOT}/bin/gsd-wired hook post-tool-use"
        }]
      }
    ]
  }
}
```

**Key plugin variables:**
- `${CLAUDE_PLUGIN_ROOT}` -- absolute path to plugin installation (changes on update)
- `${CLAUDE_PLUGIN_DATA}` -- persistent data directory (`~/.claude/plugins/data/gsd-wired/`) survives updates

## Supporting Libraries (from beads ecosystem)

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| charmbracelet/lipgloss | latest | Terminal styling | CLI output formatting (status displays, progress) |
| olebedev/when | latest | Natural language dates | Parsing deadline expressions in phase definitions |
| fsnotify/fsnotify | latest | File watching | Detecting .planning/ file changes for coexistence mode |

These are optional and should only be added if the feature requires them. Start without them.

## Alternatives Considered

| Category | Recommended | Alternative | Why Not |
|----------|-------------|-------------|---------|
| MCP SDK | modelcontextprotocol/go-sdk | mark3labs/mcp-go | Official SDK now exists with Google co-maintenance; community lib is legacy |
| MCP SDK | modelcontextprotocol/go-sdk | Write raw JSON-RPC | SDK handles protocol versioning, schema generation, transport. Rolling your own gains nothing. |
| Beads integration | Go API import | Shell out to bd CLI | Same language = direct import. CLI shelling adds overhead, parsing fragility, and process spawning per operation. |
| Beads integration | Go API + direct SQL | Go API only | Beads Go API is intentionally minimal. Complex queries (wave computation, bulk ops) need SQL. |
| DB access | Embedded Dolt driver | Dolt SQL server | No server process to manage. Embedded = same process, lower latency, simpler deployment. |
| CLI framework | cobra | urfave/cli | Beads uses cobra. Consistency matters when contributing upstream. |
| CLI framework | cobra | No CLI (MCP only) | Need CLI for debugging, manual operations, migration tooling. MCP-only means blind debugging. |
| Config | viper | Manual JSON parsing | Beads uses viper. But watch for over-engineering -- drop if config stays trivial. |
| Testing | testify + stdlib | stdlib only | Bead graph assertions are complex (deeply nested structs). Testify's `assert.Equal` is worth the dependency. |
| Language | Go | TypeScript | TypeScript has richer MCP ecosystem but creates language mismatch with beads. Can't import beads Go packages from TS. Single binary distribution is better UX than node_modules. |
| Language | Go | Rust | Overkill for this domain. No beads API access. Longer dev time for marginal perf gains on I/O-bound work. |

## What NOT to Use

| Technology | Why Not |
|------------|---------|
| mark3labs/mcp-go | Superseded by official Go SDK. Will likely converge or deprecate. |
| go-dolt (third-party) | Use official dolthub/driver. Third-party wrappers add abstraction without value. |
| gRPC | MCP uses JSON-RPC 2.0 over stdio. gRPC adds unnecessary complexity. |
| SQLite | Dolt IS the database (from beads). Don't add a second DB. |
| Redis/memcached | Local-only v1. In-process caching if needed. |
| Docker | Single binary distribution. Docker adds friction for Claude Code plugin users. |
| Protobuf | MCP protocol is JSON-based. Protobuf doesn't fit. |
| Any ORM (GORM, ent) | Beads has its own schema. Direct SQL via database/sql is the right abstraction level. |

## Installation

```bash
# Prerequisites
go install github.com/steveyegge/beads/cmd/bd@latest  # beads CLI
brew install dolt                                       # or: go install github.com/dolthub/dolt/go/cmd/dolt@latest

# Core dependencies (go.mod)
go get github.com/modelcontextprotocol/go-sdk@v1.4.1
go get github.com/steveyegge/beads@v0.61.0
go get github.com/dolthub/driver@v1.83.8
go get github.com/spf13/cobra@v1.10.2

# Dev dependencies
go get github.com/stretchr/testify@latest
```

## Go Module Init

```bash
go mod init github.com/user/gsd-wired
```

## Version Compatibility Matrix

| Component | Min Version | Tested With | Notes |
|-----------|-------------|-------------|-------|
| Go | 1.25.0 | 1.25.8 | Match beads' Go version |
| beads (bd) | v0.61.0 | v0.61.0 | Current latest |
| Dolt | v1.83.0 | v1.83.8 | Via embedded driver |
| MCP Go SDK | v1.4.0 | v1.4.1 | Spec 2025-11-25 |
| Claude Code | Current | 2026-03 | Plugin system + hooks |

## Architecture Decision: Single Binary, Dual Mode

The gsd-wired binary operates in two modes from the same executable:

1. **MCP Server Mode** (`gsd-wired serve`): Claude Code spawns this via stdio. Registers tools, handles JSON-RPC requests. Long-running process per session.

2. **Hook Handler Mode** (`gsd-wired hook <event>`): Claude Code invokes per-event. Receives JSON on stdin, returns JSON on stdout. Short-lived process per hook invocation.

3. **CLI Mode** (`gsd-wired status`, `gsd-wired migrate`, etc.): User runs directly for debugging and manual operations.

All three modes share the same `internal/` packages. The binary detects its mode from the cobra subcommand.

**Hook vs MCP tradeoff:** Hooks are separate process invocations (cold start per event), while MCP tools run in the long-lived server process. Put latency-sensitive operations in MCP tools; put policy/validation in hooks.

## Sources

- [Official MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk) -- v1.4.1, March 2026
- [MCP Go SDK pkg.go.dev](https://pkg.go.dev/github.com/modelcontextprotocol/go-sdk/mcp) -- API reference
- [Beads (steveyegge/beads)](https://github.com/steveyegge/beads) -- v0.61.0, Go package API
- [Beads Go Package](https://pkg.go.dev/github.com/steveyegge/beads) -- Storage, Transaction interfaces
- [Beads Plugin Docs](https://github.com/steveyegge/beads/blob/main/docs/PLUGIN.md) -- MCP integration
- [Dolt Embedded Driver](https://github.com/dolthub/driver) -- v1.83.8, database/sql compatible
- [Dolt Go SQL Blog](https://www.dolthub.com/blog/2025-01-24-go-sql-with-dolt/) -- Using sqlx with Dolt
- [Claude Code Hooks Reference](https://code.claude.com/docs/en/hooks) -- All 22+ hook events
- [Claude Code Plugins Reference](https://code.claude.com/docs/en/plugins-reference) -- Plugin structure, manifest schema
- [Claude Code MCP Docs](https://code.claude.com/docs/en/mcp) -- MCP server configuration
- [MCP Specification](https://modelcontextprotocol.io/specification/2025-11-25) -- Protocol spec
