# PromptShield Enterprise API Design

## Architecture Overview

This enterprise API design follows a layered architecture with proper separation of concerns:

```
┌─────────────────────────────────────────────────┐
│                 HTTP Layer                      │
│  ┌─────────────┬─────────────┬─────────────┐   │
│  │ User APIs   │ Admin APIs  │ Debug APIs  │   │
│  │ /v1/*       │ /admin/*    │ /debug/*    │   │
│  └─────────────┴─────────────┴─────────────┘   │
├─────────────────────────────────────────────────┤
│                Middleware Layer                 │
│  Auth • OIDC • Quota • Audit • Metrics         │
├─────────────────────────────────────────────────┤
│               Service Layer                     │
│  Tenant • Assignment • Audit • Usage • Rules   │
├─────────────────────────────────────────────────┤
│              Repository Layer                   │
│  PostgreSQL • Redis • Event Store              │
└─────────────────────────────────────────────────┘
```

## API Specification

### Authentication & Authorization

#### Three Security Tiers:
1. **Public** - Health checks, version info
2. **User** - Enforcement APIs (token + optional OIDC)
3. **Admin** - Management APIs (admin token + tenant isolation)

#### Security Headers:
- `Authorization: Bearer <token>` - User/Admin tokens
- `X-PS-Tenant-ID: <uuid>` - Tenant context (validated against token claims)
- `X-PS-Correlation-ID: <uuid>` - Request tracing

### API Endpoint Structure

#### Core Enforcement APIs (/v1)
```
POST   /v1/check           # Real-time policy enforcement
POST   /v1/scan            # Batch content scanning
POST   /v1/scan/async      # Asynchronous scanning
GET    /v1/jobs            # List async jobs
GET    /v1/jobs/{id}       # Get job status
DELETE /v1/jobs/{id}       # Cancel job
```

#### Tenant Management (/v1/admin/tenants)
```
POST   /v1/admin/tenants                  # Create tenant
GET    /v1/admin/tenants                  # List tenants
GET    /v1/admin/tenants/{id}             # Get tenant
PUT    /v1/admin/tenants/{id}             # Update tenant
DELETE /v1/admin/tenants/{id}             # Delete tenant
```

#### Rule-pack Assignments (/v1/admin/tenants/{id}/assignments)
```
POST   /v1/admin/tenants/{id}/assignments         # Create assignment
GET    /v1/admin/tenants/{id}/assignments         # List assignments
GET    /v1/admin/tenants/{id}/assignments?scope=x # Filter by scope
PUT    /v1/admin/assignments/{assignmentId}       # Update assignment
DELETE /v1/admin/assignments/{assignmentId}       # Delete assignment
```

#### Audit Trail (/v1/admin/audits)
```
GET    /v1/admin/audits                           # List audit events
GET    /v1/admin/audits?tenant={id}&limit=n       # Filter by tenant
GET    /v1/admin/audits/object/{type}/{objectId}  # Events for object
```

#### Usage Metering (/v1/admin/usage)
```
GET    /v1/admin/usage/minute                     # Minute-level metrics
GET    /v1/admin/usage/hour                       # Hour-level metrics  
GET    /v1/admin/usage/day                        # Day-level metrics
# Query params: tenant, start, end, group=tenant,route,decision
```

#### Quota Management (/v1/admin/tenants/{id}/quota)
```
GET    /v1/admin/tenants/{id}/quota               # Get quota limits
PUT    /v1/admin/tenants/{id}/quota               # Update quota limits
```

#### System Management (/v1/admin/system)
```
GET    /v1/admin/system/features                  # Feature flags
GET    /v1/admin/system/stats                     # Performance stats
POST   /v1/admin/system/drain                     # Graceful drain
POST   /v1/admin/system/shutdown                  # Graceful shutdown
```

#### System Diagnostics (/debug)
```
GET    /debug/status                              # System health
GET    /debug/goroutines                          # Goroutine diagnostics
GET    /debug/memory                              # Memory statistics
GET    /debug/pprof/*                            # Go profiling
```

### Response Formats

#### Success Response:
```json
{
  "data": { ... },
  "meta": {
    "correlation_id": "uuid",
    "timestamp": "2024-01-01T00:00:00Z",
    "version": "1"
  }
}
```

#### Error Response:
```json
{
  "error": {
    "code": "INVALID_TENANT",
    "message": "Tenant not found or access denied",
    "details": {
      "tenant_id": "uuid",
      "allowed_tenants": ["uuid1", "uuid2"]
    }
  },
  "meta": {
    "correlation_id": "uuid",
    "timestamp": "2024-01-01T00:00:00Z", 
    "version": "1"
  }
}
```

### Enterprise Features

#### Tenant Isolation
- All admin APIs require tenant context
- Cross-tenant access prevention
- Tenant-scoped rate limiting

#### Audit Logging
- All mutations logged with actor information
- Immutable audit trail with hash chaining
- Compliance-ready event structure

#### Rate Limiting
- Per-tenant quota enforcement
- Configurable RPS and burst limits
- Graceful degradation with 429 responses

#### Observability
- Structured logging with correlation IDs
- Prometheus metrics for all endpoints
- OpenTelemetry tracing integration
- Real-time event streaming via SSE

#### High Availability
- Circuit breakers for external dependencies
- Graceful shutdown with connection draining
- Health checks for readiness and liveness
- Partial degradation support

## Implementation Strategy

1. **Extend Options struct** with missing repositories and services
2. **Create service layer** for each domain (tenant, assignment, audit, usage)
3. **Implement middleware** for auth, quota, audit, and metrics
4. **Add comprehensive error handling** with proper HTTP status codes
5. **Add API versioning** for backward compatibility
6. **Implement rate limiting** per tenant with Redis backend
7. **Add comprehensive testing** with integration tests
8. **Add OpenAPI specification** for documentation

## Security Considerations

- Input validation and sanitization on all endpoints
- SQL injection prevention via parameterized queries
- Rate limiting to prevent DoS attacks
- Audit logging for compliance and security monitoring
- Tenant isolation to prevent data leakage
- Secure credential storage for API keys
- TLS/mTLS for inter-service communication