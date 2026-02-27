# Implementation Plan — Merkle Evidence Vault

This document converts `NEXT_STEPS.md` priorities into an actionable implementation plan. The auth/JWT layer remains intentionally scheduled last.

## Goals
- Move from test-shim to production-ready: integrity, signing, observability, CI hardening, and frontend safety.
- Keep the auth layer last for integration after foundational systems are stable.

## Current Status Snapshot
- [x] CI green and integration/e2e helper stabilized.
- [x] DB migration integration tests added (`tests/integration/migration_test.go`).
- [x] Frontend DOMPurify sanitization + tests added (`frontend/audit-dashboard/src/sanitize.js`, `frontend/audit-dashboard/src/__tests__/sanitize.test.js`).
- [x] CI/infra hardening completed for compose healthchecks/resource limits and CI timeout updates.
- [~] Scheduled fuzz workflow enabled; artifact collection/minimization follow-through still pending.

## Sequenced Work (ordered)

1. [x] Add DB migration integration tests
   - Deliverable: tests that run `persistence/migrations`, verify schema, and validate persisted evidence/audit records after upgrade paths.
   - Status: completed.

2. [ ] Harden checkpoint signing (KMS/HSM + rotation + audit)
   - Deliverable: replace test keyfile usage with pluggable KMS/HSM-backed signer, rotation workflow, and audit hooks.
   - Notes: keep env-driven signer interface in `checkpoint-svc` and `vault-api`.

3. [ ] Complete long cargo-fuzz runs + artifact collection
   - Deliverable: scheduled CI job (nightly/weekly) for `merkle-engine` fuzz runs plus crash artifact minimization/collection.
   - Status: schedule is in place; still need artifact retention/minimization loop finalized.

4. [ ] Add verifiable STHs in audit trail (replace test-only signatures)
   - Deliverable: append verifiable Signed Tree Heads (STHs) into audit records and expose STHs for verification.
   - Notes: align this with checkpoint-signing hardening.

5. [x] Frontend security baseline: DOMPurify + sanitization tests
   - Deliverable: DOMPurify integration and frontend tests validating sanitized output.
   - Remaining: formal CSP rollout can be tracked as a follow-on hardening item if desired.

6. [x] CI/infra hardening baseline
   - Deliverable: compose healthchecks, container resource limits, and CI stability hardening.
   - Status: completed.

7. [ ] Observability: Prometheus metrics, dashboards, alerts
   - Deliverable: basic metrics (ingest rate, sign ops, DB migrations), Prometheus exporter, Grafana dashboard and alerts.

8. [ ] Finalize auth layer: production JWT/JWKS integration (LAST)
   - Deliverable: production JWKS provider integration, robust JWT validation/caching/rotation behavior, and test coverage.

## Next execution focus
- Recommended next: finish item 3 (fuzz artifact pipeline), then item 2 (KMS/HSM signing), then item 4 (verifiable STHs).

---
Generated from `NEXT_STEPS.md` priorities; statuses updated to reflect completed vs pending work.
