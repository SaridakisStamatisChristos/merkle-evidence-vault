-- 002_rls.sql
BEGIN;

-- Roles: vault_api, vault_ingester, vault_auditor
CREATE ROLE vault_api NOINHERIT;
CREATE ROLE vault_ingester NOINHERIT;
CREATE ROLE vault_auditor NOINHERIT;

-- Grant minimal privileges
GRANT SELECT, INSERT ON evidence TO vault_api;
GRANT INSERT ON tree_leaves TO vault_api;
GRANT INSERT ON signed_tree_heads TO vault_api;
GRANT INSERT ON audit_log TO vault_api;

-- Revoke update/delete
REVOKE UPDATE, DELETE ON ALL TABLES IN SCHEMA public FROM PUBLIC;

COMMIT;
