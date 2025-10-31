# Security Policy

## Current Security Status (v0.2.0)

PromptShield v0.2.0 is **production-ready for LLM security gateway use** with the features and limitations documented below. The core scanning engine, Envoy integration, and multi-tenancy features are stable and battle-tested.

**Last Updated**: January 2025  
**Current Version**: v0.2.0  
**Security Contact**: See [Reporting Security Issues](#-reporting-security-issues) below

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

## ⚠️ Known Limitations & Mitigations

### Event-Driven Rule Updates
**Status**: Infrastructure complete, integration pending  
**Impact**: Rule updates require manual reload or service restart  
**Mitigation**: Use hot-reload API endpoint or rolling deployments  
**Timeline**: Full integration in v0.3.0

### Hash-Chained Database Audits
**Status**: File-based audit logs have full hash-chaining; database schema ready but service layer pending  
**Impact**: Database audit entries don't have automatic hash-chain verification  
**Mitigation**: Use file-based audit logger for tamper-evident trails  
**Timeline**: Database hash-chaining service layer in v0.3.0

### Regex Complexity Protection
**Status**: Basic complexity validation implemented  
**Impact**: Extremely complex regex patterns could cause performance degradation  
**Mitigation**: Pattern complexity limits enforced; configurable via `PS_MAX_REGEX_NODES`  
**Current Protection**: Default limits prevent ReDoS attacks

## 🔐 Production Deployment Recommendations

### Enforcer Runtime Security
**Status**: Production-ready with proper configuration  
**Requirements for production**:
- Run behind Envoy proxy with mTLS
- Configure strict request/response timeouts
- Set body size caps (`PS_ENFORCER_MAX_STREAM_BYTES`)
- Enable rate limiting per tenant
- Use TLS for all external connections
- Configure authentication tokens (`PS_ENFORCER_AUTH_TOKEN`, `PS_ENFORCER_ADMIN_TOKEN`)

### Envoy Integration Best Practices
**Example configs provided**: `envoy-config.yaml`, `docker-compose.yaml`  
**Status**: Reference implementations for local development  
**Production hardening checklist**:
- [ ] Enable mTLS between Envoy and enforcer
- [ ] Configure strict timeouts (request: 5s, response: 10s)
- [ ] Set body size limits (default: 5MB)
- [ ] Implement header allowlists
- [ ] Enable rate limiting
- [ ] Configure monitoring and SLOs
- [ ] Set up authentication and tenant isolation
- [ ] Use TLS certificates from trusted CA

**Documentation**: See `docs/Envoy.md` and `docs/ENVOY_INTEGRATION.md` for production deployment guides.

### Multi-Tenancy Security
**Status**: Fully implemented  
**Features**:
- Row-level security in database
- API token scoping per tenant
- Per-tenant rate limiting
- Isolated policy assignments
- Separate usage tracking and billing

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

Instead, please report security findings via:
- **GitHub Security Advisories**: [Report a vulnerability](https://github.com/sawyer0/promptshield-beta/security)
- **Email**: Create an issue with the `security` label (for non-critical issues)

Include:
- Description of the vulnerability
- Steps to reproduce
- Potential impact assessment
- Suggested mitigation (if any)

We will respond within 72 hours and work with you to address the issue responsibly.

### Disclosure Policy
- We follow coordinated disclosure practices
- Security fixes will be released as soon as possible
- Credit will be given to reporters (unless anonymity is requested)

## 🔄 Security Update Policy

- **Critical vulnerabilities**: Patch released ASAP (target: 7 days)
- **High severity issues**: Patch within 30 days  
- **Medium/Low issues**: Include in next regular release
- **Security advisories**: Published via GitHub Security Advisories for all severity levels

### Supported Versions

| Version | Supported          | Status |
| ------- | ------------------ | ------ |
| 0.2.x   | ✅ Yes             | Current stable release |
| 0.1.x   | ⚠️ Limited support | Upgrade recommended |
| < 0.1   | ❌ No              | Unsupported |

We recommend always running the latest stable release.

## 📚 Additional Resources

- [OWASP LLM Top 10](https://owasp.org/www-project-top-10-for-large-language-model-applications/)
- [NIST AI Risk Management Framework](https://www.nist.gov/itl/ai-risk-management-framework)
- [Anthropic's Constitutional AI](https://www.anthropic.com/constitutional-ai)
- [OpenAI Safety Best Practices](https://platform.openai.com/docs/guides/safety-best-practices)