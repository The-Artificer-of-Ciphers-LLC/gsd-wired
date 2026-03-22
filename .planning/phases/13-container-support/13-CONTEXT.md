# Phase 13: Container Support - Context

**Gathered:** 2026-03-22
**Status:** Ready for planning
**Source:** Auto-mode (Claude selected recommended defaults)

<domain>
## Phase Boundary

Developer can start a Dolt server in a container with one command using whichever runtime is available. Docker/Podman and Apple Container support, compose fragment, runtime auto-detection. Delivers CNTR-01 through CNTR-07.

</domain>

<decisions>
## Implementation Decisions

### Design principle
- **D-01:** gsdw is the developer interface. Container management is user-facing — `gsdw container start/stop` is what developers use.
- **D-02:** Support three runtimes: Apple Container (macOS 26+), Docker, Podman. Auto-detect priority: Apple Container → Docker → Podman.

### Container lifecycle
- **D-03:** `gsdw container start` launches Dolt server using detected runtime. `gsdw container stop` stops it cleanly.
- **D-04:** Uses `dolthub/dolt-sql-server` image. Port mapping: host 3307 → container 3306 (avoids MySQL collision).
- **D-05:** Data persistence: host `.beads/dolt/` mounted into container. Survives container restarts.

### Compose fragment
- **D-06:** `gsdw container start` produces `gsdw.compose.yaml` fragment. Developer can use `docker compose -f docker-compose.yml -f gsdw.compose.yaml up` to integrate.
- **D-07:** Never modify user's existing compose files. Fragment is standalone, additive only.
- **D-08:** If `gsdw.compose.yaml` already exists, refuse to overwrite without `--force`.

### Runtime detection
- **D-09:** Apple Container gated on macOS 26 + Apple Silicon. Detection: `sw_vers -productVersion` + `uname -m`.
- **D-10:** On macOS 15 or older: clear error directing developer to Docker or Podman. No silent fallback.
- **D-11:** Container runtime detection reuses Phase 12's `internal/deps/` infrastructure.

### Architecture (from research)
- **D-12:** New `internal/container/` package with runtime abstraction: `Runtime` interface implemented by `DockerRuntime`, `PodmanRuntime`, `AppleContainerRuntime`.
- **D-13:** `DetectRuntime()` returns the first available runtime in priority order.

### Claude's Discretion
- Exact Docker/Podman CLI command construction
- Apple Container CLI command differences
- Container naming convention
- Health check implementation (wait for Dolt to accept connections)
- Error messages for common container failures

</decisions>

<canonical_refs>
## Canonical References

### Existing code
- `internal/deps/check.go` — `checkContainerRuntime()` detects Docker/Podman (extend for Apple Container)
- `internal/cli/root.go` — Command registration
- `internal/cli/setup.go` — Pattern for interactive CLI commands

### Research
- `.planning/research/ARCHITECTURE.md` — Container integration architecture, network topology
- `.planning/research/PITFALLS.md` — `--net=host` doesn't work on macOS, port 3307 default
- `.planning/research/STACK.md` — dolthub/dolt-sql-server image, Apple Container CLI

### Requirements
- `.planning/REQUIREMENTS.md` — CNTR-01 through CNTR-07
- `.planning/ROADMAP.md` §Phase 13 — Success criteria

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/deps/check.go` — Container runtime detection (Docker/Podman already detected)
- `internal/cli/` — Cobra subcommand patterns
- `exec.Command` patterns throughout the codebase

### Integration Points
- `internal/cli/root.go` — New: `container` subcommand with `start`/`stop` sub-subcommands
- New package: `internal/container/` — Runtime abstraction and container lifecycle
- `.beads/dolt/` — Volume mount target for persistence

</code_context>

<deferred>
## Deferred Ideas

- Container health monitoring dashboard (CNTR-A01, v2)
- Container auto-update (CNTR-A02, v2)
- Multi-container orchestration (CNTR-A03, v2)

</deferred>

---

*Phase: 13-container-support*
*Context gathered: 2026-03-22 via auto-mode*
