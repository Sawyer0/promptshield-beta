# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

PromptShield is a streaming-first CLI security scanner for LLM prompts and responses with enterprise-grade runtime enforcement capabilities. It uses a progressive three-tier rule system: Level 1 (keyword matching), Level 2 (regex patterns), and Level 3 (semantic/LLM analysis, opt-in).

## Essential Commands

### Build and Development
```bash
make build          # Build the binary to bin/promptshield
make build-enforcer # Build ps-enforcer binary (gRPC ext_proc)
make test           # Run all tests  
make fmt            # Format Go code
make tidy           # Clean up go.mod
make lint           # Run golangci-lint (requires golangci-lint installed)
make bench          # Run all benchmarks
make bench-large    # Run large file benchmark (1GiB)
```

### Running PromptShield
```bash
./bin/promptshield scan --rulepack rules path/to/file.txt    # Basic scan
./bin/promptshield --json scan --rulepack rules file.txt     # JSON output
./bin/promptshield validate rules/                            # Validate rule packs
./bin/promptshield demo                                       # Interactive demo
./bin/promptshield config print                               # Show configuration
```

### Testing Individual Components
```bash
go test ./cmd/...                    # Test CLI commands
go test ./internal/scanner/          # Test scanner implementation
go test ./internal/semantic/openai/  # Test OpenAI provider
go test -v ./...                     # Verbose test output for all packages
go test -run TestName                # Run specific test by name
```

## Architecture Overview

PromptShield follows a layered architecture with clear separation between CLI, scanning engine, and runtime enforcement:

### Core Data Flow
1. **CLI Layer** (`cmd/`): Parses flags/env/config (Cobra/Viper) and dispatches to subcommands
2. **Application Layer** (`internal/application/`): Business logic for scan operations and rule management  
3. **Scanner Engine** (`internal/scanner/`): Streaming line-by-line scanning with rule compilation/evaluation
4. **Runtime Enforcement** (`internal/interfaces/`): HTTP and gRPC servers for real-time request filtering
5. **Output Layer** (`internal/report/`): Deterministic result formatting and reporting

### Key Packages by Layer

#### **CLI & Configuration**
- `cmd/`: CLI entry point with Cobra commands (`scan`, `validate`, `demo`, `config`, `update`, `benchmark`)
  - Configuration precedence: Flags > Environment (PS_*) > Config files > Defaults
  - Integration tests via testscript framework in `cmd/testdata/scripts/`
- `internal/config/`: Configuration management with Viper
- `internal/bootstrap/`: Dependency injection with semantic provider auto-wiring

#### **Scanning Engine** 
- `internal/scanner/`: Core streaming scanner with bounded memory design
  - **Scanner**: Line-by-line processing with ~16MiB default buffer, chunk overlap for long lines
  - **Orchestrator**: File-level parallelism with configurable worker pools
  - **High-Performance Matchers**: Aho-Corasick for L1, Bloom filters for L2/L3 gating, LRU caches
  - **Rule Evaluation**: Progressive L1→L2→L3 with early exits and fallbacks
- `internal/rules/`: RulePack YAML schema, validation, merging, and composition strategies
- `internal/discovery/`: File path expansion with glob support and vendor directory skipping

#### **Semantic Analysis (Level 3)**
- `internal/semantic/openai/`: OpenAI provider with official SDK, retryable HTTP, rate limiting
- `internal/semantic/anthropic/`: Anthropic provider with official SDK, same robustness features
- **Shared Features**: LRU caching (15min TTL), concurrency limiting, input redaction, OpenTelemetry tracing
- `internal/security/cred/`: OS keyring integration for secure API key storage

#### **Runtime Enforcement**
- `internal/interfaces/grpc/enforcer/`: Envoy ext_proc gRPC server for streaming request filtering
- `internal/interfaces/http/enforcer/`: HTTP `/check` endpoint with budget controls and metrics
- **ps-enforcer binary**: Standalone enforcement server with Prometheus metrics

#### **Infrastructure**
- `internal/shared/`: Common utilities (redaction, severity mapping, terminal UI, error handling)
- `internal/audit/`: Audit logging with SHA-256 hash chaining and rotation
- `internal/observability/`: OpenTelemetry metrics and tracing integration
- `pkg/types/`: Public API types for results and violations

### Advanced Architecture Patterns

#### **Streaming & Performance**
- **Bounded Memory**: Uses `bufio.Scanner` with configurable token buffer for large files
- **Chunk Processing**: Long lines split with configurable overlap to prevent context loss
- **Parallel Processing**: File-level workers with deterministic result ordering
- **Caching Strategy**: Multi-layer caching (regex compilation, semantic responses, rule tokens)
- **Resource Limits**: Configurable timeouts, concurrency bounds, and memory caps

#### **Rule Engine Design**
- **Progressive Evaluation**: L1 (keywords) → L2 (regex) → L3 (semantic) with early exits
- **Composition Strategies**: `all_matches` (default), `first_match`, `priority_order`
- **Context Gating**: `when`/`unless` conditions with runtime context merging
- **Rule Inheritance**: RulePack `extends` with deterministic override resolution
- **Performance Hints**: Rule-level timeouts, case sensitivity, whole-word matching

#### **Semantic Provider Architecture**
- **Official SDKs**: Uses `openai/openai-go` and `anthropics/anthropic-sdk-go` for API stability
- **HTTP Robustness**: Retryable HTTP with exponential backoff, rate limiting, circuit breaking
- **Security**: Input redaction, API key masking, secure credential storage in OS keyring
- **Observability**: Request/response logging, cache hit metrics, distributed tracing
- **Fallback Handling**: Graceful degradation to regex patterns when LLM unavailable

#### **Enterprise Features**
- **Runtime Enforcement**: gRPC ext_proc for Envoy integration, HTTP endpoints for direct integration
- **Audit Trail**: Immutable logs with hash chaining for compliance and forensics  
- **Telemetry**: Privacy-safe metrics collection with configurable backends
- **Configuration Management**: Multi-source config with validation and helpful error messages

### Configuration Architecture

PromptShield uses a sophisticated configuration system with multiple sources:

```
Precedence: CLI Flags > Environment Variables > Config Files > Defaults
```

#### **Environment Variables**
- `PS_SEMANTIC_ENABLED=true` - Enable Level 3 rules
- `PS_SEMANTIC_PROVIDER=openai|anthropic` - Select provider
- `PS_WORKERS=N` - Worker pool size (0=auto)
- `PS_OUTPUT_FORMAT=json|stylish|github` - Output format
- `PS_ALLOW_PATHS` / `PS_DENY_PATHS` - Path filtering

#### **Config File Locations**
1. `promptshield.yaml` (working directory)
2. `~/.config/promptshield/promptshield.yaml`
3. `~/.promptshield/promptshield.yaml`
4. `$XDG_CONFIG_HOME/promptshield/promptshield.yaml`

### RulePack Schema

RulePacks follow a versioned YAML schema with full validation:

```yaml
apiVersion: promptshield.io/v1
kind: RulePack
metadata:
  name: pack-name
  version: 0.1.0
# Inheritance and composition
extends:
  - path: ./base-pack.yaml
composition:
  strategy: all_matches  # or first_match, priority_order
# Rule definitions
rules:
  - id: unique-id
    level: 1|2|3
    severity: LOW|WARNING|HIGH|CRITICAL
    # Level 1: Keywords
    keywords: ["keyword1", "keyword2"]
    # Level 2: Regex patterns  
    patterns:
      - regex: "pattern"
        flags: [ignorecase, multiline]
    # Level 3: Semantic analysis
    semantic:
      model: "gpt-4o-mini"
      analysis_prompt: "Analyze: {input}"
      confidence_threshold: 0.85
      fallback_on_error: true
    # Conditional evaluation
    when:
      - context_key: "value"
    unless:
      - other_key: "exclude_value"
    # Response actions (parsed but not enforced by CLI)
    response:
      action: block|allow|flag
      message: "Custom message"
```

### Performance & Scalability

#### **Memory Management**
- Streaming design with bounded buffers prevents OOM on large files
- LRU caches with TTL for semantic responses and compiled patterns
- Configurable chunk overlap for long-line processing
- Resource limits enforced at scanner and provider levels

#### **Concurrency Model**
- File-level parallelism with configurable worker pools
- Semantic provider concurrency limiting (2 parallel by default)
- Rate limiting for external API calls (10 req/s OpenAI, 5 req/s Anthropic)
- Context-aware cancellation throughout the pipeline

#### **Optimization Features**
- Aho-Corasick for efficient multi-keyword matching
- Bloom filters for quick L2/L3 rule rejection
- Global regex compilation cache with pattern deduplication
- Literal token extraction from regex for precise gating

## Development Notes

### Testing Framework
- **Integration Tests**: testscript framework in `cmd/testdata/scripts/` for CLI behavior
- **Unit Tests**: Comprehensive coverage for scanner, providers, and rule engine
- **Benchmarks**: Performance tests including 1GiB file handling
- **Fuzzing**: Fuzz tests for rule validation and file discovery

### Security Considerations
- Input redaction prevents API key leakage in logs/traces
- Path traversal protection with `..` component rejection  
- Configurable path allow/deny lists for access control
- Secure credential storage via OS keyring integration
- Bounded resource usage prevents DoS via resource exhaustion

### Observability Integration
- **OpenTelemetry**: Distributed tracing with automatic span creation
- **Structured Logging**: slog with correlation IDs for request tracking
- **Metrics**: Prometheus metrics for runtime enforcement servers
- **Audit Trails**: SHA-256 hash-chained logs for compliance

### Extension Points
- **Custom Providers**: Implement `SemanticAnalyzer` interface for new LLM providers
- **Output Formats**: Extend `internal/report/` for custom reporting formats  
- **Rule Types**: Add new rule levels or evaluation strategies
- **Integrations**: Runtime servers provide templates for custom enforcement