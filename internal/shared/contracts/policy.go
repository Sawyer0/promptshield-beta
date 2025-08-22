package contracts

import (
	"context"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/shared/types"
)

// PolicyEvaluator defines the interface for policy evaluation and enforcement
type PolicyEvaluator interface {
	// EvaluateRequest evaluates a request against assigned policies
	EvaluateRequest(ctx context.Context, req *types.LLMRequest, policyCtx *types.PolicyContext) (*types.ScanResult, error)
	
	// EvaluateResponse evaluates a response against assigned policies
	EvaluateResponse(ctx context.Context, resp *types.LLMResponse, policyCtx *types.PolicyContext) (*types.ScanResult, error)
	
	// GetApplicablePolicies returns policies applicable to the given context
	GetApplicablePolicies(ctx context.Context, policyCtx *types.PolicyContext) ([]*types.Policy, error)
}

// PolicyEnforcer defines the interface for policy enforcement actions
type PolicyEnforcer interface {
	// EnforceDecision applies enforcement based on scan results
	EnforceDecision(ctx context.Context, result *types.ScanResult, mode types.EnforcementMode) (*types.EnforcementDecision, error)
	
	// RedactContent applies redaction to sensitive content
	RedactContent(ctx context.Context, content string, violations []types.PolicyViolation) (string, error)
	
	// QuarantineRequest handles quarantining of requests for review
	QuarantineRequest(ctx context.Context, req *types.LLMRequest, violations []types.PolicyViolation) error
}

// RuleCompiler defines the interface for rule compilation and validation
type RuleCompiler interface {
	// CompileRules compiles and validates rules for a policy
	CompileRules(ctx context.Context, policy *types.Policy) error
	
	// ValidatePolicy validates policy structure and rules
	ValidatePolicy(ctx context.Context, policy *types.Policy) error
	
	// GetRuleInfo returns information about rules in a policy
	GetRuleInfo(ctx context.Context, policyID string) ([]*types.RuleInfo, error)
}

// ScanEngine defines the interface for content scanning
type ScanEngine interface {
	// ScanText scans text content for policy violations
	ScanText(ctx context.Context, content string, policyCtx *types.PolicyContext) (*types.ScanResult, error)
	
	// ScanStream scans streaming content for policy violations
	ScanStream(ctx context.Context, stream <-chan []byte, policyCtx *types.PolicyContext) (*types.ScanResult, error)
	
	// GetScanConfig returns current scan configuration
	GetScanConfig() map[string]interface{}
}

// SemanticAnalyzer defines the interface for LLM-based semantic analysis (Level 3)
type SemanticAnalyzer interface {
	// AnalyzeContent performs semantic analysis on content
	AnalyzeContent(ctx context.Context, content string, rules []types.RuleInfo) ([]*types.PolicyViolation, error)
	
	// SupportsModel returns true if the analyzer supports the given model
	SupportsModel(model string) bool
	
	// GetProvider returns the provider name (openai, anthropic, etc.)
	GetProvider() types.Provider
	
	// HealthCheck verifies the analyzer is working correctly
	HealthCheck(ctx context.Context) error
}

// ViolationHandler defines the interface for handling policy violations
type ViolationHandler interface {
	// HandleViolation processes a detected violation
	HandleViolation(ctx context.Context, violation *types.PolicyViolation, policyCtx *types.PolicyContext) error
	
	// ReportViolation reports a violation for monitoring/alerting
	ReportViolation(ctx context.Context, violation *types.PolicyViolation, metadata map[string]interface{}) error
	
	// GetViolationHistory returns historical violations for analysis
	GetViolationHistory(ctx context.Context, tenantID string, limit int) ([]*types.PolicyViolation, error)
}

// DecisionCache defines the interface for caching enforcement decisions
type DecisionCache interface {
	// GetDecision retrieves a cached decision for the given content hash
	GetDecision(ctx context.Context, contentHash string) (*types.EnforcementDecision, error)
	
	// SetDecision caches an enforcement decision
	SetDecision(ctx context.Context, contentHash string, decision *types.EnforcementDecision) error
	
	// InvalidateDecision removes a cached decision
	InvalidateDecision(ctx context.Context, contentHash string) error
	
	// InvalidateTenant removes all cached decisions for a tenant
	InvalidateTenant(ctx context.Context, tenantID string) error
}


// PolicyService defines the interface for policy management business logic
type PolicyService interface {
	// CreatePolicy creates a new policy with validation
	CreatePolicy(ctx context.Context, policy *types.Policy) (*types.Policy, error)
	
	// UpdatePolicy updates an existing policy
	UpdatePolicy(ctx context.Context, policy *types.Policy) (*types.Policy, error)
	
	// DeletePolicy deletes a policy
	DeletePolicy(ctx context.Context, id uuid.UUID) error
	
	// GetPolicy retrieves a policy by ID
	GetPolicy(ctx context.Context, id uuid.UUID) (*types.Policy, error)
	
	// ListPolicies lists policies with filtering
	ListPolicies(ctx context.Context, filter map[string]interface{}) ([]*types.Policy, int, error)
	
	// ActivatePolicy activates a policy and triggers scanner reload
	ActivatePolicy(ctx context.Context, id uuid.UUID) error
	
	// DeactivatePolicy deactivates a policy
	DeactivatePolicy(ctx context.Context, id uuid.UUID) error
	
	// ValidatePolicy validates policy structure and rules
	ValidatePolicy(ctx context.Context, policy *types.Policy) error
	
	// TestPolicy tests content against a specific policy
	TestPolicy(ctx context.Context, policyID uuid.UUID, content string) (*types.ScanResult, error)
	
	// Real-time enforcement support
	GetActiveScanner() interface{} // Returns scanner for real-time enforcement
	HasActivePolicies() bool       // Check if any policies are active
}