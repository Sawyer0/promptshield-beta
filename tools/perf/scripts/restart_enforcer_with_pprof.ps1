param()

function Stop-Port($port) {
  $lines = netstat -ano | Select-String ":$port "
  foreach ($line in $lines) {
    if ($line.ToString().Contains('LISTENING')) {
      $pid = ($line.ToString() -split '\s+')[-1]
      try { Stop-Process -Id $pid -Force -ErrorAction Stop; Write-Output "Killed PID $pid on port $port" } catch {}
    }
  }
}

Stop-Port 8080
Stop-Port 6060

# Also kill by name (best-effort)
Get-Process -Name enforcer_local -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue

# Build enforcer_local.exe
$root = Split-Path -Parent $MyInvocation.MyCommand.Path
$repo = (Resolve-Path "$root\..\..\..").Path
$exe = Join-Path $repo 'tools\perf\bin\enforcer_local.exe'
$src = Join-Path $repo 'tools\perf\enforcer_local.go'

& go build -o $exe $src

# Start process with env vars
$env:PS_ENFORCER_FAST='1'
$env:PS_ENFORCER_DISABLE_TRACING='1'
Start-Process -FilePath $exe -WindowStyle Hidden | Out-Null

# Wait for pprof to come up on :6060
$deadline = (Get-Date).AddSeconds(20)
$ok = $false
while((Get-Date) -lt $deadline -and -not $ok){
  try{
    $resp = Invoke-WebRequest -UseBasicParsing -Uri 'http://127.0.0.1:6060/debug/pprof/'
    if ($resp.StatusCode -ge 200 -and $resp.StatusCode -lt 500){ $ok=$true }
  } catch {
    Start-Sleep -Milliseconds 500
  }
}
if (-not $ok){ throw 'pprof did not start on :6060 within 20s' }
'pprof ready'

