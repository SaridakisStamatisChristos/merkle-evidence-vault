#!/usr/bin/env pwsh
$ErrorActionPreference = 'Stop'

$root = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
Set-Location $root

$timestamp = [DateTime]::UtcNow.ToString('yyyyMMddTHHmmssZ')
$outDir = Join-Path $root ("artifacts/game-day/" + $timestamp)
New-Item -ItemType Directory -Force -Path $outDir | Out-Null

if (Get-Command bash -ErrorAction SilentlyContinue) {
  bash "scripts/game_day_merkle_down.sh"
  exit $LASTEXITCODE
}

$summary = @{
  pass = $false
  scenario = 'merkle-engine-down'
  git_sha = (git rev-parse --short HEAD)
  note = 'bash not available; run scripts/game_day_merkle_down.sh on Linux/macOS or install Git Bash on Windows'
  started_epoch = [int][double]::Parse((Get-Date -UFormat %s))
  ended_epoch = [int][double]::Parse((Get-Date -UFormat %s))
  duration_seconds = 0
}
$summary | ConvertTo-Json -Depth 6 | Set-Content -Path (Join-Path $outDir 'game_day_report.json')
Write-Host "game day stub report written to $outDir"
exit 0
