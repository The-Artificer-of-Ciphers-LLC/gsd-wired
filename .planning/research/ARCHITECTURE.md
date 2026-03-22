# Architecture Research — Container Integration Milestone

**Domain:** Container support for gsdw → bd → Dolt chain
**Researched:** 2026-03-21
**Confidence:** HIGH (bd env vars verified from source, Dolt Docker verified from official docs, Apple Container networking verified from official how-to)

---

## Context: What Already Exists

The v1.0 binary (`gsdw`) has this connection chain:

```
Claude Code
    |
    | stdio JSON-RPC
    v
gsdw (MCP server + hook dispatcher, single Go binary)
    |
    | exec.Command("bd", ..., "--json")
    | env: BEADS_DIR=.beads/
    v
bd CLI (external binary, must be on PATH)
    |
    | MySQL wire protocol
    | host: 127.0.0.1, port: 3307 (default)
    v
dolt sql-server (spawned by bd, running locally)
    |
    v
.beads/dolt/ (Dolt repository on local filesystem)
```

The `Client` struct in `internal/graph/client.go` resolves `bd` via `exec.LookPath("bd")` and sets `BEADS_DIR` on each invocation. It does not set any `BEADS_DOLT_SERVER_*` variables — those are inherited from the calling environment. This is the critical insight: **gsdw does not need to know about Dolt directly**. It only speaks to `bd`, and `bd` knows how to find Dolt.

---

## The Connection String Pattern

`bd` connects to Dolt over MySQL wire protocol. The connection parameters are:

| Parameter | Default | Override Mechanism |
|-----------|---------|-------------------|
| Host | `127.0.0.1` | `BEADS_DOLT_SERVER_HOST` env var |
| Port | `3307` | `BEADS_DOLT_SERVER_PORT` env var |
| User | `root` | `BEADS_DOLT_SERVER_USER` env var |
| Password | (empty) | `BEADS_DOLT_SERVER_PASS` env var |
| Mode | server (auto-start) | `BEADS_DOLT_SERVER_MODE=1` env var |

**Priority chain for these values (highest → lowest):**
1. Environment variables (`BEADS_DOLT_SERVER_*`)
2. `.beads/metadata.json` (local, gitignored)
3. `.beads/config.yaml` (team defaults)
4. `~/.config/bd/config.yaml` (user global)

Note: There is a known bug (beads issue #2073) where the global `~/.config/bd/config.yaml` `dolt.port` key is not consulted by the Dolt connector in some bd versions. Use environment variables or `.beads/metadata.json` as reliable overrides.

**When `bd` sees `BEADS_DOLT_SERVER_HOST` pointing to a container:**
- It skips spawning a local `dolt sql-server`
- It connects to the specified host:port directly using MySQL protocol
- The `.beads/dolt/` directory still holds the Dolt repository data (or it can be a volume mount in the container)

---

## Container Target Architecture

The containerized Dolt server replaces the locally-spawned `dolt sql-server`. Everything above `bd` in the stack is unchanged.

```
Claude Code
    |
    v
gsdw binary (unchanged)
    |
    | exec.Command("bd", ..., "--json")
    | env: BEADS_DIR=.beads/
    |     BEADS_DOLT_SERVER_HOST=127.0.0.1   <-- new
    |     BEADS_DOLT_SERVER_PORT=3307          <-- new
    v
bd CLI (unchanged)
    |
    | MySQL wire protocol → localhost:3307
    v
[Container Runtime: Docker | Podman | Apple Container]
    |
    | port mapping: host:3307 → container:3306
    v
dolthub/dolt-sql-server container
    |
    v
/var/lib/dolt/ (volume mount → host .beads/dolt/)
```

**The key architectural decision:** gsdw must inject `BEADS_DOLT_SERVER_HOST` and `BEADS_DOLT_SERVER_PORT` into the environment of every `bd` exec.Command call when a container is in use. Currently `client.go` does `cmd.Env = append(os.Environ(), "BEADS_DIR="+c.beadsDir)`. It needs to additionally append the Dolt server env vars when they are configured.

---

## Container Runtime Options

### Option A: Docker / Podman (Recommended for v1)

Both Docker and Podman use the same `-p HOST_PORT:CONTAINER_PORT` flag and are interchangeable for this use case. Port forwarding to localhost is reliable on both (on macOS, Podman machine has quirks with `127.0.0.1` — use `0.0.0.0` binding or rely on the VM's own forwarding, not explicit host IP specification in `-p`).

**Run command:**
```bash
docker run -d \
  --name gsdw-dolt \
  -p 3307:3306 \
  -e DOLT_ROOT_PASSWORD="" \
  -e DOLT_ROOT_HOST=% \
  -v "$(pwd)/.beads/dolt:/var/lib/dolt" \
  dolthub/dolt-sql-server:latest
```

**Image:** `dolthub/dolt-sql-server:latest` (official, on Docker Hub)
**Container port:** 3306 (MySQL default inside the container)
**Host port:** 3307 (matches bd's default; avoids collision with any local MySQL on 3306)
**Data volume:** `.beads/dolt` on host → `/var/lib/dolt` in container (persistence)

**Podman drop-in:** Replace `docker` with `podman` — syntax is identical.

### Option B: Apple Container (macOS 26 Tahoe only)

Apple Container requires macOS 26 (Tahoe, released September 2025). It does not support macOS 15 Sequoia in any meaningful network topology. Do not promise support on Sequoia.

**Run command:**
```bash
container run -d \
  --name gsdw-dolt \
  -p 127.0.0.1:3307:3306 \
  -v "$(pwd)/.beads/dolt:/var/lib/dolt" \
  dolthub/dolt-sql-server:latest
```

**Networking model:** Apple Container assigns dedicated IPs to containers within a VM network, but `-p` port forwarding to the host's loopback interface works the same as Docker from gsdw's perspective. From `bd`'s view, it connects to `127.0.0.1:3307` — no special handling needed vs. Docker.

**OCI compatibility:** Apple Container pulls and runs standard OCI images, so `dolthub/dolt-sql-server` works without modification.

**Gate this on macOS 26 detection.** Check `sw_vers -productVersion` and compare to `26.0`. If not macOS 26, emit a clear error: "Apple Container requires macOS 26 Tahoe. Use Docker or Podman instead."

---

## Drop-In docker-compose Fragment

**Pattern: `compose.override.yml` (auto-discovered, non-destructive)**

Docker Compose automatically reads both `compose.yml` (or `docker-compose.yml`) and `compose.override.yml` from the same directory. The override file is merged on top of the base. If `compose.override.yml` already exists in the user's project, gsdw must not overwrite it — check first and emit a warning.

**Behavior:** Adding a brand new service to `compose.override.yml` that is not in the base `compose.yml` is safe — Compose merges by appending new services. No modification to the user's existing compose file is needed.

**Fragment to write as `compose.override.yml` (or merge into it):**
```yaml
# gsdw — Dolt database for beads graph storage
# Generated by: gsdw container setup
# Safe to commit. Do not edit the data volume path.

services:
  gsdw-dolt:
    image: dolthub/dolt-sql-server:latest
    container_name: gsdw-dolt
    restart: unless-stopped
    ports:
      - "127.0.0.1:3307:3306"
    environment:
      DOLT_ROOT_HOST: "%"
    volumes:
      - ./.beads/dolt:/var/lib/dolt
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "127.0.0.1", "-P", "3306"]
      interval: 5s
      timeout: 3s
      retries: 10
      start_period: 10s
```

**Collision rule:** The service name `gsdw-dolt` and container name `gsdw-dolt` are namespaced to this tool. If the user's compose already has a service named `gsdw-dolt`, that is their problem to resolve. Document this assumption.

**Alternative pattern — standalone compose file:**
Write to `compose.gsdw.yml` and instruct users to run `docker compose -f compose.yml -f compose.gsdw.yml up`. This is strictly non-destructive but requires user action on every invocation. Prefer `compose.override.yml` for zero-friction.

---

## New Components Required

### 1. Container Configuration in gsdw

**New field on `graph.Client`:**
```go
type Client struct {
    bdPath      string
    beadsDir    string
    batchMode   bool
    doltHost    string   // empty = use bd's default (local embedded)
    doltPort    string   // empty = use bd's default (3307)
    doltUser    string   // empty = use bd's default (root)
    doltPass    string   // empty = use bd's default ("")
}
```

**Modified `run()` in `client.go`:**
```go
cmd.Env = append(os.Environ(), "BEADS_DIR="+c.beadsDir)
if c.doltHost != "" {
    cmd.Env = append(cmd.Env, "BEADS_DOLT_SERVER_HOST="+c.doltHost)
    cmd.Env = append(cmd.Env, "BEADS_DOLT_SERVER_MODE=1")
}
if c.doltPort != "" {
    cmd.Env = append(cmd.Env, "BEADS_DOLT_SERVER_PORT="+c.doltPort)
}
```

**Config persistence:** Store container connection settings in `.beads/metadata.json` (already gitignored by bd) under a `gsdw` namespace key, or in `.planning/config.json` if that is the source of truth for gsdw config. Do not add a new config file.

### 2. `gsdw container` Subcommand

New CLI subcommand with three sub-subcommands:

| Command | Action |
|---------|--------|
| `gsdw container setup` | Detect runtime (docker/podman/container), pull image, write compose fragment, print env vars to set |
| `gsdw container start` | `docker run ...` or `docker compose up gsdw-dolt -d` |
| `gsdw container stop` | `docker stop gsdw-dolt` or `docker compose stop gsdw-dolt` |
| `gsdw container status` | `docker inspect gsdw-dolt`, parse health status, test MySQL connection |
| `gsdw container logs` | `docker logs gsdw-dolt --tail 50` |

**Runtime detection order:** `apple container` (if macOS 26) → `docker` → `podman` → error.

### 3. Health Check and Connection Wizard

`gsdw container setup` must be interactive (or have a `--yes` flag for non-interactive). Steps:

1. Detect container runtime (fail if none found)
2. Check if `.beads/dolt/` exists (if not, `bd init` first)
3. Check port 3307 availability (`net.Listen("tcp", "127.0.0.1:3307")`)
4. Write compose fragment or run container
5. Wait for health check (poll MySQL on `127.0.0.1:3307` with `root` / empty pass, max 30s)
6. Write connection config to `.beads/metadata.json`: `{"dolt_server_host": "127.0.0.1", "dolt_server_port": 3307}`
7. Print: "Container running. Add to your shell: `export BEADS_DOLT_SERVER_HOST=127.0.0.1 BEADS_DOLT_SERVER_PORT=3307`"

The shell export step is needed because `bd` reads env vars at invocation time, not from `.beads/metadata.json` reliably in all versions (see bug #2073). gsdw's `Client` will inject these vars itself, but when users run `bd` directly they need the env vars set.

### 4. Dependency Detection Update

Existing dependency detection (Homebrew formula milestone) must be extended:

| Dependency | Detection | Install Guidance |
|------------|-----------|-----------------|
| `docker` | `exec.LookPath("docker")` | "Install Docker Desktop from docker.com" |
| `podman` | `exec.LookPath("podman")` | "Install Podman: `brew install podman`" |
| `container` | `exec.LookPath("container")` | "Requires macOS 26. Install from github.com/apple/container" |

At least one runtime is required when container mode is requested. If none found, fail with guidance.

---

## Network Topology

```
Host macOS
├── Port 3307 (loopback: 127.0.0.1:3307)
│   ↕ port-forward
└── Container Runtime VM
    └── Dolt container
        └── Port 3306 (MySQL, bound to 0.0.0.0)
```

gsdw binary → bd binary → `127.0.0.1:3307` → container port forward → Dolt SQL server → `/var/lib/dolt` (volume mounted from `.beads/dolt/`)

**Why 3307 instead of 3306?** Users commonly have MySQL running on 3306. bd's default for server mode is already 3307. Using 3307 avoids collision with zero configuration.

**Network isolation (docker-compose users):** If gsdw-dolt is added to an existing compose setup, it shares the default network by default. Add `networks: [gsdw]` with a dedicated network definition if isolation is needed. For the compose fragment gsdw writes, use the default network for simplicity — the database port is bound to `127.0.0.1` only, so it is not reachable from other machines even without network isolation.

---

## Data Persistence Model

**Invariant:** Dolt data must survive container restarts.

`.beads/dolt/` on the host is the canonical Dolt repository. This is the same directory bd uses in embedded mode. The volume mount is:

```
Host: <project>/.beads/dolt/
Container: /var/lib/dolt/
```

**Migration path (embedded → container):**
1. Stop embedded server: `bd dolt stop`
2. Start container (data directory already populated)
3. Set env vars / update metadata.json
4. Verify: `bd ls` should return existing beads

**Migration path (container → embedded):**
1. Stop container
2. Unset `BEADS_DOLT_SERVER_HOST` / remove from metadata.json
3. bd auto-starts embedded server from the same `.beads/dolt/` directory

This reversibility means there is no lock-in to container mode.

---

## Modified Components (vs. New)

| Component | Change Type | What Changes |
|-----------|------------|--------------|
| `internal/graph/client.go` | Modified | Add doltHost/doltPort/doltUser/doltPass fields; inject env vars in `run()` |
| `internal/graph/client.go` | Modified | Add `NewClientWithConfig(cfg ContainerConfig)` constructor |
| `internal/cli/root.go` | Modified | Register `NewContainerCmd()` subcommand |
| `internal/cli/init.go` | Modified | Accept `--use-container` flag; configure Client with container params post-setup |
| `internal/mcp/server.go` | Modified | Read container config from metadata.json and pass to `NewClient` |

| Component | Change Type | What It Does |
|-----------|------------|--------------|
| `internal/cli/container.go` | New | `gsdw container` subcommand (setup/start/stop/status/logs) |
| `internal/container/runtime.go` | New | Runtime detection (docker/podman/apple-container), exec wrapper |
| `internal/container/health.go` | New | MySQL ping health check, wait-for-ready loop |
| `internal/container/config.go` | New | Read/write container config from `.beads/metadata.json` |
| `container/compose.override.yml.tmpl` | New | Go template for the compose fragment |

---

## Build Order for This Milestone

```
Layer 0 (no deps):
    internal/container/runtime.go  — runtime detection, no gsdw deps
    internal/container/config.go   — metadata.json read/write

Layer 1 (needs Layer 0):
    internal/container/health.go   — MySQL ping using net package
    internal/graph/client.go mod   — add container config fields, inject env vars

Layer 2 (needs Layer 1):
    internal/cli/container.go      — container subcommand (setup/start/stop/status)
    internal/cli/init.go mod       — --use-container flag integration

Layer 3 (needs Layer 2):
    Apple Container-specific path  — runtime detection + macOS 26 gate
    compose fragment writer        — template + non-destructive file check
    MCP server config pickup       — server.go reads metadata.json for container params
```

**Phase boundary recommendation:** Layer 0-1 is "it connects." Layer 2 is "it's usable from CLI." Layer 3 is "it supports all three runtimes."

---

## Apple Container Specifics

- **Requirement:** macOS 26 (Tahoe). Do not support Sequoia — networking does not work reliably.
- **Hardware:** Apple Silicon (M1+) required.
- **OCI images:** Standard OCI images work. No special Apple-format images needed.
- **Port syntax:** `-p 127.0.0.1:3307:3306` — same as Docker.
- **Installation:** PKG installer from github.com/apple/container/releases. Must run `container system start` before first use.
- **Firewall:** macOS Local Network firewall must allow `container` runtime. gsdw setup wizard should print this requirement.
- **No docker-compose:** Apple Container does not have a compose equivalent. Use `container run` directly in `gsdw container start`. The compose fragment is for Docker/Podman users.
- **Detection:** `exec.LookPath("container")` plus `sw_vers -productVersion` check ≥ 26.0.

---

## Pitfalls Specific to This Integration

### Pitfall 1: bd bug #2073 — metadata.json config not always read

The beads Dolt connector has a verified bug where `dolt.port` in `~/.config/bd/config.yaml` is ignored. The fix: gsdw must inject `BEADS_DOLT_SERVER_HOST` and `BEADS_DOLT_SERVER_PORT` as explicit env vars on every `bd` exec.Command, not rely on `.beads/metadata.json` alone.

### Pitfall 2: Volume mount before bd init

If the user runs `gsdw container setup` before `bd init`, the `.beads/dolt/` directory does not exist. The Dolt container will start but see an empty volume and create a fresh (uninitialized) database. Then `bd ls` will fail because the beads schema is missing. **Fix:** `gsdw container setup` must check that `.beads/dolt/` exists and is a valid Dolt repo before starting the container. If not, run `bd init --backend dolt` first.

### Pitfall 3: Port 3307 already in use

If the user has another Dolt server or bd running in embedded mode, port 3307 may be occupied. `gsdw container setup` must check before starting. If occupied, either: stop the embedded server (`bd dolt stop`), or use port 3308 and configure accordingly.

### Pitfall 4: Apple Container macOS 26 hard requirement

Users on macOS 15 Sequoia will get mysterious networking failures if they try Apple Container. Gate on `sw_vers` check and print a clear error before attempting to run anything. Direct them to Docker Desktop or Podman instead.

### Pitfall 5: compose.override.yml stomping user content

If the user has an existing `compose.override.yml`, writing a new one destroys their customizations. The setup wizard must: (1) detect existing file, (2) print the fragment to stdout for manual merge, (3) refuse to overwrite without `--force` flag.

### Pitfall 6: Podman machine on macOS — 127.0.0.1 binding

Podman machine on macOS has a known issue with port forwarding when the host side is bound to `127.0.0.1` explicitly. Use `0.0.0.0:3307:3306` (or omit the host IP) in the Podman run command. This makes the port available on all interfaces on the host, which is acceptable for localhost-only development. Document this difference from Docker.

---

## Integration Points Summary

| Integration | gsdw Change | bd Change | Notes |
|-------------|------------|-----------|-------|
| Docker/Podman Dolt container | Inject `BEADS_DOLT_SERVER_HOST/PORT` env vars | None | bd speaks MySQL to any host |
| Apple Container Dolt | Same env vars | None | Identical from bd's view |
| compose.override.yml | New file write | None | gsdw generates it |
| Container lifecycle | New `gsdw container` commands | None | gsdw shells out to docker/podman/container |
| Health checking | New health.go | None | Direct MySQL ping from Go |
| Config persistence | Writes `.beads/metadata.json` | Reads it | Shared config file |

---

## Sources

- [Beads DOLT-BACKEND.md](https://github.com/steveyegge/beads/blob/main/docs/DOLT-BACKEND.md) — `BEADS_DOLT_SERVER_HOST`, `BEADS_DOLT_SERVER_PORT`, server mode env vars (HIGH confidence)
- [Beads issue #2073](https://github.com/steveyegge/beads/issues/2073) — config.yaml port bug, env var workaround (HIGH confidence)
- [Dolt Docker documentation](https://docs.dolthub.com/introduction/installation/docker) — `dolthub/dolt-sql-server` image, env vars, volume paths (HIGH confidence)
- [Apple Container how-to](https://github.com/apple/container/blob/main/docs/how-to.md) — port mapping syntax, networking model, macOS 26 requirement (HIGH confidence)
- [Docker Compose merge docs](https://docs.docker.com/compose/how-tos/multiple-compose-files/merge/) — `compose.override.yml` auto-discovery, merge semantics (HIGH confidence)
- [Docker Compose include docs](https://docs.docker.com/compose/how-tos/multiple-compose-files/include/) — include directive, conflict behavior (HIGH confidence)
- [Podman macOS port forwarding](https://github.com/containers/podman/issues/11528) — 127.0.0.1 binding quirk on podman machine (MEDIUM confidence, linked issue)

---

*Architecture research for: gsd-wired container integration milestone*
*Researched: 2026-03-21*
