-- 0008_snapshot_misses.sql
-- Track endpoints that did not match any snapshot mapping

CREATE TABLE endpoint_snapshot_misses (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  method TEXT NOT NULL,
  endpoint TEXT NOT NULL,
  template TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_snapshot_misses_tenant_time ON endpoint_snapshot_misses (tenant_id, created_at DESC);
CREATE INDEX idx_snapshot_misses_tenant_tpl ON endpoint_snapshot_misses (tenant_id, template);

