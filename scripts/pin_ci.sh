#!/usr/bin/env bash
set -euo pipefail

SHA="${1:-$(git rev-parse HEAD)}"
DATE="${2:-$(date -u +%F)}"
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

PACK_DIR="evidence/proof-pack/$DATE"
CI_FILE="$PACK_DIR/ci_run.txt"
mkdir -p "$PACK_DIR"

write_pending() {
  cat > "$CI_FILE" <<EOT
Proof Pack Date: $DATE
Commit SHA: $SHA

Pinned Run URL (required): https://github.com/<org>/<repo>/actions/runs/<run_id>
Workflow Name: <workflow_name>
Conclusion: success
Started At (UTC): <YYYY-MM-DDTHH:MM:SSZ>
Completed At (UTC): <YYYY-MM-DDTHH:MM:SSZ>
Run Commit SHA Observed: <head_sha>

Status: template pending pinning (set to: "pinned (run URL + metadata captured)" once filled).
EOT
}

if ! command -v gh >/dev/null 2>&1; then
  write_pending
  echo "gh CLI not found; leaving ci_run.txt as pending template"
  exit 0
fi

if ! gh auth status >/dev/null 2>&1; then
  write_pending
  echo "gh CLI not authenticated; leaving ci_run.txt as pending template"
  exit 0
fi

run_json="$(gh run list --commit "$SHA" --json databaseId,url,headSha,conclusion,name,createdAt,updatedAt -L 50 2>/dev/null || true)"
if [[ -z "$run_json" || "$run_json" == "[]" ]]; then
  write_pending
  echo "No runs found for SHA; leaving ci_run.txt as pending template"
  exit 0
fi

python - <<'PY' "$run_json" "$CI_FILE" "$DATE" "$SHA"
import json, re, sys
runs = json.loads(sys.argv[1])
ci_file, date, sha = sys.argv[2:]
success = next((r for r in runs if r.get("conclusion") == "success" and re.search(r"/actions/runs/\d+", r.get("url",""))), None)
if not success:
    open(ci_file, "w").write(f'''Proof Pack Date: {date}
Commit SHA: {sha}

Pinned Run URL (required): https://github.com/<org>/<repo>/actions/runs/<run_id>
Workflow Name: <workflow_name>
Conclusion: success
Started At (UTC): <YYYY-MM-DDTHH:MM:SSZ>
Completed At (UTC): <YYYY-MM-DDTHH:MM:SSZ>
Run Commit SHA Observed: <head_sha>

Status: template pending pinning (set to: "pinned (run URL + metadata captured)" once filled).
''')
    sys.exit(0)
open(ci_file, "w").write(f'''Proof Pack Date: {date}
Commit SHA: {sha}

Pinned Run URL (required): {success.get("url","")}
Workflow Name: {success.get("name","")}
Conclusion: {success.get("conclusion","")}
Started At (UTC): {success.get("createdAt","")}
Completed At (UTC): {success.get("updatedAt","")}
Run Commit SHA Observed: {success.get("headSha","")}

Status: pinned (run URL + metadata captured)
''')
PY

echo "CI run info written to $CI_FILE"
