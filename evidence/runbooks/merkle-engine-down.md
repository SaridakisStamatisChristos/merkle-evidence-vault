# Runbook: Merkle Engine Down (MerkleEngineDown)

**Alert:** MerkleEngineDown
**Severity:** critical
**SOC2:** PI1.4

## Impact

- All new evidence ingestion is blocked (pipeline consumer cannot append leaves)
- Proof generation endpoints return errors
- Checkpoint signing halted
- Existing evidence and audit log remain intact and queryable

## Immediate Triage (< 5 min)
```bash
# 1. Check pod state
kubectl get pods -n vault -l app=merkle-engine

# 2. Check recent logs
kubectl logs -n vault -l app=merkle-engine --tail=200 --previous

# 3. Check resource pressure
kubectl top pod -n vault -l app=merkle-engine

# 4. Describe for events (OOMKilled, ImagePullBackOff, etc.)
kubectl describe pod -n vault -l app=merkle-engine
```

## Common Causes and Remediation

### OOMKilled
```bash
# Symptom: kubectl describe shows OOMKilled
# Fix: increase memory limit
kubectl patch deployment merkle-engine -n vault \
  --type=json \
  -p='[{"op":"replace","path":"/spec/template/spec/containers/0/resources/limits/memory","value":"2Gi"}]'
```

### Signing Key Missing
```bash
# Symptom: MERKLE_SIGNING_KEY_HEX must be set in logs
# Fix: verify secret exists
kubectl get secret vault-signing-key -n vault
# If missing: re-create from key management policy procedure
```

### gRPC Bind Failure
```bash
# Symptom: "address already in use" or "permission denied" on :9444
# Fix: force pod restart — Recreate strategy ensures clean restart
kubectl rollout restart deployment/merkle-engine -n vault
kubectl rollout status deployment/merkle-engine -n vault
```

### Image Pull Failure
```bash
# Symptom: ImagePullBackOff or ErrImagePull
# Check: kubectl describe pod ... for exact error
# Fix: verify image tag exists in registry
# Rollback to last known good:
kubectl rollout undo deployment/merkle-engine -n vault
```

## Recovery Verification
```bash
# 1. Wait for pod Ready
kubectl wait --for=condition=Ready pod \
  -l app=merkle-engine -n vault --timeout=120s

# 2. Verify gRPC probe passes
kubectl exec -n vault deploy/vault-api -- \
  grpcurl -plaintext merkle-engine.vault:9444 list

# 3. Verify vault-api readiness (depends on engine)
curl -sk https://localhost:8443/readyz | jq .

# 4. Check pipeline consumer is processing
kubectl logs -n vault -l app=vault-api --tail=50 | grep "evidence committed"
```

## Tree State After Restart

The merkle-engine holds tree state **in-memory**. On restart it must
replay leaf hashes from Postgres `tree_leaves` table. This is automatic
on startup — monitor logs for:
```
INFO merkle-engine: replaying N leaves from persistence
INFO merkle-engine: tree restored tree_size=N root=abcd…
```

If replay takes > 60s for large trees, startup probe will fail.
Increase `failureThreshold` × `periodSeconds` in deployment temporarily.

## Escalation

If not resolved in 20 minutes or if tree replay fails:
1. Page Security Lead (signing key concern) and Backend Lead
2. Consider halting ingestion at load balancer to prevent data loss
3. Do NOT attempt manual DB surgery on `tree_leaves`
