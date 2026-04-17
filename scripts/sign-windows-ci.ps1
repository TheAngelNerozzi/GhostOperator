# GhostOperator - Code Signing Script for CI (Windows)
# Uses self-signed certificate + signtool.exe (available on windows-latest runners)
#
# This script:
# 1. Creates a self-signed code signing certificate
# 2. Adds it to the Windows certificate store
# 3. Signs the binary with signtool.exe
# 4. Verifies the signature
#
# Usage (in CI):
#   powershell -File scripts/sign-windows-ci.ps1 -InputFile ghost-windows-amd64.exe

param(
    [Parameter(Mandatory=$true)]
    [string]$InputFile,

    [string]$OutputFile = $InputFile,

    [string]$CertificatePassword = "ghostoperator2026sign"
)

$ErrorActionPreference = "Stop"

Write-Host "[SIGN] GhostOperator Windows Code Signing (CI)" -ForegroundColor Cyan
Write-Host ""

# ── Step 1: Create self-signed code signing certificate ──
Write-Host "[1/4] Creating self-signed code signing certificate..." -ForegroundColor Yellow

$cert = New-SelfSignedCertificate `
    -Type CodeSigningCert `
    -Subject "CN=Angel Nerozzi, O=Angel Nerozzi, C=VE" `
    -HashAlgorithm SHA256 `
    -KeyAlgorithm RSA `
    -KeyLength 4096 `
    -CertStoreLocation "Cert:\CurrentUser\My" `
    -NotAfter (Get-Date).AddYears(3) `
    -TextExtension @("2.5.29.37={text}1.3.6.1.5.5.7.3.3","2.5.29.19={text}")

Write-Host "  Thumbprint: $($cert.Thumbprint)" -ForegroundColor Gray
Write-Host "  Subject: $($cert.Subject)" -ForegroundColor Gray
Write-Host "  Expires: $($cert.NotAfter)" -ForegroundColor Gray

# ── Step 2: Export to PFX ──
Write-Host "[2/4] Exporting certificate to PFX..." -ForegroundColor Yellow

$pfxPath = Join-Path $env:TEMP "ghost-code-signing.pfx"
$certPassword = ConvertTo-SecureString -String $CertificatePassword -Force -AsPlainText
Export-PfxCertificate -Cert $cert -FilePath $pfxPath -Password $certPassword | Out-Null

Write-Host "  Exported to: $pfxPath" -ForegroundColor Gray

# ── Step 3: Sign the binary ──
Write-Host "[3/4] Signing binary with signtool.exe..." -ForegroundColor Yellow

$signtool = "${env:ProgramFiles(x86)}\Windows Kits\10\bin\10.0.22621.0\x64\signtool.exe"
if (-not (Test-Path $signtool)) {
    # Try to find signtool in any Windows Kits directory
    $signtool = Get-ChildItem -Path "${env:ProgramFiles(x86)}\Windows Kits" -Recurse -Filter "signtool.exe" -ErrorAction SilentlyContinue | Select-Object -First 1 -ExpandProperty FullName
}
if (-not $signtool) {
    Write-Host "  ERROR: signtool.exe not found!" -ForegroundColor Red
    exit 1
}

Write-Host "  Using: $signtool" -ForegroundColor Gray

& $signtool sign `
    /f $pfxPath `
    /p $CertificatePassword `
    /tr "http://timestamp.digicert.com" `
    /td SHA256 `
    /fd SHA256 `
    /c "CN=Angel Nerozzi" `
    $InputFile

if ($LASTEXITCODE -ne 0) {
    Write-Host "  ERROR: Signing failed with exit code $LASTEXITCODE" -ForegroundColor Red
    exit 1
}

Write-Host "  Signed successfully!" -ForegroundColor Green

# ── Step 4: Verify signature ──
Write-Host "[4/4] Verifying signature..." -ForegroundColor Yellow

& $signtool verify /pa /all $InputFile

if ($LASTEXITCODE -eq 0) {
    Write-Host "  Verification: PASSED" -ForegroundColor Green
} else {
    Write-Host "  Verification: WARNING - signature valid but may not chain to trusted root" -ForegroundColor Yellow
    Write-Host "  (This is expected for self-signed certificates)" -ForegroundColor Gray
}

# Cleanup
Remove-Item $pfxPath -Force -ErrorAction SilentlyContinue

Write-Host ""
Write-Host "[SIGN] Done. Signed binary: $InputFile" -ForegroundColor Green
