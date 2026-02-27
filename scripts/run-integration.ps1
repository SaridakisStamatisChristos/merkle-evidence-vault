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

# if JWKS_URL wasn't inherited (bash step might not have passed it), load from file
if (-not $env:JWKS_URL -and (Test-Path "scripts/ci_jwks_env.txt")) {
    $content = Get-Content "scripts/ci_jwks_env.txt"
    foreach ($line in $content) {
        if ($line -match '^JWKS_URL=(.*)$') {
            $env:JWKS_URL = $matches[1]
        }
    }
}
# vault-api is running locally so it can reach the stub via localhost; no translation needed
# (previous container-based logic removed)

Write-Host "E2E_API_URL=$env:E2E_API_URL"
Write-Host "E2E_INGESTER_TOKEN=$($env:E2E_INGESTER_TOKEN)"
Write-Host "E2E_AUDITOR_TOKEN=$($env:E2E_AUDITOR_TOKEN)"
Write-Host "JWKS_URL=$env:JWKS_URL"
if ($enableBool) {
    Write-Host "Enabling test JWT mode for local runs"
    $env:ENABLE_TEST_JWT = 'true'
} else {
    Write-Host "Disabling test JWT mode for this run"
    # explicitly set to false so docker-compose substitution works
    $env:ENABLE_TEST_JWT = 'false'
}
Write-Host "enableBool=$enableBool, Effective ENABLE_TEST_JWT=$env:ENABLE_TEST_JWT"

$composeFile = "ops/docker/docker-compose.yml"
if (-not (Test-Path $composeFile)) {
    Write-Error "docker-compose file not found at $composeFile"
    exit 2
}

Write-Host "Cleaning up any previous compose services (excluding vault-api)..."
# bring down everything in case leftovers exist
docker compose -f $composeFile down -v --remove-orphans || true
Write-Host "Bringing up dependency services via docker-compose (no vault-api)..."
# explicitly list the services we need; omit vault-api so we can run it locally
docker compose -f $composeFile up --build -d postgres redis zookeeper kafka merkle-engine

Write-Host "Waiting 15s for dependency services to start..."
Start-Sleep -Seconds 15
Write-Host "Waiting for Postgres container to become healthy (via docker inspect or TCP)..."
$pgHealthy = $false
$pgAttempts = 60
$pgCont = ""
try {
    $pgCont = (docker compose -f $composeFile ps -q postgres) -join ''
} catch {}
if ($pgCont -ne "") {
    for ($k = 0; $k -lt $pgAttempts; $k++) {
        try {
            $status = docker inspect --format='{{.State.Health.Status}}' $pgCont 2>$null
            if ($status -eq "healthy") { $pgHealthy = $true; break }
        } catch {}
        Start-Sleep -Seconds 2
    }
}
if (-not $pgHealthy) {
    Write-Host "Container health check didn't report healthy; attempting direct TCP connect to localhost:5432"
    for ($k = 0; $k -lt 30; $k++) {
        try {
            $tcp = New-Object System.Net.Sockets.TcpClient
            $tcp.Connect('127.0.0.1', 5432)
            $tcp.Close()
            $pgHealthy = $true
            break
        } catch {
            Start-Sleep -Seconds 1
        }
    }
}
if (-not $pgHealthy) {
    Write-Host "ERROR: Postgres did not become available within timeout; dumping postgres logs for debugging"
    if (Test-Path "/tmp/vault-api.log") { Get-Content "/tmp/vault-api.log" }
    try { docker compose -f $composeFile logs postgres | Select-Object -First 200 } catch {}
    exit 1
}
Write-Host "Postgres is available"
# If JWKS is required (test-mode disabled and JWKS_URL is set), wait for it to become available
if (-not $enableBool -and $env:JWKS_URL) {
    Write-Host "Waiting for JWKS at $env:JWKS_URL"
    $attempts = 30
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

# start vault-api locally (not in a container) so it can see host JWKS stub easily
Write-Host "Starting vault-api server locally (HTTP_ADDR=:8080)"
# ensure environment variables for vault-api match what we expect
# ensure environment variables for vault-api match what we expect
$env:DATABASE_URL = "postgres://vault_api@localhost:5432/vault?sslmode=disable"
# Set HTTP_ADDR according to requested ApiUrl. If ApiUrl contains a port, use it, otherwise default to :8080
if ($ApiUrl -match "://[^:]+:(\d+)") {
    $port = $matches[1]
    $env:HTTP_ADDR = ":$port"
} else {
    $env:HTTP_ADDR = ":8080"
}
Write-Host "Effective HTTP_ADDR=$env:HTTP_ADDR"
# other envs (ENABLE_TEST_JWT, JWKS_URL) already set above
pushd services/vault-api/cmd/server
nohup go run . > /tmp/vault-api.log 2>&1 &
popd

# short pause then show initial log entries for debugging
Start-Sleep -Seconds 2
if (Test-Path "/tmp/vault-api.log") {
    Write-Host "=== vault-api initial log ==="
    Get-Content "/tmp/vault-api.log" | Select-Object -First 20
}

# Wait for vault-api to become healthy before running tests
Write-Host "Waiting for vault-api at $ApiUrl/healthz"
$healthAttempts = 60
for ($j = 0; $j -lt $healthAttempts; $j++) {
    try {
        $h = Invoke-WebRequest -Uri ($ApiUrl + "/healthz") -UseBasicParsing -TimeoutSec 2 -ErrorAction Stop
        if ($h.StatusCode -eq 200) {
            Write-Host "vault-api is healthy"
            break
        }
    } catch {
        Start-Sleep -Seconds 1
    }
}
try {
    $check = Invoke-WebRequest -Uri ($ApiUrl + "/healthz") -UseBasicParsing -TimeoutSec 2
    if ($check.StatusCode -ne 200) {
        Write-Host "ERROR: vault-api did not become healthy; dumping /tmp/vault-api.log"
        if (Test-Path "/tmp/vault-api.log") { Get-Content "/tmp/vault-api.log" }
        exit 1
    }
} catch {
    Write-Host "ERROR: vault-api did not become healthy; dumping /tmp/vault-api.log"
    if (Test-Path "/tmp/vault-api.log") { Get-Content "/tmp/vault-api.log" }
    exit 1
}

Write-Host "Running integration tests..."
go test ./tests/integration -count=1 -v
$intExit = $LASTEXITCODE

Write-Host "Running e2e tests..."
go test ./tests/e2e -count=1 -v
$e2eExit = $LASTEXITCODE

Write-Host "Stopping local vault-api process and tearing down docker-compose..."
# kill the local vault-api if running
Get-Process -Name go -ErrorAction SilentlyContinue | Where-Object { $_.CommandLine -match "vault-api" } | ForEach-Object { Stop-Process -Id $_.Id -Force }

docker compose -f $composeFile down
Write-Host "Tearing down docker-compose (again to be sure)..."
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
