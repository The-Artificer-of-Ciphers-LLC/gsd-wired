---
phase: 6
slug: research-planning
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-21
---

# Phase 6 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | existing from Phase 1 |
| **Quick run command** | `go test ./...` |
| **Full suite command** | `go test -v -count=1 -race ./...` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./...`
- **After every plan wave:** Run `go test -v -count=1 -race ./...`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 35 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| TBD | TBD | TBD | RSRCH-01 | unit | `go test ./internal/mcp/...` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | RSRCH-02 | unit | `go test ./internal/mcp/...` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | RSRCH-03 | unit | `go test ./internal/mcp/...` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | RSRCH-04 | unit | `go test ./internal/mcp/...` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | PLAN-01 | unit | `go test ./internal/mcp/...` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | PLAN-02 | unit | `go test ./internal/mcp/...` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | PLAN-03 | unit | `go test ./internal/mcp/...` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | PLAN-04 | file check | `grep -q "validate" skills/plan/SKILL.md` | N/A | ⬜ pending |
| TBD | TBD | TBD | CMD-03 | file check | `test -f skills/plan/SKILL.md` | N/A | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] MCP tool handler tests for run_research and create_plan_beads
- [ ] SKILL.md files for research and plan

*Existing test infrastructure covers framework needs.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| 4 parallel research agents spawn and claim beads | RSRCH-02 | Requires live Claude Code Task() API | Run /gsd-wired:research, verify 4 agents spawn |
| Synthesizer auto-triggers after all 4 complete | RSRCH-04 | Requires live agent completion events | Complete research, verify summary produced |
| Plan checker iterates on rejection | PLAN-04 | Requires full plan-check cycle | Run /gsd-wired:plan, inject a gap, verify iteration |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 35s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
