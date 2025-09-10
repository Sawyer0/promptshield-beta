param(
  [string]$OutDir = "C:\\Users\\Dawan\\promptshield-v0.2.0\\tools\\perf\\out"
)

$ErrorActionPreference = 'Stop'
$log = Get-ChildItem $OutDir -Filter 'envoy_xaz_log_*.log' | Sort-Object LastWriteTime | Select-Object -Last 1
if ($null -eq $log) {
  Write-Host "No envoy_xaz_log_*.log files found in $OutDir"
  exit 0
}
$p = $log.FullName
$lines = Get-Content -Path $p
$total = $lines.Count
$flags = ($lines | Select-String -Pattern ' (UT|UF|UO|DC|LH|LR|UR|UC|DI|FI|RL|UA|DN|URX|SI|DC|NC) ' -AllMatches | Measure-Object).Count
$ext = ($lines | Select-String -Pattern 'envoy.filters.http.ext_proc' -AllMatches | Measure-Object).Count

Write-Host "Log: $p"
Write-Host "Lines: $total"
Write-Host "Flagged lines (common flags): $flags"
Write-Host "Lines with ext_proc dynamic metadata: $ext"
