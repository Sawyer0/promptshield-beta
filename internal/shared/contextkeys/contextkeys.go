package contextkeys

// Use unexported, distinct types for context keys to avoid collisions (SA1029)
// and provide package-scoped exported variables for usage.

type tenantIDKey struct{}

// TenantID is the context key for the current tenant identifier.
var TenantID tenantIDKey

type endpointPathKey struct{}

// EndpointPath is the context key for the original application endpoint path that
// PromptShield should apply rulepack assignments against (e.g., "/v1/chat/completions").
// When unset, assignment-based loading is skipped and the runtime falls back to defaults.
var EndpointPath endpointPathKey
