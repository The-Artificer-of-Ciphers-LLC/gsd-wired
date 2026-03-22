---
status: resolved
trigger: "Apple codesigning was never set up for the gsd-wired Go binary. The user was never asked to create a signing identity, yet something may have prompted them for a password. The signing is not wired into the release pipeline. User wants it properly configured."
created: 2026-03-22T00:00:00Z
updated: 2026-03-22T00:02:00Z
---

## Current Focus
<!-- OVERWRITE on each update - reflects NOW -->

hypothesis: CONFIRMED — .goreleaser.yaml has a complete notarize.macos block referencing 5 env vars (MACOS_SIGN_P12, MACOS_SIGN_PASSWORD, MACOS_NOTARY_ISSUER_ID, MACOS_NOTARY_KEY_ID, MACOS_NOTARY_KEY), but the GitHub Actions workflow only passes GPG vars (GPG_KEY, GPG_FINGERPRINT, GPG_PASSWORD). Additionally, the design decision D-08 says macOS signing should happen via LOCAL goreleaser run (not CI), yet no local Makefile target or runbook exists to guide this. The user has no instructions for setting up the Apple signing identity.
test: N/A - root cause confirmed through code reading
expecting: N/A
next_action: DONE — path correction applied and session archived

## Symptoms
<!-- Written during gathering, then IMMUTABLE -->

expected: Apple codesigning should be configured for the gsd-wired binary so releases are properly signed
actual: Codesigning was never set up — user was never asked to create signing identity
errors: None specific — it is a missing feature/configuration
reproduction: Check GoReleaser config, GitHub Actions release workflow, any signing references
started: During Phase 11 (Distribution Infrastructure), signing config was added to .goreleaser.yaml but Apple codesigning was never properly wired

## Eliminated
<!-- APPEND only - prevents re-investigating -->

- hypothesis: Apple codesigning is wired into the GitHub Actions CI workflow
  evidence: .github/workflows/release.yml only passes GPG_KEY, GPG_FINGERPRINT, GPG_PASSWORD — zero MACOS_ vars present
  timestamp: 2026-03-22T00:01:00Z

- hypothesis: The local release path has documented setup steps
  evidence: Phase 11 summaries mention "macOS signing (local release path per D-08)" as user setup but only say "Apple Developer Certificate at ~/.private-keys, App-specific password at appleid.apple.com, Run goreleaser release --clean locally" — no Makefile target, no export steps, no GitHub secrets setup guide
  timestamp: 2026-03-22T00:01:00Z

## Evidence
<!-- APPEND only - facts discovered -->

- timestamp: 2026-03-22T00:01:00Z
  checked: .goreleaser.yaml notarize.macos block (lines 48-58)
  found: Five env vars required: MACOS_SIGN_P12, MACOS_SIGN_PASSWORD, MACOS_NOTARY_ISSUER_ID, MACOS_NOTARY_KEY_ID, MACOS_NOTARY_KEY
  implication: Every goreleaser run (local OR CI) will fail if these env vars are not set

- timestamp: 2026-03-22T00:01:00Z
  checked: .github/workflows/release.yml goreleaser env block (lines 43-46)
  found: Only GITHUB_TOKEN, GSDWHOMEBREW, GPG_FINGERPRINT, GPG_PASSWORD — no MACOS_ vars
  implication: CI release will fail at notarization step; also the CI runs on ubuntu-latest which cannot codesign for Apple anyway

- timestamp: 2026-03-22T00:01:00Z
  checked: Phase 11 design decision D-08
  found: "Local GoReleaser for Apple-signed macOS releases (cert on developer's machine). GitHub Actions for Linux binaries (remote, no Apple signing)."
  implication: The architecture is intentionally split — but this split was never implemented in goreleaser config or CI workflow

- timestamp: 2026-03-22T00:01:00Z
  checked: Phase 11 11-01-SUMMARY.md key-decisions
  found: "signs section retained with GPG config — can be removed if GPG not available at release time"
  implication: GPG signing of checksums is optional/untested; Apple signing is the more important gap

- timestamp: 2026-03-22T00:01:00Z
  checked: goreleaser notarize section — what env vars actually mean
  found: MACOS_SIGN_P12 = base64-encoded .p12 cert, MACOS_SIGN_PASSWORD = .p12 export password, MACOS_NOTARY_ISSUER_ID + MACOS_NOTARY_KEY_ID + MACOS_NOTARY_KEY = App Store Connect API key (not app-specific password)
  implication: User needs Apple Developer account, Developer ID Application cert exported as .p12, AND an App Store Connect API key (different from the app-specific password mentioned in phase docs)

## Resolution
<!-- OVERWRITE as understanding evolves -->

root_cause: The .goreleaser.yaml notarize.macos block was written referencing 5 env vars (MACOS_SIGN_P12, MACOS_SIGN_PASSWORD, MACOS_NOTARY_ISSUER_ID, MACOS_NOTARY_KEY_ID, MACOS_NOTARY_KEY) but: (1) the CI workflow never had these vars added, (2) no local Makefile target exists to run goreleaser with these vars set, (3) no isEnvSet guard protected the notarize block from failing when vars are absent. The CI workflow also runs on ubuntu-latest which cannot perform Apple codesigning — the local release path per D-08 was never operationalized.
fix: |
  1. Added `enabled: '{{ isEnvSet "MACOS_SIGN_P12" }}'` to .goreleaser.yaml notarize.macos block — goreleaser gracefully skips signing when the env var is absent (protects CI ubuntu runner)
  2. Added all 5 MACOS_ env var references to .github/workflows/release.yml goreleaser step — secrets will be empty on Ubuntu but are wired for future macOS runner adoption
  3. Created Makefile with `release-mac` target — guards for all required env vars, prints clear error messages, then runs goreleaser release --clean. Also has `release-mac-snapshot` for dry runs.
  4. Makefile includes inline setup instructions (one-time steps: export .p12, create App Store Connect API key, base64-encode both)
verification: |
  - .goreleaser.yaml notarize.macos[0] has `enabled: '{{ isEnvSet "MACOS_SIGN_P12" }}'` — checked
  - .github/workflows/release.yml goreleaser env block has all 5 MACOS_ vars — checked
  - Makefile exists with release-mac and release-mac-snapshot targets — checked
  - Makefile guards all 6 required env vars with clear error messages — checked
files_changed:
  - .goreleaser.yaml
  - .github/workflows/release.yml
  - Makefile
  - .planning/phases/11-distribution/11-02-SUMMARY.md
  - .planning/phases/11-distribution/11-CONTEXT.md
  - .planning/phases/11-distribution/11-02-PLAN.md
