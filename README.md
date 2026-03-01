# Merkle Evidence Vault

This repository contains the Merkle Evidence Vault: an append-only evidence
store backed by an RFC 6962 binary Merkle tree, Ed25519-signed checkpoints,
and an offline-verifiable `.evb` bundle format.

Quick start (local dev):

```bash
export MERKLE_SIGNING_KEY_HEX=$(openssl rand -hex 32)
make dev-up          # Postgres + Redis + Redpanda
make migrate         # Run DB migrations
make build-all       # Rust + Go + React
make test            # Full test suite
make confidence      # Confidence gate (must pass)
```

See `CONFIDENCE.md` for the confidence assessment and production readiness.

## Authentication configuration (vault-api)

`services/vault-api` enforces startup auth guardrails. Use the table below to configure auth safely.

| Variable | Required | Description |
|---|---:|---|
| `AUTH_POLICY` | yes | One of `dev`, `jwks_strict`, `jwks_rbac`. |
| `ENV` (or `APP_ENV`) | recommended | Environment profile (`dev`, `prod`, etc.). |
| `DEPLOYMENT` | optional | Deployment profile; `prod` is treated as production startup. |
| `ALLOW_INSECURE_DEV` | optional | Must be `true` to allow `AUTH_POLICY=dev` outside `ENV=dev`. |
| `JWKS_URL` | required for `jwks_strict`/`jwks_rbac` | JWKS endpoint URL used for token signature verification. |
| `JWT_ISSUER` | required for `jwks_strict`/`jwks_rbac` | Required token issuer. |
| `JWT_AUDIENCE` | required for `jwks_strict`/`jwks_rbac` | Required token audience (comma-separated supported). |

### Fail-fast rules

- `AUTH_POLICY=dev` is allowed only when `ENV=dev` **or** `ALLOW_INSECURE_DEV=true`.
- When `ENV=prod` or `DEPLOYMENT=prod`, startup rejects `AUTH_POLICY=dev`.
- `AUTH_POLICY=jwks_strict` and `AUTH_POLICY=jwks_rbac` require `JWKS_URL`, `JWT_ISSUER`, and `JWT_AUDIENCE` at startup.


### Running integration & E2E tests (PowerShell)

When running the repository's integration and end-to-end tests locally on Windows/PowerShell you must export the E2E environment variables in the same shell that runs `make` or `go test`. The test harness expects an API URL and two tokens (ingester and auditor). Example (PowerShell):

```powershell
$env:E2E_API_URL        = 'http://localhost:8080'
$env:E2E_INGESTER_TOKEN = 'ingester-token-example'   # include 'ingest' or 'ingester' in the string
$env:E2E_AUDITOR_TOKEN  = 'auditor-token-example'    # include 'auditor' in the string
cd C:\Users\scsar\Desktop\merkle
make integration-test
```

Notes:
- The server in the Docker compose listens on HTTP at host port `8080` (container port `8443`), so use an `http://` URL for `E2E_API_URL` unless you enable HTTPS in the server.
- Set the variables directly in PowerShell (use `$env:...`) so `make` / `go test` inherit them — do not wrap everything in a single quoted string passed to a subshell.
- If you prefer a helper, there's a PowerShell helper script at `scripts/run-integration.ps1` (see repository).

## Fuzzing & Hardening

- ASAN fuzzing: 71,232,156 executions in 601s (~118.5k exec/s), ASAN-enabled, no crashes.  
- Final peak RSS: ~471 MB (ASAN overhead observed, no sustained leak growth).  
- Minimized corpus & recommended dictionary committed (`fuzz/corpus/minimized`, `fuzz/dict.txt`) to allow reproducible regression replay.  
- Replayable artifacts and run metadata are stored at `fuzz/last_run_meta.md`.  

To run a short bounded fuzz locally using the committed corpus:

```bash
cd services/merkle-engine
cargo fuzz run tree_append_leaf ../../fuzz/corpus/minimized -dict=../../fuzz/dict.txt -jobs=1 -workers=1
```

To replay the minimized corpus deterministically:

```bash
./fuzz/replay_minimized.sh
```

## Durability drill (backup/restore + replay verification)

Run the restore drill locally:

```bash
./scripts/drill_restore.sh
```

PowerShell:

```powershell
./scripts/drill_restore.ps1
```

Artifacts are written to `artifacts/drills/<timestamp>/` and include:
- `drill_summary.json`
- logs under `logs/`
- `verifier_output.json`



## Release governance gates

Release governance is enforced by `.github/workflows/release-governance.yml`.
A release is blocked unless all gates pass:

- unit tests + integration/e2e
- migration check (`TestApplyMigrations`)
- fuzz status gate (latest successful fuzz workflow on `main`)
- SBOM generation (repo, Go, Node, images) + vulnerability scans
- keyless signing + signature verification for SBOM artifacts
