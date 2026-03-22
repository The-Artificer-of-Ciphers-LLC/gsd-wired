---
phase: 4
slug: hook-integration
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-21
---

# Phase 4 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | existing from Phase 1 |
| **Quick run command** | `go test ./...` |
| **Full suite command** | `go test -v -count=1 -race ./...` |
| **Estimated runtime** | ~20 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./...`
- **After every plan wave:** Run `go test -v -count=1 -race ./...`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 25 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| TBD | TBD | TBD | INFRA-05 | unit+integration | `go test ./internal/hook/...` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | INFRA-06 | unit | `go test ./internal/hook/...` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | INFRA-07 | unit | `go test ./internal/hook/...` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | INFRA-08 | unit | `go test ./internal/hook/...` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] Hook handler tests in `internal/hook/`
- [ ] hookState tests with graph client integration

*Existing test infrastructure covers framework needs.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| SessionStart injects context into live Claude Code session | INFRA-05 | Requires running Claude Code | Start session with plugin installed, verify context appears |
| PreCompact saves state before context compression | INFRA-06 | Requires Claude Code compaction event | Trigger compaction, check .gsdw/ for snapshot |
| Latency budgets met under real Dolt load | ALL | Requires live database | Profile hook execution with real bd operations |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 25s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
