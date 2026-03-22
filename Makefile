.PHONY: build test release-mac release-mac-snapshot help

## build: compile gsdw binary for the current platform
build:
	go build -o gsdw ./cmd/gsdw

## test: run all tests
test:
	go test ./... -v

## release-mac: build signed + notarized macOS release locally (requires Apple Developer cert)
##
## Prerequisites (one-time setup — see docs/SIGNING.md):
##   1. Export your "Developer ID Application" cert as ~/.private_keys/developer-id-application.p12
##   2. Create an App Store Connect API key at https://appstoreconnect.apple.com/access/integrations/api
##   3. Download the .p8 key file; note the Issuer ID and Key ID from the portal
##   4. Set these env vars (add to ~/.zshenv or pass inline):
##        MACOS_SIGN_P12        — base64-encoded .p12:  base64 -i ~/.private_keys/developer-id-application.p12
##        MACOS_SIGN_PASSWORD   — the .p12 export password you set when exporting
##        MACOS_NOTARY_ISSUER_ID — Issuer ID from App Store Connect
##        MACOS_NOTARY_KEY_ID   — Key ID from App Store Connect
##        MACOS_NOTARY_KEY      — base64-encoded .p8:  base64 -i ~/.private_keys/AuthKey_2H33PL2R8M.p8
##
## Usage:
##   make release-mac                    # full signed release (pushes to GitHub + updates brew tap)
##   make release-mac-snapshot           # dry run (no publish, no tag required)
release-mac:
	@echo "==> Checking required env vars..."
	@test -n "$$MACOS_SIGN_P12"         || (echo "ERROR: MACOS_SIGN_P12 is not set. See 'make help'."; exit 1)
	@test -n "$$MACOS_SIGN_PASSWORD"    || (echo "ERROR: MACOS_SIGN_PASSWORD is not set. See 'make help'."; exit 1)
	@test -n "$$MACOS_NOTARY_ISSUER_ID" || (echo "ERROR: MACOS_NOTARY_ISSUER_ID is not set. See 'make help'."; exit 1)
	@test -n "$$MACOS_NOTARY_KEY_ID"    || (echo "ERROR: MACOS_NOTARY_KEY_ID is not set. See 'make help'."; exit 1)
	@test -n "$$MACOS_NOTARY_KEY"       || (echo "ERROR: MACOS_NOTARY_KEY is not set. See 'make help'."; exit 1)
	@test -n "$$GSDWHOMEBREW"           || (echo "ERROR: GSDWHOMEBREW is not set (homebrew tap PAT). See 'make help'."; exit 1)
	@echo "==> All env vars present. Starting goreleaser release..."
	goreleaser release --clean

## release-mac-snapshot: dry run of the macOS release (no publish, no tag required)
release-mac-snapshot:
	goreleaser release --snapshot --clean --skip=docker

## help: show this help
help:
	@grep -E '^## ' Makefile | sed 's/^## //' | column -t -s ':'
