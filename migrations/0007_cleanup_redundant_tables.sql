-- Cleanup redundant policy tables and consolidate into rulepacks
-- RulePacks ARE the policies, no need for separate tables

-- 1. First, let's add the missing columns to rulepacks
ALTER TABLE rulepacks 
    ADD COLUMN IF NOT EXISTS yaml_content TEXT,
    ADD COLUMN IF NOT EXISTS rules JSONB,
    ADD COLUMN IF NOT EXISTS is_active BOOLEAN DEFAULT true,
    ADD COLUMN IF NOT EXISTS status TEXT DEFAULT 'active' CHECK (status IN ('draft', 'active', 'archived')),
    ADD COLUMN IF NOT EXISTS enforcement_mode TEXT DEFAULT 'enforce' CHECK (enforcement_mode IN ('monitor', 'enforce', 'redact')),
    ADD COLUMN IF NOT EXISTS fail_on_severity TEXT DEFAULT 'HIGH' CHECK (fail_on_severity IN ('LOW', 'MEDIUM', 'HIGH', 'CRITICAL')),
    ADD COLUMN IF NOT EXISTS priority INTEGER DEFAULT 100,
    ADD COLUMN IF NOT EXISTS metadata JSONB DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ DEFAULT NOW();

-- 2. Migrate any existing policy data to rulepacks (if any exists)
-- Since this is a fresh DB, there shouldn't be any, but let's be safe
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM policies LIMIT 1) THEN
        -- Migrate policy data to rulepacks
        UPDATE rulepacks r
        SET 
            enforcement_mode = p.enforcement_mode,
            fail_on_severity = p.fail_on_severity,
            priority = p.priority,
            metadata = p.metadata
        FROM policies p
        WHERE p.rulepack_id = r.id;
    END IF;
END $$;

-- 3. Drop the redundant tables
DROP TABLE IF EXISTS policy_versions CASCADE;
DROP TABLE IF EXISTS policies CASCADE;

-- 4. Update rulepack_versions to include the actual rules content
ALTER TABLE rulepack_versions
    ADD COLUMN IF NOT EXISTS yaml_content TEXT,
    ADD COLUMN IF NOT EXISTS rules JSONB,
    RENAME COLUMN dsl TO rules_legacy;

-- 5. Create better indexes
CREATE INDEX IF NOT EXISTS idx_rulepacks_active ON rulepacks(is_active) WHERE is_active = true;
CREATE INDEX IF NOT EXISTS idx_rulepacks_status ON rulepacks(status);
CREATE INDEX IF NOT EXISTS idx_rulepacks_tenant_active ON rulepacks(tenant_id, is_active) WHERE is_active = true;
CREATE INDEX IF NOT EXISTS idx_rulepacks_priority ON rulepacks(priority DESC);

-- 6. Update comments
COMMENT ON TABLE rulepacks IS 'Security policies with rules for LLM protection (formerly called policies)';
COMMENT ON COLUMN rulepacks.yaml_content IS 'Original YAML content of the rulepack';
COMMENT ON COLUMN rulepacks.rules IS 'Parsed rules in JSON format for fast evaluation';
COMMENT ON COLUMN rulepacks.enforcement_mode IS 'How to enforce: monitor (log only), enforce (block), redact (remove sensitive data)';
COMMENT ON COLUMN rulepacks.fail_on_severity IS 'Minimum severity level to trigger enforcement action';
COMMENT ON COLUMN rulepacks.priority IS 'Evaluation priority (higher number = higher priority)';

-- 7. Create a simpler view for active rulepacks
CREATE OR REPLACE VIEW active_rulepacks AS
SELECT 
    r.id,
    r.tenant_id,
    t.name as tenant_name,
    r.name as rulepack_name,
    r.description,
    r.status,
    r.enforcement_mode,
    r.fail_on_severity,
    r.priority,
    r.is_active,
    jsonb_array_length(COALESCE(r.rules->'rules', '[]'::jsonb)) as rule_count,
    r.created_at,
    r.updated_at
FROM rulepacks r
JOIN tenants t ON r.tenant_id = t.id
WHERE r.is_active = true
ORDER BY r.priority DESC, r.created_at DESC;

-- 8. Update the violations table foreign key if needed
-- (It should already reference rulepacks, but let's ensure the constraint name is clear)
ALTER TABLE violations 
    DROP CONSTRAINT IF EXISTS violations_policy_id_fkey,
    DROP CONSTRAINT IF EXISTS violations_rulepack_id_fkey;

ALTER TABLE violations
    ADD CONSTRAINT violations_rulepack_id_fkey 
    FOREIGN KEY (rulepack_id) REFERENCES rulepacks(id) ON DELETE SET NULL;

COMMENT ON VIEW active_rulepacks IS 'View of all active security policies/rulepacks with rule counts';