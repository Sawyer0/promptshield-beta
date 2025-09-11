-- Platform directory and org mapping

CREATE TABLE IF NOT EXISTS platform_users (
  id TEXT PRIMARY KEY,
  email CITEXT,
  first_name TEXT,
  last_name TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX IF NOT EXISTS ux_platform_users_email ON platform_users(email);

CREATE TABLE IF NOT EXISTS tenant_memberships (
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  user_id   TEXT NOT NULL,
  role      TEXT NOT NULL DEFAULT 'member',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  deleted_at TIMESTAMPTZ,
  PRIMARY KEY (tenant_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_tenant_memberships_user ON tenant_memberships(user_id);
CREATE INDEX IF NOT EXISTS idx_tenant_memberships_tenant ON tenant_memberships(tenant_id);

CREATE TABLE IF NOT EXISTS tenant_org_links (
  provider TEXT NOT NULL,
  external_org_id TEXT NOT NULL,
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY(provider, external_org_id)
);
CREATE INDEX IF NOT EXISTS idx_tenant_org_links_tenant ON tenant_org_links(tenant_id);

