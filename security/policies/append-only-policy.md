# Append-Only Storage Policy

**Version:** 1.0
**Effective:** 2026-02-24
**SOC2 Controls:** CC9.1, PI1.4

## Statement

No evidence record, Merkle tree leaf, audit log entry, or
Signed Tree Head shall be modified or deleted after creation,
by any actor or process, under any circumstance.

## Technical Enforcement

1. **Database layer:** PostgreSQL triggers deny `UPDATE` and `DELETE`
   on `evidence`, `tree_leaves`, `audit_log`. Verified by integration
   tests `TestAuditLog_NoDeletePossible` and
   `TestEvidenceTable_NoDeletePossible`.

2. **Application layer:** vault-api emits no `UPDATE` or `DELETE`
   SQL for the above tables. Enforced via code review checklist.

3. **Role layer:** `vault_ingester` and `vault_auditor` Postgres roles
   hold no `DELETE` or `UPDATE` privileges. Only `vault_api` may
   `INSERT`. `vault_api` is constrained by trigger.

4. **Merkle log:** The Rust merkle-engine exposes only `append_leaf`.
   No mutation API exists at the RPC layer.

## Exceptions

None. There are no exceptions to this policy.
Any request to modify or delete evidence must be
refused and escalated to the Security Lead.

## Audit

Compliance verified quarterly by:
- Running `make test` integration suite
- Reviewing Postgres `pg_catalog.pg_trigger` for trigger presence
- Reviewing Postgres `information_schema.role_table_grants`
