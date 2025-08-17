-- PromptShield schema fixes and performance indexes (v0.3)

-- 1. Add missing column to assignments
ALTER TABLE IF EXISTS assignments
  ADD COLUMN IF NOT EXISTS priority INT NOT NULL DEFAULT 0;

-- Ensure unique assignment per (tenant_id, target_scope, rulepack_id)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_indexes 
        WHERE schemaname = current_schema() 
          AND indexname = 'uk_assignments_scope') THEN
        CREATE UNIQUE INDEX uk_assignments_scope
            ON assignments (tenant_id, target_scope, rulepack_id);
    END IF;
END$$;

-- 2. Add missing columns to audits table
ALTER TABLE IF EXISTS audits
  ADD COLUMN IF NOT EXISTS actor_email TEXT,
  ADD COLUMN IF NOT EXISTS metadata    JSONB;

-- 3. Performance indexes -------------------------------------------------

-- Fast latest‐version lookup per rulepack
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_indexes
        WHERE schemaname = current_schema()
          AND indexname = 'idx_rpv_rulepack_version_desc') THEN
        CREATE INDEX idx_rpv_rulepack_version_desc
          ON rulepack_versions (rulepack_id, version DESC);
    END IF;
END$$;

-- Audit queries by tenant
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_indexes
        WHERE schemaname = current_schema()
          AND indexname = 'idx_audits_tenant_ts') THEN
        CREATE INDEX idx_audits_tenant_ts
          ON audits (tenant_id, created_at DESC);
    END IF;
END$$;

-- Audit queries by object reference
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_indexes
        WHERE schemaname = current_schema()
          AND indexname = 'idx_audits_object_ts') THEN
        CREATE INDEX idx_audits_object_ts
          ON audits (object_type, object_id, created_at DESC);
    END IF;
END$$;
