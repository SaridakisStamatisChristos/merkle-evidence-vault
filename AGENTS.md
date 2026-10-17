# AGENTS.md — Repository Automation Guide

This file documents durable repository automation rules. It is not a build-session handoff.

## Standard verification

Use the smallest command that matches the change, then let pull-request CI run the repository-wide lanes. GitHub Actions must be enabled in repository settings for those CI lanes to start.

```bash
make test
make integration-test
```

For a new dated assurance snapshot:

```bash
make proof-pack-run
```

`make proof-pack-run` runs the test/integration workflow, restore drill, game-day drill, packages the latest artifacts under `evidence/proof-pack/<YYYY-MM-DD>/`, and attempts to pin CI metadata for that date.

## Evidence rules

- Treat proof packs as point-in-time assurance snapshots, not ordinary build logs.
- Preserve failed drill/game-day artifacts; a failed execution is evidence and must not be deleted or relabeled to manufacture a green result.
- Distinguish **captured/executed** from **passed**. The `pass` field in the copied result artifact is authoritative for drill outcome.
- Create a new dated proof pack for materially new execution evidence rather than rewriting old raw artifacts.
- Keep `CONFIDENCE.md` and `CONFIDENCE.yaml` aligned with observed evidence. Do not raise confidence or readiness solely because an artifact exists.
- Do not bypass integrity, security, release-governance, or reproducibility checks to obtain green status.
- Do not create releases or tags as part of proof-pack maintenance.

## Product documentation

- `README.md` explains the product and the role of evidence.
- `READINESS.md` records evergreen readiness gates and current evidence gaps.
- `evidence/README.md` defines the evidence layout and status semantics.
