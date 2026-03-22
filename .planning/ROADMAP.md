# Roadmap: gsd-wired

## Milestones

- ✅ **v1.0** — Full GSD lifecycle on beads graph (shipped 2026-03-22)
- 🔄 **v1.0 Installation Toolkit** — Distribution, setup UX, containers, connectivity (in progress)

## Phases

<details>
<summary>✅ v1.0 (Phases 1-10) — SHIPPED 2026-03-22</summary>

- [x] Phase 1: Binary Scaffold (2/2 plans) — Go binary, Cobra CLI, MCP stdio, plugin manifest
- [x] Phase 2: Graph Primitives (2/2 plans) — bd CLI wrapper, CRUD, local index, gsdw ready
- [x] Phase 3: MCP Server (2/2 plans) — 8 tools, lazy Dolt init, batched writes
- [x] Phase 4: Hook Integration (3/3 plans) — All 4 hooks with state persistence
- [x] Phase 5: Project Initialization (2/2 plans) — /gsd-wired:init, get_status, 3 init modes
- [x] Phase 6: Research + Planning (2/2 plans) — Research agents via beads, plan checker, topo sort
- [x] Phase 7: Execution + Verification (3/3 plans) — Wave execution, verification, remediation
- [x] Phase 8: Ship + Status (2/2 plans) — PR creation, phase advancement, enriched status
- [x] Phase 9: Token-Aware Context (2/2 plans) — Hot/warm/cold tiering, budget-aware SessionStart
- [x] Phase 10: Coexistence (2/2 plans) — .planning/ fallback parsers, read-only guarantee

</details>

### v1.0 Installation Toolkit (Phases 11-14)

- [ ] **Phase 11: Distribution Infrastructure** — GoReleaser pipeline, brew tap, signed macOS binary, ghcr.io image, `go install`
- [ ] **Phase 12: Setup UX** — `gsdw check-deps`, `gsdw setup` wizard, `gsdw doctor` health check
- [ ] **Phase 13: Container Support** — Docker/Podman and Apple Container runtime, `gsdw container` subcommand, compose fragment
- [ ] **Phase 14: Connectivity** — `gsdw connect` wizard, `.gsdw/connection.json`, client.go env var injection, remote fallback

## Phase Details

### Phase 11: Distribution Infrastructure
**Goal**: gsd-wired is installable by any developer through standard channels without manual build steps
**Depends on**: Nothing (pipeline work, no source changes)
**Requirements**: DIST-01, DIST-02, DIST-03, DIST-04, DIST-05, DIST-06
**Success Criteria** (what must be TRUE):
  1. A developer on macOS can run `brew tap The-Artificer-of-Ciphers-LLC/tap && brew install gsdw` and get a working binary
  2. A developer can run `go install github.com/The-Artificer-of-Ciphers-LLC/gsd-wired/cmd/gsdw@latest` and get a working binary
  3. Pushing a git tag triggers the GitHub Actions pipeline and publishes binaries for linux/amd64, linux/arm64, darwin/amd64, darwin/arm64 to GitHub Releases
  4. The macOS binary passes Gatekeeper — no "unidentified developer" warning on first launch
  5. `docker pull ghcr.io/the-artificer-of-ciphers-llc/gsdw:latest` succeeds and pulls a multi-arch image
**Plans:** 2 plans
Plans:
- [x] 11-01-PLAN.md — Version ldflags injection, --json flag, GoReleaser config, Dockerfile (checkpoint pending: homebrew tap)
- [~] 11-02-PLAN.md — GitHub Actions release workflow created (Task 1 done); checkpoint:human-verify at Task 2

### Phase 12: Setup UX
**Goal**: A developer who just installed gsdw can verify their environment and get guided to a working state in one command
**Depends on**: Phase 11 (binary must be installable before setup can be validated end-to-end)
**Requirements**: SETUP-01, SETUP-02, SETUP-03, SETUP-04, SETUP-05
**Success Criteria** (what must be TRUE):
  1. `gsdw check-deps` reports `[OK]`, `[WARN]`, or `[FAIL]` for bd, dolt, Go, and container runtime with actionable install instructions for any missing dependency
  2. `gsdw check-deps` finds bd and dolt even when they are in `$(go env GOPATH)/bin` but not in PATH
  3. `gsdw setup` walks the developer through dependency installation, offering brew, `go install`, or binary download for each missing tool
  4. `gsdw doctor` runs without modifying any files and prints a scannable status report for all dependencies, containers, and connections
**Plans:** 2 plans
Plans:
- [x] 12-01-PLAN.md — Dependency detection package (internal/deps), check-deps and doctor CLI commands
- [x] 12-02-PLAN.md — Interactive setup wizard with install method selection

### Phase 13: Container Support
**Goal**: A developer can start a Dolt server in a container with a single command using whichever runtime is available on their machine
**Depends on**: Phase 12 (container runtime detection reuses dependency detection infrastructure)
**Requirements**: CNTR-01, CNTR-02, CNTR-03, CNTR-04, CNTR-05, CNTR-06, CNTR-07
**Success Criteria** (what must be TRUE):
  1. `gsdw container start` launches a Dolt server container and reports success; `gsdw container stop` stops it cleanly
  2. On a machine with Docker, the container uses `dolthub/dolt-sql-server` with Dolt data persisted at the host `.beads/dolt/` path across container restarts
  3. Running `gsdw container start` produces a `gsdw.compose.yaml` fragment that can be passed to docker compose via `-f` without modifying any existing docker-compose file
  4. On macOS 26 + Apple Silicon with the `container` CLI installed, gsdw uses Apple Container automatically; on other machines it falls back to Docker then Podman
  5. On macOS 15 or older, attempting Apple Container produces a clear error directing the developer to Docker or Podman
**Plans**: TBD

### Phase 14: Connectivity
**Goal**: gsdw knows how to reach the Dolt server and automatically passes that configuration to every bd command, with graceful handling when the server is unreachable
**Depends on**: Phase 13 (container must exist before connection config is meaningful)
**Requirements**: CONN-01, CONN-02, CONN-03, CONN-04, CONN-05, CONN-06
**Success Criteria** (what must be TRUE):
  1. `gsdw connect` prompts for local or remote mode, collects host and port, and writes `.gsdw/connection.json`
  2. After running `gsdw connect`, every subsequent `bd` subprocess invocation in the session receives `BEADS_DOLT_SERVER_HOST` and `BEADS_DOLT_SERVER_PORT` as environment variables without any manual configuration
  3. Before any graph operation, gsdw confirms the Dolt server is reachable and prints a clear error with troubleshooting steps if it is not
  4. When the configured remote host is unreachable, gsdw offers to fall back to the local container and proceeds only after the developer confirms
**Plans**: TBD

## Progress

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 1-10 | v1.0 | 22/22 | Complete | 2026-03-22 |
| 11. Distribution Infrastructure | v1.0 Install | 1/2 | Executing (checkpoint pending) | - |
| 12. Setup UX | v1.0 Install | 2/2 | Complete | 2026-03-22 |
| 13. Container Support | v1.0 Install | 0/TBD | Not started | - |
| 14. Connectivity | v1.0 Install | 0/TBD | Not started | - |

---

*Full v1.0 details archived at: `.planning/milestones/v1.0-ROADMAP.md`*
*Requirements archived at: `.planning/milestones/v1.0-REQUIREMENTS.md`*
