$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
$relayUrl = if ($env:RELAY_TEST_URL) { $env:RELAY_TEST_URL } else { "http://127.0.0.1:8080" }
$token = if ($env:RELAY_TEST_TOKEN) { $env:RELAY_TEST_TOKEN } else { "88dd36190c2e124d00b639d317dc889d487e5851a3f94b0366021fd626ed8f01
" }
$envName = "preview"
$branch = "main"
$resultsDir = Join-Path $PSScriptRoot "results"

New-Item -ItemType Directory -Force -Path $resultsDir | Out-Null

$cases = @(
  @{ Name = "static"; Dir = "static-basic"; App = "smoke-static"; Port = 3111; Attempts = 2 },
  @{ Name = "node"; Dir = "node-basic"; App = "smoke-node"; Port = 3112; Attempts = 2 },
  @{ Name = "vite"; Dir = "vite-basic"; App = "smoke-vite"; Port = 3113; Attempts = 2 },
  @{ Name = "go"; Dir = "go-basic"; App = "smoke-go"; Port = 3114; Attempts = 2 },
  @{ Name = "python"; Dir = "python-basic"; App = "smoke-python"; Port = 3115; Attempts = 2 },
  @{ Name = "dotnet"; Dir = "dotnet-basic"; App = "smoke-dotnet"; Port = 3116; Attempts = 2 },
  @{ Name = "java"; Dir = "java-basic"; App = "smoke-java"; Port = 3117; Attempts = 2 },
  @{ Name = "rust"; Dir = "rust-basic"; App = "smoke-rust"; Port = 3118; Attempts = 2 },
  @{ Name = "cpp"; Dir = "cpp-basic"; App = "smoke-cpp"; Port = 3119; Attempts = 1 }

)

function Invoke-DeployAttempt {
  param(
    [hashtable]$Case,
    [int]$Attempt
  )

  $workdir = Join-Path $PSScriptRoot $Case.Dir
  $logFile = Join-Path $resultsDir ("{0}-attempt{1}.log" -f $Case.Name, $Attempt)
  $stopwatch = [System.Diagnostics.Stopwatch]::StartNew()

  Push-Location $workdir
  try {
    $lines = & relay deploy `
      --url $relayUrl `
      --token $token `
      --app $Case.App `
      --env $envName `
      --branch $branch `
      --dir . `
      --host-port $Case.Port `
      --stream 2>&1
    $exitCode = $LASTEXITCODE
  } finally {
    Pop-Location
    $stopwatch.Stop()
  }

  $lines | Out-File -FilePath $logFile -Encoding utf8
  $text = [string]::Join("`n", $lines)

  $buildpack = $null
  $deployId = $null
  $previewUrl = "http://127.0.0.1:{0}" -f $Case.Port
  $httpStatus = $null
  $uploadedLine = $null

  if ($text -match "selected buildpack:\s*([^\r\n]+)") {
    $buildpack = $matches[1].Trim()
  }
  if ($text -match "deploy queued:\s*id=([a-f0-9]+)") {
    $deployId = $matches[1]
  }
  $uploadedLine = ($lines | Where-Object { $_ -match "need upload:" } | Select-Object -Last 1)

  if ($exitCode -eq 0) {
    Start-Sleep -Seconds 2
    try {
      $httpStatus = & curl.exe -s -o NUL -w "%{http_code}" $previewUrl
    } catch {
      $httpStatus = "curl-error"
    }
  }

  [pscustomobject]@{
    framework = $Case.Name
    attempt = $Attempt
    app = $Case.App
    dir = $Case.Dir
    port = $Case.Port
    exit_code = $exitCode
    success = ($exitCode -eq 0)
    duration_seconds = [math]::Round($stopwatch.Elapsed.TotalSeconds, 2)
    buildpack = $buildpack
    deploy_id = $deployId
    preview_url = $previewUrl
    http_status = $httpStatus
    upload_summary = $uploadedLine
    log_file = $logFile
  }
}

$health = & curl.exe -s -o NUL -w "%{http_code}" "$relayUrl/health"
if ($health -ne "200") {
  throw "Relay agent health check failed with status '$health'"
}

$results = New-Object System.Collections.Generic.List[object]

foreach ($case in $cases) {
  for ($attempt = 1; $attempt -le $case.Attempts; $attempt++) {
    $result = Invoke-DeployAttempt -Case $case -Attempt $attempt
    $results.Add($result)
  }
}

$jsonPath = Join-Path $resultsDir "summary.json"
$results | ConvertTo-Json -Depth 6 | Out-File -FilePath $jsonPath -Encoding utf8
$results | Format-Table framework,attempt,success,exit_code,duration_seconds,buildpack,http_status,upload_summary -AutoSize
Write-Host ""
Write-Host ("Saved summary to {0}" -f $jsonPath)
