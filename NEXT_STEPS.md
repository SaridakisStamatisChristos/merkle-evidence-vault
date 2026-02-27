# NEXT_STEPS — Merkle Evidence Vault (post-merge snapshot)

This document lists discrete, prioritized tasks to move the project from
its current development/test-shim state toward a production-ready system.

Recent progress
---------------
- [x] Added `scripts/run-integration.ps1` — a PowerShell helper that sets E2E environment variables, brings up `docker-compose`, runs integration and e2e tests, and tears down the stack.
- [x] Implemented JWKS-capable JWT middleware in `services/vault-api/middleware` and removed implicit test-mode fallbacks; unit tests updated to enable explicit test-mode during local testing.
- [x] Added a `Store` abstraction and a Postgres-backed implementation (`pgStore`) under `services/vault-api/store`; exercised `pgStore` end-to-end by setting `DATABASE_URL` in compose and via the helper script.
- [x] Updated `ops/docker/docker-compose.yml` to include `DATABASE_URL` and enable local test-mode flags for reproducible developer runs.
- [x] Added `.github/workflows/ci.yml` to run formatting/linting and the integration+e2e helper in CI.
- [x] Integration and e2e tests pass locally via the helper script with the Postgres-backed store exercised; smoke/integration/e2e golden-path validated; CI run is green.
 - [x] Integration and e2e tests pass locally via the helper script with the Postgres-backed store exercised; smoke/integration/e2e golden-path validated; CI run is green. `package-lock.json` for frontend updated and committed to restore reproducible `npm ci`.
- [x] Performed branch & PR cleanup and consolidated recovered work onto `main`; pushed the recovered WIP changes and verified tests on `main`.
 - [x] Stabilized JWKS flow in CI: added a JWKS stub (`scripts/ci_jwks.go`), persisted its env to `$GITHUB_ENV`, and updated CI to use the stub (removed `ENABLE_TEST_JWT` bypass).
 - [x] Added fuzzing support for `merkle-engine`: fuzz target, `cargo-fuzz` layout, and an ASAN-enabled GitHub Actions job that runs a bounded fuzz session.
 - [x] Added checkpoint signing scaffold (`checkpoint-svc`) and integrated signing calls into `vault-api` with an integration-style test that verifies ed25519 signatures. KMS/HSM integration for production-grade key management remains pending.

Immediate next steps
--------------------
- Add PR metadata: reviewers, labels, and a brief PR description with the test run summary and known caveats for Windows/PowerShell. (Owner: repo maintainer)
- Add migration integration tests that assert the DB schema, run `persistence/migrations`, and validate persisted evidence/audit records after upgrade paths. (Owner: backend/data)
- Harden CI job matrix and stability: add healthchecks, timeouts, and resource limits for service containers used by integration/e2e runs. (Owner: infra/CI)
 - Harden CI job matrix and stability: add healthchecks, timeouts, and resource limits for service containers used by integration/e2e runs. (Owner: infra/CI)
   - Status: healthchecks/resource limits added to `ops/docker/docker-compose.yml` and CI workflow updated.
- Add scheduled longer fuzz runs (nightly or weekly) and crash artifact collection/minimization in CI to catch regressions over time. (Owner: Rust eng / infra)
- Productionize checkpoint signing: replace the test keyfile with KMS/HSM-backed keys and add rotation + audit integration. (Owner: infra/security)


- **Context**
- Current: integration and e2e tests pass locally and CI reports green. `vault-api` has
  minimal in-memory implementations for ingest/audit/checkpoints used by e2e; a Postgres-backed
  `pgStore` is available and exercised by integration runs.
- Primary unresolved risks: frontend XSS (CSP/DOMPurify), merkle-engine
  fuzz coverage, auth (JWT/JWKS), persistent storage for evidence/audit,
  cryptographic checkpoint signing.

Priority 1 — Safety & auth (0-2 weeks)
  (Completed: JWKS middleware implemented; Postgres-backed store added and exercised; CI now uses JWKS stub — test-mode bypass removed.)

Priority 2 — Integrity hardening (1-3 weeks)
- Fuzzing: initial cargo-fuzz target and ASAN CI run added and passing.
  - Status: Done (initial coverage). Next: scheduled longer runs + crash artifact collection.
   - Status: Done (initial coverage). Next: scheduled longer runs + crash artifact collection. CI scheduled fuzz workflow has been enabled; artifacts pending.
  - Owner: Rust eng
  - Estimate: ongoing (add scheduled runs 1-3 days)

- Checkpoint signing: scaffold and integration test implemented (ed25519). KMS/HSM integration and key rotation are pending.
  - Status: Partially done (service + tests present). Next: KMS/HSM productionization.
  - Owner: infra/security
  - Estimate: 1-2 weeks for KMS integration

Priority 3 — Frontend security & UX (1-3 weeks)
- Audit frontend rendering of proof data, add CSP headers and DOMPurify
  sanitization for all proof/display flows.
  - Deliverable: CSP policy, DOMPurify integration, security test
  - Owner: frontend
  - Estimate: 3-10 days

Priority 4 — CI, observability, and ops (1-2 weeks)
- Add Prometheus metrics and basic healthchecks to services; add compose
  healthchecks and resource limits.

Low priority / longer term
- Replace test-only checkpoint signature with verifiable STHs in the audit trail.
- Integrate HSM/KMS for signing keys and rotate key workflows (tracked above as next step for checkpoint signing).

How to pick the next task
------------------------
If your priority is security/hardness for production, start with Priority 1 and
Priority 2 tasks in parallel. If you want to stabilize developer workflows and
CI, implement the CI pipeline next.

Want me to start on one of these items? Reply with the task name (e.g. "JWT auth",
"Postgres audit migration", "cargo-fuzz merkle-engine", "CI pipeline") and I'll
create a scoped plan and begin implementing.
