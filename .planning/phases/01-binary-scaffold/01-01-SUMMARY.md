---
phase: 01-binary-scaffold
plan: 01
subsystem: infra
tags: [go, cobra, mcp, slog, stdio, hook, plugin]

# Dependency graph
requires: []
provides:
  - "gsdw binary compiling and running all four subcommands: serve, hook, bd, version"
  - "MCP stdio server responding to initialize via official go-sdk"
  - "Hook dispatcher parsing and validating stdin JSON, emitting JSON to stdout"
  - "bd CLI passthrough with DisableFlagParsing"
  - "slog dual-format handler always writing to stderr, never stdout"
  - "Cobra root with SilenceUsage/SilenceErrors and persistent log flags"
affects: [02-plugin-manifest, 03-mcp-tools, 04-hook-logic, 05-slash-commands]

# Tech tracking
tech-stack:
  added:
    - "github.com/spf13/cobra v1.10.2 — CLI framework"
    - "github.com/modelcontextprotocol/go-sdk v1.4.1 — official MCP SDK"
    - "log/slog (stdlib) — structured logging, dual JSON/text format"
    - "runtime/debug (stdlib) — VCS metadata for version string"
    - "encoding/json (stdlib) — hook stdin/stdout JSON encode/decode"
    - "os/exec (stdlib) — bd CLI passthrough"
  patterns:
    - "Cobra multi-mode binary: serve (long-lived), hook (short-lived), bd (passthrough), version"
    - "stderr-only slog: handler writer always os.Stderr, set before Execute() in main()"
    - "SilenceUsage+SilenceErrors on root: prevents Cobra writing to stdout on errors"
    - "Injected reader/writer in Dispatch(): testable without os.Stdin/os.Stdout"
    - "TDD: failing tests written first, implementation follows, -race verified"

key-files:
  created:
    - "cmd/gsdw/main.go — binary entry point, pre-logger stderr init, os.Exit(cli.Execute())"
    - "internal/cli/root.go — Cobra root, SilenceUsage/SilenceErrors, log flags, Execute()"
    - "internal/cli/version.go — version subcommand, prints version.String() to stdout"
    - "internal/cli/serve.go — serve subcommand, calls mcp.Serve(ctx)"
    - "internal/cli/hook.go — hook subcommand, cobra.ExactArgs(1), calls hook.Dispatch()"
    - "internal/cli/bd.go — bd passthrough, DisableFlagParsing, exec.CommandContext"
    - "internal/version/version.go — version.String() via debug.ReadBuildInfo() vcs.revision"
    - "internal/logging/logging.go — logging.Init() dual-format slog handler to os.Stderr"
    - "internal/hook/events.go — HookInput/HookOutput structs, event constants, IsValidEvent()"
    - "internal/hook/dispatcher.go — Dispatch() with injected reader/writer, event validation"
    - "internal/mcp/server.go — mcp.NewServer + mcp.StdioTransport, gsd-wired name"
    - "internal/version/version_test.go — format and fallback version tests"
    - "internal/logging/logging_test.go — stderr-only, no stdout, JSON format tests"
    - "internal/hook/dispatcher_test.go — dispatch, invalid JSON, mismatch, purity tests"
    - "internal/mcp/server_test.go — TestServeRespondsToInitialize via subprocess"
    - "go.mod — module github.com/The-Artificer-of-Ciphers-LLC/gsd-wired"
    - ".gitignore — excludes gsdw binary"
  modified:
    - "internal/cli/root.go — added NewServeCmd, NewHookCmd, NewBdCmd in Task 2"

key-decisions:
  - "go-sdk v1.4.1 used as official MCP library (not mark3labs/mcp-go) — authoritative, Google co-maintained"
  - "runtime/debug.ReadBuildInfo() for version hash — works with go install, no ldflags/Makefile required"
  - "Injected io.Reader/io.Writer in hook.Dispatch() — makes dispatcher testable without os pipe mocking"
  - "Pre-logger slog default in main() before Execute() — prevents stdout pollution before PersistentPreRunE runs"
  - "go mod tidy required after initial go get — go-sdk transitive deps not fully resolved by single go get"

patterns-established:
  - "Pattern: stderr-only logging — all slog handlers use os.Stderr; never os.Stdout"
  - "Pattern: Cobra stdout discipline — SilenceUsage+SilenceErrors on root, errors logged via slog.Error"
  - "Pattern: TDD with injected I/O — testable functions take io.Reader/io.Writer, not global os.Stdin/os.Stdout"
  - "Pattern: MCP serve is stdout-exclusive — serve subcommand writes nothing to stdout; SDK owns it entirely"

requirements-completed: [INFRA-01, INFRA-09]

# Metrics
duration: 5min
completed: 2026-03-21
---

# Phase 1 Plan 01: Binary Scaffold Summary

**Single Go binary (gsdw) with Cobra CLI, MCP stdio server via official go-sdk, hook dispatcher with stdin JSON validation, bd passthrough, and slog stderr-only logging — all four subcommands functional at stub level**

## Performance

- **Duration:** 5 min
- **Started:** 2026-03-21T13:06:59Z
- **Completed:** 2026-03-21T13:12:00Z
- **Tasks:** 2 (TDD — both had RED/GREEN phases)
- **Files modified:** 17 created, 1 modified

## Accomplishments

- gsdw binary compiles and all four subcommands (version, serve, hook, bd) work end-to-end
- MCP stdio server responds to `initialize` with valid JSON-RPC 2.0 response containing `"name":"gsd-wired"`
- Hook dispatcher validates stdin JSON, checks event name match, emits `{}` no-op response — stdout purity verified
- slog configured with stderr-only handlers in both text and JSON formats — stdout never receives log output
- 11 tests pass with `-race` including subprocess integration test for MCP serve

## Task Commits

Each task was committed atomically:

1. **Task 1: Go module, Cobra root, version, and logging foundation** - `e3c4302` (feat)
2. **Task 2: MCP serve, hook dispatcher, and bd passthrough subcommands** - `fb20322` (feat)

## Files Created/Modified

- `cmd/gsdw/main.go` — binary entry point; pre-logger stderr slog init before Execute()
- `internal/cli/root.go` — Cobra root with SilenceUsage, SilenceErrors, log flags
- `internal/cli/version.go` — prints `version.String()` to cmd.OutOrStdout()
- `internal/cli/serve.go` — calls mcp.Serve(ctx); no stdout output
- `internal/cli/hook.go` — cobra.ExactArgs(1), delegates to hook.Dispatch, exit 2 on error
- `internal/cli/bd.go` — DisableFlagParsing, exec.CommandContext passthrough to bd
- `internal/version/version.go` — `const Version = "0.1.0"`, reads vcs.revision via debug.ReadBuildInfo()
- `internal/logging/logging.go` — Init() switches level/format; handler always writes to os.Stderr
- `internal/hook/events.go` — HookInput/HookOutput structs, event constants, IsValidEvent()
- `internal/hook/dispatcher.go` — Dispatch(event, stdin, stdout) with injected I/O
- `internal/mcp/server.go` — mcp.NewServer("gsd-wired") + server.Run(ctx, &mcp.StdioTransport{})
- `go.mod` — module + cobra + go-sdk + transitive deps
- `.gitignore` — excludes gsdw binary

## Decisions Made

- Used `runtime/debug.ReadBuildInfo()` for version hash instead of `-ldflags` — works with `go install` without a Makefile
- Used injected `io.Reader`/`io.Writer` in `hook.Dispatch()` for testability — avoids needing `os.Pipe` mocking in tests
- Added pre-logger stderr slog default in `main()` before `root.Execute()` — critical to prevent stdout pollution if anything logs before `PersistentPreRunE` runs
- `go mod tidy` required after initial `go get` — go-sdk v1.4.1 has transitive dependencies (google/jsonschema-go, segmentio/encoding, yosida95/uritemplate, oauth2) not fully resolved by single `go get`

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Added .gitignore for gsdw binary**
- **Found during:** Task 2 commit staging
- **Issue:** `gsdw` binary appeared as untracked file after build; would pollute the repo if committed
- **Fix:** Created `.gitignore` excluding the `gsdw` binary and `gsdw-*` test binaries
- **Files modified:** `.gitignore`
- **Verification:** `git status` no longer shows `gsdw` as untracked
- **Committed in:** `fb20322` (Task 2 commit)

**2. [Rule 3 - Blocking] Additional go get calls needed for transitive MCP SDK deps**
- **Found during:** Task 2 build after adding internal/mcp/server.go import
- **Issue:** `go get github.com/modelcontextprotocol/go-sdk@v1.4.1` did not resolve all transitive deps; build failed with missing go.sum entries
- **Fix:** Ran `go get github.com/modelcontextprotocol/go-sdk/mcp@v1.4.1` and `go mod tidy` to pull all transitive dependencies
- **Files modified:** `go.mod`, `go.sum`
- **Verification:** `go build ./cmd/gsdw` exits 0
- **Committed in:** `fb20322` (Task 2 commit)

---

**Total deviations:** 2 auto-fixed (both Rule 3 — blocking issues)
**Impact on plan:** Both fixes necessary for build correctness and repo hygiene. No scope creep.

## Issues Encountered

None beyond the two auto-fixed blocking issues documented above.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- Binary scaffold complete: `gsdw version`, `gsdw serve`, `gsdw hook SessionStart`, `gsdw bd --help` all work
- Stdout discipline established and tested — safe foundation for MCP serve mode
- All patterns established (stderr logging, Cobra silence, injected I/O) are ready for Phase 2 to build on
- Phase 2 (plugin manifest) can wire `.claude-plugin/plugin.json`, `.mcp.json`, and `hooks/hooks.json` against this binary

## Self-Check: PASSED

All 12 source files verified present. Both task commits (e3c4302, fb20322) verified in git log.

---
*Phase: 01-binary-scaffold*
*Completed: 2026-03-21*
