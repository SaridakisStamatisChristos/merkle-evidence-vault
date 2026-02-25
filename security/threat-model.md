# Threat Model — Merkle Evidence Vault v0.1

**Date:** 2026-02-24
**Method:** STRIDE
**Scope:** vault-api, merkle-engine, checkpoint-svc, persistence, frontend
**Reviewed by:** Security Lead, Principal Architect

---

## Trust Boundaries
```
[External Client] ──TLS 1.3──► [vault-api] ──Unix/mTLS──► [merkle-engine]
                                     │
                                     ├──TCP+TLS──► [Postgres]
                                     ├──TCP──────► [Redis]
                                     └──TCP──────► [Redpanda]
```

**In-scope trust boundaries:**
- Internet → vault-api (TLS termination)
- vault-api → merkle-engine (internal gRPC)
- vault-api → Postgres (service account credentials)
- vault-api → Redis (password auth)
- CI/CD → container registry (OIDC Workload Identity)

**Out of scope:** signing key HSM, external TSA anchoring, PKI CA.

---

## STRIDE Analysis

### Spoofing

| ID | Threat | Mitigations | Residual Risk |
|----|--------|-------------|---------------|
| S1 | Attacker presents forged JWT to impersonate privileged user | JWKS validation (lestrrat-go/jwx), issuer + audience check, HTTPS-only | Low — requires OIDC provider compromise |
| S2 | Attacker intercepts gRPC traffic between vault-api and merkle-engine | Unix socket (local) or mTLS for remote; network policy restricts gRPC port | Medium — mTLS not yet enforced in v0.1 (see ADR-002) |
| S3 | DNS spoofing of JWKS endpoint | HTTPS with system cert pool; pin JWKS URL to known issuer | Low |

### Tampering

| ID | Threat | Mitigations | Residual Risk |
|----|--------|-------------|---------------|
| T1 | Attacker modifies evidence in Postgres after ingest | Append-only trigger (deny UPDATE/DELETE); RLS; Postgres WAL archiving | Low — requires DB superuser access |
| T2 | Attacker modifies Merkle leaf hash in tree_leaves | Append-only trigger on tree_leaves | Low |
| T3 | Attacker replays old evidence with modified content hash | Content hash dedup (UNIQUE constraint); leaf_data binds content_type | Low |
| T4 | Attacker tampers .evb bundle in transit | Bundle contains Ed25519-signed STH; verifier-cli rejects signature mismatch | Low |
| T5 | Attacker corrupts bundle on disk | STH signature + per-leaf inclusion proofs; both checked by verifier-cli | Low |
| T6 | MITM injects forged checkpoint | Ed25519 signature verification; attacker cannot forge without private key | Low — **degrades to Medium on key compromise** |

### Repudiation

| ID | Threat | Mitigations | Residual Risk |
|----|--------|-------------|---------------|
| R1 | Actor denies having ingested evidence | Audit log (append-only); records OIDC subject + request ID | Low |
| R2 | Admin denies deleting evidence | Deletes are physically impossible (trigger); WAL records all DDL | Low |
| R3 | Checkpoint signer denies signing a tree head | Ed25519 signature is deterministic; key_id maps to verifying key | Medium — no HSM non-repudiation in v0.1 |

### Information Disclosure

| ID | Threat | Mitigations | Residual Risk |
|----|--------|-------------|---------------|
| I1 | Evidence payload exfiltration via API | Payloads stored in object storage, not DB; API requires authentication | Medium — payload_ref exposed to vault_api service account |
| I2 | Hash oracle: adversary infers content from content_hash | SHA-256 preimage resistance; no rainbow table risk for structured data | Low |
| I3 | Signing key leak via environment variable | K8s Secret; env var injected at runtime; never logged | Medium — secrets in env are process-readable |
| I4 | Frontend XSS: proof data rendered unsanitised | **⚠ NOT YET MITIGATED** — CSP + DOMPurify audit pending | **HIGH — see CONFIDENCE.md** |
| I5 | PII in evidence labels | Labels are arbitrary strings; operators must enforce PII controls | Medium — no label content scanning in v0.1 |

### Denial of Service

| ID | Threat | Mitigations | Residual Risk |
|----|--------|-------------|---------------|
| D1 | Oversized evidence payload exhausts memory | 4 MiB limit enforced at API + engine layer | Low |
| D2 | Ingest flooding overwhelms Redpanda | Rate limit (1000 req/min/subject) + per-topic retention | Medium — rate limit is per-subject, not per-IP |
| D3 | Merkle engine OOM from large tree | K8s memory limit (1 GiB); tree stored as leaf-hash list (~32B/leaf) | Low for ≤ 30M leaves |
| D4 | DB connection pool exhaustion | pgxpool max_open_conns; readiness probe gate | Low |
| D5 | Bundle export of huge range blocks assembler | 100k leaf range cap; async assembly | Medium — assembler is single-threaded in v0.1 |

### Elevation of Privilege

| ID | Threat | Mitigations | Residual Risk |
|----|--------|-------------|---------------|
| E1 | Ingester role escalates to auditor | JWT role claim validated server-side; no client-side privilege check | Low |
| E2 | Container escape via privileged pod | PSS restricted profile; drop ALL caps; non-root user; read-only rootfs | Low |
| E3 | SQL injection in audit log query | pgx parameterised queries throughout; no string interpolation in SQL | Low |
| E4 | Path traversal in bundle filenames | Bundle entry filenames validated against allow-list path prefix | Low |

---

## Risk Register Summary

| Severity | Count | Items |
|----------|-------|-------|
| HIGH     | 1     | I4 (XSS — frontend proof display) |
| MEDIUM   | 5     | S2, R3, I3, I5, D2 |
| LOW      | 17    | All remaining |

---

## Accepted Risks (v0.1)

| ID | Acceptance Rationale |
|----|----------------------|
| S2 | mTLS between vault-api and merkle-engine deferred to v0.2; network policy limits blast radius |
| R3 | HSM integration deferred to v0.2 (ADR-002); private key in K8s Secret |
| I4 | **NOT accepted** — must be remediated before any production deployment |
| I5 | Operator responsibility; document in compliance guide |

---

## Remediation Backlog

| Priority | Item | Owner | Target |
|----------|------|-------|--------|
| P0 | I4: CSP headers + DOMPurify on proof display | Frontend Lead | v0.2 |
| P1 | S2: mTLS for vault-api ↔ merkle-engine | Infra | v0.2 |
| P1 | R3: HSM integration for signing key | Security | v0.2 |
| P2 | I5: Label content scanning (PII detection) | Backend | v0.3 |
| P2 | D2: Per-IP rate limiting in addition to per-subject | Backend | v0.2 |
| P3 | D5: Parallel bundle assembler | Backend | v0.3 |
