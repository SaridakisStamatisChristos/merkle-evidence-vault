#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

TS="$(date -u +%Y%m%dT%H%M%SZ)"
OUT_DIR="artifacts/game-day/$TS"
LOG_DIR="$OUT_DIR/logs"
mkdir -p "$LOG_DIR"

COMPOSE_FILE="ops/docker/docker-compose.yml"
API_URL="${E2E_API_URL:-http://localhost:8080}"
TOKEN="${E2E_INGESTER_TOKEN:-ingester-token-example}"
LOAD_SECONDS="${GAME_DAY_LOAD_SECONDS:-120}"
DOWN_SECONDS="${GAME_DAY_DOWN_SECONDS:-60}"

log(){ echo "[$(date -u +%H:%M:%S)] $*" | tee -a "$LOG_DIR/game_day.log"; }

wait_for_health() {
  local tries=90
  for ((i=1;i<=tries;i++)); do
    if curl -fsS "$API_URL/healthz" >/dev/null 2>&1; then
      return 0
    fi
    sleep 2
  done
  return 1
}

get_metric_line() {
  local metric="$1"
  curl -fsS "$API_URL/metrics" | awk -v m="$metric" '$1==m {print $2; exit}'
}

api_post_evidence(){
  local payload_b64="$1"
  curl -sS -o /dev/null -w "%{http_code}" -X POST "$API_URL/api/v1/evidence" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $TOKEN" \
    -d "{\"content_type\":\"text/plain\",\"payload\":\"$payload_b64\"}"
}

api_get_proof_code(){
  local id="$1"
  curl -sS -o /dev/null -w "%{http_code}" "$API_URL/api/v1/evidence/$id/proof"
}

start_epoch="$(date +%s)"
git_sha="$(git rev-parse --short HEAD)"

if ! command -v docker >/dev/null 2>&1; then
  python - <<'PY' "$OUT_DIR" "$git_sha" "$start_epoch"
import json,os,sys,time
out,sha,start=sys.argv[1:]
os.makedirs(out,exist_ok=True)
json.dump({
  "pass": False,
  "error": "docker not available in environment",
  "git_sha": sha,
  "started_epoch": int(start),
  "ended_epoch": int(time.time())
}, open(os.path.join(out, "game_day_report.json"), "w"), indent=2)
PY
  exit 0
fi

log "starting game day: merkle-engine outage"
docker compose -f "$COMPOSE_FILE" down -v --remove-orphans >"$LOG_DIR/compose_down_initial.log" 2>&1 || true
docker compose -f "$COMPOSE_FILE" up --build -d postgres redis zookeeper kafka merkle-engine vault-api checkpoint-svc >"$LOG_DIR/compose_up.log" 2>&1
wait_for_health

proof_success_pre=0
proof_fail_during_down=0
proof_success_post=0

start_metric_total="$(get_metric_line vault_api_http_requests_total || echo 0)"
start_err_metric="$(curl -fsS "$API_URL/metrics" | awk '/vault_api_http_requests_operation_total\{operation="proof",status_class="5xx"\}/{print $2; exit}' || echo 0)"

load_end=$(( $(date +%s) + LOAD_SECONDS ))
ids_file="$OUT_DIR/ids.txt"
: > "$ids_file"

log "phase 1: generating normal load for ${LOAD_SECONDS}s"
while [[ $(date +%s) -lt $load_end ]]; do
  payload="$(printf 'game-day-%s-%s' "$TS" "$RANDOM" | base64 | tr -d '\n')"
  ingest_resp="$(curl -sS -w "\n%{http_code}" -X POST "$API_URL/api/v1/evidence" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $TOKEN" \
    -d "{\"content_type\":\"text/plain\",\"payload\":\"$payload\"}")"
  body="$(echo "$ingest_resp" | head -n1)"
  code="$(echo "$ingest_resp" | tail -n1)"
  if [[ "$code" == "202" ]]; then
    id="$(python -c 'import json,sys; print(json.loads(sys.argv[1]).get("id", ""))' "$body" 2>/dev/null || true)"
    if [[ -n "$id" ]]; then
      echo "$id" >> "$ids_file"
      pcode="$(api_get_proof_code "$id")"
      [[ "$pcode" == "200" ]] && proof_success_pre=$((proof_success_pre+1))
    fi
  fi
  sleep 0.2
done

log "phase 2: stopping merkle-engine for ${DOWN_SECONDS}s"
merkle_down_start="$(date +%s)"
docker compose -f "$COMPOSE_FILE" stop merkle-engine >"$LOG_DIR/compose_stop_merkle.log" 2>&1

outage_end=$(( $(date +%s) + DOWN_SECONDS ))
while [[ $(date +%s) -lt $outage_end ]]; do
  if [[ -s "$ids_file" ]]; then
    id="$(tail -n 1 "$ids_file")"
    pcode="$(api_get_proof_code "$id" || echo 000)"
    [[ "$pcode" != "200" ]] && proof_fail_during_down=$((proof_fail_during_down+1))
  fi
  sleep 1
done
merkle_down_end="$(date +%s)"

log "phase 3: restarting merkle-engine and verifying recovery"
docker compose -f "$COMPOSE_FILE" start merkle-engine >"$LOG_DIR/compose_start_merkle.log" 2>&1
sleep 5

for _ in $(seq 1 20); do
  if curl -fsS http://localhost:8000/healthz >/dev/null 2>&1; then
    break
  fi
  sleep 2
done

if [[ -s "$ids_file" ]]; then
  while IFS= read -r id; do
    pcode="$(api_get_proof_code "$id" || echo 000)"
    [[ "$pcode" == "200" ]] && proof_success_post=$((proof_success_post+1))
  done < "$ids_file"
fi

end_metric_total="$(get_metric_line vault_api_http_requests_total || echo 0)"
end_err_metric="$(curl -fsS "$API_URL/metrics" | awk '/vault_api_http_requests_operation_total\{operation="proof",status_class="5xx"\}/{print $2; exit}' || echo 0)"

checkpoint_freshness="$(curl -fsS http://localhost:8081/metrics | awk '$1=="checkpoint_svc_last_sign_success_unixtime" {print $2; exit}' || echo 0)"

downtime_seconds=$((merkle_down_end-merkle_down_start))
requests_delta=$(( ${end_metric_total%.*} - ${start_metric_total%.*} ))
err_delta=$(( ${end_err_metric%.*} - ${start_err_metric%.*} ))

pass=true
reason=""
if [[ $downtime_seconds -lt $DOWN_SECONDS ]]; then
  pass=false
  reason+="downtime shorter than requested;"
fi
if [[ $proof_fail_during_down -lt 1 ]]; then
  pass=false
  reason+="no proof failures observed during outage;"
fi
if [[ $proof_success_post -lt 1 ]]; then
  pass=false
  reason+="no proof recovery observed after restart;"
fi

end_epoch="$(date +%s)"
python - <<'PY' "$OUT_DIR" "$pass" "$reason" "$git_sha" "$start_epoch" "$end_epoch" "$LOAD_SECONDS" "$DOWN_SECONDS" "$downtime_seconds" "$proof_success_pre" "$proof_fail_during_down" "$proof_success_post" "$requests_delta" "$err_delta" "$checkpoint_freshness"
import json,os,sys
(out_dir, passed, reason, sha, start, end, load_s, down_s, down_actual, pre_ok, down_fail, post_ok, req_delta, err_delta, cp_success_ts)=sys.argv[1:]
os.makedirs(out_dir, exist_ok=True)
report={
  "pass": passed.lower()=="true",
  "reason": reason,
  "scenario": "merkle-engine-down",
  "git_sha": sha,
  "started_epoch": int(start),
  "ended_epoch": int(end),
  "duration_seconds": int(end)-int(start),
  "config": {
    "load_seconds": int(load_s),
    "down_seconds": int(down_s)
  },
  "measurements": {
    "merkle_downtime_seconds": int(down_actual),
    "proof_success_before": int(pre_ok),
    "proof_failures_during_outage": int(down_fail),
    "proof_success_after": int(post_ok),
    "vault_requests_delta": int(req_delta),
    "proof_5xx_delta": int(err_delta),
    "checkpoint_last_success_unixtime": int(float(cp_success_ts)) if cp_success_ts else 0
  },
  "artifacts": {
    "log_dir": os.path.join(out_dir, "logs")
  }
}
json.dump(report, open(os.path.join(out_dir, "game_day_report.json"), "w"), indent=2)
PY

log "game day complete: $OUT_DIR"
exit 0
