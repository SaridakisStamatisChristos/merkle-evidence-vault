# Readiness and Operations Evidence

Merkle Evidence Vault is currently **pre-production and evidence-gated**. This document describes durable readiness criteria and the evidence required to change that posture; it intentionally avoids implementation-phase timelines and percentage-complete scoring.

## Evidence sources

Readiness decisions should be made from the repository's current assurance surfaces:

- `CONFIDENCE.md` and `CONFIDENCE.yaml` — confidence model, evidence mapping, and explicit claim boundary.
- `evidence/proof-pack/<YYYY-MM-DD>/` — dated execution snapshots.
- `artifacts/drills/<timestamp>/` and `artifacts/game-day/<timestamp>/` — raw operational drill outputs.
- `.github/workflows/ci.yml` — pull-request and branch verification.
- `.github/workflows/release-governance.yml` — release-candidate security and provenance gates.
- `evidence/runbooks/` — operational procedures.

A packaged artifact is not automatically a passing result. Where a drill artifact contains a `pass` field, that field determines the captured outcome.

## Current evidence boundary

The latest proof pack committed in the repository is dated **2026-03-01**.

| Area | Current repository evidence | Readiness interpretation |
|---|---|---|
| CI traceability | A successful Artifact Guard run is pinned in `evidence/proof-pack/2026-03-01/ci_run.txt`. | Traceability demonstrated for that workflow/run only. |
| Restore/replay drill | The pack contains an executed restore drill; `drills/restore/drill_summary.json` records `pass: false`. | Execution evidence exists, but successful restore/replay is not demonstrated by this pack. |
| Game-day drill | The pack contains an executed merkle-engine outage drill; `drills/game-day/game_day_report.json` records `pass: false`. | Execution evidence exists, but the scenario did not satisfy its pass criteria. |
| Authentication guardrails | `auth_policy_matrix.md` records the supported auth-policy matrix and production fail-fast constraints. | Control is documented and covered by repository tests; deployment-specific validation remains an operational responsibility. |
| Supply-chain evidence | `sbom/` and `signing/` locations exist in the dated proof pack without a complete final evidence set. | Release workflow capability exists; the dated proof pack is not a complete signed supply-chain dossier. |
| SLO/incident operations | No current proof pack establishes sustained SLO/burn-rate operation or a passing incident-response exercise. | Production-readiness claim remains blocked. |

## Readiness gates

A production-readiness claim requires evidence that all of the following are satisfied for the intended deployment environment and release candidate:

1. **Correctness and integration** — required unit, integration, E2E, migration, and confidence checks pass on the candidate revision.
2. **Restore and durability** — repeated strict-mode restore/replay drills pass with archived, machine-verifiable results.
3. **Failure recovery** — game-day/failover scenarios exercise the intended failure modes and satisfy their explicit pass criteria.
4. **Security and provenance** — release SBOM/vulnerability/signing verification gates pass and the resulting artifacts are retained.
5. **Authentication posture** — production profiles reject insecure auth configuration and deployment-specific configuration is validated.
6. **Operational reliability** — SLOs and burn-rate thresholds are defined, measured, and tied to an exercised incident-response path.
7. **Governance enforcement** — required repository/release protections are actually configured, not merely described by workflow files.

## Evidence maintenance

- Keep dated proof packs as historical snapshots of what was observed at that time.
- Preserve negative results. Replace a failed claim with new evidence from a later dated run; do not rewrite the failed raw artifact.
- Update `CONFIDENCE.md` / `CONFIDENCE.yaml` only when the evidence interpretation or confidence model changes.
- Recompute integrity manifests only when covered content legitimately changes.
- Do not create a release or tag merely to refresh readiness documentation.
