# PowerShell convenience script to shut down anything started by the repo
# Usage: pwsh ./scripts/cleanup.ps1

Write-Host "Stopping Docker Compose stack..."
$compose = "ops/docker/docker-compose.yml"
if (Test-Path $compose) {
    docker-compose -f $compose down
} else {
    Write-Host "docker-compose file not found, skipping."
}

Write-Host "Checking for vault-api / ci_jwks servers started with 'go run'..."
Get-Process -Name go -ErrorAction SilentlyContinue | Where-Object {
    $_.Path -and $_.Path -match 'go.exe'
} | ForEach-Object {
    # attempt to identify by command line (requires PowerShell 7+ for CommandLine property)
    try {
        $cmd = $_.CommandLine
    } catch {
        $cmd = $_.Path
    }
    if ($cmd -match 'ci_jwks.go' -or $cmd -match 'services\\vault-api') {
        Write-Host "Killing process $($_.Id) ($cmd)" -ForegroundColor Yellow
        Stop-Process -Id $_.Id -Force
    }
}

Write-Host "Closing any leftover 'vault-api' or 'ci_jwks' listeners on common ports..."
# find PIDs listening on 8080 or 8000/59133 etc
netstat -ano | Select-String ":8080" | ForEach-Object {
    $pid = ($_ -split '\s+')[-1]
    if ($pid -and $pid -ne 0) {
        Write-Host "Killing PID $pid listening on 8080" -ForegroundColor Yellow
        taskkill /F /PID $pid | Out-Null
    }
}

Write-Host "Cleanup complete." -ForegroundColor Green
