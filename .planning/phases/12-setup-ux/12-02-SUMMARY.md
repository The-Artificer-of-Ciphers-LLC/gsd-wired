---
phase: 12-setup-ux
plan: "02"
subsystem: setup-wizard
tags: [setup, cli, interactive, install-wizard, deps]
dependency_graph:
  requires: [deps.CheckAll, gsdw-check-deps]
  provides: [gsdw-setup]
  affects: [internal/cli/root.go]
tech_stack:
  added: []
  patterns: [bufio.NewReader-stdin-injection, io.Reader-io.Writer-testable-core, checkFn-injection-pattern]
key_files:
  created:
    - internal/cli/setup.go
    - internal/cli/setup_test.go
  modified:
    - internal/cli/root.go
decisions:
  - "[12-02] depInstallOptions map keyed on binary name (bd/dolt/go/docker) — clean lookup for per-dep install methods without switch statements"
  - "[12-02] brewAvailable passed as bool parameter to runSetup — testable without PATH manipulation, complements hermetic test pattern from 12-01"
  - "[12-02] numbered menu options built dynamically (brew first if available, then go install, then download) — order is deterministic, tests for [1] brew vs [1] go install work cleanly"
  - "[12-02] printNextSteps extracted as separate function — both all-OK and missing-dep paths share the same guidance block"
metrics:
  duration: "2 minutes"
  completed: "2026-03-22"
  tasks: 1
  files: 3
requirements: [SETUP-01, SETUP-05]
---

# Phase 12 Plan 02: Setup Interactive Wizard Summary

**One-liner:** `gsdw setup` interactive wizard runs check-deps first, offers numbered brew/go install/download menu per missing dep (never auto-installs), re-verifies after install, and guides to next steps.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 (RED) | Failing tests for gsdw setup wizard | de0576a | internal/cli/setup_test.go, internal/cli/setup.go (stub) |
| 1 (GREEN) | Implement gsdw setup interactive wizard | db0a184 | internal/cli/setup.go, internal/cli/root.go |

## What Was Built

**`internal/cli/setup.go`** — new file implementing the setup wizard:

- `NewSetupCmd() *cobra.Command` — cobra command registered as `gsdw setup`
- `runSetup(in io.Reader, out io.Writer, checkFn func() deps.CheckResult, brewAvailable bool) error` — testable core with injected dependencies
- `offerInstall(reader, out, dep, brewAvailable)` — renders numbered install menu for one missing dep, reads choice, prints command (never runs it)
- `depInstallOptions` map — install method details (brew/go install/download) keyed by binary name for bd, dolt, Go, container runtime
- `printNextSteps(out)` — prints Phase 13/14 guidance and `gsdw doctor` reminder

**Wizard flow (4 phases):**
1. Runs `deps.CheckAll()` and renders `[OK]/[FAIL]` output — user sees current state first
2. For each `StatusFail` dep: shows numbered install options (brew option only if brew on PATH), reads user choice, prints the command to run
3. Prompts "Press Enter after installing to re-check..." then re-runs `deps.CheckAll()`
4. Prints next steps: container runtime, connection config, health check

**`internal/cli/root.go`** — `NewSetupCmd()` added to `AddCommand` chain.

**`internal/cli/setup_test.go`** — 14 tests covering:
- All-OK path exits cleanly with satisfaction message
- Check-deps output appears before satisfaction message
- Missing dep shows install menu with skip option
- Brew option appears/hides based on `brewAvailable` flag
- User selecting go install/brew install sees the command in output
- No auto-install behavior (no "installed successfully" / "Installing" messages)
- checkFn called at least twice (initial check + re-verify)
- Next steps section present in both all-OK and missing-dep paths
- Multiple missing deps each get their own install menu
- Container setup mention in next steps guidance

## Verification

```
go test ./internal/deps/ ./internal/cli/ -count=1   # PASS: all tests pass (incl. 14 new)
go build ./cmd/gsdw                                  # PASS: compiles clean
go vet ./...                                         # PASS: no issues
```

## Deviations from Plan

None — plan executed exactly as written.

## Known Stubs

None — all functionality is fully implemented. Container runtime (`gsdw container setup`) and connection config (`gsdw connect`) are referenced in next steps guidance as future commands from Phases 13/14 per plan spec, not stubs in this file.

## Self-Check: PASSED
