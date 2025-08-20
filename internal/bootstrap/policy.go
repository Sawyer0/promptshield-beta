package bootstrap

import (
	"github.com/promptshield/promptshield/internal/application/services"
	"github.com/promptshield/promptshield/internal/infrastructure/persistence/memory"
	"github.com/promptshield/promptshield/internal/shared/contracts"
)

// PolicyDependencies holds all policy management dependencies
type PolicyDependencies struct {
	Repository contracts.PolicyRepository
	Service    contracts.PolicyService
}

// InitializePolicyDependencies creates and wires up all policy management dependencies
func InitializePolicyDependencies(
	ruleCompiler contracts.RuleCompiler,
	scanEngine contracts.ScanEngine,
	auditLogger contracts.AuditLogger,
) *PolicyDependencies {
	// Create repository (in-memory for MVP, replace with database for production)
	repository := memory.NewPolicyRepository()
	
	// Create service with dependencies
	service := services.NewPolicyService(
		repository,
		ruleCompiler,
		scanEngine,
		auditLogger,
	)
	
	return &PolicyDependencies{
		Repository: repository,
		Service:    service,
	}
}