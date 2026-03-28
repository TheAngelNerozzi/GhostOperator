# GhostOperator - Setup Interactivo v2.0
# Ejecuta este script la primera vez que clones el proyecto.

$ErrorActionPreference = "SilentlyContinue"

Write-Host ""
Write-Host "  ██████  ████████  ██████  ███████ ████████" -ForegroundColor White
Write-Host "  ██          ██   ██    ██ ██         ██   " -ForegroundColor White
Write-Host "  ██████      ██   ██    ██ ███████    ██   " -ForegroundColor White
Write-Host "  ██          ██   ██    ██      ██    ██   " -ForegroundColor White
Write-Host "  ██████      ██    ██████  ███████    ██   " -ForegroundColor White
Write-Host ""
Write-Host "  GhostOperator v2.0 - Local AI Setup" -ForegroundColor Gray
Write-Host "  Angel Nerozzi | Open Source" -ForegroundColor DarkGray
Write-Host ""

# ─────────────────────────────────────────────
# 1. Detectar Ollama
# ─────────────────────────────────────────────
$ollamaExe = "$env:LOCALAPPDATA\Programs\Ollama\ollama.exe"
$ollamaInPath = Get-Command "ollama" -ErrorAction SilentlyContinue

if ($ollamaInPath) {
    $ollamaExe = "ollama"
}

if (-not (Test-Path $ollamaExe) -and -not $ollamaInPath) {
    Write-Host "[!] Ollama no detectado en este sistema." -ForegroundColor Yellow
    Write-Host "    Descargalo de: https://ollama.com/download" -ForegroundColor DarkGray
    $instalar = Read-Host "    ¿Deseas abrir la pagina de descarga ahora? (s/n)"
    if ($instalar -eq "s") {
        Start-Process "https://ollama.com/download"
    }
    Write-Host "    Vuelve a ejecutar este script despues de instalar Ollama." -ForegroundColor Gray
    exit 1
}

Write-Host "[OK] Ollama detectado." -ForegroundColor Green

# ─────────────────────────────────────────────
# 2. Iniciar servicio Ollama si no está corriendo
# ─────────────────────────────────────────────
$ollamaRunning = Get-Process "ollama" -ErrorAction SilentlyContinue
if (-not $ollamaRunning) {
    Write-Host "[>>] Iniciando Ollama en segundo plano..." -ForegroundColor Cyan
    Start-Process -FilePath $ollamaExe -ArgumentList "serve" -WindowStyle Hidden
    Start-Sleep -Seconds 3
} else {
    Write-Host "[OK] Ollama ya esta corriendo." -ForegroundColor Green
}

# ─────────────────────────────────────────────
# 3. Verificar si moondream ya está instalado
# ─────────────────────────────────────────────
$modelosInstalados = & $ollamaExe list 2>&1
$moondreamPresente = $modelosInstalados | Select-String "moondream"

if ($moondreamPresente) {
    Write-Host "[OK] Modelo moondream ya instalado." -ForegroundColor Green
} else {
    Write-Host ""
    Write-Host "  GhostOperator usa Moondream 1.8B para ver tu pantalla." -ForegroundColor White
    Write-Host "  Es un modelo liviano (~900MB) que corre 100% local." -ForegroundColor DarkGray
    Write-Host ""
    $respuesta = Read-Host "  ¿Deseas instalar Moondream 1.8B ahora? (s/n)"
    if ($respuesta -eq "s" -or $respuesta -eq "S") {
        Write-Host ""
        Write-Host "[>>] Descargando moondream... (esto puede tardar unos minutos)" -ForegroundColor Cyan
        & $ollamaExe pull moondream
        if ($LASTEXITCODE -eq 0) {
            Write-Host "[OK] Moondream instalado exitosamente." -ForegroundColor Green
        } else {
            Write-Host "[!!] Error instalando moondream. Intenta manualmente: ollama pull moondream" -ForegroundColor Red
        }
    } else {
        Write-Host "[--] Omitiendo instalacion del modelo." -ForegroundColor DarkGray
        Write-Host "     Puedo instalarlo mas tarde con: ollama pull moondream" -ForegroundColor DarkGray
    }
}

# ─────────────────────────────────────────────
# 4. Compilar ghost.exe
# ─────────────────────────────────────────────
Write-Host ""
$compilar = Read-Host "[?] ¿Compilar ghost.exe ahora? (s/n)"
if ($compilar -eq "s" -or $compilar -eq "S") {
    Write-Host "[>>] Compilando GhostOperator..." -ForegroundColor Cyan
    & "C:\Program Files\Go\bin\go.exe" build -ldflags "-s -w" -o ghost.exe ./cmd/ghost
    if ($LASTEXITCODE -eq 0) {
        Write-Host "[OK] ghost.exe compilado exitosamente." -ForegroundColor Green
        Write-Host ""
        Write-Host "  Para iniciar: .\ghost.exe         (abre la UI en el navegador)" -ForegroundColor White
        Write-Host "  Para CLI:     .\ghost.exe start   (modo terminal)" -ForegroundColor White
    } else {
        Write-Host "[!!] Error de compilacion. Asegurate de tener Go instalado:" -ForegroundColor Red
        Write-Host "     https://go.dev/dl/" -ForegroundColor DarkGray
    }
}

Write-Host ""
Write-Host "  Ghost listo. Que el Fantasma trabaje por ti." -ForegroundColor White
Write-Host ""
