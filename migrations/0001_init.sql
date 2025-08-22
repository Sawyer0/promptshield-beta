-- PromptShield control-plane initial schema
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE tenants (
  id UUID PRIMARY KEY,
  name TEXT NOT NULL UNIQUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE rulepacks (
  id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  description TEXT,
  current_version_id UUID,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE rulepack_versions (
  id UUID PRIMARY KEY,
  rulepack_id UUID NOT NULL REFERENCES rulepacks(id) ON DELETE CASCADE,
  version INT NOT NULL,
  dsl JSONB NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('draft','approved','active','archived')),
  created_by UUID,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(rulepack_id, version)
);

ALTER TABLE rulepacks
  ADD CONSTRAINT fk_current_version
  FOREIGN KEY (current_version_id) REFERENCES rulepack_versions(id);

CREATE TABLE assignments (
  id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  rulepack_id UUID NOT NULL REFERENCES rulepacks(id) ON DELETE CASCADE,
  target_scope TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE audits (
  id UUID PRIMARY KEY,
  tenant_id UUID,
  actor_id UUID,
  action TEXT NOT NULL,
  object_type TEXT NOT NULL,
  object_id UUID NOT NULL,
  diff JSONB,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_rulepack_versions_rulepack ON rulepack_versions(rulepack_id);
CREATE INDEX idx_rulepack_versions_status ON rulepack_versions(status);
CREATE INDEX idx_rulepack_versions_dsl_gin ON rulepack_versions USING GIN (dsl);