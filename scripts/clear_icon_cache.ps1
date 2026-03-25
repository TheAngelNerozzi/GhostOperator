# Windows Icon Cache Cleaner
# Run this script as Administrator to force Windows to reload the ghost.exe icon.

Write-Host "Stopping Windows Explorer..." -ForegroundColor Cyan
Stop-Process -Name explorer -Force -ErrorAction SilentlyContinue

Write-Host "Cleaning Icon Cache..." -ForegroundColor Yellow
$CachePath = "$env:LOCALAPPDATA\Microsoft\Windows\Explorer"
Remove-Item -Path "$CachePath\iconcache*" -Force -ErrorAction SilentlyContinue

Write-Host "Restarting Windows Explorer..." -ForegroundColor Green
Start-Process explorer.exe

Write-Host "Icon Cache cleared successfully! 👻" -ForegroundColor Green
