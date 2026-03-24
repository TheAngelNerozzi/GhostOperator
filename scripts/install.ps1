# GhostOperator (GO) Installer for Windows
# This script downloads the latest release and adds it to the user path.

$Repo = "TheAngelNerozzi/GhostOperator"
$Release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest"
$Asset = $Release.assets | Where-Object { $_.name -like "*ghost.exe*" }

if (-not $Asset) {
    Write-Error "Could not find ghost.exe in the latest release."
    exit 1
}

$InstallDir = "$HOME\.ghostoperator"
if (-not (Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir
}

Write-Host "Downloading GhostOperator..." -ForegroundColor Cyan
Invoke-WebRequest -Uri $Asset.browser_download_url -OutFile "$InstallDir\ghost.exe"

# Add to PATH if not present
$Path = [Environment]::GetEnvironmentVariable("Path", "User")
if ($Path -notlike "*$InstallDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$Path;$InstallDir", "User")
    Write-Host "GhostOperator added to Path. Please restart your terminal." -ForegroundColor Green
}

Write-Host "GhostOperator (GO) installed successfully! 👻" -ForegroundColor Green
