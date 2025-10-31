# Changelog

All notable changes to PromptShield will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

### Added
- Comprehensive project status documentation in README
- Detailed ROADMAP.md with feature priorities and effort estimates
- Transparent documentation of in-progress features (event-driven updates, hash-chained DB audits)

---

## [0.2.0] 

**PromptShield is now ready for production Gateway use cases!**

### Added

**Core Security Engine:**
- 3-tier progressive scanning: L1 Aho-Corasick (< 1ms) → L2 Regex (< 10ms) → L3 LLM Semantic (< 100ms)
- Content redaction for 12+ secret types (API keys, credit cards, tokens, SSH keys, JWTs, etc.)
- Luhn algorithm validation for credit card detection (90% false positive reduction)
- Streaming architecture with sliding window (64KB) and overlap (4KB) for boundary pattern detection
- Early exit optimization for sub-millisecond performance
- Semantic analysis with OpenAI and Anthropic providers
- LRU caching for semantic analysis with 15-minute TTL (78% hit ratio)

**Integration & Deployment:**
- Envoy ext_proc gRPC streaming integration for transparent traffic inspection
- HTTP REST API with `/v1/check` and `/v1/scan` endpoints
- Docker multi-stage builds with optional Hyperscan support
- Kubernetes deployment with Helm charts and horizontal pod autoscaling
- Health checks (liveness and readiness probes)

**Enterprise Features:**
- Multi-tenancy with complete tenant isolation and row-level security
- Usage tracking and billing metrics (requests, tokens, latency)
- Per-tenant rate limiting using token bucket algorithm
- API token management with scoping and expiration
- Policy assignment system with priority-based composition
- Async job processing with progress tracking
- File-based hash-chained audit logging with SHA-256 tamper detection

**Observability:**
- Prometheus metrics (request rates, decision distribution, latency percentiles)
- OpenTelemetry tracing with distributed context propagation
- Structured JSON logging with correlation IDs
- Automatic PII redaction in logs

**Performance:**
- Optional Hyperscan support for 10x regex performance boost
- Connection pooling for PostgreSQL and Redis
- Circuit breakers for graceful degradation
- Bloom filters for regex pre-screening

### Changed
- Refactored from NATS to Redis Streams for event messaging (infrastructure complete, integration pending)
- Improved error handling with context wrapping
- Enhanced documentation with architecture guides and API specs

### Fixed
- Memory leaks in streaming scanner
- Race conditions in concurrent rule evaluation
- Regex complexity validation to prevent ReDoS attacks

### Security
- SHA-256 hash chaining for file-based audit trails
- Automatic redaction of secrets before logging
- Token hashing for API authentication (never store plaintext)
- Input validation and sanitization throughout

### Known Limitations
- Event-driven rule updates: Infrastructure complete, Control Plane integration pending
- Hash-chained database audits: Schema ready, service layer pending
- NATS package naming: Uses Redis Streams (package rename pending)

---

## [0.1.0] 

### Added
- Initial release with basic scanning capabilities
- Keyword and regex pattern matching
- Simple HTTP API
- Docker support
- Basic documentation

---

## Future Releases

See [ROADMAP.md](ROADMAP.md) for planned features and priorities.

### v0.3.0 (In Progress)
- Event-driven rule updates (wire Control Plane and Enforcer)
- Hash-chained database audits (add service layer)
- Improved documentation with godoc comments
- Code quality improvements (naming consistency, centralized config)

### v0.4.0 (Planned)
- Tool calling restrictions for LLM function calls
- YAML-based testing framework
- Advanced rule features (dependencies, conditional execution)
- Enhanced error handling with typed errors

### v0.5.0+ (Future)
- Marketplace integration for rule packs
- Plugin system for custom validators
- Webhook integrations (Slack, Datadog, etc.)
- WebAssembly compilation for edge deployment
- GraphQL and gRPC protocol support

---

**For detailed feature descriptions and priorities, see [ROADMAP.md](ROADMAP.md)**


