package grpcenforcer

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	extproc "github.com/envoyproxy/go-control-plane/envoy/service/ext_proc/v3"
	typev3 "github.com/envoyproxy/go-control-plane/envoy/type/v3"

	"github.com/google/uuid"
	health "google.golang.org/grpc/health"

	"github.com/promptshield/promptshield/internal/contracts"
	nats "github.com/promptshield/promptshield/internal/infrastructure/messaging/nats"
	"github.com/promptshield/promptshield/internal/scanner"
	"github.com/promptshield/promptshield/internal/shared/severity"
	pkgtypes "github.com/promptshield/promptshield/pkg/types"
	"golang.org/x/time/rate"
)

// TelemetryCollector is a minimal, privacy-first event emitter used by the enforcer.
// Implemented by internal/observability/telemetry.Collector.
type TelemetryCollector interface {
	Collect(eventType string, payload map[string]any)
}

// Server implements Envoy ext_proc ExternalProcessor for streaming decisions.
type Server struct {
	// Embed for forward compatibility with generated interface requirements
	extproc.UnimplementedExternalProcessorServer
	timeout         time.Duration
	maxStreamBytes  int64
	failOn          string
	rulepackPath    string
	scReq           *scanner.Scanner
	scResp          *scanner.Scanner
	telemetry       TelemetryCollector
	enforcementMode string
	healthSrv       *health.Server
	ready           bool
	// streaming performance controls
	windowLimit int
	overlap     int
	// tails for cross-chunk matching
	tailReq  []byte
	tailResp []byte
	// basic rate limiter (global)
	limiter *rate.Limiter
	// global concurrency & memory budgets
	streamSlots     chan struct{}
	inflightBytes   int64
	inflightLimit   int64
	inflightBackoff time.Duration

	// runtime rule loading
	rulepackRepo contracts.RulepackRepository
	tenantID     uuid.UUID
	rulesMutex   sync.RWMutex // protects rule reloading

	// live rule updates
	subscriber *nats.Subscriber
}

// constructor moved to constructor.go

// Process implements the Envoy External Processor with full streaming analysis and enforcement.
func (s *Server) Process(streamParam extproc.ExternalProcessor_ProcessServer) error {
	// Delegate to extracted implementation in process_stream.go
	return s.processStream(streamParam)
}

// Run wrapper removed; use internal/enforcergrpc helpers instead.

/* legacy implementation moved to run_server.go */

func header(k, v string) *corev3.HeaderValueOption {
	return &corev3.HeaderValueOption{Header: &corev3.HeaderValue{Key: k, RawValue: []byte(v)}}
}

func sendImmediateResponse(stream extproc.ExternalProcessor_ProcessServer, decision, reason string) error {
	return sendImmediateResponseWithDetails(stream, decision, reason, nil)
}

func sendImmediateResponseWithDetails(stream extproc.ExternalProcessor_ProcessServer, decision, reason string, scanResult *pkgtypes.ScanResult) error {
	var body []byte

	// Debug logging
	logger := slog.With("component","grpc-enforcer")
	if scanResult != nil {
		logger.Debug("sendImmediateResponseWithDetails called", "violations", len(scanResult.Violations))
	} else {
		logger.Debug("sendImmediateResponseWithDetails called with nil scanResult")
	}

	if scanResult != nil && len(scanResult.Violations) > 0 {
		// Create detailed JSON response with violation information
		response := map[string]interface{}{
			"blocked":    true,
			"decision":   decision,
			"reason":     reason,
			"message":    "Request blocked by PromptShield",
			"violations": make([]map[string]interface{}, len(scanResult.Violations)),
			"scan_info": map[string]interface{}{
				"total_violations": len(scanResult.Violations),
				"scan_duration_ms": scanResult.DurationMs,
			},
		}

		for i, violation := range scanResult.Violations {
			response["violations"].([]map[string]interface{})[i] = map[string]interface{}{
				"rule_id":  violation.RuleID,
				"severity": violation.Severity,
				"category": violation.Category,
				"message":  violation.Message,
				"line":     violation.Line,
				"column":   violation.Column,
			}
		}

		if jsonBody, err := json.Marshal(response); err == nil {
			body = jsonBody
		} else {
			body = []byte(`{"blocked":true,"decision":"` + decision + `","reason":"` + reason + `","message":"Request blocked by PromptShield"}`)
		}
	} else {
		// Fallback to simple JSON response
		body = []byte(`{"blocked":true,"decision":"` + decision + `","reason":"` + reason + `","message":"Request blocked by PromptShield"}`)
	}

	resp := &extproc.ImmediateResponse{
		Status: &typev3.HttpStatus{Code: typev3.StatusCode_Forbidden},
		Headers: &extproc.HeaderMutation{SetHeaders: []*corev3.HeaderValueOption{
			header("x-ps-decision", decision),
			header("x-ps-reason", reason),
			header("content-type", "application/json"),
		}},
		Body: body,
	}
	return stream.Send(&extproc.ProcessingResponse{Response: &extproc.ProcessingResponse_ImmediateResponse{ImmediateResponse: resp}})
}

// sendImmediateReplacementResponse terminates processing with a 200 OK and the provided body.
// Envoy will return this body to the client, replacing the upstream response.
func sendImmediateReplacementResponse(stream extproc.ExternalProcessor_ProcessServer, body string, reason string) error {
	// Default replacement body content if empty
	if body == "" {
		body = "[CONTENT_REPLACED]"
	}
	resp := &extproc.ImmediateResponse{
		Status:  &typev3.HttpStatus{Code: typev3.StatusCode_OK},
		Headers: &extproc.HeaderMutation{SetHeaders: []*corev3.HeaderValueOption{header("x-ps-decision", "replace"), header("x-ps-reason", reason)}},
		Body:    []byte(body),
	}
	return stream.Send(&extproc.ProcessingResponse{Response: &extproc.ProcessingResponse_ImmediateResponse{ImmediateResponse: resp}})
}

func hasThresholdHit(r pkgtypes.ScanResult, threshold string) bool {
	if threshold == "" {
		return false
	}
	for _, v := range r.Violations {
		if severity.MeetsThreshold(v.Severity, threshold) {
			return true
		}
	}
	return false
}

func firstReason(r pkgtypes.ScanResult) string {
	if len(r.Violations) == 0 {
		return "signals_detected"
	}
	return r.Violations[0].RuleID
}

// ReloadRules reloads rules from the database for this enforcer instance
func (s *Server) ReloadRules(ctx context.Context) error {
	if s.rulepackRepo == nil || s.tenantID == uuid.Nil {
		return nil // No database configured, nothing to reload
	}

	s.rulesMutex.Lock()
	defer s.rulesMutex.Unlock()

	// Load rules from database
	packs, err := LoadRulesFromDatabase(ctx, s.rulepackRepo, s.tenantID)
	if err != nil {
		return err
	}

	// Reload rules into scanners
	if len(packs) > 0 {
		s.scReq.LoadRulePacks(packs)
		s.scResp.LoadRulePacks(packs)
		logger := slog.With("component","grpc-enforcer")
		logger.Info("Reloaded rulepacks from database", "count", len(packs))
	}

	return nil
}

// Shutdown gracefully shuts down the enforcer, including the rule update subscriber
func (s *Server) Shutdown() {
	if s.subscriber != nil {
		s.subscriber.Close()
	}
}

/* helper functions moved to run_server.go
func findGRPCTLSPair() (string, string, bool) { return "","",false }
func grpcTLSMode() string { return "auto" }
func isLoopbackAddr(addr string) bool { return false }
*/

/* metrics moved to metrics.go */
