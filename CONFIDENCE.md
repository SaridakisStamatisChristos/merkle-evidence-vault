# CONFIDENCE — Merkle Evidence Vault v0.1.4

System effective confidence: **0.90** (gated pre-production)

## Summary

Confidence increased due to improved auditability and evidence accessibility from consolidated proof-packaging, not due to new executed runtime outcomes.

- **Evidence is now centralized** under a dated proof-pack folder for buyer/evaluator review.
- **Auth startup fail-fast guardrails are active** with environment-policy checks.
- **Restore drill and game-day outputs are represented in the evidence model** with explicit execution-state signaling.
- **Release governance workflows exist** with SBOM generation, vulnerability scanning, and signing verification gates.

The system remains **not yet production-ready**. The remaining gap is now mostly repeated execution evidence quality, not control design absence.

## Evidence Packaging Status (2026-03-01)

All evidence links below are centralized in: `evidence/proof-pack/2026-03-01/`

| Domain | Proven by artifact | Status |
|---|---|---|
| CI execution traceability | `ci_run.txt` with commit SHA pointer | Verified packaging |
| Restore drill evidence model | `restore_drill_summary.json` schema + source drill scripts | Verified packaging |
| Game-day evidence model | `game_day_report.json` schema + source drill scripts | Verified packaging |
| Supply chain evidence collation | `sbom/` and `signing/` evidence directories | Verified packaging |
| Auth policy guardrails | `auth_policy_matrix.md` (AUTH_POLICY matrix + prod fail-fast constraints) | Verified packaging |

## Assumptions still requiring execution proof

- Latest CI run URL in `ci_run.txt` must be pinned to a specific successful workflow run.
- Latest restore drill outputs must be copied from `artifacts/drills/<timestamp>/...` into the proof-pack.
- Latest game-day execution report must be copied into the proof-pack with MTTR fields populated.
- SBOM/signing directories must contain the most recent generated artifacts and verification outputs.

## Explicitly out of scope (this update)

- No new runtime control implementation was introduced.
- No release policy/protection configuration was modified on GitHub.
- No new operational SLO target or burn-rate threshold definitions were added.

## Current production blockers

1. **SLO and incident-response evidence gap**
   - Burn-rate/SLO operations and game-day execution evidence still need refreshed run artifacts.
2. **Sustained restore drill success threshold**
   - Need repeated strict-mode drill success in CI/production-like environments.
3. **Durability/failover depth**
   - More failure-mode validation needed beyond baseline restore path.
4. **Governance rollout enforcement**
   - Ensure release governance workflow is required in branch/release protections.

See `CONFIDENCE.yaml` for artifact-level scoring and machine-readable evidence mapping.
