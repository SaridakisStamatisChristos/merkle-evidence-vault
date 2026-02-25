# Data Retention and GDPR Policy

**Version:** 1.0
**Legal basis:** EU GDPR, Article 5(1)(e) — storage limitation
**SOC2 Controls:** CC6.7

## Data Categories

| Category | Retention | Contains PII? | Notes |
|----------|-----------|---------------|-------|
| Evidence payloads (object storage) | Indefinite (append-only) | Possibly | Operator must conduct DPIA if PII present |
| Evidence metadata (Postgres) | Indefinite | `ingested_by` (OIDC subject) | OIDC subject is pseudonymous |
| Audit log | 7 years | `actor` (OIDC subject) | SOC2 requires 1 year minimum |
| Signed Tree Heads | Indefinite | None | Cryptographic material only |
| Redis rate-limit keys | 1 minute (TTL) | None | Ephemeral |
| Redpanda messages | 7 days (default) | `ingested_by` | Configurable per-topic |

## PII Minimisation

1. Evidence payloads are stored by content hash reference;
   the API never echoes payload bytes in responses.
2. OIDC subject (`ingested_by`) is a pseudonymous identifier;
   reverse-mapping to natural persons requires IdP access.
3. Labels are operator-controlled; operators MUST NOT store
   direct identifiers (name, email, SSN) in labels.

## Right to Erasure (GDPR Article 17)

This system is append-only by design. Evidence items and audit
log entries **cannot be deleted**.

Operators deploying this system for processing personal data must:
1. Conduct a Data Protection Impact Assessment (DPIA)
2. Document the legitimate interest or legal obligation requiring
   immutability (e.g. regulatory compliance evidence)
3. Inform data subjects that erasure is technically impossible
   and legally exempted under Article 17(3)(b) or (e)

If erasure is legally required with no exemption, the entire
vault must be decommissioned and replaced.

## Data Processor Responsibilities

Anthropic/operator is the data processor. The vault operator
is the data controller. This policy describes technical controls;
data controller must maintain processing records under GDPR Art. 30.
