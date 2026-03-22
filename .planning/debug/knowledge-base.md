# GSD Debug Knowledge Base

Resolved debug sessions. Used by `gsd-debugger` to surface known-pattern hypotheses at the start of new investigations.

---

## apple-signing-not-setup — Apple codesigning env vars missing, no local release target, wrong private_keys path
- **Date:** 2026-03-22
- **Error patterns:** codesigning, notarize, MACOS_SIGN_P12, MACOS_SIGN_PASSWORD, MACOS_NOTARY_ISSUER_ID, MACOS_NOTARY_KEY_ID, MACOS_NOTARY_KEY, goreleaser, signing, release
- **Root cause:** .goreleaser.yaml notarize.macos block referenced 5 MACOS_ env vars but: CI workflow never had them, no isEnvSet guard prevented goreleaser from failing when vars absent on ubuntu runner, and no local Makefile target or runbook existed to guide local macOS signing. Planning docs also had wrong path (private-keys with hyphen) and wrong auth method (app-specific password instead of App Store Connect API key).
- **Fix:** Added `enabled: '{{ isEnvSet "MACOS_SIGN_P12" }}'` guard to .goreleaser.yaml; wired all 5 MACOS_ vars into release.yml; created Makefile with release-mac target (guards + inline setup steps); corrected path to ~/.private_keys (underscore) in all phase docs; corrected auth method docs to App Store Connect API key.
- **Files changed:** .goreleaser.yaml, .github/workflows/release.yml, Makefile, .planning/phases/11-distribution/11-02-SUMMARY.md, .planning/phases/11-distribution/11-CONTEXT.md, .planning/phases/11-distribution/11-02-PLAN.md
---
