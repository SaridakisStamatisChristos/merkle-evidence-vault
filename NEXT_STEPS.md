# NEXT_STEPS — Production Plan (2026 Refresh)

This file replaces the previous ad-hoc task list with a production-focused plan based on repository inspection.

## Production-grade estimate

**Estimated production grade: 68 / 100 ("beta-hardening")**

### Why this score
- **Strong foundations (positive):**
  - Clear architecture and operating docs exist (`README.md`, architecture ADRs, runbooks, threat model).
  - CI and fuzz workflows are present (`.github/workflows/ci.yml`, `fuzz.yml`, `ci_fuzz.yml`).
  - Integration/property/e2e test suites exist in `tests/`.
  - Security/ops artifacts exist (K8s policies, observability dashboards/alerts, policy docs).
- **Key production gaps (score reducers):**
  - Existing confidence docs still describe the system as not yet production-ready (`CONFIDENCE.md`, `CONFIDENCE.yaml`).
  - Remaining risk concentration around auth hardening, checkpoint/signing durability, and operational SLO enforcement.
  - Evidence of partial/in-progress hardening in prior plans (`IMPLEMENTATION_PLAN.md` previously marked several major items as pending).

## Top 10 next steps (priority order)

1. **Close auth/RBAC hardening to production standard**
   - Enforce strict JWT validation behavior (issuer/audience/skew/rotation failure modes).
   - Add abuse-case tests for token expiry, key rotation races, and malformed bearer handling.
   - Exit criteria: security review sign-off + no test-shim fallback paths.

2. **Productionize checkpoint signing + key lifecycle**
   - Move to KMS/HSM backed signing path in all environments above dev.
   - Implement key rotation runbook automation + signed audit records.
   - Exit criteria: successful rotation drill and verification from independent auditor flow.

3. **Make STH/checkpoint history fully durable and queryable**
   - Persist signed tree heads/checkpoints with retention and provenance metadata.
   - Add API pagination/filtering for auditor verification workflows.
   - Exit criteria: deterministic replay and verification across restart/failover.

4. **Raise fuzzing from baseline to sustained assurance**
   - Add scheduled long-run fuzz budget with artifact retention/minimization.
   - Track coverage and crash triage SLAs.
   - Exit criteria: 30-day clean run window with published artifacts.

5. **Define and enforce SLOs**
   - Set ingest latency, proof latency, checkpoint freshness, and error-budget SLOs.
   - Tie alerts to SLO burn rates.
   - Exit criteria: SLO dashboard + alert runbook validated in game day.

6. **Harden observability end-to-end**
   - Add RED/USE metrics per service and trace correlation IDs across request boundaries.
   - Validate dashboards and alert quality (low noise, actionable thresholds).
   - Exit criteria: on-call dry run resolves seeded incident using only docs + telemetry.

7. **Disaster recovery and backup verification**
   - Automate backup, restore, and integrity verification for DB + signing metadata.
   - Define RPO/RTO and exercise them.
   - Exit criteria: quarterly restore drill meets RPO/RTO targets.

8. **Supply-chain and release governance**
   - Add SBOM generation, dependency policy gates, and image signing/verification.
   - Enforce changelog + release checklist quality gates.
   - Exit criteria: signed release artifact chain and reproducible build evidence.

9. **Performance and capacity qualification**
   - Build representative load profiles for ingest/proof/checkpoint operations.
   - Identify scaling limits and tune Postgres/queue/service resource policies.
   - Exit criteria: capacity model with tested headroom for target traffic.

10. **Operational readiness and compliance evidence**
   - Consolidate runbooks into incident classes with ownership/escalation paths.
   - Produce auditable evidence bundle (security controls + test + ops drill records).
   - Exit criteria: readiness review passes with no Sev-1 open risks.

## 90-day phased execution

### Phase 1 (Weeks 1–3): Security and integrity closure
- Steps 1–4 execution start.
- Mandatory outputs:
  - Auth hardening test matrix.
  - KMS/HSM integration decision + rollout plan.
  - Durable checkpoint schema/API proposal.
  - Scheduled fuzz retention pipeline.

### Phase 2 (Weeks 4–7): Operability and resilience
- Steps 5–7 execution.
- Mandatory outputs:
  - SLOs and burn-rate alerts enabled.
  - Observability runbook validation report.
  - Backup/restore drill report.

### Phase 3 (Weeks 8–12): Release confidence and scale
- Steps 8–10 execution.
- Mandatory outputs:
  - Signed, policy-gated release pipeline.
  - Capacity report with scaling thresholds.
  - Final production-readiness evidence package.

## Definition of done for “production-ready”
- No critical/high unresolved security findings.
- All core user flows covered by deterministic integration/e2e and reliability tests.
- SLOs defined, measured, and alerting with proven incident response.
- Key management, checkpoint verification, and recovery drills validated.
- Release artifacts signed, traceable, and reproducible.
