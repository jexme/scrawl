# Собирает bin\scrawl.exe для Windows (GUI-программа без окна консоли).
# Нужны Go (версия из go.mod) и Git for Windows: из него берутся sh и patch для подготовки Gio.
#   powershell -ExecutionPolicy Bypass -File scripts\build-windows.ps1
param([string]$Out = "bin\scrawl.exe")
$ErrorActionPreference = "Stop"
Set-Location (Split-Path $PSScriptRoot -Parent)

$git = (Get-Command git.exe -ErrorAction Stop).Source
$gitUsrBin = Join-Path (Split-Path (Split-Path $git)) "usr\bin"
if (-not (Test-Path (Join-Path $gitUsrBin "sh.exe"))) { throw "не нашёл sh.exe в ${gitUsrBin}: нужен Git for Windows" }
$env:PATH = "$gitUsrBin;$env:PATH"

# Gio нужной версии с нашим патчем (повторный запуск ничего не делает, пока версия не менялась).
& sh scripts/setup-gio.sh
if ($LASTEXITCODE -ne 0) { throw "setup-gio.sh завершился с кодом $LASTEXITCODE" }

$version = (& git describe --tags --always --dirty 2>$null)
$ldflags = "-H=windowsgui"
if ($version) { $ldflags += " -X github.com/jexme/scrawl/internal/version.Version=$version" }
New-Item -ItemType Directory -Force (Split-Path $Out) | Out-Null
& go build -ldflags $ldflags -o $Out ./cmd/scrawl
if ($LASTEXITCODE -ne 0) { throw "go build завершился с кодом $LASTEXITCODE" }
Write-Host "Готово: ${Out} (${version})"
