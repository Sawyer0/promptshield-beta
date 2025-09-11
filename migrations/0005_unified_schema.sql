-- Migration: 0005_unified_schema.sql
-- Purpose: Unify policies and rulepacks schema across frontend and backend
-- Date: 2025-25-08

-- ============================================================================
-- BACKEND SCHEMA (Primary source of truth)
-- ============================================================================

-- Drop conflicting tables if they exist
DROP TABLE IF EXISTS policy_assignments CASCADE;
DROP TABLE IF EXISTS policies CASCADE;

-- 1. RulePacks table (container for rules)
-- This is the main entity users create and manage
CREATE TABLE IF NOT EXISTS rulepacks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    version VARCHAR(50) DEFAULT '1.0.0',
    author VARCHAR(255),
    category VARCHAR(100), -- security, compliance, custom
    tags TEXT[],
    
    -- Rule content (YAML format matching internal/rules)
    content TEXT NOT NULL,
    
    -- Management fields
    status VARCHAR(50) DEFAULT 'draft', -- draft, active, archived
    is_active BOOLEAN DEFAULT false,
    created_by UUID,
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    
    -- Timestamps
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    activated_at TIMESTAMP,
    
    -- Ensure unique active version per name per tenant
    CONSTRAINT unique_active_rulepack UNIQUE(tenant_id, name, is_active)
);

-- 2. Policies table (active rulepack deployments)
-- These are instantiated rulepacks with specific configurations
CREATE TABLE IF NOT EXISTS policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rulepack_id UUID NOT NULL REFERENCES rulepacks(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    
    -- Policy configuration
    name VARCHAR(255) NOT NULL,
    target_scope VARCHAR(255) NOT NULL, -- /v1/*, /api/chat/*, etc.
    enforcement_mode VARCHAR(50) DEFAULT 'enforce', -- observe, warn, enforce
    priority INTEGER DEFAULT 0,
    enabled BOOLEAN DEFAULT true,
    
    -- Runtime settings
    fail_on_severity VARCHAR(50) DEFAULT 'HIGH', -- LOW, MEDIUM, HIGH, CRITICAL
    max_processing_time_ms INTEGER DEFAULT 300,
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    -- Unique policy per scope per tenant
    CONSTRAINT unique_policy_scope UNIQUE(tenant_id, target_scope, name)
);

-- 3. Policy versions (history tracking)
CREATE TABLE IF NOT EXISTS policy_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    policy_id UUID NOT NULL REFERENCES policies(id) ON DELETE CASCADE,
    rulepack_id UUID NOT NULL REFERENCES rulepacks(id),
    version_number INTEGER NOT NULL,
    content TEXT NOT NULL,
    change_summary TEXT,
    created_by UUID,
    created_at TIMESTAMP DEFAULT NOW(),
    CONSTRAINT unique_policy_version UNIQUE(policy_id, version_number)
);

-- 4. Update violations table to reference policies
ALTER TABLE violations 
    DROP CONSTRAINT IF EXISTS violations_policy_id_fkey,
    ADD CONSTRAINT violations_policy_id_fkey 
    FOREIGN KEY (policy_id) REFERENCES policies(id) ON DELETE SET NULL;

-- Indexes for performance
CREATE INDEX idx_rulepacks_tenant ON rulepacks(tenant_id);
CREATE INDEX idx_rulepacks_active ON rulepacks(is_active) WHERE is_active = true;
CREATE INDEX idx_rulepacks_status ON rulepacks(status);
CREATE INDEX idx_rulepacks_category ON rulepacks(category);

CREATE INDEX idx_policies_tenant ON policies(tenant_id);
CREATE INDEX idx_policies_rulepack ON policies(rulepack_id);
CREATE INDEX idx_policies_scope ON policies(tenant_id, target_scope);
CREATE INDEX idx_policies_enabled ON policies(enabled) WHERE enabled = true;

CREATE INDEX idx_policy_versions_policy ON policy_versions(policy_id);

-- ============================================================================
-- FRONTEND SCHEMA UPDATES
-- ============================================================================
-- The frontend should use the same tables via API calls to the backend
-- Remove duplicate tables from frontend schema and use backend as source of truth