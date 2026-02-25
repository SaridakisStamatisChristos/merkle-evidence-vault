# NEXT_STEPS — Merkle Evidence Vault (post-merge snapshot)

This document lists discrete, prioritized tasks to move the project from
its current development/test-shim state toward a production-ready system.

Recent progress
---------------
- Added `scripts/run-integration.ps1` — a PowerShell helper that sets E2E environment variables, brings up `docker-compose`, runs integration and e2e tests, and tears down the stack.
- Documented PowerShell usage and example env exports in `README.md`.
- Created branch `feature/add-run-integration-script` and opened PR: https://github.com/SaridakisStamatisChristos/merkle-evidence-vault/pull/1
- Integration and e2e tests pass locally when run via the helper script (in-memory store).

Immediate next steps
--------------------
- Validate `pgStore` integration: run the full integration + e2e flows with `DATABASE_URL` set so the Postgres-backed store is exercised (verify schema, migrations, and correctness). (Owner: backend/data)
- Harden CI: add a GitHub Actions workflow to run `gofmt`/`go vet` and execute the integration+e2e test flows (use service containers / docker-compose or a job that builds and runs the compose stack). (Owner: infra/CI)
- Add PR metadata: reviewers, labels, and a brief PR description with the test run summary and known caveats for Windows/PowerShell. (Owner: repo maintainer)
- Add an automated check that the e2e tests use the same HTTP scheme as the server (avoid https/http mismatch in test runners). (Owner: backend/test)
- Validate `pgStore` under load and run migration integration tests against the `persistence/migrations` scripts. (Owner: backend/data)


Context
-------
- Current: property, integration and e2e tests pass locally. `vault-api` has
  minimal in-memory implementations for ingest/audit/checkpoints used by e2e.
- Primary unresolved risks: frontend XSS (CSP/DOMPurify), merkle-engine
  fuzz coverage, auth (JWT/JWKS), persistent storage for evidence/audit,
  cryptographic checkpoint signing.

Priority 1 — Safety & auth (0-2 weeks)
- Implement JWKS-backed JWT validation middleware in `services/vault-api/middleware`.
  - Deliverable: `middleware/jwt.go`, role extraction, unit tests; remove test-shim
    substring checks in handlers.
  - Owner: backend
  - Estimate: 3-5 dev days

- Migrate audit & evidence from in-memory maps to Postgres-backed repositories
  using existing `persistence/migrations` and `persistence` connector.
  - Deliverable: repository layer + migrations integration tests
  - Owner: backend/data
  - Estimate: 1-2 weeks

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
- Add GitHub Actions workflow to run `gofmt`, `go vet`, unit/property tests,
  integration and e2e (via docker-compose / service containers) on PRs.
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
