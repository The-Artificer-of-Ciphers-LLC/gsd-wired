# Phase 14: Connectivity - Research

**Researched:** 2026-03-22
**Domain:** Go TCP/SQL health checking, interactive CLI wizards, JSON config management, env var injection
**Confidence:** HIGH

## Summary

Phase 14 adds a connection configuration layer between gsdw and the Dolt SQL server. The design is already fully locked in CONTEXT.md — this research validates that all implementation choices are achievable with the standard library plus one new dependency (go-sql-driver/mysql for SQL ping), and maps out exactly where each piece of code lands.

The most important finding is that the SQL `SELECT 1` ping requires adding `github.com/go-sql-driver/mysql v1.9.3` to go.mod. The TCP dial can use stdlib `net.DialTimeout`, but `database/sql.DB.PingContext` needs a registered MySQL driver to open a connection. Every other implementation concern — config file I/O, wizard interaction, env injection, atomic write — is handled by patterns already proven in Phases 12 and 13.

**Primary recommendation:** Follow the container.go/setup.go opts-struct pattern exactly. New files: `internal/cli/connect.go`, `internal/connection/config.go` (or inline in `cli` package). New dependency: `go-sql-driver/mysql v1.9.3`. Touch points: `internal/graph/client.go:86`, `internal/cli/doctor.go`, `internal/cli/root.go:32`, `.gitignore`.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Connection wizard flow**
- D-01: `gsdw connect` auto-detects first — scans for a running Dolt server (local container on 127.0.0.1:3307). If found, offers to use it with Y/n confirmation.
- D-02: When no server found, offers three choices: (1) Start local container, (2) Configure remote host, (3) Cancel.
- D-03: Starting local container calls `gsdw container start` logic inline (reuse Phase 13 container.StartArgs).
- D-04: Remote host wizard collects: host, port (default 3306), and optional username. Password via `GSDW_DB_PASSWORD` env var — never stored in connection.json.
- D-05: Re-running `gsdw connect` with existing config shows current connection status + health, then asks "Reconfigure? [y/N]". Non-destructive by default.

**Health check & error UX**
- D-06: Health check runs before every graph operation (TCP dial + SQL `SELECT 1`). ~5ms overhead per call, fails fast with clear message instead of cryptic bd errors.
- D-07: Error messages are context-aware: detect WHY the connection failed (container not running, port not listening, DNS failure, auth failure) and provide specific remediation steps for each.
- D-08: `gsdw doctor` extended with a "Connection" section showing mode, host:port, and SQL ping status using existing [OK]/[WARN]/[FAIL] format.

**Remote fallback behavior**
- D-09: When remote host is unreachable, prompt developer: "Fall back to local container? [y/N]". Block until response — no auto-proceed, no timeout.
- D-10: If developer confirms fallback and local container is not running, start it automatically as part of the fallback flow.
- D-11: After successful fallback, ask "Make this the default? [y/N]". If yes, update connection.json to local mode. If no, use local for this session only (remote config preserved).

**Config file structure**
- D-12: `.gsdw/connection.json` uses nested structure with `active_mode` discriminator. Stores both local and remote configs so switching doesn't lose settings.
- D-13: Config shape: `{ "active_mode": "local"|"remote", "local": { "host", "port" }, "remote": { "host", "port", "user" }, "configured": "ISO-8601" }`.
- D-14: `connection.json` is gitignored — machine-specific config. Each developer runs `gsdw connect` for their environment.
- D-15: Env vars injected into bd subprocess: `BEADS_DOLT_SERVER_HOST` and `BEADS_DOLT_SERVER_PORT` — exact names from beads bug #2073. bd reads these natively.
- D-16: Injection point: `internal/graph/client.go` `run()` method at line 86, after existing `BEADS_DIR` injection. Connection config loaded from `.gsdw/connection.json`.

### Claude's Discretion
- TCP dial timeout duration (suggest 2-3 seconds)
- SQL ping query details (SELECT 1 vs SHOW DATABASES)
- Exact error message wording and formatting
- Whether to cache connection config in Client struct or reload per-call
- Test isolation patterns for health check (consistent with 13-01 DetectOpts pattern)
- Atomic write for connection.json (temp + rename pattern from index.go)

### Deferred Ideas (OUT OF SCOPE)
- Connection pooling or persistent SQL connection (v2 — current approach shells out to bd per call)
- Multi-server configuration with named profiles (v2)
- Connection string in CLAUDE.md for team sharing (v2)
- TLS/SSL connection support for remote Dolt (v2)
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| CONN-01 | `gsdw connect` configures connection to Dolt server (local container or remote host) | Wizard pattern from setup.go; runConnect(in, out, opts) core function |
| CONN-02 | Connection config stored in `.gsdw/connection.json` (host, port, mode) | ConnectionConfig struct + atomic SaveConnection() following index.go Save() pattern |
| CONN-03 | `internal/graph/client.go` injects `BEADS_DOLT_SERVER_HOST` and `BEADS_DOLT_SERVER_PORT` env vars on every bd exec when container mode is configured | Line 86 in run() — append after BEADS_DIR; load connection.json from findGsdwDir() |
| CONN-04 | Health check confirms Dolt server is reachable before proceeding | Two-phase: net.DialTimeout (TCP) + db.PingContext (SQL); requires go-sql-driver/mysql v1.9.3 |
| CONN-05 | Remote host connectivity with reachability check and common error troubleshooting | Same two-phase health check; D-07 error classification by failure type |
| CONN-06 | Automatic fallback from unreachable remote to local container with developer confirmation | D-09/D-10/D-11 fallback flow; reuses container.StartArgs() |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `net` (stdlib) | Go 1.26.1 | TCP dial for health check phase 1 | No dependency; `net.DialTimeout` is the idiomatic fast reachability check |
| `database/sql` (stdlib) | Go 1.26.1 | SQL ping for health check phase 2 | Standard Go SQL interface; `DB.PingContext` is the canonical connectivity test |
| `github.com/go-sql-driver/mysql` | v1.9.3 | MySQL wire protocol driver (Dolt is MySQL-compatible) | Only pure-Go MySQL driver; required to register `mysql` driver for `database/sql.Open` |
| `encoding/json` (stdlib) | Go 1.26.1 | connection.json marshal/unmarshal | Already used throughout codebase |
| `bufio` (stdlib) | Go 1.26.1 | Interactive prompt reading in wizard | Already used in setup.go |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `os` (stdlib) | Go 1.26.1 | Atomic write (temp + rename), env var reads | SaveConnection() and GSDW_DB_PASSWORD lookup |
| `fmt` (stdlib) | Go 1.26.1 | All output formatting | Every output line |
| `time` (stdlib) | Go 1.26.1 | ISO-8601 timestamp in config, dial timeout | `time.RFC3339` for configured field |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| go-sql-driver/mysql | dolthub/go-doltcore | go-doltcore is the Dolt-specific library but much heavier (entire Dolt engine); mysql driver is purpose-fit |
| go-sql-driver/mysql | vitess.io/vitess | vitess MySQL driver is large, enterprise-oriented; overkill for a single ping |
| TCP + SQL two-phase | TCP-only | TCP confirms port is listening, not that Dolt is healthy (could be nginx, etc.); SQL ping confirms Dolt itself responds |

**Installation:**
```bash
go get github.com/go-sql-driver/mysql@v1.9.3
```

**Version verification:** Confirmed via `go list -m -json github.com/go-sql-driver/mysql@latest` → v1.9.3, published 2025-06-13.

## Architecture Patterns

### Recommended File Layout for Phase 14
```
internal/
├── connection/
│   ├── config.go          # ConnectionConfig struct, LoadConnection(), SaveConnection()
│   └── config_test.go     # Round-trip and atomic write tests
├── cli/
│   ├── connect.go         # NewConnectCmd(), connectOpts struct, runConnect()
│   └── connect_test.go    # Hermetic wizard tests with injected I/O and fns
│   └── doctor.go          # EXTEND: add Connection section to renderDoctor()
│   └── doctor_test.go     # EXTEND: add Connection section tests
├── graph/
│   └── client.go          # EXTEND: run() at line 86, inject BEADS_DOLT_SERVER_* env vars
```

**Rationale for separate `internal/connection/` package:** The config struct and load/save logic are needed by both `internal/cli` (connect.go, doctor.go) and `internal/graph` (client.go). Placing it in a shared package avoids circular imports. Placing it inside `cli` would make graph import cli, which is wrong directionally. Placing it inside `graph` would work but mixes concerns.

### Pattern 1: connectOpts Struct with Injected Dependencies
**What:** Mirror the `startOpts`/`stopOpts` pattern from container.go — all I/O and side-effectful functions are injected fields, not called directly.
**When to use:** Every CLI command that needs hermetic tests without network calls or file system state.

```go
// Source: internal/cli/container.go (adapted)
type connectOpts struct {
    in  io.Reader
    out io.Writer

    // detectServerFn checks if a Dolt server is already listening.
    // Returns (host, port) if found, error if not.
    detectServerFn func(host, port string) error

    // healthCheckFn runs TCP+SQL ping against host:port.
    healthCheckFn func(host, port, user, password string) error

    // loadConfigFn loads .gsdw/connection.json from gsdwDir.
    loadConfigFn func(gsdwDir string) (*connection.Config, error)

    // saveConfigFn saves config atomically.
    saveConfigFn func(gsdwDir string, cfg *connection.Config) error

    // startContainerFn starts local container (reuses container start logic).
    startContainerFn func() error

    // findGsdwDirFn locates .gsdw/ by walking up.
    findGsdwDirFn func() string
}
```

### Pattern 2: ConnectionConfig Struct with Atomic Save
**What:** JSON struct mirroring D-13 config shape. Save uses temp+rename pattern from index.go.

```go
// Source: internal/graph/index.go Save() pattern (adapted)
type Config struct {
    ActiveMode string      `json:"active_mode"` // "local" or "remote"
    Local      LocalConfig `json:"local"`
    Remote     RemoteConfig `json:"remote"`
    Configured string      `json:"configured"` // RFC3339
}

type LocalConfig struct {
    Host string `json:"host"` // "127.0.0.1"
    Port string `json:"port"` // "3307"
}

type RemoteConfig struct {
    Host string `json:"host"`
    Port string `json:"port"` // default "3306"
    User string `json:"user"` // optional
}

func SaveConnection(gsdwDir string, cfg *Config) error {
    path := filepath.Join(gsdwDir, "connection.json")
    tmp := path + ".tmp"
    data, err := json.MarshalIndent(cfg, "", "  ")
    if err != nil {
        return fmt.Errorf("connection config marshal: %w", err)
    }
    if err := os.WriteFile(tmp, data, 0644); err != nil {
        return fmt.Errorf("connection config write temp: %w", err)
    }
    return os.Rename(tmp, path)
}
```

### Pattern 3: Two-Phase Health Check
**What:** TCP dial first (fast, no driver required), then SQL ping if TCP succeeds (confirms Dolt itself answers).
**When to use:** Before every graph operation (called from `run()` in client.go) and in `gsdw doctor`.

```go
// Source: net.DialTimeout stdlib + database/sql.DB.PingContext
func CheckConnectivity(host, port, user, password string, timeout time.Duration) error {
    addr := net.JoinHostPort(host, port)

    // Phase 1: TCP reachability (~1ms for local, fails fast for unreachable)
    conn, err := net.DialTimeout("tcp", addr, timeout)
    if err != nil {
        return classifyTCPError(err, host, port)
    }
    conn.Close()

    // Phase 2: SQL ping (confirms Dolt is answering, not just a port open)
    dsn := buildDSN(user, password, host, port)
    db, err := sql.Open("mysql", dsn)
    if err != nil {
        return fmt.Errorf("open mysql connection: %w", err)
    }
    defer db.Close()

    ctx, cancel := context.WithTimeout(context.Background(), timeout)
    defer cancel()
    if err := db.PingContext(ctx); err != nil {
        return classifyMySQLError(err)
    }
    return nil
}
```

**Timeout recommendation (Claude's Discretion):** 2 seconds. Rationale: local container responds in <5ms, so 2s gives ample margin for a loaded machine without making the failure path feel slow. 5s would be excessive for a developer tool.

**SQL ping query (Claude's Discretion):** Use `db.PingContext()` rather than a raw `SELECT 1`. `PingContext` is the canonical Go database/sql connectivity test — it internally sends a minimal packet and is driver-aware. `SELECT 1` requires a full query execution path; PingContext is lighter.

### Pattern 4: Error Classification by Failure Type (D-07)

```go
// Classify TCP errors to produce actionable messages
func classifyTCPError(err error, host, port string) error {
    errStr := err.Error()
    switch {
    case strings.Contains(errStr, "connection refused"):
        return fmt.Errorf(
            "connection refused on %s:%s\n"+
            "  Dolt server is not running.\n"+
            "  Fix: run 'gsdw container start' to start the local container.",
            host, port)
    case strings.Contains(errStr, "no such host"), strings.Contains(errStr, "lookup"):
        return fmt.Errorf(
            "DNS resolution failed for host %q\n"+
            "  Check that the hostname is spelled correctly.\n"+
            "  Fix: run 'gsdw connect' to reconfigure the remote host.",
            host)
    case strings.Contains(errStr, "i/o timeout"), strings.Contains(errStr, "deadline"):
        return fmt.Errorf(
            "connection timed out to %s:%s\n"+
            "  The host may be unreachable or behind a firewall.\n"+
            "  Fix: check VPN/firewall, or run 'gsdw connect' to switch to local mode.",
            host, port)
    default:
        return fmt.Errorf("TCP connect to %s:%s failed: %w", host, port, err)
    }
}
```

### Pattern 5: env var injection in client.go run()

```go
// Source: internal/graph/client.go line 86 (existing pattern)
// EXTEND: add connection env vars after BEADS_DIR injection
func (c *Client) run(ctx context.Context, args ...string) ([]byte, error) {
    args = append(args, "--json")
    cmd := exec.CommandContext(ctx, c.bdPath, args...)

    // Existing injection:
    envVars := []string{"BEADS_DIR=" + c.beadsDir}

    // New injection (Phase 14):
    if c.connConfig != nil {
        host, port := c.connConfig.ActiveHostPort()
        envVars = append(envVars,
            "BEADS_DOLT_SERVER_HOST="+host,
            "BEADS_DOLT_SERVER_PORT="+port,
        )
    }
    cmd.Env = append(os.Environ(), envVars...)
    // ...
}
```

**Caching decision (Claude's Discretion):** Cache the `*connection.Config` in the `Client` struct — loaded once at `NewClient()` construction, not reloaded per-call. Rationale: connection.json changes only when `gsdw connect` runs, not between bd calls within a session. Reloading per-call adds ~1ms file I/O for zero benefit in practice.

### Anti-Patterns to Avoid
- **Storing passwords in connection.json:** D-04 is explicit — password only from `GSDW_DB_PASSWORD` env var, never in config file. The `dsn` builder reads `os.Getenv("GSDW_DB_PASSWORD")` at health check time.
- **Auto-proceeding on remote fallback:** D-09 requires blocking on user confirmation. Never add a timeout or default-yes behavior — this is a destructive-ish choice (switching servers mid-session).
- **Running health check in `doctor.go` with network timeout blocking the display:** Doctor should use a short timeout (2s) and show `[WARN] connection not configured` if no connection.json exists, rather than hanging.
- **Using `os.WriteFile` directly for connection.json:** Use temp+rename (D-16 / atomic write pattern). Plain WriteFile leaves a partially-written file on crash.
- **Importing `internal/cli` from `internal/graph`:** Would create a circular dependency. Use `internal/connection` as the shared package.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| MySQL wire protocol | Custom TCP packet parsing | `go-sql-driver/mysql` + `database/sql` | MySQL protocol is complex (auth handshake, capabilities negotiation, packet framing); driver handles all of it |
| DNS failure detection | String-parsing custom resolver | `net.DialTimeout` error messages + `strings.Contains` | stdlib's error messages are consistent enough; no need for custom DNS lookup |
| Atomic file writes | Custom write + fsync | `os.WriteFile` + `os.Rename` (same filesystem) | Rename is atomic on POSIX filesystems for same-filesystem moves — already proven in index.go |
| Interactive prompts | Custom terminal raw mode | `bufio.NewReader(in).ReadString('\n')` | Already used in setup.go; handles piped input in tests; no need for a readline library |

**Key insight:** The codebase already has all the tools needed except the MySQL driver. Adding that one dependency unlocks the full SQL ping without any hand-rolled protocol work.

## Common Pitfalls

### Pitfall 1: SQL Open Does Not Connect
**What goes wrong:** `sql.Open("mysql", dsn)` returns no error even if the server is unreachable. Developers call Open, check err, and conclude the server is reachable — but Open only validates the DSN format.
**Why it happens:** `database/sql.Open` is designed to be lazy. The actual connection is established on the first use (ping, query, etc.).
**How to avoid:** Always call `db.PingContext(ctx)` after `sql.Open`. This is what forces the actual connection attempt.
**Warning signs:** Health check function that calls `sql.Open` but not `db.Ping` or `db.PingContext`.

### Pitfall 2: net.Dial vs net.DialTimeout on Unreachable Remote
**What goes wrong:** `net.Dial("tcp", addr)` has no timeout. On an unreachable remote host (firewall drop, not reject), Dial can block for the OS TCP timeout — up to 2 minutes on some systems.
**Why it happens:** Connection refused is instant (RST packet received). Firewall drop is silent — the OS waits for retransmit before giving up.
**How to avoid:** Always use `net.DialTimeout` with an explicit timeout (2 seconds recommended). This is the discriminator between "port closed" (instant error) and "host unreachable/filtered" (timeout error).
**Warning signs:** Health check using `net.Dial` instead of `net.DialTimeout`.

### Pitfall 3: go-sql-driver/mysql import side effect required
**What goes wrong:** `database/sql.Open("mysql", dsn)` panics with "unknown driver mysql" even after adding the dependency to go.mod.
**Why it happens:** `database/sql` uses a driver registry. The mysql driver only registers itself when its `init()` function runs, which only happens if the package is imported (even with a blank import).
**How to avoid:** Add `import _ "github.com/go-sql-driver/mysql"` (blank import) in the file that calls `sql.Open`. This triggers the `init()` registration.
**Warning signs:** Missing blank import in the health check file.

### Pitfall 4: Connection.json found in wrong directory
**What goes wrong:** `findGsdwDir()` walks up from cwd and finds a `.gsdw/` in a parent directory. The connection.json there belongs to a different project. Client injects wrong host/port into bd.
**Why it happens:** The walk-up is intentional for the config directory, but connection.json is machine+project-specific.
**How to avoid:** `client.go` should locate `.gsdw/connection.json` relative to `c.beadsDir` (the known project root), not by walking up from cwd. `beadsDir` is already the project root anchor. Use `filepath.Join(filepath.Dir(c.beadsDir), ".gsdw", "connection.json")`.
**Warning signs:** Using `findGsdwDir()` in client.go instead of deriving path from `c.beadsDir`.

### Pitfall 5: wizard runs health check inline with output gap
**What goes wrong:** The wizard prints "Checking for running Dolt server..." then blocks for 2 seconds on TCP dial before printing the result. Looks frozen.
**Why it happens:** `net.DialTimeout` blocks for the full timeout on unreachable hosts.
**How to avoid:** Print the status line before the check, then update with [OK]/[FAIL] suffix after. Or accept that the gap is only 2 seconds and document it in the output ("Checking (2s timeout)...").
**Warning signs:** Any health check blocking in a wizard without user feedback.

### Pitfall 6: MySQL DSN format for Dolt
**What goes wrong:** Using the wrong DSN format causes authentication or connection failures that look like server errors.
**Why it happens:** go-sql-driver/mysql DSN format is: `user:password@tcp(host:port)/dbname`. Empty password: `user@tcp(host:port)/`. No user: `tcp(host:port)/`. Database name must exist or be empty string.
**How to avoid:** Use `net/url` or construct DSN carefully. For health check, connect without selecting a database (empty dbname) to avoid "unknown database" errors.

```go
// Correct DSN construction for health check
func buildDSN(user, password, host, port string) string {
    // go-sql-driver/mysql format: [user[:password]@]protocol(address)[/dbname]
    auth := ""
    if user != "" && password != "" {
        auth = url.QueryEscape(user) + ":" + url.QueryEscape(password) + "@"
    } else if user != "" {
        auth = url.QueryEscape(user) + "@"
    }
    return fmt.Sprintf("%stcp(%s)/", auth, net.JoinHostPort(host, port))
}
```

## Code Examples

### Example 1: Loading connection.json (with missing-file grace)
```go
// Source: Derived from internal/graph/index.go LoadIndex() pattern
func LoadConnection(gsdwDir string) (*Config, error) {
    path := filepath.Join(gsdwDir, "connection.json")
    data, err := os.ReadFile(path)
    if err != nil {
        if os.IsNotExist(err) {
            return nil, nil // no config yet — not an error
        }
        return nil, fmt.Errorf("connection config read: %w", err)
    }
    var cfg Config
    if err := json.Unmarshal(data, &cfg); err != nil {
        return nil, fmt.Errorf("connection config unmarshal: %w", err)
    }
    return &cfg, nil
}
```

### Example 2: Doctor Connection section
```go
// Source: internal/cli/doctor.go renderDoctor() pattern (extend)
func renderConnectionSection(w io.Writer, cfg *connection.Config, healthErr error) {
    fmt.Fprintln(w, "Connection:")
    if cfg == nil {
        fmt.Fprintln(w, "  [WARN] Not configured — run gsdw connect")
        return
    }
    host, port := cfg.ActiveHostPort()
    fmt.Fprintf(w, "  Mode:    %s\n", cfg.ActiveMode)
    fmt.Fprintf(w, "  Address: %s:%s\n", host, port)
    if healthErr != nil {
        fmt.Fprintf(w, "  [FAIL] Dolt unreachable: %v\n", healthErr)
    } else {
        fmt.Fprintln(w, "  [OK]   Dolt server responding")
    }
}
```

### Example 3: Wizard auto-detect flow (D-01/D-02)
```go
// Source: Pattern derived from internal/cli/setup.go runSetup() phases
func runConnect(opts connectOpts) error {
    reader := bufio.NewReader(opts.in)

    // Phase 1: Check existing config.
    gsdwDir := opts.findGsdwDirFn()
    if gsdwDir == "" {
        return fmt.Errorf("no .gsdw/ directory found — run gsdw init first")
    }

    existing, _ := opts.loadConfigFn(gsdwDir)
    if existing != nil {
        // D-05: Show status and ask to reconfigure.
        host, port := existing.ActiveHostPort()
        fmt.Fprintf(opts.out, "Current connection: %s (%s:%s)\n", existing.ActiveMode, host, port)
        healthErr := opts.healthCheckFn(host, port, existing.Remote.User, "")
        if healthErr != nil {
            fmt.Fprintf(opts.out, "[FAIL] %v\n", healthErr)
        } else {
            fmt.Fprintln(opts.out, "[OK]   Server responding")
        }
        fmt.Fprint(opts.out, "Reconfigure? [y/N]: ")
        line, _ := reader.ReadString('\n')
        if strings.TrimSpace(strings.ToLower(line)) != "y" {
            return nil
        }
    }

    // Phase 2: Auto-detect local server (D-01).
    fmt.Fprintln(opts.out, "Scanning for running Dolt server on 127.0.0.1:3307...")
    if err := opts.detectServerFn("127.0.0.1", "3307"); err == nil {
        fmt.Fprintln(opts.out, "Found local Dolt server on 127.0.0.1:3307")
        fmt.Fprint(opts.out, "Use it? [Y/n]: ")
        line, _ := reader.ReadString('\n')
        if choice := strings.TrimSpace(strings.ToLower(line)); choice == "" || choice == "y" {
            return saveLocalConfig(opts, gsdwDir, "127.0.0.1", "3307")
        }
    }

    // Phase 3: No server found — offer choices (D-02).
    // ... (start container, configure remote, cancel)
}
```

### Example 4: Blank import for MySQL driver registration
```go
// Source: go-sql-driver/mysql README — required side-effect import
package connection

import (
    "context"
    "database/sql"
    "fmt"
    "net"
    "time"

    _ "github.com/go-sql-driver/mysql" // registers "mysql" driver via init()
)
```

### Example 5: env var injection in client.go run()
```go
// Source: internal/graph/client.go line 86 — append after existing BEADS_DIR
envVars := []string{"BEADS_DIR=" + c.beadsDir}
if c.connConfig != nil {
    host, port := c.connConfig.ActiveHostPort()
    envVars = append(envVars,
        "BEADS_DOLT_SERVER_HOST="+host,
        "BEADS_DOLT_SERVER_PORT="+port,
    )
}
cmd.Env = append(os.Environ(), envVars...)
```

### Example 6: ActiveHostPort() helper on Config
```go
// Returns the currently active host:port based on active_mode.
func (c *Config) ActiveHostPort() (host, port string) {
    switch c.ActiveMode {
    case "remote":
        return c.Remote.Host, c.Remote.Port
    default: // "local" or unset
        h := c.Local.Host
        if h == "" { h = "127.0.0.1" }
        p := c.Local.Port
        if p == "" { p = "3307" }
        return h, p
    }
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Manual env var export before bd | gsdw auto-injects env vars | Phase 14 | Developer never manually exports BEADS_DOLT_SERVER_* again |
| Cryptic bd connection errors | Context-aware error messages with remediation | Phase 14 | Developer sees "run gsdw container start" not "dial tcp: connection refused" |

**Project-specific note:** beads bug #2073 is why env var injection is the required approach. The normal config.yaml port setting in beads does not work for the Dolt server host/port — env vars are the only reliable path. This is a known upstream limitation.

## Open Questions

1. **Does `gsdw connect` need to run a health check before every `bd` call, or only on explicit user request?**
   - What we know: D-06 says "before every graph operation" — this means in `client.go run()`.
   - What's unclear: This doubles the latency of every graph call (~2-5ms TCP + SQL ping overhead). For a tool that already exits quickly, this is acceptable but worth measuring.
   - Recommendation: Implement as specified (D-06). If performance becomes an issue in practice, the connection check can be promoted to session-start only in v2.

2. **How should client.go handle missing connection.json?**
   - What we know: D-16 says inject when connection.json exists; D-15 says exact env var names.
   - What's unclear: If connection.json is absent (developer hasn't run `gsdw connect` yet), should client.go error, warn, or silently skip injection?
   - Recommendation: Silent skip — do not inject the env vars. bd without BEADS_DOLT_SERVER_* falls back to its default behavior (reads config.yaml). This preserves backward compatibility for developers who haven't run `gsdw connect` yet.

3. **Where does client.go locate connection.json?**
   - What we know: c.beadsDir is the project root anchor.
   - What's unclear: The path must be `filepath.Dir(c.beadsDir) + "/.gsdw/connection.json"` or `filepath.Join(c.beadsDir, "../.gsdw/connection.json")` — but this depends on whether .beads/ and .gsdw/ are always siblings.
   - Recommendation: They are always siblings (both in project root). Use `filepath.Join(filepath.Dir(c.beadsDir), ".gsdw", "connection.json")`. This is deterministic and doesn't require walking up.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib) — `go test ./...` |
| Config file | none (no pytest.ini/jest.config) |
| Quick run command | `go test ./internal/cli/... ./internal/connection/... ./internal/graph/...` |
| Full suite command | `go test ./...` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| CONN-01 | `gsdw connect` wizard flow (auto-detect, offer choices, collect remote config) | unit | `go test ./internal/cli/... -run TestConnect` | ❌ Wave 0 |
| CONN-01 | Re-running with existing config shows status + asks to reconfigure | unit | `go test ./internal/cli/... -run TestConnectExisting` | ❌ Wave 0 |
| CONN-02 | ConnectionConfig struct round-trips through JSON correctly | unit | `go test ./internal/connection/... -run TestConnectionConfigRoundTrip` | ❌ Wave 0 |
| CONN-02 | SaveConnection uses atomic write (temp + rename) | unit | `go test ./internal/connection/... -run TestSaveConnectionAtomic` | ❌ Wave 0 |
| CONN-03 | client.go run() injects BEADS_DOLT_SERVER_HOST + PORT when config present | unit | `go test ./internal/graph/... -run TestClientRunInjectsConnEnvVars` | ❌ Wave 0 |
| CONN-03 | client.go run() does not inject env vars when no connection.json | unit | `go test ./internal/graph/... -run TestClientRunNoConfigNoInjection` | ❌ Wave 0 |
| CONN-04 | Health check passes when server is reachable (TCP + SQL ping both succeed) | unit | `go test ./internal/connection/... -run TestCheckConnectivityOK` | ❌ Wave 0 |
| CONN-04 | Health check fails with classifiable error when server unreachable | unit | `go test ./internal/connection/... -run TestCheckConnectivityRefused` | ❌ Wave 0 |
| CONN-05 | Remote host DNS failure produces actionable error message | unit | `go test ./internal/connection/... -run TestClassifyTCPError_DNS` | ❌ Wave 0 |
| CONN-05 | Remote host timeout produces actionable error message | unit | `go test ./internal/connection/... -run TestClassifyTCPError_Timeout` | ❌ Wave 0 |
| CONN-06 | Fallback prompt blocks on user input, proceeds only on Y | unit | `go test ./internal/cli/... -run TestConnectFallback_UserConfirms` | ❌ Wave 0 |
| CONN-06 | Fallback with N preserves remote config and cancels | unit | `go test ./internal/cli/... -run TestConnectFallback_UserDeclines` | ❌ Wave 0 |
| D-08 | doctor renderDoctor includes Connection section | unit | `go test ./internal/cli/... -run TestRenderDoctor_ConnectionSection` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./internal/cli/... ./internal/connection/... ./internal/graph/...`
- **Per wave merge:** `go test ./...`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `internal/connection/config.go` — ConnectionConfig struct, LoadConnection, SaveConnection, CheckConnectivity, classifyTCPError
- [ ] `internal/connection/config_test.go` — covers CONN-02, CONN-04, CONN-05
- [ ] `internal/cli/connect.go` — NewConnectCmd, connectOpts, runConnect
- [ ] `internal/cli/connect_test.go` — covers CONN-01, CONN-06
- [ ] New dependency: `go get github.com/go-sql-driver/mysql@v1.9.3` — required for health check SQL ping

## Sources

### Primary (HIGH confidence)
- Go stdlib `net` package — `net.DialTimeout` API verified via `go doc net.DialTimeout`
- Go stdlib `database/sql` — `DB.PingContext` API verified via `go doc database/sql.DB.PingContext`
- `github.com/go-sql-driver/mysql` — v1.9.3 verified via `go list -m -json` (published 2025-06-13)
- `internal/graph/client.go` — injection point at line 86 read directly from source
- `internal/cli/container.go` — opts struct pattern read directly from source
- `internal/cli/setup.go` — wizard pattern read directly from source
- `internal/cli/doctor.go` — [OK]/[WARN]/[FAIL] rendering pattern read directly from source
- `internal/graph/index.go` — atomic Save() pattern read directly from source
- `.planning/phases/14-connectivity/14-CONTEXT.md` — all decisions read directly

### Secondary (MEDIUM confidence)
- go-sql-driver/mysql blank import requirement — standard Go driver registration pattern, consistent with all Go database/sql documentation

### Tertiary (LOW confidence)
- None — all claims are backed by direct code inspection or stdlib verification.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — verified via `go list -m` and `go doc`; go-sql-driver is the canonical MySQL driver for Go
- Architecture: HIGH — all patterns derived directly from existing codebase source files (client.go:86, index.go Save(), container.go opts struct, setup.go wizard phases)
- Pitfalls: HIGH — sql.Open laziness, net.Dial timeout behavior, and driver blank import are well-documented Go patterns; verified against stdlib documentation

**Research date:** 2026-03-22
**Valid until:** 2026-06-22 (go-sql-driver/mysql updates infrequently; all other stdlib references are stable)
