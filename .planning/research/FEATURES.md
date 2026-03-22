# Feature Landscape: Installation & Distribution Toolkit

**Domain:** CLI developer tool — installation UX, dependency detection, container setup, connectivity
**Researched:** 2026-03-21
**Confidence:** HIGH for patterns (well-documented in rustup, brew, flyctl, docker); MEDIUM for Apple Container specifics (macOS 26 requirement narrows audience)
**Milestone scope:** Adding `gsdw setup` and distribution packaging to an existing, complete Go binary (gsdw)

---

## Context

gsdw is a complete Go binary (MCP server + hooks + CLI). Two hard dependencies exist before any project work can happen: `bd` (beads CLI) and `dolt`. Neither has been on PATH for the developer building this. The binary also needs a distribution path (brew tap or `go install`) and optionally a container runtime for Dolt server.

The active requirements (from PROJECT.md) are:
- Brew formula for gsdw binary distribution
- `go install` path end-to-end
- Dependency detection (bd, dolt, Go) with guided install
- Apple Container support for macOS
- Docker/Podman standalone container support
- Drop-in docker-compose fragment (non-destructive)
- Container images for Dolt server + beads
- Connection setup wizard (local or remote, collect host/port)
- Health check and connectivity troubleshooting
- Remote host fallback to local container on failure

---

## Table Stakes

Features users expect from any `gsdw setup` command. Missing these = setup feels broken or amateur.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Binary availability via brew | macOS developer tools are expected to `brew install` — anything else feels like a draft | LOW | GoReleaser + homebrew tap is a one-time config; generates formula on every release |
| Dependency detection before first use | If `bd` or `dolt` is missing, the binary must say so clearly and immediately — not crash with an opaque error | LOW | `exec.LookPath("bd")` and `exec.LookPath("dolt")` in Go; run at startup or setup time |
| Actionable install instructions for missing deps | Users expect "X not found — run this to install it" rather than a Go panic or a URL dump | LOW | Rustup does this for MSVC on Windows; beads docs do this for icu4c on macOS |
| PATH verification after guided install | After installing a dependency, confirm it is now on PATH before continuing | LOW | Re-run `exec.LookPath` after instructing user to install; suggest shell restart if needed |
| `gsdw setup` or `gsdw doctor` command | The "run this to get started" entry point users expect from any developer CLI (brew doctor, kubectl doctor, flyctl doctor) | MEDIUM | Single command that checks all deps, runs health checks, and tells user what to fix |
| Version check on dependencies | Users need to know if their installed bd or dolt version is too old | LOW | Shell out to `bd version`, `dolt version`, parse output, compare against minimum versions |
| Non-destructive compose fragment | Any project with an existing docker-compose.yaml expects `gsdw setup` to extend it, not overwrite it | MEDIUM | Docker Compose merge: generate a `gsdw.compose.yaml` with Dolt service; user runs `docker compose -f docker-compose.yaml -f gsdw.compose.yaml up` |
| Health check command | After setup, user needs a single command to verify everything is working | LOW | Check bd binary exists, dolt binary exists, Dolt server responds on configured host:port |

---

## Differentiators

Features that make `gsdw setup` better than naive "check if binary exists and bail." Not expected by all users, but valued by those who encounter problems.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Container runtime auto-detection (Apple Container vs Docker vs Podman) | Developer tools increasingly need to pick the right container runtime. On Apple silicon with macOS 26+, Apple Container is native and lightweight; on macOS <26 or Intel, Docker/Podman is the path | MEDIUM | Check `container --version` first (Apple Container, macOS 26+, Apple Silicon only); fall back to `docker version`; fall back to `podman version`; warn clearly if none found |
| Guided container choice when multiple runtimes present | Power users may have both Docker and Apple Container installed; wizard should confirm which to use for Dolt server | LOW | Single prompt: "Found Docker and Apple Container. Which should gsdw use for Dolt? [1] Apple Container [2] Docker" |
| Local-or-remote connection wizard | "Is your Dolt server local (container) or remote (separate host)?" is a one-time config that avoids future confusion | MEDIUM | Collect host, port, credentials; write to `~/.config/gsdw/config.toml` or `.beads/gsdw.toml`; remote mode skips container management |
| Remote fallback to local container | If remote Dolt goes unreachable, gsdw can offer to spin up a local container as fallback | HIGH | Complex: requires knowing when a connection failure is transient vs permanent; adds retry/timeout logic; probably v1.x not v1.0 |
| Idempotent setup (re-run safe) | Users re-run setup after upgrade or on a new machine. It should detect what is already done and skip those steps | MEDIUM | Check each precondition before acting; "bd already installed (v1.4.2)" not "installing bd..." on re-run. Pattern: rustup's idempotent toolchain install. |
| Setup dry-run flag | `gsdw setup --dry-run` shows what would be installed/configured without doing it | LOW | Output: "Would install bd via brew", "Would write config to ~/.config/gsdw/config.toml". Matches flyctl/goreleaserdry-run pattern. |
| Structured setup output (step-by-step with status markers) | Prefix each check with `[OK]`, `[WARN]`, `[FAIL]` so users can scan a long setup output and spot issues instantly | LOW | No library needed — just consistent formatting in Go. Pattern: flyctl output, brew doctor output. |

---

## Anti-Features

Features to explicitly not build in this milestone.

| Anti-Feature | Why Avoid | What to Do Instead |
|--------------|-----------|-------------------|
| Automatic bd or dolt installation without user confirmation | Silently modifying PATH or installing system packages is hostile UX; macOS Gatekeeper and user trust require explicit consent | Print the install command, let the user run it. Optionally prompt "Should I run this? [y/N]" for brew installs. |
| Custom container image registry | Maintaining a private registry adds ops overhead; Dolt and beads already publish official images | Reference official images: `dolthub/dolt-sql-server` for Dolt, beads' own published image if it exists |
| Migration from old config schemas | Config format will stabilize after v1 ships; migration tooling before stability wastes time | Write config fresh; on version mismatch, tell user to re-run `gsdw setup` |
| Windows support | gsdw targets macOS-native Claude Code users; Windows adds container runtime complexity (WSL2, Hyper-V) | macOS only for v1; Linux secondary if needed for CI |
| Interactive TUI install wizard | Full terminal UI (bubbletea/lipgloss) for setup is over-engineered; a linear prompt sequence is sufficient | Simple `fmt.Scan` or `bufio.NewReader` prompts for the handful of setup inputs (host, port, container choice) |
| Auto-start Dolt container on every gsdw invocation | Adds 500ms+ startup latency on every CLI invocation; containers should be started once and left running | Document: run `gsdw dolt start` or `docker compose up -d` once; gsdw assumes Dolt is already running |

---

## UX Patterns From Best-In-Class Tools

Concrete patterns observed in rustup, brew, flyctl, nvm, kubectl, and Apple Container:

### Pattern 1: Layered Dependency Detection (rustup / brew doctor)
Run a sequential check for each dependency. Stop at the first blocker. Print what was found and what was not.
```
[OK]  Go 1.22.4 found at /usr/local/go/bin/go
[OK]  bd 1.4.2 found at /opt/homebrew/bin/bd
[FAIL] dolt not found on PATH
      → Install: brew install dolthub/tap/dolt
      → Then re-run: gsdw setup
```
Pattern: `exec.LookPath` for each binary, then `exec.Command(bin, "version")` to get version string.

### Pattern 2: Interactive Customization Menu (rustup-init style)
Before taking any action, show what will be done and give user an out:
```
gsdw will:
  - Start a Dolt container using Docker
  - Write connection config to ~/.config/gsdw/config.toml
  - Register gsdw as a Claude Code plugin

Proceed? [Y/n]
```
Rustup's `InstallOpts::customize` menu is the gold standard — users can change defaults before committing.

### Pattern 3: Container Runtime Priority Order (Docker Desktop style)
Auto-detect available runtimes in priority order:
1. Apple Container (`container --version`) — macOS 26+, Apple Silicon only
2. Docker (`docker version`)
3. Podman (`podman version`)
4. None found — print: "No container runtime found. Install Docker: brew install --cask docker"

Apple Container requirement: macOS 26 (Tahoe) or later, Apple Silicon only. Users on macOS 15 (Sequoia) or Intel Macs must use Docker or Podman. This is a significant platform constraint.

### Pattern 4: Non-Destructive Compose Fragment (Docker Compose merge)
Never write to the user's existing `docker-compose.yaml`. Instead:
1. Generate `gsdw.compose.yaml` with only the Dolt + beads services
2. Instruct user to run: `docker compose -f docker-compose.yaml -f gsdw.compose.yaml up -d`
3. Or if no existing compose file: generate a standalone `gsdw.compose.yaml` and run it directly

Docker Compose merge appends new services without touching existing ones. Only services with colliding names get merged.

### Pattern 5: Connection Config Wizard (flyctl-style)
Sequential prompts, safe defaults:
```
Dolt server location:
  [1] Local container (Docker/Apple Container) — recommended
  [2] Remote host

→ 1

Container runtime: Docker detected
Config saved to: ~/.config/gsdw/config.toml
```
One-time, never asked again unless `gsdw setup --reset`.

### Pattern 6: Doctor Command (brew doctor / kubectl-doctor)
A separate diagnostic command that checks everything without changing anything:
```
gsdw doctor
  [OK]  bd 1.4.2 — minimum 1.2.0
  [OK]  dolt 1.45.0 — minimum 1.30.0
  [OK]  Dolt server reachable at localhost:3306
  [WARN] gsdw plugin not registered in Claude Code settings
         → Run: gsdw plugin install
```
Output is machine-parseable if `--json` flag used. Human-readable by default.

### Pattern 7: Idempotent Re-run (rustup / beads install script)
Each setup step checks current state before acting:
- "bd already at 1.4.2, skipping"
- "Config already exists at ~/.config/gsdw/config.toml, skipping (use --reset to overwrite)"
- "Dolt container already running, skipping"

beads' own install script verifies checksums before overwriting. GoReleaser formulas do not re-install if version matches.

---

## Feature Dependencies

```
[gsdw setup]
    └──checks──> [bd binary on PATH]
    └──checks──> [dolt binary on PATH]
    └──detects──> [container runtime: apple-container | docker | podman]
    └──prompts──> [local vs remote Dolt]
    └──writes──> [~/.config/gsdw/config.toml]

[brew formula]
    └──requires──> [GoReleaser config (.goreleaser.yaml)]
    └──requires──> [homebrew-tap GitHub repo]
    └──produces──> [brew install gsdw/tap/gsdw]

[go install path]
    └──requires──> [module proxy accessible]
    └──note: CGO dependency in bd may block pure go install]

[container image: Dolt server]
    └──uses──> [dolthub/dolt-sql-server — official image]
    └──fragment──> [gsdw.compose.yaml]

[health check / gsdw doctor]
    └──requires──> [config.toml to know host:port]
    └──checks──> [TCP connect to Dolt host:port]
    └──checks──> [bd + dolt versions]
    └──checks──> [Claude Code plugin registration]

[remote fallback to local container]
    └──requires──> [health check proving remote is down]
    └──requires──> [container runtime detected]
    └──complexity: HIGH — defer to v1.x]
```

---

## Complexity Notes by Feature

| Feature | Complexity | Blocker / Risk |
|---------|------------|----------------|
| Brew formula via GoReleaser | LOW | One-time setup; GoReleaser handles formula generation on release |
| `go install` path | LOW-MEDIUM | bd has CGO deps (icu4c, zstd); gsdw itself may be CGO-free if it only shells out to bd |
| Dependency detection (bd, dolt) | LOW | `exec.LookPath` + `exec.Command(bin, "version")` — well-understood Go pattern |
| Guided install instructions | LOW | Static strings with platform-branching (brew vs curl script) |
| Apple Container detection | LOW-MEDIUM | Check `container --version`; but Apple Container requires macOS 26 + Apple Silicon — must gate and warn |
| Docker/Podman detection | LOW | `docker version --format json` or `podman version --format json` |
| Container runtime priority selection | LOW | Try apple-container, then docker, then podman; first success wins |
| Compose fragment generation | LOW | Write a static template YAML; no merge needed if user has no existing compose file |
| Non-destructive compose injection | MEDIUM | Must detect if user has existing compose file and provide two-file command |
| Connection wizard (local vs remote) | MEDIUM | Prompt loop, config write, validate host:port reachable |
| Health check command | LOW | TCP dial to host:port; run bd/dolt version; check Claude plugin manifest |
| Idempotent re-run | MEDIUM | Each step must read current state before acting |
| Remote fallback to local container | HIGH | Retry logic, transient vs permanent failure distinction, container lifecycle management |

---

## MVP Recommendation for This Milestone

Prioritize the critical path to a developer being able to `brew install gsdw` and have it work end-to-end.

**Build first:**
1. GoReleaser config producing brew tap formula (unblocks distribution)
2. Dependency detection for bd + dolt with install instructions (unblocks new users)
3. `gsdw setup` — container runtime detection, compose fragment generation, connection config write
4. `gsdw doctor` — health check with structured output

**Build second:**
5. Apple Container support (gate on macOS 26 + Apple Silicon detection)
6. Idempotent re-run behavior
7. `--dry-run` flag on setup

**Defer:**
8. Remote fallback to local container (HIGH complexity, LOW usage frequency)
9. Podman support (Docker covers the non-Apple Container case for v1)

---

## Sources

- [rustup Installation Process (DeepWiki)](https://deepwiki.com/rust-lang/rustup/5.1-rustup-init-installer) — interactive menu, PATH modification, shell detection
- [rustup.rs — The Rust toolchain installer](https://rustup.rs/) — concurrent downloads, progress feedback
- [Beads Installation Documentation](https://steveyegge.github.io/beads/getting-started/installation) — brew, curl script, CGO deps (icu4c, zstd), bd version verification
- [Dolt Installation Documentation](https://docs.dolthub.com/introduction/installation) — single binary, homebrew, source build
- [Apple Container GitHub](https://github.com/apple/container) — macOS 26+, Apple Silicon only, `container system start`, `brew install --cask container`
- [Apple Container Install Guide (4sysops)](https://4sysops.com/archives/install-apple-container-cli-running-containers-natively-on-macos-15-sequoia-and-macos-26-tahoe/) — system requirements, post-install steps
- [GoReleaser Homebrew Formulas](https://goreleaser.com/customization/homebrew_formulas/) — automated formula generation on release
- [Homebrew Tap Creation Guide](https://docs.brew.sh/How-to-Create-and-Maintain-a-Tap) — repository naming, bottle uploads
- [Docker Compose Merge Documentation](https://docs.docker.com/compose/how-tos/multiple-compose-files/merge/) — non-destructive fragment injection via -f flag
- [CLI UX Patterns (Lucas F. Costa)](https://www.lucasfcosta.com/blog/ux-patterns-cli-tools) — guided onboarding, input validation, human-readable errors
- [exec.LookPath — Go stdlib](https://pkg.go.dev/os/exec#LookPath) — binary detection pattern
- [kubectl-doctor (GitHub)](https://github.com/emirozer/kubectl-doctor) — doctor command pattern, anomaly detection without state changes
- [flyctl Launch Documentation](https://fly.io/docs/flyctl/launch/) — guided setup wizard, config file generation

---
*Feature research for: Installation and distribution toolkit milestone*
*Researched: 2026-03-21*
