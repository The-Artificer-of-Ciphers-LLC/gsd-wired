---
phase: 14
slug: connectivity
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-22
---

# Phase 14 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none — existing infrastructure |
| **Quick run command** | `go test ./internal/cli/ ./internal/graph/ -run Connect -count=1` |
| **Full suite command** | `go test ./... -count=1` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/cli/ ./internal/graph/ -run Connect -count=1`
- **After every plan wave:** Run `go test ./... -count=1`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 14-01-01 | 01 | 1 | CONN-02 | unit | `go test ./internal/cli/ -run TestConnectionConfig` | ❌ W0 | ⬜ pending |
| 14-01-02 | 01 | 1 | CONN-03 | unit | `go test ./internal/graph/ -run TestEnvInjection` | ❌ W0 | ⬜ pending |
| 14-01-03 | 01 | 1 | CONN-04 | unit | `go test ./internal/cli/ -run TestHealthCheck` | ❌ W0 | ⬜ pending |
| 14-02-01 | 02 | 2 | CONN-01 | unit | `go test ./internal/cli/ -run TestConnectWizard` | ❌ W0 | ⬜ pending |
| 14-02-02 | 02 | 2 | CONN-05 | unit | `go test ./internal/cli/ -run TestRemoteConnect` | ❌ W0 | ⬜ pending |
| 14-02-03 | 02 | 2 | CONN-06 | unit | `go test ./internal/cli/ -run TestFallback` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/cli/connect_test.go` — stubs for CONN-01, CONN-02, CONN-05, CONN-06
- [ ] `internal/graph/client_test.go` — stubs for CONN-03 (env injection)
- [ ] `internal/cli/health_test.go` — stubs for CONN-04 (health check)

*Existing go test infrastructure covers framework needs.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Interactive wizard prompts | CONN-01 | Requires stdin/stdout interaction | Run `gsdw connect` and walk through wizard |
| Doctor connection section | CONN-04 | Visual output verification | Run `gsdw doctor` and verify Connection section renders |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
