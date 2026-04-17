# GhostOperator (GO) Installer for Windows
# This script downloads the latest release from GitHub.

$Repo = "TheAngelNerozzi/GhostOperator"
$InstallDir = "$HOME\.ghostoperator"

try {
    Write-Host "Checking for latest GhostOperator release..." -ForegroundColor Cyan
    $Release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest" -ErrorAction Stop

    # Architecture detection
    $Arch = if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") { "arm64" } else { "amd64" }
    $AssetName = "ghost-windows-${Arch}.exe"
    $Asset = $Release.assets | Where-Object { $_.name -eq $AssetName }

    if (-not $Asset) {
        # Fallback: try generic name
        $Asset = $Release.assets | Where-Object { $_.name -eq "ghost.exe" }
    }

    if (-not $Asset) {
        throw "Could not find a compatible binary in the latest release (looked for $AssetName or ghost.exe)."
    }

    if (-not (Test-Path $InstallDir)) {
        New-Item -ItemType Directory -Path $InstallDir | Out-Null
    }

    Write-Host "Downloading GhostOperator ($($Asset.name))..." -ForegroundColor Cyan
    Invoke-WebRequest -Uri $Asset.browser_download_url -OutFile "$InstallDir\ghost.exe" -ErrorAction Stop

    # Add to PATH if not present
    $Path = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($Path -notlike "*$InstallDir*") {
        [Environment]::SetEnvironmentVariable("Path", "$Path;$InstallDir", "User")
        Write-Host "GhostOperator added to User PATH." -ForegroundColor Green
    }

    Write-Host "`nGhostOperator (GO) installed successfully!" -ForegroundColor Green
    Write-Host "Please restart your terminal and run: ghost --version" -ForegroundColor Yellow
}
catch {
    Write-Host "Error: $($_.Exception.Message)" -ForegroundColor Red
    Write-Host "Check your internet connection or repository status at https://github.com/$Repo" -ForegroundColor Gray
    exit 1
}
