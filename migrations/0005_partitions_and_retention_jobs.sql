-- 0005_partitions_and_retention_jobs.sql
-- Helpers to manage monthly partitions and retention, with optional pg_cron scheduling

CREATE OR REPLACE FUNCTION create_next_month_partitions()
RETURNS VOID AS $$
DECLARE
  bases TEXT[] := ARRAY['violations','scan_results','audits','usage_metrics','performance_metrics'];
  base TEXT; s DATE := (date_trunc('month', now()) + INTERVAL '1 month')::date; e DATE := (s + INTERVAL '1 month')::date;
  p TEXT;
BEGIN
  FOREACH base IN ARRAY bases LOOP
    p := format('%s_%s', base, to_char(s,'YYYY_MM'));
    EXECUTE format('CREATE TABLE IF NOT EXISTS %I PARTITION OF %I FOR VALUES FROM (%L) TO (%L);', p, base, s, e);
  END LOOP;
END; $$ LANGUAGE plpgsql;

-- Schedule monthly on the 25th at 01:00 (so partitions exist ahead of time)
DO $$ BEGIN
  PERFORM cron.schedule('create-next-month-partitions','0 1 25 * *','SELECT create_next_month_partitions();');
EXCEPTION WHEN undefined_table THEN NULL; WHEN undefined_function THEN NULL; END $$;

-- Enhanced retention (example policy; tune per environment)
CREATE OR REPLACE FUNCTION enforce_retention_policies()
RETURNS VOID AS $$
BEGIN
  -- violations: keep 30 days
  DELETE FROM ONLY violations WHERE created_at < now() - INTERVAL '30 days';
  -- scan_results: keep 7 days
  DELETE FROM ONLY scan_results WHERE created_at < now() - INTERVAL '7 days';
  -- audits: keep 90 days hot (assuming warm/cold archival done externally)
  DELETE FROM ONLY audits WHERE created_at < now() - INTERVAL '90 days';
END; $$ LANGUAGE plpgsql;

DO $$ BEGIN
  PERFORM cron.schedule('retention-policies-daily','15 2 * * *','SELECT enforce_retention_policies();');
EXCEPTION WHEN undefined_table THEN NULL; WHEN undefined_function THEN NULL; END $$;

