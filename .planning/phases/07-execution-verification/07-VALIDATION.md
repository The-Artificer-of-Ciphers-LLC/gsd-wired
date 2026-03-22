---
phase: 7
slug: execution-verification
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-21
---

# Phase 7 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Quick run command** | `go test ./...` |
| **Full suite command** | `go test -v -count=1 -race ./...` |
| **Estimated runtime** | ~35 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./...`
- **After every plan wave:** Run `go test -v -count=1 -race ./...`
- **Max feedback latency:** 40 seconds

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Parallel Task() agents execute wave tasks | EXEC-01 | Requires live Claude Code | Run /gsd-wired:execute, verify agents spawn |
| Git commits per task with plan ID | EXEC-05 | Requires real git repo state | Complete a task, check git log for plan ID |
| Verification creates remediation tasks | VRFY-03 | Requires failed verification | Inject a gap, verify new task beads created |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify
- [ ] Feedback latency < 40s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
