# Agent Hardening Patterns Implementation

This document describes the complete implementation of Agent Hardening Patterns in PromptShield, providing advanced security controls for AI agents and LLM applications.

## Overview

Agent Hardening Patterns are security mechanisms designed to protect AI systems from malicious inputs, unauthorized actions, and data leakage. PromptShield now implements all five core patterns:

1. ✅ **Action Selector** - Controls which tools/actions agents can execute
2. ✅ **Dual LLM** - Separates privileged and quarantined execution lanes  
3. ✅ **Plan-Then-Execute** - Validates execution plans before action
4. ✅ **Context Minimization** - Strips sensitive context from requests
5. ✅ **Map-Reduce** - Safely processes large documents in chunks

## Implementation Status

| Pattern | Status | Location | Features |
|---------|--------|----------|----------|
| Action Selector | ✅ Complete | `internal/interfaces/http/api/middleware_agent.go` | Tool allowlists, timeout controls, capability filtering |
| Dual LLM | ✅ Complete | `internal/interfaces/http/api/middleware_agent.go` | Lane-based enforcement, capability restrictions |
| Plan-Then-Execute | ✅ Complete | `internal/interfaces/http/api/middleware_agent.go` | Plan validation, hash verification, step tracking |
| Context Minimization | ✅ Complete | `internal/scanner/context_minimization.go` | Text masking, pattern retention, strip points |
| Map-Reduce | ✅ Complete | `internal/scanner/map_reduce.go` | Document chunking, parallel processing, result aggregation |

## Architecture Integration

### Scanner Integration

The patterns are integrated into the core scanning engine:

```go
// Scanner with agent patterns
type Scanner struct {
    // ... existing fields ...
    contextMinimizer   *ContextMinimizer
    mapReduceProcessor *MapReduceProcessor
}
```

### Rule-Based Configuration

Patterns are configured via YAML rulepacks:

```yaml
patterns:
  action_selector:
    enabled: true
    allowed_tool_query: "capability_tags:read"
    per_action_timeout_ms: 5000
    
  context_minimization:
    enabled: true
    strip_point: "after_tool_selection"
    mask_token: "<MASKED>"
    retain: ['\b\w+@\w+\.\w+\b']  # preserve emails
    
  map_reduce:
    enabled: true
    map_unit: "paragraph"
    reduce_type: "union"
```

## Usage Examples

### 1. Action Selector

Restricts agent tool usage based on capability tags:

```yaml
patterns:
  action_selector:
    enabled: true
    mode: "allowlist"
    allowed_tool_query: "capability_tags:read OR data_domains:public"
    per_action_timeout_ms: 5000
```

**Headers for enforcement:**
- `X-PS-Tool-ID`: Tool identifier
- `X-PS-Tenant-ID`: Tenant context

### 2. Context Minimization

Masks sensitive content while preserving specified patterns:

```go
minimizer := scanner.NewContextMinimizer(&rules.ContextMinimization{
    Enabled:    true,
    StripPoint: "after_tool_selection",
    MaskToken:  "<REDACTED>",
    Retain:     []string{`\b\w+@\w+\.\w+\b`}, // keep emails
})

result, err := minimizer.MinimizeContext(content, "")
```

### 3. Map-Reduce Processing

Safely processes large documents by chunking:

```go
processor := scanner.NewMapReduceProcessor(&rules.MapReduce{
    Enabled:       true,
    MapUnit:       "paragraph",
    TextMaxTokens: 2000,
    ReduceType:    "union",
})

result, err := processor.ProcessDocument(ctx, largeDocument, scanner)
```

### 4. Plan-Then-Execute

Validates execution plans against predefined steps:

```yaml
patterns:
  plan_then_execute:
    enabled: true
    max_steps: 10
    drift_policy: "strict"
```

**Required headers:**
- `X-PS-Plan`: JSON execution plan
- `X-PS-Plan-Hash`: SHA-256 hash of plan
- `X-PS-Plan-Step`: Current step index

### 5. Dual LLM

Separates execution into privileged and quarantined lanes:

```yaml
patterns:
  dual_llm:
    enabled: true
    quarantined_tools_disabled: false
    bridge_handles_only: true
```

**Lane control:**
- `X-PS-Lane`: "privileged" or "quarantined"

## Enforcement Points

### HTTP Middleware

Agent patterns are enforced via HTTP middleware that intercepts requests with agent headers:

```go
func agentEnforcementMiddleware(opt Options) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            toolID := r.Header.Get("X-PS-Tool-ID")
            if toolID == "" {
                next.ServeHTTP(w, r) // No agent context
                return
            }
            
            // Apply agent hardening patterns...
        })
    }
}
```

### Scanner Integration

Context minimization and map-reduce are applied during content scanning:

```go
func (s *Scanner) ScanContent(ctx context.Context, content string, inputName string) (types.ScanResult, error) {
    // Apply context minimization
    if s.contextMinimizer != nil && s.contextMinimizer.IsEnabled() {
        minimized, err := s.contextMinimizer.MinimizeContext(content, "")
        if err != nil {
            return types.ScanResult{}, err
        }
        content = minimized
    }
    
    // Apply map-reduce for large documents
    if s.mapReduceProcessor != nil && s.mapReduceProcessor.IsEnabled() {
        return s.mapReduceProcessor.ProcessDocument(ctx, content, s)
    }
    
    // Standard processing
    return s.scanContentDirect(ctx, content, inputName)
}
```

## Configuration Examples

### Complete Agent Hardening Configuration

See `examples/agent_hardening_demo.yaml` for a comprehensive example that demonstrates all patterns working together.

### Production Deployment

For production use, configure patterns based on your security requirements:

```yaml
# High-security research agent
patterns:
  action_selector:
    enabled: true
    allowed_tool_query: "capability_tags:read AND data_domains:public"
    
  context_minimization:
    enabled: true
    strip_point: "before_execution"
    retain: ['\b[A-Z]{2,}\b']  # preserve acronyms only
    
  plan_then_execute:
    enabled: true
    max_steps: 5
    drift_policy: "strict"
    
  dual_llm:
    enabled: true
    quarantined_tools_disabled: true  # no tools on quarantined lane
```

## Testing

Comprehensive tests are provided in:
- `internal/scanner/agent_patterns_test.go` - Pattern-specific tests
- `internal/interfaces/http/api/middleware_agent_test.go` - Integration tests

Run tests:
```bash
go test ./internal/scanner -v -run TestAgent
go test ./internal/interfaces/http/api -v -run TestAgent
```

## Performance Considerations

### Context Minimization
- Minimal overhead for small documents
- Regex processing scales with content size
- Caching of compiled patterns recommended for high throughput

### Map-Reduce
- Parallel processing improves performance for large documents
- Memory usage bounded by chunk size
- Reduce strategies affect final result size

### Action Selector
- Database queries for tool registry lookups
- Consider caching tool definitions for high-frequency usage

## Security Properties

### Threat Mitigation

1. **Prompt Injection**: Action Selector prevents unauthorized tool usage
2. **Data Exfiltration**: Context Minimization strips sensitive information
3. **Plan Deviation**: Plan-Then-Execute enforces execution constraints
4. **Privilege Escalation**: Dual LLM isolates dangerous operations
5. **Resource Exhaustion**: Map-Reduce bounds memory and processing time

### Defense in Depth

Agent Hardening Patterns work alongside traditional security rules:
- Standard keyword/regex/semantic rules catch known threats
- Agent patterns provide behavioral controls for AI systems
- Combined approach offers comprehensive protection

## Future Enhancements

Potential improvements for future versions:

1. **Dynamic Pattern Selection**: Choose patterns based on request context
2. **Machine Learning Integration**: Use ML models for pattern optimization
3. **Advanced Semantic Chunking**: Improve map-reduce chunking with embeddings
4. **Real-time Pattern Updates**: Hot-reload pattern configurations
5. **Cross-Request Context**: Maintain context across related requests

## Conclusion

The complete implementation of Agent Hardening Patterns provides PromptShield with state-of-the-art protection for AI agent systems. These patterns work together to create a comprehensive security framework that protects against both traditional and AI-specific threats.

For questions or contributions, see the main PromptShield documentation and contribution guidelines.
