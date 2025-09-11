# Aurora PostgreSQL Migrations

These are clean, ordered migrations for Aurora PostgreSQL (recommended: Serverless v2). They replace prior inconsistent migrations and implement a consistent multi-tenant schema with RLS.

## Order

1. 0001_extensions.sql
2. 0002_core_schema.sql
3. 0003_production_tables.sql
4. 0004_services.sql
5. 0005_directory_orgs.sql
6. 0007_tools_registry.sql
7. 0008_rls.sql

## Apply (Go runner)

Set your DSN as an environment variable, then run the Go script:

- PowerShell (Windows):
  - $env:AURORA_PG_DSN = "postgres://{user}:{password}@{cluster-writer-endpoint}:5432/{db}?sslmode=require"
  - go run scripts/run-aurora-migrations.go

- Bash (macOS/Linux):
  - export AURORA_PG_DSN="postgres://{user}:{password}@{cluster-writer-endpoint}:5432/{db}?sslmode=require"
  - go run scripts/run-aurora-migrations.go

Optionally set MIGRATIONS_DIR to override the default (migrations_aurora).

## Notes

- Extensions: pgcrypto, citext are enabled; pg_stat_statements is attempted but requires a parameter group with shared_preload_libraries.
- RLS: enforced via get_current_tenant_id() using app.current_tenant_id; your app must call SELECT set_tenant_context($1::uuid) per request/connection.
- Tenants: RLS is not enabled on tenants to allow validation before tenant context is set in middleware.
- Time-series: service_metrics is partitioned monthly; the migration creates partitions for current and next month.
- Emails: CITEXT ensures case-insensitive uniqueness (users, admin_users, platform_users).

