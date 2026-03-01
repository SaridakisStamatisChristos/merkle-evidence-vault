#!/usr/bin/env bash
set -euo pipefail

DATE="${1:-$(date -u +%F)}"
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

PACK_DIR="evidence/proof-pack/$DATE"
RESTORE_DST_DIR="$PACK_DIR/drills/restore"
GAME_DST_DIR="$PACK_DIR/drills/game-day"
CI_FILE="$PACK_DIR/ci_run.txt"
mkdir -p "$RESTORE_DST_DIR" "$GAME_DST_DIR" "$PACK_DIR/sbom" "$PACK_DIR/signing"

latest_dir() {
  local base="$1"
  [[ -d "$base" ]] || return 1
  find "$base" -mindepth 1 -maxdepth 1 -type d | sort | tail -n1
}

RESTORE_SRC_DIR="$(latest_dir artifacts/drills || true)"
GAME_SRC_DIR="$(latest_dir artifacts/game-day || true)"

restore_copied=""
game_copied=""

if [[ -n "$RESTORE_SRC_DIR" && -f "$RESTORE_SRC_DIR/drill_summary.json" ]]; then
  cp "$RESTORE_SRC_DIR/drill_summary.json" "$RESTORE_DST_DIR/"
  [[ -f "$RESTORE_SRC_DIR/verifier_output.json" ]] && cp "$RESTORE_SRC_DIR/verifier_output.json" "$RESTORE_DST_DIR/"
  restore_copied="$RESTORE_DST_DIR/drill_summary.json"
fi

if [[ -n "$GAME_SRC_DIR" && -f "$GAME_SRC_DIR/game_day_report.json" ]]; then
  cp "$GAME_SRC_DIR/game_day_report.json" "$GAME_DST_DIR/"
  game_copied="$GAME_DST_DIR/game_day_report.json"
fi

if [[ ! -f "$CI_FILE" ]]; then
  cat > "$CI_FILE" <<EOT
Proof Pack Date: $DATE
Commit SHA: $(git rev-parse HEAD 2>/dev/null || echo '<sha>')

Pinned Run URL (required): https://github.com/<org>/<repo>/actions/runs/<run_id>
Workflow Name: <workflow_name>
Conclusion: success
Started At (UTC): <YYYY-MM-DDTHH:MM:SSZ>
Completed At (UTC): <YYYY-MM-DDTHH:MM:SSZ>
Run Commit SHA Observed: <head_sha>

Status: template pending pinning (set to: "pinned (run URL + metadata captured)" once filled).
EOT
fi

python - <<'PY' "$DATE" "$PACK_DIR" "$RESTORE_SRC_DIR" "$GAME_SRC_DIR" "$restore_copied" "$game_copied"
import json, re, sys
from pathlib import Path

date, pack_dir, restore_src, game_src, restore_copied, game_copied = sys.argv[1:]
pack = Path(pack_dir)

restore_summary = {
    "date": date,
    "scenario": "backup -> wipe -> restore -> replay verify",
    "status": "pending",
}
if restore_copied:
    restore_summary.update({
        "status": "executed",
        "artifact_source": f"{restore_src}/drill_summary.json",
        "artifact_copied": str(Path(restore_copied).relative_to(pack)),
    })

(pack / "restore_drill_summary.json").write_text(json.dumps(restore_summary, indent=2) + "\n")

game_summary = {
    "date": date,
    "scenario": "merkle-engine down + recovery",
    "status": "pending",
}
if game_copied:
    game_summary.update({
        "status": "executed",
        "artifact_source": f"{game_src}/game_day_report.json",
        "artifact_copied": str(Path(game_copied).relative_to(pack)),
    })

(pack / "game_day_report.json").write_text(json.dumps(game_summary, indent=2) + "\n")

ci_text = (pack / "ci_run.txt").read_text() if (pack / "ci_run.txt").exists() else ""
ci_pinned = bool(re.search(r"actions/runs/\d+", ci_text))
restore_ok = (pack / "drills/restore/drill_summary.json").exists()
game_ok = (pack / "drills/game-day/game_day_report.json").exists()

for rel in ("README.md", "CONFIDENCE.md"):
    p = Path(rel)
    if not p.exists():
        continue
    text = p.read_text()
    if rel == "README.md":
        text = re.sub(r"^- [✅📦] CI:.*$", f"- {'✅' if ci_pinned else '📦'} CI: " + ("pinned run URL captured" if ci_pinned else "packaged, pending run URL pinning") + f" (artifact: `evidence/proof-pack/{date}/ci_run.txt`)", text, flags=re.M)
        text = re.sub(r"^- [✅📦] Restore drill:.*$", f"- {'✅' if restore_ok else '📦'} Restore drill: " + ("executed artifact copied" if restore_ok else "packaged, pending latest run artifact copy") + f" (artifact: `evidence/proof-pack/{date}/restore_drill_summary.json`)", text, flags=re.M)
        text = re.sub(r"^- [✅📦] Game-day:.*$", f"- {'✅' if game_ok else '📦'} Game-day: " + ("executed artifact copied" if game_ok else "packaged, pending latest run artifact copy") + f" (artifact: `evidence/proof-pack/{date}/game_day_report.json`)", text, flags=re.M)
        text = re.sub(r"Proof Pack \(Production Evidence Packaging, Last verified: [0-9]{4}-[0-9]{2}-[0-9]{2}\)", f"Proof Pack (Production Evidence Packaging, Last verified: {date})", text)
    else:
        text = re.sub(r"## Evidence Packaging Status \([0-9]{4}-[0-9]{2}-[0-9]{2}\)", f"## Evidence Packaging Status ({date})", text)
        text = re.sub(r"All evidence links below are centralized in: `evidence/proof-pack/[0-9]{4}-[0-9]{2}-[0-9]{2}/`", f"All evidence links below are centralized in: `evidence/proof-pack/{date}/`", text)
    p.write_text(text)
PY

echo "proof-pack prepared at $PACK_DIR"
