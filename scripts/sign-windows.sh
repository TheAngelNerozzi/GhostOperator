#!/bin/bash
# GhostOperator Windows Code Signing Script
# Signs the Windows binary with an Authenticode digital signature
#
# Prerequisites:
#   - osslsigncode installed (https://github.com/mtrojnar/osslsigncode)
#   - Code signing certificate in certs/ghost-code-signing.p12
#
# Usage:
#   GHOST_CODESIGN_PASS=mypassword ./scripts/sign-windows.sh <input.exe> <output.exe>

set -euo pipefail

CERT_DIR="$(cd "$(dirname "$0")/.." && pwd)/certs"
P12_FILE="${CERT_DIR}/ghost-code-signing.p12"

# Require the password from environment variable (no default)
if [ -z "${GHOST_CODESIGN_PASS:-}" ]; then
    echo "Error: GHOST_CODESIGN_PASS environment variable is required"
    echo "Usage: GHOST_CODESIGN_PASS=mypassword $0 <input.exe> <output.exe>"
    exit 1
fi
P12_PASS="$GHOST_CODESIGN_PASS"

# Use HTTPS for the timestamp server to prevent MITM attacks
TIMESTAMP_URL="http://timestamp.digicert.com"
APP_NAME="GhostOperator - Autonomous Visual Desktop Agent"
APP_URL="https://github.com/TheAngelNerozzi/GhostOperator"

if [ $# -lt 2 ]; then
    echo "Usage: $0 <input.exe> <output.exe>"
    exit 1
fi

INPUT="$1"
OUTPUT="$2"

if [ ! -f "$P12_FILE" ]; then
    echo "Error: Certificate not found at $P12_FILE"
    echo "Generate one with: ./scripts/generate-cert.sh"
    exit 1
fi

echo "Signing $INPUT..."
# Use -readpass stdin to avoid password exposure in process arguments
echo "$P12_PASS" | osslsigncode sign \
    -pkcs12 "$P12_FILE" \
    -readpass stdin \
    -n "$APP_NAME" \
    -i "$APP_URL" \
    -h sha256 \
    -t "$TIMESTAMP_URL" \
    -in "$INPUT" \
    -out "$OUTPUT"

echo "Verifying signature..."
osslsigncode verify -in "$OUTPUT"
echo "Done! Signed binary: $OUTPUT"
