package scanner

import (
	"context"
	"io"
	"strings"
	"time"

	"log/slog"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/rules"
	"github.com/promptshield/promptshield/internal/scanner"
	"github.com/promptshield/promptshield/internal/shared/contracts"
	"github.com/promptshield/promptshield/internal/shared/types"
)

// Service provides LLM Gateway scanning capabilities
type Service struct {
	scanner     *scanner.Scanner
	auditLogger contracts.AuditLogger
	logger      *slog.Logger
}

// NewService creates a new gateway scanner service
func NewService(auditLogger contracts.AuditLogger, logger *slog.Logger) *Service {
	scanner := CreateGatewayScanner()

	// Configure scanner with logging and tracing
	scanner.SetLogger(logger)

	return &Service{
		scanner:     scanner,
		auditLogger: auditLogger,
		logger:      logger,
	}
}

// LoadRulePacks loads rule packs into the scanner
func (s *Service) LoadRulePacks(packs []rules.RulePack) {
	s.scanner.LoadRulePacks(packs)
	s.logger.Info("loaded rule packs", "count", len(packs))
}

// ScanLLMRequest scans an LLM request for violations
func (s *Service) ScanLLMRequest(ctx context.Context, request *LLMRequest) (*types.ScanResult, error) {
	start := time.Now()

	// Extract content from LLM request
	content := s.extractRequestContent(request)

	// Set runtime context for tenant/user info
	s.scanner.SetRuntimeContext(map[string]string{
		"tenant_id": request.TenantID,
		"user_id":   request.UserID,
		"model":     request.Model,
		"provider":  request.Provider,
	})

	// Scan the content
	scannerResult, err := s.scanner.ScanReader(ctx, strings.NewReader(content), "llm-request")
	if err != nil {
		s.logger.Error("scanner error", "error", err, "request_id", request.RequestID)
		return nil, err
	}

	// Convert to shared types
	tenantID, _ := uuid.Parse(request.TenantID)
	result := ConvertScanResult(&scannerResult, &tenantID, request.RequestID)

	// Log audit event
	s.logAuditEvent(ctx, request, result, time.Since(start))

	s.logger.Info("scanned LLM request",
		"request_id", request.RequestID,
		"violations", len(result.Violations),
		"duration_ms", time.Since(start).Milliseconds(),
		"should_block", !result.Decision.Allow,
	)

	return result, nil
}

// ScanLLMResponse scans an LLM response for violations
func (s *Service) ScanLLMResponse(ctx context.Context, response *LLMResponse) (*types.ScanResult, error) {
	start := time.Now()

	// Extract content from LLM response
	content := s.extractResponseContent(response)

	// Set runtime context
	s.scanner.SetRuntimeContext(map[string]string{
		"tenant_id": response.TenantID,
		"user_id":   response.UserID,
		"model":     response.Model,
		"provider":  response.Provider,
		"type":      "response",
	})

	// Scan the content
	scannerResult, err := s.scanner.ScanReader(ctx, strings.NewReader(content), "llm-response")
	if err != nil {
		s.logger.Error("scanner error", "error", err, "request_id", response.RequestID)
		return nil, err
	}

	// Convert to shared types
	tenantID, _ := uuid.Parse(response.TenantID)
	result := ConvertScanResult(&scannerResult, &tenantID, response.RequestID)

	// Log audit event
	s.logAuditEvent(ctx, nil, result, time.Since(start))

	s.logger.Info("scanned LLM response",
		"request_id", response.RequestID,
		"violations", len(result.Violations),
		"duration_ms", time.Since(start).Milliseconds(),
		"should_block", !result.Decision.Allow,
	)

	return result, nil
}

// ScanStream scans a streaming LLM response
func (s *Service) ScanStream(ctx context.Context, stream io.Reader, requestID, tenantID string) (*types.ScanResult, error) {
	start := time.Now()

	// Set runtime context
	s.scanner.SetRuntimeContext(map[string]string{
		"tenant_id": tenantID,
		"type":      "stream",
	})

	// Scan the stream
	scannerResult, err := s.scanner.ScanReader(ctx, stream, "llm-stream")
	if err != nil {
		s.logger.Error("stream scanner error", "error", err, "request_id", requestID)
		return nil, err
	}

	// Convert to shared types
	parsedTenantID, _ := uuid.Parse(tenantID)
	result := ConvertScanResult(&scannerResult, &parsedTenantID, requestID)

	s.logger.Info("scanned LLM stream",
		"request_id", requestID,
		"violations", len(result.Violations),
		"duration_ms", time.Since(start).Milliseconds(),
		"should_block", !result.Decision.Allow,
	)

	return result, nil
}

// extractRequestContent extracts content from LLM request
func (s *Service) extractRequestContent(request *LLMRequest) string {
	var content strings.Builder

	// Add system prompt if present
	if request.SystemPrompt != "" {
		content.WriteString("System: ")
		content.WriteString(request.SystemPrompt)
		content.WriteString("\n\n")
	}

	// Add user messages
	for _, msg := range request.Messages {
		content.WriteString(msg.Role)
		content.WriteString(": ")
		content.WriteString(msg.Content)
		content.WriteString("\n\n")
	}

	return content.String()
}

// extractResponseContent extracts content from LLM response
func (s *Service) extractResponseContent(response *LLMResponse) string {
	var content strings.Builder

	// Add response content
	if response.Content != "" {
		content.WriteString(response.Content)
	}

	// Add function calls if present
	for _, call := range response.FunctionCalls {
		content.WriteString("\nFunction Call: ")
		content.WriteString(call.Name)
		content.WriteString("(")
		content.WriteString(call.Arguments)
		content.WriteString(")")
	}

	return content.String()
}

// logAuditEvent logs an audit event for the scan
func (s *Service) logAuditEvent(ctx context.Context, request *LLMRequest, result *types.ScanResult, duration time.Duration) {
	if s.auditLogger == nil {
		return
	}

	action := "llm_request_scanned"
	if len(result.Violations) > 0 {
		action = "llm_request_violation_detected"
	}

	// Parse request ID as UUID for audit event
	requestUUID, _ := uuid.Parse(result.RequestID)

	event := types.AuditEvent{
		Action:     action,
		ObjectType: "llm_request",
		ObjectID:   requestUUID,
		Timestamp:  time.Now(),
		Metadata: map[string]interface{}{
			"tenant_id":        result.TenantID.String(),
			"violations_count": len(result.Violations),
			"scan_duration_ms": duration.Milliseconds(),
			"should_block":     !result.Decision.Allow,
			"highest_severity": result.ScanInfo.HighestSeverity,
		},
	}

	if request != nil {
		event.Metadata["model"] = request.Model
		event.Metadata["provider"] = request.Provider
		event.Metadata["user_id"] = request.UserID
	}

	_ = s.auditLogger.LogWithContext(ctx, event)
}

// LLMRequest represents an LLM request to be scanned
type LLMRequest struct {
	RequestID    string
	TenantID     string
	UserID       string
	Model        string
	Provider     string
	SystemPrompt string
	Messages     []Message
}

// LLMResponse represents an LLM response to be scanned
type LLMResponse struct {
	RequestID     string
	TenantID      string
	UserID        string
	Model         string
	Provider      string
	Content       string
	FunctionCalls []FunctionCall
}

// Message represents a message in an LLM conversation
type Message struct {
	Role    string
	Content string
}

// FunctionCall represents a function call in an LLM response
type FunctionCall struct {
	Name      string
	Arguments string
}
