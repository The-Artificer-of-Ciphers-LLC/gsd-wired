---
phase: 11-distribution
plan: "01"
subsystem: infra
tags: [goreleaser, homebrew, docker, distroless, notarization, ldflags, version]

# Dependency graph
requires: []
provides:
  - ldflags-injected version package with BuildInfo struct and GetInfo() function
  - gsdw version --json flag for machine-readable version output
  - .goreleaser.yaml with cross-platform builds, notarization, homebrew_casks, docker images
  - Dockerfile.container using distroless/static-debian12:nonroot
  - /dist/ gitignore entry
affects: [12-installer, 13-container, 14-cicd]

# Tech tracking
tech-stack:
  added: [goreleaser v2, gcr.io/distroless/static-debian12:nonroot]
  patterns: [ldflags injection for release builds with ReadBuildInfo fallback, goreleaser yaml v2 config pattern]

key-files:
  created:
    - .goreleaser.yaml
    - Dockerfile.container
    - internal/cli/version_test.go
  modified:
    - internal/version/version.go
    - internal/version/version_test.go
    - internal/cli/version.go
    - .gitignore

key-decisions:
  - "Used dockers (stable) instead of dockers_v2 (alpha) — goreleaser not installed locally so goreleaser check could not be run; stable approach is safer"
  - "signs section retained with GPG config — can be removed if GPG not available at release time"
  - "ldflags vars are package-level (not const) to allow test override via direct assignment"

patterns-established:
  - "ldflags-first, ReadBuildInfo fallback pattern for version info — enables both goreleaser releases and go install"
  - "BuildInfo.String() and BuildInfo.JSON() as value methods — consistent JSON output shape for tooling"

requirements-completed: [DIST-01, DIST-03, DIST-04, DIST-05]

# Metrics
duration: 3min
completed: "2026-03-22"
---

# Phase 11 Plan 01: Distribution Infrastructure Foundation Summary

**GoReleaser v2 config with Apple notarization, homebrew_casks postinstall, multi-arch distroless container, and ldflags-injected version package with --json CLI flag**

## Performance

- **Duration:** ~3 min
- **Started:** 2026-03-22T02:00:58Z
- **Completed:** 2026-03-22T02:03:30Z
- **Tasks:** 2 of 3 complete (Task 3 is checkpoint:human-verify pending external setup)
- **Files modified:** 7

## Accomplishments

- Extended `internal/version/version.go` with `BuildInfo` struct, `GetInfo()` (ldflags-first + ReadBuildInfo fallback), `BuildInfo.String()` and `BuildInfo.JSON()` methods
- Added `--json` flag to `gsdw version` subcommand outputting structured JSON with version/commit/date/goVersion/platform keys
- Created `.goreleaser.yaml` with complete release pipeline: darwin/linux amd64/arm64 builds, Apple notarization (wait: true per D-09), homebrew_casks (gsdw-cc + postinstall xattr quarantine removal per D-20), docker + docker_manifests for ghcr.io multi-arch images
- Created `Dockerfile.container` using `gcr.io/distroless/static-debian12:nonroot` base per D-18
- Updated `.gitignore` with `/dist/` pattern for goreleaser artifacts

## Task Commits

Each task was committed atomically:

1. **Task 1: Extend version package with ldflags injection and JSON output** - `e4b457c` (feat)
2. **Task 2: Create GoReleaser config, Dockerfile, and update .gitignore** - `8363e88` (chore)
3. **Task 3: Create Homebrew tap repo and verify prerequisites** - PENDING (checkpoint:human-verify)

## Files Created/Modified

- `internal/version/version.go` - Added BuildInfo struct, GetInfo(), ldflags vars, JSON/String methods
- `internal/version/version_test.go` - Added tests for GetInfo fallback, BuildInfo.String, JSON, ldflags override
- `internal/cli/version.go` - Added --json flag, calls version.GetInfo().JSON() or .String()
- `internal/cli/version_test.go` - New file: tests for --json flag and human-readable output
- `.goreleaser.yaml` - New file: full release pipeline config
- `Dockerfile.container` - New file: distroless container image definition
- `.gitignore` - Added /dist/ pattern

## Decisions Made

- **dockers vs dockers_v2:** Used stable `dockers` section instead of alpha `dockers_v2`. The plan stated to fall back to `dockers + docker_manifests` if goreleaser check fails on dockers_v2. Since goreleaser is not installed locally, the stable approach was taken proactively.
- **ldflags vars as package-level vars (not unexported):** Package-level vars allow direct assignment in tests (TestLdflagsOverride) without needing build flags or reflection.
- **signs section retained:** GPG signing config included as specified. Can be removed if GPG isn't configured at release time.

## Deviations from Plan

None — plan executed exactly as written, with one proactive choice (dockers over dockers_v2) that was explicitly listed as the recommended fallback.

## Issues Encountered

- `goreleaser` binary not installed locally — `goreleaser check` could not be run. Used stable `dockers` section as the plan's documented fallback. The config can be validated in CI with pinned goreleaser v2.14.3.

## User Setup Required

Task 3 (checkpoint:human-verify) requires:
1. Create `The-Artificer-of-Ciphers-LLC/homebrew-gsdw` GitHub repo (public, with README)
2. Create fine-grained PAT `goreleaser-homebrew-tap` with contents:write on that repo
3. Add `HOMEBREW_TAP_TOKEN` secret to `gsd-wired` repo settings

## Next Phase Readiness

- Version package and GoReleaser config ready for CI workflow in Phase 14
- Homebrew tap repo setup (Task 3) required before any release can push cask updates
- dockers section targets ghcr.io — GitHub Container Registry write access (GHCR_TOKEN) needed at release time

---
*Phase: 11-distribution*
*Completed: 2026-03-22 (partial — Task 3 checkpoint pending)*

## Self-Check: PASSED

Files verified present:
- FOUND: /Users/trekkie/projects/gsd-wired/internal/version/version.go
- FOUND: /Users/trekkie/projects/gsd-wired/internal/cli/version.go
- FOUND: /Users/trekkie/projects/gsd-wired/.goreleaser.yaml
- FOUND: /Users/trekkie/projects/gsd-wired/Dockerfile.container
- FOUND: /Users/trekkie/projects/gsd-wired/.gitignore

Commits verified:
- e4b457c: feat(11-01): extend version package with ldflags injection and JSON output
- 8363e88: chore(11-01): create GoReleaser config, container Dockerfile, and update .gitignore
