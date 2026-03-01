# AGENTS.md — Merkle Evidence Vault

## Default workflow (one command)
```bash
make proof-pack-run
```

## What it does
- Runs full test suite (unit + integration/e2e).
- Runs restore drill and game-day drill.
- Creates/updates a dated proof-pack folder under `evidence/proof-pack/<YYYY-MM-DD>/`.
- Copies the latest drill artifacts into proof-pack subfolders.
- Updates proof-pack summary JSON files to reference copied artifacts.
- Updates README/CONFIDENCE statuses from 📦 to ✅ only when executed artifacts exist.
