# PromptShield Gateway (Minimal)

A minimal, zero-CLI gateway focused on Envoy integration.

- gRPC: Envoy ext_proc ExternalProcessor on :9091
- HTTP: /healthz, /readyz, /metrics, /check on :9090
- No CLI, no flags. Defaults are baked in.

For local testing with Envoy, reuse the repository `envoy-config.yaml` and `docker-compose.yaml`.