# Evidence Directory

The `evidence/` tree is a durable product and assurance surface. It is intentionally separate from ordinary build logs.

## Structure

- `ADR-*.md` — architecture decisions that define durable design choices.
- `architecture.md` — current architecture description.
- `runbooks/` — operational recovery and security procedures.
- `proof-pack/<YYYY-MM-DD>/` — dated, point-in-time evidence packages assembled from executed checks and operational artifacts.

Raw drill outputs may also live under top-level `artifacts/drills/` and `artifacts/game-day/`. A proof pack can copy selected raw outputs so the evidence snapshot is self-contained.

## Status semantics

Use these concepts consistently:

- **pending** — the evidence item has not been captured for the pack.
- **executed** — a run artifact exists. This says nothing by itself about success.
- **passed** — the executed artifact's explicit result satisfies its pass criteria.
- **failed** — the executed artifact's explicit result does not satisfy its pass criteria.
- **pinned** — the pack records an immutable workflow/run reference and associated metadata.

For drill artifacts containing a boolean `pass` field, that field is authoritative for the captured run. Do not infer success from file presence, copying, workflow completion, or a green packaging job.

## Preservation rules

1. Preserve failed and partial evidence; negative results are part of the audit trail.
2. Prefer a new dated proof pack for new execution evidence rather than rewriting old raw artifacts.
3. Do not edit historical artifacts to make a current claim look stronger.
4. Keep readiness claims in `READINESS.md` and confidence interpretation in `CONFIDENCE.md` / `CONFIDENCE.yaml`; proof packs should remain factual evidence snapshots.
5. Recompute any integrity manifest only when its covered content legitimately changes.
6. Do not include secrets, private keys, bearer tokens, or unredacted sensitive production data in committed evidence.

## Latest committed pack

The latest pack currently committed is `proof-pack/2026-03-01/`. Its CI trace is pinned, while its captured restore and game-day result artifacts both record `pass: false`. Those negative results are intentionally preserved.
