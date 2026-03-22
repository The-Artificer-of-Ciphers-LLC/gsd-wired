---
phase: 2
slug: graph-primitives
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-21
---

# Phase 2 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | existing from Phase 1 |
| **Quick run command** | `go test ./...` |
| **Full suite command** | `go test -v -count=1 -race ./...` |
| **Estimated runtime** | ~10 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./...`
- **After every plan wave:** Run `go test -v -count=1 -race ./...`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| TBD | TBD | TBD | INFRA-03 | unit+integration | `go test ./internal/graph/...` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | MAP-01 | unit | `go test ./internal/graph/...` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | MAP-02 | unit | `go test ./internal/graph/...` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | MAP-03 | unit | `go test ./internal/graph/...` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | MAP-04 | unit | `go test ./internal/graph/...` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | MAP-05 | unit | `go test ./internal/graph/...` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | MAP-06 | unit | `go test ./internal/graph/...` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/graph/` package — new package for bd wrapper
- [ ] Fake bd binary test helper for unit tests (no real bd/dolt dependency in tests)

*Existing test infrastructure from Phase 1 covers framework needs.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| bd CLI integration with real Dolt database | INFRA-03 | Requires bd + dolt installed and initialized | Run `bd init` in temp dir, exercise gsdw graph operations |
| Wave computation with real dependency graph | MAP-03 | Requires multi-bead graph with deps | Create phase + plans with deps, verify `gsdw ready` output |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
