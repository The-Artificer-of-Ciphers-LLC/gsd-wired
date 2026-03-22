---
phase: 12-setup-ux
plan: "01"
subsystem: deps-detection
tags: [deps, cli, check-deps, doctor, gopath-fallback]
dependency_graph:
  requires: []
  provides: [deps.CheckAll, gsdw-check-deps, gsdw-doctor]
  affects: [internal/cli/root.go]
tech_stack:
  added: [internal/deps package]
  patterns: [exec.LookPath-with-gopath-fallback, io.Writer-render-pattern, hermetic-PATH-isolation-in-tests]
key_files:
  created:
    - internal/deps/check.go
    - internal/deps/check_test.go
    - internal/cli/checkdeps.go
    - internal/cli/checkdeps_test.go
    - internal/cli/doctor.go
    - internal/cli/doctor_test.go
  modified:
    - internal/cli/root.go
decisions:
  - "[12-01] Tests use isolated temp-dir-only PATH to prevent real system binaries (bd at ~/.local/bin, docker) from interfering with hermetic tests"
  - "[12-01] lookInGoPath uses exec.LookPath('go') first so tests can inject a fake go binary via PATH — not exec.Command('go') directly"
  - "[12-01] renderDoctor delegates dep rendering inline rather than calling renderCheckDeps — avoids coupling the indentation style between the two commands"
  - "[12-01] checkContainerRuntime iterates docker-then-podman slice — clean extension point for Apple Container in Phase 13"
metrics:
  duration: "6 minutes"
  completed: "2026-03-22"
  tasks: 2
  files: 7
requirements: [SETUP-02, SETUP-03, SETUP-04]
---

# Phase 12 Plan 01: Dependency Detection and CLI Commands Summary

**One-liner:** `deps.CheckAll` with exec.LookPath + GOPATH/bin fallback powers `gsdw check-deps` ([OK]/[FAIL] + install help) and `gsdw doctor` (deps + project health, read-only).

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 (RED) | Failing tests for deps package | 53ae751 | internal/deps/check.go (stub), internal/deps/check_test.go |
| 1 (GREEN) | Implement deps.CheckAll | f0e41a6 | internal/deps/check.go |
| 2 (RED) | Failing tests for check-deps and doctor | 7f2bdeb | internal/cli/checkdeps.go (stub), internal/cli/doctor.go (stub), internal/cli/checkdeps_test.go, internal/cli/doctor_test.go |
| 2 (GREEN) | Implement check-deps and doctor commands | 44cfa44 | internal/cli/checkdeps.go, internal/cli/doctor.go, internal/cli/root.go |

## What Was Built

**`internal/deps` package** — new package providing dependency detection infrastructure:
- `CheckAll() CheckResult` — detects bd, dolt, Go, container runtime in order
- `lookInGoPath(binary)` — GOPATH/bin fallback for bd and dolt after `go install`
- `checkContainerRuntime()` — tries docker first, then podman (Apple Container deferred to Phase 13)
- `extractVersion()` — parses `name version X.Y.Z` output from any binary
- Install help strings for all four deps (go install / brew / curl)

**`gsdw check-deps`** — new subcommand:
- Human-readable `[OK]/[WARN]/[FAIL]` output per dep with version, path, and install help
- `--json` flag for machine-readable `{"allOK": bool, "deps": [...]}` output
- `renderCheckDeps(io.Writer, CheckResult)` extracted as pure function for testability

**`gsdw doctor`** — new subcommand:
- Calls `deps.CheckAll()` for the Dependencies section
- Finds `.beads/` via `findBeadsDir()` (walks up from cwd) and `.gsdw/` via `findGsdwDir()` (same pattern)
- Reports `[OK]` or `[WARN]` for each project directory with `gsdw init` hint
- Strictly read-only per D-09 — no file writes, no network calls
- `renderDoctor(io.Writer, CheckResult, beadsDir, gsdwDir)` extracted as pure function

**`internal/cli/root.go`** — `NewCheckDepsCmd()` and `NewDoctorCmd()` added to `AddCommand` chain.

## Test Coverage

- 8 tests in `internal/deps/check_test.go`: AllFound, BdMissing, GoPathFallback, InstallHelp, FourDeps, VersionParsing, ContainerRuntimeDockerThenPodman, LookInGoPath
- 14 tests in `internal/cli/checkdeps_test.go` + `doctor_test.go`: command registration, [OK]/[FAIL] rendering, install help, JSON output, .beads/.gsdw detection, read-only verification

## Verification

```
go test ./internal/deps/ ./internal/cli/ -count=1   # PASS: all 22 new tests pass
go build ./cmd/gsdw                                  # PASS: compiles clean
go vet ./internal/deps/ ./internal/cli/              # PASS: no issues
```

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Hermetic test PATH isolation**
- **Found during:** Task 1 (GREEN) and Task 2 (GREEN)
- **Issue:** Initial tests appended the real `origPath` to the temp dir, allowing real system binaries (`bd` at `~/.local/bin`, real `docker`) to be found, causing false positives
- **Fix:** Tests use `t.Setenv("PATH", dir)` (isolated to temp dir only) to ensure hermetic environments
- **Files modified:** `internal/deps/check_test.go`
- **Commit:** f0e41a6

**2. [Rule 1 - Bug] lookInGoPath uses LookPath("go") not exec.Command("go")**
- **Found during:** Task 1 (GREEN) — TestCheckAll_BdMissing failure analysis
- **Issue:** `exec.Command("go", "env", "GOPATH")` resolves the real system `go`, bypassing the fake `go` injected via PATH in tests, causing lookInGoPath to use real GOPATH instead of test gopath
- **Fix:** `lookInGoPath` first resolves `go` via `exec.LookPath("go")` so test PATH injection works correctly
- **Files modified:** `internal/deps/check.go`
- **Commit:** f0e41a6

## Known Stubs

None — all functionality is fully implemented. Apple Container runtime detection is intentionally deferred to Phase 13 (per plan spec), not a stub.

## Self-Check: PASSED
