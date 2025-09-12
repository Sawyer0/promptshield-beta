package application

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/domain"
)

type experimentService struct {
	repo         ExperimentRepository
	eventTracker EventTracker
}

// NewExperimentService creates a new experiment service
func NewExperimentService(repo ExperimentRepository, eventTracker EventTracker) ExperimentService {
	return &experimentService{
		repo:         repo,
		eventTracker: eventTracker,
	}
}

func (s *experimentService) CreateExperiment(ctx context.Context, experiment *domain.Experiment) error {
	experiment.ID = uuid.New()
	experiment.CreatedAt = time.Now()
	experiment.UpdatedAt = time.Now()

	if experiment.Status == "" {
		experiment.Status = domain.ExperimentStatusDraft
	}

	return s.repo.CreateExperiment(ctx, experiment)
}

func (s *experimentService) GetExperiment(ctx context.Context, experimentID uuid.UUID) (*domain.Experiment, error) {
	return s.repo.GetExperiment(ctx, experimentID)
}

func (s *experimentService) UpdateExperiment(ctx context.Context, experiment *domain.Experiment) error {
	experiment.UpdatedAt = time.Now()
	return s.repo.UpdateExperiment(ctx, experiment)
}

func (s *experimentService) ListExperiments(ctx context.Context, filters ExperimentFilters) ([]*domain.Experiment, error) {
	return s.repo.ListExperiments(ctx, filters)
}

func (s *experimentService) CreateVariant(ctx context.Context, variant *domain.ExperimentVariant) error {
	variant.ID = uuid.New()
	variant.CreatedAt = time.Now()
	variant.UpdatedAt = time.Now()

	return s.repo.CreateVariant(ctx, variant)
}

func (s *experimentService) GetVariant(ctx context.Context, variantID uuid.UUID) (*domain.ExperimentVariant, error) {
	return s.repo.GetVariant(ctx, variantID)
}

func (s *experimentService) UpdateVariant(ctx context.Context, variant *domain.ExperimentVariant) error {
	variant.UpdatedAt = time.Now()
	return s.repo.UpdateVariant(ctx, variant)
}

func (s *experimentService) ListVariants(ctx context.Context, experimentID uuid.UUID) ([]*domain.ExperimentVariant, error) {
	return s.repo.ListVariants(ctx, experimentID)
}

func (s *experimentService) AssignUserToExperiment(ctx context.Context, userID string, tenantID uuid.UUID, experimentID uuid.UUID) (*domain.ExperimentAssignment, error) {
	// Check if user is already assigned
	existing, err := s.GetUserAssignment(ctx, userID, tenantID, experimentID)
	if err == nil && existing != nil {
		return existing, nil
	}

	// Get experiment
	experiment, err := s.GetExperiment(ctx, experimentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get experiment: %w", err)
	}

	// Check if experiment is active
	if experiment.Status != domain.ExperimentStatusActive {
		return nil, fmt.Errorf("experiment is not active")
	}

	// Check traffic allocation
	if rand.Float64() > experiment.TrafficAllocation {
		return nil, fmt.Errorf("user not eligible for experiment")
	}

	// Get variants
	variants, err := s.ListVariants(ctx, experimentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get variants: %w", err)
	}

	if len(variants) == 0 {
		return nil, fmt.Errorf("no variants found for experiment")
	}

	// Select variant based on weight
	selectedVariant := s.selectVariantByWeight(variants)

	// Create assignment
	assignment := &domain.ExperimentAssignment{
		ID:           uuid.New(),
		UserID:       userID,
		TenantID:     tenantID,
		ExperimentID: experimentID,
		VariantID:    selectedVariant.ID,
		AssignedAt:   time.Now(),
	}

	if err := s.repo.CreateAssignment(ctx, assignment); err != nil {
		return nil, fmt.Errorf("failed to create assignment: %w", err)
	}

	slog.Info("User assigned to experiment variant",
		"user_id", userID,
		"tenant_id", tenantID,
		"experiment_id", experimentID,
		"variant_id", selectedVariant.ID,
		"variant_name", selectedVariant.Name,
	)

	return assignment, nil
}

func (s *experimentService) GetUserAssignment(ctx context.Context, userID string, tenantID uuid.UUID, experimentID uuid.UUID) (*domain.ExperimentAssignment, error) {
	return s.repo.GetAssignment(ctx, userID, tenantID, experimentID)
}

func (s *experimentService) TrackConversion(ctx context.Context, assignmentID uuid.UUID, conversionData map[string]interface{}) error {
	event := &ExperimentEvent{
		ID:           uuid.New(),
		AssignmentID: assignmentID,
		EventName:    "conversion",
		EventData:    conversionData,
		Timestamp:    time.Now(),
	}

	if err := s.eventTracker.TrackEvent(ctx, event); err != nil {
		return fmt.Errorf("failed to track conversion: %w", err)
	}

	// Update assignment with conversion time
	assignments, err := s.repo.ListAssignments(ctx, uuid.Nil) // This would need to be fixed to get by assignment ID
	if err != nil {
		return fmt.Errorf("failed to get assignment: %w", err)
	}

	for _, assignment := range assignments {
		if assignment.ID == assignmentID {
			now := time.Now()
			assignment.ConvertedAt = &now
			assignment.LastSeenAt = &now
			return s.repo.UpdateAssignment(ctx, assignment)
		}
	}

	return fmt.Errorf("assignment not found")
}

func (s *experimentService) TrackEvent(ctx context.Context, assignmentID uuid.UUID, eventName string, eventData map[string]interface{}) error {
	event := &ExperimentEvent{
		ID:           uuid.New(),
		AssignmentID: assignmentID,
		EventName:    eventName,
		EventData:    eventData,
		Timestamp:    time.Now(),
	}

	return s.eventTracker.TrackEvent(ctx, event)
}

func (s *experimentService) CalculateExperimentResults(ctx context.Context, experimentID uuid.UUID) (*domain.ExperimentResults, error) {
	// Get experiment
	experiment, err := s.GetExperiment(ctx, experimentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get experiment: %w", err)
	}

	// Get variants
	variants, err := s.ListVariants(ctx, experimentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get variants: %w", err)
	}

	// Get assignments
	assignments, err := s.repo.ListAssignments(ctx, experimentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get assignments: %w", err)
	}

	// Calculate results
	results := &domain.ExperimentResults{
		TotalParticipants: int64(len(assignments)),
	}

	// Group assignments by variant
	variantAssignments := make(map[uuid.UUID][]*domain.ExperimentAssignment)
	for _, assignment := range assignments {
		variantAssignments[assignment.VariantID] = append(variantAssignments[assignment.VariantID], assignment)
	}

	// Calculate variant-specific metrics
	var controlVariant, testVariant *domain.ExperimentVariant
	for _, variant := range variants {
		if variant.Type == domain.VariantTypeControl {
			controlVariant = variant
		} else if variant.Type == domain.VariantTypeTest {
			testVariant = variant
		}
	}

	if controlVariant != nil {
		controlAssignments := variantAssignments[controlVariant.ID]
		results.ControlParticipants = int64(len(controlAssignments))
		results.ControlConversionRate = s.calculateConversionRate(controlAssignments)
	}

	if testVariant != nil {
		testAssignments := variantAssignments[testVariant.ID]
		results.TestParticipants = int64(len(testAssignments))
		results.TestConversionRate = s.calculateConversionRate(testAssignments)
	}

	// Calculate overall conversion rate
	results.ConversionRate = s.calculateConversionRate(assignments)

	// Calculate lift
	if results.ControlConversionRate > 0 {
		results.Lift = ((results.TestConversionRate - results.ControlConversionRate) / results.ControlConversionRate) * 100
	}

	// Calculate statistical significance (simplified)
	results.StatisticalSignificance = s.calculateStatisticalSignificance(
		results.ControlParticipants, results.ControlConversionRate,
		results.TestParticipants, results.TestConversionRate,
	)

	// Update experiment with results
	experiment.Results = results
	experiment.UpdatedAt = time.Now()

	if err := s.repo.UpdateExperiment(ctx, experiment); err != nil {
		slog.Error("Failed to update experiment with results", "error", err, "experiment_id", experimentID)
	}

	return results, nil
}

func (s *experimentService) GetExperimentAnalytics(ctx context.Context, experimentID uuid.UUID) (*ExperimentAnalytics, error) {
	// This would be implemented with more sophisticated analytics
	// For now, return basic analytics
	return &ExperimentAnalytics{
		ExperimentID:     experimentID,
		TotalUsers:       0,
		ActiveUsers:      0,
		ConversionFunnel: make(map[string]int64),
		RevenueImpact:    0.0,
		ChurnImpact:      0.0,
		VariantBreakdown: make(map[string]*VariantAnalytics),
		TimeSeriesData:   []TimeSeriesPoint{},
	}, nil
}

func (s *experimentService) CreatePricingExperiment(ctx context.Context, experiment *domain.PricingExperiment) error {
	// Convert to base experiment
	baseExperiment := &domain.Experiment{
		Name:              experiment.Name,
		Description:       experiment.Description,
		Type:              domain.ExperimentTypePricingTier,
		Status:            experiment.Status,
		TrafficAllocation: experiment.TrafficAllocation,
		TargetCriteria:    experiment.TargetCriteria,
		StartDate:         experiment.StartDate,
		EndDate:           experiment.EndDate,
		PrimaryMetric:     experiment.PrimaryMetric,
		SecondaryMetrics:  experiment.SecondaryMetrics,
	}

	if err := s.CreateExperiment(ctx, baseExperiment); err != nil {
		return err
	}

	// Create control variant
	controlVariant := &domain.ExperimentVariant{
		ExperimentID:  baseExperiment.ID,
		Name:          "Control Pricing",
		Type:          domain.VariantTypeControl,
		Weight:        0.5,
		Configuration: experiment.ControlPricing,
	}

	if err := s.CreateVariant(ctx, controlVariant); err != nil {
		return err
	}

	// Create test variant
	testVariant := &domain.ExperimentVariant{
		ExperimentID:  baseExperiment.ID,
		Name:          "Test Pricing",
		Type:          domain.VariantTypeTest,
		Weight:        0.5,
		Configuration: experiment.TestPricing,
	}

	return s.CreateVariant(ctx, testVariant)
}

func (s *experimentService) GetPricingForUser(ctx context.Context, userID string, tenantID uuid.UUID) (map[string]interface{}, error) {
	// Get active pricing experiments
	experiments, err := s.ListExperiments(ctx, ExperimentFilters{
		Status: &[]domain.ExperimentStatus{domain.ExperimentStatusActive}[0],
		Type:   &[]domain.ExperimentType{domain.ExperimentTypePricingTier}[0],
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get pricing experiments: %w", err)
	}

	// Check if user is assigned to any pricing experiment
	for _, experiment := range experiments {
		assignment, err := s.GetUserAssignment(ctx, userID, tenantID, experiment.ID)
		if err == nil && assignment != nil {
			// Get variant configuration
			variant, err := s.GetVariant(ctx, assignment.VariantID)
			if err == nil {
				return variant.Configuration, nil
			}
		}
	}

	// Return default pricing if no experiment assignment
	return map[string]interface{}{
		"professional": map[string]interface{}{
			"monthly": 29900,
			"yearly":  299000,
		},
		"enterprise": map[string]interface{}{
			"monthly": 199900,
			"yearly":  1999000,
		},
	}, nil
}

func (s *experimentService) CreateUsageAddonExperiment(ctx context.Context, experiment *domain.UsageAddonExperiment) error {
	// Similar to pricing experiment but for usage addons
	baseExperiment := &domain.Experiment{
		Name:              experiment.Name,
		Description:       experiment.Description,
		Type:              domain.ExperimentTypeUsageAddon,
		Status:            experiment.Status,
		TrafficAllocation: experiment.TrafficAllocation,
		TargetCriteria:    experiment.TargetCriteria,
		StartDate:         experiment.StartDate,
		EndDate:           experiment.EndDate,
		PrimaryMetric:     experiment.PrimaryMetric,
		SecondaryMetrics:  experiment.SecondaryMetrics,
	}

	if err := s.CreateExperiment(ctx, baseExperiment); err != nil {
		return err
	}

	// Create control variant
	controlVariant := &domain.ExperimentVariant{
		ExperimentID:  baseExperiment.ID,
		Name:          "Control Addons",
		Type:          domain.VariantTypeControl,
		Weight:        0.5,
		Configuration: map[string]interface{}{"addons": experiment.ControlAddons},
	}

	if err := s.CreateVariant(ctx, controlVariant); err != nil {
		return err
	}

	// Create test variant
	testVariant := &domain.ExperimentVariant{
		ExperimentID:  baseExperiment.ID,
		Name:          "Test Addons",
		Type:          domain.VariantTypeTest,
		Weight:        0.5,
		Configuration: map[string]interface{}{"addons": experiment.TestAddons},
	}

	return s.CreateVariant(ctx, testVariant)
}

func (s *experimentService) GetAvailableAddonsForUser(ctx context.Context, userID string, tenantID uuid.UUID) ([]string, error) {
	// Similar to pricing, but for addons
	experiments, err := s.ListExperiments(ctx, ExperimentFilters{
		Status: &[]domain.ExperimentStatus{domain.ExperimentStatusActive}[0],
		Type:   &[]domain.ExperimentType{domain.ExperimentTypeUsageAddon}[0],
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get addon experiments: %w", err)
	}

	for _, experiment := range experiments {
		assignment, err := s.GetUserAssignment(ctx, userID, tenantID, experiment.ID)
		if err == nil && assignment != nil {
			variant, err := s.GetVariant(ctx, assignment.VariantID)
			if err == nil {
				if addons, ok := variant.Configuration["addons"].([]string); ok {
					return addons, nil
				}
			}
		}
	}

	// Return default addons
	return []string{"basic_analytics", "email_support"}, nil
}

func (s *experimentService) StartExperiment(ctx context.Context, experimentID uuid.UUID) error {
	experiment, err := s.GetExperiment(ctx, experimentID)
	if err != nil {
		return err
	}

	experiment.Status = domain.ExperimentStatusActive
	now := time.Now()
	experiment.StartDate = &now

	return s.UpdateExperiment(ctx, experiment)
}

func (s *experimentService) PauseExperiment(ctx context.Context, experimentID uuid.UUID) error {
	experiment, err := s.GetExperiment(ctx, experimentID)
	if err != nil {
		return err
	}

	experiment.Status = domain.ExperimentStatusPaused
	return s.UpdateExperiment(ctx, experiment)
}

func (s *experimentService) CompleteExperiment(ctx context.Context, experimentID uuid.UUID) error {
	experiment, err := s.GetExperiment(ctx, experimentID)
	if err != nil {
		return err
	}

	experiment.Status = domain.ExperimentStatusCompleted
	now := time.Now()
	experiment.EndDate = &now

	// Calculate final results
	results, err := s.CalculateExperimentResults(ctx, experimentID)
	if err != nil {
		slog.Error("Failed to calculate final results", "error", err, "experiment_id", experimentID)
	} else {
		experiment.Results = results
	}

	return s.UpdateExperiment(ctx, experiment)
}

func (s *experimentService) CancelExperiment(ctx context.Context, experimentID uuid.UUID) error {
	experiment, err := s.GetExperiment(ctx, experimentID)
	if err != nil {
		return err
	}

	experiment.Status = domain.ExperimentStatusCancelled
	return s.UpdateExperiment(ctx, experiment)
}

// Helper methods

func (s *experimentService) selectVariantByWeight(variants []*domain.ExperimentVariant) *domain.ExperimentVariant {
	totalWeight := 0.0
	for _, variant := range variants {
		totalWeight += variant.Weight
	}

	random := rand.Float64() * totalWeight
	current := 0.0

	for _, variant := range variants {
		current += variant.Weight
		if random <= current {
			return variant
		}
	}

	// Fallback to last variant
	return variants[len(variants)-1]
}

func (s *experimentService) calculateConversionRate(assignments []*domain.ExperimentAssignment) float64 {
	if len(assignments) == 0 {
		return 0.0
	}

	conversions := 0
	for _, assignment := range assignments {
		if assignment.ConvertedAt != nil {
			conversions++
		}
	}

	return float64(conversions) / float64(len(assignments))
}

func (s *experimentService) calculateStatisticalSignificance(controlN int64, controlRate float64, testN int64, testRate float64) bool {
	// Simplified statistical significance calculation
	// In production, use proper statistical tests like chi-square or t-test

	if controlN < 30 || testN < 30 {
		return false // Not enough samples
	}

	// Calculate standard error
	controlSE := math.Sqrt((controlRate * (1 - controlRate)) / float64(controlN))
	testSE := math.Sqrt((testRate * (1 - testRate)) / float64(testN))

	// Calculate z-score
	se := math.Sqrt(controlSE*controlSE + testSE*testSE)
	if se == 0 {
		return false
	}

	zScore := math.Abs(testRate-controlRate) / se

	// 95% confidence level (z-score > 1.96)
	return zScore > 1.96
}
