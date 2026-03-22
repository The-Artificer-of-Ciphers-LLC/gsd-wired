---
phase: 3
slug: mcp-server
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-21
---

# Phase 3 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | existing from Phase 1 |
| **Quick run command** | `go test ./...` |
| **Full suite command** | `go test -v -count=1 -race ./...` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./...`
- **After every plan wave:** Run `go test -v -count=1 -race ./...`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 20 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| TBD | TBD | TBD | INFRA-02 | unit+integration | `go test ./internal/mcp/...` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | INFRA-10 | unit | `go test ./internal/graph/...` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] MCP tool handler tests in `internal/mcp/`
- [ ] Lazy init tests (sync.Once pattern)
- [ ] Batch mode tests for graph client

*Existing test infrastructure from Phase 1-2 covers framework needs.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| MCP server responds to Claude Code tool calls | INFRA-02 | Requires live Claude Code session | Install plugin, invoke tool via Claude Code |
| Lazy init with real Dolt database | INFRA-02 | Requires bd + dolt running | Start `gsdw serve`, send tool call, verify .beads/ created |
| Batch commit on session boundary | INFRA-10 | Requires real Dolt with multiple operations | Run sequence of tool calls, check dolt log for single commit |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 20s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
