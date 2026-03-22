# Technology Stack

**Project:** gsd-wired (Claude Code Plugin: GSD + Beads/Dolt Agent Orchestration)
**Researched:** 2026-03-21 (core stack) / 2026-03-22 (distribution toolkit additions)

---

## Existing Stack (Core — Phases 1-10, Complete)

| Technology | Version | Purpose | Confidence |
|------------|---------|---------|------------|
| Go | 1.26.1 | Implementation language, single binary | HIGH |
| modelcontextprotocol/go-sdk | v1.4.1 | MCP server (JSON-RPC over stdio) | HIGH |
| github.com/steveyegge/beads | v0.61.0 | Graph persistence API | HIGH |
| github.com/dolthub/driver | v1.83.8 | Embedded Dolt DB access | HIGH |
| spf13/cobra | v1.10.2 | CLI subcommands (dual-mode binary) | HIGH |
| spf13/viper | v1.21.0 | Config file + env var management | MEDIUM |
| stretchr/testify | v1.8.3+ | Test assertions | HIGH |

See original STACK.md content below for full API patterns and architecture decisions.

---

## Distribution Toolkit Additions (New Milestone)

This section covers what to add for brew distribution, cross-platform binary release, container images (Dolt + beads), and dependency detection CLI. These are all **additive** — no changes to the core Go source other than a new `check-deps` subcommand.

### Release Tooling

| Technology | Version | Purpose | Why | Confidence |
|------------|---------|---------|-----|------------|
| GoReleaser | v2.14.3 | Cross-platform binary builds + GitHub Releases + Homebrew tap | Industry standard for Go release pipelines. Single YAML drives: multi-arch binaries, checksums, GitHub Release assets, Homebrew cask formula, and Docker image builds. Free OSS tier covers all needs. | HIGH |

**Install GoReleaser (dev dependency, not shipped):**
```bash
go install github.com/goreleaser/goreleaser/v2@latest
# or via brew for local dev
brew install goreleaser
```

**Version confirmed:** v2.14.3 released March 9, 2026. Requires Go 1.26. Source: [GoReleaser blog](https://goreleaser.com/blog/goreleaser-v2.14/).

**Do NOT use:** GoReleaser Pro. The OSS tier handles all required features (homebrew casks, multi-arch Docker, GitHub Releases). Pro adds GitLab/Gitea-specific features and alternate versioned formulas — not needed here.

### Homebrew Distribution

| Technology | Pattern | Purpose | Why | Confidence |
|------------|---------|---------|-----|------------|
| GitHub repo: `homebrew-gsd-wired` | owner/homebrew-gsd-wired | Homebrew tap | Convention: Homebrew strips `homebrew-` prefix, so `brew tap user/gsd-wired` resolves to `github.com/user/homebrew-gsd-wired`. GoReleaser auto-pushes cask formula on release. | HIGH |
| GoReleaser `homebrew_casks` section | v2.10+ syntax | Cask formula generation | `brews` (formula) is deprecated since v2.10. `homebrew_casks` is the current correct approach. Migration is rename-only for simple cases. | HIGH |

**Critical:** Use `homebrew_casks`, not `brews`. GoReleaser v2.10 deprecated `brews` (formula) because formulas are for source builds; pre-compiled binaries belong in casks. Removal planned in v3.

**.goreleaser.yaml homebrew_casks section:**
```yaml
homebrew_casks:
  - name: gsdw
    repository:
      owner: "{{ .Env.GITHUB_ORG }}"
      name: homebrew-gsd-wired
      branch: main
      token: "{{ .Env.HOMEBREW_TAP_GITHUB_TOKEN }}"
    commit_author:
      name: goreleaserbot
      email: bot@goreleaser.com
    homepage: "https://github.com/{{ .Env.GITHUB_ORG }}/gsd-wired"
    description: "GSD + Beads/Dolt workflow orchestration for Claude Code"
    license: "MIT"
    skip_upload: auto
```

**User install (once tap is published):**
```bash
brew tap user/gsd-wired
brew install gsdw
```

### Cross-Platform Binary Builds

GoReleaser `builds` section drives this. The binary `gsdw` needs macOS (arm64, amd64) and Linux (arm64, amd64). Windows is out of scope — Claude Code is macOS/Linux only.

**.goreleaser.yaml builds section:**
```yaml
before:
  hooks:
    - go mod tidy
    - go generate ./...

builds:
  - id: gsdw
    binary: gsdw
    main: ./cmd/gsdw
    goos:
      - darwin
      - linux
    goarch:
      - amd64
      - arm64
    env:
      - CGO_ENABLED=0
    ldflags:
      - -s -w
      - -X main.version={{.Version}}
      - -X main.commit={{.Commit}}
      - -X main.date={{.Date}}

archives:
  - id: gsdw
    builds: [gsdw]
    format: tar.gz
    format_overrides:
      - goos: windows
        format: zip
    name_template: "{{ .ProjectName }}_{{ .Os }}_{{ .Arch }}"

checksum:
  name_template: "checksums.txt"

changelog:
  sort: asc
  filters:
    exclude:
      - "^docs:"
      - "^test:"
      - "^chore:"
```

**CGO_ENABLED=0 is required.** The binary must be statically linked. GoReleaser cross-compiles for Linux on macOS — CGO would break this. Dolt embedded driver does NOT require CGO (confirmed: uses pure Go implementation).

### Container Images — Dolt Server

| Technology | Version | Purpose | Why | Confidence |
|------------|---------|---------|-----|------------|
| dolthub/dolt-sql-server | latest (1.x) | Official Dolt container image | Maintained by DoltHub. MySQL-compatible SQL server. Updated on every Dolt release. Supports linux/amd64 and linux/arm64. | HIGH |

**Do NOT build a custom Dolt container image.** `dolthub/dolt-sql-server` is the official image. It runs `dolt sql-server --host 0.0.0.0 --port 3306` by default. No custom Dockerfile needed.

**docker-compose fragment for Dolt server:**
```yaml
# docker-compose.dolt.yml — drop-in fragment, does not modify existing compose files
services:
  dolt:
    image: dolthub/dolt-sql-server:latest
    environment:
      DOLT_ROOT_PASSWORD: "${DOLT_ROOT_PASSWORD:-changeme}"
      DOLT_ROOT_HOST: "%"
    ports:
      - "${DOLT_PORT:-3306}:3306"
    volumes:
      - dolt-data:/var/lib/dolt
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "127.0.0.1", "-u", "root", "-p${DOLT_ROOT_PASSWORD:-changeme}"]
      interval: 5s
      timeout: 3s
      retries: 10
    restart: unless-stopped

volumes:
  dolt-data:
```

**Usage for non-destructive include with existing compose:**
```bash
# Add to existing project: does not touch their docker-compose.yml
docker compose -f docker-compose.yml -f docker-compose.dolt.yml up -d dolt
```

**Environment variables:**
- `DOLT_ROOT_PASSWORD` — root password (required for security; default `changeme` is dev-only)
- `DOLT_ROOT_HOST` — `%` means any host can connect (required for container networking)
- Port 3306 is the Dolt/MySQL default; configurable via `DOLT_PORT`

### Container Images — gsdw (beads wrapper)

| Technology | Version | Purpose | Why | Confidence |
|------------|---------|---------|-----|------------|
| gcr.io/distroless/static-debian12 | nonroot | Base image for gsdw container | Statically compiled Go binary needs zero runtime. Distroless: no shell, no package manager, minimal attack surface. `nonroot` variant runs as UID 65532 (security). | HIGH |
| GoReleaser `dockers_v2` | alpha (v2.12+) | Multi-arch container image builds | Reuses pre-built binaries — no in-container compilation. Pushes linux/amd64 and linux/arm64 manifests. Currently experimental but functional. | MEDIUM |

**Dockerfile for gsdw container:**
```dockerfile
# Dockerfile.container — used by GoReleaser, not for local dev builds
FROM gcr.io/distroless/static-debian12:nonroot
COPY gsdw /usr/local/bin/gsdw
ENTRYPOINT ["/usr/local/bin/gsdw"]
```

**GoReleaser dockers_v2 configuration:**
```yaml
dockers_v2:
  - id: gsdw
    ids: [gsdw]
    dockerfile: Dockerfile.container
    images:
      - "ghcr.io/{{ .Env.GITHUB_ORG }}/gsdw"
    tags:
      - "{{ .Version }}"
      - "latest"
    platforms:
      - linux/amd64
      - linux/arm64
```

**Note on dockers_v2 alpha status:** The feature works but is flagged experimental (scheduled to replace `dockers` + `docker_manifests` in GoReleaser v3). Use it — the old `dockers` + `docker_manifests` split approach is more complex for multi-arch. The alpha label reflects API stability, not reliability.

**What NOT to build:** Do not write a custom multi-stage Dockerfile that compiles Go inside the container. GoReleaser already produces the binary for each target arch — just COPY it in. Multi-stage builds duplicate compilation work and require Go toolchain inside the image.

### Apple Container Support (macOS)

| Technology | Version | Purpose | Why | Confidence |
|------------|---------|---------|-----|------------|
| apple/container CLI | latest (GitHub releases) | Run OCI containers natively on macOS | Announced WWDC 2025. Runs each container in its own lightweight VM. OCI-compatible — pulls from ghcr.io, Docker Hub, any standard registry. CLI is Docker-compatible (pull, run, push). | MEDIUM |

**System requirements (HIGH confidence):**
- Apple Silicon Mac only (M1/M2/M3/M4) — Intel not supported
- macOS 26 or later — takes advantage of new virtualization APIs. macOS 15 runs with limited networking.

**Installation:**
```bash
# Download signed installer from GitHub releases (no brew formula yet)
# https://github.com/apple/container/releases
```

**Key commands — identical pattern to Docker/Podman:**
```bash
# Pull and run Dolt server
container run -d \
  --name dolt \
  -e DOLT_ROOT_PASSWORD=secret \
  -e DOLT_ROOT_HOST=% \
  -p 3306:3306 \
  dolthub/dolt-sql-server:latest

# Pull gsdw image
container image pull ghcr.io/user/gsdw:latest

# Run gsdw container
container run ghcr.io/user/gsdw:latest serve

# Registry auth
container registry login ghcr.io
```

**Integration strategy:** gsdw's container support code should detect available container runtime via PATH lookup in this order: `container` (Apple) → `docker` → `podman`. Run the same docker-compose fragment with whichever runtime is found. Apple Container does not support compose natively — must invoke container CLI directly or fall back to docker compose.

**Apple Container limitation for compose:** `apple/container` CLI does not support `docker compose` or `container compose`. For compose-style orchestration (Dolt + gsdw together), fall back to `docker compose -f docker-compose.dolt.yml` when `docker` or `podman` is available, or document manual multi-container startup for Apple Container users.

### Dependency Detection CLI

This is implemented as a new `gsdw check-deps` subcommand in the existing Go binary — no new library needed.

**Detection strategy using Go stdlib only:**

```go
// internal/deps/check.go
package deps

import (
    "fmt"
    "os/exec"
)

type Dep struct {
    Name        string
    Binary      string
    Required    bool
    InstallHint string
    CheckFn     func() (string, error) // returns detected version string
}

var Required = []Dep{
    {
        Name:   "bd (beads)",
        Binary: "bd",
        Required: true,
        InstallHint: "go install github.com/steveyegge/beads/cmd/bd@latest",
        CheckFn: func() (string, error) {
            out, err := exec.Command("bd", "--version").Output()
            return string(out), err
        },
    },
    {
        Name:   "dolt",
        Binary: "dolt",
        Required: true,
        InstallHint: "brew install dolt  OR  go install github.com/dolthub/dolt/go/cmd/dolt@latest",
        CheckFn: func() (string, error) {
            out, err := exec.Command("dolt", "version").Output()
            return string(out), err
        },
    },
}

var Optional = []Dep{
    {
        Name:   "docker",
        Binary: "docker",
        Required: false,
        InstallHint: "https://docs.docker.com/get-docker/",
    },
    {
        Name:   "container (Apple)",
        Binary: "container",
        Required: false,
        InstallHint: "https://github.com/apple/container/releases",
    },
}

func Check(deps []Dep) []Result {
    // exec.LookPath for each binary, run CheckFn if found
    // return structured results for display
}
```

**What NOT to build:** Do not shell out to `brew` to check installation. `exec.LookPath` is sufficient and works regardless of how the dependency was installed (brew, go install, direct download). Do not write a shell script — the Go binary handles detection so there's no external script to maintain.

**Output format for `gsdw check-deps`:**
```
gsdw dependency check
---------------------
[OK]  bd v0.61.0     — beads graph CLI
[OK]  dolt v1.83.8   — Dolt SQL server
[--]  docker          — not found (optional, needed for container images)
                        Install: https://docs.docker.com/get-docker/
[--]  container       — not found (optional, Apple Silicon + macOS 26 only)
                        Install: https://github.com/apple/container/releases

Required dependencies: 2/2 OK
Optional dependencies: 0/2 found

Ready to use gsdw.
```

---

## Alternatives Considered (Distribution Toolkit)

| Category | Recommended | Alternative | Why Not |
|----------|-------------|-------------|---------|
| Release tooling | GoReleaser | Manual `go build` + shell scripts | GoReleaser generates checksums, release notes, Homebrew formulas, and Docker manifests automatically. Manual scripts become a maintenance burden. |
| Homebrew config | `homebrew_casks` | `brews` (formula) | `brews` is deprecated since v2.10, removed in v3. Casks are correct for pre-compiled binaries. |
| Container base | distroless/static | alpine | Alpine adds ~7MB and a shell (attack surface). Distroless is zero-runtime for static Go binaries. |
| Container base | distroless/static | scratch | `scratch` works but distroless/nonroot adds UID/GID setup and CA certs. No downside to distroless over scratch. |
| Multi-arch containers | dockers_v2 | dockers + docker_manifests | Old split approach requires separate publish step. dockers_v2 unifies build+push. Old approach is being deprecated in v3. |
| Dolt container | dolthub/dolt-sql-server | Custom Dockerfile | DoltHub maintains the official image. No reason to customize. |
| Dep detection | exec.LookPath in Go | Shell script | Shell script is a second artifact to maintain. Detection belongs in the binary users already have. |
| Apple Container | Detect at runtime | Hard-code Docker only | Apple Container is OCI-compatible. Runtime detection covers Docker, Podman, and Apple Container with the same image. |

## CI/CD Integration

GoReleaser integrates with GitHub Actions. The release workflow:

```yaml
# .github/workflows/release.yml
name: Release
on:
  push:
    tags: ['v*']

jobs:
  release:
    runs-on: ubuntu-latest
    permissions:
      contents: write      # GitHub Releases
      packages: write      # ghcr.io push

    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0   # GoReleaser needs full history for changelog

      - uses: actions/setup-go@v5
        with:
          go-version: '1.26'

      - uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      - uses: goreleaser/goreleaser-action@v6
        with:
          version: v2.14.3
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          HOMEBREW_TAP_GITHUB_TOKEN: ${{ secrets.HOMEBREW_TAP_TOKEN }}
```

**HOMEBREW_TAP_GITHUB_TOKEN** must be a separate PAT with write access to the `homebrew-gsd-wired` repo. The default GITHUB_TOKEN only has access to the current repo.

---

## Architecture Decision: Single Binary, Multiple Distribution Channels

The same `gsdw` binary is distributed via:
1. `go install github.com/user/gsd-wired/cmd/gsdw@latest` — for Go developers
2. `brew tap user/gsd-wired && brew install gsdw` — for macOS users via Homebrew
3. Container image `ghcr.io/user/gsdw:latest` — for containerized environments
4. Direct download from GitHub Releases — for Linux users without brew/Go

All channels produce the same binary. GoReleaser manages channels 2, 3, and 4 from a single `.goreleaser.yaml`.

---

## MCP Server SDK (Existing)

| Technology | Version | Purpose | Why | Confidence |
|------------|---------|---------|-----|------------|
| modelcontextprotocol/go-sdk | v1.4.1 | MCP server implementation | Official Go SDK maintained in collaboration with Google. Typed tool handlers with automatic JSON schema inference from Go structs. StdioTransport for Claude Code integration. Supports MCP spec 2025-11-25. | HIGH |

## Beads Integration (Existing)

| Technology | Version | Purpose | Why | Confidence |
|------------|---------|---------|-----|------------|
| github.com/steveyegge/beads | v0.61.0 | Graph persistence API | Go package with `Storage` interface and `Transaction` support. | HIGH |
| github.com/dolthub/driver | v1.83.8 | Embedded Dolt DB access | No server process required for local development mode. | HIGH |

## CLI Framework (Existing)

| Technology | Version | Purpose | Why | Confidence |
|------------|---------|---------|-----|------------|
| spf13/cobra | v1.10.2 | CLI subcommands | Handles the dual-mode binary (MCP server + hook dispatcher + CLI). New `check-deps` subcommand slots in here. | HIGH |

---

## Version Compatibility Matrix

| Component | Version | Notes |
|-----------|---------|-------|
| Go | 1.26.1 | Current project version; GoReleaser v2.14.3 requires Go 1.26 |
| GoReleaser | v2.14.3 | Latest as of 2026-03-09; use `homebrew_casks` not `brews` |
| dolthub/dolt-sql-server | latest (1.x) | Pull `latest`; DoltHub tags every release |
| distroless/static-debian12 | nonroot | Pin by digest in production |
| apple/container | latest | Silicon + macOS 26 only; detect at runtime |
| GoReleaser `dockers_v2` | alpha | In GoReleaser v2.12+; stable enough for production use |

---

## Sources

**GoReleaser:**
- [GoReleaser v2.14 announcement](https://goreleaser.com/blog/goreleaser-v2.14/) — current version, March 2026
- [GoReleaser deprecation notices](https://goreleaser.com/deprecations/) — confirms `brews` deprecated
- [GoReleaser homebrew casks docs](https://goreleaser.com/customization/homebrew_casks/) — current cask config fields
- [GoReleaser dockers_v2 docs](https://goreleaser.com/customization/dockers_v2/) — multi-arch container config
- [GoReleaser homebrew tap repo](https://github.com/goreleaser/homebrew-tap) — real-world tap pattern

**Dolt Container:**
- [dolthub/dolt-sql-server on Docker Hub](https://hub.docker.com/r/dolthub/dolt-sql-server) — official image
- [Dolt Docker documentation](https://docs.dolthub.com/introduction/installation/docker) — environment variables, run commands

**Apple Container:**
- [apple/container GitHub](https://github.com/apple/container) — source, releases
- [Container CLI command reference](https://github.com/apple/container/blob/main/docs/command-reference.md) — exact CLI syntax
- [Apple WWDC25 session: Meet Containerization](https://developer.apple.com/videos/play/wwdc2025/346/) — architectural overview

**Homebrew:**
- [How to Create and Maintain a Tap](https://docs.brew.sh/How-to-Create-and-Maintain-a-Tap) — official Homebrew tap docs
- [GoReleaser homebrew_formulas (deprecated)](https://goreleaser.com/customization/homebrew_formulas/) — confirms deprecation

**Distroless:**
- [gcr.io/distroless/static](https://github.com/GoogleContainerTools/distroless) — base image for static Go binaries
