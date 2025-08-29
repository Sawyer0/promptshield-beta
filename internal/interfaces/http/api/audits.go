package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/domain"
)

// registerAuditHandlers registers all audit trail endpoints
func registerAuditHandlers(r chi.Router, opt Options) {
	r.Route("/admin/audits", func(ar chi.Router) {
		ar.Use(adminAuth(opt))
		
		ar.Get("/", listAuditEventsHandler(opt))
		ar.Post("/search", searchAuditEventsHandler(opt))
		ar.Get("/export", exportAuditEventsHandler(opt))
		ar.Get("/object/{type}/{objectId}", getAuditEventsByObjectHandler(opt))
	})
}

// AuditSearchRequest represents the search criteria for audit events
type AuditSearchRequest struct {
	TenantID    *uuid.UUID `json:"tenant_id,omitempty"`
	ActorID     *uuid.UUID `json:"actor_id,omitempty"`
	ActorEmail  string     `json:"actor_email,omitempty"`
	Actions     []string   `json:"actions,omitempty"`
	ObjectTypes []string   `json:"object_types,omitempty"`
	ObjectID    *uuid.UUID `json:"object_id,omitempty"`
	StartTime   *time.Time `json:"start_time,omitempty"`
	EndTime     *time.Time `json:"end_time,omitempty"`
	Limit       int        `json:"limit,omitempty"`
	Offset      int        `json:"offset,omitempty"`
}

// AuditSearchResponse represents paginated audit results
type AuditSearchResponse struct {
	Events     []*domain.AuditEntry `json:"events"`
	Count      int           `json:"count"`
	TotalCount int           `json:"total_count,omitempty"`
	HasMore    bool          `json:"has_more"`
	Pagination *Pagination   `json:"pagination"`
}

// Pagination represents pagination information
type Pagination struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
	Total  int `json:"total,omitempty"`
}

// listAuditEventsHandler handles GET /v1/admin/audits
func listAuditEventsHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if opt.AuditRepository == nil {
			writeErrorJSON(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", 
				"Audit management not configured", nil, r)
			return
		}
		
		// Parse query parameters
		query := r.URL.Query()
		
		// Tenant filtering
		var tenantID uuid.UUID
		if tenantIDStr := query.Get("tenant"); tenantIDStr != "" {
			var err error
			tenantID, err = uuid.Parse(tenantIDStr)
			if err != nil {
				writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT", 
					"Invalid tenant ID format", map[string]interface{}{"tenant_id": tenantIDStr}, r)
				return
			}
		}
		
		// Pagination
		limit := 100 // Default limit
		if limitStr := query.Get("limit"); limitStr != "" {
			if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
				limit = l
			}
		}
		
		// Get events
		var events []*domain.AuditEntry
		var err error
		
		if tenantID != uuid.Nil {
			events, _, err = opt.AuditRepository.ListByTenant(r.Context(), tenantID, 0, limit)
		} else {
			// For admin users, show all events with a system tenant ID
			// In practice, you'd want more sophisticated filtering here
			events = []*domain.AuditEntry{} // Empty for now when no tenant specified
		}
		
		if err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, "INTERNAL_ERROR", 
				"Failed to retrieve audit events", map[string]interface{}{"error": err.Error()}, r)
			return
		}
		
		if events == nil {
			events = []*domain.AuditEntry{}
		}
		
		// Build response
		response := &AuditSearchResponse{
			Events:  events,
			Count:   len(events),
			HasMore: len(events) == limit, // Simple heuristic
			Pagination: &Pagination{
				Limit:  limit,
				Offset: 0,
			},
		}
		
		writeJSON(w, http.StatusOK, response, r)
	}
}

// searchAuditEventsHandler handles POST /v1/admin/audits/search
func searchAuditEventsHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if opt.AuditRepository == nil {
			writeErrorJSON(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", 
				"Audit management not configured", nil, r)
			return
		}
		
		var req AuditSearchRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErrorJSON(w, http.StatusBadRequest, "INVALID_REQUEST", 
				"Invalid request body", map[string]interface{}{"error": err.Error()}, r)
			return
		}
		
		// Set defaults
		if req.Limit <= 0 || req.Limit > 1000 {
			req.Limit = 100
		}
		if req.Offset < 0 {
			req.Offset = 0
		}
		
		// Validate date range
		if req.StartTime != nil && req.EndTime != nil {
			if req.EndTime.Before(*req.StartTime) {
				writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT", 
					"End time must be after start time", nil, r)
				return
			}
		}
		
		// For now, implement basic search using existing repository methods
		var events []*domain.AuditEntry
		var err error
		
		if req.TenantID != nil {
			events, _, err = opt.AuditRepository.ListByTenant(r.Context(), *req.TenantID, 0, req.Limit)
		} else if req.ObjectID != nil && len(req.ObjectTypes) > 0 {
			// Search by object
			events, _, err = opt.AuditRepository.ListByObject(r.Context(), req.ObjectTypes[0], *req.ObjectID, 0, req.Limit)
		} else {
			// No specific criteria - return empty for security
			events = []*domain.AuditEntry{}
		}
		
		if err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, "INTERNAL_ERROR", 
				"Failed to search audit events", map[string]interface{}{"error": err.Error()}, r)
			return
		}
		
		if events == nil {
			events = []*domain.AuditEntry{}
		}
		
		// Apply additional filters in memory (in production, this should be done at DB level)
		filteredEvents := filterAuditEvents(events, &req)
		
		response := &AuditSearchResponse{
			Events:  filteredEvents,
			Count:   len(filteredEvents),
			HasMore: len(filteredEvents) == req.Limit,
			Pagination: &Pagination{
				Limit:  req.Limit,
				Offset: req.Offset,
			},
		}
		
		writeJSON(w, http.StatusOK, response, r)
	}
}

// exportAuditEventsHandler handles GET /v1/admin/audits/export
func exportAuditEventsHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if opt.AuditRepository == nil {
			writeErrorJSON(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", 
				"Audit management not configured", nil, r)
			return
		}
		
		// Parse query parameters
		query := r.URL.Query()
		format := query.Get("format")
		if format == "" {
			format = "json"
		}
		
		// Validate format
		if format != "json" && format != "csv" {
			writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT", 
				"Unsupported export format", map[string]interface{}{"format": format, "supported": []string{"json", "csv"}}, r)
			return
		}
		
		// Get tenant ID
		var tenantID uuid.UUID
		if tenantIDStr := query.Get("tenant"); tenantIDStr != "" {
			var err error
			tenantID, err = uuid.Parse(tenantIDStr)
			if err != nil {
				writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT", 
					"Invalid tenant ID format", map[string]interface{}{"tenant_id": tenantIDStr}, r)
				return
			}
		}
		
		// Get events (larger limit for export)
		limit := 10000
		if limitStr := query.Get("limit"); limitStr != "" {
			if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 50000 {
				limit = l
			}
		}
		
		var events []*domain.AuditEntry
		var err error
		
		if tenantID != uuid.Nil {
			events, _, err = opt.AuditRepository.ListByTenant(r.Context(), tenantID, 0, limit)
		} else {
			events = []*domain.AuditEntry{} // Empty for security when no tenant specified
		}
		
		if err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, "INTERNAL_ERROR", 
				"Failed to export audit events", map[string]interface{}{"error": err.Error()}, r)
			return
		}
		
		if events == nil {
			events = []*domain.AuditEntry{}
		}
		
		// Set appropriate headers
		timestamp := time.Now().Format("20060102_150405")
		filename := "audit_export_" + timestamp + "." + format
		
		w.Header().Set("Content-Disposition", "attachment; filename="+filename)
		w.Header().Set("X-PS-Export-Count", strconv.Itoa(len(events)))
		
		if format == "json" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"export_info": map[string]interface{}{
					"timestamp":    time.Now().UTC().Format(time.RFC3339),
					"format":       format,
					"event_count":  len(events),
					"tenant_id":    tenantID.String(),
				},
				"events": events,
			})
		} else {
			// CSV export
			w.Header().Set("Content-Type", "text/csv")
			
			// Write CSV header
			_, _ = w.Write([]byte("timestamp,action,object_type,object_id,actor_email,tenant_id\n"))
			
			// Write events
			for _, event := range events {
				tenantIDStr := ""
				if event.TenantID != nil {
					tenantIDStr = event.TenantID.String()
				}
				
				line := event.CreatedAt.Format(time.RFC3339) + "," +
					event.Action + "," +
					event.ObjectType + "," +
					event.ObjectID.String() + "," +
					event.ActorEmail + "," +
					tenantIDStr + "\n"
				
				_, _ = w.Write([]byte(line))
			}
		}
	}
}

// getAuditEventsByObjectHandler handles GET /v1/admin/audits/object/{type}/{objectId}
func getAuditEventsByObjectHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if opt.AuditRepository == nil {
			writeErrorJSON(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", 
				"Audit management not configured", nil, r)
			return
		}
		
		objectType := chi.URLParam(r, "type")
		objectIDStr := chi.URLParam(r, "objectId")
		
		if objectType == "" {
			writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT", 
				"Object type is required", nil, r)
			return
		}
		
		objectID, err := uuid.Parse(objectIDStr)
		if err != nil {
			writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT", 
				"Invalid object ID format", map[string]interface{}{"object_id": objectIDStr}, r)
			return
		}
		
		// Get limit from query params
		limit := 100
		if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
			if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
				limit = l
			}
		}
		
		// Get events for the object
		events, _, err := opt.AuditRepository.ListByObject(r.Context(), objectType, objectID, 0, limit)
		if err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, "INTERNAL_ERROR", 
				"Failed to retrieve audit events for object", map[string]interface{}{"error": err.Error()}, r)
			return
		}
		
		if events == nil {
			events = []*domain.AuditEntry{}
		}
		
		response := &AuditSearchResponse{
			Events:  events,
			Count:   len(events),
			HasMore: len(events) == limit,
			Pagination: &Pagination{
				Limit:  limit,
				Offset: 0,
			},
		}
		
		writeJSON(w, http.StatusOK, response, r)
	}
}

// filterAuditEvents applies in-memory filtering to audit events
func filterAuditEvents(events []*domain.AuditEntry, req *AuditSearchRequest) []*domain.AuditEntry {
	var filtered []*domain.AuditEntry
	
	for _, event := range events {
		// Skip if doesn't match actor filters
		if req.ActorID != nil && (event.ActorID == nil || *event.ActorID != *req.ActorID) {
			continue
		}
		
		if req.ActorEmail != "" && !strings.Contains(strings.ToLower(event.ActorEmail), strings.ToLower(req.ActorEmail)) {
			continue
		}
		
		// Skip if doesn't match action filters
		if len(req.Actions) > 0 {
			matchAction := false
			for _, action := range req.Actions {
				if strings.EqualFold(event.Action, action) {
					matchAction = true
					break
				}
			}
			if !matchAction {
				continue
			}
		}
		
		// Skip if doesn't match object type filters
		if len(req.ObjectTypes) > 0 {
			matchObjectType := false
			for _, objType := range req.ObjectTypes {
				if strings.EqualFold(event.ObjectType, objType) {
					matchObjectType = true
					break
				}
			}
			if !matchObjectType {
				continue
			}
		}
		
		// Skip if doesn't match time range
		if req.StartTime != nil && event.CreatedAt.Before(*req.StartTime) {
			continue
		}
		
		if req.EndTime != nil && event.CreatedAt.After(*req.EndTime) {
			continue
		}
		
		filtered = append(filtered, event)
	}
	
	// Apply offset and limit
	start := req.Offset
	if start > len(filtered) {
		return []*domain.AuditEntry{}
	}
	
	end := start + req.Limit
	if end > len(filtered) {
		end = len(filtered)
	}
	
	return filtered[start:end]
}