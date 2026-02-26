Param(
    [string]$ApiUrl = 'http://localhost:8080',
    [string]$IngesterToken = 'ingester-token-example',
    [string]$AuditorToken = 'auditor-token-example',
    # allow string input so CI can pass literal text without PowerShell binding errors
    [string]$EnableTestJwt = 'true'
)

# convert string to boolean; accept typical truthy values
$enableBool = $false
if ($EnableTestJwt -ne $null) {
    try {
        $enableBool = [bool]::Parse($EnableTestJwt)
    } catch {
        # fallback: treat any non-empty string as true
        if ($EnableTestJwt -ne '') { $enableBool = $true }
    }
}

Write-Host "Setting E2E environment variables..."
$env:E2E_API_URL = $ApiUrl
if (-not $env:E2E_INGESTER_TOKEN) { $env:E2E_INGESTER_TOKEN = $IngesterToken }
if (-not $env:E2E_AUDITOR_TOKEN)  { $env:E2E_AUDITOR_TOKEN  = $AuditorToken }

Write-Host "E2E_API_URL=$env:E2E_API_URL"
Write-Host "E2E_INGESTER_TOKEN=$($env:E2E_INGESTER_TOKEN)"
Write-Host "E2E_AUDITOR_TOKEN=$($env:E2E_AUDITOR_TOKEN)"
if ($EnableTestJwt) {
    Write-Host "Enabling test JWT mode for local runs"
    $env:ENABLE_TEST_JWT = 'true'
} else {
    Write-Host "Test JWT mode disabled for this run"
    if ($env:ENABLE_TEST_JWT) { Remove-Item Env:\ENABLE_TEST_JWT }
}

$composeFile = "ops/docker/docker-compose.yml"
if (-not (Test-Path $composeFile)) {
    Write-Error "docker-compose file not found at $composeFile"
    exit 2
}

Write-Host "Bringing up docker-compose (in background)..."
docker compose -f $composeFile up --build -d

Write-Host "Waiting 15s for services to start..."
Start-Sleep -Seconds 15

# If JWKS is required (test-mode disabled and JWKS_URL is set), wait for it to become available
if (-not $EnableTestJwt -and $env:JWKS_URL) {
    Write-Host "Waiting for JWKS at $env:JWKS_URL"
    $attempts = 15
    for ($i = 0; $i -lt $attempts; $i++) {
        try {
            $resp = Invoke-WebRequest -Uri $env:JWKS_URL -UseBasicParsing -TimeoutSec 2 -ErrorAction Stop
            if ($resp.StatusCode -eq 200) {
                Write-Host "JWKS is available"
                break
            }
        } catch {
            Start-Sleep -Seconds 1
        }
    }
}

Write-Host "Running integration tests..."
go test ./tests/integration -count=1 -v
$intExit = $LASTEXITCODE

Write-Host "Running e2e tests..."
go test ./tests/e2e -count=1 -v
$e2eExit = $LASTEXITCODE

Write-Host "Tearing down docker-compose..."
docker compose -f $composeFile down
Write-Host "Tearing down docker-compose..."
docker compose -f $composeFile down

# Cleanup any CI JWKS artifacts that may have been generated
if (Test-Path "scripts/ci_jwks_env.txt") {
    Remove-Item "scripts/ci_jwks_env.txt" -ErrorAction SilentlyContinue
    Write-Host "Removed scripts/ci_jwks_env.txt"
}
if (Test-Path "scripts/jwks.json") {
    Remove-Item "scripts/jwks.json" -ErrorAction SilentlyContinue
    Write-Host "Removed scripts/jwks.json"
}
if (Test-Path "scripts/jwks_key.pem") {
    Remove-Item "scripts/jwks_key.pem" -ErrorAction SilentlyContinue
    Write-Host "Removed scripts/jwks_key.pem"
}

if ($intExit -ne 0 -or $e2eExit -ne 0) {
    Write-Host "One or more test suites failed (integration:$intExit, e2e:$e2eExit)" -ForegroundColor Red
    exit 1
}

Write-Host "Integration and e2e tests passed." -ForegroundColor Green
exit 0
