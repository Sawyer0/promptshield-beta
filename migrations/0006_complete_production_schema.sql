-- Complete Production Schema for PromptShield SaaS
-- This adds all missing tables for a production-ready system

-- =====================================================
-- 1. USERS & AUTHENTICATION
-- =====================================================

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT,
    full_name TEXT,
    role TEXT NOT NULL DEFAULT 'user' CHECK (role IN ('user', 'admin', 'owner', 'viewer')),
    is_active BOOLEAN DEFAULT true,
    email_verified BOOLEAN DEFAULT false,
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    key_hash TEXT NOT NULL UNIQUE,
    key_prefix TEXT NOT NULL, -- First 8 chars for identification
    permissions JSONB DEFAULT '["read"]',
    rate_limit INTEGER DEFAULT 1000, -- requests per minute
    last_used_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    revoked_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    ip_address INET,
    user_agent TEXT,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- =====================================================
-- 2. BILLING & SUBSCRIPTIONS
-- =====================================================

CREATE TABLE IF NOT EXISTS subscription_plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    price_monthly INTEGER NOT NULL, -- in cents
    price_yearly INTEGER, -- in cents
    features JSONB NOT NULL,
    limits JSONB NOT NULL, -- {"api_calls": 10000, "rules": 100, etc}
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Insert default plans
INSERT INTO subscription_plans (name, display_name, price_monthly, price_yearly, features, limits) VALUES
    ('free', 'Free Tier', 0, 0, 
     '["10,000 API calls/month", "5 rulepacks", "Community support"]',
     '{"api_calls_monthly": 10000, "rulepacks": 5, "rules_per_pack": 50}'),
    ('pro', 'Professional', 4900, 49000,
     '["100,000 API calls/month", "Unlimited rulepacks", "Email support", "Advanced analytics"]',
     '{"api_calls_monthly": 100000, "rulepacks": -1, "rules_per_pack": -1}'),
    ('enterprise', 'Enterprise', 29900, 299000,
     '["Unlimited API calls", "Unlimited everything", "24/7 support", "SLA", "Custom integration"]',
     '{"api_calls_monthly": -1, "rulepacks": -1, "rules_per_pack": -1}')
ON CONFLICT (name) DO NOTHING;

CREATE TABLE IF NOT EXISTS subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL UNIQUE REFERENCES tenants(id) ON DELETE CASCADE,
    plan_id UUID NOT NULL REFERENCES subscription_plans(id),
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('trial', 'active', 'past_due', 'canceled', 'paused')),
    stripe_customer_id TEXT,
    stripe_subscription_id TEXT,
    trial_ends_at TIMESTAMPTZ,
    current_period_start TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    current_period_end TIMESTAMPTZ NOT NULL,
    canceled_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS usage_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    metric_type TEXT NOT NULL CHECK (metric_type IN ('api_calls', 'data_processed_bytes', 'rules_evaluated', 'violations_detected')),
    value BIGINT NOT NULL DEFAULT 0,
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,
    metadata JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- =====================================================
-- 3. SECURITY & VIOLATIONS
-- =====================================================

CREATE TABLE IF NOT EXISTS violations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    rulepack_id UUID REFERENCES rulepacks(id) ON DELETE SET NULL,
    rule_id TEXT NOT NULL,
    severity TEXT NOT NULL CHECK (severity IN ('LOW', 'MEDIUM', 'HIGH', 'CRITICAL')),
    request_id TEXT,
    source_ip INET,
    user_agent TEXT,
    matched_content TEXT, -- Redacted version
    matched_pattern TEXT,
    action_taken TEXT CHECK (action_taken IN ('blocked', 'allowed', 'redacted', 'logged')),
    false_positive BOOLEAN DEFAULT false,
    metadata JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS threat_intelligence (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pattern TEXT NOT NULL,
    pattern_type TEXT NOT NULL CHECK (pattern_type IN ('keyword', 'regex', 'semantic')),
    category TEXT NOT NULL,
    severity TEXT NOT NULL CHECK (severity IN ('LOW', 'MEDIUM', 'HIGH', 'CRITICAL')),
    description TEXT,
    source TEXT, -- Where this intel came from
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- =====================================================
-- 4. OPERATIONS & MONITORING
-- =====================================================

CREATE TABLE IF NOT EXISTS alerts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    alert_type TEXT NOT NULL,
    severity TEXT NOT NULL CHECK (severity IN ('info', 'warning', 'error', 'critical')),
    title TEXT NOT NULL,
    description TEXT,
    metadata JSONB,
    acknowledged BOOLEAN DEFAULT false,
    acknowledged_by UUID REFERENCES users(id),
    acknowledged_at TIMESTAMPTZ,
    resolved BOOLEAN DEFAULT false,
    resolved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS webhooks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    events TEXT[] NOT NULL, -- ['violation.detected', 'rulepack.updated', etc]
    secret TEXT NOT NULL, -- For signing webhooks
    is_active BOOLEAN DEFAULT true,
    last_triggered_at TIMESTAMPTZ,
    failure_count INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- =====================================================
-- 5. CONFIGURATION
-- =====================================================

CREATE TABLE IF NOT EXISTS settings (
    key TEXT PRIMARY KEY,
    value JSONB NOT NULL,
    description TEXT,
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS tenant_settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    key TEXT NOT NULL,
    value JSONB NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, key)
);

CREATE TABLE IF NOT EXISTS feature_flags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    enabled_globally BOOLEAN DEFAULT false,
    enabled_tenants UUID[] DEFAULT '{}', -- Array of tenant IDs
    rollout_percentage INTEGER DEFAULT 0 CHECK (rollout_percentage >= 0 AND rollout_percentage <= 100),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- =====================================================
-- 6. ADMIN & DEVELOPER SPECIFIC
-- =====================================================

CREATE TABLE IF NOT EXISTS admin_users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    is_super_admin BOOLEAN DEFAULT false,
    permissions JSONB DEFAULT '{}',
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS deployment_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    environment TEXT NOT NULL CHECK (environment IN ('development', 'staging', 'production')),
    config JSONB NOT NULL,
    deployed_by UUID REFERENCES admin_users(id),
    deployed_at TIMESTAMPTZ DEFAULT NOW(),
    rollback_to UUID REFERENCES deployment_configs(id),
    is_active BOOLEAN DEFAULT false
);

-- =====================================================
-- INDEXES FOR PERFORMANCE
-- =====================================================

-- Users & Auth
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_tenant ON users(tenant_id);
CREATE INDEX idx_api_keys_tenant ON api_keys(tenant_id);
CREATE INDEX idx_api_keys_prefix ON api_keys(key_prefix);
CREATE INDEX idx_sessions_user ON sessions(user_id);
CREATE INDEX idx_sessions_expires ON sessions(expires_at);

-- Billing
CREATE INDEX idx_subscriptions_tenant ON subscriptions(tenant_id);
CREATE INDEX idx_subscriptions_status ON subscriptions(status);
CREATE INDEX idx_usage_metrics_tenant_period ON usage_metrics(tenant_id, period_start, period_end);

-- Security
CREATE INDEX idx_violations_tenant ON violations(tenant_id);
CREATE INDEX idx_violations_created ON violations(created_at DESC);
CREATE INDEX idx_violations_severity ON violations(severity);
CREATE INDEX idx_violations_request ON violations(request_id);

-- Monitoring
CREATE INDEX idx_alerts_tenant ON alerts(tenant_id);
CREATE INDEX idx_alerts_unresolved ON alerts(resolved) WHERE resolved = false;
CREATE INDEX idx_webhooks_tenant ON webhooks(tenant_id);
CREATE INDEX idx_webhooks_active ON webhooks(is_active) WHERE is_active = true;

-- =====================================================
-- ROW LEVEL SECURITY (RLS) for Supabase
-- =====================================================

-- Enable RLS on sensitive tables
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE api_keys ENABLE ROW LEVEL SECURITY;
ALTER TABLE violations ENABLE ROW LEVEL SECURITY;
ALTER TABLE subscriptions ENABLE ROW LEVEL SECURITY;

-- Basic RLS policies (you can customize these)
CREATE POLICY "Users can view own tenant data" ON users
    FOR SELECT USING (tenant_id = current_setting('app.current_tenant')::uuid);

CREATE POLICY "Users can view own tenant violations" ON violations
    FOR SELECT USING (tenant_id = current_setting('app.current_tenant')::uuid);

-- =====================================================
-- USEFUL VIEWS
-- =====================================================

CREATE OR REPLACE VIEW tenant_overview AS
SELECT 
    t.id,
    t.name,
    s.status as subscription_status,
    sp.display_name as plan_name,
    COUNT(DISTINCT u.id) as user_count,
    COUNT(DISTINCT r.id) as rulepack_count,
    COUNT(DISTINCT ak.id) as api_key_count
FROM tenants t
LEFT JOIN subscriptions s ON t.id = s.tenant_id
LEFT JOIN subscription_plans sp ON s.plan_id = sp.id
LEFT JOIN users u ON t.id = u.tenant_id
LEFT JOIN rulepacks r ON t.id = r.tenant_id
LEFT JOIN api_keys ak ON t.id = ak.tenant_id
GROUP BY t.id, t.name, s.status, sp.display_name;

COMMENT ON TABLE users IS 'User accounts with role-based access control';
COMMENT ON TABLE api_keys IS 'API keys for programmatic access to PromptShield';
COMMENT ON TABLE violations IS 'Security violations detected by PromptShield';
COMMENT ON TABLE subscriptions IS 'Tenant subscription and billing information';
COMMENT ON TABLE threat_intelligence IS 'Global threat patterns and intelligence';