package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/infrastructure/persistence/postgres"
)

// ServiceStatus represents the current state of a service
type ServiceStatus string

const (
	ServiceStatusStopped  ServiceStatus = "stopped"
	ServiceStatusStarting ServiceStatus = "starting"
	ServiceStatusRunning  ServiceStatus = "running"
	ServiceStatusStopping ServiceStatus = "stopping"
	ServiceStatusError    ServiceStatus = "error"
)

// ServiceConfig represents a tenant's service configuration
type ServiceConfig struct {
	ID              uuid.UUID         `json:"id"`
	TenantID        uuid.UUID         `json:"tenant_id"`
	ServiceName     string            `json:"service_name"`
	Status          ServiceStatus     `json:"status"`
	Enabled         bool              `json:"enabled"`
	Config          map[string]interface{} `json:"config"`
	Resources       ResourceLimits    `json:"resources"`
	LastStarted     *time.Time        `json:"last_started,omitempty"`
	LastStopped     *time.Time        `json:"last_stopped,omitempty"`
	ErrorMessage    string            `json:"error_message,omitempty"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

// ResourceLimits defines resource constraints per tenant
type ResourceLimits struct {
	MaxInstances  int    `json:"max_instances"`
	CPULimit      string `json:"cpu_limit"`      // e.g., "500m" for 0.5 CPU
	MemoryLimit   string `json:"memory_limit"`   // e.g., "512Mi"
	RequestsPerMin int    `json:"requests_per_min"`
}

// ServiceControlHandlers manages service lifecycle operations
type ServiceControlHandlers struct {
	db             postgres.DB
	serviceManager ServiceManager
	events         *EventHub
}

// ServiceManager interface for controlling services
type ServiceManager interface {
	StartService(ctx context.Context, tenantID uuid.UUID, config ServiceConfig) error
	StopService(ctx context.Context, tenantID uuid.UUID, serviceName string) error
	GetServiceStatus(ctx context.Context, tenantID uuid.UUID, serviceName string) (ServiceStatus, error)
	RestartService(ctx context.Context, tenantID uuid.UUID, serviceName string) error
	ScaleService(ctx context.Context, tenantID uuid.UUID, serviceName string, replicas int) error
}

// NewServiceControlHandlers creates new service control handlers
func NewServiceControlHandlers(db postgres.DB, manager ServiceManager, events *EventHub) *ServiceControlHandlers {
	return &ServiceControlHandlers{
		db:             db,
		serviceManager: manager,
		events:         events,
	}
}

// RegisterServiceRoutes registers all service control routes
func (h *ServiceControlHandlers) RegisterServiceRoutes(r chi.Router, opt Options) {
	r.Route("/api/v1/services", func(r chi.Router) {
		// Tenant validation (auth handled by frontend)
		if opt.DB != nil {
			r.Use(tenantValidationMiddleware(opt.DB))
		}
		
		r.Get("/", h.ListServices)           // GET /api/v1/services
		r.Post("/", h.CreateService)         // POST /api/v1/services
		r.Get("/{serviceId}", h.GetService)  // GET /api/v1/services/{id}
		r.Put("/{serviceId}", h.UpdateService) // PUT /api/v1/services/{id}
		r.Delete("/{serviceId}", h.DeleteService) // DELETE /api/v1/services/{id}
		
		// Service control endpoints
		r.Post("/{serviceId}/start", h.StartService)   // Start a service
		r.Post("/{serviceId}/stop", h.StopService)     // Stop a service
		r.Post("/{serviceId}/restart", h.RestartService) // Restart a service
		r.Post("/{serviceId}/scale", h.ScaleService)   // Scale a service
		r.Get("/{serviceId}/status", h.GetServiceStatus) // Get service status
		r.Get("/{serviceId}/logs", h.GetServiceLogs)   // Stream service logs
		r.Get("/{serviceId}/metrics", h.GetServiceMetrics) // Get service metrics
	})
}

// ListServices returns all services for a tenant
func (h *ServiceControlHandlers) ListServices(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := GetTenantID(r.Context())
	if !ok {
		writeError(w, http.StatusBadRequest, "MISSING_TENANT", "Tenant ID is required", nil)
		return
	}
	
	query := `
		SELECT id, service_name, status, enabled, config, 
		       last_started, last_stopped, error_message, created_at, updated_at
		FROM tenant_services
		WHERE tenant_id = $1 AND deleted_at IS NULL
		ORDER BY service_name
	`
	
	rows, err := h.db.QueryContext(r.Context(), query, tenantID)
	if err != nil {
		slog.Error("Failed to list services", "error", err, "tenant_id", tenantID)
		writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to list services", nil)
		return
	}
	defer rows.Close()
	
	services := []ServiceConfig{}
	for rows.Next() {
		var s ServiceConfig
		var configJSON []byte
		s.TenantID = tenantID
		
		err := rows.Scan(
			&s.ID, &s.ServiceName, &s.Status, &s.Enabled,
			&configJSON, &s.LastStarted, &s.LastStopped,
			&s.ErrorMessage, &s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			slog.Error("Failed to scan service", "error", err)
			continue
		}
		
		if configJSON != nil {
			json.Unmarshal(configJSON, &s.Config)
		}
		
		services = append(services, s)
	}
	
	writeJSON(w, http.StatusOK, services, r)
}

// StartService starts a stopped service
func (h *ServiceControlHandlers) StartService(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := GetTenantID(r.Context())
	serviceID := chi.URLParam(r, "serviceId")
	
	// Get service configuration
	service, err := h.getServiceConfig(r.Context(), tenantID, serviceID)
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "SERVICE_NOT_FOUND", "Service not found", nil)
		} else {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to get service", nil)
		}
		return
	}
	
	// Check if service is already running
	if service.Status == ServiceStatusRunning {
		writeError(w, http.StatusBadRequest, "SERVICE_ALREADY_RUNNING", "Service is already running", nil)
		return
	}
	
	// Check tenant's resource limits
	if !h.checkResourceLimits(r.Context(), tenantID, service.Resources) {
		writeError(w, http.StatusForbidden, "RESOURCE_LIMIT_EXCEEDED", "Resource limits exceeded for your plan", nil)
		return
	}
	
	// Update status to starting
	h.updateServiceStatus(r.Context(), service.ID, ServiceStatusStarting, "")
	
	// Start the service asynchronously
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		
		err := h.serviceManager.StartService(ctx, tenantID, *service)
		if err != nil {
			h.updateServiceStatus(r.Context(), service.ID, ServiceStatusError, err.Error())
			h.broadcastServiceEvent(tenantID, service.ID, "service.start.failed", map[string]interface{}{
				"service_id": service.ID,
				"error":      err.Error(),
			})
		} else {
			now := time.Now()
			h.updateServiceStatusWithTime(r.Context(), service.ID, ServiceStatusRunning, "", &now, nil)
			h.broadcastServiceEvent(tenantID, service.ID, "service.started", map[string]interface{}{
				"service_id": service.ID,
				"started_at": now,
			})
		}
	}()
	
	writeJSON(w, http.StatusAccepted, map[string]interface{}{
		"message": "Service is starting",
		"service_id": service.ID,
		"status": ServiceStatusStarting,
	}, r)
}

// StopService stops a running service
func (h *ServiceControlHandlers) StopService(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := GetTenantID(r.Context())
	serviceID := chi.URLParam(r, "serviceId")
	
	service, err := h.getServiceConfig(r.Context(), tenantID, serviceID)
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "SERVICE_NOT_FOUND", "Service not found", nil)
		} else {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to get service", nil)
		}
		return
	}
	
	if service.Status == ServiceStatusStopped {
		writeError(w, http.StatusBadRequest, "SERVICE_ALREADY_STOPPED", "Service is already stopped", nil)
		return
	}
	
	// Update status to stopping
	h.updateServiceStatus(r.Context(), service.ID, ServiceStatusStopping, "")
	
	// Stop the service asynchronously
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		
		err := h.serviceManager.StopService(ctx, tenantID, service.ServiceName)
		if err != nil {
			h.updateServiceStatus(r.Context(), service.ID, ServiceStatusError, err.Error())
			h.broadcastServiceEvent(tenantID, service.ID, "service.stop.failed", map[string]interface{}{
				"service_id": service.ID,
				"error":      err.Error(),
			})
		} else {
			now := time.Now()
			h.updateServiceStatusWithTime(r.Context(), service.ID, ServiceStatusStopped, "", nil, &now)
			h.broadcastServiceEvent(tenantID, service.ID, "service.stopped", map[string]interface{}{
				"service_id": service.ID,
				"stopped_at": now,
			})
		}
	}()
	
	writeJSON(w, http.StatusAccepted, map[string]interface{}{
		"message": "Service is stopping",
		"service_id": service.ID,
		"status": ServiceStatusStopping,
	}, r)
}

// RestartService restarts a service
func (h *ServiceControlHandlers) RestartService(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := GetTenantID(r.Context())
	serviceID := chi.URLParam(r, "serviceId")
	
	service, err := h.getServiceConfig(r.Context(), tenantID, serviceID)
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "SERVICE_NOT_FOUND", "Service not found", nil)
		} else {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to get service", nil)
		}
		return
	}
	
	// Restart asynchronously
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		
		err := h.serviceManager.RestartService(ctx, tenantID, service.ServiceName)
		if err != nil {
			h.updateServiceStatus(r.Context(), service.ID, ServiceStatusError, err.Error())
			h.broadcastServiceEvent(tenantID, service.ID, "service.restart.failed", map[string]interface{}{
				"service_id": service.ID,
				"error":      err.Error(),
			})
		} else {
			now := time.Now()
			h.updateServiceStatusWithTime(r.Context(), service.ID, ServiceStatusRunning, "", &now, nil)
			h.broadcastServiceEvent(tenantID, service.ID, "service.restarted", map[string]interface{}{
				"service_id": service.ID,
				"restarted_at": now,
			})
		}
	}()
	
	writeJSON(w, http.StatusAccepted, map[string]interface{}{
		"message": "Service is restarting",
		"service_id": service.ID,
	}, r)
}

// GetServiceStatus returns the current status of a service
func (h *ServiceControlHandlers) GetServiceStatus(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := GetTenantID(r.Context())
	serviceID := chi.URLParam(r, "serviceId")
	
	service, err := h.getServiceConfig(r.Context(), tenantID, serviceID)
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "SERVICE_NOT_FOUND", "Service not found", nil)
		} else {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to get service", nil)
		}
		return
	}
	
	// Get real-time status from service manager
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	liveStatus, err := h.serviceManager.GetServiceStatus(ctx, tenantID, service.ServiceName)
	if err != nil {
		slog.Warn("Failed to get live status", "error", err, "service_id", serviceID)
		liveStatus = service.Status // Fall back to DB status
	}
	
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"service_id": service.ID,
		"service_name": service.ServiceName,
		"status": liveStatus,
		"enabled": service.Enabled,
		"last_started": service.LastStarted,
		"last_stopped": service.LastStopped,
		"error_message": service.ErrorMessage,
		"updated_at": service.UpdatedAt,
	}, r)
}

// Helper methods

func (h *ServiceControlHandlers) getServiceConfig(ctx context.Context, tenantID uuid.UUID, serviceID string) (*ServiceConfig, error) {
	sid, err := uuid.Parse(serviceID)
	if err != nil {
		return nil, err
	}
	
	query := `
		SELECT id, tenant_id, service_name, status, enabled, config,
		       last_started, last_stopped, error_message, created_at, updated_at
		FROM tenant_services
		WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL
	`
	
	var s ServiceConfig
	var configJSON []byte
	
	err = h.db.QueryRowContext(ctx, query, sid, tenantID).Scan(
		&s.ID, &s.TenantID, &s.ServiceName, &s.Status, &s.Enabled,
		&configJSON, &s.LastStarted, &s.LastStopped,
		&s.ErrorMessage, &s.CreatedAt, &s.UpdatedAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	if configJSON != nil {
		json.Unmarshal(configJSON, &s.Config)
	}
	
	// Load resource limits from subscription
	s.Resources = h.getTenantResourceLimits(ctx, tenantID)
	
	return &s, nil
}

func (h *ServiceControlHandlers) updateServiceStatus(ctx context.Context, serviceID uuid.UUID, status ServiceStatus, errorMsg string) {
	query := `
		UPDATE tenant_services
		SET status = $2, error_message = $3, updated_at = NOW()
		WHERE id = $1
	`
	
	if _, err := h.db.ExecContext(ctx, query, serviceID, status, errorMsg); err != nil {
		slog.Error("Failed to update service status", "error", err, "service_id", serviceID)
	}
}

func (h *ServiceControlHandlers) updateServiceStatusWithTime(ctx context.Context, serviceID uuid.UUID, status ServiceStatus, errorMsg string, startTime, stopTime *time.Time) {
	query := `
		UPDATE tenant_services
		SET status = $2, error_message = $3, last_started = $4, last_stopped = $5, updated_at = NOW()
		WHERE id = $1
	`
	
	if _, err := h.db.ExecContext(ctx, query, serviceID, status, errorMsg, startTime, stopTime); err != nil {
		slog.Error("Failed to update service status with time", "error", err, "service_id", serviceID)
	}
}

func (h *ServiceControlHandlers) checkResourceLimits(ctx context.Context, tenantID uuid.UUID, limits ResourceLimits) bool {
	// Check tenant's plan limits
	query := `
		SELECT sp.limits
		FROM subscriptions s
		JOIN subscription_plans sp ON s.plan_id = sp.id
		WHERE s.tenant_id = $1 AND s.status IN ('active', 'trial')
	`
	
	var planLimits map[string]interface{}
	var limitsJSON []byte
	
	err := h.db.QueryRowContext(ctx, query, tenantID).Scan(&limitsJSON)
	if err != nil {
		slog.Error("Failed to get tenant limits", "error", err, "tenant_id", tenantID)
		return false
	}
	
	json.Unmarshal(limitsJSON, &planLimits)
	
	// Compare requested resources against plan limits
	// This is a simplified check - enhance based on your needs
	if maxInstances, ok := planLimits["max_instances"].(float64); ok {
		if limits.MaxInstances > int(maxInstances) {
			return false
		}
	}
	
	return true
}

func (h *ServiceControlHandlers) getTenantResourceLimits(ctx context.Context, tenantID uuid.UUID) ResourceLimits {
	// Get limits based on tenant's subscription plan
	query := `
		SELECT sp.limits
		FROM subscriptions s
		JOIN subscription_plans sp ON s.plan_id = sp.id
		WHERE s.tenant_id = $1 AND s.status IN ('active', 'trial')
	`
	
	var limitsJSON []byte
	var planLimits map[string]interface{}
	
	err := h.db.QueryRowContext(ctx, query, tenantID).Scan(&limitsJSON)
	if err == nil && limitsJSON != nil {
		json.Unmarshal(limitsJSON, &planLimits)
	}
	
	// Set defaults based on plan
	limits := ResourceLimits{
		MaxInstances:  1,
		CPULimit:      "500m",
		MemoryLimit:   "512Mi",
		RequestsPerMin: 100,
	}
	
	if maxInstances, ok := planLimits["max_instances"].(float64); ok {
		limits.MaxInstances = int(maxInstances)
	}
	if cpuLimit, ok := planLimits["cpu_limit"].(string); ok {
		limits.CPULimit = cpuLimit
	}
	if memLimit, ok := planLimits["memory_limit"].(string); ok {
		limits.MemoryLimit = memLimit
	}
	if rpm, ok := planLimits["requests_per_min"].(float64); ok {
		limits.RequestsPerMin = int(rpm)
	}
	
	return limits
}

func (h *ServiceControlHandlers) broadcastServiceEvent(tenantID, serviceID uuid.UUID, eventType string, data map[string]interface{}) {
	if h.events != nil {
		h.events.Publish(Event{
			Type:      eventType,
			TenantID:  tenantID.String(),
			ServiceID: serviceID.String(),
			Data:      data,
			Time:      time.Now(),
			Timestamp: time.Now(),
		})
	}
}

// Stub implementations for remaining handlers
func (h *ServiceControlHandlers) CreateService(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusNotImplemented, map[string]string{"message": "Not implemented"}, r)
}

func (h *ServiceControlHandlers) GetService(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusNotImplemented, map[string]string{"message": "Not implemented"}, r)
}

func (h *ServiceControlHandlers) UpdateService(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusNotImplemented, map[string]string{"message": "Not implemented"}, r)
}

func (h *ServiceControlHandlers) DeleteService(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusNotImplemented, map[string]string{"message": "Not implemented"}, r)
}

func (h *ServiceControlHandlers) ScaleService(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusNotImplemented, map[string]string{"message": "Not implemented"}, r)
}

func (h *ServiceControlHandlers) GetServiceLogs(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusNotImplemented, map[string]string{"message": "Not implemented"}, r)
}

func (h *ServiceControlHandlers) GetServiceMetrics(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusNotImplemented, map[string]string{"message": "Not implemented"}, r)
}

// writeJSON helper is defined in response.go