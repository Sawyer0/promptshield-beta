package audit

import (
	"github.com/promptshield/promptshield/internal/shared/contracts"
)

// NewAuditService creates a new audit service instance (no global state)
func NewAuditService(logger contracts.AuditLogger) *Service {
	return &Service{
		logger: logger,
	}
}