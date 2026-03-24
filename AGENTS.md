# gsd-wired

Go CLI + MCP server + hook dispatcher for token-efficient dev lifecycle on a Dolt-backed beads graph.

## Build & Test

```bash
go build ./cmd/gsdw              # build binary
go test ./...                     # run all 394 tests
gsdw version --json               # verify build info
gsdw check-deps --json            # check bd, dolt, go, container runtime
gsdw doctor                       # full health check
make release-mac-snapshot          # dry-run signed release
```

## Architecture

**Entry**: `cmd/gsdw/main.go` → `internal/cli/root.go` (`Execute()`)

| Package | Purpose | Key files |
|---|---|---|
| `internal/cli/` | 17 Cobra commands | `root.go` · `init.go` · `connect.go` · `doctor.go` · `container.go` · `serve.go` · `hook.go` · `ready.go` · `status.go` |
| `internal/mcp/` | 19 MCP tools + stdio server | `server.go` · `tools.go` · `init.go` · `execute_wave.go` · `verify_phase.go` · `get_tiered_context.go` |
| `internal/hook/` | 4 hook dispatchers | `dispatcher.go` · `events.go` · `session_start.go` · `pre_tool_use.go` · `post_tool_use.go` · `pre_compact.go` |
| `internal/graph/` | bd CLI wrapper (beads CRUD) | `client.go` · `bead.go` · `query.go` · `create.go` · `update.go` · `tier.go` · `index.go` |
| `internal/container/` | Docker/Podman/Apple Container | `runtime.go` · `compose.go` |
| `internal/connection/` | Dolt server config + health | `config.go` |
| `internal/deps/` | Dependency detection | `check.go` |
| `internal/compat/` | `.planning/` fallback (read-only) | `compat.go` |
| `internal/version/` | Version via ldflags + `ReadBuildInfo` | `version.go` |
| `internal/logging/` | Structured slog to stderr | `logging.go` |

**Plugin files**: `.mcp.json` (MCP config) · `hooks/hooks.json` (4 hooks) · `skills/` (8 slash commands)

## Conventions

- **Stdout discipline**: MCP server and hooks write JSON to stdout only. All logs go to stderr via `slog`.
- **Test pattern**: Tests use `internal/graph/testdata/fake_bd/` — a fake `bd` binary built at test time. Set `FAKE_BD_*` env vars to control responses.
- **Connection config**: `internal/connection/config.go` — `FlexPort` accepts both string and numeric JSON port values.
- **Container runtime detection**: Priority order in `internal/container/runtime.go`: Apple Container (macOS 26+ARM64) > Docker > Podman.
- **Cobra commands**: Each command is `New*Cmd() *cobra.Command` in `internal/cli/`, registered in `root.go`.
- **MCP tools**: Registered in `registerTools()` in `internal/mcp/tools.go`. Each handler is `handle*()` returning `(*mcpsdk.CallToolResult, error)`.
- **Hook handlers**: Dispatched by event name in `internal/hook/dispatcher.go`. Each is `handle*()` with `(ctx, raw, hookState, writer)` signature.
- **Graph client**: `internal/graph/client.go` wraps `bd` CLI. `NewClient()` for immediate writes, `NewClientBatch()` for deferred writes flushed via `FlushWrites()`.
- **Atomic file writes**: Use temp file + `os.Rename` pattern (see `internal/connection/config.go` `SaveConnection`).
- **Version**: Set via goreleaser ldflags in `.goreleaser.yaml`. Fallback reads `debug.ReadBuildInfo()` in `internal/version/version.go`.

## Non-Interactive Shell Commands

```bash
cp -f source dest                 # force overwrite, never cp without -f
mv -f source dest                 # force overwrite
rm -f file                        # force remove
rm -rf directory                  # recursive force
```

## Issue Tracking with bd

This project uses `bd` (beads) for ALL issue tracking. Do NOT use markdown TODOs.

```bash
bd ready --json                   # find unblocked work
bd create "Title" --description="Details" -t task -p 2 --json
bd update <id> --claim --json     # claim atomically
bd close <id> --reason "Done" --json
bd dolt push                      # sync to remote
```

Workflow: `bd ready` → `bd update --claim` → implement → `bd close` → `bd dolt push`

Link discovered work: `bd create "Found bug" -p 1 --deps discovered-from:<parent-id> --json`

## Session Completion

Work is NOT complete until `git push` succeeds.

```bash
go test ./...                     # quality gate
git pull --rebase && bd dolt push && git push
git status                        # must show up-to-date
```

File issues for remaining work before ending. Never stop before pushing.

<!-- caliber:managed:pre-commit -->
## Before Committing

Run `caliber refresh` before creating git commits to keep docs in sync with code changes.
After it completes, stage any modified doc files before committing:

```bash
caliber refresh && git add CLAUDE.md .claude/ .cursor/ .github/copilot-instructions.md AGENTS.md CALIBER_LEARNINGS.md 2>/dev/null
```
<!-- /caliber:managed:pre-commit -->

<!-- caliber:managed:learnings -->
## Session Learnings

Read `CALIBER_LEARNINGS.md` for patterns and anti-patterns learned from previous sessions.
These are auto-extracted from real tool usage — treat them as project-specific rules.
<!-- /caliber:managed:learnings -->
