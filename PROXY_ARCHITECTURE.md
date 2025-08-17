# PromptShield Provider-Agnostic Proxy Architecture

## Core Concept: Bring Your Own Key (BYOK)

PromptShield acts as a transparent security proxy between applications and ANY LLM provider. Customers bring their own API keys and configure which providers to use.

## Architecture Flow

```
┌─────────────┐    ┌─────────────────────┐    ┌─────────────────┐
│ Application │    │   PromptShield      │    │ LLM Provider    │
│             │───▶│   API Gateway       │───▶│ (Customer's     │
│             │    │                     │    │  Choice)        │
└─────────────┘    │ • Policy Check      │    └─────────────────┘
                   │ • Key Selection     │           │
                   │ • Request Proxy     │           │
                   │ • Response Filter   │           │
                   └─────────────────────┘           │
                              │                      │
                              ▼                      │
                   ┌─────────────────────┐           │
                   │ Tenant Key Store    │           │
                   │ • OpenAI Keys       │           │
                   │ • Anthropic Keys    │           │
                   │ • Azure OpenAI      │           │
                   │ • Custom Providers  │───────────┘
                   └─────────────────────┘
```

## API Design

### 1. Provider-Agnostic Proxy Endpoints

```
POST   /v1/proxy/chat/completions           # Universal chat endpoint
POST   /v1/proxy/completions                # Universal completion endpoint
POST   /v1/proxy/embeddings                 # Universal embeddings endpoint
POST   /v1/proxy/{provider}/{endpoint}      # Direct provider proxy

# Headers:
# X-PS-Provider: openai|anthropic|azure|custom
# X-PS-Tenant-ID: {tenant-uuid}
# Authorization: Bearer {ps-token}
```

### 2. Provider Key Management

```
POST   /v1/admin/providers/keys             # Register API key for tenant
GET    /v1/admin/providers/keys             # List tenant's registered keys
PUT    /v1/admin/providers/keys/{keyId}     # Update/rotate key
DELETE /v1/admin/providers/keys/{keyId}     # Remove key

POST   /v1/admin/providers/test             # Test provider connectivity
GET    /v1/admin/providers/supported        # List supported providers
```

### 3. Provider Configuration

```
POST   /v1/admin/providers/config           # Configure provider settings
GET    /v1/admin/providers/config           # Get provider config
PUT    /v1/admin/providers/routing          # Configure routing rules
```

## Key Management Strategy

### Per-Tenant Provider Keys
```json
{
  "tenant_id": "uuid",
  "provider_keys": {
    "openai": {
      "key_id": "openai-prod-key",
      "encrypted_key": "encrypted_api_key",
      "key_alias": "production",
      "default": true,
      "created_at": "timestamp",
      "last_used": "timestamp"
    },
    "anthropic": {
      "key_id": "anthropic-prod-key", 
      "encrypted_key": "encrypted_api_key",
      "key_alias": "production",
      "default": true
    },
    "azure_openai": {
      "key_id": "azure-east-key",
      "encrypted_key": "encrypted_api_key",
      "endpoint": "https://myorg.openai.azure.com/",
      "deployment": "gpt-4-deployment"
    }
  }
}
```

### Provider Routing Rules
```json
{
  "tenant_id": "uuid",
  "routing_rules": [
    {
      "priority": 1,
      "conditions": {
        "model": "gpt-4*",
        "route": "/v1/proxy/chat/completions"
      },
      "target": {
        "provider": "openai",
        "key_alias": "production"
      }
    },
    {
      "priority": 2,
      "conditions": {
        "model": "claude-*"
      },
      "target": {
        "provider": "anthropic", 
        "key_alias": "production"
      }
    },
    {
      "priority": 999,
      "conditions": {},
      "target": {
        "provider": "openai",
        "key_alias": "fallback"
      }
    }
  ]
}
```

## Request Flow

### 1. Application Request
```http
POST /v1/proxy/chat/completions
Authorization: Bearer ps_token_xxx
X-PS-Tenant-ID: tenant-uuid
X-PS-Provider: openai
Content-Type: application/json

{
  "model": "gpt-4",
  "messages": [{"role": "user", "content": "Hello"}]
}
```

### 2. PromptShield Processing
1. **Authentication**: Validate PS token and tenant
2. **Provider Selection**: Use X-PS-Provider header or routing rules
3. **Key Lookup**: Get tenant's API key for selected provider  
4. **Policy Enforcement**: Run pre-request scanning
5. **Request Translation**: Convert to provider-specific format if needed
6. **Proxy Request**: Forward to provider with tenant's API key
7. **Response Filtering**: Run post-response scanning
8. **Response Translation**: Normalize response format
9. **Audit Logging**: Log request/response for compliance
10. **Usage Tracking**: Record token usage and costs

### 3. Provider Response (Normalized)
```http
HTTP/1.1 200 OK
X-PS-Provider-Used: openai
X-PS-Request-ID: req-uuid
X-PS-Tokens-Used: 150
X-PS-Policy-Applied: tenant-policy-v1

{
  "id": "chatcmpl-xxx",
  "model": "gpt-4",
  "choices": [
    {
      "message": {
        "role": "assistant",
        "content": "Hello! How can I help you?"
      }
    }
  ],
  "usage": {
    "prompt_tokens": 10,
    "completion_tokens": 8,
    "total_tokens": 18
  }
}
```

## Benefits of This Architecture

1. **Provider Flexibility**: Customers use any LLM provider
2. **Key Security**: API keys never leave tenant boundaries
3. **Cost Control**: Customers pay providers directly
4. **Vendor Independence**: No lock-in to specific providers
5. **Unified Interface**: Single API for all providers
6. **Policy Consistency**: Same security rules across all providers
7. **Usage Tracking**: Centralized monitoring across providers
8. **Failover Support**: Automatic fallback between providers

## Implementation Strategy

1. **Provider Abstraction Layer**: Common interface for all providers
2. **Key Management Service**: Secure storage and rotation
3. **Request Router**: Intelligent provider selection
4. **Protocol Translator**: Normalize requests/responses
5. **Policy Engine**: Universal security enforcement
6. **Usage Aggregator**: Cross-provider metrics

This approach makes PromptShield a true enterprise API Gateway that customers can deploy with their own infrastructure and keys, while providing consistent security and monitoring across all LLM providers.