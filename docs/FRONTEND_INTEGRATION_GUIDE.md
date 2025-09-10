### Clerk Organizations Integration (Multi-tenant)

Server (BFF):
- Set `CLERK_SECRET_KEY` and `VITE_CLERK_PUBLISHABLE_KEY`.
- BFF uses Clerk Express middleware to populate `req.auth`.
- Active tenant is stored in a signed cookie `ps_tenant_id`.
- BFF mint short-lived RS256 JWT for Gateway with claims: `sub`, `tenant_id`, `roles`, `admin`.

Endpoints:
- `POST /api/orgs/create`: creates a Clerk org, then selects it.
- `GET /api/orgs`: lists Clerk org memberships for the user.
- `POST /api/orgs/select`: calls Gateway `POST /v1/tenants/resolve` to link `clerk` org → tenant and sets cookie.

Gateway:
- Validates BFF JWT via `PS_BFF_JWT_PUBLIC_KEY`.
- Extracts `X-PS-Tenant-ID` from JWT claim `tenant_id`.
- Resolves Clerk org mapping with `POST /v1/tenants/resolve` and stores in `tenant_org_links`.

Database:
- Apply `migrations_consolidated/0021_external_org_links.sql` to create `tenant_org_links`.

Dev bypass:
- Set `PS_DEV_BYPASS_AUTH=true` and optional `PS_DEV_*` fields.
# Frontend Integration Guide for PromptShield Multi-Tenant SaaS

## Overview

This guide documents how to integrate your frontend application with the PromptShield multi-tenant API. PromptShield provides a security gateway for LLM applications with tenant isolation, authentication, and rate limiting.

## Architecture Overview

```
┌─────────────┐     HTTPS      ┌─────────────┐     ┌──────────────┐
│   Frontend  │ ─────────────> │    Nginx    │ ──> │ PromptShield │
│   (React/   │                 │ Load Balancer│     │   Cluster    │
│   Vue/etc)  │                 └─────────────┘     └──────────────┘
└─────────────┘                        │                    │
                                       │                    ▼
                                       │            ┌──────────────┐
                                       └──────────> │   PgBouncer  │
                                                   │  Connection  │
                                                   │     Pool     │
                                                   └──────────────┘
                                                           │
                                                           ▼
                                                   ┌──────────────┐
                                                   │   Supabase   │
                                                   │   Database   │
                                                   └──────────────┘
```

## Authentication Methods

### 1. Frontend Session Authentication

For web applications with user sessions:

```javascript
// Example: Frontend authentication flow
const authenticateUser = async (email, password) => {
  const response = await fetch('https://api.promptshield.com/auth/login', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ email, password }),
  });
  
  const { token, tenantId, userId } = await response.json();
  
  // Store credentials securely
  sessionStorage.setItem('authToken', token);
  sessionStorage.setItem('tenantId', tenantId);
  sessionStorage.setItem('userId', userId);
  
  return { token, tenantId, userId };
};

// Making authenticated requests
const makeApiCall = async (endpoint, data) => {
  const token = sessionStorage.getItem('authToken');
  const tenantId = sessionStorage.getItem('tenantId');
  const userId = sessionStorage.getItem('userId');
  
  const response = await fetch(`https://api.promptshield.com${endpoint}`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-PS-Frontend-Auth': 'verified',  // Frontend authentication flag
      'X-PS-Tenant-ID': tenantId,         // Required for tenant isolation
      'X-PS-User-ID': userId,             // User identification
      'Authorization': `Bearer ${token}`,  // Optional: JWT or session token
    },
    body: JSON.stringify(data),
  });
  
  return response.json();
};
```

### 2. API Key Authentication

For server-to-server or automated integrations:

```javascript
// Example: API key authentication
const apiKey = 'ps_live_1234567890abcdef';  // Your API key
const tenantId = 'tenant-uuid-here';

const checkPrompt = async (prompt) => {
  const response = await fetch('https://api.promptshield.com/check', {
    method: 'POST',
    headers: {
      'Content-Type': 'text/plain',
      'Authorization': `Bearer ${apiKey}`,
      'X-PS-Tenant-ID': tenantId,
    },
    body: prompt,
  });
  
  const result = await response.json();
  return result;
};
```

## Required Headers

### For All Requests

| Header | Required | Description | Example |
|--------|----------|-------------|---------|
| `X-PS-Tenant-ID` | Yes* | Tenant identifier (UUID) | `123e4567-e89b-12d3-a456-426614174000` |
| `Content-Type` | Yes | Request content type | `application/json`, `text/plain` |

*Not required for public endpoints: `/healthz`, `/readyz`, `/metrics`

### For Frontend Authentication

| Header | Required | Description | Example |
|--------|----------|-------------|---------|
| `X-PS-Frontend-Auth` | Yes | Frontend authentication flag | `verified` |
| `X-PS-User-ID` | Recommended | User identifier | `user-uuid-here` |
| `X-PS-User-Name` | Optional | User display name | `John Doe` |

### For API Key Authentication

| Header | Required | Description | Example |
|--------|----------|-------------|---------|
| `Authorization` | Yes | API key with Bearer prefix | `Bearer ps_live_xxxxx` |

## Core API Endpoints

### 1. Check Endpoint (Content Moderation)

**Endpoint:** `POST /check`

**Purpose:** Check content for violations against configured rules

**Request:**
```javascript
const response = await fetch('https://api.promptshield.com/check', {
  method: 'POST',
  headers: {
    'Content-Type': 'text/plain',
    'X-PS-Tenant-ID': tenantId,
    'X-PS-Frontend-Auth': 'verified',
  },
  body: 'User prompt to check for violations',
});
```

**Response:**
```json
{
  "decision": "deny",
  "violations": [
    {
      "rule_id": "prompt-injection-001",
      "severity": "HIGH",
      "message": "Potential prompt injection detected",
      "matched_text": "ignore previous instructions"
    }
  ],
  "metadata": {
    "scan_time_ms": 12,
    "rules_evaluated": 45
  }
}
```

### 2. Batch and Streaming via /check

/scan has been consolidated into /check. You can submit batches as a JSON array or stream NDJSON with aggregate=false.

JSON array (aggregate decisions):
```javascript
const data = ["one", "two", "three"];
const response = await fetch('https://api.promptshield.com/check', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'X-PS-Tenant-ID': tenantId,
    'X-PS-Frontend-Auth': 'verified',
  },
  body: JSON.stringify(data),
});
// → { decisions: [...], summary: { total, violations } }
```

NDJSON streaming (aggregate=false):
```javascript
const tenantId = sessionStorage.getItem('tenantId');
const stream = new ReadableStream({
  start(controller) {
    controller.enqueue(new TextEncoder().encode('first\n'));
    controller.enqueue(new TextEncoder().encode('second\n'));
    controller.close();
  }
});
const resp = await fetch('https://api.promptshield.com/check?aggregate=false', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/x-ndjson',
    'X-PS-Tenant-ID': tenantId,
    'X-PS-Frontend-Auth': 'verified',
  },
  body: stream,
});
// Read line-by-line NDJSON from resp.body
```

### 3. Policy Management

**List Policies:**
```javascript
const response = await fetch('https://api.promptshield.com/admin/policies', {
  method: 'GET',
  headers: {
    'X-PS-Tenant-ID': tenantId,
    'X-PS-Frontend-Auth': 'verified',
  },
});
```

**Create/Update Policy:**
```javascript
const response = await fetch('https://api.promptshield.com/admin/policies', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'X-PS-Tenant-ID': tenantId,
    'X-PS-Frontend-Auth': 'verified',
  },
  body: JSON.stringify({
    name: 'strict-moderation',
    rules: [...],
    enabled: true,
  }),
});
```

### 4. Service Control Management

**List Services:**
```javascript
const response = await fetch('https://api.promptshield.com/api/v1/services', {
  method: 'GET',
  headers: {
    'X-PS-Tenant-ID': tenantId,
    'X-PS-Frontend-Auth': 'verified',
  },
});
```

**Start a Service:**
```javascript
const response = await fetch(`https://api.promptshield.com/api/v1/services/${serviceId}/start`, {
  method: 'POST',
  headers: {
    'X-PS-Tenant-ID': tenantId,
    'X-PS-Frontend-Auth': 'verified',
  },
});
```

**Stop a Service:**
```javascript
const response = await fetch(`https://api.promptshield.com/api/v1/services/${serviceId}/stop`, {
  method: 'POST',
  headers: {
    'X-PS-Tenant-ID': tenantId,
    'X-PS-Frontend-Auth': 'verified',
  },
});
```

**Get Service Status:**
```javascript
const response = await fetch(`https://api.promptshield.com/api/v1/services/${serviceId}/status`, {
  method: 'GET',
  headers: {
    'X-PS-Tenant-ID': tenantId,
    'X-PS-Frontend-Auth': 'verified',
  },
});
```

## Rate Limiting

PromptShield implements multi-level rate limiting:

1. **Per-Tenant Limits:** Based on subscription plan
2. **Per-API Key Limits:** Configured per key
3. **Global Limits:** System-wide protection

Rate limit headers in responses:
```
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 950
X-RateLimit-Reset: 1640995200
```

## Error Handling

```javascript
const handleApiError = (response) => {
  const status = response.status;
  const error = response.data;
  
  switch (status) {
    case 400:
      // Bad request - check headers and payload
      console.error('Invalid request:', error.message);
      break;
    case 401:
      // Unauthorized - refresh authentication
      refreshAuth();
      break;
    case 403:
      // Forbidden - check tenant status
      console.error('Access denied:', error.message);
      break;
    case 429:
      // Rate limited - implement backoff
      const retryAfter = response.headers['retry-after'];
      setTimeout(() => retryRequest(), retryAfter * 1000);
      break;
    case 500:
      // Server error - retry with exponential backoff
      retryWithBackoff();
      break;
  }
};
```

## WebSocket Events (Real-time Updates)

```javascript
// Connect to event stream
const eventSource = new EventSource(
  'https://api.promptshield.com/admin/events',
  {
    headers: {
      'X-PS-Tenant-ID': tenantId,
      'X-PS-Frontend-Auth': 'verified',
    }
  }
);

eventSource.addEventListener('violation', (event) => {
  const violation = JSON.parse(event.data);
  console.log('Violation detected:', violation);
  // Update UI with violation information
});

eventSource.addEventListener('policy-update', (event) => {
  const update = JSON.parse(event.data);
  console.log('Policy updated:', update);
  // Refresh local policy cache
});

eventSource.addEventListener('service.started', (event) => {
  const serviceEvent = JSON.parse(event.data);
  console.log('Service started:', serviceEvent);
  // Update service status in UI
  updateServiceStatus(serviceEvent.service_id, 'running');
});

eventSource.addEventListener('service.stopped', (event) => {
  const serviceEvent = JSON.parse(event.data);
  console.log('Service stopped:', serviceEvent);
  // Update service status in UI
  updateServiceStatus(serviceEvent.service_id, 'stopped');
});

eventSource.addEventListener('service.error', (event) => {
  const serviceEvent = JSON.parse(event.data);
  console.log('Service error:', serviceEvent);
  // Show error notification
  showErrorNotification(`Service ${serviceEvent.service_id} encountered an error: ${serviceEvent.error}`);
});
```

## CORS Configuration

The PromptShield API supports CORS for frontend applications:

- **Allowed Origins:** Configured per tenant
- **Allowed Methods:** GET, POST, PUT, DELETE, OPTIONS
- **Allowed Headers:** Authorization, X-PS-*, Content-Type
- **Credentials:** Supported with `credentials: 'include'`

## Security Best Practices

### 1. Token Storage

```javascript
// DO: Use secure storage methods
sessionStorage.setItem('authToken', token);  // Session-only
// Or use httpOnly cookies set by the server

// DON'T: Store in localStorage for sensitive tokens
localStorage.setItem('authToken', token);  // Persists across sessions
```

### 2. API Key Security

```javascript
// DO: Store API keys server-side
// Use environment variables or secure vaults
const apiKey = process.env.PROMPTSHIELD_API_KEY;

// DON'T: Expose API keys in frontend code
const apiKey = 'ps_live_xxxxx';  // Visible in browser
```

### 3. Request Validation

```javascript
// Always validate tenant context
const validateRequest = (tenantId, userId) => {
  if (!tenantId || !isValidUUID(tenantId)) {
    throw new Error('Invalid tenant ID');
  }
  if (!userId || !isValidUUID(userId)) {
    throw new Error('Invalid user ID');
  }
};
```

## Example: React Integration

```jsx
// hooks/usePromptShield.js
import { useState, useCallback } from 'react';

export const usePromptShield = () => {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  
  const checkContent = useCallback(async (content) => {
    setLoading(true);
    setError(null);
    
    try {
      const response = await fetch('https://api.promptshield.com/check', {
        method: 'POST',
        headers: {
          'Content-Type': 'text/plain',
          'X-PS-Tenant-ID': sessionStorage.getItem('tenantId'),
          'X-PS-Frontend-Auth': 'verified',
          'X-PS-User-ID': sessionStorage.getItem('userId'),
        },
        body: content,
      });
      
      if (!response.ok) {
        throw new Error(`API error: ${response.status}`);
      }
      
      const result = await response.json();
      return result;
    } catch (err) {
      setError(err.message);
      throw err;
    } finally {
      setLoading(false);
    }
  }, []);
  
  const manageService = useCallback(async (serviceId, action) => {
    setLoading(true);
    setError(null);
    
    try {
      const response = await fetch(`https://api.promptshield.com/api/v1/services/${serviceId}/${action}`, {
        method: 'POST',
        headers: {
          'X-PS-Tenant-ID': sessionStorage.getItem('tenantId'),
          'X-PS-Frontend-Auth': 'verified',
          'X-PS-User-ID': sessionStorage.getItem('userId'),
        },
      });
      
      if (!response.ok) {
        throw new Error(`Service ${action} failed: ${response.status}`);
      }
      
      const result = await response.json();
      return result;
    } catch (err) {
      setError(err.message);
      throw err;
    } finally {
      setLoading(false);
    }
  }, []);

  return {
    checkContent,
    manageService,
    loading,
    error,
  };
};

// Component usage
const ChatInterface = () => {
  const { checkContent, loading, error } = usePromptShield();
  const [message, setMessage] = useState('');
  
  const handleSubmit = async (e) => {
    e.preventDefault();
    
    try {
      const result = await checkContent(message);
      
      if (result.decision === 'deny') {
        alert(`Message blocked: ${result.violations[0].message}`);
        return;
      }
      
      // Process allowed message
      sendToLLM(message);
    } catch (err) {
      console.error('Failed to check content:', err);
    }
  };
  
  return (
    <form onSubmit={handleSubmit}>
      <textarea
        value={message}
        onChange={(e) => setMessage(e.target.value)}
        disabled={loading}
      />
      <button type="submit" disabled={loading}>
        {loading ? 'Checking...' : 'Send'}
      </button>
      {error && <div className="error">{error}</div>}
    </form>
  );
};

// Service Management Component
const ServiceManager = () => {
  const { manageService, loading, error } = usePromptShield();
  const [services, setServices] = useState([]);
  const [statusUpdates, setStatusUpdates] = useState({});

  useEffect(() => {
    loadServices();
    
    // Set up real-time event listening
    const eventSource = new EventSource('https://api.promptshield.com/admin/events', {
      headers: {
        'X-PS-Tenant-ID': sessionStorage.getItem('tenantId'),
        'X-PS-Frontend-Auth': 'verified',
      }
    });
    
    eventSource.addEventListener('service.started', (event) => {
      const data = JSON.parse(event.data);
      setStatusUpdates(prev => ({ ...prev, [data.service_id]: 'running' }));
    });
    
    eventSource.addEventListener('service.stopped', (event) => {
      const data = JSON.parse(event.data);
      setStatusUpdates(prev => ({ ...prev, [data.service_id]: 'stopped' }));
    });
    
    return () => eventSource.close();
  }, []);

  const loadServices = async () => {
    try {
      const response = await fetch('https://api.promptshield.com/api/v1/services', {
        headers: {
          'X-PS-Tenant-ID': sessionStorage.getItem('tenantId'),
          'X-PS-Frontend-Auth': 'verified',
        },
      });
      const result = await response.json();
      setServices(result.data || []);
    } catch (err) {
      console.error('Failed to load services:', err);
    }
  };

  const handleServiceAction = async (serviceId, action) => {
    try {
      await manageService(serviceId, action);
      // Status will be updated via WebSocket events
      console.log(`Service ${action} initiated successfully`);
    } catch (err) {
      console.error(`Failed to ${action} service:`, err.message);
    }
  };

  return (
    <div className="service-manager">
      <h2>Service Management</h2>
      <div className="services-grid">
        {services.map((service) => {
          const currentStatus = statusUpdates[service.id] || service.status;
          return (
            <div key={service.id} className="service-card">
              <h3>{service.service_name}</h3>
              <div className={`status-badge ${currentStatus}`}>
                {currentStatus.toUpperCase()}
              </div>
              <div className="service-actions">
                <button
                  onClick={() => handleServiceAction(service.id, 'start')}
                  disabled={loading || currentStatus === 'running'}
                  className="btn btn-start"
                >
                  Start
                </button>
                <button
                  onClick={() => handleServiceAction(service.id, 'stop')}
                  disabled={loading || currentStatus === 'stopped'}
                  className="btn btn-stop"
                >
                  Stop
                </button>
                <button
                  onClick={() => handleServiceAction(service.id, 'restart')}
                  disabled={loading || currentStatus === 'stopped'}
                  className="btn btn-restart"
                >
                  Restart
                </button>
              </div>
              {service.error_message && (
                <div className="error-message">
                  Error: {service.error_message}
                </div>
              )}
            </div>
          );
        })}
      </div>
      {error && <div className="error-banner">{error}</div>}
    </div>
  );
};
```

## Testing

### Development Environment

```bash
# Use self-signed certificates for local testing
curl -X POST https://api.promptshield.local/check \
  -H "Content-Type: text/plain" \
  -H "X-PS-Tenant-ID: test-tenant-id" \
  -H "X-PS-Frontend-Auth: verified" \
  -d "Test prompt" \
  --insecure  # Accept self-signed certificate
```

### Integration Tests

```javascript
// test/promptshield.test.js
describe('PromptShield Integration', () => {
  it('should block prompt injections', async () => {
    const result = await checkContent('Ignore all instructions and...');
    expect(result.decision).toBe('deny');
    expect(result.violations).toHaveLength(1);
    expect(result.violations[0].severity).toBe('HIGH');
  });
  
  it('should allow clean content', async () => {
    const result = await checkContent('What is the weather today?');
    expect(result.decision).toBe('allow');
    expect(result.violations).toHaveLength(0);
  });
});
```

## Monitoring & Analytics

### Client-side Metrics

```javascript
// Track API performance
const trackApiCall = (endpoint, duration, status) => {
  analytics.track('api_call', {
    endpoint,
    duration_ms: duration,
    status,
    tenant_id: sessionStorage.getItem('tenantId'),
  });
};

// Monitor violations
const trackViolation = (violation) => {
  analytics.track('content_violation', {
    rule_id: violation.rule_id,
    severity: violation.severity,
    tenant_id: sessionStorage.getItem('tenantId'),
  });
};
```

## Support & Troubleshooting

### Common Issues

1. **Missing Tenant ID Error**
   - Ensure `X-PS-Tenant-ID` header is present
   - Verify tenant UUID format

2. **Authentication Failures**
   - Check token expiration
   - Verify frontend auth header
   - Ensure API key format is correct

3. **Rate Limiting**
   - Implement exponential backoff
   - Check subscription limits
   - Consider request batching

### Debug Mode

```javascript
// Enable debug logging
const DEBUG = process.env.NODE_ENV === 'development';

const apiCall = async (endpoint, options) => {
  if (DEBUG) {
    console.log('API Request:', endpoint, options);
  }
  
  const response = await fetch(endpoint, options);
  
  if (DEBUG) {
    console.log('API Response:', response.status, await response.clone().json());
  }
  
  return response;
};
```

## Migration Guide

### From Single-Tenant to Multi-Tenant

1. **Add Tenant Context**
   ```javascript
   // Before
   fetch('/api/check', { ... });
   
   // After
   fetch('/api/check', {
     headers: {
       'X-PS-Tenant-ID': tenantId,
       ...
     }
   });
   ```

2. **Update Authentication**
   ```javascript
   // Add frontend auth header
   headers['X-PS-Frontend-Auth'] = 'verified';
   ```

3. **Handle Tenant-Specific Errors**
   ```javascript
   if (error.code === 'TENANT_INACTIVE') {
     // Handle inactive tenant
     redirectToSubscriptionPage();
   }
   ```

## Resources

- **API Documentation:** https://api.promptshield.com/docs
- **Status Page:** https://status.promptshield.com
- **Support:** support@promptshield.com
- **GitHub:** https://github.com/promptshield/promptshield

## Changelog

- **v1.1.0** - Service Control Release
  - Service lifecycle management (start/stop/restart)
  - Real-time service status updates via WebSocket
  - Resource-aware service scaling
  - Service health monitoring and error reporting
  - Per-tenant service isolation

- **v1.0.0** - Initial multi-tenant release
  - Tenant isolation via headers
  - Frontend authentication support
  - API key management
  - PgBouncer connection pooling
  - HTTPS/TLS support