# Runbook: Stale Checkpoint (CheckpointStaleness)

**Alert:** CheckpointStaleness
**Severity:** warning
**SOC2:** PI1.4
**Fires when:** No new checkpoint for > 10 minutes (2× default interval)

## Impact

- Evidence ingested since last checkpoint has no signed STH
- Compliance window: no external proof of tree state during gap
- Read/query operations unaffected
- Historical checkpoints remain valid

## Triage
```bash
# 1. Check checkpoint-svc health
kubectl get pods -n vault -l app=checkpoint-svc
kubectl logs -n vault -l app=checkpoint-svc --tail=100

# 2. Query latest checkpoint age directly
psql $DATABASE_URL -c \
  "SELECT id, tree_size, published_at,\n+          NOW() - published_at AS age\n+   FROM signed_tree_heads\n+   ORDER BY tree_size DESC LIMIT 3;"

# 3. Check merkle-engine reachability from checkpoint-svc
kubectl exec -n vault deploy/checkpoint-svc -- \
  nc -zv merkle-engine.vault.svc.cluster.local 9444
```

## Common Causes

### checkpoint-svc CrashLoopBackOff
```bash
kubectl describe pod -n vault -l app=checkpoint-svc
# If OOMKilled or config error: fix + restart
kubectl rollout restart deployment/checkpoint-svc -n vault
```

### merkle-engine Unreachable
- Follow MerkleEngineDown runbook first.
- checkpoint-svc will auto-recover once engine is back.

### Tree Size 0 (first deploy, no ingestion yet)
- checkpoint-svc skips signing when `tree_size == 0`.
- Not a real incident — ingest at least one item and wait one interval.

### DB Write Failure
```bash
kubectl logs -n vault -l app=checkpoint-svc | grep "STH insert failed"
# If Postgres is down or credentials invalid: fix DB connectivity first
```

## Manual Checkpoint Emission

If automatic recovery will take > 10 minutes and a checkpoint is
urgently needed for compliance:
```bash
# Port-forward to merkle-engine gRPC
kubectl port-forward -n vault svc/merkle-engine 9444:9444 &

# Trigger immediate checkpoint via checkpoint-svc restart
# (emits one immediately on startup)
kubectl rollout restart deployment/checkpoint-svc -n vault

# Verify emission
psql $DATABASE_URL -c \
  "SELECT tree_size, root_hash, published_at\n+   FROM signed_tree_heads ORDER BY id DESC LIMIT 1;"
```

## Post-Incident

After gap is resolved, document in incident log:
- Duration of gap
- Tree size range with no checkpoint coverage
- Whether any evidence was ingested during gap
- Whether external auditors need to be notified
