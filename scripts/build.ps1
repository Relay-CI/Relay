param(
    [string]$OutputDir,
    [string]$GOOS = $env:GOOS,
    [string]$GOARCH = $env:GOARCH
)

$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent $PSScriptRoot
if (-not $GOOS) {
    $GOOS = (& go env GOOS).Trim()
}
if (-not $GOARCH) {
    $GOARCH = (& go env GOARCH).Trim()
}

$ext = if ($GOOS -eq "windows") { ".exe" } else { "" }
if (-not $OutputDir) {
    $OutputDir = Join-Path $repoRoot (Join-Path "dist" "$GOOS-$GOARCH")
}
$OutputDir = [System.IO.Path]::GetFullPath($OutputDir)
New-Item -ItemType Directory -Force -Path $OutputDir | Out-Null

$oldGOOS = $env:GOOS
$oldGOARCH = $env:GOARCH
$env:GOOS = $GOOS
$env:GOARCH = $GOARCH

try {
    Write-Host "Building dashboard bundle..."
    Push-Location (Join-Path $repoRoot "relayd/ui-src")
    try {
        & npm install --prefer-offline
        & npm run build
    } finally {
        Pop-Location
    }

    Write-Host "Building relayd for $GOOS/$GOARCH..."
    Push-Location (Join-Path $repoRoot "relayd")
    try {
        & go build "-o" (Join-Path $OutputDir "relayd$ext") .
    } finally {
        Pop-Location
    }

    Write-Host "Building station for $GOOS/$GOARCH..."
    Push-Location (Join-Path $repoRoot "station")
    try {
        & go build "-o" (Join-Path $OutputDir "station$ext") .
        if ($GOOS -eq "windows") {
            $linuxArch = if ($GOARCH -eq "arm64") { "arm64" } else { "amd64" }
            $env:GOOS = "linux"
            $env:GOARCH = $linuxArch
            Write-Host "Building station Linux sidecar for WSL2 (linux/$linuxArch)..."
            & go build "-o" (Join-Path $OutputDir "station-linux") .
            $env:GOOS = $GOOS
            $env:GOARCH = $GOARCH
        }
    } finally {
        Pop-Location
    }
}
finally {
    if ($null -eq $oldGOOS -or $oldGOOS -eq "") {
        Remove-Item Env:GOOS -ErrorAction SilentlyContinue
    } else {
        $env:GOOS = $oldGOOS
    }
    if ($null -eq $oldGOARCH -or $oldGOARCH -eq "") {
        Remove-Item Env:GOARCH -ErrorAction SilentlyContinue
    } else {
        $env:GOARCH = $oldGOARCH
    }
}

Write-Host ""
Write-Host "Built Relay package in $OutputDir"
Write-Host " - $(Join-Path $OutputDir "relayd$ext")"
Write-Host " - $(Join-Path $OutputDir "station$ext")"
if ($GOOS -eq "windows") {
    Write-Host " - $(Join-Path $OutputDir "station-linux")"
}
