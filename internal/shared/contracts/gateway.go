package contracts

import (
	"context"
	"net/http"
	"time"

	"github.com/promptshield/promptshield/internal/shared/types"
)

// GatewayServer defines the interface for the main gateway server
type GatewayServer interface {
	// Starts the gateway server
	Start(ctx context.Context) error

	// Stop gracefully stops the gateway server
	Stop(ctx context.Context) error

	// IsReady returns true if the server is ready to accept requests
	IsReady() bool

	// IsHealthy returns true if the server is healthy
	IsHealthy() bool

	// GetServerInfo returns server information
	GetServerInfo() *types.ServerInfo
}

// ProxyGateway defines the interface for LLM proxy functionality
type ProxyGateway interface {
	// ProxyRequest proxies a request to an LLM provider
	ProxyRequest(ctx context.Context, req *types.LLMRequest, policy *types.PolicyContext) (*types.LLMResponse, error)

	// ProxyStreamingRequest proxies a streaming request to an LLM provider
	ProxyStreamingRequest(ctx context.Context, req *types.LLMRequest, policy *types.PolicyContext) (<-chan *types.LLMResponse, error)

	// ValidateRequest validates a request before proxying
	ValidateRequest(ctx context.Context, req *types.LLMRequest) error

	// GetSupportedProviders returns list of supported providers
	GetSupportedProviders() []types.Provider

	// GetProviderModels returns available models for a provider
	GetProviderModels(ctx context.Context, provider types.Provider) ([]string, error)
}

// EnforcementGateway defines the interface for policy enforcement
type EnforcementGateway interface {
	// CheckRequest evaluates a request against policies
	CheckRequest(ctx context.Context, req *types.LLMRequest, policy *types.PolicyContext) (*types.ScanResult, error)

	// CheckResponse evaluates a response against policies
	CheckResponse(ctx context.Context, resp *types.LLMResponse, policy *types.PolicyContext) (*types.ScanResult, error)

	// EnforcePolicy applies enforcement based on scan results
	EnforcePolicy(ctx context.Context, result *types.ScanResult, mode types.EnforcementMode) (*types.EnforcementDecision, error)

	// GetPolicyDecision gets a cached policy decision if available
	GetPolicyDecision(ctx context.Context, contentHash string) (*types.EnforcementDecision, error)
}

// HTTPMiddleware defines the interface for HTTP middleware
type HTTPMiddleware interface {
	// Middleware returns the HTTP middleware handler
	Middleware(next http.Handler) http.Handler

	// Name returns the middleware name
	Name() string

	// Priority returns the middleware priority (higher = earlier execution)
	Priority() int
}

// RouteHandler defines the interface for HTTP route handlers
type RouteHandler interface {
	// ServeHTTP handles HTTP requests
	ServeHTTP(w http.ResponseWriter, r *http.Request)

	// Pattern returns the route pattern
	Pattern() string

	// Methods returns supported HTTP methods
	Methods() []string

	// RequiredScopes returns required authorization scopes
	RequiredScopes() []string
}

// RequestValidator defines the interface for request validation
type RequestValidator interface {
	// ValidateRequest validates an HTTP request
	ValidateRequest(r *http.Request) error

	// ValidateHeaders validates request headers
	ValidateHeaders(headers http.Header) error

	// ValidateBody validates request body
	ValidateBody(body []byte, contentType string) error

	// GetValidationRules returns validation rules
	GetValidationRules() map[string]interface{}
}

// ResponseWriter defines the interface for writing HTTP responses
type ResponseWriter interface {
	// WriteJSON writes a JSON response
	WriteJSON(w http.ResponseWriter, status int, data interface{}) error

	// WriteError writes an error response
	WriteError(w http.ResponseWriter, err error) error

	// WriteStream writes a streaming response
	WriteStream(w http.ResponseWriter, stream <-chan []byte) error

	// SetHeaders sets response headers
	SetHeaders(w http.ResponseWriter, headers map[string]string)
}

// RequestContext defines the interface for request context management
type RequestContext interface {
	// GetTenantID returns the tenant ID from context
	GetTenantID(ctx context.Context) (string, error)

	// GetUserID returns the user ID from context
	GetUserID(ctx context.Context) (string, error)

	// GetRequestID returns the request ID from context
	GetRequestID(ctx context.Context) string

	// SetTenantID sets the tenant ID in context
	SetTenantID(ctx context.Context, tenantID string) context.Context

	// SetUserID sets the user ID in context
	SetUserID(ctx context.Context, userID string) context.Context

	// SetRequestID sets the request ID in context
	SetRequestID(ctx context.Context, requestID string) context.Context
}

// GRPCServer defines the interface for gRPC server functionality
type GRPCServer interface {
	// Start starts the gRPC server
	Start(ctx context.Context) error

	// Stop gracefully stops the gRPC server
	Stop(ctx context.Context) error

	// IsReady returns true if the server is ready
	IsReady() bool

	// RegisterService registers a gRPC service
	RegisterService(desc interface{}, impl interface{})
}

// ExtProcServer defines the interface for Envoy external processor
type ExtProcServer interface {
	// ProcessRequest processes a request through Envoy ext_proc
	ProcessRequest(ctx context.Context, req *types.ExtProcRequest) (*types.ExtProcResponse, error)

	// ProcessResponse processes a response through Envoy ext_proc
	ProcessResponse(ctx context.Context, resp *types.ExtProcResponseMsg) (*types.ExtProcResponse, error)

	// GetConfiguration returns the ext_proc configuration
	GetConfiguration() *types.ExtProcConfig
}

// LoadBalancer defines the interface for load balancing requests
type LoadBalancer interface {
	// SelectBackend selects a backend for the request
	SelectBackend(ctx context.Context, req *types.LLMRequest) (*types.Backend, error)

	// UpdateBackends updates the list of available backends
	UpdateBackends(backends []*types.Backend) error

	// GetBackends returns the current list of backends
	GetBackends() []*types.Backend

	// HealthCheck performs health checks on backends
	HealthCheck(ctx context.Context) error
}

// Gateway-specific telemetry collector (simplified interface for gateway use)
type GatewayTelemetryCollector interface {
	// Collect emits a telemetry event
	Collect(eventType string, payload map[string]interface{})

	// Shutdown gracefully shuts down the collector
	Shutdown(ctx context.Context) error
}

// GatewayMetricsCollector defines the interface for collecting gateway-specific metrics
type GatewayMetricsCollector interface {
	// RecordRequest records a request metric
	RecordRequest(ctx context.Context, method string, path string, status int, duration time.Duration)

	// RecordError records an error metric
	RecordError(ctx context.Context, errorType string, err error)

	// RecordGauge records a gauge metric
	RecordGauge(name string, value float64, tags map[string]string)

	// RecordCounter records a counter metric
	RecordCounter(name string, value float64, tags map[string]string)

	// RecordHistogram records a histogram metric
	RecordHistogram(name string, value float64, tags map[string]string)
}

// ConfigWatcher defines the interface for watching configuration changes
type ConfigWatcher interface {
	// Watch starts watching for configuration changes
	Watch(ctx context.Context, callback types.ConfigChangeCallback) error

	// Stop stops watching for configuration changes
	Stop() error

	// GetCurrentConfig returns the current configuration
	GetCurrentConfig() map[string]interface{}
}
