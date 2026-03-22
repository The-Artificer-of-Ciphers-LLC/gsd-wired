# Setup Guide

## Prerequisites

gsd-wired requires these tools on your system:

| Tool | Purpose | Install |
|------|---------|---------|
| **bd** | Beads graph CLI | `go install github.com/beads-project/beads/cmd/bd@latest` |
| **dolt** | SQL database backend | `brew install dolthub/tap/dolt` |
| **Go** 1.26+ | Build tooling | `brew install go` |
| **Container runtime** | Dolt server (optional) | Docker, Podman, or Apple Container |

Run `gsdw check-deps` to see what's installed and what's missing. Run `gsdw setup` for an interactive wizard that guides you through installation.

## Installation

### Homebrew (macOS)

```bash
brew tap The-Artificer-of-Ciphers-LLC/tap
brew install --cask gsdw-cc
```

### Go Install

```bash
go install github.com/The-Artificer-of-Ciphers-LLC/gsd-wired/cmd/gsdw@latest
```

For private repo access, set GOPRIVATE:
```bash
go env -w GOPRIVATE=github.com/The-Artificer-of-Ciphers-LLC/*
```

### Docker

```bash
docker pull ghcr.io/the-artificer-of-ciphers-llc/gsdw:latest
```

### From Source

```bash
git clone https://github.com/The-Artificer-of-Ciphers-LLC/gsd-wired.git
cd gsd-wired
go build -o gsdw ./cmd/gsdw
```

## Project Initialization

### 1. Initialize project structure

```bash
cd your-project
gsdw init
```

Creates:
- `.beads/` — Dolt-backed graph storage (if bd available)
- `.gsdw/config.json` — Project configuration
- `PROJECT.md` — Project template

### 2. Start a Dolt server

```bash
gsdw container start
```

Auto-detects your container runtime (Apple Container > Docker > Podman) and launches a Dolt server on port 3307. Data persists at `.beads/dolt/`.

Options:
- `--port 3307` — Override port (default 3307)
- `--force` — Overwrite existing gsdw.compose.yaml

### 3. Configure connection

```bash
gsdw connect
```

Interactive wizard that:
1. Scans for a running Dolt server on 127.0.0.1:3307
2. If found, offers to use it
3. If not found, offers to start a container or configure a remote host
4. Writes `.gsdw/connection.json`

### 4. Verify health

```bash
gsdw doctor
```

Shows status for:
- **Dependencies:** bd, dolt, Go, container runtime
- **Project:** .beads/ directory, .gsdw/ directory
- **Connection:** mode, address, SQL ping

## Claude Code Plugin Setup

gsd-wired auto-registers as a Claude Code plugin via `.claude-plugin/plugin.json` and `.mcp.json`. After installing gsdw, the plugin should be available in Claude Code sessions.

To verify:
1. Start a Claude Code session in your project directory
2. Type `/gsd-wired:status` — should show project dashboard
3. Type `/gsd-wired:ready` — should show unblocked tasks

## Remote Dolt Server

For team setups with a shared Dolt server:

```bash
gsdw connect
# Select "Configure remote host"
# Enter: host, port, username
```

Password is read from `GSDW_DB_PASSWORD` environment variable (never stored in connection.json).

If the remote server becomes unreachable, gsdw prompts to fall back to a local container.

## Docker Compose Integration

`gsdw container start` generates a `gsdw.compose.yaml` fragment. To integrate with existing docker-compose:

```bash
docker compose -f docker-compose.yml -f gsdw.compose.yaml up
```

gsdw never modifies your existing compose files.
