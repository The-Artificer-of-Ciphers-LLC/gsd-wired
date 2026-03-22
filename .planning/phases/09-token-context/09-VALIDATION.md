---
phase: 9
slug: token-context
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-21
---

# Phase 9 — Validation Strategy

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Quick run command** | `go test ./...` |
| **Full suite command** | `go test -v -count=1 -race ./...` |
| **Estimated runtime** | ~35 seconds |

## Sampling Rate

- **After every task commit:** Run `go test ./...`
- **After every plan wave:** Run `go test -v -count=1 -race ./...`
- **Max feedback latency:** 40 seconds

**Approval:** pending
