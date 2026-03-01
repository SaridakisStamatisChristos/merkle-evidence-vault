# AGENTS

## Full tests
```bash
make test
make integration-test
```

## Restore drill
```bash
./scripts/drill_restore.sh
```

## Game-day drill (merkle-engine down)
```bash
./scripts/game_day_merkle_down.sh
```

## Artifact locations
- Restore drill output: `artifacts/drills/<timestamp>/`
- Game-day output: `artifacts/game-day/<timestamp>/`
- Proof-pack destination: `evidence/proof-pack/2026-03-01/`

## Evidence copy + proof-pack status update (📦 → ✅)
1. Copy latest drill artifacts into proof-pack:
   - `cp artifacts/drills/<timestamp>/drill_summary.json evidence/proof-pack/2026-03-01/restore_drill_summary.json`
   - `cp artifacts/game-day/<timestamp>/game_day_report.json evidence/proof-pack/2026-03-01/game_day_report.json`
2. Pin CI run URL/commit in `evidence/proof-pack/2026-03-01/ci_run.txt`.
3. Copy newest SBOM/signing outputs into:
   - `evidence/proof-pack/2026-03-01/sbom/`
   - `evidence/proof-pack/2026-03-01/signing/`
4. Flip status markers in `README.md` from `📦` to `✅` only after the latest artifacts are copied and pinned.
