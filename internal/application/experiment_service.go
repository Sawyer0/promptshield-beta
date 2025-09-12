package application

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/domain"
)

// ExperimentService defines the interface for A/B testing and experiment management
type ExperimentService interface {
	// Experiment management
	CreateExperiment(ctx context.Context, experiment *domain.Experiment) error
	GetExperiment(ctx context.Context, experimentID uuid.UUID) (*domain.Experiment, error)
	UpdateExperiment(ctx context.Context, experiment *domain.Experiment) error
	ListExperiments(ctx context.Context, filters ExperimentFilters) ([]*domain.Experiment, error)

	// Variant management
	CreateVariant(ctx context.Context, variant *domain.ExperimentVariant) error
	GetVariant(ctx context.Context, variantID uuid.UUID) (*domain.ExperimentVariant, error)
	UpdateVariant(ctx context.Context, variant *domain.ExperimentVariant) error
	ListVariants(ctx context.Context, experimentID uuid.UUID) ([]*domain.ExperimentVariant, error)

	// Assignment and tracking
	AssignUserToExperiment(ctx context.Context, userID string, tenantID uuid.UUID, experimentID uuid.UUID) (*domain.ExperimentAssignment, error)
	GetUserAssignment(ctx context.Context, userID string, tenantID uuid.UUID, experimentID uuid.UUID) (*domain.ExperimentAssignment, error)
	TrackConversion(ctx context.Context, assignmentID uuid.UUID, conversionData map[string]interface{}) error
	TrackEvent(ctx context.Context, assignmentID uuid.UUID, eventName string, eventData map[string]interface{}) error

	// Results and analytics
	CalculateExperimentResults(ctx context.Context, experimentID uuid.UUID) (*domain.ExperimentResults, error)
	GetExperimentAnalytics(ctx context.Context, experimentID uuid.UUID) (*ExperimentAnalytics, error)

	// Pricing experiments
	CreatePricingExperiment(ctx context.Context, experiment *domain.PricingExperiment) error
	GetPricingForUser(ctx context.Context, userID string, tenantID uuid.UUID) (map[string]interface{}, error)

	// Usage addon experiments
	CreateUsageAddonExperiment(ctx context.Context, experiment *domain.UsageAddonExperiment) error
	GetAvailableAddonsForUser(ctx context.Context, userID string, tenantID uuid.UUID) ([]string, error)

	// Experiment lifecycle
	StartExperiment(ctx context.Context, experimentID uuid.UUID) error
	PauseExperiment(ctx context.Context, experimentID uuid.UUID) error
	CompleteExperiment(ctx context.Context, experimentID uuid.UUID) error
	CancelExperiment(ctx context.Context, experimentID uuid.UUID) error
}

// ExperimentFilters represents filters for listing experiments
type ExperimentFilters struct {
	Status    *domain.ExperimentStatus `json:"status,omitempty"`
	Type      *domain.ExperimentType   `json:"type,omitempty"`
	StartDate *time.Time               `json:"start_date,omitempty"`
	EndDate   *time.Time               `json:"end_date,omitempty"`
	Limit     int                      `json:"limit,omitempty"`
	Offset    int                      `json:"offset,omitempty"`
}

// ExperimentAnalytics represents analytics data for an experiment
type ExperimentAnalytics struct {
	ExperimentID     uuid.UUID                    `json:"experiment_id"`
	TotalUsers       int64                        `json:"total_users"`
	ActiveUsers      int64                        `json:"active_users"`
	ConversionFunnel map[string]int64             `json:"conversion_funnel"`
	RevenueImpact    float64                      `json:"revenue_impact"`
	ChurnImpact      float64                      `json:"churn_impact"`
	VariantBreakdown map[string]*VariantAnalytics `json:"variant_breakdown"`
	TimeSeriesData   []TimeSeriesPoint            `json:"time_series_data"`
}

// VariantAnalytics represents analytics for a specific variant
type VariantAnalytics struct {
	VariantID         uuid.UUID `json:"variant_id"`
	Users             int64     `json:"users"`
	Conversions       int64     `json:"conversions"`
	ConversionRate    float64   `json:"conversion_rate"`
	Revenue           float64   `json:"revenue"`
	AverageOrderValue float64   `json:"average_order_value"`
	ChurnRate         float64   `json:"churn_rate"`
	LifetimeValue     float64   `json:"lifetime_value"`
}

// TimeSeriesPoint represents a data point in time series
type TimeSeriesPoint struct {
	Timestamp time.Time  `json:"timestamp"`
	Value     float64    `json:"value"`
	Metric    string     `json:"metric"`
	VariantID *uuid.UUID `json:"variant_id,omitempty"`
}

// ExperimentRepository defines the interface for experiment data operations
type ExperimentRepository interface {
	// Experiment operations
	CreateExperiment(ctx context.Context, experiment *domain.Experiment) error
	GetExperiment(ctx context.Context, experimentID uuid.UUID) (*domain.Experiment, error)
	UpdateExperiment(ctx context.Context, experiment *domain.Experiment) error
	ListExperiments(ctx context.Context, filters ExperimentFilters) ([]*domain.Experiment, error)

	// Variant operations
	CreateVariant(ctx context.Context, variant *domain.ExperimentVariant) error
	GetVariant(ctx context.Context, variantID uuid.UUID) (*domain.ExperimentVariant, error)
	UpdateVariant(ctx context.Context, variant *domain.ExperimentVariant) error
	ListVariants(ctx context.Context, experimentID uuid.UUID) ([]*domain.ExperimentVariant, error)

	// Assignment operations
	CreateAssignment(ctx context.Context, assignment *domain.ExperimentAssignment) error
	GetAssignment(ctx context.Context, userID string, tenantID uuid.UUID, experimentID uuid.UUID) (*domain.ExperimentAssignment, error)
	UpdateAssignment(ctx context.Context, assignment *domain.ExperimentAssignment) error
	ListAssignments(ctx context.Context, experimentID uuid.UUID) ([]*domain.ExperimentAssignment, error)

	// Analytics operations
	GetExperimentMetrics(ctx context.Context, experimentID uuid.UUID, startDate, endDate time.Time) (map[string]interface{}, error)
	GetVariantMetrics(ctx context.Context, variantID uuid.UUID, startDate, endDate time.Time) (map[string]interface{}, error)
	GetConversionFunnel(ctx context.Context, experimentID uuid.UUID) (map[string]int64, error)
	GetTimeSeriesData(ctx context.Context, experimentID uuid.UUID, metric string, startDate, endDate time.Time) ([]TimeSeriesPoint, error)
}

// ExperimentEvent represents an event in an experiment
type ExperimentEvent struct {
	ID           uuid.UUID              `json:"id"`
	AssignmentID uuid.UUID              `json:"assignment_id"`
	EventName    string                 `json:"event_name"`
	EventData    map[string]interface{} `json:"event_data"`
	Timestamp    time.Time              `json:"timestamp"`
}

// EventTracker defines the interface for tracking experiment events
type EventTracker interface {
	TrackEvent(ctx context.Context, event *ExperimentEvent) error
	GetEvents(ctx context.Context, assignmentID uuid.UUID, eventName string, startDate, endDate time.Time) ([]*ExperimentEvent, error)
	GetConversionEvents(ctx context.Context, experimentID uuid.UUID, startDate, endDate time.Time) ([]*ExperimentEvent, error)
}
