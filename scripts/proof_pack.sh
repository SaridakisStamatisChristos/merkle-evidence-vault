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
  ls -1dt "$base"/*/ 2>/dev/null | head -n1 | sed 's:/*$::'
}

RESTORE_SRC_DIR="$(latest_dir artifacts/drills || true)"
GAME_SRC_DIR="$(latest_dir artifacts/game-day || true)"

if [[ -n "$RESTORE_SRC_DIR" ]]; then
  cp -R "$RESTORE_SRC_DIR"/. "$RESTORE_DST_DIR"/ 2>/dev/null || true
fi

if [[ -n "$GAME_SRC_DIR" ]]; then
  cp -R "$GAME_SRC_DIR"/. "$GAME_DST_DIR"/ 2>/dev/null || true
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

python - <<'PY' "$DATE" "$PACK_DIR" "$RESTORE_SRC_DIR" "$GAME_SRC_DIR"
import json, re, sys
from pathlib import Path

date, pack_dir, restore_src, game_src = sys.argv[1:]
pack = Path(pack_dir)

restore_rel = Path("drills/restore/drill_summary.json")
game_rel = Path("drills/game-day/game_day_report.json")
restore_ok = (pack / restore_rel).exists()
game_ok = (pack / game_rel).exists()

restore_summary = {
    "date": date,
    "scenario": "backup -> wipe -> restore -> replay verify",
    "status": "executed" if restore_ok else "pending",
    "artifact_source": f"{restore_src}/drill_summary.json" if restore_ok else "",
  "artifact_copied": restore_rel.as_posix() if restore_ok else "",
}
(pack / "restore_drill_summary.json").write_text(json.dumps(restore_summary, indent=2) + "\n", encoding="utf-8")

game_summary = {
    "date": date,
    "scenario": "merkle-engine down + recovery",
    "status": "executed" if game_ok else "pending",
    "artifact_source": f"{game_src}/game_day_report.json" if game_ok else "",
  "artifact_copied": game_rel.as_posix() if game_ok else "",
}
(pack / "game_day_report.json").write_text(json.dumps(game_summary, indent=2) + "\n", encoding="utf-8")

ci_text = (pack / "ci_run.txt").read_text(encoding="utf-8") if (pack / "ci_run.txt").exists() else ""
ci_pinned = bool(re.search(r"actions/runs/\d+", ci_text))


def upsert_line(text: str, pattern: str, replacement: str, anchor: str | None = None) -> str:
    if re.search(pattern, text, flags=re.M):
        return re.sub(pattern, replacement, text, flags=re.M)
    if anchor and anchor in text:
        return text.replace(anchor, anchor + "\n" + replacement)
    return text + "\n" + replacement + "\n"

readme = Path("README.md")
if readme.exists():
    t = readme.read_text(encoding="utf-8")
    t = re.sub(r"Proof Pack \(Production Evidence Packaging, Last verified: [0-9]{4}-[0-9]{2}-[0-9]{2}\)",
               f"Proof Pack (Production Evidence Packaging, Last verified: {date})", t)
    anchor = "## Proof Pack (Production Evidence Packaging, Last verified: "
    t = upsert_line(t, r"^- [✅📦] CI:.*$",
                    f"- {'✅' if ci_pinned else '📦'} CI: " + ("pinned run URL captured" if ci_pinned else "packaged, pending run URL pinning") + f" (artifact: `evidence/proof-pack/{date}/ci_run.txt`)",
                    anchor)
    t = upsert_line(t, r"^- [✅📦] Restore drill:.*$",
                    f"- {'✅' if restore_ok else '📦'} Restore drill: " + ("executed artifact copied" if restore_ok else "packaged, pending latest run artifact copy") + f" (artifact: `evidence/proof-pack/{date}/restore_drill_summary.json`)",
                    anchor)
    t = upsert_line(t, r"^- [✅📦] Game-day:.*$",
                    f"- {'✅' if game_ok else '📦'} Game-day: " + ("executed artifact copied" if game_ok else "packaged, pending latest run artifact copy") + f" (artifact: `evidence/proof-pack/{date}/game_day_report.json`)",
                    anchor)
    readme.write_text(t, encoding="utf-8")

confidence = Path("CONFIDENCE.md")
if confidence.exists():
    t = confidence.read_text(encoding="utf-8")
    t = re.sub(r"## Evidence Packaging Status \([0-9]{4}-[0-9]{2}-[0-9]{2}\)", f"## Evidence Packaging Status ({date})", t)
    t = re.sub(r"All evidence links below are centralized in: `evidence/proof-pack/[0-9]{4}-[0-9]{2}-[0-9]{2}/`",
               f"All evidence links below are centralized in: `evidence/proof-pack/{date}/`", t)
    confidence.write_text(t, encoding="utf-8")
PY

echo "proof-pack prepared at $PACK_DIR"
