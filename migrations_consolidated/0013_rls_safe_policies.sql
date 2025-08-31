-- Idempotent RLS helper functions and policies creation
-- Safe to run multiple times on an existing database.

-- =============================
-- Helper functions (create/replace)
-- =============================

CREATE OR REPLACE FUNCTION get_current_tenant_id()
RETURNS UUID
LANGUAGE plpgsql
STABLE SECURITY DEFINER
AS $$
DECLARE
    tenant_id UUID;
BEGIN
    -- Try to get tenant_id from JWT claims (if middleware sets it)
    BEGIN
        SELECT (current_setting('request.jwt.claims', true)::json->>'tenant_id')::uuid INTO tenant_id;
    EXCEPTION
        WHEN OTHERS THEN
            tenant_id := NULL;
    END;

    -- Fallback: session variable set by app
    IF tenant_id IS NULL THEN
        BEGIN
            tenant_id := current_setting('app.current_tenant_id')::uuid;
        EXCEPTION
            WHEN OTHERS THEN
                tenant_id := NULL;
        END;
    END IF;
    RETURN tenant_id;
END;
$$;

CREATE OR REPLACE FUNCTION is_platform_admin()
RETURNS BOOLEAN
LANGUAGE plpgsql
STABLE SECURITY DEFINER
AS $$
DECLARE
    is_admin BOOLEAN := FALSE;
    user_email TEXT;
BEGIN
    BEGIN
        SELECT (current_setting('request.jwt.claims', true)::json->>'email') INTO user_email;
    EXCEPTION WHEN OTHERS THEN
        user_email := NULL;
    END;

    IF user_email IS NOT NULL THEN
        -- Optional admin_users table check (best-effort)
        BEGIN
            PERFORM 1 FROM admin_users WHERE email = user_email AND is_active = true;
            IF FOUND THEN
                is_admin := TRUE;
            END IF;
        EXCEPTION WHEN OTHERS THEN
            is_admin := FALSE;
        END;
    END IF;

    -- Allow admin access when no tenant context set (operational maintenance)
    IF NOT is_admin AND get_current_tenant_id() IS NULL THEN
        is_admin := TRUE;
    END IF;

    RETURN is_admin;
END;
$$;

CREATE OR REPLACE FUNCTION set_tenant_context(tenant_uuid UUID)
RETURNS VOID
LANGUAGE plpgsql
SECURITY DEFINER
AS $$
BEGIN
    PERFORM set_config('app.current_tenant_id', tenant_uuid::text, false);
END;
$$;

CREATE OR REPLACE FUNCTION validate_tenant_access(tenant_uuid UUID)
RETURNS BOOLEAN
LANGUAGE plpgsql
STABLE SECURITY DEFINER
AS $$
BEGIN
    IF is_platform_admin() THEN
        RETURN TRUE;
    END IF;
    RETURN tenant_uuid = get_current_tenant_id();
END;
$$;

-- Optionally grant execute to application role when it exists
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'promptshield_api') THEN
    GRANT EXECUTE ON FUNCTION get_current_tenant_id() TO promptshield_api;
    GRANT EXECUTE ON FUNCTION is_platform_admin() TO promptshield_api;
    GRANT EXECUTE ON FUNCTION set_tenant_context(UUID) TO promptshield_api;
    GRANT EXECUTE ON FUNCTION validate_tenant_access(UUID) TO promptshield_api;
  END IF;
END$$;

-- =============================
-- Enable RLS on known tables (if they exist)
-- =============================
DO $$ BEGIN
  PERFORM 1 FROM information_schema.tables WHERE table_schema='public' AND table_name='assignments';
  IF FOUND THEN EXECUTE 'ALTER TABLE assignments ENABLE ROW LEVEL SECURITY'; END IF;

  PERFORM 1 FROM information_schema.tables WHERE table_schema='public' AND table_name='audits';
  IF FOUND THEN EXECUTE 'ALTER TABLE audits ENABLE ROW LEVEL SECURITY'; END IF;

  PERFORM 1 FROM information_schema.tables WHERE table_schema='public' AND table_name='rulepacks';
  IF FOUND THEN EXECUTE 'ALTER TABLE rulepacks ENABLE ROW LEVEL SECURITY'; END IF;

  PERFORM 1 FROM information_schema.tables WHERE table_schema='public' AND table_name='rulepack_assignments';
  IF FOUND THEN EXECUTE 'ALTER TABLE rulepack_assignments ENABLE ROW LEVEL SECURITY'; END IF;
END $$;

-- =============================
-- Create tenant isolation policies (idempotent)
-- =============================

-- Helper to create a standard tenant_isolation_policy if not exists on a table
DO $$
DECLARE
  tbl TEXT;
BEGIN
  FOREACH tbl IN ARRAY ARRAY[
    'assignments',
    'audits',
    'rulepacks',
    'rulepack_assignments'
  ] LOOP
    IF EXISTS (
      SELECT 1 FROM information_schema.tables WHERE table_schema='public' AND table_name=tbl
    ) THEN
      IF NOT EXISTS (
        SELECT 1 FROM pg_policies WHERE schemaname='public' AND tablename=tbl AND policyname='tenant_isolation_policy'
      ) THEN
        EXECUTE format('CREATE POLICY tenant_isolation_policy ON %I FOR ALL USING (tenant_id = get_current_tenant_id() OR is_platform_admin())', tbl);
      END IF;
    END IF;
  END LOOP;
END$$;

-- Tenants table: policy differs
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.tables WHERE table_schema='public' AND table_name='tenants'
  ) THEN
    -- Enable RLS
    EXECUTE 'ALTER TABLE tenants ENABLE ROW LEVEL SECURITY';
    -- Create policy if missing
    IF NOT EXISTS (
      SELECT 1 FROM pg_policies WHERE schemaname='public' AND tablename='tenants' AND policyname='tenant_access_policy'
    ) THEN
      EXECUTE 'CREATE POLICY tenant_access_policy ON tenants FOR ALL USING (id = get_current_tenant_id() OR is_platform_admin())';
    END IF;
  END IF;
END$$;

