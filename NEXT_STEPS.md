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
- [x] Performed branch & PR cleanup and consolidated recovered work onto `main`; pushed the recovered WIP changes and verified tests on `main`.

Immediate next steps
--------------------
- Add PR metadata: reviewers, labels, and a brief PR description with the test run summary and known caveats for Windows/PowerShell. (Owner: repo maintainer)
- Remove `ENABLE_TEST_JWT` from CI and replace with a JWKS stub or provide CI-signed JWKS tokens so CI enforces real JWT verification. (Owner: infra/security)
- Add migration integration tests that assert the DB schema, run `persistence/migrations`, and validate persisted evidence/audit records after upgrade paths. (Owner: backend/data)
- Harden CI job matrix and stability: add healthchecks, timeouts, and resource limits for service containers used by integration/e2e runs. (Owner: infra/CI)
- Add a lightweight JWKS test harness (local stub server) for developer runs so tests can exercise real JWKS validation without external dependencies. (Owner: backend/test)


- **Context**
- Current: integration and e2e tests pass locally and CI reports green. `vault-api` has
  minimal in-memory implementations for ingest/audit/checkpoints used by e2e; a Postgres-backed
  `pgStore` is available and exercised by integration runs.
- Primary unresolved risks: frontend XSS (CSP/DOMPurify), merkle-engine
  fuzz coverage, auth (JWT/JWKS), persistent storage for evidence/audit,
  cryptographic checkpoint signing.

Priority 1 — Safety & auth (0-2 weeks)
  (Completed: JWKS middleware implemented; Postgres-backed store added and exercised.)

Priority 2 — Integrity hardening (1-3 weeks)
- Fuzzing: add and run cargo-fuzz targets for `merkle-engine::tree::append_leaf`.
  - Deliverable: reproducible fuzz runs with 60+ seconds coverage and fixes
  - Owner: Rust eng
  - Estimate: 3-7 days

- Implement cryptographic checkpoint signing with proper key protection.
  - Deliverable: checkpoint-svc signs STHs with ed25519; keys in HSM or KMS.
  - Owner: infra/security
  - Estimate: 1-2 weeks

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
- Integrate HSM/KMS for signing keys and rotate key workflows.

How to pick the next task
------------------------
If your priority is security/hardness for production, start with Priority 1 and
Priority 2 tasks in parallel. If you want to stabilize developer workflows and
CI, implement the CI pipeline next.

Want me to start on one of these items? Reply with the task name (e.g. "JWT auth",
"Postgres audit migration", "cargo-fuzz merkle-engine", "CI pipeline") and I'll
create a scoped plan and begin implementing.
