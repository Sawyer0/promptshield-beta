param(
  [string]$OutDir = "C:\\Users\\Dawan\\promptshield-v0.2.0\\tools\\perf\\out"
)

$ErrorActionPreference = 'Stop'
$log = Get-ChildItem $OutDir -Filter 'envoy_xaz_log_*.log' | Sort-Object LastWriteTime | Select-Object -Last 1
if ($null -eq $log) { Write-Host "No envoy_xaz_log_*.log files found in $OutDir"; exit 0 }

$p = $log.FullName
$lines = Get-Content -Path $p
$decisions = @{}
$reasons = @{}
$meta_present = 0

foreach ($line in $lines) {
  try {
    $obj = $line | ConvertFrom-Json
  } catch { continue }
  if ($obj.decision) {
    $d = $obj.decision.ToString().ToLower()
    if (-not $decisions.ContainsKey($d)) { $decisions[$d] = 0 }
    $decisions[$d]++
  }
  if ($obj.reason) {
    $r = $obj.reason.ToString().ToLower()
    if (-not $reasons.ContainsKey($r)) { $reasons[$r] = 0 }
    $reasons[$r]++
  }
  if ($obj.ext_proc_meta -and $obj.ext_proc_meta.ToString().Trim().Length -gt 0) {
    $meta_present++
  }
}

Write-Host ("Analyzed log: {0}" -f $p)
Write-Host ("Total lines: {0}" -f $lines.Count)
Write-Host ("ext_proc dynamic metadata non-empty lines: {0}" -f $meta_present)
Write-Host "Decisions:"
$decisions.GetEnumerator() | Sort-Object -Property Name | ForEach-Object { Write-Host ("  {0}: {1}" -f $_.Name, $_.Value) }
Write-Host "Reasons:"
$reasons.GetEnumerator() | Sort-Object -Property Name | ForEach-Object { Write-Host ("  {0}: {1}" -f $_.Name, $_.Value) }

