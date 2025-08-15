# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

PromptShield is an Envoy-integrated LLM API Gateway with enterprise-grade runtime enforcement. It uses a progressive three-tier rule system: Level 1 (keyword matching), Level 2 (regex patterns), and Level 3 (semantic/LLM analysis, opt-in).

## Essential Commands

### Build and Development
```bash
make build-enforcer # Build ps-enforcer binary (HTTP + gRPC ext_proc)
make test           # Run all tests  
make fmt            # Format Go code
make tidy           # Clean up go.mod
make lint           # Run golangci-lint (requires golangci-lint installed)
make bench          # Run all benchmarks
make bench-large    # Run large file benchmark (1GiB)
```

### Running the Enforcer
```bash
./bin/ps-enforcer &  # Starts HTTP (:9090) and gRPC ext_proc (:9091)
# Health & readiness
curl -sf http://127.0.0.1:9090/readyz
curl -sf http://127.0.0.1:9090/healthz
# Metrics
curl -sf http://127.0.0.1:9090/metrics | head -50
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

- `PS_SEMANTIC_ENABLED=true` — Enable Level 3 rules
- `PS_SEMANTIC_PROVIDER=openai|anthropic` — Select provider
- `PS_WORKERS=N` — Worker pool size (0=auto)
- `PS_ALLOW_PATHS` / `PS_DENY_PATHS` — Path filtering
- Enforcer: `PS_ENFORCER_*` — addresses, TLS, budgets, streaming windows/overlap
- OIDC: `PS_ENFORCER_OIDC_ISSUER`, `PS_ENFORCER_OIDC_AUDIENCE` — Enterprise authentication

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