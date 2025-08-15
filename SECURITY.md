# Security Policy

## Current Security Status (v0.2.0)

PromptShield v0.2.0 is **safe for production Gateway use** with the limitations documented below. The enforcement core is production-ready for HTTP `/v1/check` and Envoy integration, with enterprise features planned for future releases.

## ✅ Security Features (Production Ready)

### Data Protection
- **Redaction system**: Automatically redacts OpenAI, Anthropic, and common API key patterns in logs and audit trails
- **Bounded memory**: Streaming architecture prevents memory exhaustion attacks
- **Input sanitization**: Basic validation prevents malformed YAML and regex patterns

### Network Security  
- **TLS-only**: All external API calls use HTTPS/TLS
- **Provider isolation**: Semantic analysis providers are sandboxed with timeouts
- **No data retention**: External providers don't store analyzed content (by design)

### Configuration Security
- **Credential hierarchy**: Environment variables > OS keychain > interactive prompts (no credentials in config files)
- **Validation**: Strict YAML schema validation prevents configuration injection
- **Principle of least privilege**: Default configurations are secure-by-default

## ⚠️ Current Limitations

### Audit Trail Security
**Status**: SHA-256 hash chain with canonical serialization  
**Impact**: Tamper-evident audit trails (still beta; schema may evolve)  
**Mitigation**: Suitable for operational forensics; pair with secure storage  
**Timeline**: Landed in v0.2.x; further hardening in v0.3.0

### Input Validation (Planned: v0.3.0)
**Issue**: No path traversal or injection protection  
**Impact**: Malicious file paths could access unintended files  
**Mitigation**: Use with trusted input files only  
**Timeline**: Comprehensive validation in v0.3.0

### Resource Limits (Planned: v0.3.0)
**Issue**: No protection against extremely large files or complex patterns  
**Impact**: Potential DoS via resource exhaustion  
**Mitigation**: Monitor resource usage, avoid untrusted large files  
**Timeline**: Configurable limits in v0.3.0

## 🚨 Not Production Ready

### ps-enforcer Runtime
**Status**: Experimental (HTTP + gRPC ext_proc)  
**Issue**: Not production-hardened (authz, quotas, tenancy, and SLOs incomplete)  
**Impact**: Unsafe for production access control without a sidecar proxy and upstream policy controls  
**Mitigation**: Run behind Envoy with tight budgets and mTLS.  
**Protection**: Previously required `PS_EXPERIMENTAL=true`; gate removed in v0.2.x. Treat as early-stage and run behind a proxy with strict limits.  
**Timeline**: Hardening in upcoming releases

### Envoy API Proxy Example
- **Status**: Included for local evaluation and testing
- **Files**: `envoy-config.yaml`, `docker-compose.yaml`
- **Behavior**: Envoy listens on 8080 and routes to `backend:8080`, integrating with `ps-enforcer` via `ext_authz` (HTTP :9090) and `ext_proc` (gRPC :9091) for header/body inspection.
- **Risk**: Example config is not production‑hardened. Do not deploy as‑is.
- **Hardening required for production**: mTLS between Envoy and enforcer; strict request/response timeouts and budgets; body size caps; header allowlists; rate limits; SLOs/monitoring; authentication and tenancy on enforcer.
- **Docs**: See `docs/Envoy.md` and `docs/ENVOY_INTEGRATION.md` for guidance and reference configurations.

### RulePack Features
**Status**: Implemented  
**Notes**: `extends/overrides`, imports, and composition (`all_matches`/`first_match`/`priority_order`) are implemented. `response` actions are supported in Gateway decisions; body mutation for redaction is available via Envoy `ext_proc`.

## 🔒 Security Best Practices

### For Development
```bash
# Use environment variables for API keys
export OPENAI_API_KEY="sk-..."
export ANTHROPIC_API_KEY="anthropic-..."

# Enable debug logging for troubleshooting
export PS_DEBUG=true

# Validate RulePacks before deployment
./bin/promptshield validate rules/
```

### For CI/CD
```bash
# Use Gateway decisions with JSON payloads and headers
curl -s -X POST http://localhost:9090/v1/check --data-binary @payload.txt -H 'content-type: text/plain' -i

# Enable audit logging in production
export PS_AUDIT_FILE=/var/log/promptshield/audit.log
```

### For Production Deployment
```yaml
enforcer:
  listen: ":9090"
  grpc_listen: ":9091"
audit_file: /var/log/promptshield/audit.log
redaction:
  enabled: true

# Semantic analysis (optional)
semantic:
  enabled: true
  provider: openai  # or anthropic
  cache_ttl: 900s   # 15 minutes
```

## 🛡️ Security Hardening (v0.3.0 Preview)

The next release will include comprehensive security hardening:

### Cryptographic Audit Trails
```go
// SHA-256 hash chain for tamper detection
func (l *AuditLogger) calculateHash(event AuditEvent) string {
    h := sha256.New()
    h.Write([]byte(event.Data))
    h.Write([]byte(event.PrevHash))
    return hex.EncodeToString(h.Sum(nil))
}
```

### Input Validation
```go
// Path traversal protection
func validatePath(path string) error {
    if strings.Contains(path, "..") {
        return fmt.Errorf("path traversal detected")
    }
    // Additional validation...
}
```

### Resource Limits
```yaml
# Future configuration
limits:
  max_file_size: 100MB
  max_pattern_length: 1000
  scan_timeout: 30s
  max_cache_entries: 1000
```

## 📋 Security Checklist

Before deploying PromptShield in production:

- [ ] **Read limitations**: Understand current security limitations
- [ ] **Validate RulePacks**: Test all rules before deployment
- [ ] **Secure credentials**: Use environment variables or OS keychain
- [ ] **Enable audit logging**: Configure audit trails for compliance
- [ ] **Monitor resource usage**: Watch for memory/CPU spikes
- [ ] **Update regularly**: Subscribe to security announcements
- [ ] **Test thoroughly**: Validate scanning results in staging environment

## 📞 Reporting Security Issues

**Do not open public issues for security vulnerabilities.**

Instead, please email security findings to: `security@promptshield.io`

Include:
- Description of the vulnerability
- Steps to reproduce
- Potential impact assessment
- Suggested mitigation (if any)

We will respond within 48 hours and work with you to address the issue responsibly.

## 🔄 Security Update Policy

- **Critical vulnerabilities**: Patch within 7 days
- **High severity issues**: Patch within 30 days  
- **Medium/Low issues**: Include in next regular release
- **Security advisories**: Published for all severity levels

## 📚 Additional Resources

- [OWASP LLM Top 10](https://owasp.org/www-project-top-10-for-large-language-model-applications/)
- [NIST AI Risk Management Framework](https://www.nist.gov/itl/ai-risk-management-framework)
- [Anthropic's Constitutional AI](https://www.anthropic.com/constitutional-ai)
- [OpenAI Safety Best Practices](https://platform.openai.com/docs/guides/safety-best-practices)