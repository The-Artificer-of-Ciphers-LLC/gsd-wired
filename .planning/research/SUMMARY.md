# Project Research Summary

**Project:** gsd-wired (Claude Code Plugin: GSD + Beads/Dolt Agent Orchestration)
**Domain:** Developer CLI tool — Claude Code plugin with container distribution and dependency management
**Researched:** 2026-03-21
**Confidence:** HIGH

## Executive Summary

gsd-wired is a complete Go binary (MCP server + hook dispatcher + CLI) that now needs a distribution and setup layer. The core product (Phases 1-10) is already built: gsdw connects Claude Code to the beads graph via bd CLI, which in turn speaks MySQL wire protocol to a Dolt database. The new milestone adds what makes the tool actually installable by others: a Homebrew tap via GoReleaser, a `gsdw setup` wizard for dependency detection, container support for the Dolt server, and a `gsdw doctor` health check command. No changes to core Go source are needed except a new `check-deps` / `container` subcommand tree and targeted modifications to `client.go` to inject Dolt server env vars.

The recommended approach is to build distribution-first (GoReleaser + brew tap), then setup UX (dependency detection, connection wizard), then container runtime support in priority order: Docker first (widest audience), Apple Container second (gate on macOS 26 + Apple Silicon). The `gsdw container setup` command must be idempotent, interactive, and explicitly non-destructive toward any existing `compose.override.yml`. The key insight from architecture research is that gsdw does not speak to Dolt directly — it only speaks to bd, and bd's connection to Dolt is controlled by `BEADS_DOLT_SERVER_*` env vars that gsdw must inject on every `exec.Command("bd", ...)` call when a container is configured.

The critical risks are: (1) the beads bug #2073 where `~/.config/bd/config.yaml` port config is silently ignored, meaning gsdw must inject env vars directly rather than relying on config files; (2) Apple Container's hard macOS 26 + Apple Silicon requirement, which narrows the audience and must be gated with a clear version check; and (3) `compose.override.yml` stomping, which must be handled with explicit file-existence checks before any write. Start with GoReleaser config to unblock distribution, then build setup/doctor, then container support.

---

## Key Findings

### Recommended Stack

The core stack is already locked in at HIGH confidence: Go 1.26.1, modelcontextprotocol/go-sdk v1.4.1 (MCP server), github.com/steveyegge/beads v0.61.0, github.com/dolthub/driver v1.83.8, spf13/cobra v1.10.2. The new milestone adds GoReleaser v2.14.3 as a dev dependency (not shipped) to drive all distribution channels from a single `.goreleaser.yaml`.

The container distribution strategy is: `dolthub/dolt-sql-server:latest` as the official Dolt container image (no custom Dockerfile), `gcr.io/distroless/static-debian12:nonroot` as the gsdw container base, and GoReleaser `dockers_v2` (alpha but stable) for multi-arch image builds. CGO_ENABLED=0 is required for cross-compilation — Dolt's embedded driver is pure Go. The binary must be distributed via four channels: `go install`, brew tap, ghcr.io container, and direct GitHub Release download — GoReleaser manages the last three from one config.

**Core technologies:**
- GoReleaser v2.14.3: cross-platform binary builds, Homebrew cask generation, GitHub Releases, container image push — industry standard, OSS tier covers all needs
- `homebrew_casks` (not `brews`): correct GoReleaser syntax since v2.10 for pre-compiled binaries; `brews` is deprecated, removal in v3
- dolthub/dolt-sql-server: official Dolt container image; maintained by DoltHub; no custom Dockerfile needed
- distroless/static-debian12:nonroot: zero-runtime base for the gsdw container; smaller attack surface than alpine; UID 65532 for security
- exec.LookPath (Go stdlib): dependency detection for bd, dolt, docker, podman, container — no library needed
- apple/container CLI: macOS 26 + Apple Silicon only; OCI-compatible; detect at runtime via PATH lookup

### Expected Features

The feature set divides into a critical path (GoReleaser config + dependency detection + gsdw setup + gsdw doctor) and enhancement layer (Apple Container support, idempotent re-run, dry-run flag). Remote fallback to local container is explicitly deferred to v1.x due to HIGH complexity and LOW usage frequency. Windows is explicitly out of scope. Interactive TUI wizards (bubbletea) are anti-features — simple sequential prompts suffice.

**Must have (table stakes):**
- `brew tap user/gsd-wired && brew install gsdw` — macOS developer expectation; GoReleaser generates formula on every release
- Dependency detection for bd + dolt on PATH with actionable install instructions — absence must produce a clear error, not a Go panic
- `gsdw setup` or `gsdw check-deps` command — entry point new users expect from any developer CLI
- Non-destructive compose fragment — never overwrite user's existing `compose.override.yml`
- Health check command — single command to verify everything is working post-setup
- Structured output with `[OK]` / `[WARN]` / `[FAIL]` prefix — scannable for spot issues

**Should have (differentiators):**
- Container runtime auto-detection (Apple Container > Docker > Podman priority order)
- Guided connection wizard (local vs. remote Dolt, collect host/port, write to config file)
- Idempotent re-run — detect what is already done and skip those steps
- `--dry-run` flag on setup
- Version check on dependencies against minimum supported versions

**Defer (v1.x+):**
- Remote fallback to local container on connection failure — HIGH complexity, retry/timeout logic required
- Podman support — Docker covers the non-Apple-Container case for v1
- Migration tooling for config schema changes — stabilize schema first

### Architecture Approach

The container integration is architecturally minimal on the gsdw side. The key change is in `internal/graph/client.go`: add `doltHost`, `doltPort`, `doltUser`, `doltPass` fields to the `Client` struct and append `BEADS_DOLT_SERVER_*` env vars in the `run()` method when they are set. gsdw does not manage Dolt directly — it injects connection parameters into the bd subprocess environment. All other container lifecycle management (start/stop/status/logs) goes in a new `internal/cli/container.go` subcommand that shells out to the detected container runtime.

The build order is: Layer 0 (runtime detection, config read/write — no gsdw deps) → Layer 1 (health check, client.go modification) → Layer 2 (container CLI subcommand, init.go flags) → Layer 3 (Apple Container path, compose fragment writer, MCP server config pickup). This layering means the Docker path can ship before Apple Container is complete.

**Major components:**
1. `internal/container/runtime.go` — detect and exec docker/podman/apple-container; no gsdw deps
2. `internal/container/health.go` — MySQL ping health check, wait-for-ready loop using net package
3. `internal/container/config.go` — read/write container config in `.beads/metadata.json`
4. `internal/cli/container.go` — `gsdw container` subcommand (setup/start/stop/status/logs)
5. `internal/graph/client.go` (modified) — inject BEADS_DOLT_SERVER_* env vars when container is configured
6. GoReleaser pipeline — `.goreleaser.yaml` + GitHub Actions release workflow + homebrew-gsd-wired tap repo

### Critical Pitfalls

1. **Hook stdout pollution kills JSON parsing** — any non-JSON output to stdout from a hook handler breaks Claude Code's parser; `fmt.Println` in hook handlers is fatal; wrap all bd subprocess output and only forward parsed JSON; use `os.Stderr` for all logging. Must be established in Phase 1.

2. **beads bug #2073: config.yaml port not read** — `~/.config/bd/config.yaml dolt.port` is silently ignored in some bd versions; gsdw must inject `BEADS_DOLT_SERVER_HOST` and `BEADS_DOLT_SERVER_PORT` as explicit env vars on every `bd exec.Command` call, not rely on config file inheritance.

3. **PreCompact hook is non-blocking** — PreCompact cannot block compaction; if the Dolt commit is slow (200-500ms) compaction proceeds and state is lost; design a two-stage pattern: write to a local WAL/staging area first (fast), then sync to Dolt asynchronously. Must be designed before any hook logic is built.

4. **Dolt write amplification per hook** — 50-100 PostToolUse hooks in an active session = 50-100 Dolt writes; each write creates content-addressed chunks; auto-stats at 30s intervals causes perpetual CPU burn on repos >2GB; batch commits at wave boundaries, disable or extend auto-stats, match hooks only to state-changing tools (Write/Edit/Bash, not Read/Glob/Grep).

5. **compose.override.yml stomping** — if the user has an existing `compose.override.yml`, writing a new one destroys their customizations; setup wizard must check for file existence, refuse to overwrite without `--force`, and print the fragment to stdout for manual merge.

6. **Apple Container macOS 26 hard requirement** — users on macOS 15 Sequoia get mysterious networking failures; gate on `sw_vers -productVersion >= 26.0` check and print a clear error before attempting anything; direct them to Docker Desktop or Podman.

7. **Volume mount before bd init** — if `gsdw container setup` runs before `bd init`, `.beads/dolt/` does not exist; container starts with an empty volume and creates an uninitialized Dolt DB; `bd ls` then fails; setup wizard must check for `.beads/dolt/` existence and run `bd init --backend dolt` first if needed.

---

## Implications for Roadmap

Based on combined research, the new milestone has a clear critical path: distribution infrastructure enables user acquisition, setup UX enables onboarding, container support enables production use. These build on each other and should be sequenced accordingly.

### Phase 1: Distribution Infrastructure
**Rationale:** Nothing else matters if users can't install the tool; GoReleaser config is a one-time setup that unblocks all other distribution channels; no source changes required.
**Delivers:** `brew tap user/gsd-wired && brew install gsdw` works; multi-arch binaries on GitHub Releases; ghcr.io container image published; `go install` path verified.
**Addresses:** Binary availability via brew (table stakes), multiple distribution channels.
**Avoids:** Manual release scripts becoming maintenance burden; using deprecated `brews` instead of `homebrew_casks`.
**Stack required:** GoReleaser v2.14.3, GitHub Actions release workflow, homebrew-gsd-wired tap repo, distroless/static base image, dockers_v2 for multi-arch.

### Phase 2: Dependency Detection and Setup Command
**Rationale:** Once the binary is installable, new users need a guided path to first use; dependency detection is LOW complexity and HIGH value; must be correct before container support is built on top of it.
**Delivers:** `gsdw check-deps` with structured [OK]/[WARN]/[FAIL] output; `gsdw setup` with connection wizard; config written to `~/.config/gsdw/config.toml`; `gsdw doctor` health check.
**Addresses:** Dependency detection (table stakes), actionable install instructions, health check command, `gsdw setup` / `gsdw doctor` entry point.
**Avoids:** Hook stdout pollution — all setup output goes to stderr or uses structured format; MCP server startup latency by keeping lazy initialization.
**Implements:** `internal/deps/check.go` (new), `gsdw check-deps` subcommand, connection wizard prompts, health check TCP dial.

### Phase 3: Docker Container Support for Dolt
**Rationale:** Docker is widest-coverage runtime (macOS + Linux); architecture research confirms the change is isolated to `client.go` env var injection + new `container.go` subcommand; this is the highest-value container runtime to implement first.
**Delivers:** `gsdw container setup` with Docker; `gsdw container start/stop/status/logs`; compose fragment generated non-destructively; Dolt data persisted via `.beads/dolt` volume mount; BEADS_DOLT_SERVER_* injected on every bd call.
**Addresses:** Docker/Podman container support, non-destructive compose fragment, container images for Dolt server.
**Avoids:** beads bug #2073 (inject env vars directly, not via config file); compose.override.yml stomping (check existence, refuse without --force); volume mount before bd init (check .beads/dolt exists first); port 3307 collision check.
**Implements:** `internal/container/runtime.go`, `internal/container/health.go`, `internal/container/config.go`, `internal/cli/container.go`, client.go modification.

### Phase 4: Apple Container Support
**Rationale:** Apple Container is macOS 26 + Apple Silicon only; narrower audience than Docker; architecture is identical from bd's view (same env vars, same port mapping syntax); the macOS version gate must be implemented correctly before any user-facing code runs.
**Delivers:** Apple Container detected and used when available on macOS 26+ Apple Silicon; `gsdw container setup` works with `container run` syntax; firewall requirement documented; compose fragment deferred (Apple Container has no compose equivalent — use `container run` directly).
**Addresses:** Apple Container support for macOS (as differentiator).
**Avoids:** macOS 26 hard requirement gate failure (sw_vers check before any container run attempt); no docker-compose equivalent for Apple Container (fall back to direct run commands).
**Implements:** macOS version detection in `runtime.go`, Apple Container branch in container subcommand, `container system start` requirement in setup wizard.

### Phase Ordering Rationale

- Distribution (Phase 1) must come first because it has zero source dependencies and enables everything else; a tool that can't be installed cannot be evaluated.
- Setup/detection (Phase 2) must precede container support because `gsdw container setup` builds on the same dependency detection infrastructure and connection wizard patterns.
- Docker (Phase 3) before Apple Container (Phase 4) because Docker has wider platform coverage and simpler networking; the runtime detection abstraction built in Phase 3 makes Phase 4 an additive branch, not a rewrite.
- The core hook pitfalls (stdout discipline, PreCompact non-blocking, write batching) were addressed in Phases 1-3 of the original milestone and do not need re-addressing here — the existing codebase already embeds these patterns.

### Research Flags

Phases likely needing deeper research during planning:
- **Phase 1:** GoReleaser `dockers_v2` is flagged alpha — verify current behavior against v2.14.3 docs before writing the full pipeline; confirm GITHUB_TOKEN permissions for ghcr.io push.
- **Phase 4:** Apple Container `container system start` networking behavior on macOS 26 — verify that firewall prompts do not block automated setup; check if Local Network permission is required for port forwarding.

Phases with standard patterns (skip additional research):
- **Phase 2:** exec.LookPath pattern is well-documented Go stdlib; connection wizard prompts are standard bufio patterns; no novel research needed.
- **Phase 3:** Docker run + compose fragment patterns are exhaustively documented; dolthub/dolt-sql-server official image behavior is well-known; BEADS_DOLT_SERVER_* env vars are documented in beads DOLT-BACKEND.md.

---

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | All core technologies have official docs verified; GoReleaser v2.14.3 confirmed; beads/Dolt versions confirmed from existing go.mod |
| Features | HIGH | Patterns from rustup, brew, flyctl, kubectl-doctor are well-documented; Apple Container feature set confirmed from official GitHub repo |
| Architecture | HIGH | bd env var behavior verified from beads DOLT-BACKEND.md source; Dolt container behavior verified from DoltHub official docs; Apple Container port mapping verified from official how-to |
| Pitfalls | HIGH | Most pitfalls verified against official Claude Code docs, DoltHub production reports, and beads GitHub issues; bug #2073 is a confirmed open issue |

**Overall confidence:** HIGH

### Gaps to Address

- **beads bug #2073 workaround durability**: the env var injection workaround is correct today but may be fixed in a future bd version; add a version check and a comment in client.go noting the bug so the workaround can be removed when upstream fixes it.
- **dockers_v2 alpha stability**: the feature works but API could change before GoReleaser v3 stabilizes it; pin to GoReleaser v2.14.3 in the GitHub Actions workflow (`version: v2.14.3`, not `latest`) to prevent unexpected breakage.
- **Apple Container compose limitation**: Apple Container has no compose equivalent; if a user wants Dolt + another service orchestrated together, the answer is "use Docker" — document this explicitly rather than trying to bridge it.
- **Podman 127.0.0.1 binding quirk on macOS**: Podman machine on macOS has a known port forwarding issue with explicit `127.0.0.1` binding; if Podman is supported in v1, use `0.0.0.0:3307:3306` for the run command — document the difference from Docker.

---

## Sources

### Primary (HIGH confidence)

**GoReleaser:**
- [GoReleaser v2.14 announcement](https://goreleaser.com/blog/goreleaser-v2.14/) — current version, March 2026
- [GoReleaser deprecation notices](https://goreleaser.com/deprecations/) — confirms `brews` deprecated in v2.10
- [GoReleaser homebrew casks docs](https://goreleaser.com/customization/homebrew_casks/) — current cask config fields
- [GoReleaser dockers_v2 docs](https://goreleaser.com/customization/dockers_v2/) — multi-arch container config

**Beads / Dolt:**
- [Beads DOLT-BACKEND.md](https://github.com/steveyegge/beads/blob/main/docs/DOLT-BACKEND.md) — BEADS_DOLT_SERVER_* env vars, server mode
- [Beads issue #2073](https://github.com/steveyegge/beads/issues/2073) — config.yaml port bug, env var workaround
- [Dolt Docker documentation](https://docs.dolthub.com/introduction/installation/docker) — official image, env vars, volume paths
- [dolthub/dolt-sql-server on Docker Hub](https://hub.docker.com/r/dolthub/dolt-sql-server) — official image

**Apple Container:**
- [apple/container GitHub](https://github.com/apple/container) — source, releases, macOS 26 requirement
- [Container CLI how-to](https://github.com/apple/container/blob/main/docs/how-to.md) — port mapping, networking model
- [Apple WWDC25 session: Meet Containerization](https://developer.apple.com/videos/play/wwdc2025/346/) — architectural overview

**Homebrew:**
- [How to Create and Maintain a Tap](https://docs.brew.sh/How-to-Create-and-Maintain-a-Tap) — official tap docs

**Docker Compose:**
- [Docker Compose merge docs](https://docs.docker.com/compose/how-tos/multiple-compose-files/merge/) — compose.override.yml auto-discovery, merge semantics

### Secondary (MEDIUM confidence)

- [Apple Container Install Guide (4sysops)](https://4sysops.com/archives/install-apple-container-cli-running-containers-natively-on-macos-15-sequoia-and-macos-26-tahoe/) — system requirements, post-install steps
- [Podman macOS port forwarding issue #11528](https://github.com/containers/podman/issues/11528) — 127.0.0.1 binding quirk on podman machine
- [Production Dolt auto-stats report](https://gist.github.com/l0g1x/ef6dc1a971fa124e8d5939f3115b4e7d) — CPU burn on repos >2GB at 30s interval
- [CLI UX Patterns (Lucas F. Costa)](https://www.lucasfcosta.com/blog/ux-patterns-cli-tools) — guided onboarding, human-readable errors

### Tertiary (LOW confidence)

- [Multi-agent error amplification research](https://towardsdatascience.com/why-your-multi-agent-system-is-failing-escaping-the-17x-error-trap-of-the-bag-of-agents/) — 17x error amplification in unstructured multi-agent systems; informs orchestration validation gate design

---
*Research completed: 2026-03-21*
*Ready for roadmap: yes*
