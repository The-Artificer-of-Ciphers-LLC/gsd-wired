# gsd-wired

Token-efficient development lifecycle on a versioned graph. Fuses [GSD](https://github.com/anthropics/claude-code) workflow orchestration with [Beads](https://github.com/beads-project/beads)/Dolt graph persistence so Claude Code subagents pull only the context they need.

## What It Does

gsd-wired is a Claude Code plugin (MCP server + hooks + skills) that replaces markdown-file state management with a Dolt-backed graph. Phases map to epic beads, plans map to task beads, and waves are computed from the dependency graph. The result: subagent prompts contain claimed bead context instead of entire files, cutting token usage dramatically.

**Single Go binary** serves as MCP server, hook dispatcher, and CLI tool.

## Quick Start

### Install

```bash
# Homebrew (macOS)
brew tap The-Artificer-of-Ciphers-LLC/gsdw
brew install --cask gsdw-cc

# Go
go install github.com/The-Artificer-of-Ciphers-LLC/gsd-wired/cmd/gsdw@latest

# Docker
docker pull ghcr.io/the-artificer-of-ciphers-llc/gsdw:latest
```

### Prerequisites

gsd-wired requires [bd](https://github.com/beads-project/beads) and [dolt](https://github.com/dolthub/dolt) on your PATH:

```bash
gsdw check-deps    # Verify what's installed
gsdw setup         # Interactive wizard to install missing deps
```

### Initialize a Project

```bash
gsdw init                      # Create .beads/ + .gsdw/ + PROJECT.md
gsdw container start           # Start local Dolt server
gsdw connect                   # Configure connection to Dolt
gsdw doctor                    # Verify everything is healthy
```

### Use via Claude Code

The plugin registers slash commands for the full GSD lifecycle:

```
/gsd-wired:init [full|quick|pr]   # Deep questioning flow
/gsd-wired:research N              # Parallel research agents
/gsd-wired:plan N                  # Dependency-aware task planning
/gsd-wired:execute N               # Wave-based parallel execution
/gsd-wired:verify N                # Acceptance criteria verification
/gsd-wired:ship N                  # PR creation + phase advancement
/gsd-wired:status                  # Project dashboard
/gsd-wired:ready                   # Unblocked tasks
```

## Architecture

```
cmd/gsdw/main.go          Entry point
internal/
  cli/                     17 Cobra commands
  mcp/                     18 MCP tools + server
  hook/                    4 hook dispatchers
  graph/                   Beads graph client (bd wrapper)
  container/               Docker/Podman/Apple Container runtime
  connection/              Dolt connection config + health check
  deps/                    Dependency detection
  compat/                  .planning/ fallback (read-only)
  version/                 Version info (ldflags + ReadBuildInfo)
  logging/                 Structured logging (slog)
skills/                    8 slash command manifests
hooks/                     Hook entry points
.claude-plugin/            Plugin manifest
```

### How It Works

1. **Hooks** inject graph context automatically:
   - `SessionStart` loads active project state into Claude's context
   - `PreToolUse` injects relevant bead context before tool calls
   - `PostToolUse` records tool outcomes to the graph
   - `PreCompact` saves state before context compaction

2. **MCP tools** expose the full lifecycle to Claude:
   - Phase/plan/bead CRUD operations
   - Wave computation from dependency graph
   - Token-aware context tiering (hot/warm/cold)
   - PR summary generation from bead metadata

3. **Skills** orchestrate multi-agent workflows:
   - Research spawns 4 parallel agents (stack, features, architecture, pitfalls)
   - Execute runs task waves in parallel, validates acceptance criteria
   - Verify checks criteria against codebase, creates remediation tasks

## CLI Reference

| Command | Description |
|---------|-------------|
| `gsdw version [--json]` | Print version information |
| `gsdw check-deps [--json]` | Check required dependencies |
| `gsdw setup` | Interactive dependency installation wizard |
| `gsdw doctor` | Full environment and project health check |
| `gsdw init` | Initialize beads directory and project files |
| `gsdw container start [--port N] [--force]` | Start Dolt server container |
| `gsdw container stop` | Stop Dolt container |
| `gsdw connect` | Configure Dolt server connection |
| `gsdw status` | Show project status from graph |
| `gsdw ready [--phase N] [--json]` | Show unblocked tasks |
| `gsdw serve` | Start MCP stdio server |
| `gsdw hook <event>` | Dispatch a hook event |
| `gsdw bd [args...]` | Passthrough to bd CLI |

## Configuration

### .gsdw/connection.json

Created by `gsdw connect`. Stores local and remote Dolt server configuration:

```json
{
  "active_mode": "local",
  "local": { "host": "127.0.0.1", "port": "3307" },
  "remote": { "host": "db.example.com", "port": "3306", "user": "admin" },
  "configured": "2026-03-22T00:00:00Z"
}
```

Environment variables injected into every `bd` subprocess:
- `BEADS_DOLT_SERVER_HOST` — from active connection config
- `BEADS_DOLT_SERVER_PORT` — from active connection config
- `GSDW_DB_PASSWORD` — set externally for authenticated connections

### Container Runtimes

`gsdw container start` auto-detects the available runtime:

1. **Apple Container** (macOS 26 + Apple Silicon)
2. **Docker** (fallback)
3. **Podman** (fallback)

Produces a `gsdw.compose.yaml` fragment for Docker/Podman integration.

## Compatibility

gsd-wired coexists with `.planning/` directories used by vanilla GSD. The `compat` package reads ROADMAP.md and STATE.md as fallback when beads graph is unavailable. All `.planning/` access is read-only.

## Development

```bash
go test ./...              # Run all tests (~340 across 11 packages)
go build ./cmd/gsdw        # Build binary
gsdw version --json        # Verify build
```

## Release

Local release with Apple codesigning + notarization:

```bash
# One-time: create .env.release with signing credentials (see Makefile for details)
make release-mac-snapshot    # Dry run (no publish)
make release-mac             # Full release (push to GitHub + brew tap)
```

Release pipeline:
- Cross-platform binaries (darwin/linux x amd64/arm64)
- macOS binaries signed with Developer ID Application certificate
- macOS binaries notarized by Apple (via `codesign` + `xcrun notarytool`)
- Homebrew cask auto-published to tap repo
- Docker images built in CI (skipped locally)

## License

MIT
