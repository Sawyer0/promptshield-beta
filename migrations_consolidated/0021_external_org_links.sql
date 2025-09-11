-- Map external identity provider organizations to internal tenants
-- Provider examples: 'clerk'
CREATE TABLE IF NOT EXISTS tenant_org_links (
  provider TEXT NOT NULL,
  external_org_id TEXT NOT NULL,
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY(provider, external_org_id)
);

CREATE INDEX IF NOT EXISTS idx_tenant_org_links_tenant ON tenant_org_links(tenant_id);

-- Optional RLS (follows same pattern as other tenant tables); disabled by default
-- DO $$ BEGIN
--   IF NOT EXISTS (
--     SELECT 1 FROM pg_policies WHERE schemaname='public' AND tablename='tenant_org_links' AND policyname='tenant_isolation_policy'
--   ) THEN
--     EXECUTE 'ALTER TABLE tenant_org_links ENABLE ROW LEVEL SECURITY';
--     EXECUTE 'CREATE POLICY tenant_isolation_policy ON tenant_org_links USING (tenant_id = get_current_tenant_id() OR is_platform_admin())';
--   END IF;
-- END $$;


