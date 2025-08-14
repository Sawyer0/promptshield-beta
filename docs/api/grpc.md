# gRPC API — Envoy External Processor (ext_proc)

The enforcer implements Envoy's External Processor API (v3):

- Service: `envoy.service.ext_proc.v3.ExternalProcessor`
- Method: `rpc Process(stream ProcessingRequest) returns (stream ProcessingResponse)`

Reference: Envoy docs — `envoy.extensions.filters.http.ext_proc.v3`.

## Endpoint

- Address: `PS_ENFORCER_GRPC_ADDR` (default `:9091`)
- TLS/mTLS:
  - `PS_ENFORCER_GRPC_TLS_CERT`, `PS_ENFORCER_GRPC_TLS_KEY` enable TLS
  - `PS_ENFORCER_GRPC_TLS_CLIENT_CA` enables and enforces mTLS

## Processing Flow

- Request headers → continue
- Request body chunks (buffered/streamed) → scan incrementally using a sliding window
- Response headers/body → optional scan with redaction support
- Immediate quarantine on threshold hit (ImmediateResponse)
- Decision headers injected (`x-ps-decision`, `x-ps-reason`)

## Body Mutation (Redaction)

When response rules specify `response.action: redact|mask`, the enforcer may mutate body chunks using ext_proc `BodyResponse` with `CommonResponse.body_mutation` set to the replacement body. Enable via `PS_ENFORCER_REDACTION_MUTATION=true` and set desired `PS_ENFORCER_ENFORCEMENT_MODE`.

- Non-streamed replacement per chunk:
  - Set `CommonResponse.body_mutation = BodyMutation{ body: <redacted bytes> }`
  - Ext Proc reference: `CommonResponse.body_mutation` and `BodyMutation.body` in Envoy docs.
- For FULL_DUPLEX_STREAMED, use `BodyMutation.streamed_response`; current implementation uses the non-streamed `body` variant.

Reference: Envoy External Processing API — BodyResponse/CommonResponse/BodyMutation [link](https://www.envoyproxy.io/docs/envoy/latest/api-v3/service/ext_proc/v3/external_processor.proto.html)

## Metrics

Prometheus counters/histograms exported by the process:

- `ps_extproc_streams_total{decision}`
- `ps_extproc_bytes_total`
- `ps_extproc_stream_duration_seconds{decision}`

## Streaming, Backpressure, and Limits

- Sliding window bounds memory per stream: `PS_ENFORCER_STREAM_WINDOW`, overlap `PS_ENFORCER_STREAM_OVERLAP`
- Global concurrency cap: `PS_ENFORCER_MAX_STREAMS`
- Global rate limiting: `PS_ENFORCER_RPS` and `PS_ENFORCER_RPS_BURST`
- Enforcement modes: `PS_ENFORCER_ENFORCEMENT_MODE=observe|redact|quarantine|enforce`
- Health service: `grpc.health.v1.Health` registered and reflects ext_proc readiness

## Timeouts and Limits

- Per-request timeout: default 300ms (`PS_ENFORCER_TIMEOUT`)
- Max stream bytes: default 5,000,000 (`PS_ENFORCER_MAX_STREAM_BYTES`)
- Fail-on severity threshold: default `HIGH` (`PS_ENFORCER_FAIL_ON`)
