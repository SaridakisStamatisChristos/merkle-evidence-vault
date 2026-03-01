#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

TS="$(date -u +%Y%m%dT%H%M%SZ)"
OUT_DIR="artifacts/drills/$TS"
LOG_DIR="$OUT_DIR/logs"
mkdir -p "$LOG_DIR"

COMPOSE_FILE="ops/docker/docker-compose.yml"
API_URL="${E2E_API_URL:-http://localhost:8080}"
TOKEN="${E2E_INGESTER_TOKEN:-ingester-token-example}"
SEED_COUNT="${DRILL_SEED_COUNT:-8}"
STRICT_MODE="${DRILL_STRICT:-false}"

START_TS="$(date +%s)"
GIT_SHA="$(git rev-parse --short HEAD)"

log(){ echo "[$(date -u +%H:%M:%S)] $*" | tee -a "$LOG_DIR/drill.log"; }

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

api_get(){
  local path="$1"
  curl -sS -w "\n%{http_code}" -H "Authorization: Bearer $TOKEN" "$API_URL/api/v1$path"
}

api_post_evidence(){
  local payload_b64="$1"
  curl -sS -w "\n%{http_code}" -X POST "$API_URL/api/v1/evidence" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $TOKEN" \
    -d "{\"content_type\":\"text/plain\",\"payload\":\"$payload_b64\"}"
}

log "starting restore drill ts=$TS sha=$GIT_SHA"

if ! command -v docker >/dev/null 2>&1; then
  python - <<'PY2' "$OUT_DIR" "$GIT_SHA" "$START_TS"
import json,sys,time,os
out,sha,start=sys.argv[1:]
os.makedirs(out,exist_ok=True)
summary={
  "pass": False,
  "git_sha": sha,
  "started_epoch": int(start),
  "ended_epoch": int(time.time()),
  "duration_seconds": 0,
  "verifier_exit_code": 127,
  "error": "docker not available in environment",
  "artifact_dir": out,
}
json.dump(summary,open(os.path.join(out,'drill_summary.json'),'w'),indent=2)
PY2
  log "docker not available; wrote failure summary only"
  if [[ "$STRICT_MODE" == "true" ]]; then
    exit 127
  fi
  exit 0
fi

docker compose -f "$COMPOSE_FILE" down -v --remove-orphans >"$LOG_DIR/compose_down_initial.log" 2>&1 || true
docker compose -f "$COMPOSE_FILE" up --build -d postgres redis zookeeper kafka merkle-engine vault-api >"$LOG_DIR/compose_up_seed.log" 2>&1

if ! wait_for_health; then
  log "vault-api did not become healthy"
fi

IDS_FILE="$OUT_DIR/seed_ids.txt"
: > "$IDS_FILE"
for i in $(seq 1 "$SEED_COUNT"); do
  payload="$(printf 'drill-evidence-%02d-%s' "$i" "$TS" | base64 | tr -d '\n')"
  resp="$(api_post_evidence "$payload")"
  body="$(echo "$resp" | head -n1)"
  code="$(echo "$resp" | tail -n1)"
  echo "$body" > "$LOG_DIR/seed_${i}.json"
  if [[ "$code" == "202" ]]; then
    id="$(python - <<'PY' "$body"
import json,sys
print(json.loads(sys.argv[1]).get('id',''))
PY
)"
    [[ -n "$id" ]] && echo "$id" >> "$IDS_FILE"
  fi
  sleep 0.1
 done

python - <<'PY' "$API_URL" "$TOKEN" "$IDS_FILE" "$OUT_DIR/before_state.json"
import json,sys,time,urllib.request
api,tok,ids_file,out=sys.argv[1:]
ids=[x.strip() for x in open(ids_file) if x.strip()]
headers={"Authorization":f"Bearer {tok}"}

def get(path):
    req=urllib.request.Request(api+"/api/v1"+path,headers=headers)
    try:
        with urllib.request.urlopen(req,timeout=5) as r:
            return r.getcode(),json.loads(r.read().decode())
    except Exception:
        return 0,{}

for _ in range(60):
    ready=True
    for i in ids:
        c,b=get(f"/evidence/{i}")
        if c!=200 or b.get("leaf_index") is None:
            ready=False
            break
    if ready: break
    time.sleep(1)

cc,cp=get("/checkpoints/latest")
vc,vp=get("/checkpoints/latest/verify")
proofs=[]
for i in ids[:3]:
    pc,pb=get(f"/evidence/{i}/proof")
    proofs.append({"id":i,"http_code":pc,"root":pb.get("root","")})

state={"checkpoint_http":cc,"checkpoint":cp,"checkpoint_verify_http":vc,"checkpoint_verify":vp,"proofs":proofs,"ids":ids}
open(out,"w").write(json.dumps(state,indent=2))
PY

PG_CID="$(docker compose -f "$COMPOSE_FILE" ps -q postgres)"
if [[ -n "$PG_CID" ]]; then
  docker exec -i "$PG_CID" pg_dump -U vault_api -d vault -Fc > "$OUT_DIR/backup.dump" 2>"$LOG_DIR/pg_dump.log" || true
fi

docker compose -f "$COMPOSE_FILE" down -v --remove-orphans >"$LOG_DIR/compose_down_restore.log" 2>&1 || true
docker compose -f "$COMPOSE_FILE" up --build -d postgres redis zookeeper kafka merkle-engine vault-api >"$LOG_DIR/compose_up_restore.log" 2>&1
wait_for_health || true

PG_CID="$(docker compose -f "$COMPOSE_FILE" ps -q postgres)"
if [[ -n "$PG_CID" && -s "$OUT_DIR/backup.dump" ]]; then
  cat "$OUT_DIR/backup.dump" | docker exec -i "$PG_CID" pg_restore -U vault_api -d vault --clean --if-exists >"$LOG_DIR/pg_restore.log" 2>&1 || true
fi

python - <<'PY' "$API_URL" "$TOKEN" "$OUT_DIR/before_state.json" "$OUT_DIR/after_state.json" "$OUT_DIR/drill_input.json"
import json,sys,urllib.request
api,tok,before_file,after_file,drill_file=sys.argv[1:]
before=json.load(open(before_file))
ids=before.get('ids',[])
headers={"Authorization":f"Bearer {tok}"}

def get(path):
    req=urllib.request.Request(api+"/api/v1"+path,headers=headers)
    try:
        with urllib.request.urlopen(req,timeout=5) as r:
            return r.getcode(),json.loads(r.read().decode())
    except Exception:
        return 0,{}

cc,cp=get('/checkpoints/latest')
vc,vp=get('/checkpoints/latest/verify')
proofs=[]
for i in ids[:3]:
    pc,pb=get(f'/evidence/{i}/proof')
    proofs.append({"id":i,"http_code":pc,"root":pb.get('root','')})
after={"checkpoint_http":cc,"checkpoint":cp,"checkpoint_verify_http":vc,"checkpoint_verify":vp,"proofs":proofs}
json.dump(after,open(after_file,'w'),indent=2)

inp={
  "expected_root": before.get("checkpoint",{}).get("root_hash",""),
  "before": {
    "checkpoint": before.get("checkpoint",{}),
    "checkpoint_verify": before.get("checkpoint_verify",{}),
    "proofs": before.get("proofs",[]),
  },
  "after": {
    "checkpoint": after.get("checkpoint",{}),
    "checkpoint_verify": after.get("checkpoint_verify",{}),
    "proofs": after.get("proofs",[]),
  }
}
json.dump(inp,open(drill_file,'w'),indent=2)
PY

set +e
go run ./services/verifier-cli/cmd/verifier --drill-input "$OUT_DIR/drill_input.json" --output "$OUT_DIR/verifier_output.json" >"$LOG_DIR/verifier_stdout.log" 2>"$LOG_DIR/verifier_stderr.log"
VERIFY_EXIT=$?
set -e

END_TS="$(date +%s)"
DURATION=$((END_TS-START_TS))
python - <<'PY' "$OUT_DIR" "$GIT_SHA" "$START_TS" "$END_TS" "$DURATION" "$VERIFY_EXIT"
import json,sys,os
out,sha,start,end,duration,vexit=sys.argv[1:]
ver_path=os.path.join(out,'verifier_output.json')
ver={}
if os.path.exists(ver_path):
    ver=json.load(open(ver_path))
summary={
  "pass": bool(ver.get('pass')),
  "git_sha": sha,
  "started_epoch": int(start),
  "ended_epoch": int(end),
  "duration_seconds": int(duration),
  "verifier_exit_code": int(vexit),
  "artifact_dir": out,
}
json.dump(summary,open(os.path.join(out,'drill_summary.json'),'w'),indent=2)
PY

log "drill complete: $OUT_DIR"
if [[ "$STRICT_MODE" == "true" && "$VERIFY_EXIT" -ne 0 ]]; then
  exit "$VERIFY_EXIT"
fi
exit 0
