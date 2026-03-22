# Phase 14: Connectivity - Context

**Gathered:** 2026-03-22
**Status:** Ready for planning

<domain>
## Phase Boundary

gsdw knows how to reach the Dolt server and automatically passes that configuration to every bd command, with graceful handling when the server is unreachable. Delivers `gsdw connect` wizard, `.gsdw/connection.json`, env var injection into bd subprocess calls, health check before graph ops, and remote-to-local fallback. Covers CONN-01 through CONN-06.

</domain>

<decisions>
## Implementation Decisions

### Connection wizard flow
- **D-01:** `gsdw connect` auto-detects first — scans for a running Dolt server (local container on 127.0.0.1:3307). If found, offers to use it with Y/n confirmation.
- **D-02:** When no server found, offers three choices: (1) Start local container, (2) Configure remote host, (3) Cancel.
- **D-03:** Starting local container calls `gsdw container start` logic inline (reuse Phase 13 container.StartArgs).
- **D-04:** Remote host wizard collects: host, port (default 3306), and optional username. Password via `GSDW_DB_PASSWORD` env var — never stored in connection.json.
- **D-05:** Re-running `gsdw connect` with existing config shows current connection status + health, then asks "Reconfigure? [y/N]". Non-destructive by default.

### Health check & error UX
- **D-06:** Health check runs before every graph operation (TCP dial + SQL `SELECT 1`). ~5ms overhead per call, fails fast with clear message instead of cryptic bd errors.
- **D-07:** Error messages are context-aware: detect WHY the connection failed (container not running, port not listening, DNS failure, auth failure) and provide specific remediation steps for each.
- **D-08:** `gsdw doctor` extended with a "Connection" section showing mode, host:port, and SQL ping status using existing [OK]/[WARN]/[FAIL] format.

### Remote fallback behavior
- **D-09:** When remote host is unreachable, prompt developer: "Fall back to local container? [y/N]". Block until response — no auto-proceed, no timeout.
- **D-10:** If developer confirms fallback and local container is not running, start it automatically as part of the fallback flow.
- **D-11:** After successful fallback, ask "Make this the default? [y/N]". If yes, update connection.json to local mode. If no, use local for this session only (remote config preserved).

### Config file structure
- **D-12:** `.gsdw/connection.json` uses nested structure with `active_mode` discriminator. Stores both local and remote configs so switching doesn't lose settings.
- **D-13:** Config shape: `{ "active_mode": "local"|"remote", "local": { "host", "port" }, "remote": { "host", "port", "user" }, "configured": "ISO-8601" }`.
- **D-14:** `connection.json` is gitignored — machine-specific config. Each developer runs `gsdw connect` for their environment.
- **D-15:** Env vars injected into bd subprocess: `BEADS_DOLT_SERVER_HOST` and `BEADS_DOLT_SERVER_PORT` — exact names from beads bug #2073. bd reads these natively.
- **D-16:** Injection point: `internal/graph/client.go` `run()` method at line 86, after existing `BEADS_DIR` injection. Connection config loaded from `.gsdw/connection.json`.

### Claude's Discretion
- TCP dial timeout duration (suggest 2-3 seconds)
- SQL ping query details (SELECT 1 vs SHOW DATABASES)
- Exact error message wording and formatting
- Whether to cache connection config in Client struct or reload per-call
- Test isolation patterns for health check (consistent with 13-01 DetectOpts pattern)
- Atomic write for connection.json (temp + rename pattern from index.go)

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Connection injection point
- `internal/graph/client.go` — `run()` method (line 82-121), `BEADS_DIR` env injection at line 86. This is where `BEADS_DOLT_SERVER_HOST` and `BEADS_DOLT_SERVER_PORT` must be added.

### Container integration (Phase 13 dependency)
- `internal/container/runtime.go` — `Runtime` interface, `DetectRuntime()`, `ContainerConfig` struct. Reuse for fallback container start.
- `internal/cli/container.go` — `startOpts` struct pattern, `runContainerStart()`. Model `connectOpts`/`runConnect()` after this.

### Wizard patterns
- `internal/cli/setup.go` — Interactive wizard pattern with injected I/O. `runSetup()` phases: check → offer → verify → next steps.
- `internal/cli/doctor.go` — Health check rendering with [OK]/[WARN]/[FAIL] format. Extend with Connection section.

### Config file patterns
- `internal/cli/init.go` — `.gsdw/` directory creation, `gsdwConfig` struct, `json.MarshalIndent()`.
- `internal/graph/index.go` — Atomic write pattern (temp + rename) for `SaveConnection()`.

### Command registration
- `internal/cli/root.go` — `AddCommand()` at line 31. Add `NewConnectCmd()` here.

### Requirements
- `.planning/REQUIREMENTS.md` — CONN-01 through CONN-06

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/container/runtime.go` — `DetectRuntime()` + `StartArgs()` for inline container start during wizard/fallback
- `internal/cli/setup.go` — `runSetup()` wizard pattern with injected I/O (io.Reader, io.Writer)
- `internal/cli/container.go` — `startOpts`/`stopOpts` struct pattern with injected dependencies for hermetic testing
- `internal/deps/check.go` — `Status` enum (OK/Warn/Fail) and `CheckResult` type
- `internal/graph/index.go` — `LoadIndex()`/`Save()` patterns for JSON config files

### Established Patterns
- All CLI commands use opts struct with injected functions (detectFn, execFn, etc.) for testability
- Core logic always in separate `runXxx(out io.Writer, opts xxxOpts)` function, not in Cobra RunE
- `doctor.go` `findGsdwDir()` walks up directory tree to locate `.gsdw/` — reuse for connection config lookup

### Integration Points
- `internal/graph/client.go:86` — Single injection point for all env vars passed to bd subprocesses
- `internal/cli/root.go:31` — `NewConnectCmd()` registration
- `internal/cli/doctor.go` — New "Connection" render section
- `.gitignore` — Add `.gsdw/connection.json`

</code_context>

<deferred>
## Deferred Ideas

- Connection pooling or persistent SQL connection (v2 — current approach shells out to bd per call)
- Multi-server configuration with named profiles (v2)
- Connection string in CLAUDE.md for team sharing (v2)
- TLS/SSL connection support for remote Dolt (v2)

</deferred>

---

*Phase: 14-connectivity*
*Context gathered: 2026-03-22*
