# Phase 11: Distribution Infrastructure - Context

**Gathered:** 2026-03-22
**Status:** Ready for planning

<domain>
## Phase Boundary

gsd-wired installable by any developer through standard channels without manual build steps. GoReleaser pipeline, Homebrew cask, ADC-signed macOS binaries, ghcr.io container image, clean `go install` path. Delivers DIST-01 through DIST-06.

</domain>

<decisions>
## Implementation Decisions

### Homebrew tap
- **D-01:** Tap repo: `The-Artificer-of-Ciphers-LLC/homebrew-gsdw` (project-specific)
- **D-02:** Public repo — `brew install gsdw-cc` works without auth tokens
- **D-03:** Cask name: `gsdw-cc` — install command: `brew tap The-Artificer-of-Ciphers-LLC/gsdw && brew install gsdw-cc`
- **D-04:** Tap repo includes README with troubleshooting and platform requirements

### ADC signing
- **D-05:** Apple Developer Certificate signing for macOS binaries only (not Linux)
- **D-06:** Signing certificate at `~/.private-keys` — export .p12 for GitHub Actions secret
- **D-07:** Team ID (org account) — app-specific password needs creation at appleid.apple.com
- **D-08:** Local GoReleaser for Apple-signed macOS releases (cert on developer's machine). GitHub Actions for Linux binaries (remote, no Apple signing).
- **D-09:** Notarization: wait for completion before publishing release (30-90 seconds)
- **D-10:** .p12 export password stored as dedicated GitHub secret

### Release versioning
- **D-11:** Both ldflags override + ReadBuildInfo fallback. GoReleaser injects exact tag version via `-ldflags -X`. `go install` users get module version via ReadBuildInfo.
- **D-12:** Version format: `1.0.0 (abc1234)` — matches tag, includes git hash
- **D-13:** `gsdw version --json` output for machine consumption: version, commit, build date, platform
- **D-14:** Pre-release tags supported: `v1.0.0-rc.1` style for testing before final release

### From research (locked)
- **D-15:** GoReleaser v2.14.3 (pinned in GitHub Actions — dockers_v2 is alpha)
- **D-16:** `homebrew_casks` section in .goreleaser.yaml (not `brews` — deprecated since v2.10)
- **D-17:** `CGO_ENABLED=0` for all builds — ensures clean `go install` path
- **D-18:** `gcr.io/distroless/static-debian12:nonroot` as gsdw container base image
- **D-19:** Bot PAT with `contents:write` on tap repo for GoReleaser to push cask updates
- **D-20:** xattr Gatekeeper fix in cask postinstall (backup if notarization is delayed)

### Claude's Discretion
- GoReleaser YAML structure and exact configuration
- GitHub Actions workflow structure (trigger, jobs, steps)
- Dockerfile.container contents
- How to split local vs CI release workflows
- Makefile or task runner for local release process

</decisions>

<specifics>
## Specific Ideas

- Local release flow: developer runs `goreleaser release` on their Mac with signing cert available. Produces signed+notarized macOS binaries + unsigned Linux binaries. Pushes to GitHub Releases + updates brew tap.
- CI flow: GitHub Actions triggers on tag push, builds Linux binaries, pushes container image to ghcr.io. macOS binaries come from the local release.
- The `gsdw version --json` addition is the only Go source change in this phase. Everything else is config files (.goreleaser.yaml, Dockerfile, GitHub Actions workflow, cask definition).

</specifics>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Existing codebase
- `internal/version/version.go` — Current version logic using runtime/debug.ReadBuildInfo
- `internal/cli/version.go` — Current `gsdw version` subcommand (add --json flag)
- `cmd/gsdw/main.go` — Binary entry point
- `go.mod` — Module path `github.com/The-Artificer-of-Ciphers-LLC/gsd-wired`
- `.claude-plugin/plugin.json` — Plugin manifest

### Research
- `.planning/research/STACK.md` — GoReleaser config patterns, brew tap setup
- `.planning/research/PITFALLS.md` — Gatekeeper, bot PAT, brews deprecation

### Project context
- `.planning/REQUIREMENTS.md` — DIST-01 through DIST-06
- `.planning/ROADMAP.md` §Phase 11 — Success criteria (5 items)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/version/version.go` — ReadBuildInfo pattern (extend with ldflags + JSON)
- `internal/cli/version.go` — Cobra version subcommand (add --json flag)

### Established Patterns
- Cobra subcommands for CLI
- `go build ./cmd/gsdw` produces the binary
- `.gitignore` has `/gsdw` for built binary

### Integration Points
- `go.mod` module path drives `go install` URL
- `.goreleaser.yaml` (new) — GoReleaser config at repo root
- `.github/workflows/release.yml` (new) — CI pipeline
- `Dockerfile` (new) — Container image build
- Tap repo (external) — `The-Artificer-of-Ciphers-LLC/homebrew-gsdw`

</code_context>

<deferred>
## Deferred Ideas

- Windows binary support (DIST-A01, v2)
- Scoop/Chocolatey packages (DIST-A02, v2)
- Linux package managers (DIST-A03, v2)
- Automated release notes generation from SUMMARY.md files

</deferred>

---

*Phase: 11-distribution*
*Context gathered: 2026-03-22*
