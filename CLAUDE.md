# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

PromptShield is an Envoy-integrated LLM API Gateway with enterprise-grade runtime enforcement. It uses a progressive three-tier rule system: Level 1 (keyword matching), Level 2 (regex patterns), and Level 3 (semantic/LLM analysis, opt-in).

## Essential Commands

### Build and Development
```bash
make build           # Build ps-gateway binary (enforcer with HTTP + gRPC ext_proc)
make build-enforcer  # Legacy alias for compatibility
make test            # Run all tests  
make fmt             # Format Go code
make tidy            # Clean up go.mod
make lint            # Run golangci-lint (requires golangci-lint installed)
make bench           # Run all benchmarks
make bench-quick     # Run quick P95 L1/L2 benchmark
make bench-large     # Run large file benchmark (1GiB)
```

### Running Tests
```bash
# Run all tests
go test ./...

# Run tests for a specific package
go test ./internal/scanner
go test ./internal/interfaces/http/enforcer

# Run a single test by name
go test ./internal/scanner -run TestScannerStreaming
go test ./gateway -run TestHTTPCheck_SLA

# Run tests with verbose output
go test -v ./internal/scanner

# Run tests with race detector
go test -race ./...

# Run integration tests only
go test ./gateway -run Integration
go test ./internal/interfaces -run Integration

# Run SLA tests (requires PS_ENFORCE_SLA=1)
PS_ENFORCE_SLA=1 go test ./gateway -run TestHTTPCheck_SLA -count=1
PS_ENFORCE_SLA=1 go test ./gateway -run TestGRPCExtProc_SLA -count=1
```

### Running the Enforcer
```bash
./bin/ps-gateway &   # Starts HTTP (:9090) and gRPC ext_proc (:9091)
# Or with custom configuration
PS_ENFORCER_ADDR=:8080 PS_ENFORCER_GRPC_ADDR=:9092 ./bin/ps-gateway

# Health & readiness
curl -sf http://127.0.0.1:9090/readyz
curl -sf http://127.0.0.1:9090/healthz

# Metrics
curl -sf http://127.0.0.1:9090/metrics | grep -E 'ps_enforcer_requests_total|ps_extproc_streams_total'

# Direct check endpoint
curl -s -X POST http://localhost:9090/v1/check \
  -H 'content-type: text/plain' \
  --data 'hello world' -i
```

### Docker Demo
```bash
# Start full stack (Envoy + Enforcer + Backend)
docker compose up -d --build
# Or with mode override
MODE=enforce docker compose up -d --build

# Run demo tests
make demo-observe    # Start in observe mode
make demo-enforce    # Start in enforce mode
make clean-prompt    # Test benign prompt
make inj-prompt      # Test injection prompt
make ssn-prompt      # Test PII detection
make demo-stop       # Stop and cleanup
```

## Architecture Overview

PromptShield follows a layered architecture focused on runtime enforcement and policy evaluation:

### Core Data Flow
1. **Gateway Layer** (Envoy + HTTP): Envoy forwards to enforcer via `ext_authz` (headers) and `ext_proc` (streaming bodies); HTTP `/v1/check` for direct decisions
2. **Application Layer** (`internal/application/`): Business logic for policy evaluation and services  
3. **Scanner Engine** (`internal/scanner/`): Streaming scanning with rule compilation/evaluation
4. **Runtime Enforcement** (`internal/interfaces/`): HTTP and gRPC servers for real-time request filtering
5. **Audit/Telemetry** (`internal/audit`, `internal/observability/`): Deterministic events, Prometheus, OTel

### Key Packages by Layer

#### **Configuration & Bootstrap**
- `internal/config/`: Configuration management with Viper
- `internal/bootstrap/`: Dependency injection with semantic provider wiring

#### **Scanning Engine** 
- `internal/scanner/`: Core streaming scanner with bounded memory design
  - **Scanner**: Sliding-window processing for streams; chunk overlap for long lines
  - **High-Performance Matchers**: Aho-Corasick for L1, Bloom filters for L2/L3 gating, LRU caches
  - **Rule Evaluation**: Progressive L1→L2→L3 with early exits and fallbacks
- `internal/rules/`: RulePack YAML schema, validation, merging, and composition strategies

#### **Semantic Analysis (Level 3)**
- `internal/semantic/openai/`: OpenAI provider with official SDK, retryable HTTP, rate limiting
- `internal/semantic/anthropic/`: Anthropic provider with official SDK, same robustness features
- **Shared Features**: LRU caching (15min TTL), concurrency limiting, input redaction, OpenTelemetry tracing
- `internal/security/cred/`: OS keyring integration for secure API key storage

#### **Runtime Enforcement**
- `internal/interfaces/grpc/enforcer/`: Envoy ext_proc gRPC server for streaming request filtering
- `internal/interfaces/http/enforcer/`: HTTP `/v1/check`, `/healthz`, `/readyz`, `/metrics` with budget controls and metrics
- **ps-enforcer binary**: Standalone enforcement server with Prometheus metrics

#### **Infrastructure**
- `internal/shared/`: Common utilities (redaction, severity mapping, error handling)
- `internal/audit/`: Audit logging with SHA-256 hash chaining and rotation
- `internal/observability/`: OpenTelemetry metrics and tracing integration
- `pkg/types/`: Public API types for results and violations

### Advanced Architecture Patterns

#### **Streaming & Performance**
- **Bounded Memory**: Sliding-window scanning with configurable window/overlap
- **Parallel Processing**: Worker pools with deterministic ordering of results
- **Caching Strategy**: Multi-layer caching (regex compilation, semantic responses, rule tokens)
- **Resource Limits**: Configurable timeouts, concurrency bounds, and memory caps

#### **Rule Engine Design**
- **Progressive Evaluation**: L1 (keywords) → L2 (regex) → L3 (semantic) with early exits
- **Composition Strategies**: `all_matches` (default), `first_match`, `priority_order`
- **Context Gating**: `when`/`unless` conditions with runtime context
- **Rule Inheritance**: RulePack `extends` with deterministic override resolution

#### **Semantic Provider Architecture**
- **Official SDKs**: Uses `openai/openai-go` and `anthropics/anthropic-sdk-go`
- **HTTP Robustness**: Retryable HTTP with exponential backoff, rate limiting, circuit breaking
- **Security**: Input redaction, API key masking, secure credential storage in OS keyring
- **Observability**: Request/response logging, cache hit metrics, distributed tracing
- **Fallback Handling**: Graceful degradation to regex patterns when LLM unavailable

#### **Enterprise Features**
- **Runtime Enforcement**: gRPC ext_proc for Envoy, HTTP `/v1/check` for direct integration
- **Audit Trail**: Immutable logs with hash chaining for compliance  
- **Telemetry**: Privacy-safe metrics collection with configurable backends
- **Configuration Management**: Env-first config with validation and helpful errors

### Configuration Overview

PromptShield uses environment-first configuration (no CLI flags):

#### Core Settings
- `PS_WORKERS=N` — Worker pool size (0=auto)
- `PS_ALLOW_PATHS` / `PS_DENY_PATHS` — Path filtering
- `PS_TELEMETRY=1` — Enable OpenTelemetry
- `PS_TELEMETRY_ENDPOINT=otel-collector:4317` — OTel collector endpoint

#### Enforcer Configuration
- `PS_ENFORCER_ADDR=:9090` — HTTP listener address
- `PS_ENFORCER_GRPC_ADDR=:9091` — gRPC ext_proc address  
- `PS_ENFORCER_RULEPACK=rules/prompt-injection.yaml` — RulePack to load
- `PS_ENFORCER_TIMEOUT=300ms` — Per-request timeout
- `PS_ENFORCER_MAX_BODY_BYTES=1048576` — Max body size (1MB default)
- `PS_ENFORCER_FAIL_ON=HIGH` — Fail on severity threshold
- `PS_ENFORCER_ENFORCEMENT_MODE=observe|redact|quarantine|enforce` — Enforcement mode
- `PS_ENFORCER_REDACTION_MUTATION=true` — Enable body mutation in ext_proc

#### Semantic Analysis (Level 3)
- `PS_SEMANTIC_ENABLED=true` — Enable Level 3 rules
- `PS_SEMANTIC_PROVIDER=openai|anthropic` — Select provider
- `PS_SEMANTIC_TIMEOUT=100ms` — LLM call timeout
- `PS_SEMANTIC_CACHE_TTL=15m` — Response cache duration

#### OIDC Authentication
- `PS_ENFORCER_OIDC_ISSUER` — OIDC issuer URL
- `PS_ENFORCER_OIDC_AUDIENCE` — Expected audience claim

#### RulePack Management
- `PS_MAX_RULEPACK_KB=1024` — Max RulePack size (KB)
- `PS_MAX_RULES=1000` — Max rules per pack
- `PS_RULEPACK_RETENTION=10` — Keep last N versions

### Development Notes

#### Testing Framework
- **Unit Tests**: Comprehensive coverage for scanner, providers, and rule engine
- **Integration Tests**: Envoy + enforcer tests in `gateway/`, HTTP tests in `gateway/http_test.go`
- **Benchmarks**: Performance tests including 1GiB handling
- **Fuzzing**: Fuzz tests for rule validation and discovery

#### Security Considerations
- Input redaction prevents API key leakage in logs/traces
- TLS/mTLS between Envoy and enforcer; optional HTTP auth tokens
- Bounded resource usage prevents DoS via resource exhaustion

#### Observability Integration
- **OpenTelemetry**: Distributed tracing with automatic spans
- **Structured Logging**: slog with correlation IDs for request tracking
- **Metrics**: Prometheus metrics for runtime enforcement servers
- **Audit Trails**: SHA-256 hash-chained logs for compliance

#### Extension Points
- **Custom Providers**: Implement `SemanticAnalyzer` interface for new LLM providers
- **Rule Types**: Add new rule levels or evaluation strategies
- **Integrations**: Runtime servers provide templates for custom enforcement

## Code Organization Guidelines

### Package Structure
- `cmd/` — Thin command entry points, orchestration only (no business logic)
- `internal/` — Core business logic, organized by layer:
  - `application/` — Business services and commands
  - `interfaces/` — HTTP/gRPC servers and handlers
  - `scanner/` — Streaming scanner engine
  - `rules/` — Rule loading, validation, and evaluation
  - `semantic/` — LLM provider integrations
  - `audit/` — Audit logging and event store
  - `observability/` — Metrics, tracing, telemetry
- `pkg/types/` — Public API types (ScanResult, Violation)
- `rules/` — Built-in RulePack examples
- `docs/` — Documentation and API specs

### Development Practices
- **No global state**: Pass dependencies via constructors
- **Context everywhere**: Accept context.Context for cancellation/timeouts
- **Structured errors**: Wrap with context using `fmt.Errorf("context: %w", err)`
- **Deterministic output**: Sort results, stable ordering in parallel operations
- **Streaming-first**: Never load entire files; use bounded buffers
- **Worker pools**: Size with PS_WORKERS; maintain deterministic result ordering

### Performance Targets
- **Level 1 (keywords)**: < 1ms latency
- **Level 2 (regex)**: < 10ms latency  
- **Level 3 (semantic)**: < 100ms latency (with caching)
- **Gateway overhead**: < 50ms P95 added latency
- **Throughput**: ≥ 10 MB/s per instance
- **Memory**: Bounded at 64KB sliding window default

### Security Requirements
- **Input redaction**: Never log raw inputs; redact tokens/keys
- **Resource bounds**: Enforce timeouts, memory limits, max sizes
- **Regex safety**: Limit complexity to prevent ReDoS
- **API key storage**: Use OS keyring via `internal/security/cred/`
- **Audit trail**: SHA-256 hash-chained immutable logs
- you are a senior software engineer who specializes in Production Proxy Gateway Development in Go. When doing any task , you always think first. You ensure that the implementation is production grade, and it is not overengineered. You follow Go and Proxy API Gateway best practices when building production applications.