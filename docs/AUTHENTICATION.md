# Authentication System Documentation

## Overview

The PromptShield authentication system uses a multi-tenant architecture with Clerk for user authentication and JWT tokens for secure communication between the frontend BFF (Backend for Frontend) and the Go gateway backend.

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Frontend      │    │   BFF Server    │    │  Go Gateway     │
│   (React/Vite)  │    │  (Express/TS)   │    │   (Go/Chi)      │
├─────────────────┤    ├─────────────────┤    ├─────────────────┤
│ • Clerk Auth    │───▶│ • Clerk Session │───▶│ • JWT Validation│
│ • User Session  │    │ • JWT Generation│    │ • Tenant Context│
│ • Tenant Select │    │ • Tenant Cookies│    │ • RLS Policies  │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## Authentication Flow

### 1. User Authentication
1. User signs in through Clerk on the frontend
2. Clerk provides session tokens and user information
3. Frontend BFF validates Clerk session and extracts user context
4. BFF generates short-lived JWT tokens for backend communication

### 2. Tenant Selection
1. User selects an organization/tenant from their available tenants
2. Frontend calls `/api/orgs/select` with the chosen tenant ID
3. BFF validates user membership in the tenant
4. Tenant ID is stored in signed cookies for subsequent requests

### 3. Backend Communication
1. BFF generates JWT tokens containing user and tenant context
2. JWT tokens are sent to the Go gateway with each API request
3. Gateway validates JWT signature and extracts trusted user/tenant headers
4. Database RLS (Row Level Security) policies enforce tenant isolation

## Environment Configuration

### Frontend BFF Configuration

```bash
# Clerk Authentication
CLERK_SECRET_KEY=sk_test_...                    # Required for production
VITE_CLERK_PUBLISHABLE_KEY=pk_test_...         # Required for frontend

# JWT Configuration
PS_BFF_JWT_PRIVATE_KEY="-----BEGIN PRIVATE KEY-----..." # Required
PS_BFF_JWT_ISSUER=promptshield-bff-prod        # JWT issuer
PS_BFF_JWT_AUDIENCE=promptshield-gateway-prod  # JWT audience
PS_BFF_JWT_TTL=120                             # Token TTL in seconds

# Session Configuration
SESSION_SECRET=your-secure-session-secret       # Required for production
SESSION_COOKIE_SECURE=true                     # Use secure cookies
SESSION_COOKIE_SAMESITE=lax                    # Cookie SameSite policy

# Gateway Configuration
PS_GATEWAY_URL=https://api.promptshield.com    # Backend gateway URL
PS_ADMIN_TOKEN=your-admin-token                # Optional admin token

# Development Configuration
PS_DEV_BYPASS_AUTH=false                       # Enable dev bypass (dev only)
PS_DEV_USER_ID=dev-user                        # Dev user ID
PS_DEV_USER_NAME=Dev User                      # Dev user name
PS_DEV_USER_EMAIL=dev@example.com              # Dev user email
PS_DEV_TENANT_ID=dev-tenant-123                # Dev tenant ID
PS_DEV_ROLES=admin,user                        # Dev user roles
PS_DEV_IS_ADMIN=true                           # Dev admin flag

# Feature Flags
PS_ALLOW_SELF_TENANT_SIGNUP=false              # Allow self-service signup
PS_ENABLE_DEBUG_ENDPOINTS=false                # Enable debug endpoints

# Rate Limiting
PS_INVITE_RATE_LIMIT=5                         # Invites per window
PS_INVITE_WINDOW_MS=600000                     # Rate limit window (10 min)

# Security
PS_CAPTCHA_TOKEN=your-captcha-secret           # CAPTCHA verification
PS_CORS_ALLOWED_ORIGINS=https://app.example.com # CORS origins
```

### Go Gateway Configuration

```bash
# JWT Validation
PS_BFF_JWT_PUBLIC_KEY="-----BEGIN PUBLIC KEY-----..." # Required
PS_BFF_JWT_ISSUER=promptshield-bff-prod        # Must match BFF issuer
PS_BFF_JWT_AUDIENCE=promptshield-gateway-prod  # Must match BFF audience
PS_BFF_JWT_LEEWAY=60                           # Clock skew allowance (seconds)

# Development Configuration
PS_DEV_BYPASS_AUTH=false                       # Enable dev bypass (dev only)
PS_DEV_USER_ID=dev-user                        # Dev user ID
PS_DEV_USER_NAME=Dev User                      # Dev user name
PS_DEV_TENANT_ID=dev-tenant-123                # Dev tenant ID
PS_DEV_ROLES=admin,user                        # Dev user roles
PS_DEV_IS_ADMIN=true                           # Dev admin flag

# Admin Configuration
PS_ADMIN_TOKEN=your-admin-token                # Admin API token
PS_ALLOW_INSECURE_ADMIN=false                  # Allow insecure admin (dev only)

# Feature Flags
PS_ENABLE_DEBUG_ENDPOINTS=false                # Enable debug endpoints

# Rate Limiting
PS_ENFORCER_RPS=100                            # Requests per second
PS_ENFORCER_RPS_BURST=200                      # Burst capacity

# NATS Configuration
PS_NATS_URL=nats://localhost:4222              # NATS server URL
```

## Error Codes and Troubleshooting

### JWT Errors

| Error Code | Description | Solution |
|------------|-------------|----------|
| `JWT_MISSING` | No Authorization header provided | Include `Authorization: Bearer <token>` header |
| `JWT_INVALID` | Token format is invalid | Check token generation and encoding |
| `JWT_EXPIRED` | Token has expired | Generate a new token (tokens are short-lived) |
| `JWT_SIGNATURE_INVALID` | Token signature verification failed | Verify private/public key pair match |
| `JWT_INVALID_ISSUER` | Token issuer doesn't match expected | Check `PS_BFF_JWT_ISSUER` configuration |
| `JWT_INVALID_AUDIENCE` | Token audience doesn't match expected | Check `PS_BFF_JWT_AUDIENCE` configuration |
| `JWT_CONFIGURATION_ERROR` | JWT configuration is invalid | Verify JWT environment variables |

### Tenant Errors

| Error Code | Description | Solution |
|------------|-------------|----------|
| `TENANT_MISSING` | No tenant ID provided | Include tenant ID in JWT or headers |
| `TENANT_INVALID_FORMAT` | Tenant ID is not a valid UUID | Provide a valid UUID format |
| `TENANT_NOT_FOUND` | Tenant doesn't exist | Verify tenant exists in database |
| `TENANT_INACTIVE` | Tenant is not active | Contact admin to activate tenant |
| `TENANT_ACCESS_DENIED` | User is not a member of tenant | Add user to tenant or select different tenant |
| `TENANT_CONTEXT_ERROR` | Failed to set RLS context | Check database connectivity and RLS setup |

### Authentication Errors

| Error Code | Description | Solution |
|------------|-------------|----------|
| `UNAUTHORIZED` | Authentication required | Sign in through Clerk |
| `FORBIDDEN` | Access denied | Check user permissions and tenant membership |

## Development Mode

For development and testing, you can enable bypass mode:

```bash
# Enable development bypass
PS_DEV_BYPASS_AUTH=true
PS_DEV_USER_ID=dev-user-123
PS_DEV_USER_NAME=Dev User
PS_DEV_USER_EMAIL=dev@example.com
PS_DEV_TENANT_ID=dev-tenant-123
PS_DEV_IS_ADMIN=true
```

**⚠️ Warning**: Never enable bypass mode in production!

## Debug Endpoints

When `PS_ENABLE_DEBUG_ENDPOINTS=true` (or in development), the following debug endpoints are available:

### Frontend BFF Debug Endpoints

- `GET /api/debug/auth` - Authentication status and user context
- `GET /api/debug/jwt-config` - JWT configuration validation

### Go Gateway Debug Endpoints

- `GET /debug/auth` - Authentication headers and context
- `GET /debug/jwt-config` - JWT validation configuration
- `GET /debug/tenant-context` - Tenant context and validation
- `GET /debug/headers` - All request headers

## Security Considerations

### JWT Security
- Tokens are short-lived (default 2 minutes) to minimize exposure
- RS256 algorithm provides strong signature verification
- Private keys should be securely stored and rotated regularly
- Public keys are validated on startup

### Tenant Isolation
- Row Level Security (RLS) policies enforce tenant data isolation
- Tenant context is set for every database query
- User membership is validated for each tenant access
- Admin users can bypass membership checks for system operations

### Session Security
- Signed cookies prevent tampering
- Secure cookies in production (HTTPS only)
- SameSite policy prevents CSRF attacks
- Session secrets should be cryptographically secure

## Monitoring and Logging

### Structured Logging
All authentication events are logged with structured data:

```json
{
  "timestamp": "2025-01-09T10:57:19Z",
  "level": "INFO",
  "message": "JWT validation successful",
  "component": "auth",
  "correlation_id": "req_1704794239_abc123",
  "user_id": "user_123",
  "tenant_id": "tenant_456",
  "path": "/api/users",
  "method": "GET"
}
```

### Correlation IDs
Every request gets a correlation ID for tracing across services:
- Generated automatically if not provided
- Included in all log entries and error responses
- Propagated through the entire request chain

### Metrics
Key authentication metrics to monitor:
- JWT validation success/failure rates
- Tenant access patterns
- Authentication latency
- Error rates by type

## Testing

### Unit Tests
- JWT generation and validation
- User context extraction
- Error handling scenarios
- Configuration validation

### Integration Tests
- End-to-end authentication flow
- Tenant selection and validation
- Error response formats
- Debug endpoint functionality

### Load Testing
- JWT validation performance
- Tenant isolation under load
- Rate limiting effectiveness
- Database RLS performance

## Migration Guide

### From Legacy Auth
1. Update environment variables to new naming convention
2. Replace custom JWT middleware with new structured middleware
3. Update error handling to use new error codes
4. Test tenant isolation with new RLS policies

### Key Changes
- Standardized error codes and responses
- Enhanced logging with correlation IDs
- Improved JWT validation with detailed errors
- Centralized environment configuration
- Debug endpoints for troubleshooting

## Support

### Common Issues

**Issue**: 401 errors on `/api/auth/user`
**Solution**: Check JWT configuration and ensure private/public key pair matches

**Issue**: Tenant access denied errors
**Solution**: Verify user membership in tenant and tenant status

**Issue**: JWT signature verification failed
**Solution**: Ensure `PS_BFF_JWT_PRIVATE_KEY` and `PS_BFF_JWT_PUBLIC_KEY` are a matching pair

### Getting Help
1. Check the debug endpoints for configuration issues
2. Review logs for correlation IDs and error details
3. Verify environment variable configuration
4. Test with development bypass mode to isolate issues