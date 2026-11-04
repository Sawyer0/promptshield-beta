# PromptShield Rule Pack Examples & Samples

This directory contains example rule packs and templates for learning and customization.

## 📚 Learning Examples

### Basic Examples
- **`basic-security-legacy.yaml`** - Original basic rule pack (reference)
- **`example.yaml`** - Simple rule pack template
- **`example-composition-first-match.yaml`** - Composition strategy example

### Feature-Specific Examples
- **`pii-detection.yaml`** - Comprehensive PII detection template
- **`semantic-anthropic.yaml`** - Level 3 semantic analysis examples
- **`shadow-ai-detection.yaml`** - Unauthorized AI tool usage monitoring

## 🏢 Industry-Specific Templates

### Healthcare
- **`healthcare-industry.yaml`** - HIPAA compliance patterns
- Focus: PHI protection, medical data classification

### Financial Services  
- **`financial-services.yaml`** - Financial compliance rules
- Focus: PCI DSS, SOX compliance, financial data protection

### Legal
- **`legal-industry.yaml`** - Legal industry patterns
- Focus: Attorney-client privilege, legal document protection

### General Security
- **`comprehensive-security.yaml`** - Full security rule template

## 🎯 Composition Examples

### Priority-Based Processing
- **`example-composition-priority-order.yaml`** - Rule priority examples

### Strategy Examples
```yaml
# First match (fast)
composition:
  strategy: first_match

# All matches (comprehensive)  
composition:
  strategy: all_matches

# Priority order (custom)
composition:
  strategy: priority_order
```

## 🔧 Usage Patterns

### Copy and Customize
```bash
# Start with an industry template
cp docs/samples/healthcare-industry.yaml rules/my-healthcare-rules.yaml
# Edit and deploy
PS_ENFORCER_RULEPACK=rules/my-healthcare-rules.yaml ./ps-enforcer
```

### Build on Production Rules
```yaml
# Pull in essentials and an industry pack (paths are relative to this file)
imports:
  - ../rules/essentials.yaml
  - ./healthcare-industry.yaml
```

### Test Before Deploy
```bash
# Validate custom rules
curl -X POST http://localhost:9090/v1/rulepacks/validate \
  -H "Authorization: Bearer admin-token" \
  --data-binary @my-custom-rules.yaml
```

## 📖 Rule Development Guide

### 1. Start Simple
```yaml
apiVersion: promptshield.io/v1
kind: RulePack
metadata:
  name: my-rules
rules:
  - id: simple-keyword
    level: 1
    keywords: ["sensitive term"]
```

### 2. Add Patterns
```yaml
  - id: pattern-rule
    level: 2
    patterns:
      - regex: "\\b[A-Z]{3}-\\d{6}\\b"
```

### 3. Context Conditions
```yaml
  - id: conditional-rule
    level: 1
    keywords: ["internal"]
    when:
      match:
        direction: ["request"]
```

### 4. Response Actions
```yaml
response:
  action: quarantine  # or block, warn, flag, redact, replace
  message: "Custom violation message"
```

## 🎨 Customization Examples

### Industry Adaptation
```yaml
# Financial services adaptation
imports:
  - ../rules/comprehensive.yaml

rules:
  - id: fin-account-number
    name: Bank account detection
    level: 2
    severity: CRITICAL
    patterns:
      - regex: "\\b\\d{8,17}\\b"
    when:
      match:
        direction: ["request"]
```

### Performance Tuning
```yaml
performance:
  max_length: 32768        # Skip regex on longer lines
  per_rule_timeout: 50ms   # Faster timeout
```

### Response Customization
```yaml
  - id: custom-response
    keywords: ["confidential"]
    response:
      action: replace
      replacement: "[CONFIDENTIAL_CONTENT_REMOVED]"
      message: "Confidential content detected and removed"
```

## 🧪 Testing & Validation

### Local Testing
```bash
# Test against sample text
echo "test content with sk-1234567890abcdef" | \
  curl -X POST http://localhost:9090/v1/check \
  -H "Content-Type: text/plain" --data-binary @-
```

### Rule Validation
```bash
# Validate YAML syntax and rule logic
curl -X POST http://localhost:9090/v1/rulepacks/validate \
  --data-binary @docs/samples/my-rules.yaml
```

## 📝 Best Practices

1. **Start with production rules** (`essentials.yaml` or `comprehensive.yaml`)
2. **Extend, don't replace** - use `extends` to build on proven rules
3. **Test incrementally** - add rules one at a time
4. **Monitor performance** - watch for timeout issues
5. **Validate before deploy** - use the validation API
6. **Document custom rules** - include clear descriptions

## 🔗 Related Documentation

- **`docs/RulePacks.md`** - Complete DSL reference
- **`rules/README.md`** - Production rule pack guide
- **API docs** - `/v1/rulepacks` endpoints for runtime management