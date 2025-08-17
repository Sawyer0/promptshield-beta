# PromptShield Gateway (Minimal)

A minimal, zero-CLI gateway focused on Envoy integration.

- gRPC: Envoy ext_proc ExternalProcessor on :9091
- HTTP: /healthz, /readyz, /metrics, /check on :9090
- No CLI, no flags. Defaults are baked in.

For local testing with Envoy, reuse the repository `envoy-config.yaml` and `docker-compose.yaml`.

### Benchmarks & SLAs

- HTTP benchmark (64KB body):
```bash
go test -run=^$ ./gateway -bench BenchmarkGatewayHTTPCheck64KB -benchmem -count=1
```

- gRPC ext_proc benchmark:
```bash
go test -run=^$ ./gateway -bench BenchmarkGatewayGRPCExtProc_SetupOnly -benchmem -count=1
```

- SLA tests (opt‑in):
```bash
PS_ENFORCE_SLA=1 go test ./gateway -run TestHTTPCheck_SLA -count=1
PS_ENFORCE_SLA=1 go test ./gateway -run TestGRPCExtProc_SLA -count=1
```

See `../docs/Performance.md` for full details and scanner benchmarks.