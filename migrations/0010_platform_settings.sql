-- Migration: 0010_platform_settings.sql
-- Description: Add platform settings table for SaaS admin configuration
-- Author: PromptShield Development Team

-- Create platform_settings table
CREATE TABLE IF NOT EXISTS platform_settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    settings JSONB NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by VARCHAR(255) NOT NULL DEFAULT 'system',
    
    -- Indexes for performance
    CONSTRAINT platform_settings_updated_by_check CHECK (updated_by != ''),
    CONSTRAINT platform_settings_settings_check CHECK (settings IS NOT NULL)
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_platform_settings_updated_at ON platform_settings(updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_platform_settings_updated_by ON platform_settings(updated_by);

-- Create GIN index on JSONB settings for efficient JSON queries
CREATE INDEX IF NOT EXISTS idx_platform_settings_gin ON platform_settings USING GIN (settings);

-- Add RLS policy for platform settings (admin-only access)
ALTER TABLE platform_settings ENABLE ROW LEVEL SECURITY;

-- Platform admins can view and modify all settings
DROP POLICY IF EXISTS platform_settings_admin_policy ON platform_settings;
CREATE POLICY platform_settings_admin_policy ON platform_settings
    FOR ALL TO authenticated
    USING (is_platform_admin())
    WITH CHECK (is_platform_admin());

-- Create function to get current platform settings
CREATE OR REPLACE FUNCTION get_platform_settings()
RETURNS JSONB
LANGUAGE plpgsql
STABLE SECURITY DEFINER
AS $$
DECLARE
    current_settings JSONB;
BEGIN
    -- Get the most recent settings
    SELECT settings INTO current_settings
    FROM platform_settings
    ORDER BY updated_at DESC
    LIMIT 1;
    
    -- Return settings or null if none exist
    RETURN current_settings;
END;
$$;

-- Create function to update platform settings with validation
CREATE OR REPLACE FUNCTION update_platform_settings(
    new_settings JSONB,
    updated_by_user VARCHAR(255) DEFAULT 'system'
)
RETURNS UUID
LANGUAGE plpgsql
SECURITY DEFINER
AS $$
DECLARE
    settings_id UUID;
    required_keys TEXT[] := ARRAY['platform', 'defaults', 'email', 'notifications', 'security'];
    key TEXT;
BEGIN
    -- Validate that required keys exist
    FOREACH key IN ARRAY required_keys
    LOOP
        IF NOT (new_settings ? key) THEN
            RAISE EXCEPTION 'Missing required settings key: %', key;
        END IF;
    END LOOP;
    
    -- Validate platform name
    IF (new_settings -> 'platform' ->> 'name') IS NULL OR 
       length(new_settings -> 'platform' ->> 'name') < 1 THEN
        RAISE EXCEPTION 'Platform name is required';
    END IF;
    
    -- Validate tenant limits
    IF (new_settings -> 'defaults' -> 'tenant_limits' ->> 'max_requests_per_day')::INTEGER <= 0 THEN
        RAISE EXCEPTION 'Max requests per day must be positive';
    END IF;
    
    IF (new_settings -> 'defaults' -> 'tenant_limits' ->> 'max_rulepacks')::INTEGER <= 0 THEN
        RAISE EXCEPTION 'Max rulepacks must be positive';
    END IF;
    
    -- Generate new UUID for this settings version
    settings_id := gen_random_uuid();
    
    -- Insert new settings record
    INSERT INTO platform_settings (id, settings, updated_at, updated_by)
    VALUES (settings_id, new_settings, NOW(), updated_by_user);
    
    -- Clean up old settings (keep last 10 versions)
    DELETE FROM platform_settings
    WHERE id NOT IN (
        SELECT id FROM platform_settings
        ORDER BY updated_at DESC
        LIMIT 10
    );
    
    RETURN settings_id;
END;
$$;

-- Create function to get settings history
CREATE OR REPLACE FUNCTION get_platform_settings_history(
    history_limit INTEGER DEFAULT 10,
    history_offset INTEGER DEFAULT 0
)
RETURNS TABLE(
    id UUID,
    settings JSONB,
    updated_at TIMESTAMPTZ,
    updated_by VARCHAR(255)
)
LANGUAGE plpgsql
STABLE SECURITY DEFINER
AS $$
BEGIN
    -- Validate parameters
    IF history_limit <= 0 OR history_limit > 1000 THEN
        history_limit := 10;
    END IF;
    
    IF history_offset < 0 THEN
        history_offset := 0;
    END IF;
    
    -- Return settings history
    RETURN QUERY
    SELECT ps.id, ps.settings, ps.updated_at, ps.updated_by
    FROM platform_settings ps
    ORDER BY ps.updated_at DESC
    LIMIT history_limit
    OFFSET history_offset;
END;
$$;

-- Create function to reset to default settings
CREATE OR REPLACE FUNCTION reset_platform_settings(
    reset_by_user VARCHAR(255) DEFAULT 'system'
)
RETURNS UUID
LANGUAGE plpgsql
SECURITY DEFINER
AS $$
DECLARE
    default_settings JSONB;
    settings_id UUID;
BEGIN
    -- Define default platform settings
    default_settings := '{
        "platform": {
            "name": "PromptShield Platform",
            "description": "Enterprise LLM Security Gateway",
            "allow_self_signup": false,
            "require_approval": true,
            "default_trial_days": 14,
            "maintenance_mode": false
        },
        "defaults": {
            "tenant_limits": {
                "max_requests_per_day": 10000,
                "max_rulepacks": 10,
                "max_users": 5,
                "max_api_keys": 10,
                "max_retention_days": 30
            },
            "features": {
                "semantic_analysis_enabled": true,
                "audit_logging_enabled": true,
                "usage_tracking_enabled": true,
                "quota_management_enabled": false,
                "async_jobs_enabled": false
            }
        },
        "email": {
            "smtp_port": 587,
            "from_name": "PromptShield Platform",
            "enable_tls": true
        },
        "notifications": {
            "alert_thresholds": {
                "high_error_rate": 0.05,
                "high_latency_ms": 1000,
                "high_memory_usage_mb": 1024,
                "high_cpu_percent": 80,
                "disk_usage_percent": 85
            },
            "email_notifications": true,
            "notification_types": ["error", "warning", "maintenance"]
        },
        "security": {
            "session_timeout_hours": 24,
            "require_mfa": false,
            "rate_limit_rps": 100,
            "rate_limit_burst": 200,
            "max_failed_attempts": 5,
            "password_min_length": 8,
            "password_require_special": true
        }
    }'::JSONB;
    
    -- Use the update function to save default settings
    settings_id := update_platform_settings(default_settings, reset_by_user);
    
    RETURN settings_id;
END;
$$;

-- Insert default settings if none exist
DO $$
BEGIN
    -- Check if any settings exist
    IF NOT EXISTS (SELECT 1 FROM platform_settings LIMIT 1) THEN
        -- Insert default settings
        PERFORM reset_platform_settings('migration_init');
        RAISE NOTICE 'Default platform settings created';
    ELSE
        RAISE NOTICE 'Platform settings already exist, skipping default creation';
    END IF;
END;
$$;

-- Add comments for documentation
COMMENT ON TABLE platform_settings IS 'Platform-wide configuration settings for SaaS admin management';
COMMENT ON COLUMN platform_settings.id IS 'Unique identifier for each settings version';
COMMENT ON COLUMN platform_settings.settings IS 'JSONB storage for all platform configuration';
COMMENT ON COLUMN platform_settings.updated_at IS 'Timestamp when settings were last updated';
COMMENT ON COLUMN platform_settings.updated_by IS 'User or system that updated the settings';

COMMENT ON FUNCTION get_platform_settings() IS 'Retrieves the current platform settings';
COMMENT ON FUNCTION update_platform_settings(JSONB, VARCHAR) IS 'Updates platform settings with validation';
COMMENT ON FUNCTION get_platform_settings_history(INTEGER, INTEGER) IS 'Retrieves historical platform settings';
COMMENT ON FUNCTION reset_platform_settings(VARCHAR) IS 'Resets platform settings to defaults';

-- Grant permissions to authenticated users (RLS will handle authorization)
GRANT SELECT, INSERT, UPDATE, DELETE ON platform_settings TO authenticated;
GRANT EXECUTE ON FUNCTION get_platform_settings() TO authenticated;
GRANT EXECUTE ON FUNCTION update_platform_settings(JSONB, VARCHAR) TO authenticated;
GRANT EXECUTE ON FUNCTION get_platform_settings_history(INTEGER, INTEGER) TO authenticated;
GRANT EXECUTE ON FUNCTION reset_platform_settings(VARCHAR) TO authenticated;