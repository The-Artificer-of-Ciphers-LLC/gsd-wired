---
phase: 14-connectivity
plan: "02"
subsystem: connectivity
tags: [go, cobra, wizard, bufio, interactive-cli, connection-config, doctor]

# Dependency graph
requires:
  - phase: 14-01
    provides: connection.Config, LoadConnection, SaveConnection, CheckConnectivity
  - phase: 13-container-support
    provides: container.DetectRuntime, Runtime interface

provides:
  - internal/cli/connect.go with connectOpts struct, NewConnectCmd, runConnect wizard
  - internal/cli/doctor.go extended with Connection section (Mode/Address/health status)
  - NewConnectCmd registered in root.go AddCommand chain

affects: [gsdw doctor output, gsdw connect wizard, all users setting up connectivity]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - connectOpts dependency injection (same hermetic test pattern as startOpts, setupOpts)
    - bufio.NewReader(in) for interactive prompts — consistent with setup.go wizard pattern
    - renderDoctor extended with optional connection params — nil-safe, backward compatible

key-files:
  created:
    - internal/cli/connect.go
    - internal/cli/connect_test.go
  modified:
    - internal/cli/doctor.go
    - internal/cli/doctor_test.go
    - internal/cli/root.go

key-decisions:
  - "connectOpts injects detectServerFn and healthCheckFn separately — detectServerFn uses no auth (2s timeout) for auto-detect; healthCheckFn uses GSDW_DB_PASSWORD for authenticated health check"
  - "readLine helper wraps bufio ReadString+TrimSpace — avoids repeated pattern in wizard branches"
  - "renderDoctor extended via new parameters not a separate function — keeps single render call site in NewDoctorCmd RunE, nil-safe for tests that don't care about connection"
  - "defaultStartContainer uses container.DetectRuntime with empty DetectOpts — real defaults, not injectable, because this is the production path only (wizard tests inject startContainerFn)"

# Metrics
duration: 10min
completed: 2026-03-22
---

# Phase 14 Plan 02: Connect Wizard and Doctor Integration Summary

**gsdw connect interactive wizard (auto-detect, container start, remote config, fallback) with gsdw doctor Connection section showing mode/address/health — completes Phase 14 connectivity story end to end**

## Performance

- **Duration:** ~10 min
- **Started:** 2026-03-22T04:24:35Z
- **Completed:** 2026-03-22T04:34:28Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments

- Created `internal/cli/connect.go` with full D-01 through D-05 wizard flow: auto-detect local server, three-choice menu (container/remote/cancel), remote config collection with default port, GSDW_DB_PASSWORD env var integration
- Created `internal/cli/connect_test.go` with 12 hermetic tests covering all wizard paths via injected stubs (no real network or container calls)
- Extended `internal/cli/doctor.go`: renderDoctor now accepts `connCfg *connection.Config` and `connHealthErr error`; NewDoctorCmd RunE loads connection config and runs CheckConnectivity before rendering the new Connection section
- Added 4 new TestRenderDoctor_Connection* tests and updated all existing renderDoctor call sites with two new nil params
- Wired `NewConnectCmd()` into root.go AddCommand chain
- All 87 CLI tests pass; full suite 11/11 packages green; binary compiles clean

## Task Commits

1. **Task 1: Create gsdw connect wizard** - `7195a46` (feat — 12 tests RED confirmed, all 12 GREEN)
2. **Task 2: Extend doctor with Connection section and wire NewConnectCmd** - `9dcabf0` (feat — 4 new tests RED confirmed, all 87 CLI tests GREEN)

## Files Created/Modified

- `internal/cli/connect.go` - connectOpts struct, NewConnectCmd, runConnect, handleStartContainer, handleConfigureRemote, handleRemoteFallback, doSaveLocalConfig, doSaveRemoteConfig, defaultStartContainer, readLine
- `internal/cli/connect_test.go` - 12 tests: TestConnectAutoDetectFound, TestConnectAutoDetectFound_Decline, TestConnectNoServer_StartContainer, TestConnectNoServer_ConfigureRemote, TestConnectNoServer_Cancel, TestConnectExistingConfig_KeepCurrent, TestConnectExistingConfig_Reconfigure, TestConnectRemoteFallback_UserConfirms, TestConnectRemoteFallback_UserDeclines, TestConnectFallback_SessionOnly, TestConnectRemoteDefaultPort, TestConnectNoGsdwDir
- `internal/cli/doctor.go` - renderDoctor extended with connCfg/connHealthErr params and Connection section; NewDoctorCmd RunE loads and checks connection config
- `internal/cli/doctor_test.go` - 4 new connection tests, all existing calls updated to pass nil,nil for new params
- `internal/cli/root.go` - NewConnectCmd() added to AddCommand chain

## Decisions Made

- `connectOpts` injects `detectServerFn` and `healthCheckFn` as separate functions — detectServerFn uses no auth (2s timeout) for auto-detect scan; healthCheckFn uses GSDW_DB_PASSWORD for authenticated checks on existing config and remote mode
- `readLine` helper wraps `bufio.Reader.ReadString('\n')` + TrimSpace — eliminates repeated pattern across wizard branches
- `renderDoctor` extended via new optional parameters rather than a new function — keeps single render call site clean; nil-safe so all 8 existing tests continue passing unchanged (just updated call sites)
- `defaultStartContainer` uses `container.DetectRuntime` with empty `DetectOpts` (real defaults, not injectable) — wizard tests inject `startContainerFn` directly so the production path doesn't need test injection

## Deviations from Plan

None - plan executed exactly as written.

## Known Stubs

None — all exported functions are fully implemented. `defaultStartContainer` is the real production container start; `NewConnectCmd` wires all real dependencies.

## Self-Check: PASSED
