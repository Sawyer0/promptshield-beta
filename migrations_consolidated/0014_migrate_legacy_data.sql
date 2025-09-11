-- Migrate legacy tables into canonical schema (non-destructive, idempotent-ish)
-- 1) assignments -> rulepack_assignments

DO $$
DECLARE
  has_priority BOOLEAN;
  has_updated  BOOLEAN;
  has_enabled  BOOLEAN;
  has_rulepack BOOLEAN;
  has_policy   BOOLEAN;
  src_rulepack_col TEXT;
  sql TEXT;
BEGIN
  IF to_regclass('public.assignments') IS NULL THEN
    RETURN;
  END IF;

  SELECT EXISTS (
    SELECT 1 FROM information_schema.columns WHERE table_schema='public' AND table_name='assignments' AND column_name='priority'
  ) INTO has_priority;
  SELECT EXISTS (
    SELECT 1 FROM information_schema.columns WHERE table_schema='public' AND table_name='assignments' AND column_name='updated_at'
  ) INTO has_updated;
  SELECT EXISTS (
    SELECT 1 FROM information_schema.columns WHERE table_schema='public' AND table_name='assignments' AND column_name='enabled'
  ) INTO has_enabled;
  SELECT EXISTS (
    SELECT 1 FROM information_schema.columns WHERE table_schema='public' AND table_name='assignments' AND column_name='rulepack_id'
  ) INTO has_rulepack;
  SELECT EXISTS (
    SELECT 1 FROM information_schema.columns WHERE table_schema='public' AND table_name='assignments' AND column_name='policy_id'
  ) INTO has_policy;

  IF has_rulepack THEN
    src_rulepack_col := 'rulepack_id';
  ELSIF has_policy THEN
    src_rulepack_col := 'policy_id';
  ELSE
    RAISE NOTICE 'assignments table lacks rulepack_id/policy_id, skipping migration';
    RETURN;
  END IF;

  sql := 'INSERT INTO rulepack_assignments (id, tenant_id, rulepack_id, target_scope, priority, enabled, created_at, updated_at) ' ||
         'SELECT a.id, a.tenant_id, a.'||src_rulepack_col||', a.target_scope, ' ||
         CASE WHEN has_priority THEN 'COALESCE(a.priority,100)' ELSE '100' END || ', ' ||
         CASE WHEN has_enabled  THEN 'COALESCE(a.enabled,true)'  ELSE 'true' END || ', ' ||
         'a.created_at, ' ||
         CASE WHEN has_updated  THEN 'COALESCE(a.updated_at,NOW())' ELSE 'NOW()' END ||
         ' FROM assignments a ON CONFLICT (tenant_id, target_scope, rulepack_id) DO NOTHING';
  EXECUTE sql;
END$$;

-- 2) audit_events -> audits
DO $$
BEGIN
  IF to_regclass('public.audit_events') IS NOT NULL THEN
    INSERT INTO audits (tenant_id, actor_id, actor_email, action, object_type, object_id, metadata, created_at)
    SELECT 
      CASE WHEN ae.tenant_id ~ '^[0-9a-fA-F-]{36}$' THEN ae.tenant_id::uuid ELSE NULL END,
      CASE WHEN ae.user_id   ~ '^[0-9a-fA-F-]{36}$' THEN ae.user_id::uuid ELSE NULL END,
      ae.user_name,
      COALESCE(ae.action, ae.event_type),
      ae.resource_type,
      CASE WHEN ae.resource_id ~ '^[0-9a-fA-F-]{36}$' THEN ae.resource_id::uuid ELSE NULL END,
      COALESCE(ae.details, '{}'::jsonb) || jsonb_build_object('migrated_from','audit_events'),
      COALESCE(ae."timestamp", NOW())
    FROM audit_events ae
    WHERE NOT EXISTS (
      SELECT 1 FROM audits x
      WHERE x.action = COALESCE(ae.action, ae.event_type)
        AND x.object_type = ae.resource_type
        AND x.created_at = COALESCE(ae."timestamp", NOW())
    );
  END IF;
END$$;
