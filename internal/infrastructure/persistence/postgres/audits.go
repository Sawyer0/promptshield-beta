package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/domain"
	"github.com/promptshield/promptshield/internal/shared/contracts"
	"github.com/promptshield/promptshield/internal/shared/types"
)

// Ensure pgAuditRepo implements domain.AuditRepository
var _ domain.AuditRepository = (*pgAuditRepo)(nil)

type pgAuditRepo struct{ db *Pool }

func AuditRepo(db *Pool) domain.AuditRepository { return &pgAuditRepo{db: db} }

func (r *pgAuditRepo) Create(ctx context.Context, entry *domain.AuditEntry) error {
	q := `INSERT INTO audits (id, tenant_id, actor_id, actor_type, actor_email, action, object_type, object_id, before_data, after_data, metadata, hash, prev_hash) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`

	_, err := r.db.Raw().Exec(ctx, q,
		entry.ID,
		entry.TenantID,
		entry.ActorID,
		entry.ActorType,
		entry.ActorEmail,
		entry.Action,
		entry.ObjectType,
		entry.ObjectID,
		entry.Before,
		entry.After,
		entry.Metadata,
		entry.Hash,
		entry.PrevHash,
	)
	if err != nil {
		return fmt.Errorf("create audit: %w", err)
	}
	return nil
}

func (r *pgAuditRepo) ListByTenant(ctx context.Context, tenantID uuid.UUID, offset, limit int) ([]*domain.AuditEntry, int, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}

	// Count total for pagination
	countQuery := `SELECT COUNT(*) FROM audits WHERE tenant_id = $1`
	var total int
	if err := r.db.Raw().QueryRow(ctx, countQuery, tenantID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count audits: %w", err)
	}

	q := `SELECT id, tenant_id, actor_id, actor_type, actor_email, action, object_type, object_id, before_data, after_data, metadata, created_at, hash, prev_hash
		FROM audits 
		WHERE tenant_id = $1 
		ORDER BY created_at DESC 
		OFFSET $2 LIMIT $3`

	rows, err := r.db.Raw().Query(ctx, q, tenantID, offset, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("list audits: %w", err)
	}
	defer rows.Close()

	var audits []*domain.AuditEntry
	for rows.Next() {
		var a domain.AuditEntry
		if err := rows.Scan(&a.ID, &a.TenantID, &a.ActorID, &a.ActorType, &a.ActorEmail,
			&a.Action, &a.ObjectType, &a.ObjectID, &a.Before, &a.After, &a.Metadata, &a.CreatedAt, &a.Hash, &a.PrevHash); err != nil {
			return nil, 0, fmt.Errorf("scan audit: %w", err)
		}
		audits = append(audits, &a)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return audits, total, nil
}

func (r *pgAuditRepo) Get(ctx context.Context, id uuid.UUID) (*domain.AuditEntry, error) {
	q := `SELECT id, tenant_id, actor_id, actor_type, actor_email, action, object_type, object_id, before_data, after_data, metadata, created_at, hash, prev_hash
		FROM audits WHERE id = $1`

	var a domain.AuditEntry
	err := r.db.Raw().QueryRow(ctx, q, id).Scan(&a.ID, &a.TenantID, &a.ActorID, &a.ActorType, &a.ActorEmail,
		&a.Action, &a.ObjectType, &a.ObjectID, &a.Before, &a.After, &a.Metadata, &a.CreatedAt, &a.Hash, &a.PrevHash)
	if err != nil {
		return nil, fmt.Errorf("get audit: %w", err)
	}
	return &a, nil
}

func (r *pgAuditRepo) ListByObject(ctx context.Context, objectType string, objectID uuid.UUID, offset, limit int) ([]*domain.AuditEntry, int, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}

	// Count total for pagination
	countQuery := `SELECT COUNT(*) FROM audits WHERE object_type = $1 AND object_id = $2`
	var total int
	if err := r.db.Raw().QueryRow(ctx, countQuery, objectType, objectID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count audits by object: %w", err)
	}

	q := `SELECT id, tenant_id, actor_id, actor_type, actor_email, action, object_type, object_id, before_data, after_data, metadata, created_at, hash, prev_hash
		FROM audits 
		WHERE object_type = $1 AND object_id = $2
		ORDER BY created_at DESC 
		OFFSET $3 LIMIT $4`

	rows, err := r.db.Raw().Query(ctx, q, objectType, objectID, offset, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("list audits by object: %w", err)
	}
	defer rows.Close()

	var audits []*domain.AuditEntry
	for rows.Next() {
		var a domain.AuditEntry
		if err := rows.Scan(&a.ID, &a.TenantID, &a.ActorID, &a.ActorType, &a.ActorEmail,
			&a.Action, &a.ObjectType, &a.ObjectID, &a.Before, &a.After, &a.Metadata, &a.CreatedAt, &a.Hash, &a.PrevHash); err != nil {
			return nil, 0, fmt.Errorf("scan audit: %w", err)
		}
		audits = append(audits, &a)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return audits, total, nil
}

func (r *pgAuditRepo) ListByAction(ctx context.Context, action string, offset, limit int) ([]*domain.AuditEntry, int, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}

	// Count total for pagination
	countQuery := `SELECT COUNT(*) FROM audits WHERE action = $1`
	var total int
	if err := r.db.Raw().QueryRow(ctx, countQuery, action).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count audits by action: %w", err)
	}

	q := `SELECT id, tenant_id, actor_id, actor_type, actor_email, action, object_type, object_id, before_data, after_data, metadata, created_at, hash, prev_hash
		FROM audits
		WHERE action = $1
		ORDER BY created_at DESC
		OFFSET $2 LIMIT $3`

	rows, err := r.db.Raw().Query(ctx, q, action, offset, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("list audits by action: %w", err)
	}
	defer rows.Close()

	var audits []*domain.AuditEntry
	for rows.Next() {
		var a domain.AuditEntry
		if err := rows.Scan(&a.ID, &a.TenantID, &a.ActorID, &a.ActorType, &a.ActorEmail,
			&a.Action, &a.ObjectType, &a.ObjectID, &a.Before, &a.After, &a.Metadata, &a.CreatedAt, &a.Hash, &a.PrevHash); err != nil {
			return nil, 0, fmt.Errorf("scan audit: %w", err)
		}
		audits = append(audits, &a)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return audits, total, nil
}

// pgAuditEventStore implements contracts.AuditEventStore with PostgreSQL backend
type pgAuditEventStore struct {
	db *Pool
}

// NewAuditEventStore creates a new PostgreSQL audit event store
func NewAuditEventStore(db *Pool) contracts.AuditEventStore {
	return &pgAuditEventStore{db: db}
}

// Store stores a single audit event
func (s *pgAuditEventStore) Store(ctx context.Context, event *types.AuditEvent) error {
	// Convert to domain model for persistence
	entry := &domain.AuditEntry{
		TenantID:   event.TenantID,
		ActorID:    event.ActorID,
		ActorType:  domain.ActorType(event.ActorType),
		ActorEmail: event.ActorEmail,
		Action:     event.Action,
		ObjectType: event.ObjectType,
		ObjectID:   event.ObjectID,
		Before:     toRawMessage(event.Before),
		After:      toRawMessage(event.After),
		Metadata:   toRawMessage(event.Metadata),
		CreatedAt:  event.Timestamp,
		Hash:       hashAuditEvent(event),
		PrevHash:   getPrevHash(ctx, event, s.db),
	}

	entry.ID = uuid.New()

	// Use existing Create method
	return AuditRepo(s.db).Create(ctx, entry)
}

// StoreBatch stores multiple audit events
func (s *pgAuditEventStore) StoreBatch(ctx context.Context, events []*types.AuditEvent) error {
	if len(events) == 0 {
		return nil
	}

	// Use transaction for batch insert
	tx, err := s.db.Raw().Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	for _, event := range events {
		entry := &domain.AuditEntry{
			TenantID:   event.TenantID,
			ActorID:    event.ActorID,
			ActorType:  domain.ActorType(event.ActorType),
			ActorEmail: event.ActorEmail,
			Action:     event.Action,
			ObjectType: event.ObjectType,
			ObjectID:   event.ObjectID,
			Before:     toRawMessage(event.Before),
			After:      toRawMessage(event.After),
			Metadata:   toRawMessage(event.Metadata),
			Hash:       hashAuditEvent(event),
			PrevHash:   getPrevHash(ctx, event, s.db),
		}
		entry.ID = uuid.New()

		q := `INSERT INTO audits (id, tenant_id, actor_id, actor_type, actor_email, action, object_type, object_id, before_data, after_data, metadata, hash, prev_hash)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`

		_, err := tx.Exec(ctx, q,
			entry.ID, entry.TenantID, entry.ActorID, entry.ActorType, entry.ActorEmail,
			entry.Action, entry.ObjectType, entry.ObjectID, entry.Before, entry.After,
			entry.Metadata, entry.Hash, entry.PrevHash)
		if err != nil {
			return fmt.Errorf("store audit event %s: %w", entry.ID, err)
		}
	}

	return tx.Commit(ctx)
}

// Retrieve retrieves audit events by filter
func (s *pgAuditEventStore) Retrieve(ctx context.Context, filter *types.AuditFilter) ([]*types.AuditEvent, error) {
	if filter == nil {
		filter = &types.AuditFilter{}
	}

	var conditions []string
	var args []interface{}
	argIdx := 1

	// Build WHERE clause
	if filter.TenantID != "" {
		conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argIdx))
		args = append(args, filter.TenantID)
		argIdx++
	}

	if filter.ActorID != "" {
		conditions = append(conditions, fmt.Sprintf("actor_id = $%d", argIdx))
		args = append(args, filter.ActorID)
		argIdx++
	}

	if filter.Action != "" {
		conditions = append(conditions, fmt.Sprintf("action = $%d", argIdx))
		args = append(args, filter.Action)
		argIdx++
	}

	if filter.ObjectType != "" {
		conditions = append(conditions, fmt.Sprintf("object_type = $%d", argIdx))
		args = append(args, filter.ObjectType)
		argIdx++
	}

	if filter.ObjectID != nil {
		conditions = append(conditions, fmt.Sprintf("object_id = $%d", argIdx))
		args = append(args, *filter.ObjectID)
		argIdx++
	}

	if filter.StartTime != nil && !filter.StartTime.IsZero() {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argIdx))
		args = append(args, *filter.StartTime)
		argIdx++
	}

	if filter.EndTime != nil && !filter.EndTime.IsZero() {
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", argIdx))
		args = append(args, *filter.EndTime)
		argIdx++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Build query with pagination
	limit := 100 // default
	if filter.Limit > 0 && filter.Limit <= 1000 {
		limit = filter.Limit
	}

	offset := 0
	if filter.Offset > 0 {
		offset = filter.Offset
	}

	query := fmt.Sprintf(`
		SELECT id, tenant_id, actor_id, actor_type, actor_email, action, object_type, object_id,
			   before_data, after_data, metadata, created_at, hash, prev_hash
		FROM audits
		%s
		ORDER BY created_at DESC
		OFFSET %d LIMIT %d`, whereClause, offset, limit)

	rows, err := s.db.Raw().Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("retrieve audit events: %w", err)
	}
	defer rows.Close()

	var events []*types.AuditEvent
	for rows.Next() {
		var event types.AuditEvent
		var before, after, metadata []byte
		var id uuid.UUID

		err := rows.Scan(&id, &event.TenantID, &event.ActorID, &event.ActorType, &event.ActorEmail,
			&event.Action, &event.ObjectType, &event.ObjectID, &before, &after, &metadata,
			&event.Timestamp, &event.Hash, &event.PrevHash)
		if err != nil {
			return nil, fmt.Errorf("scan audit event: %w", err)
		}

		// Parse JSON fields
		if len(before) > 0 {
			json.Unmarshal(before, &event.Before)
		}
		if len(after) > 0 {
			json.Unmarshal(after, &event.After)
		}
		if len(metadata) > 0 {
			json.Unmarshal(metadata, &event.Metadata)
		}

		events = append(events, &event)
	}

	return events, nil
}

// GetByID retrieves an audit event by ID
func (s *pgAuditEventStore) GetByID(ctx context.Context, eventID string) (*types.AuditEvent, error) {
	id, err := uuid.Parse(eventID)
	if err != nil {
		return nil, fmt.Errorf("invalid event ID: %w", err)
	}

	entry, err := AuditRepo(s.db).Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get audit event: %w", err)
	}

	// Convert to types.AuditEvent
	event := &types.AuditEvent{
		TenantID:   entry.TenantID,
		ActorID:    entry.ActorID,
		ActorType:  types.ActorType(entry.ActorType),
		ActorEmail: entry.ActorEmail,
		Action:     entry.Action,
		ObjectType: entry.ObjectType,
		ObjectID:   entry.ObjectID,
		Timestamp:  entry.CreatedAt,
		Hash:       entry.Hash,
		PrevHash:   entry.PrevHash,
	}

	// Parse JSON fields
	if len(entry.Before) > 0 {
		json.Unmarshal(entry.Before, &event.Before)
	}
	if len(entry.After) > 0 {
		json.Unmarshal(entry.After, &event.After)
	}
	if len(entry.Metadata) > 0 {
		json.Unmarshal(entry.Metadata, &event.Metadata)
	}

	return event, nil
}

// Count returns the count of audit events matching filter
func (s *pgAuditEventStore) Count(ctx context.Context, filter *types.AuditFilter) (int64, error) {
	if filter == nil {
		filter = &types.AuditFilter{}
	}

	var conditions []string
	var args []interface{}
	argIdx := 1

	// Build WHERE clause (same as Retrieve)
	if filter.TenantID != "" {
		conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argIdx))
		args = append(args, filter.TenantID)
		argIdx++
	}

	if filter.ActorID != "" {
		conditions = append(conditions, fmt.Sprintf("actor_id = $%d", argIdx))
		args = append(args, filter.ActorID)
		argIdx++
	}

	if filter.Action != "" {
		conditions = append(conditions, fmt.Sprintf("action = $%d", argIdx))
		args = append(args, filter.Action)
		argIdx++
	}

	if filter.ObjectType != "" {
		conditions = append(conditions, fmt.Sprintf("object_type = $%d", argIdx))
		args = append(args, filter.ObjectType)
		argIdx++
	}

	if filter.ObjectID != nil {
		conditions = append(conditions, fmt.Sprintf("object_id = $%d", argIdx))
		args = append(args, *filter.ObjectID)
		argIdx++
	}

	if filter.StartTime != nil && !filter.StartTime.IsZero() {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argIdx))
		args = append(args, *filter.StartTime)
		argIdx++
	}

	if filter.EndTime != nil && !filter.EndTime.IsZero() {
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", argIdx))
		args = append(args, *filter.EndTime)
		argIdx++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	query := fmt.Sprintf("SELECT COUNT(*) FROM audits %s", whereClause)

	var count int64
	err := s.db.Raw().QueryRow(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count audit events: %w", err)
	}

	return count, nil
}

// Delete deletes audit events matching filter (for compliance retention)
func (s *pgAuditEventStore) Delete(ctx context.Context, filter *types.AuditFilter) error {
	if filter == nil {
		return fmt.Errorf("filter cannot be nil for delete operation")
	}

	var conditions []string
	var args []interface{}
	argIdx := 1

	// Build WHERE clause
	if filter.TenantID != "" {
		conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argIdx))
		args = append(args, filter.TenantID)
		argIdx++
	}

	if filter.ActorID != "" {
		conditions = append(conditions, fmt.Sprintf("actor_id = $%d", argIdx))
		args = append(args, filter.ActorID)
		argIdx++
	}

	if filter.Action != "" {
		conditions = append(conditions, fmt.Sprintf("action = $%d", argIdx))
		args = append(args, filter.Action)
		argIdx++
	}

	if filter.ObjectType != "" {
		conditions = append(conditions, fmt.Sprintf("object_type = $%d", argIdx))
		args = append(args, filter.ObjectType)
		argIdx++
	}

	if filter.ObjectID != nil {
		conditions = append(conditions, fmt.Sprintf("object_id = $%d", argIdx))
		args = append(args, *filter.ObjectID)
		argIdx++
	}

	if filter.StartTime != nil && !filter.StartTime.IsZero() {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argIdx))
		args = append(args, *filter.StartTime)
		argIdx++
	}

	if filter.EndTime != nil && !filter.EndTime.IsZero() {
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", argIdx))
		args = append(args, *filter.EndTime)
		argIdx++
	}

	if len(conditions) == 0 {
		return fmt.Errorf("delete operation requires at least one filter condition")
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")
	query := fmt.Sprintf("DELETE FROM audits %s", whereClause)

	result, err := s.db.Raw().Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete audit events: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("no audit events matched the delete filter")
	}

	return nil
}

// Archive archives old audit events
func (s *pgAuditEventStore) Archive(ctx context.Context, olderThan time.Time) error {
	// Move old events to audit_archive table
	archiveQuery := `
		INSERT INTO audit_archive (id, tenant_id, actor_id, actor_type, actor_email, action, object_type, object_id, before_data, after_data, metadata, created_at, hash, prev_hash, archived_at)
		SELECT id, tenant_id, actor_id, actor_type, actor_email, action, object_type, object_id, before_data, after_data, metadata, created_at, hash, prev_hash, $1
		FROM audits
		WHERE created_at < $2`

	_, err := s.db.Raw().Exec(ctx, archiveQuery, time.Now(), olderThan)
	if err != nil {
		return fmt.Errorf("archive audit events: %w", err)
	}

	// Delete archived events from main table
	deleteQuery := "DELETE FROM audits WHERE created_at < $1"
	_, err = s.db.Raw().Exec(ctx, deleteQuery, olderThan)
	if err != nil {
		return fmt.Errorf("delete archived audit events: %w", err)
	}

	return nil
}

// Verify verifies audit log integrity
func (s *pgAuditEventStore) Verify(ctx context.Context, timeRange types.TimeRange) (*types.AuditVerification, error) {
	// Get events in time range
	filter := &types.AuditFilter{
		StartTime: &timeRange.Start,
		EndTime:   &timeRange.End,
	}

	events, err := s.Retrieve(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("retrieve events for verification: %w", err)
	}

	verification := &types.AuditVerification{
		TimeRange:      timeRange,
		TotalEvents:    int64(len(events)),
		VerifiedAt:     time.Now(),
		HashChainValid: true,
	}

	// Verify hash chain
	var prevHash string
	var validEvents, invalidEvents int64

	for _, event := range events {
		// Verify current event's hash
		expectedHash := hashAuditEvent(event)
		if event.Hash != expectedHash {
			invalidEvents++
			verification.InvalidEvents++
			verification.CorruptedEvents = append(verification.CorruptedEvents, eventID(event))
			verification.HashChainValid = false
			continue
		}

		// Verify chain continuity
		if event.PrevHash != prevHash {
			invalidEvents++
			verification.InvalidEvents++
			verification.MissingEvents = append(verification.MissingEvents, eventID(event))
			verification.HashChainValid = false
			continue
		}

		validEvents++
		prevHash = event.Hash
	}

	verification.ValidEvents = validEvents
	verification.InvalidEvents = invalidEvents

	return verification, nil
}

// Helper functions

func toRawMessage(data interface{}) []byte {
	if data == nil {
		return nil
	}
	jsonData, _ := json.Marshal(data)
	return jsonData
}

func hashAuditEvent(event *types.AuditEvent) string {
	h := sha256.New()

	// Hash key components
	h.Write([]byte(event.Action))
	h.Write([]byte(event.ObjectType))
	h.Write(event.ObjectID[:])
	h.Write([]byte(event.Timestamp.String()))

	if event.ActorID != nil {
		h.Write(event.ActorID[:])
	}
	if event.TenantID != nil {
		h.Write(event.TenantID[:])
	}

	// Hash data fields
	if event.Before != nil {
		beforeData, _ := json.Marshal(event.Before)
		h.Write(beforeData)
	}
	if event.After != nil {
		afterData, _ := json.Marshal(event.After)
		h.Write(afterData)
	}
	if event.Metadata != nil {
		metaData, _ := json.Marshal(event.Metadata)
		h.Write(metaData)
	}

	return hex.EncodeToString(h.Sum(nil))
}

func getPrevHash(ctx context.Context, event *types.AuditEvent, db *Pool) string {
	// Get the most recent event before this one
	query := `
		SELECT hash FROM audits
		WHERE created_at <= $1
		ORDER BY created_at DESC, id DESC
		LIMIT 1`

	var prevHash string
	db.Raw().QueryRow(ctx, query, event.Timestamp).Scan(&prevHash)
	return prevHash
}

func eventID(event *types.AuditEvent) string {
	if event == nil {
		return ""
	}
	return event.ObjectID.String()
}
