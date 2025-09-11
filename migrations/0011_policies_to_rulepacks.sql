-- Migration: 0011_policies_to_rulepacks.sql
-- Purpose: Eliminate duplicate "policies" concept. Use RulePacks exclusively.
-- - Create rulepack_assignments referencing rulepacks
-- - Migrate data from policy_assignments/policies if present
-- - Drop policies and policy_versions tables and old policy_assignments

BEGIN;

-- 1) Create new rulepack_assignments table if it doesn't exist
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

-- 1b) Ensure violations no longer reference policies
-- Drop policy_id FK/column if present; add rulepack_id if missing
DO $$
BEGIN
    IF to_regclass('public.violations') IS NOT NULL THEN
        -- Drop FK if it exists
        BEGIN
            EXECUTE 'ALTER TABLE violations DROP CONSTRAINT IF EXISTS violations_policy_id_fkey';
        EXCEPTION WHEN undefined_object THEN
            -- ignore
        END;
        -- Drop column if it exists
        IF EXISTS (
            SELECT 1 FROM information_schema.columns 
            WHERE table_schema='public' AND table_name='violations' AND column_name='policy_id'
        ) THEN
            EXECUTE 'ALTER TABLE violations DROP COLUMN policy_id';
        END IF;
        -- Add rulepack_id if missing
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.columns 
            WHERE table_schema='public' AND table_name='violations' AND column_name='rulepack_id'
        ) THEN
            EXECUTE 'ALTER TABLE violations ADD COLUMN rulepack_id UUID REFERENCES rulepacks(id) ON DELETE SET NULL';
        END IF;
    END IF;
END$$;

-- 2) Migrate data from policy_assignments -> rulepack_assignments if both exist
DO $$
BEGIN
    IF to_regclass('public.policy_assignments') IS NOT NULL THEN
        -- If policies table exists with rulepack_id, use it to map policy_id -> rulepack_id
        IF to_regclass('public.policies') IS NOT NULL THEN
            EXECUTE $$
                INSERT INTO rulepack_assignments (id, tenant_id, rulepack_id, target_scope, priority, enabled, created_at, updated_at)
                SELECT pa.id, pa.tenant_id, p.rulepack_id, pa.target_scope, pa.priority, pa.enabled, pa.created_at, pa.updated_at
                FROM policy_assignments pa
                JOIN policies p ON p.id = pa.policy_id
                ON CONFLICT (tenant_id, target_scope, rulepack_id) DO UPDATE SET
                    priority = EXCLUDED.priority,
                    enabled = EXCLUDED.enabled,
                    updated_at = EXCLUDED.updated_at
            $$;
        ELSE
            -- No policies table; assume policy_id already points at rulepacks (best effort)
            EXECUTE $$
                INSERT INTO rulepack_assignments (id, tenant_id, rulepack_id, target_scope, priority, enabled, created_at, updated_at)
                SELECT pa.id, pa.tenant_id, pa.policy_id, pa.target_scope, pa.priority, pa.enabled, pa.created_at, pa.updated_at
                FROM policy_assignments pa
                ON CONFLICT (tenant_id, target_scope, rulepack_id) DO UPDATE SET
                    priority = EXCLUDED.priority,
                    enabled = EXCLUDED.enabled,
                    updated_at = EXCLUDED.updated_at
            $$;
        END IF;
    END IF;
END$$;

-- 3) Drop old tables that duplicate the RulePack concept (if present)
DROP TABLE IF EXISTS policy_versions CASCADE;
DROP TABLE IF EXISTS policies CASCADE;
DROP TABLE IF EXISTS policy_assignments CASCADE;

COMMIT;


