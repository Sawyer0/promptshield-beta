param(
  [string]$Base = 'http://localhost:8080',
  [string]$Path = '/check',
  [double]$FlagRate = 0.02,
  [string]$Duration = '10m',
  [int[]]$Multipliers = @(8,12,16),
  [string]$TenantId = '00000000-0000-0000-0000-000000000001',
  [switch]$Smoke
)

$ErrorActionPreference = 'Stop'

# Resolve paths
$ScriptRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
$RepoRoot = Split-Path -Parent $ScriptRoot
$k6Script = Join-Path $ScriptRoot 'perf_mix.js'
$outDir = Join-Path $ScriptRoot 'out'
New-Item -ItemType Directory -Path $outDir -Force | Out-Null

# Check k6
if (-not (Get-Command k6 -ErrorAction SilentlyContinue)) {
  Write-Host 'k6 is not installed. Install via winget or Chocolatey:' -ForegroundColor Yellow
  Write-Host '  winget install grafana.k6' -ForegroundColor Yellow
  Write-Host '  choco install k6 -y' -ForegroundColor Yellow
  exit 1
}

# Compute cores and VUs
$cores = [Environment]::ProcessorCount

# Prepare CSV output
$timestamp = Get-Date -Format 'yyyyMMdd_HHmmss'
$csvPath = Join-Path $outDir ("perf_summary_$timestamp.csv")
"run_label,cores,vus,duration_s,http_reqs,count_flagged,p95_ms,p99_ms,p99_9_ms,lat_1_4KB_p95_ms,lat_8_32KB_p95_ms,lat_64_256KB_p95_ms,flag_true_p95_ms,flag_false_p95_ms" | Out-File -FilePath $csvPath -Encoding utf8

function Run-One([int]$vus, [string]$label) {
  $sumPath = Join-Path $outDir ("summary_${label}_vus${vus}_$timestamp.json")
  $envArgs = @(
    '--env', "BASE=$Base",
    '--env', "PATH=$Path",
    '--env', "FLAG_RATE=$FlagRate",
    '--env', "TENANT_ID=$TenantId"
  )
  $args = @('run', $k6Script, '--vus', $vus, '--duration', $Duration, '--summary-export', $sumPath) + $envArgs
  Write-Host "Running: k6 $($args -join ' ')" -ForegroundColor Cyan
  k6 @args | Out-Host
  return $sumPath
}

function Parse-And-Append([string]$sumPath, [int]$vus, [string]$label) {
  $j = Get-Content $sumPath -Raw | ConvertFrom-Json

  # Metrics extraction with guards
  $m = $j.metrics
  $http = $m.http_req_duration
  $http_p95 = $http.values.'p(95)'
  $http_p99 = $http.values.'p(99)'
  $http_p999 = $http.values.'p(99.9)'
  $reqs = $m.http_reqs
  $reqCount = if ($reqs) { [int]$reqs.count } else { 0 }
  $flagged = $m.flagged_requests
  $flagCount = if ($flagged) { [int]$flagged.count } else { 0 }

  $b1 = $m.lat_1_4KB_ms
  $b2 = $m.lat_8_32KB_ms
  $b3 = $m.lat_64_256KB_ms
  $fT = $m.lat_flag_true_ms
  $fF = $m.lat_flag_false_ms

  $b1p95 = if ($b1) { $b1.values.'p(95)' } else { '' }
  $b2p95 = if ($b2) { $b2.values.'p(95)' } else { '' }
  $b3p95 = if ($b3) { $b3.values.'p(95)' } else { '' }
  $fTp95 = if ($fT) { $fT.values.'p(95)' } else { '' }
  $fFp95 = if ($fF) { $fF.values.'p(95)' } else { '' }

  # Duration seconds (approx) from test state if available, else parse from $Duration
  $durSec = if ($j.state -and $j.state.testRunDurationMs) { [math]::Round($j.state.testRunDurationMs/1000.0,2) } else {
    if ($Duration -match '^(\d+)([smh])$') {
      $n = [double]$Matches[1]
      switch ($Matches[2]) {
        's' { $n }
        'm' { $n*60 }
        'h' { $n*3600 }
      }
    } else { '' }
  }

  $line = "$label,$cores,$vus,$durSec,$reqCount,$flagCount,$http_p95,$http_p99,$http_p999,$b1p95,$b2p95,$b3p95,$fTp95,$fFp95"
  Add-Content -Path $csvPath -Value $line
}

if ($Smoke) {
  $vus = [Math]::Max(1, $cores * 4)
  $sum = Run-One -vus $vus -label 'smoke'
  Parse-And-Append -sumPath $sum -vus $vus -label 'smoke'
  Write-Host "Smoke complete. CSV: $csvPath" -ForegroundColor Green
  exit 0
}

foreach ($m in $Multipliers) {
  $vus = $cores * $m
  $label = "m$m"
  $sum = Run-One -vus $vus -label $label
  Parse-And-Append -sumPath $sum -vus $vus -label $label
}

Write-Host "Done. Summaries in $outDir and CSV at $csvPath" -ForegroundColor Green

