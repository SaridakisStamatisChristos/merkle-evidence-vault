# Runbook: Signing Key Rotation

**Trigger:** Scheduled rotation, suspected compromise, or personnel change
**Risk:** High — incorrect rotation invalidates all future checkpoint verification
**Approval required:** Security Lead + Principal Architect

## Pre-Rotation Checklist

- [ ] Notify all bundle holders of upcoming key rotation
- [ ] Confirm current tree size and latest STH are stable
- [ ] Schedule maintenance window (checkpoint-svc will be restarted)
- [ ] Prepare new key material (see Key Generation below)
- [ ] Back up current public key hex

## Step 1 — Generate New Key
```bash
# Generate new Ed25519 seed (32 random bytes as hex)
NEW_KEY_HEX=$(openssl rand -hex 32)
echo "New key hex: $NEW_KEY_HEX"

# Derive public key using verifier-cli
NEW_PUB_HEX=$(verifier-cli keygen --seed-hex "$NEW_KEY_HEX" --print-pub)
echo "New public key: $NEW_PUB_HEX"

# Store securely — never echo to logs in production
```

## Step 2 — Record Last Checkpoint Under Old Key
```bash
# Get current STH (signed with old key)
LAST_STH=$(psql $DATABASE_URL -t -A -c \
  "SELECT row_to_json(s) FROM signed_tree_heads s\n+   ORDER BY tree_size DESC LIMIT 1;")
echo "$LAST_STH" > /tmp/last-sth-old-key.json
echo "Last tree_size under old key: $(echo $LAST_STH | jq .tree_size)"
```

## Step 3 — Update Kubernetes Secret
```bash
# Update the secret (triggers rolling restart of dependents)
kubectl create secret generic vault-signing-key \
  --from-literal=hex="$NEW_KEY_HEX" \
  -n vault \
  --dry-run=client -o yaml | kubectl apply -f -
```

## Step 4 — Restart Services
```bash
# Restart in order: checkpoint-svc first, then vault-api
kubectl rollout restart deployment/checkpoint-svc -n vault
kubectl rollout status  deployment/checkpoint-svc -n vault --timeout=120s

kubectl rollout restart deployment/vault-api -n vault
kubectl rollout status  deployment/vault-api -n vault --timeout=120s

kubectl rollout restart deployment/merkle-engine -n vault
kubectl rollout status  deployment/merkle-engine -n vault --timeout=120s
```

## Step 5 — Verify First Checkpoint Under New Key
```bash
# Wait one checkpoint interval (default 5 min), then:
psql $DATABASE_URL -c \
  "SELECT id, tree_size, key_id, published_at\n+   FROM signed_tree_heads ORDER BY id DESC LIMIT 3;"

# Confirm key_id changed (SHA-256 of new verifying key)
EXPECTED_KEY_ID=$(verifier-cli keygen --seed-hex "$NEW_KEY_HEX" --print-key-id)
ACTUAL_KEY_ID=$(psql $DATABASE_URL -t -A -c \
  "SELECT key_id FROM signed_tree_heads ORDER BY id DESC LIMIT 1;")

if [ "$EXPECTED_KEY_ID" = "$ACTUAL_KEY_ID" ]; then
  echo "✓ Key rotation successful: new key_id=$ACTUAL_KEY_ID"
else
  echo "✗ Key ID mismatch — investigate before proceeding"
  exit 1
fi
```

## Step 6 — Publish Continuity Notice
```bash
# Create a human-readable continuity record
cat > /tmp/key-rotation-$(date +%Y%m%d).json <",
  "new_key_id":       "$ACTUAL_KEY_ID",
  "last_tree_size_old_key": $(echo $LAST_STH | jq .tree_size),
  "first_tree_size_new_key": $(psql $DATABASE_URL -t -A -c \
    "SELECT tree_size FROM signed_tree_heads WHERE key_id='$ACTUAL_KEY_ID' \
     ORDER BY tree_size ASC LIMIT 1;"),
  "continuity_proof": "see consistency proof between above sizes",
  "approved_by":      ""
}
EOF

# Store in evidence vault as an ingestion event
curl -sk -X POST https://localhost:8443/api/v1/evidence \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\n+    \"content_type\": \"application/json\",\n+    \"payload\": $(cat /tmp/key-rotation-$(date +%Y%m%d).json | base64),\n+    \"labels\": {\"type\": \"key-rotation\", \"key_id\": \"$ACTUAL_KEY_ID\"}\n+  }"
```

## Step 7 — Notify Bundle Holders

Notify all parties who have previously downloaded `.evb` bundles:
- Bundles signed with the old key remain valid under the old public key
- New bundles will carry the new key_id
- Provide `NEW_PUB_HEX` for verification of new bundles
- Provide `last-sth-old-key.json` for chain-of-custody reference

## Rollback Procedure

If rotation fails or services do not start:
```bash
# Restore old key from secure backup
kubectl create secret generic vault-signing-key \
  --from-literal=hex="$OLD_KEY_HEX" \
  -n vault \
  --dry-run=client -o yaml | kubectl apply -f -

kubectl rollout restart deployment/checkpoint-svc \
  deployment/vault-api deployment/merkle-engine -n vault
```
