---
status: complete
phase: 14-connectivity
source: [14-01-SUMMARY.md, 14-02-SUMMARY.md]
started: 2026-03-22T05:00:00Z
updated: 2026-03-22T05:30:00Z
---

## Current Test

[testing complete]

## Tests

### 1. Connection config round-trip
expected: Create a `.gsdw/connection.json` file manually with local mode config. Run `gsdw doctor`. The Connection section should appear showing Mode: local, Address: 127.0.0.1:3307.
result: pass

### 2. gsdw connect auto-detect (no server running)
expected: Run `gsdw connect` with no Dolt server running. Should show "Scanning for Dolt server..." then "No Dolt server found" and present three choices: (1) Start local container, (2) Configure remote host, (3) Cancel.
result: pass

### 3. gsdw connect remote configuration
expected: Run `gsdw connect`, select "Configure remote host". Should prompt for host, port (default 3306), and optional username. After entering values, should test the connection and write `.gsdw/connection.json` with the remote config.
result: pass

### 4. gsdw connect reconfigure flow
expected: With an existing `.gsdw/connection.json`, run `gsdw connect`. Should display current connection status (mode, host:port, health) and ask "Reconfigure? [y/N]". Pressing N or Enter should exit without changes.
result: pass

### 5. gsdw doctor Connection section
expected: Run `gsdw doctor` with a valid `.gsdw/connection.json`. Output should include a "Connection:" section with [OK], [WARN], or [FAIL] status indicators for mode, address, and SQL ping — consistent with the existing Dependencies section format.
result: pass

### 6. Env var injection into bd
expected: After running `gsdw connect` with a local config, any subsequent `bd` command invoked by gsdw should receive `BEADS_DOLT_SERVER_HOST` and `BEADS_DOLT_SERVER_PORT` as environment variables. Verify by checking that `gsdw ready` or `gsdw status` attempts to connect to the configured host:port.
result: pass

### 7. Health check error messages
expected: Configure a remote connection to an unreachable host (e.g., `nonexistent.example.com:3307`). Run any graph command. Should get a clear error like "Cannot reach Dolt at nonexistent.example.com:3307 — Host not found. Check hostname." with troubleshooting guidance.
result: pass

### 8. Binary compiles and all tests pass
expected: Run `go build -o /dev/null ./cmd/gsdw` (should compile without errors) and `go test ./... -count=1` (all tests should pass, including the new connection and connect packages).
result: pass

## Summary

total: 8
passed: 8
issues: 0
pending: 0
skipped: 0
blocked: 0

## Gaps

[none]
