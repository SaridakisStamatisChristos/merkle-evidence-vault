#!/usr/bin/env pwsh
$ErrorActionPreference = 'Stop'

$root = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
Set-Location $root

$timestamp = [DateTime]::UtcNow.ToString('yyyyMMddTHHmmssZ')
$outDir = Join-Path $root ("artifacts/drills/" + $timestamp)
$logDir = Join-Path $outDir "logs"
New-Item -ItemType Directory -Force -Path $logDir | Out-Null

$env:DRILL_OUT_DIR = $outDir

if (Get-Command bash -ErrorAction SilentlyContinue) {
  bash "scripts/drill_restore.sh"
  exit $LASTEXITCODE
}

$summary = @{
  pass = $false
  git_sha = (git rev-parse --short HEAD)
  note = 'bash not available; run scripts/drill_restore.sh on Linux/macOS or install Git Bash on Windows'
  started_epoch = [int][double]::Parse((Get-Date -UFormat %s))
  ended_epoch = [int][double]::Parse((Get-Date -UFormat %s))
  duration_seconds = 0
  verifier_exit_code = 127
  artifact_dir = $outDir
}
$summary | ConvertTo-Json -Depth 6 | Set-Content -Path (Join-Path $outDir 'drill_summary.json')
Write-Host "drill stub summary written to $outDir"
exit 0
