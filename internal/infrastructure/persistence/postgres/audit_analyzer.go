package postgres

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/shared/contracts"
	"github.com/promptshield/promptshield/internal/shared/types"
)

// pgAuditAnalyzer implements contracts.AuditAnalyzer with PostgreSQL backend
type pgAuditAnalyzer struct {
	eventStore contracts.AuditEventStore
}

// NewAuditAnalyzer creates a new PostgreSQL audit analyzer
func NewAuditAnalyzer(eventStore contracts.AuditEventStore) contracts.AuditAnalyzer {
	return &pgAuditAnalyzer{
		eventStore: eventStore,
	}
}

// AnalyzePatterns analyzes patterns in audit data
func (a *pgAuditAnalyzer) AnalyzePatterns(ctx context.Context, timeRange types.TimeRange) (*types.AuditPatternAnalysis, error) {
	// Get all events in time range
	filter := &types.AuditFilter{
		StartTime: &timeRange.Start,
		EndTime:   &timeRange.End,
	}

	events, err := a.eventStore.Retrieve(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("retrieve events for pattern analysis: %w", err)
	}

	analysis := &types.AuditPatternAnalysis{
		TimeRange:   timeRange,
		Patterns:    make([]*types.AuditPattern, 0),
		Trends:      make([]*types.AuditTrend, 0),
		GeneratedAt: time.Now(),
	}

	// Analyze different patterns
	analysis.Patterns = append(analysis.Patterns, a.analyzeUserBehaviorPatterns(events)...)
	analysis.Patterns = append(analysis.Patterns, a.analyzeResourceAccessPatterns(events)...)
	analysis.Patterns = append(analysis.Patterns, a.analyzeTimeBasedPatterns(events)...)

	// Analyze trends
	analysis.Trends = a.analyzeTrends(events, timeRange)

	// Detect anomalies
	anomalies, err := a.DetectAnomalies(ctx, timeRange)
	if err != nil {
		return nil, fmt.Errorf("detect anomalies: %w", err)
	}
	analysis.Anomalies = anomalies

	// Generate correlations
	analysis.Correlations = a.correlateEvents(ctx, events)

	// Calculate risk indicators
	analysis.RiskIndicators = a.calculateRiskIndicators(events)

	return analysis, nil
}

// DetectAnomalies detects anomalous activities in audit data
func (a *pgAuditAnalyzer) DetectAnomalies(ctx context.Context, timeRange types.TimeRange) ([]*types.AuditAnomaly, error) {
	filter := &types.AuditFilter{
		StartTime: &timeRange.Start,
		EndTime:   &timeRange.End,
	}

	events, err := a.eventStore.Retrieve(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("retrieve events for anomaly detection: %w", err)
	}

	anomalies := make([]*types.AuditAnomaly, 0)

	// Analyze for different types of anomalies
	anomalies = append(anomalies, a.detectFailedLoginAnomalies(events)...)
	anomalies = append(anomalies, a.detectPrivilegeEscalationAnomalies(events)...)
	anomalies = append(anomalies, a.detectDataAccessAnomalies(events)...)
	anomalies = append(anomalies, a.detectTimeBasedAnomalies(events)...)

	return anomalies, nil
}

// GetTrendAnalysis returns trend analysis for audit events
func (a *pgAuditAnalyzer) GetTrendAnalysis(ctx context.Context, timeRange types.TimeRange, granularity time.Duration) (*types.AuditTrendAnalysis, error) {
	events, err := a.eventStore.Retrieve(ctx, &types.AuditFilter{
		StartTime: &timeRange.Start,
		EndTime:   &timeRange.End,
	})
	if err != nil {
		return nil, fmt.Errorf("retrieve events for trend analysis: %w", err)
	}

	analysis := &types.AuditTrendAnalysis{
		TimeRange:   timeRange,
		Granularity: granularity.String(),
		Trends:      make([]*types.AuditTrend, 0),
	}

	// Group events by time buckets
	buckets := a.groupEventsByTimeBuckets(events, granularity)

	// Analyze trends for different metrics
	analysis.Trends = append(analysis.Trends, a.analyzeEventVolumeTrend(buckets)...)
	analysis.Trends = append(analysis.Trends, a.analyzeErrorRateTrend(buckets)...)
	analysis.Trends = append(analysis.Trends, a.analyzeUserActivityTrend(buckets)...)

	return analysis, nil
}

// GetRiskAssessment returns risk assessment based on audit data
func (a *pgAuditAnalyzer) GetRiskAssessment(ctx context.Context, entityID string) (*types.RiskAssessment, error) {
	// Get events for this entity
	startTime := time.Now().Add(-30 * 24 * time.Hour)
	endTime := time.Now()
	events, err := a.eventStore.Retrieve(ctx, &types.AuditFilter{
		ObjectID:  &uuid.UUID{},                         // TODO: Parse entityID properly
		StartTime: &startTime, // Last 30 days
		EndTime:   &endTime,
	})
	if err != nil {
		return nil, fmt.Errorf("retrieve events for risk assessment: %w", err)
	}

	assessment := &types.RiskAssessment{
		EntityID:        entityID,
		OverallRisk:     "LOW",
		RiskScore:       0,
		RiskFactors:     make([]*types.RiskFactor, 0),
		Recommendations: make([]string, 0),
		AssessedAt:      time.Now(),
	}

	// Calculate risk based on various factors
	riskFactors := a.calculateRiskFactors(events)
	assessment.RiskFactors = riskFactors

	// Calculate overall risk score
	totalScore := 0.0
	for _, factor := range riskFactors {
		totalScore += factor.Score
	}
	assessment.RiskScore = totalScore

	// Determine risk level
	if totalScore >= 8.0 {
		assessment.OverallRisk = "CRITICAL"
		assessment.Recommendations = append(assessment.Recommendations, "Immediate security review required")
	} else if totalScore >= 6.0 {
		assessment.OverallRisk = "HIGH"
		assessment.Recommendations = append(assessment.Recommendations, "Enhanced monitoring recommended")
	} else if totalScore >= 4.0 {
		assessment.OverallRisk = "MEDIUM"
		assessment.Recommendations = append(assessment.Recommendations, "Regular monitoring advised")
	}

	return assessment, nil
}

// CorrelateEvents correlates related audit events
func (a *pgAuditAnalyzer) CorrelateEvents(ctx context.Context, events []*types.AuditEvent) ([]*types.EventCorrelation, error) {
	correlations := make([]*types.EventCorrelation, 0)

	// Group events by user session (same user, close in time)
	userSessions := a.groupEventsByUserSession(events)

	for sessionID, sessionEvents := range userSessions {
		if len(sessionEvents) < 2 {
			continue
		}

		// Look for suspicious patterns in the session
		correlation := a.analyzeSessionForCorrelation(sessionID, sessionEvents)
		if correlation != nil {
			correlations = append(correlations, correlation)
		}
	}

	// Look for events affecting the same resource
	resourceGroups := a.groupEventsByResource(events)
	for resourceID, resourceEvents := range resourceGroups {
		if len(resourceEvents) > 5 { // Multiple accesses to same resource
			correlation := &types.EventCorrelation{
				ID:           uuid.New().String(),
				Type:         "resource_access_cluster",
				Description:  fmt.Sprintf("Multiple accesses to resource %s", resourceID),
				Severity:     "LOW",
				Events:       resourceEvents,
				CorrelatedAt: time.Now(),
			}
			correlations = append(correlations, correlation)
		}
	}

	return correlations, nil
}

// GetBehaviorBaseline establishes behavioral baseline from audit data
func (a *pgAuditAnalyzer) GetBehaviorBaseline(ctx context.Context, userID string, timeRange types.TimeRange) (*types.BehaviorBaseline, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	events, err := a.eventStore.Retrieve(ctx, &types.AuditFilter{
		ActorID:   userUUID.String(),
		StartTime: &timeRange.Start,
		EndTime:   &timeRange.End,
	})
	if err != nil {
		return nil, fmt.Errorf("retrieve events for baseline: %w", err)
	}

	baseline := &types.BehaviorBaseline{
		UserID:         userID,
		TimeRange:      timeRange,
		NormalPatterns: make(map[string]interface{}),
		EstablishedAt:  time.Now(),
	}

	// Analyze normal patterns
	baseline.NormalPatterns = a.extractNormalPatterns(events)

	return baseline, nil
}

// CompareToBaseline compares current activity to established baseline
func (a *pgAuditAnalyzer) CompareToBaseline(ctx context.Context, userID string, activity []*types.AuditEvent) (*types.BaselineComparison, error) {
	baseline, err := a.GetBehaviorBaseline(ctx, userID, types.TimeRange{
		Start: time.Now().Add(-30 * 24 * time.Hour),
		End:   time.Now(),
	})
	if err != nil {
		return nil, fmt.Errorf("get baseline for comparison: %w", err)
	}

	comparison := &types.BaselineComparison{
		UserID:          userID,
		Baseline:        baseline,
		CurrentActivity: activity,
		Deviations:      make([]*types.Deviation, 0),
		OverallMatch:    "NORMAL",
		ComparedAt:      time.Now(),
	}

	// Compare current activity to baseline
	deviations := a.compareActivityToBaseline(activity, baseline)
	comparison.Deviations = deviations

	// Determine if activity is anomalous
	if len(deviations) > 3 {
		comparison.OverallMatch = "ANOMALOUS"
	} else if len(deviations) > 1 {
		comparison.OverallMatch = "SUSPICIOUS"
	}

	return comparison, nil
}

// Helper functions for analysis

func (a *pgAuditAnalyzer) analyzeUserBehaviorPatterns(events []*types.AuditEvent) []*types.AuditPattern {
	patterns := make([]*types.AuditPattern, 0)

	// Group by user
	userGroups := make(map[string][]*types.AuditEvent)
	for _, event := range events {
		userID := "unknown"
		if event.ActorID != nil {
			userID = event.ActorID.String()
		}
		userGroups[userID] = append(userGroups[userID], event)
	}

	// Analyze each user's behavior
	for userID, userEvents := range userGroups {
		if len(userEvents) < 5 {
			continue // Not enough data
		}

		// Check for repetitive actions
		actionCounts := make(map[string]int)
		for _, event := range userEvents {
			actionCounts[event.Action]++
		}

		// Find most common action
		var mostCommonAction string
		var maxCount int
		for action, count := range actionCounts {
			if count > maxCount {
				maxCount = count
				mostCommonAction = action
			}
		}

		if maxCount > len(userEvents)/2 { // More than 50% of actions are the same
			patterns = append(patterns, &types.AuditPattern{
				Type:        "repetitive_behavior",
				Description: fmt.Sprintf("User %s performs action '%s' repeatedly (%d/%d events)", userID, mostCommonAction, maxCount, len(userEvents)),
				Severity:    "LOW",
				Users:       []string{userID},
				Actions:     []string{mostCommonAction},
				Confidence:  float64(maxCount) / float64(len(userEvents)),
			})
		}
	}

	return patterns
}

func (a *pgAuditAnalyzer) analyzeResourceAccessPatterns(events []*types.AuditEvent) []*types.AuditPattern {
	patterns := make([]*types.AuditPattern, 0)

	// Group by resource
	resourceGroups := a.groupEventsByResource(events)

	for resourceID, resourceEvents := range resourceGroups {
		if len(resourceEvents) < 3 {
			continue
		}

		// Check for access from unusual locations or times
		uniqueUsers := make(map[string]bool)
		for _, event := range resourceEvents {
			if event.ActorID != nil {
				uniqueUsers[event.ActorID.String()] = true
			}
		}

		if len(uniqueUsers) == 1 {
			// Same user accessing resource repeatedly
			userID := ""
			for id := range uniqueUsers {
				userID = id
				break
			}

			patterns = append(patterns, &types.AuditPattern{
				Type:        "resource_ownership",
				Description: fmt.Sprintf("Resource %s accessed exclusively by user %s", resourceID, userID),
				Severity:    "INFO",
				Resources:   []string{resourceID},
				Users:       []string{userID},
				Confidence:  1.0,
			})
		}
	}

	return patterns
}

func (a *pgAuditAnalyzer) analyzeTimeBasedPatterns(events []*types.AuditEvent) []*types.AuditPattern {
	patterns := make([]*types.AuditPattern, 0)

	// Group by hour of day
	hourlyActivity := make(map[int][]*types.AuditEvent)
	for _, event := range events {
		hour := event.Timestamp.Hour()
		hourlyActivity[hour] = append(hourlyActivity[hour], event)
	}

	// Find peak hours
	var peakHour int
	var maxActivity int
	for hour, hourEvents := range hourlyActivity {
		if len(hourEvents) > maxActivity {
			maxActivity = len(hourEvents)
			peakHour = hour
		}
	}

	if maxActivity > len(events)/4 { // Peak hour has more than 25% of activity
		patterns = append(patterns, &types.AuditPattern{
			Type:        "peak_activity",
			Description: fmt.Sprintf("Peak activity at hour %d (%d events, %.1f%% of total)", peakHour, maxActivity, float64(maxActivity)/float64(len(events))*100),
			Severity:    "INFO",
			Confidence:  float64(maxActivity) / float64(len(events)),
		})
	}

	return patterns
}

func (a *pgAuditAnalyzer) analyzeTrends(events []*types.AuditEvent, timeRange types.TimeRange) []*types.AuditTrend {
	trends := make([]*types.AuditTrend, 0)

	// Group events by day
	dailyBuckets := a.groupEventsByTimeBuckets(events, 24*time.Hour)

	// Calculate daily event volume trend
	dailyVolumes := make([]int, 0)
	days := make([]time.Time, 0)

	for _, bucket := range dailyBuckets {
		dailyVolumes = append(dailyVolumes, len(bucket))
		if len(bucket) > 0 {
			days = append(days, bucket[0].Timestamp)
		}
	}

	if len(dailyVolumes) >= 3 {
		trend := a.calculateTrend(dailyVolumes)
		trends = append(trends, &types.AuditTrend{
			Metric:       "daily_event_volume",
			TimeRange:    timeRange,
			Granularity:  24 * time.Hour,
			DataPoints:   a.convertToTrendDataPoints(days, dailyVolumes),
			Direction:    trend.direction,
			ChangeRate:   trend.rate,
			Significance: trend.significance,
		})
	}

	return trends
}

func (a *pgAuditAnalyzer) detectFailedLoginAnomalies(events []*types.AuditEvent) []*types.AuditAnomaly {
	anomalies := make([]*types.AuditAnomaly, 0)

	// Look for failed login patterns
	for _, event := range events {
		if strings.Contains(event.Action, "login") && strings.Contains(event.Action, "failed") {
			// Check if this is part of a pattern
			userID := "unknown"
			if event.ActorID != nil {
				userID = event.ActorID.String()
			}

			// Count recent failed logins for this user
			failedCount := 0
			for _, otherEvent := range events {
				if otherEvent.ActorID != nil && otherEvent.ActorID.String() == userID &&
					strings.Contains(otherEvent.Action, "login") &&
					strings.Contains(otherEvent.Action, "failed") &&
					otherEvent.Timestamp.After(event.Timestamp.Add(-1*time.Hour)) {
					failedCount++
				}
			}

			if failedCount >= 5 {
				anomalies = append(anomalies, &types.AuditAnomaly{
					ID:          uuid.New().String(),
					Type:        "brute_force_attempt",
					Severity:    "HIGH",
					Description: fmt.Sprintf("Multiple failed login attempts for user %s (%d attempts)", userID, failedCount),
					DetectedAt:  time.Now(),
					UserID:      userID,
					Confidence:  0.9,
					Events:      []*types.AuditEvent{event},
					Status:      "open",
				})
			}
		}
	}

	return anomalies
}

func (a *pgAuditAnalyzer) detectPrivilegeEscalationAnomalies(events []*types.AuditEvent) []*types.AuditAnomaly {
	anomalies := make([]*types.AuditAnomaly, 0)

	// Look for privilege escalation patterns
	for _, event := range events {
		if strings.Contains(event.Action, "role") && strings.Contains(event.Action, "assign") {
			userID := "unknown"
			if event.ActorID != nil {
				userID = event.ActorID.String()
			}

			anomalies = append(anomalies, &types.AuditAnomaly{
				ID:          uuid.New().String(),
				Type:        "privilege_escalation",
				Severity:    "MEDIUM",
				Description: fmt.Sprintf("Privilege escalation detected for user %s", userID),
				DetectedAt:  time.Now(),
				UserID:      userID,
				Confidence:  0.7,
				Events:      []*types.AuditEvent{event},
				Status:      "open",
			})
		}
	}

	return anomalies
}

func (a *pgAuditAnalyzer) detectDataAccessAnomalies(events []*types.AuditEvent) []*types.AuditAnomaly {
	anomalies := make([]*types.AuditAnomaly, 0)

	// Look for unusual data access patterns
	for _, event := range events {
		if strings.Contains(event.Action, "access") && event.ObjectType == "sensitive_data" {
			userID := "unknown"
			if event.ActorID != nil {
				userID = event.ActorID.String()
			}

			// Check if this user normally accesses this type of data
			normalAccess := false
			for _, otherEvent := range events {
				if otherEvent.ActorID != nil && otherEvent.ActorID.String() == userID &&
					otherEvent.ObjectType == event.ObjectType &&
					!otherEvent.Timestamp.Equal(event.Timestamp) {
					normalAccess = true
					break
				}
			}

			if !normalAccess {
				anomalies = append(anomalies, &types.AuditAnomaly{
					ID:          uuid.New().String(),
					Type:        "unusual_data_access",
					Severity:    "MEDIUM",
					Description: fmt.Sprintf("Unusual access to sensitive data by user %s", userID),
					DetectedAt:  time.Now(),
					UserID:      userID,
					Resource:    event.ObjectID.String(),
					Confidence:  0.6,
					Events:      []*types.AuditEvent{event},
					Status:      "open",
				})
			}
		}
	}

	return anomalies
}

func (a *pgAuditAnalyzer) detectTimeBasedAnomalies(events []*types.AuditEvent) []*types.AuditAnomaly {
	anomalies := make([]*types.AuditAnomaly, 0)

	// Look for activity at unusual hours
	for _, event := range events {
		hour := event.Timestamp.Hour()
		if hour < 6 || hour > 22 { // Outside normal business hours
			userID := "unknown"
			if event.ActorID != nil {
				userID = event.ActorID.String()
			}

			anomalies = append(anomalies, &types.AuditAnomaly{
				ID:          uuid.New().String(),
				Type:        "off_hours_activity",
				Severity:    "LOW",
				Description: fmt.Sprintf("Off-hours activity by user %s at %s", userID, event.Timestamp.Format("15:04")),
				DetectedAt:  time.Now(),
				UserID:      userID,
				Confidence:  0.4,
				Events:      []*types.AuditEvent{event},
				Status:      "open",
			})
		}
	}

	return anomalies
}

// Utility functions

func (a *pgAuditAnalyzer) groupEventsByUserSession(events []*types.AuditEvent) map[string][]*types.AuditEvent {
	sessions := make(map[string][]*types.AuditEvent)

	for _, event := range events {
		userID := "unknown"
		if event.ActorID != nil {
			userID = event.ActorID.String()
		}

		// Simple session grouping: events within 30 minutes of each other
		sessionKey := ""
		for key, sessionEvents := range sessions {
			if strings.HasPrefix(key, userID+"_") {
				// Check if event fits in this session
				lastEvent := sessionEvents[len(sessionEvents)-1]
				if event.Timestamp.Sub(lastEvent.Timestamp) < 30*time.Minute {
					sessionKey = key
					break
				}
			}
		}

		if sessionKey == "" {
			sessionKey = fmt.Sprintf("%s_%d", userID, len(sessions))
		}

		sessions[sessionKey] = append(sessions[sessionKey], event)
	}

	return sessions
}

func (a *pgAuditAnalyzer) groupEventsByResource(events []*types.AuditEvent) map[string][]*types.AuditEvent {
	groups := make(map[string][]*types.AuditEvent)

	for _, event := range events {
		resourceID := event.ObjectID.String()
		groups[resourceID] = append(groups[resourceID], event)
	}

	return groups
}

func (a *pgAuditAnalyzer) groupEventsByTimeBuckets(events []*types.AuditEvent, granularity time.Duration) map[string][]*types.AuditEvent {
	buckets := make(map[string][]*types.AuditEvent)

	for _, event := range events {
		// Round timestamp to granularity
		bucket := event.Timestamp.Truncate(granularity)
		bucketKey := bucket.Format(time.RFC3339)
		buckets[bucketKey] = append(buckets[bucketKey], event)
	}

	return buckets
}

func (a *pgAuditAnalyzer) calculateTrend(values []int) struct {
	direction    string
	rate         float64
	significance float64
} {
	if len(values) < 2 {
		return struct {
			direction    string
			rate         float64
			significance float64
		}{"stable", 0, 0}
	}

	// Simple linear regression
	n := float64(len(values))
	sumX := 0.0
	sumY := 0.0
	sumXY := 0.0
	sumXX := 0.0

	for i, val := range values {
		x := float64(i)
		y := float64(val)
		sumX += x
		sumY += y
		sumXY += x * y
		sumXX += x * x
	}

	slope := (n*sumXY - sumX*sumY) / (n*sumXX - sumX*sumX)

	direction := "stable"
	if math.Abs(slope) < 0.1 {
		direction = "stable"
	} else if slope > 0 {
		direction = "increasing"
	} else {
		direction = "decreasing"
	}

	// Calculate significance (simplified)
	mean := sumY / n
	ssRes := 0.0
	for i, val := range values {
		predicted := slope*float64(i) + (sumY-slope*sumX)/n
		ssRes += math.Pow(float64(val)-predicted, 2)
	}
	rSquared := 1 - (ssRes / (n * mean * mean))

	return struct {
		direction    string
		rate         float64
		significance float64
	}{direction, slope, rSquared}
}

func (a *pgAuditAnalyzer) convertToTrendDataPoints(times []time.Time, values []int) []*types.TrendDataPoint {
	points := make([]*types.TrendDataPoint, len(times))
	for i, t := range times {
		points[i] = &types.TrendDataPoint{
			Timestamp: t,
			Value:     float64(values[i]),
		}
	}
	return points
}

func (a *pgAuditAnalyzer) calculateRiskFactors(events []*types.AuditEvent) []*types.RiskFactor {
	factors := make([]*types.RiskFactor, 0)

	// Failed access factor
	failedCount := 0
	for _, event := range events {
		if strings.Contains(event.Action, "failed") || strings.Contains(event.Action, "denied") {
			failedCount++
		}
	}
	if failedCount > 0 {
		factors = append(factors, &types.RiskFactor{
			Name:        "failed_access_attempts",
			Description: fmt.Sprintf("%d failed access attempts", failedCount),
			Score:       float64(failedCount) * 0.5,
		})
	}

	// Privilege changes factor
	privilegeCount := 0
	for _, event := range events {
		if strings.Contains(event.Action, "role") || strings.Contains(event.Action, "permission") {
			privilegeCount++
		}
	}
	if privilegeCount > 0 {
		factors = append(factors, &types.RiskFactor{
			Name:        "privilege_changes",
			Description: fmt.Sprintf("%d privilege changes", privilegeCount),
			Score:       float64(privilegeCount) * 2.0,
		})
	}

	// Sensitive data access factor
	sensitiveCount := 0
	for _, event := range events {
		if event.ObjectType == "sensitive_data" || event.ObjectType == "user_data" {
			sensitiveCount++
		}
	}
	if sensitiveCount > 0 {
		factors = append(factors, &types.RiskFactor{
			Name:        "sensitive_data_access",
			Description: fmt.Sprintf("%d sensitive data accesses", sensitiveCount),
			Score:       float64(sensitiveCount) * 1.5,
		})
	}

	return factors
}

func (a *pgAuditAnalyzer) analyzeSessionForCorrelation(sessionID string, events []*types.AuditEvent) *types.EventCorrelation {
	if len(events) < 3 {
		return nil
	}

	// Look for suspicious patterns: many failed attempts followed by success
	failed := 0
	success := 0

	for _, event := range events {
		if strings.Contains(event.Action, "failed") {
			failed++
		} else if strings.Contains(event.Action, "success") || strings.Contains(event.Action, "login") {
			success++
		}
	}

	if failed > success*2 && success > 0 {
		return &types.EventCorrelation{
			ID:           uuid.New().String(),
			Type:         "brute_force_success",
			Description:  fmt.Sprintf("Session %s shows pattern of failed attempts followed by success", sessionID),
			Severity:     "HIGH",
			Events:       events,
			CorrelatedAt: time.Now(),
		}
	}

	return nil
}

func (a *pgAuditAnalyzer) extractNormalPatterns(events []*types.AuditEvent) map[string]interface{} {
	patterns := make(map[string]interface{})

	if len(events) == 0 {
		return patterns
	}

	// Analyze typical hours
	hours := make(map[int]int)
	for _, event := range events {
		hours[event.Timestamp.Hour()]++
	}

	// Find most common hours
	var commonHours []int
	for hour, count := range hours {
		if count > len(events)/10 { // At least 10% of activity
			commonHours = append(commonHours, hour)
		}
	}
	sort.Ints(commonHours)
	patterns["typical_hours"] = commonHours

	// Analyze typical actions
	actions := make(map[string]int)
	for _, event := range events {
		actions[event.Action]++
	}

	var commonActions []string
	for action, count := range actions {
		if count > len(events)/5 { // At least 20% of activity
			commonActions = append(commonActions, action)
		}
	}
	patterns["typical_actions"] = commonActions

	return patterns
}

func (a *pgAuditAnalyzer) compareActivityToBaseline(activity []*types.AuditEvent, baseline *types.BehaviorBaseline) []*types.Deviation {
	deviations := make([]*types.Deviation, 0)

	patterns := baseline.NormalPatterns

	// Check for unusual hours
	if typicalHours, ok := patterns["typical_hours"].([]int); ok {
		for _, event := range activity {
			hour := event.Timestamp.Hour()
			isTypical := false
			for _, typical := range typicalHours {
				if hour == typical {
					isTypical = true
					break
				}
			}

			if !isTypical {
				deviations = append(deviations, &types.Deviation{
					Type:        "unusual_hour",
					Description: fmt.Sprintf("Activity at unusual hour: %d", hour),
					Severity:    "LOW",
					Event:       event,
				})
			}
		}
	}

	// Check for unusual actions
	if typicalActions, ok := patterns["typical_actions"].([]string); ok {
		for _, event := range activity {
			isTypical := false
			for _, typical := range typicalActions {
				if event.Action == typical {
					isTypical = true
					break
				}
			}

			if !isTypical {
				deviations = append(deviations, &types.Deviation{
					Type:        "unusual_action",
					Description: fmt.Sprintf("Unusual action: %s", event.Action),
					Severity:    "MEDIUM",
					Event:       event,
				})
			}
		}
	}

	return deviations
}

func (a *pgAuditAnalyzer) analyzeEventVolumeTrend(buckets map[string][]*types.AuditEvent) []*types.AuditTrend {
	// Convert to time series
	type bucketData struct {
		time  time.Time
		count int
	}

	var data []bucketData
	for timeStr, events := range buckets {
		if t, err := time.Parse(time.RFC3339, timeStr); err == nil {
			data = append(data, bucketData{t, len(events)})
		}
	}

	sort.Slice(data, func(i, j int) bool {
		return data[i].time.Before(data[j].time)
	})

	// Calculate trend
	values := make([]int, len(data))
	for i, d := range data {
		values[i] = d.count
	}

	trend := a.calculateTrend(values)

	var dataPoints []*types.TrendDataPoint
	for _, d := range data {
		dataPoints = append(dataPoints, &types.TrendDataPoint{
			Timestamp: d.time,
			Value:     float64(d.count),
		})
	}

	return []*types.AuditTrend{
		{
			Metric:       "event_volume",
			Granularity:  24 * time.Hour,
			DataPoints:   dataPoints,
			Direction:    trend.direction,
			ChangeRate:   trend.rate,
			Significance: trend.significance,
		},
	}
}

func (a *pgAuditAnalyzer) analyzeErrorRateTrend(_ map[string][]*types.AuditEvent) []*types.AuditTrend {
	// Similar implementation for error rate trends
	return []*types.AuditTrend{}
}

func (a *pgAuditAnalyzer) analyzeUserActivityTrend(_ map[string][]*types.AuditEvent) []*types.AuditTrend {
	// Similar implementation for user activity trends
	return []*types.AuditTrend{}
}

// correlateEvents correlates related audit events
func (a *pgAuditAnalyzer) correlateEvents(_ context.Context, events []*types.AuditEvent) []*types.EventCorrelation {
	correlations := make([]*types.EventCorrelation, 0)

	// Group events by user session (same user, close in time)
	userSessions := a.groupEventsByUserSession(events)

	for sessionID, sessionEvents := range userSessions {
		if len(sessionEvents) < 2 {
			continue
		}

		// Look for suspicious patterns in the session
		correlation := a.analyzeSessionForCorrelation(sessionID, sessionEvents)
		if correlation != nil {
			correlations = append(correlations, correlation)
		}
	}

	// Look for events affecting the same resource
	resourceGroups := a.groupEventsByResource(events)
	for resourceID, resourceEvents := range resourceGroups {
		if len(resourceEvents) > 5 { // Multiple accesses to same resource
			correlation := &types.EventCorrelation{
				ID:           uuid.New().String(),
				Type:         "resource_access_cluster",
				Description:  fmt.Sprintf("Multiple accesses to resource %s", resourceID),
				Severity:     "LOW",
				Events:       resourceEvents,
				CorrelatedAt: time.Now(),
			}
			correlations = append(correlations, correlation)
		}
	}

	return correlations
}

// calculateRiskIndicators calculates risk indicators from audit events
func (a *pgAuditAnalyzer) calculateRiskIndicators(events []*types.AuditEvent) []*types.RiskIndicator {
	indicators := make([]*types.RiskIndicator, 0)

	// Calculate failed access risk
	failedCount := 0
	totalCount := len(events)

	for _, event := range events {
		if strings.Contains(event.Action, "failed") || strings.Contains(event.Action, "denied") {
			failedCount++
		}
	}

	if totalCount > 0 {
		failureRate := float64(failedCount) / float64(totalCount)
		if failureRate > 0.1 { // More than 10% failures
			indicators = append(indicators, &types.RiskIndicator{
				Type:        "high_failure_rate",
				Description: fmt.Sprintf("High failure rate: %.1f%% (%d/%d)", failureRate*100, failedCount, totalCount),
				Severity:    "MEDIUM",
				Score:       failureRate * 100,
			})
		}
	}

	// Calculate privilege escalation risk
	privilegeCount := 0
	for _, event := range events {
		if strings.Contains(event.Action, "role") && strings.Contains(event.Action, "assign") {
			privilegeCount++
		}
	}

	if privilegeCount > 0 {
		indicators = append(indicators, &types.RiskIndicator{
			Type:        "privilege_changes",
			Description: fmt.Sprintf("%d privilege changes detected", privilegeCount),
			Severity:    "HIGH",
			Score:       float64(privilegeCount) * 10,
		})
	}

	return indicators
}

