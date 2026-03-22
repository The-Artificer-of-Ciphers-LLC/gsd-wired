#!/bin/bash
# Sign and notarize a macOS binary using native codesign + notarytool.
# Called by GoReleaser via binary_signs for all builds — skips non-darwin.
#
# Required env vars:
#   MACOS_SIGN_P12          - base64-encoded .p12 certificate
#   MACOS_SIGN_PASSWORD     - .p12 export password
#   MACOS_NOTARY_ISSUER_ID  - App Store Connect issuer ID
#   MACOS_NOTARY_KEY_ID     - App Store Connect key ID
#   MACOS_NOTARY_KEY        - base64-encoded .p8 key
#
# Usage: sign-and-notarize.sh <binary-path>

set -euo pipefail

BINARY="$1"
SIG_FILE="${2:-${BINARY}.sig}"

# Skip non-darwin binaries (GoReleaser calls this for all platforms)
if [[ "$BINARY" != *"darwin"* ]]; then
    echo "Skipping non-darwin binary: $BINARY"
    echo "skipped" > "$SIG_FILE"
    exit 0
fi

# Skip if signing env vars not set
if [ -z "${MACOS_SIGN_P12:-}" ]; then
    echo "MACOS_SIGN_P12 not set, skipping signing"
    echo "skipped" > "$SIG_FILE"
    exit 0
fi

TMPDIR_SIGN=$(mktemp -d)
trap 'security list-keychains -d user -s $ORIGINAL_KEYCHAINS 2>/dev/null; rm -rf "$TMPDIR_SIGN"' EXIT

# Decode cert and key to temp files
P12_FILE="$TMPDIR_SIGN/cert.p12"
P8_FILE="$TMPDIR_SIGN/notary-key.p8"
echo "$MACOS_SIGN_P12" | base64 -d > "$P12_FILE"
echo "$MACOS_NOTARY_KEY" | base64 -d > "$P8_FILE"

# Create a temporary keychain for signing
KEYCHAIN="$TMPDIR_SIGN/signing.keychain-db"
KEYCHAIN_PWD="temporary-$(openssl rand -hex 12)"
security create-keychain -p "$KEYCHAIN_PWD" "$KEYCHAIN"
security set-keychain-settings "$KEYCHAIN"
security unlock-keychain -p "$KEYCHAIN_PWD" "$KEYCHAIN"

# Import the certificate
security import "$P12_FILE" -k "$KEYCHAIN" -P "$MACOS_SIGN_PASSWORD" -T /usr/bin/codesign -A
# Allow codesign to access the key without prompt
security set-key-partition-list -S apple-tool:,apple: -s -k "$KEYCHAIN_PWD" "$KEYCHAIN" 2>/dev/null || true

# Add temp keychain to search list (preserve existing)
ORIGINAL_KEYCHAINS=$(security list-keychains -d user | tr -d '"' | tr '\n' ' ')
security list-keychains -d user -s "$KEYCHAIN" $ORIGINAL_KEYCHAINS

# Find the signing identity (prefer Developer ID, fall back to any codesigning identity)
IDENTITY=$(security find-identity -v -p codesigning "$KEYCHAIN" | grep -o '"[^"]*"' | head -1 | tr -d '"')
if [ -z "$IDENTITY" ]; then
    echo "ERROR: No codesigning identity found in certificate"
    exit 1
fi
echo "Signing with: $IDENTITY"

if [[ "$IDENTITY" != *"Developer ID"* ]]; then
    echo "WARNING: Using '$IDENTITY' — for App Store distribution, use a 'Developer ID Application' certificate"
fi

# Sign the binary
codesign --force --options runtime --keychain "$KEYCHAIN" --sign "$IDENTITY" --timestamp "$BINARY"
echo "Signed: $BINARY"

# Create a zip for notarization
ZIP_FILE="$TMPDIR_SIGN/$(basename "$BINARY").zip"
ditto -c -k --keepParent "$BINARY" "$ZIP_FILE"

# Submit for notarization
echo "Submitting for notarization..."
xcrun notarytool submit "$ZIP_FILE" \
    --issuer "$MACOS_NOTARY_ISSUER_ID" \
    --key-id "$MACOS_NOTARY_KEY_ID" \
    --key "$P8_FILE" \
    --wait \
    --timeout 10m

echo "Notarization complete: $BINARY"

# Write a marker signature file
echo "signed-and-notarized" > "$SIG_FILE"
