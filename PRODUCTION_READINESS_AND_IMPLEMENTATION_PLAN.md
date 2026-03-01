# Production Readiness Estimate & Implementation Plan (2026-03)

## Executive estimate

**Estimated production readiness: 76 / 100 (late-stage pre-production).**

The codebase has strong testing, architecture, security controls, and operational scaffolding, but it is not yet production-ready due to a small number of high-impact closure items in durability, auth policy consistency, release governance, and runbook/SLO validation.

## Evidence snapshot

### What is strong now
- **Broad test surface exists** across unit, property, integration, and e2e (`tests/`, middleware tests, signer/metrics tests).
- **Security controls are materially improved** (JWT/JWKS middleware with strict modes, role checks, claim checks; frontend sanitization + CSP policy modules).
- **Operational artifacts are present** (Kubernetes manifests, network policy, dashboards, alerts, runbooks, threat model).
- **Fuzzing process and replay artifacts exist** with corpus + metadata retained.

### Remaining blockers to production claim
1. **Confidence documents still assert non-production posture**, and currently conflict with improved code paths.
2. **Checkpoint durability and recovery validation** need explicit launch-gate evidence (repeatable restore + replay verification).
3. **Auth policy rollout consistency** must be validated across environments (dev/test strictness exceptions, no accidental fallback in prod).
4. **Release governance hardening** (SBOM/signing/enforcement evidence) needs explicit must-pass gates.
5. **SLO + incident drill evidence** needs one end-to-end validation cycle with measured MTTR/error budget handling.

## 30-60-90 day implementation plan

## Phase 1 (Days 0-30): Launch blockers closure

### Stream A — Security and auth finalization
- Freeze an environment matrix for `AUTH_POLICY` and JWT requirements.
- Add a startup fail-fast check in production profiles when required JWT env vars are missing.
- Add one integration test pack for malformed tokens, key rotation race behavior, and missing `kid` in strict mode.

**Exit criteria**
- Production profile cannot start with insecure auth fallback.
- Auth test matrix passes in CI.

### Stream B — Checkpoint and data durability
- Verify durable checkpoint persistence paths and retention behavior.
- Execute backup/restore + replay drill and record evidence artifact.
- Add a deterministic replay verification command to CI/nightly.

**Exit criteria**
- Successful restore + replay from backup in a fresh environment.

## Phase 2 (Days 31-60): Operability and reliability proof

### Stream C — SLOs and observability
- Finalize SLOs for ingest latency, proof latency, checkpoint freshness, and error rate.
- Wire burn-rate alerts to severity levels and on-call action runbooks.
- Run one game-day scenario (Merkle engine unavailable + backlog growth).

**Exit criteria**
- Alert-to-mitigation flow validated with acceptable MTTR.

### Stream D — Assurance and adversarial testing
- Schedule periodic long-run fuzz jobs and automated crash triage outputs.
- Expand negative-path tests for API authz, pipeline idempotency windows, and checkpoint verification.

**Exit criteria**
- No unresolved high-severity findings from the extended test/fuzz cycle.

## Phase 3 (Days 61-90): Release readiness and governance

### Stream E — Supply chain and release policy
- Enforce SBOM generation and signed image/artifact verification in release pipeline.
- Add release gate requiring: tests, fuzz status, security scan, migration check, and rollback recipe.

### Stream F — Documentation and readiness alignment
- Update `CONFIDENCE.md` and `CONFIDENCE.yaml` with current implementation reality and measured results.
- Produce a final production-readiness report mapping each risk to objective evidence.

**Exit criteria**
- Release pipeline blocks non-compliant artifacts.
- Readiness review can trace every control to test/drill evidence.

## Priority backlog (ordered)
1. Production fail-fast auth configuration guardrails.
2. Restore/replay drill automation and evidence capture.
3. SLO definitions + burn-rate alerts + game-day execution.
4. CI/nightly fuzz and adversarial regression expansion.
5. Signed release + SBOM + policy-gated deploy chain.
6. Confidence document refresh and final launch checklist.

## Definition of production ready
- No open critical/high security issues.
- Deterministic restore + replay validated from backup.
- SLOs enforced with alerting and demonstrated incident response.
- Auth/signing configuration cannot silently degrade in production.
- Release artifacts are signed, traceable, and policy-verified.
