# PromptShield Enforcer API Documentation

This directory contains the API documentation for the PromptShield Enforcer (ps-enforcer).

- HTTP API (OpenAPI): `openapi.yaml`
- gRPC API (Envoy External Processor): `grpc.md`
- Metrics (Prometheus): `metrics.md`
- Security (mTLS, tokens): `security.md`

Quick links:
- Deployment and Envoy wiring: see `docs/ENVOY_INTEGRATION.md`
- Runtime architecture: see `docs/Runtime-Architecture.md`

## Overview

The enforcer exposes:
- HTTP endpoints under `/v1` for health, version, decisions (`/check`), rulepacks/config/admin, events, stats, usage, and Prometheus metrics at `/metrics`.
  - `/check` supports single payloads, aggregate JSON arrays, and NDJSON streaming (aggregate=false).
- gRPC service implementing Envoy's External Processing API (streaming) for body inspection and mutations.
- Prometheus metrics for observability and SLOs.

## OpenAPI preview

Render the OpenAPI spec locally (requires `redoc-cli` or `openapi-generator`):

```bash
npx redoc-cli serve docs/api/openapi.yaml --watch
```

## Compatibility
- The HTTP API is stable for `/v1/healthz`, `/metrics`, `/v1/version`, and `/v1/check` (including aggregate and NDJSON modes), plus associated admin endpoints.
- The gRPC API leverages Envoy's `ext_proc` v3 interface and follows its contract.
