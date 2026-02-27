# NEXT_STEPS — Merkle Evidence Vault (post-merge snapshot)

This document tracks prioritized tasks moving the project from development/test-shim state toward production readiness.

## Recent progress
- [x] Added `scripts/run-integration.ps1` to stand up compose, run integration/e2e, and clean up.
- [x] Implemented JWKS-capable JWT middleware in `services/vault-api/middleware`; removed implicit test-mode fallback.
- [x] Added `Store` abstraction and Postgres-backed `pgStore` in `services/vault-api/store`; exercised in integration/e2e paths.
- [x] Updated `ops/docker/docker-compose.yml` with service healthchecks/resource limits and reproducible env wiring.
- [x] Added `.github/workflows/ci.yml` for formatting/lint + integration/e2e helper execution.
- [x] Stabilized CI JWKS flow with `scripts/ci_jwks.go` and CI env handoff (without `ENABLE_TEST_JWT` bypass).
- [x] Added fuzzing support for `merkle-engine` (fuzz target + ASAN-enabled CI bounded runs).
- [x] Added checkpoint signing scaffold (`checkpoint-svc`) and integration-style signature verification test.
- [x] Added migration integration tests (`tests/integration/migration_test.go`).
- [x] Added frontend sanitization (`DOMPurify`) and sanitization unit tests.

## Immediate next steps
- [x] Add PR metadata process hygiene (reviewers/labels/template summary conventions). (Owner: repo maintainer)
- [x] Add migration integration tests asserting schema + upgrade-path persistence. (Owner: backend/data)
- [x] Harden CI stability with compose healthchecks, timeouts, and resource limits. (Owner: infra/CI)
- [x] Complete scheduled longer fuzz runs with crash artifact collection/minimization and retention policy. (Owner: Rust eng / infra)
- [x] Productionize checkpoint signing by replacing test keyfiles with KMS/HSM-backed keys plus rotation/audit integration. (Owner: infra/security)

## Priority roadmap

### Priority 1 — Safety & auth (0–2 weeks)
- [x] JWKS middleware implemented and CI stabilized with JWKS stub.
- [x] Final production JWT/JWKS hardening pass implemented for strict bearer parsing, claim validation (iss/aud/exp/nbf/iat), configurable alg/skew policy, and documented failure-mode review.

### Priority 2 — Integrity hardening (1–3 weeks)
- [x] Fuzzing baseline complete; scheduled runs enabled; artifact-minimization pipeline + retention policy implemented.
- [x] Checkpoint signing scaffold now supports KMS-provider abstraction, env-based key wiring, and signing audit logs.

### Priority 3 — Frontend security & UX (1–3 weeks)
- [x] DOMPurify sanitization path and tests added.
- [x] Add/verify CSP header policy for all dashboard/API-served frontend responses.

### Priority 4 — CI, observability, and ops (1–2 weeks)
- [x] Compose healthchecks/resource limits and CI hardening baseline completed.
- [ ] Add Prometheus metrics and dashboard/alerts across `vault-api`, `merkle-engine`, and `checkpoint-svc`.

## Low priority / longer term
- [~] Replace test-only checkpoint signature flow with verifiable STH endpoints in the audit trail (latest verification endpoint added; full persisted STH history still pending).

## Suggested execution order
1. Add observability metrics + dashboards/alerts.
2. Expand checkpoint flow from latest-only verification to persisted verifiable STH history.
