-- 002_rls.sql
BEGIN;

-- Roles: vault_api, vault_ingester, vault_auditor
DO $$
BEGIN
	IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'vault_api') THEN
		CREATE ROLE vault_api NOINHERIT;
	END IF;
	IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'vault_ingester') THEN
		CREATE ROLE vault_ingester NOINHERIT;
	END IF;
	IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'vault_auditor') THEN
		CREATE ROLE vault_auditor NOINHERIT;
	END IF;
END
$$;

-- Grant minimal privileges (grant will fail if role missing; use IF EXISTS wrapper)
DO $$
BEGIN
	IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'vault_api') THEN
		GRANT SELECT, INSERT ON evidence TO vault_api;
		GRANT INSERT ON tree_leaves TO vault_api;
		GRANT INSERT ON signed_tree_heads TO vault_api;
		GRANT INSERT ON audit_log TO vault_api;
	END IF;
END
$$;

-- Revoke update/delete
REVOKE UPDATE, DELETE ON ALL TABLES IN SCHEMA public FROM PUBLIC;

COMMIT;
