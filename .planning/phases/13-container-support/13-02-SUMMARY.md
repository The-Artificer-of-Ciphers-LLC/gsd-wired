---
phase: 13-container-support
plan: "02"
subsystem: cli/container
tags: [container, docker, podman, apple-container, cli, cobra, tdd]
dependency_graph:
  requires: [internal/container/runtime.go, internal/container/compose.go]
  provides: [internal/cli/container.go, internal/cli/root.go]
  affects: [cmd/gsdw]
tech_stack:
  added: [net (port availability check)]
  patterns: [Cobra subcommand, injectable opts struct, TDD RED/GREEN]
key_files:
  created:
    - internal/cli/container.go
    - internal/cli/container_test.go
  modified:
    - internal/cli/root.go
decisions:
  - "startOpts/stopOpts structs inject all dependencies (detectFn, composeFn, execFn, checkPort, statFn) — same hermetic test pattern as 12-01/13-01"
  - "composeFn only called for docker/podman runtimes — apple-container skips compose fragment"
  - "defaultCheckPort uses net.Listen on 127.0.0.1:{port}, closes listener on success — port free returns nil"
  - "beads-dir pre-flight uses injected statFn (default: os.Stat) — clean test isolation without filesystem setup"
metrics:
  duration: "17 min"
  completed: "2026-03-22T03:47:00Z"
  tasks: 2
  files: 3
---

# Phase 13 Plan 02: Container CLI Subcommands Summary

`gsdw container start/stop` subcommands with runtime detection, pre-flight checks (beads dir, port availability), compose fragment generation for Docker/Podman, and full dependency injection for hermetic testing.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 (RED) | Container tests | ea16363 | internal/cli/container_test.go |
| 1 (GREEN) | Container implementation | 24cfce0 | internal/cli/container.go |
| 2 | Wire container into root | afc3923 | internal/cli/root.go |

## What Was Built

### internal/cli/container.go

Three exported commands and two testable core functions:

- **NewContainerCmd()**: Parent "container" command with start/stop subcommands.

- **NewContainerStartCmd()**: Flags: `--force` (overwrite compose fragment), `--port` (default 3307), `--beads-dir` (default .beads/dolt). Delegates to `runContainerStart`.

- **NewContainerStopCmd()**: Delegates to `runContainerStop`.

- **runContainerStart logic**:
  1. Detect runtime via `detectFn` — on failure, print install guidance for Docker/Podman/Apple Container and return error
  2. Print "Using runtime: {name}"
  3. Pre-flight: check `.beads/dolt/` exists via `statFn` — error with "run `bd init --backend dolt` first" (Pitfall 2)
  4. Pre-flight: check port availability via `checkPort` — error with "Port {port} already in use" (Pitfall 3)
  5. Resolve absolute path for beads dir
  6. For Docker/Podman: call `composeFn` to write gsdw.compose.yaml; Apple Container skips this step
  7. Print and execute start command via `execFn`
  8. Print success: port and data persistence location

- **runContainerStop logic**: Detects runtime, builds stop args, prints and executes stop command.

- **defaultCheckPort**: `net.Listen("tcp", "127.0.0.1:{port}")`, close on success (port free), error if occupied.

### internal/cli/root.go

Added `NewContainerCmd()` to the `root.AddCommand(...)` call — `gsdw container` now available in the binary.

## Test Coverage

- 17 tests in internal/cli (container_test.go)
- All pass: `go test ./internal/cli/ -run "TestContainer|TestRunContainer" -count=1`
- Full binary build passes: `go build ./cmd/gsdw`
- `go vet ./...` clean

Note: `TestPostToolUseBeadUpdate` in internal/hook is a pre-existing failure (bd binary not on PATH in test environment) — unrelated to this plan.

## Deviations from Plan

None — plan executed exactly as written.

## Known Stubs

None. Container start/stop fully implemented with all pre-flight checks, compose generation, and exec invocation.

## Self-Check: PASSED

Files verified:
- internal/cli/container.go: EXISTS
- internal/cli/container_test.go: EXISTS
- internal/cli/root.go: MODIFIED (NewContainerCmd added)

Commits verified:
- ea16363 (test RED container tests)
- 24cfce0 (feat GREEN container implementation)
- afc3923 (feat wire container into root)
