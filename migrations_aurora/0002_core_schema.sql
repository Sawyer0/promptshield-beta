-- Core multi-tenant schema

-- Tenants
CREATE TABLE IF NOT EXISTS tenants (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL UNIQUE,
  status VARCHAR(50) DEFAULT 'active',
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_tenants_status ON tenants(status);

-- Rulepacks (policy containers)
CREATE TABLE IF NOT EXISTS rulepacks (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  description TEXT,
  yaml_content TEXT,
  rules JSONB,
  current_version_id UUID,
  is_active BOOLEAN DEFAULT true,
  status TEXT DEFAULT 'active' CHECK (status IN ('draft','active','archived')),
  enforcement_mode TEXT DEFAULT 'enforce' CHECK (enforcement_mode IN ('monitor','enforce','redact')),
  fail_on_severity TEXT DEFAULT 'HIGH' CHECK (fail_on_severity IN ('LOW','MEDIUM','HIGH','CRITICAL')),
  priority INTEGER DEFAULT 100,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Rulepack versions
CREATE TABLE IF NOT EXISTS rulepack_versions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  rulepack_id UUID NOT NULL REFERENCES rulepacks(id) ON DELETE CASCADE,
  version INT NOT NULL,
  dsl JSONB NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('draft','approved','active','archived')),
  created_by UUID,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(rulepack_id, version)
);

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint c
    JOIN pg_class t ON c.conrelid = t.oid
    WHERE c.conname = 'fk_current_version' AND t.relname = 'rulepacks'
  ) THEN
    ALTER TABLE rulepacks
      ADD CONSTRAINT fk_current_version
      FOREIGN KEY (current_version_id) REFERENCES rulepack_versions(id);
  END IF;
END $$;

-- Assignments
CREATE TABLE IF NOT EXISTS assignments (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  rulepack_id UUID NOT NULL REFERENCES rulepacks(id) ON DELETE CASCADE,
  target_scope TEXT NOT NULL,
  priority INT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(tenant_id, target_scope, rulepack_id)
);

-- Audits
CREATE TABLE IF NOT EXISTS audits (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  actor_id UUID,
  actor_email CITEXT,
  action TEXT NOT NULL,
  object_type TEXT NOT NULL,
  object_id UUID NOT NULL,
  diff JSONB,
  metadata JSONB,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_rulepack_versions_rulepack ON rulepack_versions(rulepack_id);
CREATE INDEX IF NOT EXISTS idx_rulepack_versions_status ON rulepack_versions(status);
CREATE INDEX IF NOT EXISTS idx_rulepack_versions_dsl_gin ON rulepack_versions USING GIN (dsl);
CREATE INDEX IF NOT EXISTS idx_rpv_rulepack_version_desc ON rulepack_versions (rulepack_id, version DESC);
CREATE INDEX IF NOT EXISTS idx_audits_tenant_ts ON audits (tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audits_object_ts ON audits (object_type, object_id, created_at DESC);

