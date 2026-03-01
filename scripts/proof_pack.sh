#!/usr/bin/env bash
set -euo pipefail

DATE="${1:-$(date -u +%F)}"
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

PACK_DIR="evidence/proof-pack/$DATE"
RESTORE_DST_DIR="$PACK_DIR/drills/restore"
GAME_DST_DIR="$PACK_DIR/drills/game-day"
mkdir -p "$RESTORE_DST_DIR" "$GAME_DST_DIR" "$PACK_DIR/sbom" "$PACK_DIR/signing"

latest_dir() {
  local base="$1"
  [[ -d "$base" ]] || return 1
  find "$base" -mindepth 1 -maxdepth 1 -type d | sort | tail -n1
}

RESTORE_SRC_DIR="$(latest_dir artifacts/drills || true)"
GAME_SRC_DIR="$(latest_dir artifacts/game-day || true)"

if [[ -n "$RESTORE_SRC_DIR" && -f "$RESTORE_SRC_DIR/drill_summary.json" ]]; then
  cp "$RESTORE_SRC_DIR/drill_summary.json" "$RESTORE_DST_DIR/"
  [[ -f "$RESTORE_SRC_DIR/verifier_output.json" ]] && cp "$RESTORE_SRC_DIR/verifier_output.json" "$RESTORE_DST_DIR/"
fi

if [[ -n "$GAME_SRC_DIR" && -f "$GAME_SRC_DIR/game_day_report.json" ]]; then
  cp "$GAME_SRC_DIR/game_day_report.json" "$GAME_DST_DIR/"
fi

python - <<'PY' "$DATE" "$PACK_DIR" "$RESTORE_SRC_DIR" "$GAME_SRC_DIR"
import json, os, sys
from pathlib import Path

date, pack_dir, restore_src, game_src = sys.argv[1:]
pack = Path(pack_dir)
restore_src_path = Path(restore_src) if restore_src else None
game_src_path = Path(game_src) if game_src else None

restore_dst = pack / "restore_drill_summary.json"
game_dst = pack / "game_day_report.json"

restore_payload = {
  "date": date,
  "scenario": "backup -> wipe -> restore -> replay verify",
  "status": "missing_artifacts",
}
if restore_src_path and (restore_src_path / "drill_summary.json").exists():
  copied = "drills/restore/drill_summary.json"
  restore_payload.update({
    "status": "executed_artifact_copied",
    "copied_from": str(restore_src_path / "drill_summary.json"),
    "copied_artifact": copied,
  })
  try:
    restore_payload["drill_summary"] = json.loads((pack / copied).read_text())
  except Exception:
    pass

restore_dst.write_text(json.dumps(restore_payload, indent=2) + "\n")

game_payload = {
  "date": date,
  "scenario": "merkle-engine down + recovery",
  "status": "missing_artifacts",
}
if game_src_path and (game_src_path / "game_day_report.json").exists():
  copied = "drills/game-day/game_day_report.json"
  game_payload.update({
    "status": "executed_artifact_copied",
    "copied_from": str(game_src_path / "game_day_report.json"),
    "copied_artifact": copied,
  })
  try:
    game_payload["game_day_summary"] = json.loads((pack / copied).read_text())
  except Exception:
    pass

game_dst.write_text(json.dumps(game_payload, indent=2) + "\n")

# README/CONFIDENCE status flip only when executed artifacts exist
ready = (
    (pack / "drills/restore/drill_summary.json").exists()
    and (pack / "drills/game-day/game_day_report.json").exists()
)
if ready:
  for rel in ("README.md", "CONFIDENCE.md"):
    p = Path(rel)
    if not p.exists():
      continue
    text = p.read_text()
    text = text.replace("📦", "✅")
    p.write_text(text)
PY

echo "proof-pack prepared at $PACK_DIR"
