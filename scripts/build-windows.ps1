# Builds bin\scrawl.exe for Windows: a GUI program without a console window, with an icon, a manifest
# (DPI, long paths) and version information. With -Package it also puts into dist\ the
# release archive and its SHA-256. Requires Go (the version from go.mod) and Git for
# Windows: sh and patch for preparing Gio come from it.
#   powershell -ExecutionPolicy Bypass -File scripts\build-windows.ps1 [-Version v0.1.0] [-Package]
param(
    [string]$Out = "bin\scrawl.exe",
    [string]$Version = "",
    [switch]$Package
)
$ErrorActionPreference = "Stop"
# Without cgo: it is not needed on Windows, and if the machine has gcc, Go would otherwise
# try to build the C files of the Linux packages.
$env:CGO_ENABLED = "0"
Set-Location (Split-Path $PSScriptRoot -Parent)

# git.exe may be in Git\cmd, Git\bin or Git\mingw64\bin: go up until usr\bin\sh.exe is found.
$gitUsrBin = $null
$dir = Split-Path (Get-Command git.exe -ErrorAction Stop).Source
while ($dir -and -not $gitUsrBin) {
    if (Test-Path (Join-Path $dir "usr\bin\sh.exe")) { $gitUsrBin = Join-Path $dir "usr\bin" }
    $dir = Split-Path $dir
}
if (-not $gitUsrBin) { throw "usr\bin\sh.exe was not found next to git.exe: Git for Windows is required" }
$env:PATH = "$gitUsrBin;$env:PATH"

# Gio of the required version with our patch (a rerun does nothing while the version has
# not changed).
& sh scripts/setup-gio.sh
if ($LASTEXITCODE -ne 0) { throw "setup-gio.sh exited with code $LASTEXITCODE" }

if (-not $Version) { $Version = (& git describe --tags --always --dirty 2>$null) }
$major = 0; $minor = 0; $patch = 0
if ($Version -match '^v?(\d+)\.(\d+)\.(\d+)') { $major = [int]$Matches[1]; $minor = [int]$Matches[2]; $patch = [int]$Matches[3] }

# The icon, manifest and file version go into the .exe as a resource (.syso next to
# main.go, not stored in git).
$syso = "cmd\scrawl\resource_windows_amd64.syso"
Push-Location packaging\windows
try {
    & go run github.com/josephspurrier/goversioninfo/cmd/goversioninfo@v1.4.1 -64 `
        -ver-major $major -ver-minor $minor -ver-patch $patch `
        -product-ver-major $major -product-ver-minor $minor -product-ver-patch $patch `
        -file-version "$Version" -product-version "$Version" `
        -o "..\..\$syso" versioninfo.json
    if ($LASTEXITCODE -ne 0) { throw "goversioninfo exited with code $LASTEXITCODE" }
} finally { Pop-Location }

$ldflags = "-H=windowsgui"
if ($Version) { $ldflags += " -X github.com/jexme/scrawl/internal/version.Version=$Version" }
New-Item -ItemType Directory -Force (Split-Path $Out) | Out-Null
& go build -trimpath -ldflags $ldflags -o $Out ./cmd/scrawl
if ($LASTEXITCODE -ne 0) { throw "go build exited with code $LASTEXITCODE" }
Write-Host "Done: ${Out} (${Version})"

if ($Package) {
    $name = "scrawl-$Version-windows-amd64"
    $dir = Join-Path "dist" $name
    Remove-Item $dir, "dist\$name.zip", "dist\$name.zip.sha256", "dist\$name.exe", "dist\$name.exe.sha256" -Recurse -Force -ErrorAction SilentlyContinue
    New-Item -ItemType Directory -Force $dir | Out-Null
    Copy-Item $Out (Join-Path $dir "scrawl.exe")
    # The bare .exe is also published on its own: the program is portable, the archive only adds docs.
    Copy-Item $Out "dist\$name.exe"
    $exeHash = (Get-FileHash "dist\$name.exe" -Algorithm SHA256).Hash.ToLower()
    "$exeHash  $name.exe" | Set-Content "dist\$name.exe.sha256" -Encoding ascii
    Copy-Item LICENSE, THIRD_PARTY_NOTICES.md, README.md, docs\translations\README.ru.md, docs\WINDOWS.md $dir
    # We write the archive ourselves: Compress-Archive and ZipFile.CreateFromDirectory in
    # Windows PowerShell 5.1 put backslashes in the names, and such an archive extracts
    # incorrectly on other systems.
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
    Write-Host "Archive: dist\$name.zip"
    Write-Host "SHA-256: $hash"
}
