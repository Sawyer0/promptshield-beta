package validation

import (
	"time"
	
	"github.com/google/uuid"
	sharederrors "github.com/promptshield/promptshield/internal/shared/errors"
)

// ValidateTenantID validates tenant ID string and returns UUID
func ValidateTenantID(tenantID string) (uuid.UUID, error) {
	// Check if tenant ID is provided
	if err := Required("tenant_id", tenantID); err != nil {
		return uuid.Nil, sharederrors.Forbidden("tenant context required")
	}
	
	// Validate UUID format
	if err := UUID("tenant_id", tenantID); err != nil {
		return uuid.Nil, err
	}
	
	// Convert to UUID (safe since validated)
	tenantUUID, _ := uuid.Parse(tenantID)
	return tenantUUID, nil
}

// ValidateTimeRange validates and parses time range parameters
func ValidateTimeRange(startStr, endStr string, defaultDuration time.Duration) (start, end time.Time, err error) {
	end = time.Now().UTC()
	start = end.Add(-defaultDuration)
	
	if startStr != "" {
		if t, parseErr := time.Parse(time.RFC3339, startStr); parseErr != nil {
			return start, end, sharederrors.InvalidFieldValue("start", startStr, "RFC3339 timestamp")
		} else {
			start = t
		}
	}
	
	if endStr != "" {
		if t, parseErr := time.Parse(time.RFC3339, endStr); parseErr != nil {
			return start, end, sharederrors.InvalidFieldValue("end", endStr, "RFC3339 timestamp")  
		} else {
			end = t
		}
	}
	
	// Validate time range logic
	if !start.Before(end) {
		return start, end, sharederrors.InvalidFieldValue("time_range", "", "start must be before end")
	}
	
	return start, end, nil
}