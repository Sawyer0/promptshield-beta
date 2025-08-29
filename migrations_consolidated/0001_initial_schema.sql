-- ================================================================
-- INITIAL SCHEMA - CONSOLIDATED FROM 0001 + 0003 + TENANT FIXES
-- ================================================================
-- Creates: Core tables with all fixes applied from the beginning
-- Includes: Performance indexes and column additions
-- Date: Consolidated 2025-08-27

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ================================================================
-- CORE BUSINESS TABLES
-- ================================================================

-- 1. TENANTS (Master tenant registry)
CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    status VARCHAR(50) DEFAULT 'active',
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_tenants_status ON tenants(status);

-- 2. RULEPACKS (Security policy containers)
CREATE TABLE rulepacks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT,
    
    -- Rule content and versioning
    yaml_content TEXT,
    rules JSONB,
    current_version_id UUID,
    
    -- Management fields  
    is_active BOOLEAN DEFAULT true,
    status TEXT DEFAULT 'active' CHECK (status IN ('draft', 'active', 'archived')),
    enforcement_mode TEXT DEFAULT 'enforce' CHECK (enforcement_mode IN ('monitor', 'enforce', 'redact')),
    fail_on_severity TEXT DEFAULT 'HIGH' CHECK (fail_on_severity IN ('LOW', 'MEDIUM', 'HIGH', 'CRITICAL')),
    priority INTEGER DEFAULT 100,
    metadata JSONB DEFAULT '{}',
    
    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- 3. RULEPACK VERSIONS (Version history)
CREATE TABLE rulepack_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rulepack_id UUID NOT NULL REFERENCES rulepacks(id) ON DELETE CASCADE,
    version INT NOT NULL,
    dsl JSONB NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('draft','approved','active','archived')),
    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    UNIQUE(rulepack_id, version)
);

-- Add foreign key constraint for current version
ALTER TABLE rulepacks
    ADD CONSTRAINT fk_current_version
    FOREIGN KEY (current_version_id) REFERENCES rulepack_versions(id);

-- 4. ASSIGNMENTS (Policy assignments to tenants)
CREATE TABLE assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    rulepack_id UUID NOT NULL REFERENCES rulepacks(id) ON DELETE CASCADE,
    target_scope TEXT NOT NULL,
    priority INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 5. AUDITS (Audit trail)
CREATE TABLE audits (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    actor_id UUID,
    actor_email TEXT,
    action TEXT NOT NULL,
    object_type TEXT NOT NULL,
    object_id UUID NOT NULL,
    diff JSONB,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ================================================================
-- PERFORMANCE INDEXES
-- ================================================================

-- RulePack indexes
CREATE INDEX idx_rulepack_versions_rulepack ON rulepack_versions(rulepack_id);
CREATE INDEX idx_rulepack_versions_status ON rulepack_versions(status);
CREATE INDEX idx_rulepack_versions_dsl_gin ON rulepack_versions USING GIN (dsl);
CREATE INDEX idx_rpv_rulepack_version_desc ON rulepack_versions (rulepack_id, version DESC);

-- Assignment indexes
CREATE UNIQUE INDEX uk_assignments_scope ON assignments (tenant_id, target_scope, rulepack_id);

-- Audit indexes
CREATE INDEX idx_audits_tenant_ts ON audits (tenant_id, created_at DESC);
CREATE INDEX idx_audits_object_ts ON audits (object_type, object_id, created_at DESC);

-- ================================================================
-- COMMENTS FOR DOCUMENTATION
-- ================================================================

COMMENT ON TABLE tenants IS 'Master tenant registry for multi-tenant SaaS';
COMMENT ON TABLE rulepacks IS 'Security policy containers with rules and configuration';
COMMENT ON TABLE rulepack_versions IS 'Version history for rulepacks';
COMMENT ON TABLE assignments IS 'Assignments of rulepacks to tenant scopes';
COMMENT ON TABLE audits IS 'Audit trail for all system changes';