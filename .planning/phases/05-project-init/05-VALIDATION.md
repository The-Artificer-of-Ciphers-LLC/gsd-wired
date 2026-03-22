---
phase: 5
slug: project-init
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-21
---

# Phase 5 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | existing from Phase 1 |
| **Quick run command** | `go test ./...` |
| **Full suite command** | `go test -v -count=1 -race ./...` |
| **Estimated runtime** | ~25 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./...`
- **After every plan wave:** Run `go test -v -count=1 -race ./...`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| TBD | TBD | TBD | INIT-01 | unit | `go test ./internal/mcp/...` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | INIT-02 | unit | `go test ./internal/mcp/...` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | INIT-03 | unit | `go test ./internal/graph/...` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | INIT-04 | file check | `test -f PROJECT.md` | N/A | ⬜ pending |
| TBD | TBD | TBD | INIT-05 | unit | `go test ./internal/mcp/...` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | CMD-01 | file check | `test -f skills/init/SKILL.md` | N/A | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] MCP tool handler tests for init_project and get_status
- [ ] SKILL.md files for init and status slash commands

*Existing test infrastructure covers framework needs.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| /gsd-wired:init launches questioning flow | CMD-01 | Requires live Claude Code session with plugin | Install plugin, run /gsd-wired:init, verify questions appear |
| Interactive questioning adapts to answers | INIT-02 | Requires Claude conversation | Run full init, verify follow-up questions reference prior answers |
| Status auto-shows on session start | CMD-01 | Requires live SessionStart hook | Start new session, verify dashboard appears |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
