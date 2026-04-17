#!/bin/bash
# GhostOperator Code Signing Certificate Generator
# Generates a self-signed code signing certificate for Windows binaries
#
# IMPORTANT: Self-signed certificates will show as "Unknown Publisher" in Windows
# For full SmartScreen trust, purchase a certificate from a trusted CA:
#   - DigiCert: https://www.digicert.com/signing/code-signing-certificates
#   - Sectigo: https://sectigo.com/ssl-certificates-tls/code-signing
#   - Or use SignPath Foundation (free for OSS): https://signpath.org
#
# Usage:
#   ./scripts/generate-cert.sh
#   GHOST_CODESIGN_PASS=mypassword ./scripts/generate-cert.sh

set -euo pipefail

CERT_DIR="$(cd "$(dirname "$0")/.." && pwd)/certs"
mkdir -p "$CERT_DIR"

KEY_FILE="${CERT_DIR}/ghost-code-signing.key"
CRT_FILE="${CERT_DIR}/ghost-code-signing.crt"
P12_FILE="${CERT_DIR}/ghost-code-signing.p12"
CNF_FILE="$(mktemp /tmp/ghost_codesign.XXXXXX.cnf)"
trap 'rm -f "$CNF_FILE"' EXIT

# Require the password from environment variable (no default)
if [ -z "${GHOST_CODESIGN_PASS:-}" ]; then
    echo "Error: GHOST_CODESIGN_PASS environment variable is required"
    echo "Usage: GHOST_CODESIGN_PASS=mypassword $0"
    exit 1
fi
P12_PASS="$GHOST_CODESIGN_PASS"

echo "Generating code signing certificate..."

# Generate RSA 4096-bit private key
openssl genrsa -out "$KEY_FILE" 4096

# Create config
cat > "$CNF_FILE" << EOF
[req]
default_bits = 4096
prompt = no
default_md = sha256
distinguished_name = dn
x509_extensions = v3_code_sign

[dn]
CN = GhostOperator
O = GhostOperator
L = Caracas
ST = Distrito Capital
C = VE

[v3_code_sign]
basicConstraints = critical, CA:FALSE
keyUsage = critical, digitalSignature
extendedKeyUsage = critical, codeSigning
subjectKeyIdentifier = hash
authorityKeyIdentifier = keyid:always, issuer
EOF

# Create self-signed certificate (valid 3 years)
openssl req -new -x509 -key "$KEY_FILE" -out "$CRT_FILE" \
    -days 1095 -config "$CNF_FILE"

# Export to PKCS12 for osslsigncode
openssl pkcs12 -export -out "$P12_FILE" \
    -inkey "$KEY_FILE" \
    -in "$CRT_FILE" \
    -passout pass:"$P12_PASS"

echo "Certificate generated successfully!"
echo "  Private Key: $KEY_FILE"
echo "  Certificate: $CRT_FILE"
echo "  PKCS12:      $P12_FILE"
echo ""
echo "IMPORTANT: This is a self-signed certificate. Windows will still show"
echo "'Unknown Publisher' until you purchase a trusted CA certificate."
echo "See README.md for code signing options."
