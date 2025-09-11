-- Drop legacy/duplicate tables now that data has been migrated to canonical schema
-- WARNING: This removes old tables in the public schema only.

DO $$
BEGIN
  -- Legacy policy assignments table (replaced by rulepack_assignments)
  IF to_regclass('public.policy_assignments') IS NOT NULL THEN
    EXECUTE 'DROP TABLE IF EXISTS public.policy_assignments CASCADE';
  END IF;

  -- Older assignments table (superseded by rulepack_assignments)
  IF to_regclass('public.assignments') IS NOT NULL THEN
    EXECUTE 'DROP TABLE IF EXISTS public.assignments CASCADE';
  END IF;

  -- Legacy audit log table (superseded by audits)
  IF to_regclass('public.audit_events') IS NOT NULL THEN
    EXECUTE 'DROP TABLE IF EXISTS public.audit_events CASCADE';
  END IF;

  -- Early/typo table name (rule_packs)
  IF to_regclass('public.rule_packs') IS NOT NULL THEN
    EXECUTE 'DROP TABLE IF EXISTS public.rule_packs CASCADE';
  END IF;

  -- Legacy sessions table (the app uses session_store via connect-pg-simple)
  IF to_regclass('public.sessions') IS NOT NULL THEN
    EXECUTE 'DROP TABLE IF EXISTS public.sessions CASCADE';
  END IF;
END$$;

