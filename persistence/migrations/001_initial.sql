-- 001_initial.sql
BEGIN;

CREATE TABLE evidence (
  id TEXT PRIMARY KEY,
  content_type TEXT NOT NULL,
  content_hash TEXT NOT NULL UNIQUE,
  payload_ref TEXT,
  labels JSONB,
  ingested_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
  ingested_by TEXT,
  leaf_index BIGINT NULL
);

CREATE TABLE tree_leaves (
  leaf_index BIGINT PRIMARY KEY,
  leaf_hash TEXT NOT NULL,
  inserted_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now()
);

CREATE TABLE signed_tree_heads (
  id BIGSERIAL PRIMARY KEY,
  tree_size BIGINT NOT NULL,
  root_hash TEXT NOT NULL,
  key_id TEXT NOT NULL,
  signature TEXT NOT NULL,
  published_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now()
);

CREATE TABLE audit_log (
  id BIGSERIAL PRIMARY KEY,
  action TEXT NOT NULL,
  resource_id TEXT,
  actor TEXT NOT NULL,
  created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
  metadata JSONB
);

-- Append-only trigger to prevent UPDATE/DELETE on critical tables
CREATE OR REPLACE FUNCTION deny_update_delete() RETURNS trigger AS $$
BEGIN
  IF TG_OP = 'UPDATE' OR TG_OP = 'DELETE' THEN
    RAISE EXCEPTION 'modification not allowed on append-only table';
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER evidence_no_update BEFORE UPDATE OR DELETE ON evidence
  FOR EACH ROW EXECUTE FUNCTION deny_update_delete();
CREATE TRIGGER tree_leaves_no_update BEFORE UPDATE OR DELETE ON tree_leaves
  FOR EACH ROW EXECUTE FUNCTION deny_update_delete();
CREATE TRIGGER audit_log_no_update BEFORE UPDATE OR DELETE ON audit_log
  FOR EACH ROW EXECUTE FUNCTION deny_update_delete();

COMMIT;
