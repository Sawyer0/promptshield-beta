package postgres

import (
	"log/slog"

	"github.com/promptshield/promptshield/internal/shared/contracts"
	"github.com/promptshield/promptshield/internal/shared/types"
)

// AuditServiceFactory creates and configures all audit-related services
type AuditServiceFactory struct {
	db     *Pool
	logger *slog.Logger
}

// NewAuditServiceFactory creates a new audit service factory
func NewAuditServiceFactory(db *Pool, logger *slog.Logger) *AuditServiceFactory {
	return &AuditServiceFactory{
		db:     db,
		logger: logger,
	}
}

// CreateEventStore creates an audit event store
func (f *AuditServiceFactory) CreateEventStore() contracts.AuditEventStore {
	return NewAuditEventStore(f.db)
}

// CreateTrailManager creates an audit trail manager
func (f *AuditServiceFactory) CreateTrailManager(eventStore contracts.AuditEventStore) contracts.AuditTrailManager {
	return NewAuditTrailManager(eventStore)
}

// CreateReporter creates an audit reporter
func (f *AuditServiceFactory) CreateReporter(eventStore contracts.AuditEventStore) contracts.AuditReporter {
	return NewAuditReporter(eventStore)
}

// CreateAnalyzer creates an audit analyzer
func (f *AuditServiceFactory) CreateAnalyzer(eventStore contracts.AuditEventStore) contracts.AuditAnalyzer {
	return NewAuditAnalyzer(eventStore)
}

// CreateCompliance creates an audit compliance service
func (f *AuditServiceFactory) CreateCompliance(eventStore contracts.AuditEventStore) contracts.AuditCompliance {
	return NewAuditCompliance(eventStore)
}

// CreateNotifier creates an audit notifier
func (f *AuditServiceFactory) CreateNotifier(eventStore contracts.AuditEventStore) contracts.AuditNotifier {
	return NewAuditNotifier(eventStore, f.logger)
}

// CreateHashChain creates an audit hash chain
func (f *AuditServiceFactory) CreateHashChain(eventStore contracts.AuditEventStore) contracts.AuditHashChain {
	return NewAuditHashChain(eventStore)
}

// CreateAuditServices creates all audit services with proper dependencies
func (f *AuditServiceFactory) CreateAuditServices() *AuditServices {
	// Create event store first (foundation service)
	eventStore := f.CreateEventStore()

	// Create all other services with event store dependency
	return &AuditServices{
		EventStore:   eventStore,
		TrailManager: f.CreateTrailManager(eventStore),
		Reporter:     f.CreateReporter(eventStore),
		Analyzer:     f.CreateAnalyzer(eventStore),
		Compliance:   f.CreateCompliance(eventStore),
		Notifier:     f.CreateNotifier(eventStore),
		HashChain:    f.CreateHashChain(eventStore),
	}
}

// AuditServices holds all audit-related services
type AuditServices struct {
	EventStore   contracts.AuditEventStore
	TrailManager contracts.AuditTrailManager
	Reporter     contracts.AuditReporter
	Analyzer     contracts.AuditAnalyzer
	Compliance   contracts.AuditCompliance
	Notifier     contracts.AuditNotifier
	HashChain    contracts.AuditHashChain
}

// InitializeAuditSystem initializes the complete audit system
func (f *AuditServiceFactory) InitializeAuditSystem() (*AuditServices, error) {
	services := f.CreateAuditServices()

	f.logger.Info("Audit system initialized",
		"services", []string{
			"event_store",
			"trail_manager",
			"reporter",
			"analyzer",
			"compliance",
			"notifier",
			"hash_chain",
		})

	return services, nil
}

// CreateDefaultAlerts creates default audit alerts
func (f *AuditServiceFactory) CreateDefaultAlerts() []*types.AuditAlert {
	return []*types.AuditAlert{
		{
			ID:          "security-violations",
			Name:        "Security Violations",
			Description: "Alert on security policy violations",
			Type:        "security",
			Severity:    "HIGH",
			Condition:   "security_violation",
			Enabled:     true,
			Recipients:  []string{"security-team@promptshield.io"},
		},
		{
			ID:          "failed-logins",
			Name:        "Failed Login Attempts",
			Description: "Alert on multiple failed login attempts",
			Type:        "security",
			Severity:    "MEDIUM",
			Condition:   "failed_login",
			Enabled:     true,
			Recipients:  []string{"security-team@promptshield.io"},
		},
		{
			ID:          "compliance-violations",
			Name:        "Compliance Violations",
			Description: "Alert on compliance policy violations",
			Type:        "compliance",
			Severity:    "HIGH",
			Condition:   "compliance_violation",
			Enabled:     true,
			Recipients:  []string{"compliance@promptshield.io"},
		},
		{
			ID:          "privilege-changes",
			Name:        "Privilege Changes",
			Description: "Alert on privilege escalation events",
			Type:        "access",
			Severity:    "MEDIUM",
			Condition:   "privilege_change",
			Enabled:     true,
			Recipients:  []string{"admin@promptshield.io"},
		},
	}
}
