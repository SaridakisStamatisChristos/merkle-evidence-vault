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
Conclusion: <conclusion>
Started At (UTC): <YYYY-MM-DDTHH:MM:SSZ>
Completed At (UTC): <YYYY-MM-DDTHH:MM:SSZ>
Run Commit SHA Observed: <head_sha>

Status: template pending pinning (set to: "pinned (run URL + metadata captured)" once filled).
EOT
fi

python - <<'PY' "$DATE" "$PACK_DIR" "$RESTORE_SRC_DIR" "$GAME_SRC_DIR"
import json
import re
import sys
from pathlib import Path


date, pack_dir, restore_src, game_src = sys.argv[1:]
pack = Path(pack_dir)

restore_rel = Path("drills/restore/drill_summary.json")
game_rel = Path("drills/game-day/game_day_report.json")


def observed_result(path: Path):
    if not path.exists():
        return "pending", None
    try:
        payload = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError):
        return "executed", None
    passed = payload.get("pass")
    return "executed", passed if isinstance(passed, bool) else None


def result_label(status: str, passed):
    if status == "pending":
        return "pending"
    if passed is True:
        return "passed"
    if passed is False:
        return "failed"
    return "unknown"


restore_status, restore_pass = observed_result(pack / restore_rel)
game_status, game_pass = observed_result(pack / game_rel)

restore_summary = {
    "date": date,
    "scenario": "backup -> wipe -> restore -> replay verify",
    "status": restore_status,
    "result": result_label(restore_status, restore_pass),
    "artifact_source": f"{restore_src}/drill_summary.json" if restore_status == "executed" else "",
    "artifact_copied": restore_rel.as_posix() if restore_status == "executed" else "",
}
(pack / "restore_drill_summary.json").write_text(
    json.dumps(restore_summary, indent=2) + "\n", encoding="utf-8"
)

game_summary = {
    "date": date,
    "scenario": "merkle-engine down + recovery",
    "status": game_status,
    "result": result_label(game_status, game_pass),
    "artifact_source": f"{game_src}/game_day_report.json" if game_status == "executed" else "",
    "artifact_copied": game_rel.as_posix() if game_status == "executed" else "",
}
(pack / "game_day_report.json").write_text(
    json.dumps(game_summary, indent=2) + "\n", encoding="utf-8"
)

ci_text = (pack / "ci_run.txt").read_text(encoding="utf-8") if (pack / "ci_run.txt").exists() else ""
ci_pinned = bool(re.search(r"actions/runs/\d+", ci_text))
ci_success = bool(re.search(r"^Conclusion:\s*success\s*$", ci_text, flags=re.M | re.I))


def replace_status_line(text: str, prefix: str, replacement: str) -> str:
    pattern = rf"^- [✅📦⚠️❌] {re.escape(prefix)}:.*$"
    if re.search(pattern, text, flags=re.M):
        return re.sub(pattern, replacement, text, flags=re.M)
    return text


def drill_line(label: str, status: str, passed, artifact: str) -> str:
    if status == "pending":
        return f"- 📦 {label}: packaged, pending an executed result (artifact: `{artifact}`)."
    if passed is True:
        return f"- ✅ {label}: executed and passed (artifact: `{artifact}`)."
    if passed is False:
        return f"- ⚠️ {label}: executed, but the captured result failed (artifact: `{artifact}`)."
    return f"- ⚠️ {label}: executed, but no boolean pass result was recorded (artifact: `{artifact}`)."


readme = Path("README.md")
if readme.exists():
    text = readme.read_text(encoding="utf-8")
    text = re.sub(
        r"### Latest repository proof pack: [0-9]{4}-[0-9]{2}-[0-9]{2}",
        f"### Latest repository proof pack: {date}",
        text,
    )
    text = re.sub(
        r"## Proof Pack \(Production Evidence Packaging, Last verified: [0-9]{4}-[0-9]{2}-[0-9]{2}\)",
        f"### Latest repository proof pack: {date}",
        text,
    )

    if ci_pinned and ci_success:
        ci_line = f"- ✅ CI trace: a successful workflow run is pinned in `evidence/proof-pack/{date}/ci_run.txt`."
    elif ci_pinned:
        ci_line = f"- ⚠️ CI trace: a workflow run is pinned, but its conclusion is not recorded as success in `evidence/proof-pack/{date}/ci_run.txt`."
    else:
        ci_line = f"- 📦 CI trace: proof-pack metadata exists, but a successful run URL is not yet pinned in `evidence/proof-pack/{date}/ci_run.txt`."

    text = replace_status_line(text, "CI trace", ci_line)
    text = replace_status_line(
        text,
        "Restore drill",
        drill_line(
            "Restore drill",
            restore_status,
            restore_pass,
            f"evidence/proof-pack/{date}/drills/restore/drill_summary.json",
        ),
    )
    text = replace_status_line(
        text,
        "Game-day drill",
        drill_line(
            "Game-day drill",
            game_status,
            game_pass,
            f"evidence/proof-pack/{date}/drills/game-day/game_day_report.json",
        ),
    )
    readme.write_text(text, encoding="utf-8")

print(
    json.dumps(
        {
            "proof_pack": str(pack),
            "ci_pinned": ci_pinned,
            "ci_success": ci_success,
            "restore": {"status": restore_status, "result": result_label(restore_status, restore_pass)},
            "game_day": {"status": game_status, "result": result_label(game_status, game_pass)},
        },
        sort_keys=True,
    )
)
PY

echo "proof-pack prepared at $PACK_DIR"
