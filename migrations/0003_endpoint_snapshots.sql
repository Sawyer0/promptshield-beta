-- 0003_endpoint_snapshots.sql
-- Materialized endpoint → rulepack mapping snapshots and refresh function

CREATE TABLE endpoint_rulepack_snapshots (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  method TEXT NOT NULL,                    -- GET|POST|ANY
  endpoint_template TEXT NOT NULL,         -- normalized template '/v1/tools/:id'
  rulepack_ids UUID[] NOT NULL,            -- ordered by priority
  generated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(tenant_id, method, endpoint_template)
);

-- Normalize method
CREATE OR REPLACE FUNCTION norm_method(m TEXT)
RETURNS TEXT AS $$
BEGIN
  IF m IS NULL OR upper(m) IN ('','*','ANY') THEN RETURN 'ANY'; END IF;
  RETURN upper(m);
END; $$ LANGUAGE plpgsql IMMUTABLE;

-- Simple path normalization in SQL (UUIDs, numeric IDs)
CREATE OR REPLACE FUNCTION normalize_path_template(p TEXT)
RETURNS TEXT AS $$
DECLARE s TEXT;
BEGIN
  IF p IS NULL OR p = '' THEN RETURN '/'; END IF;
  s := regexp_replace(p, '/[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}', '/:uuid', 'g');
  s := regexp_replace(s, '/[0-9]+', '/:id', 'g');
  RETURN s;
END; $$ LANGUAGE plpgsql IMMUTABLE;

-- Refresh function (full rebuild for tenant)
CREATE OR REPLACE FUNCTION refresh_endpoint_snapshots(p_tenant UUID)
RETURNS VOID AS $$
BEGIN
  DELETE FROM endpoint_rulepack_snapshots WHERE tenant_id = p_tenant;
  INSERT INTO endpoint_rulepack_snapshots (tenant_id, method, endpoint_template, rulepack_ids, generated_at)
  SELECT
    a.tenant_id,
    norm_method(a.method) AS method,
    normalize_path_template(a.target_scope) AS endpoint_template,
    ARRAY_AGG(a.rulepack_id ORDER BY a.priority DESC, a.updated_at ASC) AS rulepack_ids,
    now()
  FROM rulepack_assignments a
  JOIN rulepacks r ON r.id = a.rulepack_id AND r.is_active = true AND r.status = 'active'
  WHERE a.enabled = true AND a.tenant_id = p_tenant
  GROUP BY a.tenant_id, norm_method(a.method), normalize_path_template(a.target_scope);
END; $$ LANGUAGE plpgsql;

-- Convenience: refresh all tenants (lightweight)
CREATE OR REPLACE FUNCTION refresh_all_endpoint_snapshots()
RETURNS VOID AS $$
BEGIN
  PERFORM refresh_endpoint_snapshots(t.id) FROM tenants t WHERE t.status = 'active';
END; $$ LANGUAGE plpgsql;

-- Best-effort cron every 5 minutes (if pg_cron available)
DO $$ BEGIN
  PERFORM cron.schedule('endpoint-snapshots-5m','*/5 * * * *','SELECT refresh_all_endpoint_snapshots();');
EXCEPTION WHEN undefined_table THEN NULL; WHEN undefined_function THEN NULL; END $$;

