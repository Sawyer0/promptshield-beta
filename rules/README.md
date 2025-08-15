# PromptShield Production Rule Packs

This directory contains production-ready rule packs for immediate enterprise deployment.

## 🚀 Production Rule Packs

### `essentials.yaml` (Default)
**Zero-configuration security for immediate protection**
- Essential prompt injection detection
- Critical PII patterns (SSN, credit cards)
- API key protection (OpenAI, AWS, generic patterns)
- Response filtering for credential leaks
- **Performance**: Optimized for speed (50ms timeout)
- **Use case**: Quick deployment, minimal overhead

### `comprehensive.yaml`
**Full enterprise security coverage**
- Extends essentials with advanced protection
- Comprehensive API key detection (GitHub, Stripe, Google, Slack, Discord)
- Cloud credentials (AWS, GCP, Azure)
- Database connection strings and JWT tokens
- Cryptocurrency addresses
- Advanced prompt injection patterns
- **Performance**: Thorough scanning (100ms timeout)
- **Use case**: Maximum security for sensitive deployments

### `prompt-injection.yaml`
**Specialized LLM threat protection**
- Comprehensive prompt injection detection
- Jailbreak and DAN attempts
- Role manipulation and context escape
- Encoded injection attempts
- Semantic analysis for sophisticated attacks
- **Use case**: LLM-focused security, research environments

## 📁 Example & Learning Resources

See `docs/samples/` for:
- Industry-specific examples (healthcare, finance, legal)
- PII detection templates
- Shadow AI monitoring
- Custom composition strategies
- Legacy reference implementations

## 🔧 Quick Start

### Default (Recommended)
```bash
# Uses essentials.yaml automatically
docker run promptshield/enforcer
```

### Custom Rule Pack
```bash
# Use comprehensive protection
PS_ENFORCER_RULEPACK=rules/comprehensive.yaml ./ps-enforcer

# Use specialized prompt injection focus
PS_ENFORCER_RULEPACK=rules/prompt-injection.yaml ./ps-enforcer
```

### Kubernetes
```yaml
env:
  - name: PS_ENFORCER_RULEPACK
    value: "/rules/comprehensive.yaml"  # or essentials.yaml
```

## 📊 Rule Pack Comparison

| Feature | Essentials | Comprehensive | Prompt Injection |
|---------|------------|---------------|------------------|
| Prompt injection | ✅ Basic | ✅ Advanced | ✅ Specialized |
| API keys | ✅ Common | ✅ Extensive | ❌ None |
| PII detection | ✅ Critical | ✅ Full | ❌ None |
| Performance | 🚀 Fast | ⚡ Moderate | 🚀 Fast |
| Response filtering | ✅ Yes | ✅ Enhanced | ❌ None |
| Rules count | ~10 | ~30+ | ~15 |
| Startup time | <100ms | <200ms | <100ms |

## 🛡️ Security Levels

- **Level 1**: Keyword matching (fastest)
- **Level 2**: Regex patterns (balanced)
- **Level 3**: Semantic analysis (most accurate, requires API keys)

All production packs are optimized for Levels 1-2 by default.

## 🔄 Customization

Start with a production pack and extend:

```yaml
# custom-rules.yaml
apiVersion: promptshield.io/v1
kind: RulePack
metadata:
  name: custom
  
extends:
  - path: ./essentials.yaml
  
rules:
  - id: custom-rule
    # your custom rules here
```

## 📝 Rule Development

For custom rule development, see:
- `docs/samples/` for examples
- `docs/RulePacks.md` for DSL documentation
- API endpoint `/v1/rulepacks/validate` for testing