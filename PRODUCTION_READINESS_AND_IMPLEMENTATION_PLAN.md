# Production Readiness Estimate & Implementation Plan (2026-03 Refresh)

## Executive estimate

**Estimated production readiness: 84 / 100 (pre-production with gated release controls).**

The repository now includes concrete fail-fast auth startup guardrails, automated restore-drill tooling, and must-pass release governance workflows (SBOM, vulnerability scanning, signing verification). Remaining work is concentrated in operational validation and sustained evidence quality.

## Current state (implemented)

### Completed since the previous plan revision
- **Auth startup guardrails are implemented**
  - Startup now rejects insecure `AUTH_POLICY=dev` usage outside approved contexts.
  - Strict modes require required JWT/JWKS environment variables.
- **Durability drill automation is implemented**
  - Backup/restore + replay drill scripts exist for shell/PowerShell.
  - Drill artifacts are written under `artifacts/drills/<timestamp>/`.
  - Nightly/manual drill workflow uploads artifacts.
- **Release governance gates are implemented**
  - CI now generates SBOM/vulnerability reports.
  - Release workflow now enforces must-pass tests/integration/migrations/fuzz status.
  - Release workflow enforces SBOM signing and verification.

## Remaining blockers to production claim
1. **Operational proof of reliability (SLO/game-day)**
   - SLO definitions, burn-rate calibration, and at least one game-day evidence package are still required.
2. **Drill quality hardening**
   - Restore drills should be consistently passing under strict mode in CI and in a production-like environment.
3. **Durability model validation under failure stress**
   - Add deeper chaos/failover scenarios for checkpoint/replay paths.
4. **Release governance adoption maturity**
   - Ensure the new release workflow is set as a required branch/release protection gate.

## 30-60-90 day execution plan

## Phase 1 (Days 0-30): Operationalization of newly added controls

### Stream A — Auth rollout confidence
- Verify environment matrix in each deployment target (`dev`, `staging`, `prod`).
- Add a deployment checklist proving fail-fast behavior in non-dev environments.

**Exit criteria**
- No deployment profile can boot with insecure auth policy configuration.

### Stream B — Restore drill hardening
- Run restore drill on schedule with strict mode in at least one stable CI environment.
- Attach drill summaries and verifier outputs to release-readiness evidence.

**Exit criteria**
- Consecutive successful drill runs with machine-verifiable outputs.

## Phase 2 (Days 31-60): Reliability validation

### Stream C — SLOs and runbooks
- Finalize ingest/proof/checkpoint SLOs and burn-rate thresholds.
- Execute one game-day and record MTTR + follow-up actions.

**Exit criteria**
- On-call flow validated end-to-end from alert to mitigation.

### Stream D — Failure-mode expansion
- Add failover/restart scenarios for checkpoint replay and proof consistency.
- Add durability assertions for backup restore in migration-adjacent changes.

**Exit criteria**
- No unresolved high-severity issues in resilience test matrix.

## Phase 3 (Days 61-90): Launch readiness closure

### Stream E — Governance enforcement finalization
- Require release-governance workflow in release policy.
- Confirm signing verification is non-bypassable for release candidates.

### Stream F — Final readiness package
- Consolidate auth, drill, SLO, security-scan, and signing evidence in a single launch dossier.

**Exit criteria**
- Formal go/no-go decision can be made from objective evidence without gaps.

## Priority backlog (ordered)
1. SLO + burn-rate + game-day evidence package.
2. Strict-mode restore drill reliability in CI.
3. Failure-stress durability matrix (replay/failover scenarios).
4. Branch/release protection alignment with governance workflows.
5. Final launch dossier with traceable evidence links.

## Definition of production ready
- No open critical/high security vulnerabilities accepted for release.
- Auth startup policy is fail-fast and validated in production profiles.
- Restore/replay drill passes consistently with archived evidence.
- SLOs are measured and incident response is demonstrated.
- Release artifacts and SBOMs are signed and verification is mandatory.
