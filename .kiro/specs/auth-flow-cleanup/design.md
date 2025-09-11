# Design Document

## Overview

This design addresses the authentication flow issues in the multi-tenant PromptShield system by standardizing the JWT-based communication between the frontend BFF and the Go gateway backend, improving error handling, and ensuring consistent tenant context flow.

## Architecture

### Current State Analysis

The system currently has these components:
- **Frontend BFF (Express/TypeScript)**: Handles Clerk authentication, generates JWTs for backend communication
- **Go Gateway Backend**: Validates JWTs, enforces tenant isolation, handles business logic
- **Authentication Flow**: Clerk → BFF → JWT → Gateway

### Issues Identified

1. **JWT Configuration Mismatch**: Private key formatting and environment variable inconsistencies
2. **Middleware Order**: Tenant validation running before proper JWT validation
3. **Error Propagation**: Generic 401 errors without specific failure reasons
4. **Dev Mode Inconsistency**: Bypass flags not working uniformly across components

## Components and Interfaces

### 1. JWT Authentication Service (Frontend BFF)

**Purpose**: Generate and manage JWT tokens for backend communication

**Key Functions**:
- `generateGatewayJWT(userContext)`: Create RS256 JWT with user and tenant claims
- `extractUserContext(req)`: Extract user info from Clerk session and cookies
- `validateJWTConfig()`: Startup validation of JWT configuration

**Configuration**:
```typescript
interface JWTConfig {
  privateKey: string;     // PS_BFF_JWT_PRIVATE_KEY (PEM format)
  issuer: string;         // PS_BFF_JWT_ISSUER
  audience: string;       // PS_BFF_JWT_AUDIENCE  
  ttl: number;           // Token TTL in seconds (default: 120)
}
```

**Improvements**:
- Add private key format validation and auto-correction
- Implement startup configuration validation
- Add detailed error logging for JWT generation failures

### 2. JWT Validation Middleware (Go Backend)

**Purpose**: Validate incoming JWT tokens and extract user context

**Key Functions**:
- `jwtAuthMiddleware()`: Validate RS256 JWT signatures and claims
- `parseRSAPublicKeyFromPEM()`: Parse public key with better error handling
- `extractClaimsFromJWT()`: Extract and validate JWT claims

**Configuration**:
```go
type JWTConfig struct {
    PublicKey    *rsa.PublicKey
    Issuer       string
    Audience     string
    ClockSkew    time.Duration
}
```

**Improvements**:
- Add structured error responses with specific failure reasons
- Improve public key parsing with better error messages
- Add request correlation IDs for debugging

### 3. Tenant Context Middleware (Go Backend)

**Purpose**: Validate tenant access and set database context for RLS

**Key Functions**:
- `tenantValidationMiddleware()`: Validate tenant exists and user has access
- `setTenantContextInDB()`: Set RLS context for database queries
- `validateTenantMembership()`: Check user membership in tenant

**Improvements**:
- Prioritize tenant ID from JWT claims over headers
- Add specific error codes for different failure scenarios
- Improve membership validation with better error messages

### 4. Error Handling System

**Purpose**: Provide consistent, debuggable error responses

**Error Response Format**:
```json
{
  "error": {
    "code": "UNAUTHORIZED",
    "message": "JWT signature verification failed",
    "details": {
      "reason": "invalid_signature",
      "request_id": "req_123",
      "timestamp": "2025-01-09T10:57:19Z"
    }
  }
}
```

**Error Codes**:
- `JWT_INVALID`: Token format or signature issues
- `JWT_EXPIRED`: Token has expired
- `JWT_MISSING`: No Authorization header provided
- `TENANT_NOT_FOUND`: Tenant doesn't exist
- `TENANT_INACTIVE`: Tenant is not active
- `TENANT_ACCESS_DENIED`: User not a member of tenant

## Data Models

### JWT Payload Structure

```typescript
interface JWTPayload {
  sub: string;           // User ID from Clerk
  name?: string;         // User display name
  email?: string;        // User email
  tenant_id?: string;    // Active tenant UUID
  roles?: string[];      // User roles in tenant
  admin?: boolean;       // Admin flag
  iss: string;          // Issuer (BFF)
  aud: string;          // Audience (Gateway)
  exp: number;          // Expiration timestamp
  iat: number;          // Issued at timestamp
}
```

### User Context Structure

```go
type UserContext struct {
    UserID   string   `json:"user_id"`
    Name     string   `json:"name,omitempty"`
    Email    string   `json:"email,omitempty"`
    TenantID string   `json:"tenant_id,omitempty"`
    Roles    []string `json:"roles,omitempty"`
    IsAdmin  bool     `json:"is_admin"`
}
```

## Error Handling

### JWT Validation Errors

1. **Missing Token**: Return 401 with `JWT_MISSING` code
2. **Invalid Format**: Return 401 with `JWT_INVALID` code and format details
3. **Signature Failure**: Return 401 with `JWT_INVALID` code and signature error
4. **Expired Token**: Return 401 with `JWT_EXPIRED` code and expiration time
5. **Invalid Claims**: Return 401 with `JWT_INVALID` code and claim validation details

### Tenant Validation Errors

1. **Missing Tenant**: Return 400 with `TENANT_REQUIRED` code
2. **Invalid Tenant ID**: Return 400 with `TENANT_INVALID_FORMAT` code
3. **Tenant Not Found**: Return 404 with `TENANT_NOT_FOUND` code
4. **Tenant Inactive**: Return 403 with `TENANT_INACTIVE` code
5. **Access Denied**: Return 403 with `TENANT_ACCESS_DENIED` code

### Development Mode Handling

When `PS_DEV_BYPASS_AUTH=true`:
- JWT validation is skipped
- Default dev user context is injected
- Tenant validation uses dev tenant ID
- All errors include additional debugging information

## Testing Strategy

### Unit Tests

1. **JWT Generation Tests**:
   - Valid token generation with all claims
   - Private key format validation
   - Configuration validation
   - Error handling for missing/invalid keys

2. **JWT Validation Tests**:
   - Valid token validation
   - Expired token handling
   - Invalid signature detection
   - Malformed token handling

3. **Tenant Validation Tests**:
   - Valid tenant access
   - Invalid tenant ID handling
   - Membership validation
   - RLS context setting

### Integration Tests

1. **End-to-End Auth Flow**:
   - Clerk login → JWT generation → Backend validation
   - Tenant selection → Cookie setting → Backend access
   - Error propagation through the full stack

2. **Dev Mode Tests**:
   - Bypass functionality across all components
   - Dev user injection
   - Dev tenant handling

### Error Scenario Tests

1. **Configuration Errors**:
   - Missing JWT keys
   - Invalid key formats
   - Mismatched issuer/audience

2. **Runtime Errors**:
   - Network failures
   - Database connectivity issues
   - Invalid tenant states

## Implementation Plan

### Phase 1: JWT System Fixes
- Fix private key parsing and validation
- Improve error messages and logging
- Add configuration validation

### Phase 2: Middleware Chain Cleanup
- Standardize middleware order
- Improve error propagation
- Add request correlation IDs

### Phase 3: Tenant Context Improvements
- Prioritize JWT claims for tenant ID
- Add specific error codes
- Improve membership validation

### Phase 4: Development Experience
- Enhance dev mode consistency
- Add debugging endpoints
- Improve error documentation