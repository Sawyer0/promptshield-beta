-- 0004_audit_merkle_roots.sql
-- Daily Merkle roots for immutable audit verification

CREATE TABLE audit_daily_roots (
  day DATE PRIMARY KEY,
  root TEXT NOT NULL,
  prev_root TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Row hash helper for audits
CREATE OR REPLACE FUNCTION audit_row_hash(a audits)
RETURNS TEXT AS $$
DECLARE h TEXT;
BEGIN
  h := encode(digest(
        convert_to(COALESCE(a.tenant_id::text,'') || '|' || COALESCE(a.action,'') || '|' ||
                    COALESCE(a.object_type,'') || '|' || COALESCE(a.object_id::text,'') || '|' ||
                    COALESCE(a.integrity_hash,'') || '|' || COALESCE(a.created_at::text,''), 'UTF8')
      ,'sha256'),'hex');
  RETURN h;
END; $$ LANGUAGE plpgsql IMMUTABLE;

-- Compute root for a given day; link prev_root
CREATE OR REPLACE FUNCTION compute_audit_root(p_day DATE)
RETURNS TEXT AS $$
DECLARE
  cur_root TEXT := '0';
  row_rec audits%ROWTYPE;
  prev TEXT;
BEGIN
  SELECT root INTO prev FROM audit_daily_roots WHERE day = (p_day - INTERVAL '1 day')::date;
  IF prev IS NULL THEN prev := '0'; END IF;

  FOR row_rec IN
    SELECT * FROM audits WHERE created_at >= p_day AND created_at < (p_day + INTERVAL '1 day')
    ORDER BY created_at, id
  LOOP
    cur_root := encode(digest(convert_to(cur_root || '|' || audit_row_hash(row_rec), 'UTF8'), 'sha256'),'hex');
  END LOOP;

  INSERT INTO audit_daily_roots(day, root, prev_root, created_at)
  VALUES (p_day, cur_root, prev, now())
  ON CONFLICT (day) DO UPDATE SET root = EXCLUDED.root, prev_root = EXCLUDED.prev_root, created_at = now();

  RETURN cur_root;
END; $$ LANGUAGE plpgsql;

-- Seal previous day daily root (run shortly after midnight)
CREATE OR REPLACE FUNCTION seal_yesterday_root()
RETURNS VOID AS $$
BEGIN
  PERFORM compute_audit_root((now() - INTERVAL '1 day')::date);
END; $$ LANGUAGE plpgsql;

-- Schedule daily at 00:10 if pg_cron is present
DO $$ BEGIN
  PERFORM cron.schedule('audit-daily-root','10 0 * * *','SELECT seal_yesterday_root();');
EXCEPTION WHEN undefined_table THEN NULL; WHEN undefined_function THEN NULL; END $$;

