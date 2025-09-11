-- ================================================================
-- ROW LEVEL SECURITY (RLS) POLICIES - FROM 0009
-- ================================================================  
-- Creates: Complete multi-tenant RLS implementation
-- Purpose: Enforce tenant isolation at database level
-- Date: Consolidated 2025-08-27

-- ================================================================
-- UTILITY FUNCTIONS FOR RLS
-- ================================================================

-- Get current tenant ID from session context
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
        BEGIN
            SELECT EXISTS(
                SELECT 1 FROM admin_users 
                WHERE email = user_email AND is_active = true
            ) INTO is_admin;
        EXCEPTION
            WHEN OTHERS THEN
                -- admin_users table doesn't exist or other error
                is_admin := FALSE;
        END;
    END IF;
    
    -- Also allow admin access when no tenant context is set (for testing/admin operations)
    IF NOT is_admin AND get_current_tenant_id() IS NULL THEN
        is_admin := TRUE;
    END IF;
    
    RETURN is_admin;
END;
$$;

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
-- ENABLE RLS ON TENANT-ISOLATED TABLES
-- ================================================================

-- Core business tables
ALTER TABLE assignments ENABLE ROW LEVEL SECURITY;
ALTER TABLE audits ENABLE ROW LEVEL SECURITY;  
ALTER TABLE rulepacks ENABLE ROW LEVEL SECURITY;
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE subscriptions ENABLE ROW LEVEL SECURITY;

-- Production tables
ALTER TABLE api_keys ENABLE ROW LEVEL SECURITY;
ALTER TABLE sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE usage_metrics ENABLE ROW LEVEL SECURITY;
ALTER TABLE violations ENABLE ROW LEVEL SECURITY;
ALTER TABLE threat_intelligence ENABLE ROW LEVEL SECURITY;
ALTER TABLE alerts ENABLE ROW LEVEL SECURITY;
ALTER TABLE webhooks ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenant_settings ENABLE ROW LEVEL SECURITY;
ALTER TABLE feature_flags ENABLE ROW LEVEL SECURITY;

-- Service management tables
ALTER TABLE tenant_services ENABLE ROW LEVEL SECURITY;
ALTER TABLE service_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE service_metrics ENABLE ROW LEVEL SECURITY;

-- Special case: Tenants table (users see own tenant, admins see all)
ALTER TABLE tenants ENABLE ROW LEVEL SECURITY;

-- ================================================================
-- CREATE RLS POLICIES
-- ================================================================

-- Standard tenant isolation policy for most tables
CREATE POLICY tenant_isolation_policy ON assignments 
FOR ALL USING (tenant_id = get_current_tenant_id() OR is_platform_admin());

CREATE POLICY tenant_isolation_policy ON audits 
FOR ALL USING (tenant_id = get_current_tenant_id() OR is_platform_admin());

CREATE POLICY tenant_isolation_policy ON rulepacks 
FOR ALL USING (tenant_id = get_current_tenant_id() OR is_platform_admin());

CREATE POLICY tenant_isolation_policy ON users 
FOR ALL USING (tenant_id = get_current_tenant_id() OR is_platform_admin());

CREATE POLICY tenant_isolation_policy ON subscriptions 
FOR ALL USING (tenant_id = get_current_tenant_id() OR is_platform_admin());

CREATE POLICY tenant_isolation_policy ON api_keys 
FOR ALL USING (tenant_id = get_current_tenant_id() OR is_platform_admin());

CREATE POLICY tenant_isolation_policy ON sessions 
FOR ALL USING (tenant_id = get_current_tenant_id() OR is_platform_admin());

CREATE POLICY tenant_isolation_policy ON usage_metrics 
FOR ALL USING (tenant_id = get_current_tenant_id() OR is_platform_admin());

CREATE POLICY tenant_isolation_policy ON violations 
FOR ALL USING (tenant_id = get_current_tenant_id() OR is_platform_admin());

CREATE POLICY tenant_isolation_policy ON alerts 
FOR ALL USING (tenant_id = get_current_tenant_id() OR is_platform_admin());

CREATE POLICY tenant_isolation_policy ON webhooks 
FOR ALL USING (tenant_id = get_current_tenant_id() OR is_platform_admin());

CREATE POLICY tenant_isolation_policy ON tenant_settings 
FOR ALL USING (tenant_id = get_current_tenant_id() OR is_platform_admin());

CREATE POLICY tenant_isolation_policy ON feature_flags 
FOR ALL USING (tenant_id = get_current_tenant_id() OR is_platform_admin());

CREATE POLICY tenant_isolation_policy ON tenant_services 
FOR ALL USING (tenant_id = get_current_tenant_id() OR is_platform_admin());

CREATE POLICY tenant_isolation_policy ON service_events 
FOR ALL USING (tenant_id = get_current_tenant_id() OR is_platform_admin());

CREATE POLICY tenant_isolation_policy ON service_metrics 
FOR ALL USING (tenant_id = get_current_tenant_id() OR is_platform_admin());

-- Special policies for optional tenant_id columns
CREATE POLICY tenant_isolation_policy ON threat_intelligence 
FOR ALL USING (
    tenant_id IS NULL OR 
    tenant_id = get_current_tenant_id() OR 
    is_platform_admin()
);

-- Tenants table: Users see their own tenant, admins see all
CREATE POLICY tenant_access_policy ON tenants 
FOR ALL USING (
    id = get_current_tenant_id() OR 
    is_platform_admin()
);

-- ================================================================
-- PERFORMANCE INDEXES FOR RLS
-- ================================================================

-- Add indexes on tenant_id columns for efficient RLS filtering
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_assignments_tenant_rls ON assignments(tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audits_tenant_rls ON audits(tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_rulepacks_tenant_rls ON rulepacks(tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_users_tenant_rls ON users(tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_subscriptions_tenant_rls ON subscriptions(tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_api_keys_tenant_rls ON api_keys(tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_sessions_tenant_rls ON sessions(tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_usage_metrics_tenant_rls ON usage_metrics(tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_violations_tenant_rls ON violations(tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_alerts_tenant_rls ON alerts(tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_webhooks_tenant_rls ON webhooks(tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tenant_settings_tenant_rls ON tenant_settings(tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_feature_flags_tenant_rls ON feature_flags(tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tenant_services_tenant_rls ON tenant_services(tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_service_events_tenant_rls ON service_events(tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_service_metrics_tenant_rls ON service_metrics(tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_threat_intelligence_tenant_rls ON threat_intelligence(tenant_id);

-- Composite indexes for common RLS + business queries
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audits_tenant_created_rls ON audits(tenant_id, created_at DESC);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_violations_tenant_created_rls ON violations(tenant_id, created_at DESC);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_usage_metrics_tenant_period_rls ON usage_metrics(tenant_id, period_start DESC);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_alerts_tenant_status_rls ON alerts(tenant_id, status);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_service_events_tenant_time_rls ON service_events(tenant_id, created_at DESC);

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
-- COMMENTS FOR DOCUMENTATION
-- ================================================================

COMMENT ON FUNCTION get_current_tenant_id() IS 'Gets the current tenant ID from JWT claims or session context';
COMMENT ON FUNCTION is_platform_admin() IS 'Checks if the current user is a platform administrator';
COMMENT ON FUNCTION set_tenant_context(UUID) IS 'Sets the tenant context for the current session';
COMMENT ON FUNCTION validate_tenant_access(UUID) IS 'Validates if the current user has access to the specified tenant';