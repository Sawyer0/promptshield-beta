# PromptShield v0.2.0-beta Shipping Guide

## 🚢 Ready to Ship Checklist

### ✅ Safety Measures Implemented
- [x] **RulePack validation**: Prevents usage of unimplemented features with clear error messages
- [x] **ps-enforcer protection**: Requires `PS_EXPERIMENTAL=true` with clear experimental notice
- [x] **mTLS support (HTTP + gRPC)**: Optional TLS and client-auth for runtime surfaces (see docs)
- [x] **Kubernetes manifests**: Deployment with HPA, PDB, and Prometheus ServiceMonitor
- [x] **Grafana dashboard**: Ready-to-import dashboard for enforcer metrics
- [x] **Production readiness matrix**: Clear documentation of what's safe vs. not safe
- [x] **Security documentation**: Honest assessment of current limitations
- [x] **Feature roadmap**: Clear timeline for missing features

### ✅ Documentation Complete
- [x] **README.md**: Updated with production readiness matrix and security notes
- [x] **ROADMAP.md**: Clear timeline and priorities for future releases
- [x] **SECURITY.md**: Comprehensive security policy and current limitations
- [x] **CHANGELOG.md**: Release notes for v0.2.0-beta
- [x] **API Docs**: HTTP OpenAPI (`docs/api/openapi.yaml`), gRPC ref (`docs/api/grpc.md`), metrics (`docs/api/metrics.md`), provider guides (`docs/api/providers/`) and security (`docs/api/security.md`)
- [x] **Runtime integration**: Envoy/Kubernetes/mTLS (`docs/ENVOY_INTEGRATION.md`), k8s manifests (`deployments/kubernetes/enforcer.yaml`)
- [x] **Observability**: Grafana dashboard (`monitoring/dashboards/promptshield-enforcer.json`)

## 🎯 Release Process

### 1. Pre-Release Testing
```bash
# Build and test core functionality
make build
make test

# Test validation prevents broken features
echo 'extends: ["nonexistent"]' > test-extends.yaml
./bin/promptshield validate test-extends.yaml  # Should fail with clear error

# Test ps-enforcer startup
./bin/ps-enforcer &  # Starts HTTP (:9090) and gRPC ext_proc (:9091) by default

# Verify readiness and metrics endpoints
curl -sf http://127.0.0.1:9090/readyz
curl -sf http://127.0.0.1:9090/metrics | head -50

# Verify gRPC health
grpcurl -plaintext localhost:9091 grpc.health.v1.Health/Check

# Optional: Kubernetes smoke test (requires cluster)
kubectl create ns promptshield || true
kubectl -n promptshield create secret generic ps-enforcer-tls \
  --from-file=server.crt=/path/to/server.crt \
  --from-file=server.key=/path/to/server.key \
  --from-file=ca.crt=/path/to/ca.crt || true
kubectl -n promptshield apply -f deployments/kubernetes/enforcer.yaml
kubectl -n promptshield get deploy/promptshield-enforcer
```

### 2. Version Tagging
```bash
# Update version in CHANGELOG.md with actual date
# Update version references in documentation if needed

# Create and push tag
git add -A
git commit -m "Release v0.2.0-beta

- Add feature validation to prevent usage of unimplemented features
- Add experimental gates for ps-enforcer
- Update documentation with production readiness matrix
- Add comprehensive security policy and roadmap"

git tag -a v0.2.0-beta -m "PromptShield v0.2.0-beta - Beta Release

Production-ready CLI scanner for LLM security with honest limitations."

git push origin main
git push origin v0.2.0-beta
```

### 3. GitHub Release
Create GitHub release with this description:

```markdown
# PromptShield v0.2.0-beta - Beta Release 🛡️

**PromptShield is now ready for production CLI use cases!**

This beta release provides a production-ready CLI scanner for LLM security with honest documentation about current limitations.

## ✅ What's Ready for Production

- **CLI Scanning**: 3-tier rule system (keywords → regex → semantic analysis)
- **Semantic Analysis**: OpenAI & Anthropic providers with caching
- **Output Formats**: JSON, stylish, GitHub, NDJSON, and more
- **CI/CD Integration**: Auto-detection, deterministic output, shell completion
- **Configuration**: Flexible hierarchy with environment variables and config files

## ⚠️ Current Limitations (Honestly Documented)

- **Audit trails**: SHA-256 hash chain (tamper-evident); ensure secure storage and rotation
- **Input validation**: No path traversal protection (trusted files only)  
- **ps-enforcer**: Stub implementation with experimental gate protection

## 🛡️ Safety Features

- **Feature validation**: Prevents usage of unimplemented RulePack features
- **Clear error messages**: Point to roadmap for missing functionality
- **Experimental gates**: ps-enforcer requires explicit opt-in
- **mTLS**: Optional client-auth on HTTP `/check` and gRPC ext_proc endpoints
- **Budgets**: Request timeouts and stream size caps (documented), conservative defaults

## 📋 Perfect For

- Pre-commit hooks and CI/CD pipeline scanning
- Batch analysis of training data and prompt libraries
- Security audits and development workflows
- Testing LLM applications for security issues

## 🚀 Coming Next (v0.3.0 in 4-6 weeks)

- SHA-256 audit hashing (tamper-evident trails)
- Input validation and path traversal protection
- Resource limits and DoS protection
- Enhanced redaction for cloud provider tokens

## 📚 Documentation

- [Production Readiness Matrix](README.md#-production-readiness)
- [Security Policy](SECURITY.md) 
- [Complete Roadmap](ROADMAP.md)
- [Runtime Integration (Envoy/K8s/mTLS)](docs/ENVOY_INTEGRATION.md)
- [HTTP API (OpenAPI)](docs/api/openapi.yaml)
- [gRPC API (ext_proc)](docs/api/grpc.md)
- [Metrics & Dashboard](docs/api/metrics.md), Grafana: `monitoring/dashboards/promptshield-enforcer.json`
- [Usage Examples](README.md#quickstart)

## 🔧 Runtime Enhancements in v0.2.0

- Optional TLS/mTLS for HTTP `/check` and gRPC ext_proc
- Kubernetes deployment with HPA and PodDisruptionBudget
- Prometheus ServiceMonitor for metrics scraping
- Grafana dashboard for decision rate, latency, and bytes processed
- HTTP OpenAPI and gRPC reference docs added
- Anthropic/OpenAI provider docs and example RulePacks (e.g., `rules/semantic-anthropic.yaml`)

**Download the binary or build from source to get started!**
```

### 4. Binary Distribution (Optional)
```bash
# Cross-compile binaries for distribution
GOOS=linux GOARCH=amd64 make build
mv bin/promptshield bin/promptshield-linux-amd64

GOOS=darwin GOARCH=amd64 make build  
mv bin/promptshield bin/promptshield-darwin-amd64

GOOS=windows GOARCH=amd64 make build
mv bin/promptshield bin/promptshield-windows-amd64.exe

# Attach to GitHub release
```

## 📢 Communication Strategy

### Target Audiences

**Primary: Security-conscious developers**
- Message: "Production-ready CLI scanner with honest limitations"
- Channels: Security forums, Reddit r/netsec, Twitter
- Key points: Works now, clear roadmap, transparent about limitations

**Secondary: LLM developers**  
- Message: "Add security scanning to your LLM development workflow"
- Channels: AI/ML communities, Hacker News, dev blogs
- Key points: Easy integration, multiple output formats, semantic analysis

### Sample Announcement

```markdown
🚢 PromptShield v0.2.0-beta is live!

A production-ready CLI scanner for LLM security. What makes this different:

✅ Works reliably RIGHT NOW for CI/CD and batch scanning
⚠️ Honestly documents current limitations (audit hashing, input validation)
🛡️ Prevents accidental usage of incomplete features
🗺️ Clear roadmap for security hardening (v0.3.0 in 4-6 weeks)

Perfect for teams who want LLM security scanning today, not "eventually."

GitHub: [link]
Docs: [link]
```

## 🔍 Post-Release Monitoring

### Success Metrics
- GitHub stars and forks
- Issues opened (feature requests vs. bugs)
- Downloads/binary usage
- Community feedback on limitations

### Expected User Feedback
- **"Why can't I use extends?"** → Point to validation error message and roadmap
- **"Is this production ready?"** → Point to production readiness matrix
- **"When will ps-enforcer work?"** → Point to v0.4.0 timeline

### Red Flags to Watch
- Users bypassing experimental gates (indicates docs unclear)
- Bug reports about "silent failures" (indicates validation gaps)
- Security issues with current implementation

### Runtime SLOs to watch
- ExtProc p95 latency (no L3): ≤ 100ms
- ExtProc throughput per instance: ≥ 5k decisions/sec sustained for 15m
- Memory ≤ 500MB at p95 workload
- 99.9% availability for decision path (monthly)

## 🔄 Iteration Plan

Based on user feedback:

**Week 1-2 after release:**
- Monitor issues and discussions
- Quick fixes for documentation clarity
- Patch any critical bugs

**Week 3-4:**
- Analyze usage patterns
- Prioritize v0.3.0 features based on user needs
- Begin security hardening implementation

**Month 2:**
- Release v0.3.0 with security fixes
- Promote from "beta" to stable
- Begin v0.4.0 runtime enforcement work

This shipping approach balances honesty about limitations with confidence in what works, building trust through transparency rather than overpromising.