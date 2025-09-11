-- Tools registry per tenant

CREATE TABLE IF NOT EXISTS tools (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  tool_id TEXT NOT NULL,
  name TEXT NOT NULL,
  description TEXT DEFAULT '',
  capability_tags JSONB NOT NULL DEFAULT '[]'::jsonb,
  data_domains JSONB NOT NULL DEFAULT '[]'::jsonb,
  side_effect TEXT NOT NULL DEFAULT 'none' CHECK (side_effect IN ('none','reversible','irreversible')),
  auth_scope TEXT NOT NULL DEFAULT 'user-delegated' CHECK (auth_scope IN ('user-delegated','service-account')),
  arg_schema JSONB DEFAULT NULL,
  risk_score INT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_tools_tenant_toolid ON tools(tenant_id, tool_id);
CREATE INDEX IF NOT EXISTS idx_tools_tenant ON tools(tenant_id);

