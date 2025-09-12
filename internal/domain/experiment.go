package domain

import (
	"time"

	"github.com/google/uuid"
)

// ExperimentStatus represents the status of an A/B test experiment
type ExperimentStatus string

const (
	ExperimentStatusDraft     ExperimentStatus = "draft"
	ExperimentStatusActive    ExperimentStatus = "active"
	ExperimentStatusPaused    ExperimentStatus = "paused"
	ExperimentStatusCompleted ExperimentStatus = "completed"
	ExperimentStatusCancelled ExperimentStatus = "cancelled"
)

// ExperimentType represents the type of experiment
type ExperimentType string

const (
	ExperimentTypePricingTier    ExperimentType = "pricing_tier"
	ExperimentTypeUsageAddon     ExperimentType = "usage_addon"
	ExperimentTypeFeatureFlag    ExperimentType = "feature_flag"
	ExperimentTypeUIOptimization ExperimentType = "ui_optimization"
)

// VariantType represents the type of variant
type VariantType string

const (
	VariantTypeControl VariantType = "control"
	VariantTypeTest    VariantType = "test"
)

// Experiment represents an A/B test experiment
type Experiment struct {
	ID          uuid.UUID        `json:"id" db:"id"`
	Name        string           `json:"name" db:"name"`
	Description string           `json:"description" db:"description"`
	Type        ExperimentType   `json:"type" db:"type"`
	Status      ExperimentStatus `json:"status" db:"status"`

	// Traffic allocation (0.0 to 1.0)
	TrafficAllocation float64 `json:"traffic_allocation" db:"traffic_allocation"`

	// Target audience criteria
	TargetCriteria map[string]interface{} `json:"target_criteria" db:"target_criteria"`

	// Experiment duration
	StartDate *time.Time `json:"start_date" db:"start_date"`
	EndDate   *time.Time `json:"end_date" db:"end_date"`

	// Success metrics
	PrimaryMetric    string   `json:"primary_metric" db:"primary_metric"`
	SecondaryMetrics []string `json:"secondary_metrics" db:"secondary_metrics"`

	// Results
	Results *ExperimentResults `json:"results,omitempty" db:"results"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`

	// Related data
	Variants []ExperimentVariant `json:"variants,omitempty"`
}

// ExperimentVariant represents a variant in an A/B test
type ExperimentVariant struct {
	ID           uuid.UUID   `json:"id" db:"id"`
	ExperimentID uuid.UUID   `json:"experiment_id" db:"experiment_id"`
	Name         string      `json:"name" db:"name"`
	Type         VariantType `json:"type" db:"type"`
	Weight       float64     `json:"weight" db:"weight"` // Traffic weight (0.0 to 1.0)

	// Variant configuration
	Configuration map[string]interface{} `json:"configuration" db:"configuration"`

	// Results
	Results *VariantResults `json:"results,omitempty" db:"results"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// ExperimentResults represents the results of an experiment
type ExperimentResults struct {
	TotalParticipants       int64      `json:"total_participants"`
	ControlParticipants     int64      `json:"control_participants"`
	TestParticipants        int64      `json:"test_participants"`
	ConversionRate          float64    `json:"conversion_rate"`
	ControlConversionRate   float64    `json:"control_conversion_rate"`
	TestConversionRate      float64    `json:"test_conversion_rate"`
	Lift                    float64    `json:"lift"` // Percentage improvement
	ConfidenceLevel         float64    `json:"confidence_level"`
	StatisticalSignificance bool       `json:"statistical_significance"`
	PValue                  float64    `json:"p_value"`
	CompletedAt             *time.Time `json:"completed_at"`
}

// VariantResults represents the results of a specific variant
type VariantResults struct {
	Participants      int64   `json:"participants"`
	Conversions       int64   `json:"conversions"`
	ConversionRate    float64 `json:"conversion_rate"`
	Revenue           float64 `json:"revenue"`
	AverageOrderValue float64 `json:"average_order_value"`
	ChurnRate         float64 `json:"churn_rate"`
	LifetimeValue     float64 `json:"lifetime_value"`
}

// ExperimentAssignment represents a user's assignment to an experiment variant
type ExperimentAssignment struct {
	ID           uuid.UUID `json:"id" db:"id"`
	UserID       string    `json:"user_id" db:"user_id"`
	TenantID     uuid.UUID `json:"tenant_id" db:"tenant_id"`
	ExperimentID uuid.UUID `json:"experiment_id" db:"experiment_id"`
	VariantID    uuid.UUID `json:"variant_id" db:"variant_id"`
	AssignedAt   time.Time `json:"assigned_at" db:"assigned_at"`

	// Tracking data
	FirstSeenAt *time.Time `json:"first_seen_at" db:"first_seen_at"`
	LastSeenAt  *time.Time `json:"last_seen_at" db:"last_seen_at"`
	ConvertedAt *time.Time `json:"converted_at" db:"converted_at"`
}

// PricingExperiment represents a pricing-specific experiment
type PricingExperiment struct {
	Experiment
	ControlPricing map[string]interface{} `json:"control_pricing"`
	TestPricing    map[string]interface{} `json:"test_pricing"`
}

// UsageAddonExperiment represents a usage addon experiment
type UsageAddonExperiment struct {
	Experiment
	ControlAddons []string `json:"control_addons"`
	TestAddons    []string `json:"test_addons"`
}
