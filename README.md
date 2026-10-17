# Merkle Evidence Vault

Merkle Evidence Vault is an append-only evidence store backed by an RFC 6962 binary Merkle tree, Ed25519-signed checkpoints, and an offline-verifiable `.evb` bundle format.

Quick start (local development):

```bash
export MERKLE_SIGNING_KEY_HEX=$(openssl rand -hex 32)
make dev-up          # Postgres + Redis + Redpanda
make migrate         # Run DB migrations
make build-all       # Rust + Go + React
make test            # Core unit tests
make confidence      # Confidence gate/report
```

## Assurance and evidence

Evidence is part of the product surface in this repository, not disposable build output. The project deliberately retains dated proof packs, restore and game-day drill artifacts, confidence assertions, architecture decisions, runbooks, and reproducibility material so an evaluator can distinguish implemented controls from executed verification.

- `CONFIDENCE.md` / `CONFIDENCE.yaml` describe the current assurance model and its evidence boundaries.
- `evidence/README.md` explains the evidence layout and status semantics.
- `READINESS.md` defines the current pre-production readiness gates without implementation-phase chronology.
- `evidence/proof-pack/<YYYY-MM-DD>/` contains point-in-time evidence snapshots. A captured failed drill remains evidence and must not be relabeled as a pass.

### Latest repository proof pack: 2026-03-01

- ✅ CI trace: a successful **Artifact Guard** workflow run is pinned in `evidence/proof-pack/2026-03-01/ci_run.txt`.
- ⚠️ Restore drill: execution artifacts were captured, but the retained result reports `pass: false` in `evidence/proof-pack/2026-03-01/drills/restore/drill_summary.json`.
- ⚠️ Game-day drill: execution artifacts were captured, but the retained result reports `pass: false` in `evidence/proof-pack/2026-03-01/drills/game-day/game_day_report.json`.
- 📦 Supply chain: proof-pack directories exist for SBOM/signing evidence, but the dated pack does not contain the final generated evidence set.
- ✅ Auth policy: the policy matrix is packaged at `evidence/proof-pack/2026-03-01/auth_policy_matrix.md`.

The existence of an evidence artifact means it was captured; it does **not** imply that the underlying check passed. See `READINESS.md` before making readiness claims.

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

Run from the repository root and export the E2E environment variables in the same shell that runs `make` or `go test`:

```powershell
$env:E2E_API_URL        = 'http://localhost:8080'
$env:E2E_INGESTER_TOKEN = 'ingester-token-example'   # include 'ingest' or 'ingester' in the string
$env:E2E_AUDITOR_TOKEN  = 'auditor-token-example'    # include 'auditor' in the string
make integration-test
```

Notes:
- The server in Docker Compose listens on HTTP at host port `8080` (container port `8443`), so use an `http://` URL for `E2E_API_URL` unless you enable HTTPS in the server.
- Set the variables directly in PowerShell (use `$env:...`) so `make` / `go test` inherit them.
- `scripts/run-integration.ps1` is available for the CI-compatible integration flow.

## Fuzzing & hardening

- ASAN fuzzing: 71,232,156 executions in 601s (~118.5k exec/s), ASAN-enabled, no crashes.
- Final peak RSS: ~471 MB (ASAN overhead observed, no sustained leak growth).
- Minimized corpus and dictionary are committed under `fuzz/corpus/minimized` and `fuzz/dict.txt` for regression replay.
- Replayable artifacts and run metadata are stored at `fuzz/last_run_meta.md`.

Run a short bounded fuzz using the committed corpus:

```bash
cd services/merkle-engine
cargo fuzz run tree_append_leaf ../../fuzz/corpus/minimized -dict=../../fuzz/dict.txt -jobs=1 -workers=1
```

Replay the minimized corpus deterministically:

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

Artifacts are written to `artifacts/drills/<timestamp>/` and include `drill_summary.json`, logs, and verifier output. Inspect the `pass` field in the summary; artifact creation alone is not a passing result.

## Release governance gates

Release governance is enforced by `.github/workflows/release-governance.yml`. A release candidate is gated on:

- unit tests plus integration/E2E verification;
- migration check (`TestApplyMigrations`);
- fuzz-status evidence from `main`;
- SBOM generation and vulnerability scans;
- signing and signature verification for SBOM artifacts.

No release or tag is created by the proof-pack workflow.
