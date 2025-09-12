-- 0002_content_addressed_blobs.sql
-- Add content-addressed blob metadata and hashing triggers for RulePacks

-- Columns for rulepacks
ALTER TABLE rulepacks
  ADD COLUMN IF NOT EXISTS yaml_hash TEXT,
  ADD COLUMN IF NOT EXISTS yaml_blob_url TEXT,
  ADD COLUMN IF NOT EXISTS storage_backend TEXT NOT NULL DEFAULT 'db'; -- db|s3|gcs

-- Columns for rulepack_versions
ALTER TABLE rulepack_versions
  ADD COLUMN IF NOT EXISTS yaml_hash TEXT,
  ADD COLUMN IF NOT EXISTS dsl_hash TEXT,
  ADD COLUMN IF NOT EXISTS yaml_blob_url TEXT,
  ADD COLUMN IF NOT EXISTS dsl_blob_url TEXT,
  ADD COLUMN IF NOT EXISTS storage_backend TEXT NOT NULL DEFAULT 'db';

-- Helper: hex sha256 of TEXT
CREATE OR REPLACE FUNCTION hex_sha256(txt TEXT)
RETURNS TEXT AS $$
BEGIN
  IF txt IS NULL THEN RETURN NULL; END IF;
  RETURN encode(digest(convert_to(txt,'UTF8'),'sha256'),'hex');
END; $$ LANGUAGE plpgsql;

-- Triggers to auto-hash on change
CREATE OR REPLACE FUNCTION trg_rulepacks_hash()
RETURNS trigger AS $$
BEGIN
  IF TG_OP = 'INSERT' OR NEW.yaml_content IS DISTINCT FROM OLD.yaml_content THEN
    NEW.yaml_hash := hex_sha256(NEW.yaml_content);
    NEW.content_size_bytes := COALESCE(length(NEW.yaml_content),0);
  END IF;
  RETURN NEW;
END; $$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS rulepacks_hash_trg ON rulepacks;
CREATE TRIGGER rulepacks_hash_trg
BEFORE INSERT OR UPDATE ON rulepacks
FOR EACH ROW EXECUTE PROCEDURE trg_rulepacks_hash();

CREATE OR REPLACE FUNCTION trg_rulepack_versions_hash()
RETURNS trigger AS $$
BEGIN
  IF TG_OP = 'INSERT' OR NEW.yaml_content IS DISTINCT FROM OLD.yaml_content THEN
    NEW.yaml_hash := hex_sha256(NEW.yaml_content);
  END IF;
  IF TG_OP = 'INSERT' OR NEW.dsl IS DISTINCT FROM OLD.dsl THEN
    NEW.dsl_hash := hex_sha256(CASE WHEN NEW.dsl IS NULL THEN NULL ELSE NEW.dsl::text END);
  END IF;
  NEW.content_size_bytes := COALESCE(length(NEW.yaml_content),0);
  RETURN NEW;
END; $$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS rulepack_versions_hash_trg ON rulepack_versions;
CREATE TRIGGER rulepack_versions_hash_trg
BEFORE INSERT OR UPDATE ON rulepack_versions
FOR EACH ROW EXECUTE PROCEDURE trg_rulepack_versions_hash();

