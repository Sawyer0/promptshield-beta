package contextkeys

// Use unexported, distinct types for context keys to avoid collisions (SA1029)
// and provide package-scoped exported variables for usage.

type tenantIDKey struct{}

// TenantID is the context key for the current tenant identifier.
var TenantID tenantIDKey
