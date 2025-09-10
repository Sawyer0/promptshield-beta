param()

function Stop-Port($port) {
  $lines = netstat -ano | Select-String ":$port "
  foreach ($line in $lines) {
    if ($line.ToString().Contains('LISTENING')) {
      $procId = ($line.ToString() -split '\s+')[-1]
      try { Stop-Process -Id $procId -Force -ErrorAction Stop; Write-Output "Killed PID $procId on port $port" } catch {}
    }
  }
}

Stop-Port 9091
Get-Process -Name extproc_local -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue

$root = Split-Path -Parent $MyInvocation.MyCommand.Path
$repo = (Resolve-Path "$root\..\").Path
$exe = Join-Path $repo 'bin\\extproc_local.exe'
$src = Join-Path $repo 'extproc_local.go'

# Ensure bin dir
$bindir = Split-Path -Parent $exe; if (!(Test-Path $bindir)) { New-Item -ItemType Directory -Path $bindir | Out-Null }

& go build -o $exe $src
if ($LASTEXITCODE -ne 0) { throw 'go build failed' }

# Start with pprof on 6061 (default) and any desired analyzer flags already in environment
$env:PPROF_ADDR = 'localhost:6061'
# Perf tuning env defaults (caller may override)
if (-not $env:PS_ENFORCER_STREAM_WINDOW) { $env:PS_ENFORCER_STREAM_WINDOW = '20480' }
if (-not $env:PS_ENFORCER_STREAM_OVERLAP) { $env:PS_ENFORCER_STREAM_OVERLAP = '4096' }
if (-not $env:PS_MAX_REGEX_LEN) { $env:PS_MAX_REGEX_LEN = '512' }
if (-not $env:PS_SEMANTIC_REQUIRE_CACHE_HIT) { $env:PS_SEMANTIC_REQUIRE_CACHE_HIT = 'true' }
Start-Process -FilePath $exe -WindowStyle Hidden | Out-Null

# Wait for gRPC 9091 and pprof 6061 readiness
$deadline = (Get-Date).AddSeconds(20)
$grpcOk = $false
$pprofOk = $false
while((Get-Date) -lt $deadline -and (-not $grpcOk -or -not $pprofOk)){
  try {
    # pprof
    if (-not $pprofOk) {
      $r = Invoke-WebRequest -UseBasicParsing -Uri 'http://127.0.0.1:6061/debug/pprof/'
      if ($r.StatusCode -ge 200 -and $r.StatusCode -lt 500) { $pprofOk=$true }
    }
  } catch {}
  try {
    # crude port check for 9091
    $lines = netstat -ano | Select-String ':9091 '
    if ($lines -and ($lines | ForEach-Object { $_.ToString() }) -match 'LISTENING') { $grpcOk=$true }
  } catch {}
  Start-Sleep -Milliseconds 300
}
if (-not $grpcOk) { throw 'gRPC 9091 not listening' }
if (-not $pprofOk) { throw 'pprof 6061 not ready' }
'extproc pprof ready'

