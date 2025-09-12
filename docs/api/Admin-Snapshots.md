# Admin Snapshots API

These endpoints help inspect and manage the materialized endpoint → rulepack mapping and snapshot misses.

All routes require admin authorization and run on the Gateway (Go) API.

## List Snapshots

- GET `/v1/admin/tenants/{tenant_id}/snapshots`

Response:
```
{
  "tenant_id": "...",
  "count": 2,
  "snapshots": [
    {
      "method": "POST",
      "endpoint_template": "/v1/tools/:id",
      "rulepack_ids": ["...uuid...","...uuid..."],
      "generated_at": "2025-09-12T10:00:00Z"
    }
  ]
}
```

## Refresh Snapshots

- POST `/v1/admin/tenants/{tenant_id}/snapshots/refresh`

Triggers a rebuild using current assignments and active rulepacks.

Response:
```
{ "status": "ok", "tenant_id": "..." }
```

## Snapshot Misses (Aggregated or Raw)

- GET `/v1/admin/tenants/{tenant_id}/snapshots/misses?from=RFC3339&to=RFC3339&raw=true|false`

Defaults: last 24h, `raw=false` (aggregated).

Aggregated response (`raw=false`):
```
{
  "tenant_id": "...",
  "from": "2025-09-11T10:00:00Z",
  "to": "2025-09-12T10:00:00Z",
  "raw": false,
  "count": 1,
  "misses": [
    { "method": "POST", "template": "/v1/tools/:id", "misses": 42, "first_seen": "...", "last_seen": "..." }
  ]
}
```

Raw response (`raw=true`):
```
{
  "tenant_id": "...",
  "raw": true,
  "misses": [ { "method": "POST", "endpoint": "/v1/tools/123", "template": "/v1/tools/:id", "created_at": "..." } ],
  "count": 12
}
```

## Seal Yesterday's Audit Root

- POST `/v1/admin/system/audits/seal`

Computes and stores the Merkle root for yesterday’s `audits` rows.

Response:
```
{ "status": "ok", "sealed": "yesterday" }
```

## Notes

- Snapshots are auto-refreshed by a DB trigger when assignments change.
- A periodic `refresh_all_endpoint_snapshots()` can be scheduled via pg_cron if desired.
- Misses are best-effort logging to identify routes that need patterns or template adjustments.
