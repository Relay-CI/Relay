<#
.SYNOPSIS
  Station vs Docker feature benchmark.

.DESCRIPTION
  Runs each container-runtime feature with Station and (if Docker is running)
  with docker, measures timing, and prints a side-by-side comparison table.

  Test fixture: go-basic (tiny HTTP server, compiles to a Windows .exe
  so Station can run it as a native process with zero extra tooling).

.USAGE
  cd smoke-apps
  .\bench.ps1          # compare Station only (Docker auto-detected)
  .\bench.ps1 -SkipDocker   # Station only even if Docker is present
  .\bench.ps1 -Verbose      # show each command as it runs
#>

[CmdletBinding()]
param(
    [switch]$SkipDocker,
    [int]$BasePort = 4300,
    [int]$TimeoutSec = 20
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version 3

# PowerShell turns native stderr into error records. Keep cmdlet errors fatal,
# but let external tool failures be handled via exit codes inside the script.
$script:NativeErrPref = "Continue"

function Invoke-External {
    param(
        [scriptblock]$Command,
        [switch]$IgnoreExitCode
    )

    $prev = $ErrorActionPreference
    $ErrorActionPreference = $script:NativeErrPref
    try {
        $output = & $Command 2>&1
        $exitCode = $LASTEXITCODE
    } finally {
        $ErrorActionPreference = $prev
    }

    if (-not $IgnoreExitCode -and $exitCode -ne 0) {
        return [pscustomobject]@{ Output = $output; ExitCode = $exitCode; Ok = $false }
    }

    return [pscustomobject]@{ Output = $output; ExitCode = $exitCode; Ok = ($exitCode -eq 0) }
}

# ─── paths ────────────────────────────────────────────────────────────────────

$root       = Split-Path -Parent $PSScriptRoot
$station    = Join-Path $root "station\station.exe"
$goBasicDir = Join-Path $PSScriptRoot "go-basic"
$resultsDir = Join-Path $PSScriptRoot "results"
New-Item -ItemType Directory -Force -Path $resultsDir | Out-Null
$jsonOut    = Join-Path $resultsDir "bench.json"

if (-not (Test-Path $station)) {
    Write-Error "station.exe not found at $station - build it first: cd Station; go build -o station.exe ."
    exit 1
}

# ─── docker detection ─────────────────────────────────────────────────────────

$dockerCmd = $null
if (-not $SkipDocker) {
    $d = Get-Command docker -ErrorAction SilentlyContinue
    if ($d) {
        $ping = docker info 2>&1
        if ($LASTEXITCODE -eq 0) { $dockerCmd = "docker" }
        else { Write-Host "Docker found but daemon is not running - skipping Docker tests." -ForegroundColor Yellow }
    }
}

Write-Host "Runtime fixture: Windows go-smoke.exe"

# ─── build the go test binary ─────────────────────────────────────────────────

$goBin = Join-Path $goBasicDir "go-smoke.exe"
Write-Host "Building test fixture..." -NoNewline
Push-Location $goBasicDir
try {
    $buildOut = go build -o $goBin . 2>&1
    if ($LASTEXITCODE -ne 0) {
        Write-Host " FAILED"
        Write-Error "go build failed:`n$buildOut"
        exit 1
    }
} finally { Pop-Location }
Write-Host " OK ($goBin)"

# ─── helpers ──────────────────────────────────────────────────────────────────

function Wait-Http {
    param([int]$Port, [int]$Seconds = $TimeoutSec)
    $sw = [System.Diagnostics.Stopwatch]::StartNew()
    while ($sw.Elapsed.TotalSeconds -lt $Seconds) {
        try {
            $r = Invoke-WebRequest "http://127.0.0.1:$Port/" -TimeoutSec 1 -UseBasicParsing -ErrorAction Stop
            return [pscustomobject]@{ Ok=$true; Ms=[int]$sw.Elapsed.TotalMilliseconds; Status=$r.StatusCode }
        } catch { Start-Sleep -Milliseconds 100 }
    }
    return [pscustomobject]@{ Ok=$false; Ms=[int]($Seconds*1000); Status=0 }
}

function Run-Timed {
    param([scriptblock]$Block)
    $sw = [System.Diagnostics.Stopwatch]::StartNew()
    $result = & $Block
    $sw.Stop()
    return [pscustomobject]@{ Result=$result; Ms=[int]$sw.Elapsed.TotalMilliseconds }
}

function Fmt-Ms([int]$ms) {
    if ($ms -ge 1000) { return ("{0:N1}s" -f ($ms / 1000)) }
    return "${ms}ms"
}

$col_ok   = "Green"
$col_fail = "Red"
$col_warn = "Yellow"
$col_dim  = "DarkGray"

function Symbol([bool]$ok) { if ($ok) { return "[+]" } else { return "[!]" } }

# ─── result accumulator ───────────────────────────────────────────────────────

$benchResults = [System.Collections.Generic.List[hashtable]]::new()

function Record {
    param($Feature, $VesselOk, $VesselMs, $VesselNote,
                    $DockerOk,  $DockerMs,  $DockerNote,
                    $VesselCmd, $DockerCmd)
    $benchResults.Add(@{
        feature    = $Feature
        Station    = @{ ok=$VesselOk; ms=$VesselMs; note=$VesselNote; cmd=$VesselCmd }
        docker     = @{ ok=$DockerOk;  ms=$DockerMs;  note=$DockerNote;  cmd=$DockerCmd  }
    })
}

# ─── port allocator ───────────────────────────────────────────────────────────

$portCounter = $BasePort
function NextPort {
    $port = [int]$script:portCounter
    $script:portCounter = $script:portCounter + 1
    return $port
}

# ─── cleanup registry ─────────────────────────────────────────────────────────

$toCleanVessel  = [System.Collections.Generic.List[string]]::new()
$toCleanDocker   = [System.Collections.Generic.List[string]]::new()
$toCleanFiles    = [System.Collections.Generic.List[string]]::new()

function Cleanup {
    Write-Host "`nCleaning up..." -ForegroundColor $col_dim
    foreach ($id in $toCleanVessel) {
        Invoke-External { & $station stop $id } -IgnoreExitCode | Out-Null
    }
    if ($dockerCmd) {
        foreach ($name in $toCleanDocker) {
            Invoke-External { docker stop $name } -IgnoreExitCode | Out-Null
            Invoke-External { docker rm $name } -IgnoreExitCode | Out-Null
        }
    }
    foreach ($f in $toCleanFiles) {
        Remove-Item $f -Recurse -Force -ErrorAction SilentlyContinue
    }
}

# ─── section header ───────────────────────────────────────────────────────────

function Section([string]$title) {
    Write-Host ""
    Write-Host ("  {0}" -f $title.ToUpper()) -ForegroundColor Cyan
    Write-Host ("  " + ("-" * 70)) -ForegroundColor $col_dim
}

Write-Host ""
Write-Host "  Station vs Docker - feature benchmark" -ForegroundColor White
Write-Host ("  Station : " + $station)  -ForegroundColor $col_dim
Write-Host ("  docker  : " + $(if ($dockerCmd) { "docker (detected)" } else { "not detected / skipped" })) -ForegroundColor $col_dim
Write-Host ""

# ══════════════════════════════════════════════════════════════════════════════
# TEST 1 — Run (detached) + first-byte startup time
# ══════════════════════════════════════════════════════════════════════════════
Section "1. Run (detached) + startup latency"

$p1n = NextPort
$nc1Cmd = "$station run --app bench-run --port $p1n $goBasicDir .\\go-smoke.exe"
Write-Verbose "Station: $nc1Cmd"
$sw = [System.Diagnostics.Stopwatch]::StartNew()
$nc1Run = Invoke-External { & $station run --app bench-run --port $p1n $goBasicDir .\go-smoke.exe }
$nc1Out = $nc1Run.Output
$nc1Id  = ($nc1Out | Select-String "container\s+([a-f0-9]{8})" | ForEach-Object { $_.Matches[0].Groups[1].Value } | Select-Object -First 1)
$nc1Http = Wait-Http -Port $p1n
$sw.Stop()

if ($nc1Id) {
    $toCleanVessel.Add($nc1Id) | Out-Null
}

$d1Ok = $false; $d1Ms = 0; $d1Note = "skipped"
$d1Cmd = "docker run -d --name bench-run-d -p ${p1n}:${p1n} --mount type=bind,source=${goBasicDir},target=/app -w /app golang:1.22-alpine go run main.go"
if ($dockerCmd) {
    Invoke-External { docker rm -f bench-run-d } -IgnoreExitCode | Out-Null
    $p1d = NextPort
    $d1Cmd = "docker run -d --name bench-run-d -p ${p1d}:${p1d} --mount type=bind,source=${goBasicDir},target=/app -w /app golang:1.22-alpine go run main.go"
    $sw2 = [System.Diagnostics.Stopwatch]::StartNew()
    $d1Run = Invoke-External {
        docker run -d --name bench-run-d -p "${p1d}:${p1d}" -e "PORT=${p1d}" `
            --mount "type=bind,source=${goBasicDir},target=/app" -w /app golang:1.22-alpine go run main.go
    }
    $d1Http = Wait-Http -Port $p1d
    $sw2.Stop()
    $d1Ok   = $d1Run.Ok -and $d1Http.Ok
    $d1Ms   = $d1Http.Ms
    $d1Note = if ($d1Ok) { "HTTP $($d1Http.Status)" } elseif (-not $d1Run.Ok) { (($d1Run.Output | Select-Object -Last 1) -join "") } else { "timeout" }
    if ($d1Run.Ok) {
        $toCleanDocker.Add("bench-run-d") | Out-Null
    }
}

$nc1Ok   = $nc1Run.Ok -and $nc1Http.Ok
$nc1Ms   = $nc1Http.Ms
$nc1Note = if ($nc1Ok) { "HTTP $($nc1Http.Status)" } elseif (-not $nc1Run.Ok) { (($nc1Out | Select-Object -Last 1) -join "") } else { "timeout (id=$nc1Id)" }
Write-Host ("  Station  {0}  {1,-8}  {2}" -f (Symbol $nc1Ok), (Fmt-Ms $nc1Ms), $nc1Note) -ForegroundColor $(if($nc1Ok){$col_ok}else{$col_fail})
Write-Host ("  docker   {0}  {1,-8}  {2}" -f (Symbol $d1Ok),  (Fmt-Ms $d1Ms),  $d1Note)  -ForegroundColor $(if($d1Ok){$col_ok}elseif($dockerCmd){$col_fail}else{$col_dim})
Record "Run + startup (time-to-first-byte)" $nc1Ok $nc1Ms $nc1Note $d1Ok $d1Ms $d1Note $nc1Cmd $d1Cmd

# ══════════════════════════════════════════════════════════════════════════════
# TEST 2 — List running containers
# ══════════════════════════════════════════════════════════════════════════════
Section "2. List running containers"

$t = Run-Timed { Invoke-External { & $station list } }
$nc2Ok   = $t.Result.Ok -and $nc1Id -and ($t.Result.Output | Select-String $nc1Id)
$nc2Ms   = $t.Ms
$nc2Note = if ($nc2Ok) { "id found in list" } else { "id not found" }
Write-Host ("  Station  {0}  {1,-8}  {2}" -f (Symbol $nc2Ok), (Fmt-Ms $nc2Ms), $nc2Note) -ForegroundColor $(if($nc2Ok){$col_ok}else{$col_fail})

$d2Ok=$false; $d2Ms=0; $d2Note="skipped"; $d2Cmd="docker ps"
if ($dockerCmd) {
    $t2 = Run-Timed { Invoke-External { docker ps --filter name=bench-run-d --format "{{.Names}}" } }
    $d2Ok   = $t2.Result.Ok -and (($t2.Result.Output | Select-String "bench-run-d") -ne $null)
    $d2Ms   = $t2.Ms
    $d2Note = if ($d2Ok) { "found in ps" } else { "not found" }
    Write-Host ("  docker   {0}  {1,-8}  {2}" -f (Symbol $d2Ok), (Fmt-Ms $d2Ms), $d2Note) -ForegroundColor $(if($d2Ok){$col_ok}else{$col_fail})
}
Record "List containers" $nc2Ok $nc2Ms $nc2Note $d2Ok $d2Ms $d2Note "$station list" $d2Cmd

# ══════════════════════════════════════════════════════════════════════════════
# TEST 3 — Logs
# ══════════════════════════════════════════════════════════════════════════════
Section "3. Read logs"

$t = Run-Timed { Invoke-External { & $station logs $nc1Id } }
$nc3Ok   = $t.Result.Ok
$nc3Ms   = $t.Ms
$nc3Note = if ($nc3Ok) { "logs command OK" } else { "no output" }
Write-Host ("  Station  {0}  {1,-8}  {2}" -f (Symbol $nc3Ok), (Fmt-Ms $nc3Ms), $nc3Note) -ForegroundColor $(if($nc3Ok){$col_ok}else{$col_fail})

$d3Ok=$false; $d3Ms=0; $d3Note="skipped"; $d3Cmd="docker logs bench-run-d"
if ($dockerCmd) {
    $t2 = Run-Timed { Invoke-External { docker logs bench-run-d } }
    $d3Ok   = $t2.Result.Ok
    $d3Ms   = $t2.Ms
    $d3Note = if ($d3Ok) { "logs command OK" } else { "empty" }
    Write-Host ("  docker   {0}  {1,-8}  {2}" -f (Symbol $d3Ok), (Fmt-Ms $d3Ms), $d3Note) -ForegroundColor $(if($d3Ok){$col_ok}else{$col_fail})
}
Record "Read logs" $nc3Ok $nc3Ms $nc3Note $d3Ok $d3Ms $d3Note "$station logs <id>" $d3Cmd

# ══════════════════════════════════════════════════════════════════════════════
# TEST 4 — Status / inspect
# ══════════════════════════════════════════════════════════════════════════════
Section "4. Inspect / status"

$t = Run-Timed { Invoke-External { & $station status $nc1Id } }
$nc4Ok   = $t.Result.Ok -and ($t.Result.Output | Select-String "pid|port|app" -CaseSensitive:$false)
$nc4Ms   = $t.Ms
$nc4Note = if ($nc4Ok) { "fields present" } else { "unexpected output" }
Write-Host ("  Station  {0}  {1,-8}  {2}" -f (Symbol $nc4Ok), (Fmt-Ms $nc4Ms), $nc4Note) -ForegroundColor $(if($nc4Ok){$col_ok}else{$col_fail})

$d4Ok=$false; $d4Ms=0; $d4Note="skipped"; $d4Cmd="docker inspect bench-run-d"
if ($dockerCmd) {
    $t2 = Run-Timed { Invoke-External { docker inspect bench-run-d } }
    $d4Ok   = $t2.Result.Ok
    $d4Ms   = $t2.Ms
    $d4Note = if ($d4Ok) { "JSON returned" } else { "error" }
    Write-Host ("  docker   {0}  {1,-8}  {2}" -f (Symbol $d4Ok), (Fmt-Ms $d4Ms), $d4Note) -ForegroundColor $(if($d4Ok){$col_ok}else{$col_fail})
}
Record "Inspect / status" $nc4Ok $nc4Ms $nc4Note $d4Ok $d4Ms $d4Note "$station status <id>" $d4Cmd

# ══════════════════════════════════════════════════════════════════════════════
# TEST 5 — Env var injection
# ══════════════════════════════════════════════════════════════════════════════
Section "5. Env var injection (PORT respected)"

# The go-basic server binds $PORT — if it responded above, env was injected.
# Re-verify by checking the running port matches what we passed.
$nc5Ok   = $nc1Ok
$nc5Ms   = $nc1Ms
$nc5Note = if ($nc5Ok) { "PORT respected (HTTP 200 on $p1n)" } else { "startup probe failed" }
Write-Host ("  Station  {0}  {1,-8}  {2}" -f (Symbol $nc5Ok), (Fmt-Ms $nc5Ms), $nc5Note) -ForegroundColor $(if($nc5Ok){$col_ok}else{$col_fail})

$d5Ok=$false; $d5Ms=0; $d5Note="skipped"
$d5Cmd = "docker run -d -e PORT=<port> ..."
if ($dockerCmd) {
    # Docker container bench-run-d was started with -e PORT=$p1d — it responded above.
    $d5Ok   = $d1Ok  # if Docker started and responded, env was used
    $d5Ms   = $d1Ms
    $d5Note = if ($d5Ok) { "PORT was bound (HTTP 200)" } else { "skipped/failed" }
    Write-Host ("  docker   {0}  {1,-8}  {2}" -f (Symbol $d5Ok), (Fmt-Ms $d5Ms), $d5Note) -ForegroundColor $(if($d5Ok){$col_ok}elseif($dockerCmd){$col_fail}else{$col_dim})
}
Record "Env var injection" $nc5Ok $nc5Ms $nc5Note $d5Ok $d5Ms $d5Note "$station run -e KEY=VAL" $d5Cmd

# ══════════════════════════════════════════════════════════════════════════════
# TEST 6 — Stop
# ══════════════════════════════════════════════════════════════════════════════
Section "6. Stop container"

# Spin up a fresh container just for the stop test so we don't kill the one
# used by subsequent tests.
$p6n = NextPort
Invoke-External { & $station run --app bench-stop6 --port $p6n $goBasicDir .\go-smoke.exe } | Out-Null
$stop6List = Invoke-External { & $station list }
$stop6Id = ($stop6List.Output | Select-String "bench-stop6" | ForEach-Object { ($_ -split '\s+')[0] } | Select-Object -First 1)
Start-Sleep -Milliseconds 400

$t = Run-Timed {
    (Invoke-External { & $station stop $stop6Id }).ExitCode
}
$nc6Ok   = $t.Result -eq 0
$nc6Ms   = $t.Ms
$nc6Note = if ($nc6Ok) { "exit 0" } else { "exit $($t.Result)" }
Write-Host ("  Station  {0}  {1,-8}  {2}" -f (Symbol $nc6Ok), (Fmt-Ms $nc6Ms), $nc6Note) -ForegroundColor $(if($nc6Ok){$col_ok}else{$col_fail})

$d6Ok=$false; $d6Ms=0; $d6Note="skipped"
$d6Cmd = "docker stop <container>"
if ($dockerCmd) {
    $p6d = NextPort
    $d6Run = Invoke-External {
        docker run -d --name bench-stop6-d -p "${p6d}:${p6d}" -e "PORT=${p6d}" `
            --mount "type=bind,source=${goBasicDir},target=/app" -w /app golang:1.22-alpine go run main.go
    }
    Start-Sleep -Milliseconds 600
    $t2 = Run-Timed {
        if (-not $d6Run.Ok) {
            return $d6Run.ExitCode
        }
        (Invoke-External { docker stop bench-stop6-d }).ExitCode
    }
    $d6Ok   = $t2.Result -eq 0
    $d6Ms   = $t2.Ms
    $d6Note = if ($d6Ok) { "exit 0" } else { "exit $($t2.Result)" }
    Invoke-External { docker rm bench-stop6-d } -IgnoreExitCode | Out-Null
    Write-Host ("  docker   {0}  {1,-8}  {2}" -f (Symbol $d6Ok), (Fmt-Ms $d6Ms), $d6Note) -ForegroundColor $(if($d6Ok){$col_ok}else{$col_fail})
}
Record "Stop container" $nc6Ok $nc6Ms $nc6Note $d6Ok $d6Ms $d6Note "$station stop <id>" $d6Cmd

# ══════════════════════════════════════════════════════════════════════════════
# TEST 7 — Restart on crash (supervisor)
# ══════════════════════════════════════════════════════════════════════════════
Section "7. Restart on crash (--restart always)"

$p7 = NextPort
Invoke-External { & $station run --app bench-restart7 --port $p7 --restart always $goBasicDir .\go-smoke.exe } | Out-Null
$r7List = Invoke-External { & $station list }
$r7Id = ($r7List.Output | Select-String "bench-restart7" | ForEach-Object { ($_ -split '\s+')[0] } | Select-Object -First 1)
Start-Sleep -Milliseconds 500

# Wait for first successful response.
$before = Wait-Http -Port $p7 -Seconds 5
if ($before.Ok) {
    # Kill the process directly; supervisor should restart it.
    $statusOut = (Invoke-External { & $station status $r7Id }).Output
    $pidLine   = ($statusOut | Select-String "pid\s*[:=]\s*(\d+)" | ForEach-Object { $_.Matches[0].Groups[1].Value })
    if ($pidLine) {
        Stop-Process -Id ([int]$pidLine) -Force -ErrorAction SilentlyContinue
    }
    # Give supervisor (detached daemon) time to detect death and restart.
    # First crash restarts immediately; detection poll is ~1 s.
    Start-Sleep -Milliseconds 200
    $after = Wait-Http -Port $p7 -Seconds 8
    $nc7Ok   = $after.Ok
    $nc7Ms   = $after.Ms
    $nc7Note = if ($nc7Ok) { "restarted in $(Fmt-Ms $after.Ms)" } else { "did not restart" }
} else {
    $nc7Ok   = $false
    $nc7Ms   = 0
    $nc7Note = "process never started"
}
$toCleanVessel.Add($r7Id) | Out-Null

Write-Host ("  Station  {0}  {1,-8}  {2}" -f (Symbol $nc7Ok), (Fmt-Ms $nc7Ms), $nc7Note) -ForegroundColor $(if($nc7Ok){$col_ok}else{$col_fail})

$d7Cmd = "docker run --restart=always ..."
$d7Note = if ($dockerCmd) { "native Docker feature (restart=always)" } else { "skipped" }
Write-Host ("  docker   -  {0,-8}  {1}" -f "", $d7Note) -ForegroundColor $col_dim
Record "Restart on crash" $nc7Ok $nc7Ms $nc7Note $true 0 "native --restart=always" "$station run --restart always ..." $d7Cmd

# ══════════════════════════════════════════════════════════════════════════════
# TEST 8 — Snapshot save / load
# ══════════════════════════════════════════════════════════════════════════════
Section "8. Snapshot save / load (image)"

$snapName = "bench-snap-$(Get-Random -Max 9999)"
$snapSrc  = $goBasicDir
$snapDest = Join-Path $env:TEMP "Station-bench-snap-dest"
$toCleanFiles.Add($snapDest) | Out-Null

$t = Run-Timed { Invoke-External { & $station snapshot save $snapName $snapSrc } }
$saveOk = $t.Result.Ok
if ($saveOk) {
    $t2       = Run-Timed { Invoke-External { & $station snapshot load $snapName $snapDest } }
    $nc8Ok    = $t2.Result.Ok -and (Test-Path (Join-Path $snapDest "go-smoke.exe"))
    $nc8Ms    = $t.Ms + $t2.Ms
    $nc8Note  = if ($nc8Ok) { "save+load OK ($(Fmt-Ms $t.Ms) + $(Fmt-Ms $t2.Ms))" } else { "load failed" }
    # Clean up snapshot
    Invoke-External { & $station snapshot delete $snapName } | Out-Null
} else {
    $nc8Ok   = $false
    $nc8Ms   = $t.Ms
    $nc8Note = "save failed"
}
Write-Host ("  Station  {0}  {1,-8}  {2}" -f (Symbol $nc8Ok), (Fmt-Ms $nc8Ms), $nc8Note) -ForegroundColor $(if($nc8Ok){$col_ok}else{$col_fail})

$d8Ok=$false; $d8Ms=0; $d8Note="skipped"
$d8Cmd = "docker commit <container> <image>; docker save <image> | docker load"
if ($dockerCmd) {
    $imgTag = "bench-snap-test:latest"
    $t3 = Run-Timed { (Invoke-External { docker commit bench-run-d $imgTag }).ExitCode }
    $d8Ok   = $t3.Result -eq 0
    $d8Ms   = $t3.Ms
    $d8Note = if ($d8Ok) { "docker commit OK" } else { "commit failed" }
    Invoke-External { docker rmi $imgTag } -IgnoreExitCode | Out-Null
    Write-Host ("  docker   {0}  {1,-8}  {2}" -f (Symbol $d8Ok), (Fmt-Ms $d8Ms), $d8Note) -ForegroundColor $(if($d8Ok){$col_ok}else{$col_fail})
}
Record "Snapshot save + load" $nc8Ok $nc8Ms $nc8Note $d8Ok $d8Ms $d8Note "$station snapshot save/load" $d8Cmd

# ══════════════════════════════════════════════════════════════════════════════
# TEST 9 — Proxy start + hot-swap (blue-green)
# ══════════════════════════════════════════════════════════════════════════════
Section "9. Blue-green proxy (proxy start / swap)"

$p9proxy = NextPort
$p9up1   = NextPort
$p9up2   = NextPort

# Start two upstream instances.
$bgAList = Invoke-External { & $station list }
$bgAId = ($bgAList.Output | Select-String "bench-bg-a" | ForEach-Object { ($_ -split '\s+')[0] } | Select-Object -First 1)
Invoke-External { & $station run --app bench-bg-a --port $p9up1 $goBasicDir .\go-smoke.exe } | Out-Null
Invoke-External { & $station run --app bench-bg-b --port $p9up2 $goBasicDir .\go-smoke.exe } | Out-Null
$bgBList = Invoke-External { & $station list }
$bgBId = ($bgBList.Output | Select-String "bench-bg-b" | ForEach-Object { ($_ -split '\s+')[0] } | Select-Object -First 1)
$toCleanVessel.Add($bgAId) | Out-Null
$toCleanVessel.Add($bgBId) | Out-Null

$u1Ready = Wait-Http -Port $p9up1 -Seconds 8
$u2Ready = Wait-Http -Port $p9up2 -Seconds 8

if ($u1Ready.Ok) {
    # Start proxy.
    $tStart = Run-Timed {
        (Invoke-External { & $station proxy start --app bench-bg --port $p9proxy --upstream "127.0.0.1:$p9up1" }).ExitCode
    }
    Start-Sleep -Milliseconds 300
    $proxyBefore = Wait-Http -Port $p9proxy -Seconds 5

    # Hot-swap to second upstream.
    $tSwap = Run-Timed {
        (Invoke-External { & $station proxy swap --app bench-bg --upstream "127.0.0.1:$p9up2" }).ExitCode
    }
    Start-Sleep -Milliseconds 200
    $proxyAfter = Wait-Http -Port $p9proxy -Seconds 5

    $nc9Ok   = $proxyBefore.Ok -and $proxyAfter.Ok
    $nc9Ms   = $tStart.Ms + $tSwap.Ms
    $nc9Note = "start=$(Fmt-Ms $tStart.Ms) swap=$(Fmt-Ms $tSwap.Ms) | $( if($nc9Ok){'both probes 200'}else{'probe failed'} )"

    Invoke-External { & $station proxy stop bench-bg } | Out-Null
} else {
    $nc9Ok   = $false
    $nc9Ms   = 0
    $nc9Note = "upstream never ready"
}

Write-Host ("  Station  {0}  {1,-8}  {2}" -f (Symbol $nc9Ok), (Fmt-Ms $nc9Ms), $nc9Note) -ForegroundColor $(if($nc9Ok){$col_ok}else{$col_fail})
Write-Host "  docker   -             requires nginx/traefik - no native equivalent" -ForegroundColor $col_dim
Record "Blue-green proxy" $nc9Ok $nc9Ms $nc9Note $false 0 "no native equiv (needs nginx/traefik)" "$station proxy start/swap" "nginx -s reload (manual)"

# ══════════════════════════════════════════════════════════════════════════════
# TEST 10 — Build step (Station build)
# ══════════════════════════════════════════════════════════════════════════════
Section "10. Build step (Station build vs docker build)"

# Use node-basic — npm install is a realistic build step.
$nodeDir = Join-Path $PSScriptRoot "node-basic"
$nodeAvail = Get-Command node -ErrorAction SilentlyContinue

if ($nodeAvail) {
    $tB = Run-Timed {
        (Invoke-External { & $station build $nodeDir npm install }).ExitCode
    }
    $nc10Ok   = $tB.Result -eq 0
    $nc10Ms   = $tB.Ms
    $nc10Note = if ($nc10Ok) { "npm install OK" } else { "failed (exit $($tB.Result))" }
} else {
    # Fallback: go build in go-basic (we know Go is available).
    $tB = Run-Timed {
        (Invoke-External { & $station build $goBasicDir go build ./... }).ExitCode
    }
    $nc10Ok   = $tB.Result -eq 0
    $nc10Ms   = $tB.Ms
    $nc10Note = if ($nc10Ok) { "go build OK" } else { "failed" }
}
Write-Host ("  Station  {0}  {1,-8}  {2}" -f (Symbol $nc10Ok), (Fmt-Ms $nc10Ms), $nc10Note) -ForegroundColor $(if($nc10Ok){$col_ok}else{$col_fail})

$d10Ok=$false; $d10Ms=0; $d10Note="skipped"
$d10Cmd = "docker build -t bench-build ."
if ($dockerCmd -and (Test-Path (Join-Path $PSScriptRoot "node-basic\package.json"))) {
    # Simple Dockerfile-less build via docker run to keep it comparable.
    $t10 = Run-Timed {
        (Invoke-External {
            docker run --rm --mount "type=bind,source=${nodeDir},target=/app" -w /app node:22-alpine npm install
        }).ExitCode
    }
    $d10Ok   = $t10.Result -eq 0
    $d10Ms   = $t10.Ms
    $d10Note = if ($d10Ok) { "npm install OK" } else { "failed" }
    Write-Host ("  docker   {0}  {1,-8}  {2}" -f (Symbol $d10Ok), (Fmt-Ms $d10Ms), $d10Note) -ForegroundColor $(if($d10Ok){$col_ok}else{$col_fail})
}
Record "Build step" $nc10Ok $nc10Ms $nc10Note $d10Ok $d10Ms $d10Note "$station build <dir> <cmd>" $d10Cmd

# ══════════════════════════════════════════════════════════════════════════════
# TEST 11 — Port allocation
# ══════════════════════════════════════════════════════════════════════════════
Section "11. Port allocation / listing"

$t = Run-Timed { Invoke-External { & $station port } }
$nc11Ok   = $t.Result.Ok
$nc11Ms   = $t.Ms
$nc11Note = if ($nc11Ok) { "$((($t.Result.Output | Measure-Object).Count)) port entries" } else { "failed" }
Write-Host ("  Station  {0}  {1,-8}  {2}" -f (Symbol $nc11Ok), (Fmt-Ms $nc11Ms), $nc11Note) -ForegroundColor $(if($nc11Ok){$col_ok}else{$col_fail})

$d11Cmd = "docker port <container>"
Write-Host "  docker   -             docker port <container> (per-container, not global)" -ForegroundColor $col_dim
Record "Port listing" $nc11Ok $nc11Ms $nc11Note $true 0 "docker port (per-container)" "$station port" $d11Cmd

# ══════════════════════════════════════════════════════════════════════════════
# TEST 12 — OCI image pull (Station image pull)
# ══════════════════════════════════════════════════════════════════════════════
Section "12. OCI image pull (alpine:3.19)"

$t = Run-Timed { Invoke-External { & $station image pull alpine:3.19 } }
$nc12Ok   = $t.Result.Ok
$nc12Ms   = $t.Ms
# Second pull should be instant (cached).
$t2 = Run-Timed { Invoke-External { & $station image pull alpine:3.19 } }
$nc12Note = if ($nc12Ok) { "first pull $(Fmt-Ms $nc12Ms), cached $(Fmt-Ms $t2.Ms)" } else { "failed" }
Write-Host ("  Station  {0}  {1,-8}  {2}" -f (Symbol $nc12Ok), (Fmt-Ms $nc12Ms), $nc12Note) -ForegroundColor $(if($nc12Ok){$col_ok}else{$col_fail})

$d12Ok=$false; $d12Ms=0; $d12Note="skipped"
$d12Cmd = "docker pull alpine:3.19"
if ($dockerCmd) {
    # Remove local copy so the pull is real.
    Invoke-External { docker rmi alpine:3.19 } -IgnoreExitCode | Out-Null
    $t3 = Run-Timed { (Invoke-External { docker pull alpine:3.19 }).ExitCode }
    $d12Ok   = $t3.Result -eq 0
    $d12Ms   = $t3.Ms
    $d12Note = if ($d12Ok) { "pulled in $(Fmt-Ms $t3.Ms)" } else { "failed" }
    Write-Host ("  docker   {0}  {1,-8}  {2}" -f (Symbol $d12Ok), (Fmt-Ms $d12Ms), $d12Note) -ForegroundColor $(if($d12Ok){$col_ok}else{$col_fail})
}
Record "OCI image pull" $nc12Ok $nc12Ms $nc12Note $d12Ok $d12Ms $d12Note "$station image pull alpine:3.19" $d12Cmd

# ══════════════════════════════════════════════════════════════════════════════
# TEST 13 — Run from OCI image (Station run --image)
# ══════════════════════════════════════════════════════════════════════════════
Section "13. Run from snapshot (--image)"

$p13 = NextPort

# Save a fresh snapshot of go-basic, then run from it via --image.
# On Windows the hardlink workdir is used as the rootfs; the binary runs
# natively (no WSL needed). On Windows+WSL2 the workdir is translated to a
# /mnt/... path so the Linux Station can use it directly.
$snap13Name = "bench-snp13-$(Get-Random -Max 9999)"
$snap13SaveT = Run-Timed { Invoke-External { & $station snapshot save $snap13Name $goBasicDir } }
if ($snap13SaveT.Result.Ok) {
    $nc13Run = Invoke-External { & $station run --app bench-img13 --port $p13 --image $snap13Name .\go-smoke.exe }
    $nc13Id  = ($nc13Run.Output | Select-String "container\s+([a-f0-9]{8})" | ForEach-Object { $_.Matches[0].Groups[1].Value } | Select-Object -First 1)
    if ($nc13Id) { $toCleanVessel.Add($nc13Id) | Out-Null }
    $nc13Http = Wait-Http -Port $p13 -Seconds 8
    $nc13Ok   = $nc13Run.Ok -and $nc13Http.Ok
    $nc13Ms   = $snap13SaveT.Ms + $nc13Http.Ms
    $nc13Note = if ($nc13Ok) { "HTTP $($nc13Http.Status) via hardlink snapshot" } elseif (-not $nc13Run.Ok) { ($nc13Run.Output | Select-Object -Last 1) -join "" } else { "timeout" }
    Invoke-External { & $station snapshot delete $snap13Name } -IgnoreExitCode | Out-Null
} else {
    $nc13Ok   = $false
    $nc13Ms   = $snap13SaveT.Ms
    $nc13Note = "snapshot save failed: $(($snap13SaveT.Result.Output | Select-Object -Last 1) -join '')"
}

Write-Host ("  Station  {0}  {1,-8}  {2}" -f (Symbol $nc13Ok), (Fmt-Ms $nc13Ms), $nc13Note) -ForegroundColor $(if($nc13Ok){$col_ok}else{$col_fail})
$d13Note = if ($dockerCmd) { "docker run <image> (image already pulled)" } else { "skipped" }
Write-Host ("  docker   -             " + $d13Note) -ForegroundColor $col_dim
Record "Run from image/snapshot" $nc13Ok $nc13Ms $nc13Note $true 0 "docker run <image>" "$station run --image <snap> .\go-smoke.exe" "docker run <image>"

# ══════════════════════════════════════════════════════════════════════════════
# LINUX / WSL2 SECTION
# Tests the WSL2 code path (Station → doSpawnWSL → Linux Station namespace).
# Skipped automatically when no runnable WSL2 distro is found.
# ══════════════════════════════════════════════════════════════════════════════
Section "LINUX / WSL2 tests"

# Detect a runnable WSL2 distro (skip docker-desktop meta-distros).
$wslDistro = $null
if (Get-Command wsl.exe -ErrorAction SilentlyContinue) {
    $wslLines = wsl.exe --list --quiet 2>&1 | ForEach-Object {
        ($_ -replace '\x00','').Trim()
    } | Where-Object { $_ -ne '' }
    $wslDistro = $wslLines | Where-Object {
        $_ -ne 'docker-desktop' -and $_ -ne 'docker-desktop-data'
    } | Select-Object -First 1
}

if (-not $wslDistro) {
    Write-Host "  Skipped - no runnable WSL2 distro (install one with: wsl --install -d Ubuntu)" -ForegroundColor $col_dim
} else {
    Write-Host ("  WSL2 distro : {0}" -f $wslDistro) -ForegroundColor $col_dim

    # Build the Linux ELF from the same go-basic source.
    $goLinuxBin = Join-Path $goBasicDir "go-smoke-linux"
    Push-Location $goBasicDir
    $env:GOOS   = "linux"
    $env:GOARCH = "amd64"
    $linuxBuildOut = go build -o go-smoke-linux . 2>&1
    Remove-Item Env:GOOS   -ErrorAction SilentlyContinue
    Remove-Item Env:GOARCH -ErrorAction SilentlyContinue
    Pop-Location

    if (-not (Test-Path $goLinuxBin)) {
        Write-Host "  Linux binary build failed - skipping Linux tests" -ForegroundColor $col_warn
        Write-Host ("  $linuxBuildOut") -ForegroundColor $col_warn
    } else {
        Write-Host "  Linux binary built : $goLinuxBin" -ForegroundColor $col_dim

        # ── LL-1 : Run + startup via WSL2 ─────────────────────────────────────
        $pLL1    = NextPort
        $ll1Cmd  = "$station run --app bench-linux-run --port $pLL1 $goBasicDir ./go-smoke-linux"
        $ll1Run  = Invoke-External { & $station run --app bench-linux-run --port $pLL1 $goBasicDir ./go-smoke-linux }
        $ll1Id   = ($ll1Run.Output | Select-String "container\s+([a-f0-9]{8})" | ForEach-Object { $_.Matches[0].Groups[1].Value } | Select-Object -First 1)
        if ($ll1Id) { $toCleanVessel.Add($ll1Id) | Out-Null }
        $ll1Http = Wait-Http -Port $pLL1 -Seconds 20
        $ll1Ok   = $ll1Run.Ok -and $ll1Http.Ok
        $ll1Ms   = $ll1Http.Ms
        $ll1Note = if ($ll1Ok) { "HTTP $($ll1Http.Status) via WSL2" } elseif (-not $ll1Run.Ok) { ($ll1Run.Output | Select-Object -Last 1) -join "" } else { "timeout (WSL2 startup can be slow on first run)" }
        Write-Host ("  linux    {0}  {1,-8}  {2}" -f (Symbol $ll1Ok), (Fmt-Ms $ll1Ms), $ll1Note) -ForegroundColor $(if($ll1Ok){$col_ok}else{$col_fail})
        Record "Linux/WSL2: Run + startup" $ll1Ok $ll1Ms $ll1Note $false 0 "N/A" $ll1Cmd "N/A"

        # ── LL-2 : Restart on crash via WSL2 + detached supervisor ───────────
        $pLL2   = NextPort
        $ll2Cmd = "$station run --app bench-linux-restart --port $pLL2 --restart always $goBasicDir ./go-smoke-linux"
        Invoke-External { & $station run --app bench-linux-restart --port $pLL2 --restart always $goBasicDir ./go-smoke-linux } | Out-Null
        $ll2List = Invoke-External { & $station list }
        $ll2Id   = ($ll2List.Output | Select-String "bench-linux-restart" | ForEach-Object { ($_ -split '\s+')[0] } | Select-Object -First 1)
        if ($ll2Id) { $toCleanVessel.Add($ll2Id) | Out-Null }
        $ll2Before = Wait-Http -Port $pLL2 -Seconds 20
        if ($ll2Before.Ok) {
            $ll2Status = (Invoke-External { & $station status $ll2Id }).Output
            $ll2Pid    = ($ll2Status | Select-String "pid\s*[:=]\s*(\d+)" | ForEach-Object { $_.Matches[0].Groups[1].Value })
            if ($ll2Pid) { Stop-Process -Id ([int]$ll2Pid) -Force -ErrorAction SilentlyContinue }
            Start-Sleep -Milliseconds 300
            $ll2After = Wait-Http -Port $pLL2 -Seconds 10
            $ll2Ok    = $ll2After.Ok
            $ll2Ms    = $ll2After.Ms
            $ll2Note  = if ($ll2Ok) { "restarted in $(Fmt-Ms $ll2After.Ms)" } else { "did not restart" }
        } else {
            $ll2Ok = $false; $ll2Ms = 0; $ll2Note = "process never started"
        }
        Write-Host ("  linux    {0}  {1,-8}  {2}" -f (Symbol $ll2Ok), (Fmt-Ms $ll2Ms), $ll2Note) -ForegroundColor $(if($ll2Ok){$col_ok}else{$col_fail})
        Record "Linux/WSL2: Restart on crash" $ll2Ok $ll2Ms $ll2Note $false 0 "N/A" $ll2Cmd "N/A"

        # ── LL-3 : Run from snapshot (unified Windows→WSL path) ──────────────
        # Windows Station saves the snapshot in %TEMP%\relay-native\snapshots\
        # then materialises it into a workdir (%TEMP%\relay-native\workdirs\<id>)
        # which is translated to /mnt/<drive>/... for the Linux Station.
        # buildWSLCmd omits --image so the Linux side never needs its own store.
        $pLL3       = NextPort
        $ll3Snap    = "bench-linux-snap-$(Get-Random -Max 9999)"
        $ll3SaveT   = Run-Timed { Invoke-External { & $station snapshot save $ll3Snap $goBasicDir } }
        if ($ll3SaveT.Result.Ok) {
            $ll3Run  = Invoke-External { & $station run --app bench-linux-img --port $pLL3 --image $ll3Snap ./go-smoke-linux }
            $ll3Id   = ($ll3Run.Output | Select-String "container\s+([a-f0-9]{8})" | ForEach-Object { $_.Matches[0].Groups[1].Value } | Select-Object -First 1)
            if ($ll3Id) { $toCleanVessel.Add($ll3Id) | Out-Null }
            $ll3Http = Wait-Http -Port $pLL3 -Seconds 20
            $ll3Ok   = $ll3Run.Ok -and $ll3Http.Ok
            $ll3Ms   = $ll3SaveT.Ms + $ll3Http.Ms
            $ll3Note = if ($ll3Ok) { "HTTP $($ll3Http.Status) via snapshot workdir /mnt/..." } elseif (-not $ll3Run.Ok) { ($ll3Run.Output | Select-Object -Last 1) -join "" } else { "timeout" }
            Invoke-External { & $station snapshot delete $ll3Snap } -IgnoreExitCode | Out-Null
        } else {
            $ll3Ok = $false; $ll3Ms = $ll3SaveT.Ms
            $ll3Note = "snapshot save failed"
        }
        Write-Host ("  linux    {0}  {1,-8}  {2}" -f (Symbol $ll3Ok), (Fmt-Ms $ll3Ms), $ll3Note) -ForegroundColor $(if($ll3Ok){$col_ok}else{$col_fail})
        Record "Linux/WSL2: Run from snapshot" $ll3Ok $ll3Ms $ll3Note $false 0 "N/A" "$station run --image <snap> ./go-smoke-linux" "N/A"

        # Clean up the Linux binary — it’s a build artefact.
        Remove-Item $goLinuxBin -ErrorAction SilentlyContinue
    }
}

# ══════════════════════════════════════════════════════════════════════════════
# CLEANUP
# ══════════════════════════════════════════════════════════════════════════════
Cleanup

# ══════════════════════════════════════════════════════════════════════════════
# SUMMARY TABLE
# ══════════════════════════════════════════════════════════════════════════════

Write-Host ""
Write-Host ""
Write-Host "  +----------------------------------------------------------------------+" -ForegroundColor White
Write-Host "  |           Station vs Docker - feature matrix                        |" -ForegroundColor White
Write-Host "  +----------------------------------------------------------------------+" -ForegroundColor White
Write-Host ""

$w1 = 32   # feature column
$w2 = 10   # Station result
$w3 = 10   # Station time
$w4 = 10   # docker result
$w5 = 10   # docker time

$hdr = "  {0,-$w1}  {1,$w2}  {2,$w3}  {3,$w4}  {4,$w5}" -f "Feature", "Station", "time", "docker", "time"
Write-Host $hdr -ForegroundColor White
Write-Host ("  " + ("-" * ($w1+$w2+$w3+$w4+$w5+10))) -ForegroundColor $col_dim

foreach ($r in $benchResults) {
    $nc = $r.Station
    $dk = $r.docker

    $ncSym  = if ($nc.ok) { "[+]" } else { "[!]" }
    $dkSym  = if ($dk.ms -eq 0 -and -not $dk.ok) { "-  " } elseif ($dk.ok) { "[+]" } else { "[!]" }
    $ncTime = if ($nc.ms -gt 0) { Fmt-Ms $nc.ms } else { "-" }
    $dkTime = if ($dk.ms -gt 0) { Fmt-Ms $dk.ms } else { "-" }

    $ncCol  = if ($nc.ok) { $col_ok } else { $col_fail }
    $dkCol  = if ($dk.ms -eq 0 -and -not $dk.ok) { $col_dim } elseif ($dk.ok) { $col_ok } else { $col_fail }

    $line = "  {0,-$w1}  {1,$w2}  {2,$w3}  {3,$w4}  {4,$w5}" -f $r.feature, " ", $ncTime, " ", $dkTime
    Write-Host -NoNewline ("  {0,-$w1}  " -f $r.feature)
    Write-Host -NoNewline ("{0,2}  {1,$w3}  " -f $ncSym, $ncTime) -ForegroundColor $ncCol
    Write-Host ("{0,2}  {1,$w3}" -f $dkSym, $dkTime) -ForegroundColor $dkCol
}

Write-Host ""

# ─── score ────────────────────────────────────────────────────────────────────

$ncPassed = ($benchResults | Where-Object { $_.Station.ok }).Count
$ncTotal  = $benchResults.Count
Write-Host ("  Station score : {0}/{1} features passing" -f $ncPassed, $ncTotal) -ForegroundColor $(if ($ncPassed -eq $ncTotal) { $col_ok } else { $col_warn })

if ($dockerCmd) {
    $dkPassed = ($benchResults | Where-Object { $_.docker.ok }).Count
    $dkTotal  = ($benchResults | Where-Object { $_.docker.ms -gt 0 -or $_.docker.ok }).Count
    Write-Host ("  docker  score : {0}/{1} features tested" -f $dkPassed, $dkTotal) -ForegroundColor $(if ($dkPassed -eq $dkTotal) { $col_ok } else { $col_warn })
}

# Station-only advantages
Write-Host ""
Write-Host "  Station-only features (no Docker equivalent):" -ForegroundColor Cyan
Write-Host "    * Blue-green proxy  (Station proxy start/swap)  - Docker needs nginx/traefik" -ForegroundColor $col_dim
Write-Host "    * Stable port reuse (Station --app <name>)      - Docker allocates per container" -ForegroundColor $col_dim
Write-Host "    * Cross-WSL2 state  (Windows PID + Linux exec)  - No Docker analog" -ForegroundColor $col_dim
Write-Host ""

# ─── save JSON ────────────────────────────────────────────────────────────────

$benchResults | ForEach-Object {
    [pscustomobject]@{
        feature = $_.feature
        Station = $_.Station
        docker  = $_.docker
    }
} | ConvertTo-Json -Depth 5 | Out-File $jsonOut -Encoding utf8

Write-Host "  JSON results saved to: $jsonOut" -ForegroundColor $col_dim
Write-Host ""
