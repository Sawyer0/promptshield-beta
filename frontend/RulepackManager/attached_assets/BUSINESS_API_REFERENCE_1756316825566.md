# PromptShield Business API Reference

**For SaaS Platform Owners**

This document covers all administrative endpoints available to PromptShield platform owners for managing their SaaS business operations.

---

## Authentication

All business endpoints require admin authentication:

```bash
# Primary method - Bearer token
Authorization: Bearer <your-admin-token>

# Alternative method - Custom header
X-PS-Admin-Token: <your-admin-token>

# Development only - bypasses auth (if PS_DEV_MODE=true)
X-PS-Frontend-Auth: verified
```

---

## 1. Tenant Management

Manage your customer accounts (tenants) across your platform.

### Create Tenant
Create a new customer account when they sign up for your service.

```http
POST /admin/tenants
```

**Request Body:**
```json
{
  "name": "Acme Corporation",
  "email": "admin@acme.com", 
  "plan": "enterprise",
  "status": "active",
  "settings": {
    "max_requests_per_day": 100000,
    "max_rulepacks": 50,
    "semantic_analysis_enabled": true
  }
}
```

**Response (201):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "Acme Corporation",
  "email": "admin@acme.com",
  "plan": "enterprise", 
  "status": "active",
  "settings": {
    "max_requests_per_day": 100000,
    "max_rulepacks": 50,
    "semantic_analysis_enabled": true
  },
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:00Z"
}
```

### List All Tenants
Get a list of all your customers with pagination support.

```http
GET /admin/tenants?limit=50&offset=0&status=active
```

**Query Parameters:**
- `limit` (optional): Number of results (default: 50, max: 1000)
- `offset` (optional): Pagination offset (default: 0)  
- `status` (optional): Filter by status (`active`, `suspended`, `deleted`)
- `plan` (optional): Filter by plan (`free`, `pro`, `enterprise`)

**Response (200):**
```json
{
  "tenants": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "Acme Corporation",
      "email": "admin@acme.com",
      "plan": "enterprise",
      "status": "active",
      "user_count": 15,
      "rulepack_count": 8,
      "last_active": "2024-01-15T09:45:00Z",
      "created_at": "2024-01-10T14:20:00Z"
    }
  ],
  "total_count": 250,
  "has_more": true,
  "pagination": {
    "limit": 50,
    "offset": 0,
    "next_offset": 50
  }
}
```

### Get Tenant Details
Get detailed information about a specific customer.

```http
GET /admin/tenants/{tenant_id}
```

**Response (200):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "Acme Corporation",
  "email": "admin@acme.com",
  "plan": "enterprise",
  "status": "active",
  "settings": {
    "max_requests_per_day": 100000,
    "max_rulepacks": 50,
    "semantic_analysis_enabled": true
  },
  "usage_stats": {
    "requests_today": 45230,
    "requests_this_month": 1250000,
    "rulepacks_active": 8,
    "last_request": "2024-01-15T10:25:00Z"
  },
  "billing_info": {
    "subscription_status": "active",
    "next_billing_date": "2024-02-01T00:00:00Z",
    "overdue_amount": 0
  },
  "created_at": "2024-01-10T14:20:00Z",
  "updated_at": "2024-01-15T10:30:00Z"
}
```

### Update Tenant
Update customer plan, settings, or status.

```http
PUT /admin/tenants/{tenant_id}
```

**Request Body:**
```json
{
  "plan": "pro",
  "status": "suspended",
  "suspension_reason": "payment_overdue",
  "settings": {
    "max_requests_per_day": 50000,
    "max_rulepacks": 25
  }
}
```

**Response (200):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "Acme Corporation", 
  "plan": "pro",
  "status": "suspended",
  "suspension_reason": "payment_overdue",
  "updated_at": "2024-01-15T10:35:00Z"
}
```

### Delete Tenant
Permanently delete a customer account and all their data.

```http
DELETE /admin/tenants/{tenant_id}
```

**Response (204):** No content

---

## 2. System Health & Monitoring

Monitor your platform's health and performance.

### Get System Features
Check which features are enabled on your platform.

```http
GET /admin/system/features
```

**Response (200):**
```json
{
  "async_jobs": true,
  "l3_semantic": true, 
  "tenant_management": true,
  "audit_logging": true,
  "usage_tracking": true,
  "quota_management": true
}
```

### Get System Statistics
Get platform-wide performance metrics.

```http
GET /admin/system/stats
```

**Response (200):**
```json
{
  "decisions_total": {
    "allow": 8500000,
    "deny": 150000,
    "quarantine": 25000
  },
  "p95_latency_ms": 45.2,
  "requests_total": 8675000,
  "errors_total": 1250,
  "uptime": "15h30m45s",
  "version": "v0.2.0"
}
```

### Get System Information
Get detailed platform information including license and build details.

```http
GET /admin/system/info
```

**Response (200):**
```json
{
  "version": "v0.2.0",
  "commit": "abc123def456",
  "build_date": "2024-01-15T08:00:00Z",
  "go_version": "go1.21.5",
  "platform": "linux/amd64",
  "start_time": "2024-01-15T06:30:00Z",
  "uptime": "15h30m45s",
  "license": {
    "org": "PromptShield SaaS Platform",
    "tier": "enterprise", 
    "expires_at": "2025-12-31T23:59:59Z",
    "licensed": true,
    "entitlements": {
      "max_tenants": 1000,
      "max_requests_per_day": 10000000
    }
  },
  "features": {
    "async_jobs": true,
    "l3_semantic": true,
    "tenant_management": true,
    "audit_logging": true,
    "usage_tracking": true,
    "quota_management": true
  }
}
```

### Drain System
Put the system into maintenance mode (stop accepting new requests).

```http
POST /admin/system/drain
```

**Response (202):**
```json
{
  "message": "System drain initiated",
  "timestamp": "2024-01-15T10:45:00Z"
}
```

### Shutdown System
Gracefully shutdown the platform with optional delay.

```http
POST /admin/system/shutdown?delay=300
```

**Query Parameters:**
- `delay` (optional): Delay in seconds before shutdown (default: 0)

**Response (202):**
```json
{
  "message": "System shutdown initiated",
  "timestamp": "2024-01-15T10:45:00Z",
  "delay_seconds": 300
}
```

---

## 3. Platform Diagnostics

Detailed system diagnostics and debugging information.

### Get System Status
Comprehensive health check of all system components.

```http
GET /debug/status
```

**Response (200):**
```json
{
  "status": "healthy",
  "timestamp": "2024-01-15T10:45:00Z",
  "version": "v0.2.0",
  "uptime": "15h30m45s",
  "components": {
    "database": {
      "status": "healthy",
      "message": "Database connectivity OK",
      "last_check": "2024-01-15T10:45:00Z"
    },
    "license": {
      "status": "healthy", 
      "message": "License valid",
      "last_check": "2024-01-15T10:45:00Z",
      "details": {
        "expires_at": "2025-12-31T23:59:59Z"
      }
    },
    "memory": {
      "status": "healthy",
      "message": "Memory usage normal",
      "last_check": "2024-01-15T10:45:00Z", 
      "details": {
        "used_mb": 512.5,
        "total_mb": 2048.0
      }
    },
    "goroutines": {
      "status": "healthy",
      "message": "Goroutine count normal",
      "last_check": "2024-01-15T10:45:00Z",
      "details": {
        "count": 45
      }
    }
  },
  "metrics": {
    "goroutines": 45,
    "memory_used_mb": 512.5,
    "memory_total_mb": 2048.0,
    "inflight_bytes": 0
  },
  "environment": {
    "go_version": "go1.21.5",
    "platform": "linux/amd64", 
    "num_cpu": 8
  }
}
```

### Get Goroutine Information
Monitor goroutine count and stack traces for debugging.

```http
GET /debug/goroutines?stacks=true
```

**Query Parameters:**
- `stacks` (optional): Include stack traces (`true`/`false`, default: `false`)

**Response (200):**
```json
{
  "count": 45,
  "timestamp": "2024-01-15T10:45:00Z",
  "stacks": "goroutine 1 [running]:\nmain.main()\n..."
}
```

### Get Memory Statistics
Detailed memory usage and garbage collection statistics.

```http
GET /debug/memory
```

**Response (200):**
```json
{
  "timestamp": "2024-01-15T10:45:00Z",
  "alloc_mb": 512.5,
  "total_alloc_mb": 8950.2,
  "sys_mb": 2048.0,
  "num_gc": 127,
  "gc_pause_ns": 15000000,
  "heap_objects": 125430,
  "heap_in_use_mb": 480.2,
  "heap_released_mb": 1024.5,
  "stack_in_use_mb": 32.1
}
```

### Simple Health Check
Basic health check endpoint for load balancers.

```http
GET /debug/health
```

**Response (200):**
```json
{
  "status": "healthy",
  "timestamp": "2024-01-15T10:45:00Z",
  "uptime": "15h30m45s", 
  "version": "v0.2.0",
  "components": {
    "database": "healthy",
    "usage_store": "healthy",
    "quota_store": "healthy"
  }
}
```

---

## 4. Platform-Wide Audit Trail

Monitor and audit activities across all customers.

### List Audit Events
Get platform-wide audit events with filtering.

```http
GET /admin/audits?tenant={tenant_id}&limit=100&offset=0
```

**Query Parameters:**
- `tenant` (optional): Filter by specific tenant ID
- `limit` (optional): Number of results (default: 100, max: 1000)
- `offset` (optional): Pagination offset

**Response (200):**
```json
{
  "events": [
    {
      "id": "audit-550e8400-e29b-41d4-a716-446655440000",
      "tenant_id": "550e8400-e29b-41d4-a716-446655440000",
      "action": "tenant.created",
      "object_type": "tenant",
      "object_id": "550e8400-e29b-41d4-a716-446655440000", 
      "actor_id": "admin-user-id",
      "actor_email": "admin@promptshield.com",
      "metadata": {
        "tenant_name": "Acme Corporation",
        "plan": "enterprise"
      },
      "created_at": "2024-01-15T10:30:00Z"
    }
  ],
  "count": 1,
  "has_more": false,
  "pagination": {
    "limit": 100,
    "offset": 0
  }
}
```

### Search Audit Events
Advanced search across audit events with filters.

```http
POST /admin/audits/search
```

**Request Body:**
```json
{
  "tenant_id": "550e8400-e29b-41d4-a716-446655440000",
  "actions": ["tenant.created", "tenant.suspended", "rulepack.created"],
  "object_types": ["tenant", "rulepack"],
  "actor_email": "admin@acme.com",
  "start_time": "2024-01-01T00:00:00Z",
  "end_time": "2024-01-31T23:59:59Z",
  "limit": 500,
  "offset": 0
}
```

**Response (200):**
```json
{
  "events": [
    {
      "id": "audit-550e8400-e29b-41d4-a716-446655440000",
      "tenant_id": "550e8400-e29b-41d4-a716-446655440000", 
      "action": "rulepack.created",
      "object_type": "rulepack",
      "object_id": "rulepack-123",
      "actor_email": "admin@acme.com",
      "metadata": {
        "rulepack_name": "Anti-Injection Rules",
        "version": 1
      },
      "created_at": "2024-01-15T10:30:00Z"
    }
  ],
  "count": 1,
  "total_count": 127,
  "has_more": true,
  "pagination": {
    "limit": 500,
    "offset": 0
  }
}
```

### Export Audit Events
Export audit data for compliance and reporting.

```http
GET /admin/audits/export?tenant={tenant_id}&format=csv&limit=10000
```

**Query Parameters:**
- `tenant` (optional): Filter by tenant ID
- `format`: Export format (`json`, `csv`)  
- `limit` (optional): Max records to export (default: 10000, max: 50000)

**Response Headers:**
```
Content-Type: application/json | text/csv
Content-Disposition: attachment; filename=audit_export_20240115_104500.json
X-PS-Export-Count: 1250
```

**Response (CSV format):**
```csv
timestamp,action,object_type,object_id,actor_email,tenant_id
2024-01-15T10:30:00Z,tenant.created,tenant,550e8400-e29b-41d4-a716-446655440000,admin@promptshield.com,550e8400-e29b-41d4-a716-446655440000
```

### Get Audit Events by Object
Get audit trail for a specific object (tenant, rulepack, etc.).

```http
GET /admin/audits/object/{object_type}/{object_id}?limit=100
```

**Response (200):**
```json
{
  "events": [
    {
      "id": "audit-550e8400-e29b-41d4-a716-446655440001",
      "action": "tenant.updated",
      "object_type": "tenant", 
      "object_id": "550e8400-e29b-41d4-a716-446655440000",
      "actor_email": "admin@promptshield.com",
      "metadata": {
        "changes": {
          "plan": {"from": "pro", "to": "enterprise"}
        }
      },
      "created_at": "2024-01-15T10:35:00Z"
    }
  ],
  "count": 1,
  "has_more": false
}
```

---

## 5. Platform Observability

Monitor platform performance and get real-time insights.

### Get Platform Metrics
Prometheus-format metrics for monitoring systems.

```http
GET /metrics
```

**Response (200):**
```
# HELP ps_enforcer_requests_total Total number of enforcement requests
# TYPE ps_enforcer_requests_total counter
ps_enforcer_requests_total{decision="allow"} 8500000
ps_enforcer_requests_total{decision="deny"} 150000
ps_enforcer_requests_total{decision="quarantine"} 25000

# HELP ps_enforcer_request_duration_seconds Request processing duration
# TYPE ps_enforcer_request_duration_seconds histogram
ps_enforcer_request_duration_seconds_bucket{le="0.001"} 7500000
ps_enforcer_request_duration_seconds_bucket{le="0.01"} 8600000
ps_enforcer_request_duration_seconds_bucket{le="0.1"} 8675000
ps_enforcer_request_duration_seconds_sum 125430.5
ps_enforcer_request_duration_seconds_count 8675000
```

### Get Platform Statistics  
High-level platform statistics for dashboards.

```http
GET /stats
```

**Response (200):**
```json
{
  "decisions_total": {
    "allow": 8500000,
    "deny": 150000, 
    "quarantine": 25000
  },
  "p95_latency_ms": 45.2
}
```

### Get Usage Metrics
Platform usage metrics across all tenants.

```http
GET /usage
```

**Response (200):**
```json
{
  "window_start": "2024-01-15T09:45:00Z",
  "window_end": "2024-01-15T10:45:00Z",
  "counts": 125430,
  "bytes": 5000000000,
  "tenants_active": 89,
  "top_tenants": [
    {
      "tenant_id": "550e8400-e29b-41d4-a716-446655440000",
      "tenant_name": "Acme Corporation", 
      "requests": 25430,
      "bytes": 750000000
    }
  ]
}
```

### Stream Platform Events
Real-time event stream for monitoring dashboards.

```http
GET /events?types=tenant,system
```

**Query Parameters:**
- `types` (optional): Comma-separated event types to filter

**Response (Server-Sent Events):**
```
Content-Type: text/event-stream
Cache-Control: no-cache
Connection: keep-alive

event: ready
data: {"status":"ok"}

event: tenant
data: {"type":"tenant.created","timestamp":"2024-01-15T10:45:00Z","data":{"tenant_id":"550e8400-e29b-41d4-a716-446655440000","name":"New Customer"}}

event: system  
data: {"type":"system.health","timestamp":"2024-01-15T10:46:00Z","data":{"status":"healthy","components_healthy":4}}
```

---

## 6. License Management

Manage your platform license and entitlements.

### Get License Information
View current platform license details.

```http
GET /license
```

**Response (200):**
```json
{
  "org": "PromptShield SaaS Platform",
  "tier": "enterprise",
  "expires_at": "2025-12-31T23:59:59Z",
  "licensed": true,
  "entitlements": {
    "max_tenants": 1000,
    "max_requests_per_day": 10000000,
    "semantic_analysis": true,
    "audit_retention_days": 365
  }
}
```

### Update License
Update your platform license key.

```http
POST /license
```

**Request Body:**
```json
{
  "key": "your-new-license-key-here"
}
```

**Response (204):** No content

---

## 7. General Platform Endpoints

Basic platform information and health checks.

### Platform Health
Simple health check for load balancers.

```http
GET /healthz
```

**Response (200):** `ok`

### Platform Readiness
Readiness check for Kubernetes deployments.

```http
GET /readyz  
```

**Response (200):** `ready`

### Platform Version
Get platform version information.

```http
GET /version
```

**Response (200):**
```json
{
  "version": "v0.2.0",
  "commit": "abc123def456", 
  "build_date": "2024-01-15T08:00:00Z"
}
```

---

## Error Responses

All endpoints use consistent error response format:

```json
{
  "error": {
    "code": "INVALID_ARGUMENT",
    "message": "Tenant name is required",
    "details": {
      "field": "name",
      "reason": "required_field_missing"
    },
    "request_id": "req-550e8400-e29b-41d4-a716-446655440000"
  }
}
```

**Common Error Codes:**
- `INVALID_ARGUMENT` (400): Bad request parameters
- `UNAUTHORIZED` (401): Authentication required
- `FORBIDDEN` (403): Insufficient permissions  
- `NOT_FOUND` (404): Resource not found
- `CONFLICT` (409): Resource already exists
- `RESOURCE_EXHAUSTED` (429): Rate limit exceeded
- `INTERNAL_ERROR` (500): Server error
- `SERVICE_UNAVAILABLE` (503): Service temporarily unavailable

---

## Rate Limits

Admin endpoints have higher rate limits:

- **Standard endpoints**: 1000 requests/hour per admin token
- **Export endpoints**: 10 requests/hour per admin token  
- **Real-time streams**: 5 concurrent connections per admin token

Rate limit headers are included in responses:
```
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 995
X-RateLimit-Reset: 1642248000
```

---

## SDK Examples

### JavaScript/Node.js
```javascript
class PromptShieldAdminAPI {
  constructor(baseURL, adminToken) {
    this.baseURL = baseURL;
    this.headers = {
      'Authorization': `Bearer ${adminToken}`,
      'Content-Type': 'application/json'
    };
  }

  async getAllTenants(options = {}) {
    const params = new URLSearchParams(options);
    const response = await fetch(`${this.baseURL}/admin/tenants?${params}`, {
      headers: this.headers
    });
    return response.json();
  }

  async createTenant(tenantData) {
    const response = await fetch(`${this.baseURL}/admin/tenants`, {
      method: 'POST',
      headers: this.headers,
      body: JSON.stringify(tenantData)
    });
    return response.json();
  }

  async getSystemHealth() {
    const response = await fetch(`${this.baseURL}/debug/status`, {
      headers: this.headers
    });
    return response.json();
  }

  streamEvents(eventTypes = []) {
    const params = eventTypes.length ? `?types=${eventTypes.join(',')}` : '';
    return new EventSource(`${this.baseURL}/events${params}`, {
      headers: this.headers
    });
  }
}
```

### Python
```python
import requests
from typing import Dict, List, Optional

class PromptShieldAdminAPI:
    def __init__(self, base_url: str, admin_token: str):
        self.base_url = base_url.rstrip('/')
        self.headers = {
            'Authorization': f'Bearer {admin_token}',
            'Content-Type': 'application/json'
        }

    def get_all_tenants(self, limit: int = 50, offset: int = 0, 
                       status: Optional[str] = None) -> Dict:
        params = {'limit': limit, 'offset': offset}
        if status:
            params['status'] = status
            
        response = requests.get(
            f'{self.base_url}/admin/tenants',
            headers=self.headers,
            params=params
        )
        response.raise_for_status()
        return response.json()

    def create_tenant(self, tenant_data: Dict) -> Dict:
        response = requests.post(
            f'{self.base_url}/admin/tenants',
            headers=self.headers,
            json=tenant_data
        )
        response.raise_for_status()
        return response.json()

    def get_system_health(self) -> Dict:
        response = requests.get(
            f'{self.base_url}/debug/status',
            headers=self.headers
        )
        response.raise_for_status()
        return response.json()
```

---

## Best Practices

### Security
- **Always use HTTPS** in production
- **Rotate admin tokens** regularly (every 90 days)
- **Limit admin access** to necessary personnel only
- **Monitor admin activities** through audit logs
- **Use rate limiting** to prevent abuse

### Performance  
- **Use pagination** for large result sets
- **Cache responses** where appropriate (system info, features)
- **Stream events** instead of polling for real-time updates
- **Export data** during off-peak hours
- **Monitor resource usage** via debug endpoints

### Monitoring
- **Set up alerts** on system health endpoints
- **Monitor error rates** and response times  
- **Track tenant growth** and usage patterns
- **Audit admin activities** regularly
- **Export compliance data** monthly

---

This comprehensive API reference covers all administrative endpoints available for managing your PromptShield SaaS platform.