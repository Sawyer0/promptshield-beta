param(
  [string]$Base = 'http://localhost:18001',
  [string]$Duration = '3m',
  [int]$Multiplier = 12
)

$ErrorActionPreference = 'Stop'

$root = (Resolve-Path (Join-Path $PSScriptRoot '..\..\..')).Path
$out = Join-Path $root 'tools\perf\out'
New-Item -ItemType Directory -Path $out -Force | Out-Null

$ts = Get-Date -Format 'yyyyMMdd_HHmmss'
$cpu = Join-Path $out ("pprof_extproc_cpu_" + $ts + ".pb.gz")
$heap = Join-Path $out ("pprof_extproc_heap_" + $ts + ".pb.gz")
$log = Join-Path $out ("envoy_xaz_log_" + $ts + ".log")

# Start CPU profile capture (180s) in a separate, hidden PowerShell process
Start-Process -WindowStyle Hidden -FilePath powershell -ArgumentList @(
  '-NoProfile','-Command',
  "Invoke-WebRequest -UseBasicParsing -Uri 'http://127.0.0.1:6061/debug/pprof/profile?seconds=180' -OutFile '$cpu'"
) | Out-Null

# Run the perf sweep
& (Join-Path $root 'tools\perf\run_perf_sweep.ps1') -Base $Base -Duration $Duration -Multipliers $Multiplier

# Capture heap profile at the end of the run
Invoke-WebRequest -UseBasicParsing -Uri 'http://127.0.0.1:6061/debug/pprof/heap' -OutFile $heap

# Fetch Envoy xAZ container logs for the last 5 minutes
try {
  docker logs envoy-extproc-xaz --since 5m | Out-File -FilePath $log -Encoding utf8
} catch {}

Write-Host "CPU profile: $cpu"
Write-Host "Heap profile: $heap"
Write-Host "Envoy log: $log"

