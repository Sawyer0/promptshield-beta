package services

import (
	"testing"

	"github.com/promptshield/promptshield/internal/repository"
)

// TestServiceMigrationToFactory verifies that services can be created using the factory pattern
func TestServiceMigrationToFactory(t *testing.T) {
	// Create test repository factory
	factory, err := repository.NewTestRepositoryFactory(nil, nil)
	if err != nil {
		t.Fatalf("Failed to create test repository factory: %v", err)
	}

	// Test RulepackService creation
	rulepackService := RulepackServiceFromFactory(factory, nil)
	if rulepackService == nil {
		t.Fatal("RulepackServiceFromFactory returned nil")
	}

	// Test PolicyService creation
	policyService := PolicyServiceFromFactory(factory)
	if policyService == nil {
		t.Fatal("PolicyServiceFromFactory returned nil")
	}

	// Test PolicyScannerService creation
	policyScannerService := PolicyScannerServiceFromFactory(factory)
	if policyScannerService == nil {
		t.Fatal("PolicyScannerServiceFromFactory returned nil")
	}

	// Test NewServicesFromFactory
	services := NewServicesFromFactory(factory, nil)
	if services == nil {
		t.Fatal("NewServicesFromFactory returned nil")
	}
	if services.Rulepack == nil {
		t.Fatal("NewServicesFromFactory returned nil Rulepack service")
	}
	if services.Policy == nil {
		t.Fatal("NewServicesFromFactory returned nil Policy service")
	}
	if services.PolicyScanner == nil {
		t.Fatal("NewServicesFromFactory returned nil PolicyScanner service")
	}
}