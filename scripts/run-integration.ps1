Param(
    [string]$ApiUrl = 'http://localhost:8080',
    [string]$IngesterToken = 'ingester-token-example',
    [string]$AuditorToken = 'auditor-token-example'
)

Write-Host "Setting E2E environment variables..."
$env:E2E_API_URL = $ApiUrl
if (-not $env:E2E_INGESTER_TOKEN) { $env:E2E_INGESTER_TOKEN = $IngesterToken }
if (-not $env:E2E_AUDITOR_TOKEN)  { $env:E2E_AUDITOR_TOKEN  = $AuditorToken }

Write-Host "E2E_API_URL=$env:E2E_API_URL"
Write-Host "E2E_INGESTER_TOKEN=$($env:E2E_INGESTER_TOKEN)"
Write-Host "E2E_AUDITOR_TOKEN=$($env:E2E_AUDITOR_TOKEN)"

$composeFile = "ops/docker/docker-compose.yml"
if (-not (Test-Path $composeFile)) {
    Write-Error "docker-compose file not found at $composeFile"
    exit 2
}

Write-Host "Bringing up docker-compose (in background)..."
docker-compose -f $composeFile up --build -d

Write-Host "Waiting 15s for services to start..."
Start-Sleep -Seconds 15

Write-Host "Running integration tests..."
go test ./tests/integration -count=1 -v
$intExit = $LASTEXITCODE

Write-Host "Running e2e tests..."
go test ./tests/e2e -count=1 -v
$e2eExit = $LASTEXITCODE

Write-Host "Tearing down docker-compose..."
docker-compose -f $composeFile down

if ($intExit -ne 0 -or $e2eExit -ne 0) {
    Write-Host "One or more test suites failed (integration:$intExit, e2e:$e2eExit)" -ForegroundColor Red
    exit 1
}

Write-Host "Integration and e2e tests passed." -ForegroundColor Green
exit 0
