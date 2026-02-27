# Implementation Plan — Merkle Evidence Vault

This document converts `NEXT_STEPS.md` priorities into an actionable implementation plan. The auth/JWT layer is intentionally scheduled last to match your request.

## Goals
- Move from test-shim to production-ready: integrity, signing, observability, CI hardening, and frontend safety.
- Keep the auth layer last for integration after foundational systems are stable.

## Status Update
- CI: green (recent push).  
- Lockfile: `frontend/audit-dashboard/package-lock.json` regenerated and committed.  
- Completed: DB migration integration tests, frontend DOMPurify tests, CI/infra hardening.  
- In progress: scheduled long fuzz runs (CI scheduled job running); awaiting artifact collection and crash analysis.

## Sequenced Work (ordered for implementation)

1. Add DB migration integration tests
   - Deliverable: tests that run `persistence/migrations`, verify schema, and validate persisted evidence/audit records after upgrade paths.
   - Owner: backend/data
   - Estimate: 2-4 days
   - Notes: Add a dedicated test container for Postgres; use `scripts/run-integration.ps1` as template.
   - Status: completed (added `tests/integration/migration_test.go`).

2. Harden checkpoint signing (KMS/HSM + rotation + audit)
   - Deliverable: replace test keyfile usage with pluggable KMS/HSM-backed signer, rotation workflow, and audit hooks.
   - Owner: infra/security
   - Estimate: 1-2 weeks
   - Notes: Keep an env-driven pluggable signer interface in `checkpoint-svc` and `vault-api`.

3. Schedule longer cargo-fuzz runs & artifact collection
   - Deliverable: scheduled CI job (nightly/weekly) for `merkle-engine` fuzz runs and crash artifact minimization/collection.
   - Owner: Rust eng / infra
   - Estimate: 1-3 days to add scheduling; ongoing maintenance
   - Status: scheduled in CI (daily/scheduled workflow running); awaiting results/artifacts.
   - Notes: Integrate crash artifact upload and retention policy.

4. Add verifiable STHs in audit trail (replace test-only signatures)
   - Deliverable: append verifiable Signed Tree Heads (STHs) into audit records and expose STHs for verification.
   - Owner: backend/merkle
   - Estimate: 3-7 days
   - Notes: Align with checkpoint signing changes.

5. Frontend security: CSP, DOMPurify, display sanitization tests
   - Deliverable: CSP policy, DOMPurify integration for proof/UI flows, frontend security tests validating sanitized output.
   - Owner: frontend
   - Estimate: 3-10 days
   - Status: sanitizer and unit test added; lockfile updated.
   - Notes: Add automated tests that render proofs and assert no unsafe HTML is emitted.

6. CI/infra hardening: healthchecks, timeouts, resource limits
   - Deliverable: compose healthchecks, container limits, CI job timeouts, and more robust integration matrix.
   - Owner: infra/CI
   - Estimate: 3-7 days
   - Status: healthchecks and resource limits added to `ops/docker/docker-compose.yml` and CI updated.
   - Notes: Use `ops/docker/docker-compose.yml` updates and improve GitHub Actions matrix.

7. Observability: Prometheus metrics, dashboards, alerts
   - Deliverable: basic metrics (ingest rate, sign ops, DB migrations), Prometheus exporter, Grafana dashboard and alert rules.
   - Owner: observability/infra
   - Estimate: 3-7 days
   - Notes: Instrument `vault-api`, `merkle-engine`, and `checkpoint-svc`.

8. Finalize auth layer: production JWT/JWKS integration (LAST)
   - Deliverable: integrate production JWKS provider, robust JWT validation, caching, rotation handling, and test coverage.
   - Owner: infra/security / backend
   - Estimate: 3-10 days
   - Notes: Because other systems (KMS, DB, STHs, frontend) will be stabilized first, this step should integrate into a hardened surface.

## Quick next actions
- Pick the first step to start (recommended: DB migration integration tests).  
- For step 1, I can scaffold test harness changes and a CI job to run migrations — say `yes` to proceed.

---
Generated from `NEXT_STEPS.md` priorities; auth intentionally scheduled last per request.
