package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/shared/contracts"
	"github.com/promptshield/promptshield/internal/shared/types"
)

// pgAuditHashChain implements contracts.AuditHashChain with PostgreSQL backend
type pgAuditHashChain struct {
	eventStore contracts.AuditEventStore
}

// NewAuditHashChain creates a new PostgreSQL audit hash chain
func NewAuditHashChain(eventStore contracts.AuditEventStore) contracts.AuditHashChain {
	return &pgAuditHashChain{
		eventStore: eventStore,
	}
}

// AppendEvent appends an event to the hash chain
func (h *pgAuditHashChain) AppendEvent(ctx context.Context, event *types.AuditEvent) (string, error) {
	if event == nil {
		return "", fmt.Errorf("event cannot be nil")
	}

	// Calculate event hash
	eventHash := h.calculateEventHash(event)

	// Get previous hash from the most recent event
	prevHash, err := h.getLatestHash(ctx)
	if err != nil {
		return "", fmt.Errorf("get latest hash: %w", err)
	}

	// Set hash chain information
	event.Hash = eventHash
	event.PrevHash = prevHash

	// Store the event
	err = h.eventStore.Store(ctx, event)
	if err != nil {
		return "", fmt.Errorf("store event with hash chain: %w", err)
	}

	return eventHash, nil
}

// VerifyChain verifies the integrity of the hash chain
func (h *pgAuditHashChain) VerifyChain(ctx context.Context, startHash string, endHash string) (*types.ChainVerification, error) {
	// Get events in the specified range
	startTime := time.Now().Add(-24 * time.Hour)
	endTime := time.Now()
	filter := &types.AuditFilter{
		// For simplicity, get recent events - in production you'd filter by hash range
		StartTime: &startTime,
		EndTime:   &endTime,
	}

	events, err := h.eventStore.Retrieve(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("retrieve events for chain verification: %w", err)
	}

	verification := &types.ChainVerification{
		StartHash:      startHash,
		EndHash:        endHash,
		TotalEvents:    int64(len(events)),
		VerifiedEvents: 0,
		BrokenLinks:    make([]*types.BrokenLink, 0),
		IsValid:        true,
		VerifiedAt:     time.Now(),
	}

	if len(events) == 0 {
		return verification, nil
	}

	// Sort events by timestamp for verification
	sort.Slice(events, func(i, j int) bool {
		return events[i].Timestamp.Before(events[j].Timestamp)
	})

	// Verify the chain
	var prevHash string
	for i, event := range events {
		// Verify current event's hash
		expectedHash := h.calculateEventHash(event)
		if event.Hash != expectedHash {
			verification.IsValid = false
			verification.BrokenLinks = append(verification.BrokenLinks, &types.BrokenLink{
				Position:     int64(i),
				ExpectedHash: expectedHash,
				ActualHash:   event.Hash,
				EventID:      event.ObjectID.String(),
			})
			continue
		}

		// Verify chain continuity (except for the first event)
		if i > 0 && event.PrevHash != prevHash {
			verification.IsValid = false
			verification.BrokenLinks = append(verification.BrokenLinks, &types.BrokenLink{
				Position:     int64(i),
				ExpectedHash: prevHash,
				ActualHash:   event.PrevHash,
				EventID:      event.ObjectID.String(),
			})
		}

		verification.VerifiedEvents++
		prevHash = event.Hash
	}

	return verification, nil
}

// GetChainInfo returns information about the hash chain
func (h *pgAuditHashChain) GetChainInfo(ctx context.Context) (*types.ChainInfo, error) {
	// Get latest hash
	latestHash, err := h.getLatestHash(ctx)
	if err != nil {
		return nil, fmt.Errorf("get latest hash: %w", err)
	}

	// Get total events
	totalEvents, err := h.eventStore.Count(ctx, &types.AuditFilter{})
	if err != nil {
		return nil, fmt.Errorf("count total events: %w", err)
	}

	// Get first and last events
	recentEvents, err := h.eventStore.Retrieve(ctx, &types.AuditFilter{
		Limit: 1,
	})
	if err != nil {
		return nil, fmt.Errorf("get recent events: %w", err)
	}

	var firstEvent, lastEvent *types.AuditEvent
	if len(recentEvents) > 0 {
		lastEvent = recentEvents[0]
	}

	// Get oldest event (this is inefficient but works for demonstration)
	oldStart := time.Now().Add(-365 * 24 * time.Hour)
	oldEnd := time.Now().Add(-364 * 24 * time.Hour)
	oldEvents, err := h.eventStore.Retrieve(ctx, &types.AuditFilter{
		StartTime: &oldStart, // Last year
		EndTime:   &oldEnd,   // 1 day ago
		Limit:     1,
	})
	if err == nil && len(oldEvents) > 0 {
		firstEvent = oldEvents[0]
	}

	info := &types.ChainInfo{
		CurrentHash: latestHash,
		TotalEvents: totalEvents,
		FirstEvent:  firstEvent,
		LastEvent:   lastEvent,
		GeneratedAt: time.Now(),
	}

	return info, nil
}

// RepairChain attempts to repair a broken hash chain
func (h *pgAuditHashChain) RepairChain(ctx context.Context, fromEventID string) error {
	// Parse the event ID
	eventUUID, err := uuid.Parse(fromEventID)
	if err != nil {
		return fmt.Errorf("invalid event ID: %w", err)
	}

	// Get the event to repair from
	event, err := h.eventStore.GetByID(ctx, fromEventID)
	if err != nil {
		return fmt.Errorf("get event for repair: %w", err)
	}

	// Recalculate hash for this event
	newHash := h.calculateEventHash(event)

	// Get the previous event's hash
	filter := &types.AuditFilter{
		EndTime: &event.Timestamp,
	}
	prevEvents, err := h.eventStore.Retrieve(ctx, filter)
	if err != nil {
		return fmt.Errorf("get previous events: %w", err)
	}

	var prevHash string
	if len(prevEvents) > 1 {
		// Find the most recent event before this one
		sort.Slice(prevEvents, func(i, j int) bool {
			return prevEvents[i].Timestamp.After(prevEvents[j].Timestamp)
		})
		for _, prevEvent := range prevEvents {
			if prevEvent.ObjectID != eventUUID && prevEvent.Timestamp.Before(event.Timestamp) {
				prevHash = prevEvent.Hash
				break
			}
		}
	}

	// Update the event with corrected hashes
	// Note: In a real implementation, you'd need to update the database directly
	// or provide an update method in the event store
	_ = newHash
	_ = prevHash

	return fmt.Errorf("repair chain functionality requires database update methods not yet implemented")
}

// ExportChain exports the hash chain for verification
func (h *pgAuditHashChain) ExportChain(ctx context.Context, timeRange types.TimeRange) ([]byte, error) {
	events, err := h.eventStore.Retrieve(ctx, &types.AuditFilter{
		StartTime: &timeRange.Start,
		EndTime:   &timeRange.End,
	})
	if err != nil {
		return nil, fmt.Errorf("retrieve events for export: %w", err)
	}

	// Sort events chronologically
	sort.Slice(events, func(i, j int) bool {
		return events[i].Timestamp.Before(events[j].Timestamp)
	})

	chainExport := &types.ChainExport{
		TimeRange:   timeRange,
		Events:      events,
		TotalEvents: int64(len(events)),
		ExportedAt:  time.Now(),
		ExportedBy:  "system",
	}

	// Calculate chain integrity
	if len(events) > 0 {
		chainExport.FirstHash = events[0].Hash
		chainExport.LastHash = events[len(events)-1].Hash
	}

	return json.MarshalIndent(chainExport, "", "  ")
}

// ValidateEvent validates an event against the hash chain
func (h *pgAuditHashChain) ValidateEvent(ctx context.Context, eventID string) (*types.EventValidation, error) {
	event, err := h.eventStore.GetByID(ctx, eventID)
	if err != nil {
		return nil, fmt.Errorf("get event for validation: %w", err)
	}

	validation := &types.EventValidation{
		EventID:     eventID,
		EventHash:   event.Hash,
		IsValid:     true,
		ValidatedAt: time.Now(),
	}

	// Validate event hash
	expectedHash := h.calculateEventHash(event)
	if event.Hash != expectedHash {
		validation.IsValid = false
		validation.Errors = append(validation.Errors, fmt.Sprintf("hash_mismatch: Event hash does not match calculated hash (expected: %s, actual: %s)", expectedHash, event.Hash))
	}

	// Validate chain continuity
	if event.PrevHash != "" {
		// Get previous events to verify chain
		filter := &types.AuditFilter{
			EndTime: &event.Timestamp,
		}
		prevEvents, err := h.eventStore.Retrieve(ctx, filter)
		if err != nil {
			return nil, fmt.Errorf("get previous events: %w", err)
		}

		foundPrev := false
		for _, prevEvent := range prevEvents {
			if prevEvent.Hash == event.PrevHash {
				foundPrev = true
				break
			}
		}

		if !foundPrev {
			validation.IsValid = false
			validation.Errors = append(validation.Errors, fmt.Sprintf("chain_broken: Previous hash not found in chain (expected: %s)", event.PrevHash))
		}
	}

	return validation, nil
}

// Helper functions

func (h *pgAuditHashChain) calculateEventHash(event *types.AuditEvent) string {
	hasher := sha256.New()

	// Hash core event data
	hasher.Write([]byte(event.Action))
	hasher.Write([]byte(event.ObjectType))
	hasher.Write(event.ObjectID[:])
	hasher.Write([]byte(event.Timestamp.Format(time.RFC3339Nano)))

	if event.ActorID != nil {
		hasher.Write(event.ActorID[:])
	}
	if event.TenantID != nil {
		hasher.Write(event.TenantID[:])
	}

	// Hash structured data
	if event.Before != nil {
		beforeBytes, _ := json.Marshal(event.Before)
		hasher.Write(beforeBytes)
	}
	if event.After != nil {
		afterBytes, _ := json.Marshal(event.After)
		hasher.Write(afterBytes)
	}
	if event.Metadata != nil {
		metaBytes, _ := json.Marshal(event.Metadata)
		hasher.Write(metaBytes)
	}

	return hex.EncodeToString(hasher.Sum(nil))
}

func (h *pgAuditHashChain) getLatestHash(ctx context.Context) (string, error) {
	// Get the most recent event
	events, err := h.eventStore.Retrieve(ctx, &types.AuditFilter{
		Limit: 1,
	})
	if err != nil {
		return "", fmt.Errorf("get latest event: %w", err)
	}

	if len(events) == 0 {
		// This is the first event in the chain
		return "", nil
	}

	return events[0].Hash, nil
}



