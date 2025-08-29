package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/shared/contracts"
	"github.com/promptshield/promptshield/internal/shared/types"
)

// pgAuditTrailManager implements contracts.AuditTrailManager with PostgreSQL backend
type pgAuditTrailManager struct {
	eventStore contracts.AuditEventStore
}

// NewAuditTrailManager creates a new PostgreSQL audit trail manager
func NewAuditTrailManager(eventStore contracts.AuditEventStore) contracts.AuditTrailManager {
	return &pgAuditTrailManager{
		eventStore: eventStore,
	}
}

// CreateTrail creates a new audit trail for an entity
func (m *pgAuditTrailManager) CreateTrail(ctx context.Context, entityID string, entityType string) error {
	// Validate entity ID format
	_, err := uuid.Parse(entityID)
	if err != nil {
		return fmt.Errorf("invalid entity ID format: %w", err)
	}

	// In PostgreSQL, the trail is implicitly created when events are stored
	// We could add a trails table for metadata, but for now we'll work with events directly
	// Create an initial trail marker event
	trailEvent := &types.AuditEvent{
		TenantID:   nil, // Will be set by caller if needed
		ActorID:    nil,
		ActorType:  types.ActorTypeSystem,
		ActorEmail: "system@promptshield.io",
		Action:     "trail.create",
		ObjectType: entityType,
		ObjectID:   uuid.MustParse(entityID),
		Before:     nil,
		After: map[string]interface{}{
			"trail_created": true,
			"entity_type":   entityType,
		},
		Metadata: map[string]interface{}{
			"trail_operation": "create",
		},
		Timestamp: time.Now(),
	}

	return m.eventStore.Store(ctx, trailEvent)
}

// GetTrail retrieves the audit trail for an entity
func (m *pgAuditTrailManager) GetTrail(ctx context.Context, entityID string, entityType string) ([]*types.AuditEvent, error) {
	entityUUID, err := uuid.Parse(entityID)
	if err != nil {
		return nil, fmt.Errorf("invalid entity ID format: %w", err)
	}

	filter := &types.AuditFilter{
		ObjectID:   &entityUUID,
		ObjectType: entityType,
	}

	events, err := m.eventStore.Retrieve(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("retrieve audit trail: %w", err)
	}

	// Sort events by timestamp (should already be sorted, but ensure consistency)
	sort.Slice(events, func(i, j int) bool {
		return events[i].Timestamp.Before(events[j].Timestamp)
	})

	return events, nil
}

// AppendToTrail appends an event to an audit trail
func (m *pgAuditTrailManager) AppendToTrail(ctx context.Context, entityID string, event *types.AuditEvent) error {
	entityUUID, err := uuid.Parse(entityID)
	if err != nil {
		return fmt.Errorf("invalid entity ID format: %w", err)
	}

	// Ensure the event belongs to the correct entity
	event.ObjectID = entityUUID

	return m.eventStore.Store(ctx, event)
}

// GetTrailSummary returns a summary of the audit trail
func (m *pgAuditTrailManager) GetTrailSummary(ctx context.Context, entityID string) (*types.AuditTrailSummary, error) {
	events, err := m.GetTrail(ctx, entityID, "") // Get all events for entity
	if err != nil {
		return nil, fmt.Errorf("get trail for summary: %w", err)
	}

	if len(events) == 0 {
		return &types.AuditTrailSummary{
			EntityID:    entityID,
			TotalEvents: 0,
			TimeRange: types.TimeRange{
				Start: time.Now(),
				End:   time.Now(),
			},
			EventTypes: make(map[string]int64),
		}, nil
	}

	summary := &types.AuditTrailSummary{
		EntityID:    entityID,
		TotalEvents: int64(len(events)),
		FirstEvent:  events[0],
		LastEvent:   events[len(events)-1],
		TimeRange: types.TimeRange{
			Start: events[0].Timestamp,
			End:   events[len(events)-1].Timestamp,
		},
		EventTypes: make(map[string]int64),
		Users:      make([]string, 0),
		Resources:  make([]string, 0),
		Actions:    make([]string, 0),
	}

	userSet := make(map[string]bool)
	resourceSet := make(map[string]bool)
	actionSet := make(map[string]bool)

	// Analyze events
	for _, event := range events {
		// Count event types
		summary.EventTypes[event.Action]++

		// Track unique users
		if event.ActorID != nil {
			userID := event.ActorID.String()
			if !userSet[userID] {
				userSet[userID] = true
				summary.Users = append(summary.Users, userID)
			}
		}

		// Track resources (objects)
		resourceID := event.ObjectID.String()
		if !resourceSet[resourceID] {
			resourceSet[resourceID] = true
			summary.Resources = append(summary.Resources, resourceID)
		}

		// Track actions
		if !actionSet[event.Action] {
			actionSet[event.Action] = true
			summary.Actions = append(summary.Actions, event.Action)
		}
	}

	// Calculate risk score based on activity patterns
	summary.RiskScore = m.calculateRiskScore(events)

	return summary, nil
}

// ExportTrail exports an audit trail in the specified format
func (m *pgAuditTrailManager) ExportTrail(ctx context.Context, entityID string, format string) ([]byte, error) {
	events, err := m.GetTrail(ctx, entityID, "")
	if err != nil {
		return nil, fmt.Errorf("get trail for export: %w", err)
	}

	switch format {
	case "json":
		return m.exportAsJSON(events)
	case "csv":
		return m.exportAsCSV(events)
	case "xml":
		return m.exportAsXML(events)
	default:
		return nil, fmt.Errorf("unsupported export format: %s", format)
	}
}

// ValidateTrail validates audit trail integrity
func (m *pgAuditTrailManager) ValidateTrail(ctx context.Context, entityID string) (*types.TrailValidation, error) {
	events, err := m.GetTrail(ctx, entityID, "")
	if err != nil {
		return nil, fmt.Errorf("get trail for validation: %w", err)
	}

	validation := &types.TrailValidation{
		EntityID:    entityID,
		IsValid:     true,
		TotalEvents: int64(len(events)),
		ValidatedAt: time.Now(),
	}

	if len(events) == 0 {
		return validation, nil
	}

	// Validate chronological order
	for i := 1; i < len(events); i++ {
		if events[i].Timestamp.Before(events[i-1].Timestamp) {
			validation.IsValid = false
			validation.Errors = append(validation.Errors, "events are not in chronological order")
			break
		}
	}

	// Check for gaps in sequence (basic check)
	expectedSequence := int64(0)
	for _, event := range events {
		if event.Metadata != nil {
			if seq, ok := event.Metadata["sequence_id"].(float64); ok {
				if int64(seq) != expectedSequence {
					validation.Warnings = append(validation.Warnings, "potential gap in event sequence")
				}
				expectedSequence++
			}
		}
	}

	// Validate hash chain if events have hash information
	if len(events) > 0 && events[0].Hash != "" {
		var prevHash string
		for _, event := range events {
			if event.PrevHash != prevHash {
				validation.IsValid = false
				validation.Errors = append(validation.Errors, "hash chain is broken")
				break
			}
			prevHash = event.Hash
		}
	}

	return validation, nil
}

// MergeTrails merges multiple audit trails
func (m *pgAuditTrailManager) MergeTrails(ctx context.Context, entityIDs []string) ([]*types.AuditEvent, error) {
	if len(entityIDs) == 0 {
		return nil, fmt.Errorf("no entity IDs provided")
	}

	allEvents := make([]*types.AuditEvent, 0)

	for _, entityID := range entityIDs {
		events, err := m.GetTrail(ctx, entityID, "")
		if err != nil {
			return nil, fmt.Errorf("get trail for entity %s: %w", entityID, err)
		}
		allEvents = append(allEvents, events...)
	}

	// Sort all events by timestamp
	sort.Slice(allEvents, func(i, j int) bool {
		return allEvents[i].Timestamp.Before(allEvents[j].Timestamp)
	})

	return allEvents, nil
}

// Helper functions

func (m *pgAuditTrailManager) calculateRiskScore(events []*types.AuditEvent) float64 {
	if len(events) == 0 {
		return 0.0
	}

	score := 0.0

	// Risk factors
	for _, event := range events {
		switch event.Action {
		case "violation.detected", "request.blocked":
			score += 2.0
		case "policy.delete", "provider_key.revoke":
			score += 1.5
		case "tenant.suspend", "user.suspend":
			score += 1.0
		case "request.allowed":
			score += 0.1
		default:
			score += 0.5
		}
	}

	// Normalize by number of events and time span
	if len(events) > 1 {
		timeSpan := events[len(events)-1].Timestamp.Sub(events[0].Timestamp)
		if timeSpan.Hours() > 24 {
			// Reduce score for events spread over time
			score *= 24 / timeSpan.Hours()
		}
	}

	// Cap at 10.0
	if score > 10.0 {
		score = 10.0
	}

	return score
}

func (m *pgAuditTrailManager) exportAsJSON(events []*types.AuditEvent) ([]byte, error) {
	return json.MarshalIndent(events, "", "  ")
}

func (m *pgAuditTrailManager) exportAsCSV(events []*types.AuditEvent) ([]byte, error) {
	if len(events) == 0 {
		return []byte("timestamp,action,actor_type,actor_email,object_type,object_id\n"), nil
	}

	var csvData string
	csvData += "timestamp,action,actor_type,actor_email,object_type,object_id,before,after\n"

	for _, event := range events {
		before, _ := json.Marshal(event.Before)
		after, _ := json.Marshal(event.After)

		csvData += fmt.Sprintf("%s,%s,%s,%s,%s,%s,%q,%q\n",
			event.Timestamp.Format(time.RFC3339),
			event.Action,
			event.ActorType,
			event.ActorEmail,
			event.ObjectType,
			event.ObjectID.String(),
			string(before),
			string(after))
	}

	return []byte(csvData), nil
}

func (m *pgAuditTrailManager) exportAsXML(events []*types.AuditEvent) ([]byte, error) {
	xml := "<audit_trail>\n"

	for _, event := range events {
		xml += "  <event>\n"
		xml += fmt.Sprintf("    <timestamp>%s</timestamp>\n", event.Timestamp.Format(time.RFC3339))
		xml += fmt.Sprintf("    <action>%s</action>\n", event.Action)
		xml += fmt.Sprintf("    <actor_type>%s</actor_type>\n", event.ActorType)
		xml += fmt.Sprintf("    <actor_email>%s</actor_email>\n", event.ActorEmail)
		xml += fmt.Sprintf("    <object_type>%s</object_type>\n", event.ObjectType)
		xml += fmt.Sprintf("    <object_id>%s</object_id>\n", event.ObjectID.String())

		if event.Before != nil {
			before, _ := json.Marshal(event.Before)
			xml += fmt.Sprintf("    <before>%s</before>\n", string(before))
		}

		if event.After != nil {
			after, _ := json.Marshal(event.After)
			xml += fmt.Sprintf("    <after>%s</after>\n", string(after))
		}

		xml += "  </event>\n"
	}

	xml += "</audit_trail>"
	return []byte(xml), nil
}
