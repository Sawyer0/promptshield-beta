-- ================================================================
-- ROW LEVEL SECURITY (RLS) POLICIES FOR MULTI-TENANT ISOLATION
-- ================================================================

-- Enable RLS on tenant-isolated tables
ALTER TABLE active_rulepacks ENABLE ROW LEVEL SECURITY;
ALTER TABLE assignments ENABLE ROW LEVEL SECURITY;
ALTER TABLE audits ENABLE ROW LEVEL SECURITY;
ALTER TABLE rulepacks ENABLE ROW LEVEL SECURITY;
ALTER TABLE rulepack_versions ENABLE ROW LEVEL SECURITY;
ALTER TABLE violations ENABLE ROW LEVEL SECURITY;
ALTER TABLE sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE usage_metrics ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenant_settings ENABLE ROW LEVEL SECURITY;
ALTER TABLE webhooks ENABLE ROW LEVEL SECURITY;
ALTER TABLE alerts ENABLE ROW LEVEL SECURITY;
ALTER TABLE feature_flags ENABLE ROW LEVEL SECURITY;
ALTER TABLE threat_intelligence ENABLE ROW LEVEL SECURITY;

-- ================================================================
-- UTILITY FUNCTIONS FOR RLS
-- ================================================================

-- Get current tenant ID from JWT token or session
CREATE OR REPLACE FUNCTION get_current_tenant_id()
RETURNS UUID
LANGUAGE plpgsql
STABLE SECURITY DEFINER
AS $$
DECLARE
    tenant_id UUID;
BEGIN
    -- Try to get tenant_id from JWT token first
    SELECT (current_setting('request.jwt.claims', true)::json->>'tenant_id')::uuid INTO tenant_id;
    
    -- If no JWT, try session variable
    IF tenant_id IS NULL THEN
        BEGIN
            tenant_id := current_setting('app.current_tenant_id')::uuid;
        EXCEPTION
            WHEN OTHERS THEN
                tenant_id := NULL;
        END;
    END IF;
    
    RETURN tenant_id;
END;
$$;

-- Check if current user is a platform admin
CREATE OR REPLACE FUNCTION is_platform_admin()
RETURNS BOOLEAN
LANGUAGE plpgsql
STABLE SECURITY DEFINER
AS $$
DECLARE
    is_admin BOOLEAN := FALSE;
    user_email TEXT;
BEGIN
    -- Get user email from JWT
    SELECT (current_setting('request.jwt.claims', true)::json->>'email') INTO user_email;
    
    -- Check if user is in admin_users table
    IF user_email IS NOT NULL THEN
        SELECT EXISTS(
            SELECT 1 FROM admin_users 
            WHERE email = user_email AND enabled = true
        ) INTO is_admin;
    END IF;
    
    RETURN is_admin;
END;
$$;

-- ================================================================
-- RLS POLICIES FOR TENANT-ISOLATED TABLES
-- ================================================================

-- ACTIVE_RULEPACKS: Tenants can only see/modify their own active rulepacks
DROP POLICY IF EXISTS tenant_isolation_policy ON active_rulepacks;
CREATE POLICY tenant_isolation_policy ON active_rulepacks
    FOR ALL
    USING (tenant_id = get_current_tenant_id() OR is_platform_admin());

-- ASSIGNMENTS: Tenants can only see/modify their own policy assignments
DROP POLICY IF EXISTS tenant_isolation_policy ON assignments;
CREATE POLICY tenant_isolation_policy ON assignments
    FOR ALL
    USING (tenant_id = get_current_tenant_id() OR is_platform_admin());

-- AUDITS: Tenants can only see their own audit logs
DROP POLICY IF EXISTS tenant_isolation_policy ON audits;
CREATE POLICY tenant_isolation_policy ON audits
    FOR ALL
    USING (tenant_id = get_current_tenant_id() OR is_platform_admin());

-- RULEPACKS: Tenants can only see/modify their own rulepacks
DROP POLICY IF EXISTS tenant_isolation_policy ON rulepacks;
CREATE POLICY tenant_isolation_policy ON rulepacks
    FOR ALL
    USING (tenant_id = get_current_tenant_id() OR is_platform_admin());

-- RULEPACK_VERSIONS: Tenants can only see their own rulepack versions
DROP POLICY IF EXISTS tenant_isolation_policy ON rulepack_versions;
CREATE POLICY tenant_isolation_policy ON rulepack_versions
    FOR ALL
    USING (tenant_id = get_current_tenant_id() OR is_platform_admin());

-- VIOLATIONS: Tenants can only see their own violations
DROP POLICY IF EXISTS tenant_isolation_policy ON violations;
CREATE POLICY tenant_isolation_policy ON violations
    FOR ALL
    USING (tenant_id = get_current_tenant_id() OR is_platform_admin());

-- SESSIONS: Users can only see their own sessions within their tenant
DROP POLICY IF EXISTS tenant_isolation_policy ON sessions;
CREATE POLICY tenant_isolation_policy ON sessions
    FOR ALL
    USING (tenant_id = get_current_tenant_id() OR is_platform_admin());

-- USAGE_METRICS: Tenants can only see their own usage metrics
DROP POLICY IF EXISTS tenant_isolation_policy ON usage_metrics;
CREATE POLICY tenant_isolation_policy ON usage_metrics
    FOR ALL
    USING (tenant_id = get_current_tenant_id() OR is_platform_admin());

-- TENANT_SETTINGS: Tenants can only see/modify their own settings
DROP POLICY IF EXISTS tenant_isolation_policy ON tenant_settings;
CREATE POLICY tenant_isolation_policy ON tenant_settings
    FOR ALL
    USING (tenant_id = get_current_tenant_id() OR is_platform_admin());

-- WEBHOOKS: Tenants can only see/modify their own webhooks
DROP POLICY IF EXISTS tenant_isolation_policy ON webhooks;
CREATE POLICY tenant_isolation_policy ON webhooks
    FOR ALL
    USING (tenant_id = get_current_tenant_id() OR is_platform_admin());

-- ALERTS: Tenants can only see their own alerts
DROP POLICY IF EXISTS tenant_isolation_policy ON alerts;
CREATE POLICY tenant_isolation_policy ON alerts
    FOR ALL
    USING (tenant_id = get_current_tenant_id() OR is_platform_admin());

-- FEATURE_FLAGS: Tenants can only see their own feature flags
DROP POLICY IF EXISTS tenant_isolation_policy ON feature_flags;
CREATE POLICY tenant_isolation_policy ON feature_flags
    FOR ALL
    USING (tenant_id = get_current_tenant_id() OR is_platform_admin());

-- THREAT_INTELLIGENCE: Tenants can only see their own threat intelligence data
DROP POLICY IF EXISTS tenant_isolation_policy ON threat_intelligence;
CREATE POLICY tenant_isolation_policy ON threat_intelligence
    FOR ALL
    USING (tenant_id = get_current_tenant_id() OR is_platform_admin());

-- ================================================================
-- SPECIAL POLICIES FOR SHARED TABLES
-- ================================================================

-- USERS: Enable RLS but allow cross-tenant visibility for admins and user management
ALTER TABLE users ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS user_access_policy ON users;
CREATE POLICY user_access_policy ON users
    FOR ALL
    USING (
        -- Users can see themselves
        id = (current_setting('request.jwt.claims', true)::json->>'sub')::uuid
        -- Admins within same tenant can see tenant users
        OR (tenant_id = get_current_tenant_id() AND is_platform_admin())
        -- Platform admins can see all users
        OR is_platform_admin()
    );

-- TENANTS: Only platform admins can manage tenants
ALTER TABLE tenants ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_management_policy ON tenants;
CREATE POLICY tenant_management_policy ON tenants
    FOR ALL
    USING (
        -- Users can see their own tenant info
        id = get_current_tenant_id()
        -- Platform admins can see all tenants
        OR is_platform_admin()
    );

-- ================================================================
-- SECURITY FUNCTIONS FOR APPLICATION USE
-- ================================================================

-- Set tenant context for application connections
CREATE OR REPLACE FUNCTION set_tenant_context(tenant_uuid UUID)
RETURNS VOID
LANGUAGE plpgsql
SECURITY DEFINER
AS $$
BEGIN
    PERFORM set_config('app.current_tenant_id', tenant_uuid::text, false);
END;
$$;

-- Validate tenant access for API calls
CREATE OR REPLACE FUNCTION validate_tenant_access(tenant_uuid UUID)
RETURNS BOOLEAN
LANGUAGE plpgsql
STABLE SECURITY DEFINER
AS $$
BEGIN
    -- Platform admins have access to all tenants
    IF is_platform_admin() THEN
        RETURN TRUE;
    END IF;
    
    -- Regular users can only access their own tenant
    RETURN tenant_uuid = get_current_tenant_id();
END;
$$;

-- ================================================================
-- GRANTS FOR APPLICATION ROLES
-- ================================================================

-- Create application role for API server
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'promptshield_api') THEN
        CREATE ROLE promptshield_api;
    END IF;
END
$$;

-- Grant necessary permissions to API role
GRANT USAGE ON SCHEMA public TO promptshield_api;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO promptshield_api;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO promptshield_api;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO promptshield_api;

-- ================================================================
-- INDEXES FOR PERFORMANCE WITH RLS
-- ================================================================

-- Add indexes on tenant_id columns for efficient RLS filtering
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_active_rulepacks_tenant_id ON active_rulepacks(tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_assignments_tenant_id ON assignments(tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audits_tenant_id ON audits(tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_rulepacks_tenant_id ON rulepacks(tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_rulepack_versions_tenant_id ON rulepack_versions(tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_violations_tenant_id ON violations(tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_sessions_tenant_id ON sessions(tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_usage_metrics_tenant_id ON usage_metrics(tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tenant_settings_tenant_id ON tenant_settings(tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_webhooks_tenant_id ON webhooks(tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_alerts_tenant_id ON alerts(tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_feature_flags_tenant_id ON feature_flags(tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_threat_intelligence_tenant_id ON threat_intelligence(tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_users_tenant_id ON users(tenant_id);

-- Composite indexes for common queries
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audits_tenant_created ON audits(tenant_id, created_at DESC);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_violations_tenant_created ON violations(tenant_id, created_at DESC);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_usage_metrics_tenant_period ON usage_metrics(tenant_id, period_start DESC);

-- ================================================================
-- COMMENTS FOR DOCUMENTATION
-- ================================================================

COMMENT ON FUNCTION get_current_tenant_id() IS 'Gets the current tenant ID from JWT claims or session context';
COMMENT ON FUNCTION is_platform_admin() IS 'Checks if the current user is a platform administrator';
COMMENT ON FUNCTION set_tenant_context(UUID) IS 'Sets the tenant context for the current session';
COMMENT ON FUNCTION validate_tenant_access(UUID) IS 'Validates if the current user has access to the specified tenant';

-- ================================================================
-- VERIFICATION QUERIES (FOR TESTING)
-- ================================================================

-- Test RLS policies are working
-- These should be run by your test suite:

/*
-- Test 1: Set tenant context and verify isolation
SELECT set_tenant_context('550e8400-e29b-41d4-a716-446655440001'::uuid);
SELECT COUNT(*) FROM rulepacks; -- Should only show tenant 1 rulepacks

-- Test 2: Switch tenant context
SELECT set_tenant_context('550e8400-e29b-41d4-a716-446655440002'::uuid);
SELECT COUNT(*) FROM rulepacks; -- Should only show tenant 2 rulepacks

-- Test 3: Verify audit isolation
SELECT COUNT(*) FROM audits; -- Should only show current tenant's audits
*/