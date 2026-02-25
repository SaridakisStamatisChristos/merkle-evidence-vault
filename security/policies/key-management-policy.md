# Signing Key Management Policy

**Version:** 1.0
**SOC2 Controls:** CC6.1, PI1.4
**Related ADR:** ADR-002

## Signing Key Specification

- Algorithm: Ed25519 (RFC 8032)
- Key size: 32-byte seed → 32-byte verifying key
- Representation: hex-encoded, 64 characters

## Key Lifecycle

### Generation
```bash
# Generate with OpenSSL (FIPS-compatible path):
openssl genpkey -algorithm ed25519 -outform DER \
  | xxd -p -c 32 | tail -1
# Store in K8s Secret immediately; never write to disk.
```

### Storage
- Stored as Kubernetes Secret `vault-signing-key` in namespace `vault`
- Access restricted to `vault-api` and `checkpoint-svc` service accounts
- Rotation requires explicit continuity proof ceremony (see below)
- **DO NOT** store in git, CI environment variables, or log files

### Rotation Ceremony
1. Generate new key (above procedure)
2. Run `verifier-cli consistency-proof --old-key <old.pub> --new-key <new.pub>`
   to produce a transition proof
3. Publish transition proof to `vault.checkpoints` Redpanda topic
4. Update K8s Secret → rolling restart of checkpoint-svc
5. Old key may be archived (not deleted) after 90 days

### Compromise Response
1. Immediately rotate key (above ceremony)
2. All historical STHs signed by compromised key must be
   re-anchored via RFC 3161 TSA (out of scope v0.1)
3. Notify all bundle holders to re-verify against new checkpoints
4. File security incident report within 24 hours

## HSM Roadmap
HSM integration targeted for v0.2.
Until then, K8s Secret is the sole protection mechanism.
This is an accepted risk documented in the threat model (R3).
