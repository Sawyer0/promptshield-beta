# Tenant Isolation & Row Level Security (RLS) Guide

Complete guide for understanding and using PromptShield's multi-tenant database isolation.

## 🔒 Overview

PromptShield implements **Row Level Security (RLS)** to ensure complete data isolation between tenants. Every database query automatically filters data based on the current tenant context, preventing any cross-tenant data access.

### Security Guarantees

✅ **Complete Data Isolation**: Each tenant can only access their own data  
✅ **Zero Cross-Tenant Leaks**: Impossible to access another tenant's data  
✅ **Automatic Filtering**: All queries are automatically tenant-scoped  
✅ **Platform Admin Override**: Admins can access all tenant data when needed  
✅ **Audit Trail Isolation**: Each tenant's audit logs are completely separate  

## 📋 Database Tables with RLS

### Tenant-Isolated Tables (RLS Enabled)
These tables automatically filter by tenant_id:

- `active_rulepacks` - Active security policies per tenant
- `assignments` - Policy assignments per tenant  
- `audits` - Audit logs per tenant
- `rulepacks` - Custom rulepacks per tenant
- `rulepack_versions` - Version history per tenant
- `violations` - Security violations per tenant
- `sessions` - User sessions per tenant
- `usage_metrics` - Usage statistics per tenant
- `tenant_settings` - Tenant configuration
- `webhooks` - Webhook configurations per tenant
- `alerts` - Security alerts per tenant
- `feature_flags` - Feature toggles per tenant
- `threat_intelligence` - Threat data per tenant

### Global Tables (No RLS)
These tables are shared across the platform:

- `tenants` - Master tenant registry (admin access only)
- `users` - Global user directory (partial RLS)
- `admin_users` - Platform administrators
- `api_keys` - Global API key management
- `subscription_plans` - Available subscription plans
- `subscriptions` - Billing data (isolated via application logic)

## 🚀 Implementation

### 1. Database Functions

The RLS system uses these key functions:

```sql
-- Set tenant context for the current session
SELECT set_tenant_context('tenant-uuid-here');

-- Get current tenant ID from session context
SELECT get_current_tenant_id();

-- Validate tenant access (returns boolean)
SELECT validate_tenant_access('tenant-uuid-here');

-- Check if current user is platform admin
SELECT is_platform_admin();
```

### 2. Go Application Integration

#### Middleware Implementation

The tenant validation middleware automatically sets database context:

```go
// In internal/interfaces/http/api/middleware_tenant.go
func tenantValidationMiddleware(db *sql.DB) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Extract tenant ID from X-PS-Tenant-ID header
            tenantID := extractTenantID(r)
            
            // CRITICAL: Set tenant context in database for RLS
            if err := setTenantContextInDB(db, tenantID); err != nil {
                // Reject request if RLS context fails
                http.Error(w, "Tenant context error", 500)
                return
            }
            
            // Add tenant to request context
            ctx := context.WithValue(r.Context(), "tenant_id", tenantID)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
```

#### Database Wrapper

Use the TenantDB wrapper for automatic context setting:

```go
// Create tenant-aware database wrapper
tdb := postgres.NewTenantDB(db)

// All queries automatically set tenant context
rows, err := tdb.QueryContext(ctx, "SELECT * FROM rulepacks")
// This query will only return current tenant's rulepacks
```

## 🧪 Testing Tenant Isolation

### Automated Test Script

Run the isolation test:

```bash
cd scripts
go run test-tenant-isolation.go
```

Expected output:
```
🧪 Testing Tenant Isolation with RLS
=====================================
✅ Connected to database

📝 Test 1: Creating tenant-specific rulepacks...
✅ Created rulepacks for both tenants

🔒 Test 2: Verifying tenant isolation...
Tenant 1 sees 1 rulepacks
Tenant 2 sees 1 rulepacks

🚫 Test 3: Testing cross-tenant access prevention...
✅ Tenant 1 correctly cannot access Tenant 2's rulepack

👑 Test 4: Testing platform admin access...
Platform admin sees 2 rulepacks (should see all)

📊 Test 5: Testing audit log isolation...
Tenant 1 sees 1 audit events
Tenant 2 sees 1 audit events

✅ Tenant isolation is working correctly!
```

### Manual Testing

Test with different tenant IDs using curl:

```bash
# Tenant 1 requests
curl -H "X-PS-Tenant-ID: 11111111-1111-1111-1111-111111111111" \
     http://localhost:9090/rulepacks

# Tenant 2 requests  
curl -H "X-PS-Tenant-ID: 22222222-2222-2222-2222-222222222222" \
     http://localhost:9090/rulepacks

# Should return different data sets
```

## 🔧 Configuration

### Environment Variables

```bash
# Database connection (required)
PS_PG_DSN=postgresql://user:pass@host:5432/database

# Tenant ID for single-tenant deployments (optional)
PS_TENANT_ID=your-tenant-uuid-here
```

### Headers Required

All API requests must include:

```bash
X-PS-Tenant-ID: your-tenant-uuid-here
X-PS-Frontend-Auth: verified  # or valid auth token
```

## 🔍 Monitoring & Debugging

### Verify RLS Status

Check which tables have RLS enabled:

```sql
SELECT schemaname, tablename, rowsecurity 
FROM pg_tables 
WHERE schemaname = 'public' AND rowsecurity = true
ORDER BY tablename;
```

### Debug Tenant Context

Check current tenant context:

```sql
SELECT get_current_tenant_id() as current_tenant;
```

### View RLS Policies

List all RLS policies:

```sql
SELECT schemaname, tablename, policyname, permissive, roles, cmd, qual
FROM pg_policies
WHERE schemaname = 'public'
ORDER BY tablename, policyname;
```

## ⚠️ Security Considerations

### Critical Requirements

1. **Always Set Tenant Context**: Every request MUST set tenant context
2. **Validate Headers**: Always validate X-PS-Tenant-ID header format
3. **Handle Failures**: Reject requests if tenant context setting fails
4. **Audit Access**: Log all tenant context changes
5. **Test Regularly**: Run isolation tests in CI/CD pipeline

### Common Pitfalls

❌ **Don't Skip Validation**: Never skip tenant ID validation  
❌ **Don't Ignore Errors**: Always handle tenant context errors  
❌ **Don't Use Raw SQL**: Use the TenantDB wrapper when possible  
❌ **Don't Cache Cross-Tenant**: Avoid caching data across tenant boundaries  

## 🚨 Emergency Procedures

### Tenant Data Breach Response

If you suspect cross-tenant data access:

1. **Immediate**: Stop all API traffic
2. **Investigate**: Check audit logs for suspicious activity
3. **Verify**: Run tenant isolation tests
4. **Fix**: Restore RLS policies if needed
5. **Report**: Document and report the incident

### RLS Policy Recovery

If RLS policies are accidentally dropped:

```bash
# Re-run the RLS migration
psql $PS_PG_DSN -f migrations/0009_rls_policies.sql

# Verify policies are restored
psql $PS_PG_DSN -c "SELECT COUNT(*) FROM pg_policies WHERE schemaname = 'public';"
```

## 📚 Additional Resources

- [PostgreSQL RLS Documentation](https://www.postgresql.org/docs/current/ddl-rowsecurity.html)
- [Multi-Tenant SaaS Architecture Guide](docs/MULTI_TENANT_FRONTEND_GUIDE.md)
- [Security Best Practices](docs/SECURITY.md)
- [API Integration Guide](docs/FRONTEND_API_GUIDE.md)

---

**Important**: Tenant isolation is a critical security feature. Any changes to RLS policies or tenant middleware should be thoroughly tested and reviewed.