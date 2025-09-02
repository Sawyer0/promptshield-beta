-- Idempotent sync for existing databases already containing core tables.
-- Creates only missing tables/columns/indexes needed by the current backend.

-- 0) Ensure required extensions (idempotent)
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- 1) Ensure rulepack_assignments exists (canonical mapping of rulepacks to scopes)
CREATE TABLE IF NOT EXISTS rulepack_assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    rulepack_id UUID NOT NULL REFERENCES rulepacks(id) ON DELETE CASCADE,
    target_scope VARCHAR(255) NOT NULL,
    priority INTEGER DEFAULT 0,
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, target_scope, rulepack_id)
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_rulepack_assignments_tenant ON rulepack_assignments(tenant_id);
CREATE INDEX IF NOT EXISTS idx_rulepack_assignments_scope ON rulepack_assignments(tenant_id, target_scope);
CREATE INDEX IF NOT EXISTS idx_rulepack_assignments_rulepack ON rulepack_assignments(rulepack_id);

-- 2) Ensure rulepacks has expected columns
ALTER TABLE rulepacks ADD COLUMN IF NOT EXISTS yaml_content TEXT;
ALTER TABLE rulepacks ADD COLUMN IF NOT EXISTS rules JSONB;
ALTER TABLE rulepacks ADD COLUMN IF NOT EXISTS current_version_id UUID;
ALTER TABLE rulepacks ADD COLUMN IF NOT EXISTS is_active BOOLEAN DEFAULT true;
ALTER TABLE rulepacks ADD COLUMN IF NOT EXISTS status TEXT DEFAULT 'active';
ALTER TABLE rulepacks ADD COLUMN IF NOT EXISTS enforcement_mode TEXT DEFAULT 'monitor';
ALTER TABLE rulepacks ADD COLUMN IF NOT EXISTS fail_on_severity TEXT DEFAULT 'HIGH';
ALTER TABLE rulepacks ADD COLUMN IF NOT EXISTS priority INTEGER DEFAULT 100;
ALTER TABLE rulepacks ADD COLUMN IF NOT EXISTS metadata JSONB DEFAULT '{}';

-- 3) Foreign key for current_version_id if both tables exist
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.tables WHERE table_schema='public' AND table_name='rulepacks'
  ) AND EXISTS (
    SELECT 1 FROM information_schema.tables WHERE table_schema='public' AND table_name='rulepack_versions'
  ) THEN
    -- Add FK if not present
    IF NOT EXISTS (
      SELECT 1 FROM information_schema.table_constraints 
      WHERE table_schema='public' AND table_name='rulepacks' AND constraint_name='fk_current_version'
    ) THEN
      BEGIN
        ALTER TABLE rulepacks
          ADD CONSTRAINT fk_current_version FOREIGN KEY (current_version_id)
          REFERENCES rulepack_versions(id);
      EXCEPTION WHEN duplicate_object THEN
        -- ignore
      END;
    END IF;
  END IF;
END$$;

-- 4) Enable RLS on core tables if present (idempotent)
DO $$ BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name='assignments') THEN
    EXECUTE 'ALTER TABLE assignments ENABLE ROW LEVEL SECURITY';
  END IF;
  IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name='audits') THEN
    EXECUTE 'ALTER TABLE audits ENABLE ROW LEVEL SECURITY';
  END IF;
  IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name='rulepacks') THEN
    EXECUTE 'ALTER TABLE rulepacks ENABLE ROW LEVEL SECURITY';
  END IF;
  IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name='rulepack_assignments') THEN
    EXECUTE 'ALTER TABLE rulepack_assignments ENABLE ROW LEVEL SECURITY';
  END IF;
END$$;

-- 5) Ensure audits table has required columns for current backend (idempotent)
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema='public' AND table_name='audits') THEN
    -- Add missing columns used by audit services
    IF NOT EXISTS (
      SELECT 1 FROM information_schema.columns 
      WHERE table_schema='public' AND table_name='audits' AND column_name='actor_type'
    ) THEN
      EXECUTE 'ALTER TABLE audits ADD COLUMN actor_type TEXT';
    END IF;
    IF NOT EXISTS (
      SELECT 1 FROM information_schema.columns 
      WHERE table_schema='public' AND table_name='audits' AND column_name='before_data'
    ) THEN
      EXECUTE 'ALTER TABLE audits ADD COLUMN before_data JSONB';
    END IF;
    IF NOT EXISTS (
      SELECT 1 FROM information_schema.columns 
      WHERE table_schema='public' AND table_name='audits' AND column_name='after_data'
    ) THEN
      EXECUTE 'ALTER TABLE audits ADD COLUMN after_data JSONB';
    END IF;
    IF NOT EXISTS (
      SELECT 1 FROM information_schema.columns 
      WHERE table_schema='public' AND table_name='audits' AND column_name='hash'
    ) THEN
      EXECUTE 'ALTER TABLE audits ADD COLUMN hash TEXT';
    END IF;
    IF NOT EXISTS (
      SELECT 1 FROM information_schema.columns 
      WHERE table_schema='public' AND table_name='audits' AND column_name='prev_hash'
    ) THEN
      EXECUTE 'ALTER TABLE audits ADD COLUMN prev_hash TEXT';
    END IF;
  END IF;
END$$;

