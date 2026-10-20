# Собирает bin\scrawl.exe для Windows: GUI-программа без окна консоли, со значком, манифестом
# (DPI, длинные пути) и сведениями о версии. С -Package ещё кладёт в dist\ архив для релиза и его SHA-256.
# Нужны Go (версия из go.mod) и Git for Windows: из него берутся sh и patch для подготовки Gio.
#   powershell -ExecutionPolicy Bypass -File scripts\build-windows.ps1 [-Version v0.1.0] [-Package]
param(
    [string]$Out = "bin\scrawl.exe",
    [string]$Version = "",
    [switch]$Package
)
$ErrorActionPreference = "Stop"
# Без cgo: на Windows он не нужен, а если на машине есть gcc, Go иначе пытается собрать C-файлы Linux-пакетов.
$env:CGO_ENABLED = "0"
Set-Location (Split-Path $PSScriptRoot -Parent)

# git.exe может лежать в Git\cmd, Git\bin или Git\mingw64\bin: поднимаемся вверх, пока не найдём usr\bin\sh.exe.
$gitUsrBin = $null
$dir = Split-Path (Get-Command git.exe -ErrorAction Stop).Source
while ($dir -and -not $gitUsrBin) {
    if (Test-Path (Join-Path $dir "usr\bin\sh.exe")) { $gitUsrBin = Join-Path $dir "usr\bin" }
    $dir = Split-Path $dir
}
if (-not $gitUsrBin) { throw "не нашёл usr\bin\sh.exe рядом с git.exe: нужен Git for Windows" }
$env:PATH = "$gitUsrBin;$env:PATH"

# Gio нужной версии с нашим патчем (повторный запуск ничего не делает, пока версия не менялась).
& sh scripts/setup-gio.sh
if ($LASTEXITCODE -ne 0) { throw "setup-gio.sh завершился с кодом $LASTEXITCODE" }

if (-not $Version) { $Version = (& git describe --tags --always --dirty 2>$null) }
$major = 0; $minor = 0; $patch = 0
if ($Version -match '^v?(\d+)\.(\d+)\.(\d+)') { $major = [int]$Matches[1]; $minor = [int]$Matches[2]; $patch = [int]$Matches[3] }

# Значок, манифест и версия файла попадают в .exe ресурсом (.syso рядом с main.go, в git не хранится).
$syso = "cmd\scrawl\resource_windows_amd64.syso"
Push-Location packaging\windows
try {
    & go run github.com/josephspurrier/goversioninfo/cmd/goversioninfo@v1.4.1 -64 `
        -ver-major $major -ver-minor $minor -ver-patch $patch `
        -product-ver-major $major -product-ver-minor $minor -product-ver-patch $patch `
        -file-version "$Version" -product-version "$Version" `
        -o "..\..\$syso" versioninfo.json
    if ($LASTEXITCODE -ne 0) { throw "goversioninfo завершился с кодом $LASTEXITCODE" }
} finally { Pop-Location }

$ldflags = "-H=windowsgui"
if ($Version) { $ldflags += " -X github.com/jexme/scrawl/internal/version.Version=$Version" }
New-Item -ItemType Directory -Force (Split-Path $Out) | Out-Null
& go build -trimpath -ldflags $ldflags -o $Out ./cmd/scrawl
if ($LASTEXITCODE -ne 0) { throw "go build завершился с кодом $LASTEXITCODE" }
Write-Host "Готово: ${Out} (${Version})"

if ($Package) {
    $name = "scrawl-$Version-windows-amd64"
    $dir = Join-Path "dist" $name
    Remove-Item $dir, "dist\$name.zip", "dist\$name.zip.sha256" -Recurse -Force -ErrorAction SilentlyContinue
    New-Item -ItemType Directory -Force $dir | Out-Null
    Copy-Item $Out (Join-Path $dir "scrawl.exe")
    Copy-Item LICENSE, THIRD_PARTY_NOTICES.md, README.md, README.ru.md, docs\WINDOWS.md $dir
    # Архив пишем сами: Compress-Archive и ZipFile.CreateFromDirectory в Windows PowerShell 5.1 кладут в имена
    # обратные косые, и такой архив неправильно распаковывается в других системах.
    Add-Type -AssemblyName System.IO.Compression, System.IO.Compression.FileSystem
    $distRoot = (Resolve-Path "dist").Path
    $fs = [IO.File]::Create((Join-Path $distRoot "$name.zip"))
    $za = New-Object IO.Compression.ZipArchive($fs, [IO.Compression.ZipArchiveMode]::Create)
    foreach ($f in Get-ChildItem $dir -Recurse -File) {
        $entry = $f.FullName.Substring($distRoot.Length + 1).Replace('\', '/')
        [IO.Compression.ZipFileExtensions]::CreateEntryFromFile($za, $f.FullName, $entry, [IO.Compression.CompressionLevel]::Optimal) | Out-Null
    }
    $za.Dispose(); $fs.Dispose()
    $hash = (Get-FileHash "dist\$name.zip" -Algorithm SHA256).Hash.ToLower()
    "$hash  $name.zip" | Set-Content "dist\$name.zip.sha256" -Encoding ascii
    Write-Host "Архив: dist\$name.zip"
    Write-Host "SHA-256: $hash"
}
