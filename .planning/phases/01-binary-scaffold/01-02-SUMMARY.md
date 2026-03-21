---
phase: 01-binary-scaffold
plan: 02
subsystem: infra
tags: [go, json, mcp, hooks, plugin, testing, integration-tests]

# Dependency graph
requires:
  - phase: 01-01
    provides: "gsdw binary with serve, hook, bd, version subcommands and stdout-only slog"
provides:
  - ".claude-plugin/plugin.json with name gsd-wired, version 0.1.0, description, and author"
  - ".mcp.json registering gsdw serve as the MCP server at repo root"
  - "hooks/hooks.json registering all four hook events (SessionStart, PreCompact, PreToolUse, PostToolUse) at repo root"
  - "cmd/gsdw/manifest_test.go validating all three config files as correct JSON"
  - "cmd/gsdw/smoke_test.go proving binary builds, version format, stdout purity, and MCP initialize response"
affects: [03-mcp-tools, 04-hook-logic, 05-slash-commands]

# Tech tracking
tech-stack:
  added:
    - "bufio (stdlib) — used in TestServeRespondesToInitialize to read server response without race"
  patterns:
    - "StdoutPipe + bufio.Scanner in goroutine with channel — race-free pattern for reading subprocess stdout concurrently"
    - "repoRoot() helper walking to go.mod — makes tests portable regardless of test working directory"
    - "buildBinary(t) helper building to t.TempDir() — binary cached per test, temp dir cleanup automatic"

key-files:
  created:
    - ".claude-plugin/plugin.json — plugin manifest: name, version, description, author"
    - ".mcp.json — MCP server registration: gsdw serve"
    - "hooks/hooks.json — hook event registrations: all four Claude Code hook events"
    - "cmd/gsdw/manifest_test.go — config file validation: TestPluginManifestValid, TestMcpJsonValid, TestHooksJsonValid, TestHooksNotInsideClaudePlugin"
    - "cmd/gsdw/smoke_test.go — binary integration tests: TestBinaryBuilds, TestVersionOutput, TestHookStdoutPurity, TestHookInvalidEvent, TestServeRespondesToInitialize"
  modified:
    - ".gitignore — fixed: changed 'gsdw' to '/gsdw' to anchor to repo root (was matching cmd/gsdw directory)"

key-decisions:
  - "No minAppVersion in plugin.json — per D-12, no minimum Claude Code version pinned for maximum compatibility"
  - "Manifest grows per phase — only current deliverables registered (D-09); no forward stubs for Phase 3+ tools"
  - "StdoutPipe + bufio.Scanner over bytes.Buffer for subprocess stdout — avoids concurrent read/write race caught by -race"

patterns-established:
  - "Pattern: repoRoot() walker — portable test helper; works regardless of which directory go test runs from"
  - "Pattern: buildBinary(t) helper — builds to t.TempDir() so binary is cleaned up automatically; safe for parallel tests"
  - "Pattern: StdoutPipe+goroutine+channel for subprocess stdout — race-free reading; do NOT use bytes.Buffer as cmd.Stdout when polling concurrently"

requirements-completed: [INFRA-04]

# Metrics
duration: 9min
completed: 2026-03-21
---

# Phase 1 Plan 02: Plugin Registration and Integration Tests Summary

**Three Claude Code plugin registration JSON files (plugin.json, .mcp.json, hooks/hooks.json) plus 9-test integration suite proving binary builds, stdout purity, and MCP initialize handshake — all passing with -race**

## Performance

- **Duration:** 9 min
- **Started:** 2026-03-21T13:18:53Z
- **Completed:** 2026-03-21T13:28:00Z
- **Tasks:** 2
- **Files modified:** 5 created, 1 modified

## Accomplishments

- All three plugin registration files at correct locations: `.claude-plugin/plugin.json`, `.mcp.json` at repo root, `hooks/hooks.json` at repo root
- 9 integration tests all pass with `go test ./... -race`: manifest validation, binary smoke, stdout purity, MCP handshake
- `TestHookStdoutPurity` proves stdout produces only clean JSON with empty stderr at default log level
- `TestServeRespondesToInitialize` proves MCP stdio handshake works — server responds with `"name":"gsd-wired"` in serverInfo
- `TestHooksNotInsideClaudePlugin` guards against the most common plugin mistake (Pitfall 3 from RESEARCH.md)

## Task Commits

Each task was committed atomically:

1. **Task 1: Plugin manifest, MCP registration, and hooks registration files** - `cc6f4cf` (feat)
2. **Task 2: Integration tests for manifests, stdout purity, and binary smoke tests** - `1fc6afc` (feat)
3. **Task 2 race fix: Eliminate data race in TestServeRespondesToInitialize** - `5c52c51` (fix)

## Files Created/Modified

- `.claude-plugin/plugin.json` — plugin manifest: name "gsd-wired", version "0.1.0", description, author "The Artificer of Ciphers LLC"; no minAppVersion
- `.mcp.json` — MCP server registration at repo root: `{"mcpServers":{"gsd-wired":{"command":"gsdw","args":["serve"]}}}`
- `hooks/hooks.json` — hook event registrations at repo root: SessionStart, PreCompact, PreToolUse, PostToolUse each pointing to `gsdw hook <Event>`
- `cmd/gsdw/manifest_test.go` — 4 tests: TestPluginManifestValid, TestMcpJsonValid, TestHooksJsonValid, TestHooksNotInsideClaudePlugin
- `cmd/gsdw/smoke_test.go` — 5 tests: TestBinaryBuilds, TestVersionOutput, TestHookStdoutPurity, TestHookInvalidEvent, TestServeRespondesToInitialize
- `.gitignore` — bug fix: `gsdw` → `/gsdw` to anchor the binary exclusion to repo root only

## Decisions Made

- No `minAppVersion` in plugin.json — per D-12, pinning a minimum Claude Code version reduces compatibility without benefit at this stage
- Plugin manifest strictly current-phase only — no forward declarations for Phase 3 MCP tools or Phase 5 slash commands (per D-09)
- `StdoutPipe` + `bufio.Scanner` in goroutine with channel for `TestServeRespondesToInitialize` — the race detector caught a write/read race when using `bytes.Buffer` as `cmd.Stdout` while polling concurrently; StdoutPipe is the correct pattern

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed .gitignore anchoring for gsdw binary**
- **Found during:** Task 2 commit staging
- **Issue:** `.gitignore` had `gsdw` (unanchored), which Git matched against the `cmd/gsdw` directory path component, causing `git add cmd/gsdw/manifest_test.go` to fail with "ignored by .gitignore"
- **Fix:** Changed `gsdw` to `/gsdw` in `.gitignore` to anchor the pattern to the repo root only
- **Files modified:** `.gitignore`
- **Verification:** `git add cmd/gsdw/manifest_test.go cmd/gsdw/smoke_test.go` succeeded after fix
- **Committed in:** `1fc6afc` (Task 2 commit)

**2. [Rule 1 - Bug] Fixed data race in TestServeRespondesToInitialize**
- **Found during:** Overall verification run with `-race` flag
- **Issue:** Test used `bytes.Buffer` as `cmd.Stdout`, then polled `stdoutBuf.Bytes()` in a loop while the exec goroutine was concurrently writing to the same buffer — a data race detected by Go's race detector
- **Fix:** Switched to `cmd.StdoutPipe()` + `bufio.Scanner` in a goroutine sending to a channel; test goroutine selects on the channel with timeout — eliminates the concurrent access entirely
- **Files modified:** `cmd/gsdw/smoke_test.go`
- **Verification:** `go test ./... -race -count=1 -timeout=60s` exits 0
- **Committed in:** `5c52c51` (fix commit)

---

**Total deviations:** 2 auto-fixed (both Rule 1 — bugs)
**Impact on plan:** Both fixes necessary for correctness and race-free test execution. No scope creep.

## Issues Encountered

None beyond the two auto-fixed bugs documented above.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- Phase 1 complete: all INFRA-01, INFRA-04, INFRA-09 requirements satisfied
- Binary scaffold proven end-to-end: version, serve (MCP), hook (all 4 events), bd passthrough
- Plugin registration files at correct locations — ready for Claude Code to discover
- 19 tests pass with `-race` across all packages
- Phase 2 (MCP tools) can add tool registrations to plugin.json and implement real tool handlers in Phase 3

## Self-Check: PASSED

All 5 created files verified present. All 3 task commits (cc6f4cf, 1fc6afc, 5c52c51) verified in git log.

---
*Phase: 01-binary-scaffold*
*Completed: 2026-03-21*
