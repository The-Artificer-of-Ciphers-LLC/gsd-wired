---
status: awaiting_human_verify
trigger: "docker build fails with unknown flag: --provenance"
created: 2026-03-22T00:00:00Z
updated: 2026-03-22T00:00:00Z
---

## Current Focus

hypothesis: GoReleaser v2 defaults to `use: buildx` for docker builds, which invokes `docker buildx build` with `--provenance`. The local docker CLI has no buildx plugin, and the Docker daemon is not running (no socket). For snapshot/local dev the fix is to add `use: docker` to each dockers entry to force the legacy builder, and add `skip: '{{ if .IsSnapshot }}true{{ end }}'` or use `--skip=docker` in the snapshot Make target.
test: Read goreleaser docs on `dockers.use` field and verify `use: docker` suppresses --provenance flag.
expecting: Adding `use: docker` forces plain `docker build` without --provenance; adding `--skip=docker` to snapshot target avoids docker entirely during local dev.
next_action: Decide approach (skip docker in snapshot vs use legacy builder), implement, verify config parses cleanly.

## Symptoms

expected: `make release-mac-snapshot` should complete all steps including docker image builds
actual: Signing and notarization pass, but docker build fails with "unknown flag: --provenance"
errors: |
  docker build failed: failed to build ghcr.io/the-artificer-of-ciphers-llc/gsdw:1.1.1-SNAPSHOT-65c8746-amd64: exit status 125
  DEPRECATED: The legacy builder is deprecated and will be removed in a future release.
  Install the buildx component to build images with BuildKit: https://docs.docker.com/go/buildx/
  unknown flag: --provenance
reproduction: source ~/.zshenv && make release-mac-snapshot
started: First time running goreleaser locally — docker was already installed but without buildx

## Eliminated

- hypothesis: GoReleaser config explicitly sets --provenance flag
  evidence: build_flag_templates in .goreleaser.yaml only has --pull, --label=*, --platform=*. GoReleaser itself injects --provenance when use defaults to buildx.
  timestamp: 2026-03-22T00:00:00Z

- hypothesis: Docker daemon is running but missing buildx
  evidence: `docker info` shows "failed to connect to docker API at unix:///var/run/docker.sock" — daemon is not running at all. docker CLI is version 29.1.2 but no buildx plugin is registered (`docker: unknown command: docker buildx`).
  timestamp: 2026-03-22T00:00:00Z

## Evidence

- timestamp: 2026-03-22T00:00:00Z
  checked: .goreleaser.yaml dockers section
  found: Two docker entries (amd64, arm64) with build_flag_templates but no `use:` field. GoReleaser v2 defaults `use` to `buildx`, which calls `docker buildx build` and passes `--provenance=false` (or similar). The legacy docker CLI does not know this flag.
  implication: Adding `use: docker` per entry will force `docker build` (no --provenance). Alternatively, skip docker entirely in snapshot mode.

- timestamp: 2026-03-22T00:00:00Z
  checked: docker buildx availability
  found: `docker: unknown command: docker buildx` — buildx plugin not installed. Docker daemon is also not running (no socket).
  implication: Snapshot runs on this machine cannot succeed with docker builds regardless of builder. Best local-dev fix is to skip docker in snapshot mode entirely via `--skip=docker`.

- timestamp: 2026-03-22T00:00:00Z
  checked: goreleaser version
  found: 2.14.3, darwin/arm64
  implication: Version confirms use of GoReleaser v2 which defaults dockers.use to buildx.

- timestamp: 2026-03-22T00:00:00Z
  checked: Makefile release-mac-snapshot target
  found: `goreleaser release --snapshot --clean` — no --skip flags.
  implication: Adding `--skip=docker` to the snapshot target will prevent docker build attempts during local dev where docker daemon may not be running or buildx unavailable.

## Resolution

root_cause: GoReleaser v2 defaults `dockers[].use` to `buildx`, causing it to invoke `docker buildx build` with a `--provenance` flag. The local docker install has no buildx plugin. Additionally, the Docker daemon is not running locally. The `.goreleaser.yaml` has no `use: docker` override and the Makefile snapshot target has no `--skip=docker` guard.

fix: Two-part fix:
  1. Add `use: docker` to both dockers entries in .goreleaser.yaml — this forces the legacy `docker build` command (no --provenance injection) for environments where buildx is unavailable.
  2. Update `release-mac-snapshot` in Makefile to pass `--skip=docker` so local snapshot runs never attempt docker builds (docker daemon not guaranteed to be running locally).

verification: goreleaser check passes (config valid, only pre-existing deprecation warnings). Fix removes --provenance injection by forcing `use: docker`; snapshot target skips docker builds entirely to avoid needing a running daemon.
files_changed: [.goreleaser.yaml, Makefile]
