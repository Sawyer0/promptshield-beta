-- RLS utilities and policies

-- Get current tenant ID from session context
CREATE OR REPLACE FUNCTION get_current_tenant_id()
RETURNS uuid
LANGUAGE plpgsql
STABLE SECURITY DEFINER
AS $$
DECLARE tenant_id uuid;
BEGIN
  BEGIN
    tenant_id := current_setting('app.current_tenant_id', true)::uuid;
  EXCEPTION WHEN others THEN
    tenant_id := NULL;
  END;
  RETURN tenant_id;
END;
$$;

-- Set tenant context (app must call per request/connection); returns boolean for app code
CREATE OR REPLACE FUNCTION set_tenant_context(tenant_uuid uuid)
RETURNS boolean
LANGUAGE plpgsql
SECURITY DEFINER
AS $$
BEGIN
  PERFORM set_config('app.current_tenant_id', tenant_uuid::text, false);
  RETURN true;
END;
$$;

-- Enable RLS on tenant-scoped tables (do NOT enable on platform/global tables)
ALTER TABLE IF EXISTS assignments ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS audits ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS rulepacks ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS rulepack_versions ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS users ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS subscriptions ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS api_keys ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS usage_metrics ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS violations ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS threat_intelligence ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS alerts ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS webhooks ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS tenant_settings ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS feature_flags ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS tenant_services ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS service_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS service_metrics ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS tools ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS tenant_memberships ENABLE ROW LEVEL SECURITY;

-- Standard tenant isolation policy for tables with tenant_id
DO $$
DECLARE tbl TEXT;
BEGIN
  FOREACH tbl IN ARRAY ARRAY[
    'assignments','audits','rulepacks','users','subscriptions','api_keys','sessions',
    'usage_metrics','violations','alerts','webhooks','tenant_settings','feature_flags',
    'tenant_services','service_events','service_metrics','tools','tenant_memberships'
  ]
  LOOP
    EXECUTE format('CREATE POLICY %I_tenant_isolation ON %I FOR ALL USING (tenant_id = get_current_tenant_id())', tbl, tbl);
  END LOOP;
END $$;

-- Special policy for rulepack_versions (no tenant_id column): link via rulepacks
CREATE POLICY rulepack_versions_tenant_isolation ON rulepack_versions
  FOR ALL USING (
    EXISTS (
      SELECT 1 FROM rulepacks rp
      WHERE rp.id = rulepack_versions.rulepack_id
        AND rp.tenant_id = get_current_tenant_id()
    )
  );

-- Special policy for threat_intelligence: allow global rows (tenant_id IS NULL)
CREATE POLICY threat_intel_policy ON threat_intelligence
  FOR ALL USING (
    tenant_id IS NULL OR tenant_id = get_current_tenant_id()
  );

-- Helpful tenant_id indexes for RLS filtering
CREATE INDEX IF NOT EXISTS idx_assignments_tenant_rls ON assignments(tenant_id);
CREATE INDEX IF NOT EXISTS idx_audits_tenant_rls ON audits(tenant_id);
CREATE INDEX IF NOT EXISTS idx_rulepacks_tenant_rls ON rulepacks(tenant_id);
CREATE INDEX IF NOT EXISTS idx_users_tenant_rls ON users(tenant_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_tenant_rls ON subscriptions(tenant_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_tenant_rls ON api_keys(tenant_id);
CREATE INDEX IF NOT EXISTS idx_sessions_tenant_rls ON sessions(tenant_id);
CREATE INDEX IF NOT EXISTS idx_usage_metrics_tenant_rls ON usage_metrics(tenant_id);
CREATE INDEX IF NOT EXISTS idx_violations_tenant_rls ON violations(tenant_id);
CREATE INDEX IF NOT EXISTS idx_alerts_tenant_rls ON alerts(tenant_id);
CREATE INDEX IF NOT EXISTS idx_webhooks_tenant_rls ON webhooks(tenant_id);
CREATE INDEX IF NOT EXISTS idx_tenant_settings_tenant_rls ON tenant_settings(tenant_id);
CREATE INDEX IF NOT EXISTS idx_feature_flags_tenant_rls ON feature_flags(tenant_id);
CREATE INDEX IF NOT EXISTS idx_tenant_services_tenant_rls ON tenant_services(tenant_id);
CREATE INDEX IF NOT EXISTS idx_service_events_tenant_rls ON service_events(tenant_id);
CREATE INDEX IF NOT EXISTS idx_service_metrics_tenant_rls ON service_metrics(tenant_id);
CREATE INDEX IF NOT EXISTS idx_tools_tenant_rls ON tools(tenant_id);
CREATE INDEX IF NOT EXISTS idx_tenant_memberships_tenant_rls ON tenant_memberships(tenant_id);

