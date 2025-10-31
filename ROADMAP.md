# PromptShield Roadmap

This document outlines the development roadmap for PromptShield, including completed features, in-progress work, and planned enhancements.

---

## ✅ Completed (v0.2.0)

### Core Security Engine
- [x] 3-tier progressive scanning (L1: Aho-Corasick, L2: Regex, L3: LLM Semantic)
- [x] Streaming architecture with constant memory usage
- [x] Sliding window with overlap for boundary pattern detection
- [x] Early exit optimization
- [x] Content redaction for 12+ secret types
- [x] Luhn algorithm for credit card validation
- [x] Semantic analysis with OpenAI and Anthropic providers
- [x] Semantic caching with 15-minute TTL

### Integration & Deployment
- [x] Envoy ext_proc gRPC streaming integration
- [x] HTTP REST API (`/v1/check`, `/v1/scan`)
- [x] Docker multi-stage builds
- [x] Kubernetes deployment with Helm charts
- [x] Health checks (liveness and readiness probes)
- [x] Horizontal pod autoscaling support

### Enterprise Features
- [x] Multi-tenancy with tenant isolation
- [x] Usage tracking and billing metrics
- [x] Per-tenant rate limiting
- [x] API token management with scoping
- [x] Policy assignment system
- [x] Async job processing
- [x] File-based hash-chained audit logging

### Observability
- [x] Prometheus metrics
- [x] OpenTelemetry tracing
- [x] Structured JSON logging
- [x] Automatic PII redaction in logs

### Performance
- [x] LRU caching for semantic analysis
- [x] Optional Hyperscan support
- [x] Connection pooling
- [x] Circuit breakers

---

## 🔄 In Progress (v0.3.0)

### Event-Driven Architecture
- [ ] Wire Control Plane to publish rule update events
- [ ] Wire Enforcer to subscribe to rule updates
- [ ] Add hot-reload capability for rules
- [ ] Implement zero-downtime rule updates
- **Estimated Effort**: 2-3 days
- **Priority**: High
- **Blocker**: None (infrastructure complete)

### Hash-Chained Database Audits
- [ ] Create audit service layer
- [ ] Implement hash calculation for database entries
- [ ] Add chain verification endpoint
- [ ] Background job for periodic chain validation
- **Estimated Effort**: 1-2 days
- **Priority**: Medium
- **Blocker**: None (schema ready)

### Documentation
- [ ] Add godoc comments to all exported functions
- [ ] Create architecture decision records (ADRs)
- [ ] Write deployment best practices guide
- [ ] Add troubleshooting runbook
- **Estimated Effort**: 3-4 days
- **Priority**: Medium

### Code Quality
- [ ] Rename `internal/infrastructure/messaging/nats/` to `redis/`
- [ ] Standardize naming conventions (New* prefix)
- [ ] Centralize configuration management
- [ ] Add typed errors for better error handling
- **Estimated Effort**: 2-3 days
- **Priority**: Low

---

## 📋 Planned (v0.4.0)

### LLM-Specific Security
- [ ] Tool calling restrictions (whitelist/blacklist)
- [ ] Function filtering for LLM function calls
- [ ] Parameter validation for tool calls
- [ ] Context injection detection
- **Estimated Effort**: 1-2 weeks
- **Priority**: High
- **Dependencies**: Research on LLM tool calling patterns

### Testing Framework
- [ ] YAML-based test cases for rules
- [ ] `should_detect` and `should_not_detect` assertions
- [ ] Automated regression testing
- [ ] Performance benchmarking suite
- **Estimated Effort**: 1 week
- **Priority**: Medium

### Advanced Rule Features
- [ ] Rule dependencies and ordering
- [ ] Conditional rule execution
- [ ] Dynamic rule parameters
- [ ] Rule versioning and rollback
- **Estimated Effort**: 2 weeks
- **Priority**: Medium

---

## 🚀 Future (v0.5.0+)

### Marketplace & Distribution
- [ ] Public rule pack marketplace
- [ ] Rule pack signing and verification
- [ ] Community-contributed rules
- [ ] Rule pack ratings and reviews
- **Estimated Effort**: 3-4 weeks
- **Priority**: Low
- **Dependencies**: Community adoption

### Plugin System
- [ ] Custom validator plugins (Python, JavaScript)
- [ ] Custom transformer plugins
- [ ] Plugin SDK and documentation
- [ ] Plugin marketplace
- **Estimated Effort**: 3-4 weeks
- **Priority**: Low

### Integrations
- [ ] Webhook support (Slack, Discord, PagerDuty)
- [ ] Datadog metrics integration
- [ ] Splunk logging integration
- [ ] SIEM integration (Elastic, Sumo Logic)
- **Estimated Effort**: 2-3 weeks
- **Priority**: Medium

### Advanced Deployment
- [ ] WebAssembly compilation for edge deployment
- [ ] Browser SDK for client-side scanning
- [ ] Mobile SDKs (iOS, Android)
- [ ] Serverless deployment (AWS Lambda, Cloudflare Workers)
- **Estimated Effort**: 4-6 weeks
- **Priority**: Low

### Protocol Support
- [ ] GraphQL schema-aware policies
- [ ] gRPC request/response inspection
- [ ] WebSocket message filtering
- [ ] MQTT message scanning
- **Estimated Effort**: 2-3 weeks
- **Priority**: Medium

---

## 🎯 Long-Term Vision

### AI-Powered Features
- [ ] Automatic rule generation from examples
- [ ] Anomaly detection using ML
- [ ] Adaptive policies based on threat intelligence
- [ ] Zero-day prompt injection detection

### Compliance & Certification
- [ ] FedRAMP certification
- [ ] SOC 2 Type II audit
- [ ] HIPAA compliance validation
- [ ] PCI DSS certification

### Enterprise Features
- [ ] Multi-region deployment
- [ ] Active-active replication
- [ ] Disaster recovery automation
- [ ] Advanced analytics dashboard

---

## 📊 Priority Matrix

| Feature | Priority | Effort | Impact | Status |
|---------|----------|--------|--------|--------|
| Event-driven rule updates | High | Low | High | In Progress |
| Hash-chained DB audits | Medium | Low | Medium | In Progress |
| Tool calling restrictions | High | Medium | High | Planned |
| Testing framework | Medium | Low | Medium | Planned |
| Marketplace integration | Low | High | Medium | Future |
| Plugin system | Low | High | Low | Future |
| WebAssembly support | Low | High | Medium | Future |

---

## 🤝 Contributing

We welcome contributions! If you're interested in working on any of these features:

1. Check the [Issues](https://github.com/sawyer0/promptshield-beta/issues) page for open tasks
2. Comment on an issue to claim it
3. Read [CONTRIBUTING.md](CONTRIBUTING.md) for development guidelines
4. Submit a pull request with your changes

For major features, please open an issue first to discuss the approach.

---

## 📝 Changelog

See [CHANGELOG.md](CHANGELOG.md) for detailed release notes.

---

## 💬 Feedback

Have ideas for features not on this roadmap? Open an issue with the `enhancement` label or start a discussion in [GitHub Discussions](https://github.com/sawyer0/promptshield-beta/discussions).


