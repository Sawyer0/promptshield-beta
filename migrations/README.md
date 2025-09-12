# Migrations (Clean Architecture)

This directory contains a clean, end-to-end schema aligned with DATABASE_ARCHITECTURE.md. It is designed for a fresh database (no data). If you previously had migrations under `migrations_consolidated/` or `migrations_aurora/`, they were removed as part of this reset.

## How to apply

- Postgres/Aurora (psql):

```
psql "$PS_PG_DSN" -v ON_ERROR_STOP=1 -f migrations/0001_init_schema.sql
```

- The script is idempotent enough to run once on an empty database. It creates:
  - Core tables: tenants, users, platform_users, tenant_memberships, platform_settings
  - Rulepacks + versions, rulepack_assignments
  - Tools, tenant_settings
  - Partitioned ops tables: violations, scan_results, audits (+ current/next month partitions)
  - Analytics: usage_metrics, performance_metrics (partitioned)
  - Compliance: compliance_evidence
  - Indexes/partial indexes for hot paths
  - RLS policies and tenant context helpers (set_tenant_context, get_current_tenant_id, is_platform_admin)
  - Optional pg_cron schedule for retention (best effort)

## Tenant context

At connection/session start in the app, set:

```
SELECT set_tenant_context('00000000-0000-0000-0000-000000000001'::uuid);
```

Platform/admin actions may set:

```
SELECT set_platform_admin(true);
```

## Notes

- Time-partitioned tables use monthly partitions and BRIN for efficient scans.
- Rulepack JSONB and YAML are stored together; content-addressed blobs can be added later.
- Adjust retention policies and pg_cron usage per environment.
