package enforcerhttp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/shared/events"
	"github.com/promptshield/promptshield/internal/shared/types"
	"github.com/promptshield/promptshield/internal/util/tracing"
	"go.opentelemetry.io/otel"
)

// PolicySyncClient syncs policies from the frontend database
type PolicySyncClient struct {
	frontendURL string
	adminToken  string
	logger      *slog.Logger
	httpClient  *http.Client
}

var policySyncTracer = otel.Tracer("promptshield/policy_sync")

// NewPolicySyncClient creates a new policy sync client
func NewPolicySyncClient() *PolicySyncClient {
	frontendURL := os.Getenv("PS_FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:3000" // Default frontend URL
	}

	adminToken := os.Getenv("PS_FRONTEND_TOKEN")
	if adminToken == "" {
		adminToken = os.Getenv("PS_ENFORCER_ADMIN_TOKEN")
	}

	return &PolicySyncClient{
		frontendURL: frontendURL,
		adminToken:  adminToken,
		logger:      slog.With("component", "policy-sync"),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// SyncActivePolicies fetches active policies from frontend and loads them
func (c *PolicySyncClient) SyncActivePolicies(ctx context.Context) error {
	c.logger.Info("Starting policy sync from frontend database")

	// Skip sync if frontend URL is not configured
	if c.frontendURL == "" {
		c.logger.Debug("Frontend URL not configured, skipping policy sync")
		return nil
	}

	// Fetch active policies from frontend
	policies, err := c.fetchActivePolicies(ctx)
	if err != nil {
		// Don't fail startup if frontend is not available
		c.logger.Warn("Failed to fetch active policies from frontend", "error", err)
		return nil
	}

	if len(policies) == 0 {
		c.logger.Info("No active policies found in frontend database")
		return nil
	}

	c.logger.Info("Found active policies to sync", "count", len(policies))

	// Publish PolicyActivated events for each active policy
	eventBus := events.GlobalEventBus()
	activated := 0

	for _, policy := range policies {
		// Parse policy ID
		policyID, err := uuid.Parse(policy.ID)
		if err != nil {
			c.logger.Warn("Invalid policy ID, skipping", "id", policy.ID, "error", err)
			continue
		}

		// Create PolicyActivated event
		event := &events.PolicyActivated{
			BaseEvent: events.BaseEvent{
				ID:   uuid.New().String(),
				Type: events.EventTypePolicyActivated,
				Time: time.Now(),
			},
			PolicyID: policyID,
			PolicyData: types.Policy{
				ID:        policyID,
				Name:      policy.Name,
				Content:   policy.Content,
				Type:      types.PolicyType(policy.Type),
				Version:   policy.Version,
				CreatedAt: policy.CreatedAt,
				UpdatedAt: policy.UpdatedAt,
			},
		}

		// Publish event to load policy into scanner
		if err := eventBus.PublishSync(ctx, event); err != nil {
			c.logger.Error("Failed to activate policy on startup",
				"policy_id", policyID,
				"policy_name", policy.Name,
				"error", err)
			continue
		}

		activated++
		c.logger.Debug("Activated policy from frontend",
			"policy_id", policyID,
			"policy_name", policy.Name)
	}

	c.logger.Info("Policy sync completed",
		"total_found", len(policies),
		"activated", activated)

	return nil
}

// fetchActivePolicies retrieves active policies from the frontend API
func (c *PolicySyncClient) fetchActivePolicies(ctx context.Context) ([]PolicyResponse, error) {
	url := fmt.Sprintf("%s/api/policies", c.frontendURL)

	ctx, span := tracing.TraceHTTPRequest(policySyncTracer, ctx, http.MethodGet, "/api/policies")
	defer span.End()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication if configured
	if c.adminToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.adminToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch policies: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("frontend returned status %d: %s", resp.StatusCode, string(body))
	}

	var policies []PolicyResponse
	if err := json.NewDecoder(resp.Body).Decode(&policies); err != nil {
		return nil, fmt.Errorf("failed to decode policies: %w", err)
	}

	// Filter only active policies
	var activePolicies []PolicyResponse
	for _, policy := range policies {
		if policy.IsActive {
			activePolicies = append(activePolicies, policy)
		}
	}

	return activePolicies, nil
}

// PolicyResponse represents the policy data from frontend API
type PolicyResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Content   string    `json:"content"`
	Version   int       `json:"version"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// StartPolicySync initializes policy sync on startup
func StartPolicySync(ctx context.Context) {
	// Only sync if enabled
	if os.Getenv("PS_DISABLE_POLICY_SYNC") == "1" {
		return
	}

	// Create sync client
	client := NewPolicySyncClient()

	// Perform initial sync with retry
	go func() {
		// Wait a bit for services to be ready
		time.Sleep(2 * time.Second)

		// Try to sync with retries
		maxRetries := 3
		for i := 0; i < maxRetries; i++ {
			if err := client.SyncActivePolicies(ctx); err == nil {
				break
			}

			if i < maxRetries-1 {
				// Wait before retry
				time.Sleep(time.Duration(i+1) * 5 * time.Second)
			}
		}
	}()

	// Optionally set up periodic sync
	if os.Getenv("PS_ENABLE_PERIODIC_SYNC") == "1" {
		go func() {
			ticker := time.NewTicker(5 * time.Minute)
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					_ = client.SyncActivePolicies(ctx)
				case <-ctx.Done():
					return
				}
			}
		}()
	}
}
