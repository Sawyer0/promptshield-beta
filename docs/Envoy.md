### Envoy Integration (ext_authz + ext_proc)

This guide shows how to wire Envoy to PromptShield Enforcer (`promptshield-enforcer` aka `ps-enforcer`). Use `ext_authz` for fast header/context gating and `ext_proc` to stream bodies for content inspection. The gRPC server implements `envoy.service.ext_proc.v3.ExternalProcessor.Process` and performs scanner-backed decisions with budgets (timeout and max stream bytes).

#### Minimal ext_authz (HTTP service)

```yaml
static_resources:
  listeners:
  - name: app_listener
    address: { socket_address: { address: 0.0.0.0, port_value: 8080 } }
    filter_chains:
    - filters:
      - name: envoy.filters.network.http_connection_manager
        typed_config:
          "@type": type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager
          stat_prefix: ingress_http
          route_config: { name: local_route, virtual_hosts: [{ name: app, domains: ["*"], routes: [{ match: { prefix: "/" }, route: { cluster: app_backend } }]}]}
          http_filters:
          - name: envoy.filters.http.ext_authz
            typed_config:
              "@type": type.googleapis.com/envoy.extensions.filters.http.ext_authz.v3.ExtAuthz
              transport_api_version: V3
              http_service:
                server_uri:
                  uri: http://ps-enforcer:9090
                  cluster: ext_authz_cluster
                  timeout: 0.300s
                authorization_request:
                  allowed_headers:
                    patterns:
                      - exact: ":method"
                      - exact: ":path"
                      - exact: "authorization"
                      - exact: "x-tenant-id"
                authorization_response:
                  allowed_upstream_headers:
                    patterns: [{ exact: "x-ps-decision" }, { exact: "x-ps-reason" }]
          - name: envoy.filters.http.router
  clusters:
  - name: app_backend
    connect_timeout: 0.25s
    type: STATIC
    load_assignment: { cluster_name: app_backend, endpoints: [{ lb_endpoints: [{ endpoint: { address: { socket_address: { address: app, port_value: 9000 }}}}]}]}
  - name: ext_authz_cluster
    connect_timeout: 0.25s
    type: STATIC
    load_assignment: { cluster_name: ext_authz_cluster, endpoints: [{ lb_endpoints: [{ endpoint: { address: { socket_address: { address: ps-enforcer, port_value: 9090 }}}}]}]}
```

Note: `ext_authz` alone does not stream bodies. For response inspection, add the External Processing filter.

#### ext_proc for streaming response bodies

```yaml
  http_filters:
  - name: envoy.filters.http.ext_proc
    typed_config:
      "@type": type.googleapis.com/envoy.extensions.filters.http.ext_proc.v3.ExternalProcessor
      grpc_service:
        envoy_grpc: { cluster_name: ps_ext_proc }
       message_timeout: 0.300s
       processing_mode:
         request_header_mode: SEND
         response_header_mode: SEND
         response_body_mode: BUFFERED_PARTIAL  # STREAMED for large payloads
         request_body_mode: NONE
  - name: envoy.filters.http.router

clusters:
- name: ps_ext_proc
  type: STRICT_DNS
  connect_timeout: 0.25s
  lb_policy: ROUND_ROBIN
  http2_protocol_options: {}
  load_assignment:
    cluster_name: ps_ext_proc
    endpoints:
    - lb_endpoints:
      - endpoint: { address: { socket_address: { address: ps-enforcer, port_value: 9091 } } }
```

In this setup, `ps-enforcer` must expose a gRPC server on 9091 implementing `envoy.service.ext_proc.v3.ExternalProcessor` to process response bodies.

##### Response body mutation (redaction)

When rules specify `response.action: redact|mask`, PromptShield sends a `BodyResponse` with `CommonResponse.body_mutation.body` set to the redacted bytes. Envoy replaces the corresponding body chunk.

Reference: Envoy External Processing API — `CommonResponse.body_mutation` and `BodyMutation`.

Decision headers injected by enforcer:

```
x-ps-decision: allow|quarantine|deny
x-ps-reason:  <rule_id|timeout|body_limit>
```

#### Security

- Use mTLS between Envoy and `ps-enforcer`.
- Limit body sizes (Envoy buffer limits + enforcer `max_stream_bytes`).
- Enforce budgets/timeouts; enforcer terminates early on first threshold hit and sets decision headers.

#### Runtime tuning

- Enforcement modes via env: `PS_ENFORCER_ENFORCEMENT_MODE=observe|redact|quarantine|enforce`
- Enable body mutations for redaction: `PS_ENFORCER_REDACTION_MUTATION=true`
- Sliding window and overlap: `PS_ENFORCER_STREAM_WINDOW`, `PS_ENFORCER_STREAM_OVERLAP`
- Global concurrency cap: `PS_ENFORCER_MAX_STREAMS`
- Request rate limit: `PS_ENFORCER_RPS` and `PS_ENFORCER_RPS_BURST`
- Readiness and health: HTTP `/readyz`, `/healthz`; gRPC `grpc.health.v1`

#### Troubleshooting

- Start with `ext_authz` only to validate header flows.
- Add `ext_proc` later; test with small payloads, then increase limits.
- Use Prometheus/Grafana dashboards to watch p95 and escalation rates. Enforcer exposes Prometheus metrics for ext_proc: `ps_extproc_streams_total{decision}`, `ps_extproc_bytes_total`, `ps_extproc_stream_duration_seconds{decision}` (process-level registry).


