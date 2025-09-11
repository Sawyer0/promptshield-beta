-- Aurora PostgreSQL required extensions
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS citext;

-- Optional: requires parameter group with shared_preload_libraries = 'pg_stat_statements'
DO $$
BEGIN
  CREATE EXTENSION IF NOT EXISTS pg_stat_statements;
EXCEPTION WHEN OTHERS THEN
  RAISE NOTICE 'pg_stat_statements not available or not preloaded; skipping';
END $$;

