# Metrics Catalog (Prometheus)

This document catalogs the Prometheus metrics emitted by the enforcer.

## Enforcer (HTTP endpoints)

- `ps_enforcer_requests_total{path,code}` — Total HTTP requests by path and HTTP status
- `ps_enforcer_decisions_total{decision}` — Total decisions made (allow|quarantine|deny)
- `ps_enforcer_request_duration_seconds_bucket` — Request duration histogram buckets
- `ps_enforcer_request_duration_seconds_sum`
- `ps_enforcer_request_duration_seconds_count`
- `ps_http_bytes_total{direction,path}` — Total HTTP bytes in/out by path (captured at API router)
- `ps_scan_request_duration_seconds_bucket{mode}` — `/check` aggregate/NDJSON processing duration by mode (aggregate|ndjson)
- `ps_scan_events_total{route}` — NDJSON decision events emitted per route

Example PromQL:
- Rate by decision: `sum by (decision) (rate(ps_enforcer_decisions_total[5m]))`
- p95 latency: `histogram_quantile(0.95, sum(rate(ps_enforcer_request_duration_seconds_bucket[5m])) by (le))`

## Enforcer (gRPC ext_proc)

- `ps_extproc_streams_total{decision}` — Total gRPC ext_proc streams by decision
- `ps_extproc_bytes_total` — Bytes observed across streams
- `ps_extproc_stream_duration_seconds_bucket` — Stream duration histogram buckets
- `ps_extproc_stream_duration_seconds_sum`
- `ps_extproc_stream_duration_seconds_count`

Example PromQL:
- Streams rate: `sum by (decision) (rate(ps_extproc_streams_total[5m]))`
- p95 stream duration: `histogram_quantile(0.95, sum(rate(ps_extproc_stream_duration_seconds_bucket[5m])) by (le))`

## Deprecated

Historical CLI scan metrics have been removed from the Gateway distribution.

## Grafana

A ready-to-import Grafana dashboard is available at `monitoring/dashboards/promptshield-enforcer.json`.
