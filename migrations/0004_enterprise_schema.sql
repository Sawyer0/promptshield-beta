-- Enterprise API Gateway Schema
-- Migration: 0004_enterprise_schema.sql

-- 1. Update tenants table to match domain model
ALTER TABLE tenants 
ADD COLUMN IF NOT EXISTS status VARCHAR(50) DEFAULT 'active',
ADD COLUMN IF NOT EXISTS metadata JSONB DEFAULT '{}',
ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT NOW();

-- Create index for tenant status queries
CREATE INDEX IF NOT EXISTS idx_tenants_status ON tenants(status);

-- 2. Provider API Keys (encrypted LLM provider credentials)
CREATE TABLE IF NOT EXISTS provider_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL, -- openai, anthropic, azure
    key_alias VARCHAR(100) NOT NULL,
    encrypted_key TEXT NOT NULL,
    endpoint VARCHAR(500),
    deployment VARCHAR(100), -- for Azure
    is_default BOOLEAN DEFAULT false,
    status VARCHAR(50) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT NOW(),
    last_used TIMESTAMP,
    rotated_at TIMESTAMP,
    UNIQUE(tenant_id, provider, key_alias)
);

CREATE INDEX idx_provider_keys_tenant ON provider_keys(tenant_id);
CREATE INDEX idx_provider_keys_provider ON provider_keys(tenant_id, provider);
CREATE INDEX idx_provider_keys_default ON provider_keys(tenant_id, provider, is_default) WHERE is_default = true;

-- 3. Policies/Rulepacks (enhanced from existing)
CREATE TABLE IF NOT EXISTS policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    version INTEGER DEFAULT 1,
    content TEXT NOT NULL, -- YAML/JSON rules
    type VARCHAR(50) DEFAULT 'custom',
    created_by UUID,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(name, version)
);

CREATE INDEX idx_policies_name ON policies(name);
CREATE INDEX idx_policies_type ON policies(type);

-- 4. Policy Assignments (which policies apply to which tenant/routes)
CREATE TABLE IF NOT EXISTS policy_assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    policy_id UUID NOT NULL REFERENCES policies(id) ON DELETE CASCADE,
    target_scope VARCHAR(255) NOT NULL, -- /v1/openai/*, /v1/anthropic/*
    priority INTEGER DEFAULT 0,
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(tenant_id, target_scope, policy_id)
);

CREATE INDEX idx_policy_assignments_tenant ON policy_assignments(tenant_id);
CREATE INDEX idx_policy_assignments_scope ON policy_assignments(tenant_id, target_scope);
CREATE INDEX idx_policy_assignments_policy ON policy_assignments(policy_id);

-- 5. Enhanced Audit Trail (immutable, hash-chained)
CREATE TABLE IF NOT EXISTS audit_trail (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(id),
    actor_id UUID,
    actor_type VARCHAR(50) NOT NULL, -- user, system, api_key
    actor_email VARCHAR(255),
    action VARCHAR(100) NOT NULL, -- tenant.create, key.rotate, etc.
    object_type VARCHAR(50) NOT NULL,
    object_id UUID NOT NULL,
    before JSONB,
    after JSONB,
    metadata JSONB,
    created_at TIMESTAMP DEFAULT NOW(),
    hash VARCHAR(64), -- SHA-256 of entry
    prev_hash VARCHAR(64) -- chain to previous
);

CREATE INDEX idx_audit_trail_tenant ON audit_trail(tenant_id, created_at);
CREATE INDEX idx_audit_trail_object ON audit_trail(object_type, object_id, created_at);
CREATE INDEX idx_audit_trail_actor ON audit_trail(actor_id, created_at);
CREATE INDEX idx_audit_trail_created_at ON audit_trail(created_at);

-- 6. Usage Metrics (partitioned by time)
CREATE TABLE IF NOT EXISTS usage_metrics (
    id UUID DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    timestamp TIMESTAMP NOT NULL,
    window_bucket VARCHAR(20) NOT NULL, -- minute, hour, day
    provider VARCHAR(50),
    endpoint VARCHAR(255),
    request_count BIGINT DEFAULT 0,
    token_count BIGINT DEFAULT 0,
    prompt_tokens BIGINT DEFAULT 0,
    completion_tokens BIGINT DEFAULT 0,
    violations INTEGER DEFAULT 0,
    blocked_requests INTEGER DEFAULT 0,
    latency_p50 FLOAT,
    latency_p95 FLOAT,
    latency_p99 FLOAT,
    PRIMARY KEY (id, timestamp)
) PARTITION BY RANGE (timestamp);

-- Create initial partitions for current year
DO $$
DECLARE
    start_date DATE := DATE_TRUNC('month', CURRENT_DATE);
    end_date DATE := start_date + INTERVAL '1 month';
    partition_name TEXT;
BEGIN
    FOR i IN 0..11 LOOP
        partition_name := 'usage_metrics_' || TO_CHAR(start_date, 'YYYY_MM');
        EXECUTE format('CREATE TABLE IF NOT EXISTS %I PARTITION OF usage_metrics 
                       FOR VALUES FROM (%L) TO (%L)', 
                       partition_name, start_date, end_date);
        start_date := end_date;
        end_date := start_date + INTERVAL '1 month';
    END LOOP;
END $$;

CREATE INDEX idx_usage_metrics_tenant_time ON usage_metrics(tenant_id, timestamp);
CREATE INDEX idx_usage_metrics_provider ON usage_metrics(tenant_id, provider, timestamp);

-- 7. Quotas (rate limiting per tenant)
CREATE TABLE IF NOT EXISTS quotas (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID UNIQUE NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    requests_per_minute INTEGER,
    requests_per_hour INTEGER,
    tokens_per_hour BIGINT,
    max_prompt_tokens INTEGER,
    max_completion_tokens INTEGER,
    updated_at TIMESTAMP DEFAULT NOW()
);

-- 8. API Tokens (for client authentication)
CREATE TABLE IF NOT EXISTS api_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255),
    scopes TEXT[], -- array of permissions
    expires_at TIMESTAMP,
    last_used TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    revoked_at TIMESTAMP
);

CREATE INDEX idx_api_tokens_tenant ON api_tokens(tenant_id);
CREATE INDEX idx_api_tokens_hash ON api_tokens(token_hash) WHERE revoked_at IS NULL;
CREATE INDEX idx_api_tokens_expires ON api_tokens(expires_at) WHERE expires_at IS NOT NULL AND revoked_at IS NULL;

-- Add updated_at trigger for tenants
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_tenants_updated_at BEFORE UPDATE ON tenants
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_policy_assignments_updated_at BEFORE UPDATE ON policy_assignments
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_policies_updated_at BEFORE UPDATE ON policies
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_quotas_updated_at BEFORE UPDATE ON quotas
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();