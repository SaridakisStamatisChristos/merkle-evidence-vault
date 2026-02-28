# IMPLEMENTATION_PLAN — Path to Production (Execution Version)

This execution plan implements `NEXT_STEPS.md` and replaces the previous implementation document.

## Baseline assessment

- Current estimated grade: **68/100**.
- Target grade for production launch: **90+/100**.
- Planning horizon: **12 weeks**.

## Workstreams

## WS1 — Security & Identity (Owner: Platform Security)
**Objective:** production-grade authN/authZ and key handling.

### Tasks
- Finalize strict JWT/JWKS validation policy and negative-path tests.
- Remove/forbid any non-production auth bypass behavior in runtime config.
- Integrate KMS/HSM signing path for checkpoint operations.
- Add key rotation automation and audit trail hooks.

### Deliverables
- `auth-hardening-report.md` (new artifact in `evidence/`).
- Rotation drill output and signed verification transcript.

### Exit criteria
- Security review approved.
- Rotation drill executed successfully twice (including rollback case).

---

## WS2 — Integrity & Verifiability (Owner: Core Backend)
**Objective:** durable verifiable checkpoints and replay safety.

### Tasks
- Persist STH/checkpoint history with immutable append semantics.
- Add auditor-focused API for listing/filtering/history replay.
- Add integration tests for restart/failover replay verification.

### Deliverables
- Migration(s) for durable checkpoint storage.
- API contract updates and verifier tests.

### Exit criteria
- Independent verifier can confirm proofs/checkpoints after restart.
- No data-loss in simulated failure scenarios.

---

## WS3 — Reliability, SLOs, and Observability (Owner: SRE)
**Objective:** measurable reliability and operable incidents.

### Tasks
- Define service SLOs (ingest, proof, checkpoint freshness, error rate).
- Implement burn-rate alerts and severity mapping.
- Add telemetry completeness checks (metrics/traces/log correlation).
- Run incident game day against real alerts/runbooks.

### Deliverables
- SLO dashboard + alert rules.
- Game day report with remediation actions.

### Exit criteria
- On-call can detect/triage/mitigate seeded incident within target MTTR.

---

## WS4 — Quality & Assurance (Owner: QA + Language Owners)
**Objective:** sustained confidence via testing and fuzz assurance.

### Tasks
- Expand abuse-case and property-test coverage for critical paths.
- Schedule long-run fuzz jobs with artifact retention and minimization.
- Add regression replay checks in CI for minimized crash corpus.

### Deliverables
- Weekly fuzz summary artifact.
- Updated test matrix mapped to critical risks.

### Exit criteria
- 30 days of clean fuzz runs or documented mitigations for findings.

---

## WS5 — Release Engineering & Compliance Evidence (Owner: DevEx)
**Objective:** repeatable, signed, auditable releases.

### Tasks
- Generate SBOMs and enforce dependency policy gates.
- Sign release artifacts/images and verify signatures in deploy pipeline.
- Add production readiness checklist to release process.

### Deliverables
- Signed release manifest.
- Evidence bundle linking tests, scans, drills, and approvals.

### Exit criteria
- Release pipeline blocks unsigned or policy-violating artifacts.

## Milestones

### Milestone A (Week 3)
- WS1 and WS2 design complete; auth/key policy frozen.

### Milestone B (Week 6)
- Durable checkpoint persistence + initial SLO dashboards live.

### Milestone C (Week 9)
- Game day complete; fuzz retention and replay loop operational.

### Milestone D (Week 12)
- Signed release chain + readiness review passed (go/no-go).

## Risks and mitigations
- **Risk:** KMS/HSM integration delays.
  - **Mitigation:** provide adapter abstraction and staged provider rollout.
- **Risk:** Alert noise reduces trust.
  - **Mitigation:** tune thresholds during game day and postmortem loops.
- **Risk:** hidden scale bottlenecks.
  - **Mitigation:** run capacity tests before launch gate, not after.

## Launch gate checklist
- [ ] Security sign-off (no unresolved high/critical).
- [ ] Integrity verification sign-off (durable checkpoints + replay).
- [ ] SLO/observability sign-off (alerts and dashboards validated).
- [ ] QA sign-off (test matrix + fuzz confidence window).
- [ ] Release sign-off (signed artifacts + SBOM + approvals).
