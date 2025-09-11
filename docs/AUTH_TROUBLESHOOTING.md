# Authentication Troubleshooting Guide

## Quick Diagnosis

### Step 1: Check Debug Endpoints

Enable debug endpoints and check configuration:

```bash
# Enable debug endpoints
export PS_ENABLE_DEBUG_ENDPOINTS=true

# Check frontend JWT configuration
curl http://localhost:8096/api/debug/jwt-config

# Check backend JWT configuration  
curl http://localhost:8098/debug/jwt-config

# Check authentication status
curl -H "Authorization: Bearer <your-token>" http://localhost:8098/debug/auth
```

### Step 2: Verify Environment Variables

**Frontend BFF Required Variables:**
```bash
echo "CLERK_SECRET_KEY: ${CLERK_SECRET_KEY:0:10}..."
echo "PS_BFF_JWT_PRIVATE_KEY: ${PS_BFF_JWT_PRIVATE_KEY:0:50}..."
echo "PS_BFF_JWT_ISSUER: $PS_BFF_JWT_ISSUER"
echo "PS_BFF_JWT_AUDIENCE: $PS_BFF_JWT_AUDIENCE"
```

**Go Gateway Required Variables:**
```bash
echo "PS_BFF_JWT_PUBLIC_KEY: ${PS_BFF_JWT_PUBLIC_KEY:0:50}..."
echo "PS_BFF_JWT_ISSUER: $PS_BFF_JWT_ISSUER"
echo "PS_BFF_JWT_AUDIENCE: $PS_BFF_JWT_AUDIENCE"
```

### Step 3: Test with Development Bypass

Temporarily enable bypass mode to isolate the issue:

```bash
export PS_DEV_BYPASS_AUTH=true
export PS_DEV_USER_ID=test-user
export PS_DEV_TENANT_ID=test-tenant

# Test the endpoint
curl http://localhost:8096/api/auth/user
```

## Common Error Scenarios

### 1. "JWT_MISSING" Error

**Symptoms:**
```json
{
  "error": {
    "code": "JWT_MISSING",
    "message": "Authorization header is required"
  }
}
```

**Causes & Solutions:**

1. **Missing Authorization Header**
   ```bash
   # Wrong
   curl http://localhost:8098/api/users
   
   # Correct
   curl -H "Authorization: Bearer <token>" http://localhost:8098/api/users
   ```

2. **Frontend Not Generating Tokens**
   - Check if user is authenticated in frontend
   - Verify JWT configuration in BFF
   - Check browser network tab for token generation

3. **Proxy Configuration Issues**
   - Ensure Authorization header is forwarded
   - Check reverse proxy configuration

### 2. "JWT_SIGNATURE_INVALID" Error

**Symptoms:**
```json
{
  "error": {
    "code": "JWT_SIGNATURE_INVALID",
    "message": "JWT signature verification failed"
  }
}
```

**Causes & Solutions:**

1. **Mismatched Key Pair**
   ```bash
   # Generate matching key pair
   openssl genrsa -out private.pem 2048
   openssl rsa -in private.pem -pubout -out public.pem
   
   # Set environment variables
   export PS_BFF_JWT_PRIVATE_KEY="$(cat private.pem)"
   export PS_BFF_JWT_PUBLIC_KEY="$(cat public.pem)"
   ```

2. **Key Format Issues**
   ```bash
   # Ensure proper PEM format with newlines
   echo "$PS_BFF_JWT_PRIVATE_KEY" | head -1
   # Should output: -----BEGIN PRIVATE KEY-----
   ```

3. **Environment Variable Corruption**
   - Check for extra quotes or escape characters
   - Verify newlines are preserved in multi-line keys

### 3. "JWT_EXPIRED" Error

**Symptoms:**
```json
{
  "error": {
    "code": "JWT_EXPIRED",
    "message": "JWT token has expired",
    "details": {
      "expired_at": "2025-01-09T10:55:19Z",
      "current_time": "2025-01-09T10:57:19Z"
    }
  }
}
```

**Causes & Solutions:**

1. **Clock Skew**
   ```bash
   # Increase leeway (default 60 seconds)
   export PS_BFF_JWT_LEEWAY=120
   ```

2. **Token Caching Issues**
   - Frontend may be caching expired tokens
   - Clear browser cache/localStorage
   - Implement token refresh logic

3. **Long Request Processing**
   - Increase token TTL for development
   ```bash
   export PS_BFF_JWT_TTL=300  # 5 minutes
   ```

### 4. "TENANT_NOT_FOUND" Error

**Symptoms:**
```json
{
  "error": {
    "code": "TENANT_NOT_FOUND",
    "message": "Tenant does not exist",
    "details": {
      "tenant_id": "123e4567-e89b-12d3-a456-426614174000"
    }
  }
}
```

**Causes & Solutions:**

1. **Tenant Doesn't Exist in Database**
   ```sql
   -- Check if tenant exists
   SELECT id, name, created_at FROM tenants WHERE id = '123e4567-e89b-12d3-a456-426614174000';
   
   -- Create tenant if needed
   INSERT INTO tenants (id, name, created_at, updated_at) 
   VALUES ('123e4567-e89b-12d3-a456-426614174000', 'Test Tenant', NOW(), NOW());
   ```

2. **Tenant ID Format Issues**
   ```bash
   # Ensure valid UUID format
   echo "123e4567-e89b-12d3-a456-426614174000" | grep -E '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'
   ```

3. **Soft-Deleted Tenant**
   ```sql
   -- Check for soft-deleted tenants
   SELECT id, name, deleted_at FROM tenants WHERE id = '123e4567-e89b-12d3-a456-426614174000';
   
   -- Restore if needed
   UPDATE tenants SET deleted_at = NULL WHERE id = '123e4567-e89b-12d3-a456-426614174000';
   ```

### 5. "TENANT_ACCESS_DENIED" Error

**Symptoms:**
```json
{
  "error": {
    "code": "TENANT_ACCESS_DENIED",
    "message": "User is not a member of this tenant",
    "details": {
      "tenant_id": "123e4567-e89b-12d3-a456-426614174000",
      "user_id": "user_123"
    }
  }
}
```

**Causes & Solutions:**

1. **Missing Tenant Membership**
   ```sql
   -- Check membership
   SELECT * FROM tenant_memberships 
   WHERE tenant_id = '123e4567-e89b-12d3-a456-426614174000' 
   AND user_id = 'user_123';
   
   -- Add membership
   INSERT INTO tenant_memberships (tenant_id, user_id, role, created_at, updated_at)
   VALUES ('123e4567-e89b-12d3-a456-426614174000', 'user_123', 'member', NOW(), NOW());
   ```

2. **Soft-Deleted Membership**
   ```sql
   -- Check for soft-deleted memberships
   SELECT * FROM tenant_memberships 
   WHERE tenant_id = '123e4567-e89b-12d3-a456-426614174000' 
   AND user_id = 'user_123' 
   AND deleted_at IS NOT NULL;
   
   -- Restore membership
   UPDATE tenant_memberships SET deleted_at = NULL 
   WHERE tenant_id = '123e4567-e89b-12d3-a456-426614174000' 
   AND user_id = 'user_123';
   ```

### 6. Configuration Errors

**Symptoms:**
```json
{
  "error": {
    "code": "JWT_CONFIGURATION_ERROR",
    "message": "JWT authentication configuration error"
  }
}
```

**Diagnosis Steps:**

1. **Check Key Format**
   ```bash
   # Validate private key
   echo "$PS_BFF_JWT_PRIVATE_KEY" | openssl rsa -check -noout
   
   # Validate public key
   echo "$PS_BFF_JWT_PUBLIC_KEY" | openssl rsa -pubin -text -noout
   ```

2. **Verify Key Pair Match**
   ```bash
   # Extract public key from private key
   echo "$PS_BFF_JWT_PRIVATE_KEY" | openssl rsa -pubout > derived_public.pem
   
   # Compare with configured public key
   diff <(echo "$PS_BFF_JWT_PUBLIC_KEY") derived_public.pem
   ```

3. **Test JWT Generation**
   ```bash
   # Use debug endpoint to test
   curl http://localhost:8096/api/debug/jwt-config
   ```

## Development Workflow

### 1. Local Development Setup

```bash
# 1. Enable development mode
export PS_DEV_BYPASS_AUTH=true
export PS_DEV_USER_ID=dev-user
export PS_DEV_USER_NAME="Dev User"
export PS_DEV_USER_EMAIL=dev@example.com
export PS_DEV_TENANT_ID=dev-tenant

# 2. Generate development keys
openssl genrsa -out dev-private.pem 2048
openssl rsa -in dev-private.pem -pubout -out dev-public.pem

export PS_BFF_JWT_PRIVATE_KEY="$(cat dev-private.pem)"
export PS_BFF_JWT_PUBLIC_KEY="$(cat dev-public.pem)"
export PS_BFF_JWT_ISSUER=promptshield-dev
export PS_BFF_JWT_AUDIENCE=promptshield-gateway-dev

# 3. Enable debug endpoints
export PS_ENABLE_DEBUG_ENDPOINTS=true

# 4. Start services
npm run dev  # Frontend BFF
go run main.go  # Go Gateway
```

### 2. Testing Authentication Flow

```bash
# 1. Test dev bypass
curl http://localhost:8096/api/auth/user

# 2. Test JWT generation
curl http://localhost:8096/api/debug/auth

# 3. Test backend validation
curl -H "Authorization: Bearer $(curl -s http://localhost:8096/api/debug/auth | jq -r '.jwt.testToken')" \
     http://localhost:8098/debug/auth

# 4. Test tenant context
curl -H "Authorization: Bearer <token>" \
     -H "X-PS-Tenant-ID: dev-tenant" \
     http://localhost:8098/debug/tenant-context
```

### 3. Production Deployment Checklist

- [ ] `PS_DEV_BYPASS_AUTH=false`
- [ ] Strong `SESSION_SECRET` set
- [ ] Valid Clerk keys configured
- [ ] JWT key pair generated and configured
- [ ] `PS_ENABLE_DEBUG_ENDPOINTS=false`
- [ ] HTTPS enabled (`SESSION_COOKIE_SECURE=true`)
- [ ] CORS origins configured
- [ ] Database RLS policies enabled
- [ ] Monitoring and logging configured

## Monitoring and Alerting

### Key Metrics to Monitor

1. **Authentication Success Rate**
   ```
   auth_success_rate = successful_auths / total_auth_attempts
   ```

2. **JWT Validation Errors**
   ```
   jwt_error_rate = jwt_validation_errors / total_jwt_validations
   ```

3. **Tenant Access Denials**
   ```
   tenant_denial_rate = tenant_access_denied / total_tenant_requests
   ```

### Log Analysis

**Find Authentication Failures:**
```bash
grep "Authentication failed" /var/log/app.log | jq '.correlation_id'
```

**Find JWT Errors:**
```bash
grep "JWT validation failed" /var/log/app.log | jq '.error_code' | sort | uniq -c
```

**Find Tenant Issues:**
```bash
grep "Tenant validation failed" /var/log/app.log | jq '.details.tenant_id' | sort | uniq -c
```

### Health Checks

**Frontend BFF Health:**
```bash
curl http://localhost:8096/healthz
curl http://localhost:8096/api/debug/jwt-config
```

**Go Gateway Health:**
```bash
curl http://localhost:8098/healthz
curl http://localhost:8098/debug/jwt-config
```

## Emergency Procedures

### 1. Authentication System Down

**Immediate Actions:**
1. Enable dev bypass temporarily (if safe)
2. Check service logs for errors
3. Verify database connectivity
4. Check JWT key configuration

**Recovery Steps:**
1. Identify root cause from logs
2. Fix configuration issues
3. Restart affected services
4. Disable dev bypass
5. Monitor authentication success rate

### 2. JWT Key Compromise

**Immediate Actions:**
1. Generate new key pair
2. Update environment variables
3. Restart all services
4. Invalidate existing sessions

**Recovery Steps:**
1. Rotate keys in all environments
2. Update deployment configurations
3. Monitor for authentication errors
4. Communicate to users if needed

### 3. Tenant Isolation Breach

**Immediate Actions:**
1. Check RLS policies are enabled
2. Verify tenant context setting
3. Audit recent tenant access logs
4. Disable affected tenants if needed

**Recovery Steps:**
1. Fix RLS policy issues
2. Audit data access patterns
3. Restore proper tenant isolation
4. Re-enable affected tenants
5. Implement additional monitoring