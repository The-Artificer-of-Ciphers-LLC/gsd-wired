---
phase: 14-connectivity
plan: "01"
subsystem: connectivity
tags: [go, mysql, connection-config, env-injection, tcp, sql-ping]

# Dependency graph
requires:
  - phase: 13-container-support
    provides: container runtime abstraction and .gsdw/ directory convention

provides:
  - internal/connection package with Config struct, LoadConnection, SaveConnection, CheckConnectivity, classifyTCPError, buildDSN, ActiveHostPort
  - graph.Client injects BEADS_DOLT_SERVER_HOST and BEADS_DOLT_SERVER_PORT when .gsdw/connection.json exists
  - go-sql-driver/mysql v1.9.3 dependency added to go.mod

affects: [14-02-connect-wizard, all graph client consumers]

# Tech tracking
tech-stack:
  added: [github.com/go-sql-driver/mysql v1.9.3, filippo.io/edwards25519 v1.1.0 (transitive)]
  patterns:
    - atomic write via temp+rename (same pattern as index.go Save)
    - nil-on-missing LoadConnection (os.IsNotExist check, returns nil nil not error)
    - two-phase connectivity check (TCP dial then SQL ping)
    - connConfig derived from filepath.Dir(beadsDir)/.gsdw, not cwd walk-up

key-files:
  created:
    - internal/connection/config.go
    - internal/connection/config_test.go
    - internal/graph/client_test.go
  modified:
    - internal/graph/client.go
    - internal/graph/testdata/fake_bd/main.go
    - go.mod
    - go.sum

key-decisions:
  - "loadConnConfig derives .gsdw from filepath.Dir(beadsDir)/.gsdw per Pitfall 4 — avoids cwd walk-up which would be wrong in test contexts"
  - "NewClientWithPath now loads connection config from disk for correctness — test injection still possible via direct c.connConfig assignment"
  - "FAKE_BD_ENV_CAPTURE_FILE added to fake_bd: captures full env map as JSON for hermetic env var verification without running real bd"
  - "url.QueryEscape used in buildDSN for user and password — safe encoding for special characters in MySQL DSN"
  - "classifyTCPError checks both 'no such host' and 'lookup' substrings — covers cross-platform DNS error message variations"

patterns-established:
  - "Connection package follows index.go atomic write pattern: marshal to .tmp then os.Rename to final path"
  - "Two-phase health check: TCP dial first (classifyTCPError for user-friendly messages), SQL ping second (Dolt-specific error)"
  - "connConfig is nil-safe everywhere: run() checks nil before calling ActiveHostPort(), silent skip on no config"

requirements-completed: [CONN-02, CONN-03, CONN-04]

# Metrics
duration: 4min
completed: 2026-03-22
---

# Phase 14 Plan 01: Connectivity Foundation Summary

**Connection config package (Config/Load/Save/CheckConnectivity) with two-phase TCP+SQL health check, and graph client env var injection of BEADS_DOLT_SERVER_HOST/PORT using go-sql-driver/mysql v1.9.3**

## Performance

- **Duration:** 4 min
- **Started:** 2026-03-22T04:20:35Z
- **Completed:** 2026-03-22T04:24:35Z
- **Tasks:** 2
- **Files modified:** 7

## Accomplishments
- Created `internal/connection` package with Config struct (D-13 JSON shape), atomic Save (temp+rename), graceful Load (nil on missing file, not error), and two-phase CheckConnectivity (TCP dial + SQL ping per D-06)
- Error classification in classifyTCPError provides context-aware messages for connection refused, DNS failure, and timeout with specific Fix guidance per D-07
- Extended graph.Client with connConfig field, loadConnConfig() helper, and run() env var injection (BEADS_DOLT_SERVER_HOST/PORT) per D-15/D-16/beads bug #2073
- Added FAKE_BD_ENV_CAPTURE_FILE to fake_bd for hermetic env var verification in tests
- All 13 connection package tests pass; all 55 graph package tests pass; 324 total tests pass

## Task Commits

Each task was committed atomically:

1. **Task 1: Create internal/connection package** - `5a3e1fc` (test+feat combined — tests and implementation committed together; all 13 tests verified passing)
2. **Task 2: Wire env var injection into graph client.go run()** - `1a46dd4` (feat — RED verified via build failure, GREEN via 55 passing tests)

## Files Created/Modified
- `internal/connection/config.go` - Config, LocalConfig, RemoteConfig structs; LoadConnection, SaveConnection, CheckConnectivity, classifyTCPError, buildDSN, ActiveHostPort
- `internal/connection/config_test.go` - 13 tests: round-trip, atomic save, load missing, host port defaults, TCP error classification, DSN building
- `internal/graph/client.go` - Added connConfig field, loadConnConfig() helper, updated all 4 constructors, env var injection in run()
- `internal/graph/client_test.go` - 4 new tests: InjectsConnEnvVars, NoConfigNoInjection, ConfigFromGsdwDir, ConfigMissing
- `internal/graph/testdata/fake_bd/main.go` - Added FAKE_BD_ENV_CAPTURE_FILE env var capture support
- `go.mod` - Added github.com/go-sql-driver/mysql v1.9.3
- `go.sum` - Updated with mysql driver and transitive dependency hashes

## Decisions Made
- `loadConnConfig` derives `.gsdw` from `filepath.Dir(beadsDir)/.gsdw` rather than cwd walk-up per Pitfall 4 in research — avoids incorrect path resolution in test contexts where cwd is not the project root
- `NewClientWithPath` now loads connection config from disk on construction — this is a behavioral change from the original stub, but necessary for `TestClientConnConfigFromGsdwDir` and for test injection via direct struct field assignment (connConfig is exported within package)
- `FAKE_BD_ENV_CAPTURE_FILE` writes the full `os.Environ()` map to a JSON file — all env vars captured, not filtered, so tests can assert both presence and absence
- `url.QueryEscape` used in `buildDSN` for user and password to safely encode special characters in the MySQL DSN format
- `classifyTCPError` checks both `"no such host"` and `"lookup"` for DNS errors — covers cross-platform error message variations

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
- One transient test failure in `TestPostToolUseBeadUpdate` (hook package) on first full suite run — passed on re-run, confirmed pre-existing flakiness unrelated to this plan's changes.

## Known Stubs

None - all exported functions are fully implemented with real behavior.

## Next Phase Readiness
- `internal/connection` package is the complete foundation for Plan 14-02 (connect wizard)
- `LoadConnection` and `SaveConnection` ready for the wizard to write connection.json
- `CheckConnectivity` ready for the `gsdw connect --test` subcommand in Plan 14-02
- `client.go` env injection tested and active — beads bug #2073 workaround in place

## Self-Check: PASSED

All created files verified on disk. All task commits verified in git history.

---
*Phase: 14-connectivity*
*Completed: 2026-03-22*
