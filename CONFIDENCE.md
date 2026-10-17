# CONFIDENCE — Merkle Evidence Vault v0.1.4

System raw confidence: **0.90**  
System effective confidence: **0.87** (gated pre-production)

## Summary

This assessment remains **pre-production**. The confidence model reflects implemented control coverage and evidence accessibility; it does not convert the presence of a proof-pack artifact into a passing runtime result.

The latest committed proof pack is dated `2026-03-01`. It centralizes CI, auth, restore-drill, game-day, and supply-chain evidence, but the captured restore and game-day runs both record `pass: false`. Those negative results are intentionally retained.

## Evidence packaging status (2026-03-01)

All evidence links below are centralized in `evidence/proof-pack/2026-03-01/`.

| Domain | Evidence | Observed state |
|---|---|---|
| CI traceability | `ci_run.txt` | Pinned successful **Artifact Guard** run for the recorded commit. |
| Restore drill | `restore_drill_summary.json` and `drills/restore/drill_summary.json` | Executed; captured result is **failed** (`pass: false`). |
| Game-day drill | `game_day_report.json` and `drills/game-day/game_day_report.json` | Executed; captured result is **failed** (`pass: false`). |
| Supply chain | `sbom/` and `signing/` | Evidence locations packaged; complete final generated evidence set is not present in this dated pack. |
| Auth policy | `auth_policy_matrix.md` | Packaged policy matrix and production fail-fast constraints. |

## Evidence still required for a stronger readiness claim

- Successful, repeated strict-mode restore/replay runs with machine-verifiable outputs.
- A game-day/failure-recovery exercise that satisfies its explicit pass criteria and records actionable recovery evidence.
- A complete retained SBOM/vulnerability/signing evidence set for the relevant release candidate.
- Defined and measured SLO/burn-rate thresholds tied to an exercised incident-response path.
- Verification that repository/release protections actually require the intended governance checks.

## Explicitly out of scope for this confidence snapshot

- No claim that the `2026-03-01` restore or game-day exercise passed.
- No claim that GitHub branch/release protection settings are configured merely because workflow files exist.
- No new operational SLO target or burn-rate threshold is inferred from repository structure.
- No production-ready or production-proven claim is made.

## Current production blockers

1. **Restore/replay success evidence** — the retained dated drill result is a failure and does not establish successful recovery.
2. **Failure-recovery/game-day success evidence** — the retained dated scenario did not meet its pass criteria.
3. **SLO and incident-response evidence** — sustained operational objectives and exercised response remain unproven here.
4. **Durability/failover depth** — broader failure-mode validation is still needed beyond the captured scenarios.
5. **Supply-chain dossier completeness** — the dated proof pack does not contain the full generated SBOM/signing evidence set.
6. **Governance rollout enforcement** — required-check/protection configuration must be verified separately from workflow definitions.

See `CONFIDENCE.yaml` for artifact-level scoring and machine-readable evidence mapping, and `READINESS.md` for evergreen readiness gates.
