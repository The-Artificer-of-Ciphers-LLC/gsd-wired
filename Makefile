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
##   4. Copy .env.release.example to .env.release and fill in the values
##
## Usage:
##   make release-mac                    # full signed release (sources .env.release automatically)
##   make release-mac-snapshot           # dry run (no publish, no tag required)
release-mac:
	@test -f .env.release || (echo "ERROR: .env.release not found. See Makefile header for setup."; exit 1)
	@. ./.env.release && \
		echo "==> Loaded .env.release" && \
		test -n "$$MACOS_SIGN_P12"         || (echo "ERROR: MACOS_SIGN_P12 is not set."; exit 1) && \
		test -n "$$MACOS_SIGN_PASSWORD"    || (echo "ERROR: MACOS_SIGN_PASSWORD is not set."; exit 1) && \
		test -n "$$MACOS_NOTARY_ISSUER_ID" || (echo "ERROR: MACOS_NOTARY_ISSUER_ID is not set."; exit 1) && \
		test -n "$$MACOS_NOTARY_KEY_ID"    || (echo "ERROR: MACOS_NOTARY_KEY_ID is not set."; exit 1) && \
		test -n "$$MACOS_NOTARY_KEY"       || (echo "ERROR: MACOS_NOTARY_KEY is not set."; exit 1) && \
		test -n "$$GSDWHOMEBREW"           || (echo "ERROR: GSDWHOMEBREW is not set."; exit 1) && \
		test -n "$$GITHUB_TOKEN"           || (echo "ERROR: GITHUB_TOKEN is not set."; exit 1) && \
		echo "==> All env vars present. Starting goreleaser release..." && \
		goreleaser release --clean

## release-mac-snapshot: dry run of the macOS release (no publish, no tag required)
release-mac-snapshot:
	@test -f .env.release || (echo "ERROR: .env.release not found. See Makefile header for setup."; exit 1)
	@. ./.env.release && goreleaser release --snapshot --clean

## help: show this help
help:
	@grep -E '^## ' Makefile | sed 's/^## //' | column -t -s ':'
