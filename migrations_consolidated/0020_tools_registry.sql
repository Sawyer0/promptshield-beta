-- Tools registry table: capability-based tool metadata per tenant
CREATE TABLE IF NOT EXISTS tools (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  tool_id TEXT NOT NULL,
  name TEXT NOT NULL,
  description TEXT DEFAULT '',
  capability_tags JSONB NOT NULL DEFAULT '[]'::jsonb,
  data_domains JSONB NOT NULL DEFAULT '[]'::jsonb,
  side_effect TEXT NOT NULL DEFAULT 'none', -- none | reversible | irreversible
  auth_scope TEXT NOT NULL DEFAULT 'user-delegated', -- user-delegated | service-account
  arg_schema JSONB DEFAULT NULL,
  risk_score INT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_tools_tenant_toolid ON tools(tenant_id, tool_id);
CREATE INDEX IF NOT EXISTS idx_tools_tenant ON tools(tenant_id);

-- Optional RLS for multi-tenant isolation
DO $$ BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_policies WHERE schemaname = 'public' AND tablename = 'tools'
  ) THEN
    ALTER TABLE tools ENABLE ROW LEVEL SECURITY;
    CREATE POLICY tools_tenant_isolation ON tools
      USING (tenant_id::text = current_setting('app.tenant_id', true));
  END IF;
END $$;


