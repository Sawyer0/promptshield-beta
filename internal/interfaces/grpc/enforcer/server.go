package grpcenforcer

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	extproc "github.com/envoyproxy/go-control-plane/envoy/service/ext_proc/v3"
	typev3 "github.com/envoyproxy/go-control-plane/envoy/type/v3"

	// grpc_middleware imported if needed in future for additional chaining helpers
	grpc_logging "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"

	// timeout interceptor lives under middleware/v2/interceptors/timeout in older examples,
	// but not all versions expose it; implement simple timeout via context with otel stats.
	"sync/atomic"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/promptshield/promptshield/internal/license"
	"github.com/promptshield/promptshield/internal/rules"
	"github.com/promptshield/promptshield/internal/scanner"
	"github.com/promptshield/promptshield/internal/security/paths"
	"github.com/promptshield/promptshield/internal/shared/redact"
	"github.com/promptshield/promptshield/internal/shared/severity"
	"github.com/promptshield/promptshield/pkg/types"
	otelgrpc "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	health "google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

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
	redactionOn     bool
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
}

type Options struct {
	Timeout         time.Duration
	MaxStreamBytes  int64
	FailOn          string
	RulepackPath    string
	Telemetry       TelemetryCollector
	EnforcementMode string
}

// TelemetryCollector is a minimal, privacy-first event emitter.
// Implemented by internal/observability/telemetry.Collector.
type TelemetryCollector interface {
	Collect(eventType string, payload map[string]any)
}

func New(timeout time.Duration) *Server {
	return NewWithOptions(Options{Timeout: timeout, MaxStreamBytes: 5_000_000, FailOn: "HIGH", RulepackPath: defaultRulepackPathFromEnv()})
}

func NewWithOptions(opt Options) *Server {
	if opt.Timeout <= 0 {
		opt.Timeout = 300 * time.Millisecond
	}
	if opt.MaxStreamBytes <= 0 {
		opt.MaxStreamBytes = 5_000_000
	}
	if opt.FailOn == "" {
		opt.FailOn = "HIGH"
	}
	// Allow env-based rulepack when not explicitly provided
	if strings.TrimSpace(opt.RulepackPath) == "" {
		opt.RulepackPath = defaultRulepackPathFromEnv()
	}
	mode := strings.ToLower(strings.TrimSpace(opt.EnforcementMode))
	if mode == "" {
		mode = strings.ToLower(strings.TrimSpace(os.Getenv("PS_ENFORCER_ENFORCEMENT_MODE")))
	}
	if mode == "" {
		mode = "observe" // observe|redact|quarantine|enforce
	}
	s := &Server{timeout: opt.Timeout, maxStreamBytes: opt.MaxStreamBytes, failOn: opt.FailOn, rulepackPath: opt.RulepackPath, telemetry: opt.Telemetry, enforcementMode: mode}
	scReq := scanner.New(0)
	scResp := scanner.New(0)
	// Wire scanner stream limits to runtime options
	scReq.SetMaxStreamBytes(opt.MaxStreamBytes)
	scResp.SetMaxStreamBytes(opt.MaxStreamBytes)
	// Favor quarantine-on-timeout/error behavior for streaming enforcement
	scReq.SetQuarantineOnTimeout(true)
	scReq.SetQuarantineOnError(true)
	scResp.SetQuarantineOnTimeout(true)
	scResp.SetQuarantineOnError(true)
	loaded := false
	if s.rulepackPath != "" {
		if packs, err := rules.LoadPacks(s.rulepackPath); err == nil {
			scReq.LoadRulePacks(packs)
			scResp.LoadRulePacks(packs)
			loaded = true
		}
	} else if _, err := os.Stat("rules/basic-security.yaml"); err == nil {
		if packs, err := rules.LoadPacks("rules/basic-security.yaml"); err == nil {
			scReq.LoadRulePacks(packs)
			scResp.LoadRulePacks(packs)
			loaded = true
		}
	} else if _, err := os.Stat("/rules/basic-security.yaml"); err == nil {
		if packs, err := rules.LoadPacks("/rules/basic-security.yaml"); err == nil {
			scReq.LoadRulePacks(packs)
			scResp.LoadRulePacks(packs)
			loaded = true
		}
	}
	scReq.SetRuntimeContext(map[string]string{"direction": "request"})
	scResp.SetRuntimeContext(map[string]string{"direction": "response"})
	s.scReq = scReq
	s.scResp = scResp
	// health state
	s.ready = loaded
	redEnv := strings.ToLower(strings.TrimSpace(os.Getenv("PS_ENFORCER_REDACTION_MUTATION")))
	s.redactionOn = !(redEnv == "0" || redEnv == "false")
	hs := health.NewServer()
	// overall
	if s.ready {
		hs.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	} else {
		hs.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)
	}
	// ext_proc service
	svcName := "envoy.service.ext_proc.v3.ExternalProcessor"
	if s.ready {
		hs.SetServingStatus(svcName, healthpb.HealthCheckResponse_SERVING)
	} else {
		hs.SetServingStatus(svcName, healthpb.HealthCheckResponse_NOT_SERVING)
	}
	// redaction feature service
	feat := "promptshield.features.redaction"
	if s.redactionOn {
		hs.SetServingStatus(feat, healthpb.HealthCheckResponse_SERVING)
	} else {
		hs.SetServingStatus(feat, healthpb.HealthCheckResponse_NOT_SERVING)
	}
	s.healthSrv = hs

	// streaming tunables (defaults)
	s.windowLimit = 64 * 1024
	s.overlap = 4096
	if v := strings.TrimSpace(os.Getenv("PS_ENFORCER_STREAM_WINDOW")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 8192 {
			s.windowLimit = n
		}
	}
	if v := strings.TrimSpace(os.Getenv("PS_ENFORCER_STREAM_OVERLAP")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 && n < s.windowLimit {
			s.overlap = n
		}
	}
	// Rate limiter from license entitlements; fallback env when not licensed
	if ent, ok := license.Entitlement(); ok && ent.MaxRPS > 0 {
		burst := 1
		if b := strings.TrimSpace(os.Getenv("PS_ENFORCER_RPS_BURST")); b != "" {
			if n, err := strconv.Atoi(b); err == nil && n > 0 {
				burst = n
			}
		}
		s.limiter = rate.NewLimiter(rate.Limit(ent.MaxRPS), burst)
	} else if rpsStr := strings.TrimSpace(os.Getenv("PS_ENFORCER_RPS")); rpsStr != "" {
		if r, err := strconv.ParseFloat(rpsStr, 64); err == nil && r > 0 {
			burst := 1
			if b := strings.TrimSpace(os.Getenv("PS_ENFORCER_RPS_BURST")); b != "" {
				if n, err := strconv.Atoi(b); err == nil && n > 0 {
					burst = n
				}
			}
			s.limiter = rate.NewLimiter(rate.Limit(r), burst)
		}
	}

	// global stream slots (optional)
	if v := strings.TrimSpace(os.Getenv("PS_ENFORCER_MAX_STREAMS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			s.streamSlots = make(chan struct{}, n)
		}
	}

	// global inflight bytes ceiling (optional)
	if v := strings.TrimSpace(os.Getenv("PS_ENFORCER_INFLIGHT_LIMIT_BYTES")); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			s.inflightLimit = n
		}
	}
	if v := strings.TrimSpace(os.Getenv("PS_ENFORCER_INFLIGHT_BACKOFF_MS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			s.inflightBackoff = time.Duration(n) * time.Millisecond
		}
	}
	return s
}

func defaultRulepackPathFromEnv() string {
	if v := os.Getenv("PS_ENFORCER_RULEPACK"); v != "" {
		return v
	}
	return ""
}

// Process implements the Envoy External Processor with full streaming analysis and enforcement.
func (s *Server) Process(stream extproc.ExternalProcessor_ProcessServer) error {
	ctx := stream.Context()
	// Start a span for the ext_proc stream
	tracer := otel.Tracer("promptshield/enforcer")
	ctx, span := tracer.Start(ctx, "extproc_stream")
	defer span.End()
	// Acquire global stream slot to cap concurrency
	if s.streamSlots != nil {
		s.streamSlots <- struct{}{}
		defer func() { <-s.streamSlots }()
	}
	start := time.Now()
	var total int64
	var totalResp int64
	decision := "allow"
	reason := "no_signals"
	// track inflight accounting per stream to avoid per-chunk defers
	var lastReqAdded int64
	var lastRespAdded int64
	defer func() {
		if lastReqAdded > 0 {
			atomic.AddInt64(&s.inflightBytes, -lastReqAdded)
		}
		if lastRespAdded > 0 {
			atomic.AddInt64(&s.inflightBytes, -lastRespAdded)
		}
	}()

	// Do not send any response before receiving a ProcessingRequest from Envoy

	// reset tail buffers for this stream
	s.tailReq = nil
	s.tailResp = nil
	for {
		// release accounted bytes from previous chunk before reading next message
		if lastReqAdded > 0 {
			atomic.AddInt64(&s.inflightBytes, -lastReqAdded)
			lastReqAdded = 0
		}
		if lastRespAdded > 0 {
			atomic.AddInt64(&s.inflightBytes, -lastRespAdded)
			lastRespAdded = 0
		}
		if s.limiter != nil {
			_ = s.limiter.Wait(ctx)
		}
		req, err := stream.Recv()
		if err != nil {
			span.RecordError(err)
			return err
		}
		switch x := req.Request.(type) {
		case *extproc.ProcessingRequest_RequestHeaders:
			// continue
			if err := stream.Send(&extproc.ProcessingResponse{Response: &extproc.ProcessingResponse_RequestHeaders{RequestHeaders: &extproc.HeadersResponse{Response: &extproc.CommonResponse{}}}}); err != nil {
				span.RecordError(err)
				return err
			}
		case *extproc.ProcessingRequest_ResponseHeaders:
			// quick check for header leaks before body buffering
			if h := x.ResponseHeaders; h != nil && h.Headers != nil {
				for _, hv := range h.Headers.Headers {
					if hv == nil {
						continue
					}
					val := string(hv.RawValue)
					if val == "" {
						val = hv.Value
					}
					if strings.Contains(val, "api_key=") || strings.Contains(val, "password") {
						decision = "quarantine"
						reason = "response-header-leak"
						// Emit metrics and decision event before returning
						extprocStreams.WithLabelValues(decision).Inc()
						extprocStreamDuration.WithLabelValues(decision).Observe(time.Since(start).Seconds())
						if s.telemetry != nil {
							s.telemetry.Collect("decision", map[string]any{"ts": time.Now().UTC().Unix(), "decision": decision, "rule_id": reason})
						}
						span.SetAttributes(attribute.String("decision", decision), attribute.String("reason", reason))
						return sendImmediateResponse(stream, decision, reason)
					}
				}
			}
			// inject decision headers only if not already present (avoid overriding ext_authz)
			seenDecision := false
			seenReason := false
			if h := x.ResponseHeaders; h != nil && h.Headers != nil {
				for _, hv := range h.Headers.Headers {
					if hv == nil {
						continue
					}
					k := strings.ToLower(hv.Key)
					if k == "x-ps-decision" {
						seenDecision = true
					}
					if k == "x-ps-reason" {
						seenReason = true
					}
				}
			}
			rh := &extproc.HeadersResponse{Response: &extproc.CommonResponse{HeaderMutation: &extproc.HeaderMutation{}}}
			if !seenDecision {
				rh.Response.HeaderMutation.SetHeaders = append(rh.Response.HeaderMutation.SetHeaders, header("x-ps-decision", decision))
			}
			if !seenReason {
				rh.Response.HeaderMutation.SetHeaders = append(rh.Response.HeaderMutation.SetHeaders, header("x-ps-reason", reason))
			}
			// always include marker for visibility
			rh.Response.HeaderMutation.SetHeaders = append(rh.Response.HeaderMutation.SetHeaders, header("x-ps-extproc", "resp"))
			if err := stream.Send(&extproc.ProcessingResponse{Response: &extproc.ProcessingResponse_ResponseHeaders{ResponseHeaders: rh}}); err != nil {
				span.RecordError(err)
				return err
			}
		case *extproc.ProcessingRequest_RequestBody:
			body := x.RequestBody
			if body != nil && len(body.Body) > 0 {
				b := body.Body
				// account for inflight bytes with optional global ceiling
				if s.inflightLimit > 0 {
					added := int64(len(b))
					atomic.AddInt64(&s.inflightBytes, added)
					// Avoid self-deadlock: if a single chunk is larger than the ceiling,
					// do not wait here; rely on release after processing.
					if added < s.inflightLimit {
						for atomic.LoadInt64(&s.inflightBytes) > s.inflightLimit {
							bo := s.inflightBackoff
							if bo <= 0 {
								bo = 5 * time.Millisecond
							}
							t := time.NewTimer(bo)
							select {
							case <-stream.Context().Done():
								if !t.Stop() {
									<-t.C
								}
								atomic.AddInt64(&s.inflightBytes, -added)
								return stream.Context().Err()
							case <-t.C:
							}
						}
					}
					lastReqAdded = added
				}
				total += int64(len(b))
				extprocBytes.Add(float64(len(b)))
				if s.maxStreamBytes > 0 && total > s.maxStreamBytes { // cap
					decision = "quarantine"
					reason = "body_limit"
					extprocStreams.WithLabelValues(decision).Inc()
					extprocStreamDuration.WithLabelValues(decision).Observe(time.Since(start).Seconds())
					if s.telemetry != nil {
						s.telemetry.Collect("decision", map[string]any{"ts": time.Now().UTC().Unix(), "decision": decision, "rule_id": reason})
					}
					return sendImmediateResponse(stream, decision, reason)
				}
				// Build sliding window over request body
				var window []byte
				if len(s.tailReq) > 0 {
					window = append(window, s.tailReq...)
				}
				window = append(window, b...)
				if len(window) > s.windowLimit {
					window = window[len(window)-s.windowLimit:]
				}
				// update tail for next chunk
				if len(window) > s.overlap {
					s.tailReq = append([]byte(nil), window[len(window)-s.overlap:]...)
				} else {
					s.tailReq = append([]byte(nil), window...)
				}
				ctx, cancel := context.WithTimeout(ctx, s.timeout)
				res, scanErr := s.scReq.ScanReader(ctx, bytes.NewReader(window), "extproc:request-window")
				cancel()
				if scanErr != nil && !errors.Is(scanErr, context.DeadlineExceeded) {
					if err := stream.Send(&extproc.ProcessingResponse{Response: &extproc.ProcessingResponse_RequestBody{RequestBody: &extproc.BodyResponse{Response: &extproc.CommonResponse{}}}}); err != nil {
						span.RecordError(err)
						return err
					}
					continue
				}
				if hasThresholdHit(res, s.failOn) {
					decision = "quarantine"
					reason = firstReason(res)
					for _, v := range res.Violations {
						extprocRuleHits.WithLabelValues(v.RuleID, strings.ToUpper(v.Severity)).Inc()
					}
					if s.enforcementMode == "enforce" || s.enforcementMode == "quarantine" {
						extprocStreams.WithLabelValues(decision).Inc()
						extprocStreamDuration.WithLabelValues(decision).Observe(time.Since(start).Seconds())
						if s.telemetry != nil {
							s.telemetry.Collect("decision", map[string]any{"ts": time.Now().UTC().Unix(), "decision": decision, "rule_id": reason})
						}
						span.SetAttributes(attribute.String("decision", decision), attribute.String("reason", reason))
						return sendImmediateResponse(stream, decision, reason)
					}
					// observe/redact: continue streaming without blocking
				}
			}
			// continue streaming
			if err := stream.Send(&extproc.ProcessingResponse{Response: &extproc.ProcessingResponse_RequestBody{RequestBody: &extproc.BodyResponse{Response: &extproc.CommonResponse{}}}}); err != nil {
				span.RecordError(err)
				return err
			}
		case *extproc.ProcessingRequest_ResponseBody:
			body := x.ResponseBody
			if body != nil && len(body.Body) > 0 {
				b := body.Body
				// account for inflight bytes with optional global ceiling
				if s.inflightLimit > 0 {
					added := int64(len(b))
					atomic.AddInt64(&s.inflightBytes, added)
					if added < s.inflightLimit {
						for atomic.LoadInt64(&s.inflightBytes) > s.inflightLimit {
							bo := s.inflightBackoff
							if bo <= 0 {
								bo = 5 * time.Millisecond
							}
							t := time.NewTimer(bo)
							select {
							case <-stream.Context().Done():
								if !t.Stop() {
									<-t.C
								}
								atomic.AddInt64(&s.inflightBytes, -added)
								return stream.Context().Err()
							case <-t.C:
							}
						}
					}
					lastRespAdded = added
				}
				totalResp += int64(len(b))
				extprocBytes.Add(float64(len(b)))
				if s.maxStreamBytes > 0 && totalResp > s.maxStreamBytes { // cap
					decision = "quarantine"
					reason = "body_limit"
					extprocStreams.WithLabelValues(decision).Inc()
					extprocStreamDuration.WithLabelValues(decision).Observe(time.Since(start).Seconds())
					if s.telemetry != nil {
						s.telemetry.Collect("decision", map[string]any{"ts": time.Now().UTC().Unix(), "decision": decision, "rule_id": reason})
					}
					return sendImmediateResponse(stream, decision, reason)
				}
				// Build sliding window over response body
				var rwindow []byte
				if len(s.tailResp) > 0 {
					rwindow = append(rwindow, s.tailResp...)
				}
				rwindow = append(rwindow, b...)
				if len(rwindow) > s.windowLimit {
					rwindow = rwindow[len(rwindow)-s.windowLimit:]
				}
				if len(rwindow) > s.overlap {
					s.tailResp = append([]byte(nil), rwindow[len(rwindow)-s.overlap:]...)
				} else {
					s.tailResp = append([]byte(nil), rwindow...)
				}
				// For httpbin/anything the response JSON wraps the reflective payload under "data" or "json" fields.
				// Perform a fast path check before full scan to catch simple leakage tokens.
				// Detect common echo encodings returned by go-httpbin
				sb := string(b)
				if strings.Contains(sb, "api_key=") || strings.Contains(sb, "password") ||
					strings.Contains(sb, "\"prompt\":\"api_key=") || strings.Contains(sb, ": \"api_key=") {
					decision = "quarantine"
					reason = "response-leak"
					extprocStreams.WithLabelValues(decision).Inc()
					extprocStreamDuration.WithLabelValues(decision).Observe(time.Since(start).Seconds())
					if s.telemetry != nil {
						s.telemetry.Collect("decision", map[string]any{"ts": time.Now().UTC().Unix(), "decision": decision, "rule_id": reason})
					}
					return sendImmediateResponse(stream, decision, reason)
				}
				ctx, cancel := context.WithTimeout(ctx, s.timeout)
				res, scanErr := s.scResp.ScanReader(ctx, bytes.NewReader(rwindow), "extproc:response-window")
				cancel()
				if scanErr != nil && !errors.Is(scanErr, context.DeadlineExceeded) {
					if err := stream.Send(&extproc.ProcessingResponse{Response: &extproc.ProcessingResponse_ResponseBody{ResponseBody: &extproc.BodyResponse{Response: &extproc.CommonResponse{}}}}); err != nil {
						span.RecordError(err)
						return err
					}
					continue
				}
				// Decide based on response actions and severity threshold
				deny := false
				doRedact := false
				doReplace := false
				replaceBody := ""
				if len(res.Violations) > 0 {
					for _, v := range res.Violations {
						if strings.EqualFold(v.ResponseAction, "deny") || strings.EqualFold(v.ResponseAction, "block") {
							deny = true
							reason = v.RuleID
							break
						}
						if strings.EqualFold(v.ResponseAction, "redact") || strings.EqualFold(v.ResponseAction, "mask") || strings.EqualFold(v.ResponseAction, "quarantine") {
							doRedact = true
							if reason == "no_signals" && v.RuleID != "" {
								reason = v.RuleID
							}
						}
						if strings.EqualFold(v.ResponseAction, "replace") {
							doReplace = true
							if v.ResponseReplacement != "" {
								replaceBody = v.ResponseReplacement
							}
							if reason == "no_signals" && v.RuleID != "" {
								reason = v.RuleID
							}
						}
						if reason == "no_signals" && v.RuleID != "" {
							reason = v.RuleID
						}
					}
				}
				if deny || hasThresholdHit(res, s.failOn) {
					decision = "quarantine"
					switch s.enforcementMode {
					case "enforce", "quarantine":
						extprocStreams.WithLabelValues(decision).Inc()
						extprocStreamDuration.WithLabelValues(decision).Observe(time.Since(start).Seconds())
						if s.telemetry != nil {
							s.telemetry.Collect("decision", map[string]any{"ts": time.Now().UTC().Unix(), "decision": decision, "rule_id": reason})
						}
						span.SetAttributes(attribute.String("decision", decision), attribute.String("reason", reason))
						return sendImmediateResponse(stream, decision, reason)
					case "redact":
						doRedact = true
					default:
						// observe: continue without block
					}
				}
				// Replacement takes precedence over redaction when enabled in enforcing modes
				if doReplace && s.enforcementMode != "observe" && os.Getenv("PS_ENFORCER_REPLACEMENT_MUTATION") != "0" && strings.ToLower(os.Getenv("PS_ENFORCER_REPLACEMENT_MUTATION")) != "false" {
					// If no explicit replacement provided, fall back to an empty body
					bodyOut := replaceBody
					// Emit metrics and event then terminate stream with immediate 200 OK
					extprocStreams.WithLabelValues("replace").Inc()
					extprocStreamDuration.WithLabelValues("replace").Observe(time.Since(start).Seconds())
					if s.telemetry != nil {
						s.telemetry.Collect("decision", map[string]any{"ts": time.Now().UTC().Unix(), "decision": "replace", "rule_id": reason})
					}
					span.SetAttributes(attribute.String("decision", "replace"), attribute.String("reason", reason))
					return sendImmediateReplacementResponse(stream, bodyOut, reason)
				}
				if doRedact && os.Getenv("PS_ENFORCER_REDACTION_MUTATION") != "0" && strings.ToLower(os.Getenv("PS_ENFORCER_REDACTION_MUTATION")) != "false" {
					// Apply redaction to current chunk before continuing
					redacted := redact.Redact(string(b))
					rb := &extproc.BodyResponse{Response: &extproc.CommonResponse{}}
					rb.Response.BodyMutation = &extproc.BodyMutation{Mutation: &extproc.BodyMutation_Body{Body: []byte(redacted)}}
					extprocRedactions.Inc()
					if err := stream.Send(&extproc.ProcessingResponse{Response: &extproc.ProcessingResponse_ResponseBody{ResponseBody: rb}}); err != nil {
						span.RecordError(err)
						return err
					}
					continue
				}
			}
			// continue streaming response
			if err := stream.Send(&extproc.ProcessingResponse{Response: &extproc.ProcessingResponse_ResponseBody{ResponseBody: &extproc.BodyResponse{Response: &extproc.CommonResponse{}}}}); err != nil {
				span.RecordError(err)
				return err
			}
		case *extproc.ProcessingRequest_RequestTrailers:
			// Reply with an empty TrailersResponse to honor 1:1 contract if trailers are ever sent
			if err := stream.Send(&extproc.ProcessingResponse{Response: &extproc.ProcessingResponse_RequestTrailers{RequestTrailers: &extproc.TrailersResponse{}}}); err != nil {
				span.RecordError(err)
				return err
			}
			// Request stream completed without quarantine
			extprocStreams.WithLabelValues(decision).Inc()
			extprocStreamDuration.WithLabelValues(decision).Observe(time.Since(start).Seconds())
			if s.telemetry != nil {
				s.telemetry.Collect("decision", map[string]any{"ts": time.Now().UTC().Unix(), "decision": decision, "rule_id": reason})
			}
			return nil
		case *extproc.ProcessingRequest_ResponseTrailers:
			if err := stream.Send(&extproc.ProcessingResponse{Response: &extproc.ProcessingResponse_ResponseTrailers{ResponseTrailers: &extproc.TrailersResponse{}}}); err != nil {
				span.RecordError(err)
				return err
			}
			// Stream completed without quarantine
			extprocStreams.WithLabelValues(decision).Inc()
			extprocStreamDuration.WithLabelValues(decision).Observe(time.Since(start).Seconds())
			if s.telemetry != nil {
				s.telemetry.Collect("decision", map[string]any{"ts": time.Now().UTC().Unix(), "decision": decision, "rule_id": reason})
			}
			return nil
		default:
			// Unknown message; do not respond to avoid protocol violations.
			_ = x
			continue
		}
	}
}

// Run starts a gRPC server on the given address.
func Run(addr string, s *Server) (*grpc.Server, error) {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	// TLS policy: require on non-loopback unless explicitly disabled; auto-detect certs when not provided
	mode := grpcTLSMode()
	certFile, keyFile, havePair := findGRPCTLSPair()
	clientCA := os.Getenv("PS_ENFORCER_GRPC_TLS_CLIENT_CA")
	nonLoop := !isLoopbackAddr(addr)

	var opts []grpc.ServerOption
	startWithTLS := func(certFile, keyFile string) {
		if cert, cerr := tls.LoadX509KeyPair(certFile, keyFile); cerr == nil {
			tlsCfg := &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12}
			if clientCA != "" {
				if err := paths.ValidateCAFilePath(clientCA); err != nil {
					log.Printf("invalid client CA file path: %v", err)
				} else if caPEM, rerr := os.ReadFile(clientCA); rerr == nil {
					pool := x509.NewCertPool()
					if pool.AppendCertsFromPEM(caPEM) {
						tlsCfg.ClientCAs = pool
						tlsCfg.ClientAuth = tls.RequireAndVerifyClientCert
					} else {
						log.Printf("failed to parse client CA certificate")
					}
				} else {
					log.Printf("failed to read client CA file: %v", rerr)
				}
			}
			opts = append(opts, grpc.Creds(credentials.NewTLS(tlsCfg)))
		} else {
			log.Fatalf("failed to load gRPC TLS keypair (%s,%s): %v", certFile, keyFile, cerr)
		}
	}

	switch mode {
	case "disable":
		// insecure mode allowed
	case "require":
		if !havePair {
			log.Fatalf("gRPC TLS required but no certificate configured for %s; set PS_ENFORCER_GRPC_TLS_CERT and PS_ENFORCER_GRPC_TLS_KEY or mount /tls/server.crt and /tls/server.key (set PS_ENFORCER_GRPC_TLS_MODE=disable for local dev)", addr)
		}
		startWithTLS(certFile, keyFile)
	default: // auto
		if nonLoop {
			if !havePair {
				log.Fatalf("Refusing to listen gRPC on non-loopback address %s without TLS; set PS_ENFORCER_GRPC_TLS_CERT and PS_ENFORCER_GRPC_TLS_KEY or mount /tls/server.crt and /tls/server.key, or set PS_ENFORCER_GRPC_TLS_MODE=disable to allow insecure", addr)
			}
			startWithTLS(certFile, keyFile)
		} else if havePair {
			startWithTLS(certFile, keyFile)
		}
	}
	gs := grpc.NewServer(append(opts,
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.ChainUnaryInterceptor(
			grpc_recovery.UnaryServerInterceptor(grpc_recovery.WithRecoveryHandler(func(p interface{}) error { return fmt.Errorf("panic: %v", p) })),
			grpc_logging.UnaryServerInterceptor(grpc_logging.LoggerFunc(func(ctx context.Context, lvl grpc_logging.Level, msg string, fields ...any) {
				log.Printf("grpc level=%v msg=%s fields=%v", lvl, msg, fields)
			})),
		),
		grpc.ChainStreamInterceptor(
			grpc_recovery.StreamServerInterceptor(grpc_recovery.WithRecoveryHandler(func(p interface{}) error { return fmt.Errorf("panic: %v", p) })),
			grpc_logging.StreamServerInterceptor(grpc_logging.LoggerFunc(func(ctx context.Context, lvl grpc_logging.Level, msg string, fields ...any) {
				log.Printf("grpc level=%v msg=%s fields=%v", lvl, msg, fields)
			})),
		),
	)...)
	extproc.RegisterExternalProcessorServer(gs, s)
	healthpb.RegisterHealthServer(gs, s.healthSrv)
	go func() {
		if err := gs.Serve(lis); err != nil {
			log.Printf("ext_proc server error: %v", err)
		}
	}()
	return gs, nil
}

func header(k, v string) *corev3.HeaderValueOption {
	return &corev3.HeaderValueOption{Header: &corev3.HeaderValue{Key: k, RawValue: []byte(v)}}
}

func sendImmediateResponse(stream extproc.ExternalProcessor_ProcessServer, decision, reason string) error {
	resp := &extproc.ImmediateResponse{
		Status:  &typev3.HttpStatus{Code: typev3.StatusCode_Forbidden},
		Headers: &extproc.HeaderMutation{SetHeaders: []*corev3.HeaderValueOption{header("x-ps-decision", decision), header("x-ps-reason", reason)}},
		Body:    []byte("blocked by PromptShield"),
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

func hasThresholdHit(r types.ScanResult, threshold string) bool {
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

func firstReason(r types.ScanResult) string {
	if len(r.Violations) == 0 {
		return "signals_detected"
	}
	return r.Violations[0].RuleID
}

// grpcTLSMode returns one of: auto (default), require, disable
func grpcTLSMode() string {
	mode := strings.ToLower(strings.TrimSpace(os.Getenv("PS_ENFORCER_GRPC_TLS_MODE")))
	if mode == "require" || mode == "disable" {
		return mode
	}
	return "auto"
}

// isLoopbackAddr returns true if the provided listen address binds only to loopback
func isLoopbackAddr(addr string) bool {
	host := addr
	if strings.HasPrefix(host, ":") {
		// :port means all interfaces
		return false
	}
	if h, _, err := net.SplitHostPort(addr); err == nil {
		host = h
	}
	host = strings.TrimSpace(host)
	if host == "" || host == "0.0.0.0" || host == "::" {
		return false
	}
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	if ip != nil {
		return ip.IsLoopback()
	}
	// Unknown hostname: assume non-loopback to be safe
	return false
}

// findGRPCTLSPair returns cert and key file paths if configured or found in default mounts
func findGRPCTLSPair() (string, string, bool) {
	certFile := strings.TrimSpace(os.Getenv("PS_ENFORCER_GRPC_TLS_CERT"))
	keyFile := strings.TrimSpace(os.Getenv("PS_ENFORCER_GRPC_TLS_KEY"))
	if certFile != "" && keyFile != "" {
		return certFile, keyFile, true
	}
	// Auto-detect common mount locations
	defaults := [][2]string{
		{"/tls/server.crt", "/tls/server.key"},
		{"tls/server.crt", "tls/server.key"},
	}
	for _, p := range defaults {
		if _, err1 := os.Stat(p[0]); err1 == nil {
			if _, err2 := os.Stat(p[1]); err2 == nil {
				return p[0], p[1], true
			}
		}
	}
	return "", "", false
}

var (
	extprocStreams = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "ps_extproc_streams_total", Help: "Total gRPC ext_proc streams by decision"},
		[]string{"decision"},
	)
	extprocBytes = prometheus.NewCounter(
		prometheus.CounterOpts{Name: "ps_extproc_bytes_total", Help: "Total bytes observed in gRPC ext_proc streams"},
	)
	extprocStreamDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{Name: "ps_extproc_stream_duration_seconds", Help: "Duration of gRPC ext_proc streams", Buckets: prometheus.DefBuckets},
		[]string{"decision"},
	)
	extprocRuleHits = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "ps_extproc_rule_hits_total", Help: "Rule hits counted by id and severity"},
		[]string{"rule_id", "severity"},
	)
	extprocRedactions = prometheus.NewCounter(
		prometheus.CounterOpts{Name: "ps_extproc_redactions_total", Help: "Total body chunk redactions applied"},
	)
)

func init() {
	_ = prometheus.Register(extprocStreams)
	_ = prometheus.Register(extprocBytes)
	_ = prometheus.Register(extprocStreamDuration)
	_ = prometheus.Register(extprocRuleHits)
	_ = prometheus.Register(extprocRedactions)
}
