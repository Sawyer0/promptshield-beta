package grpcenforcer

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"io"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	extproc "github.com/envoyproxy/go-control-plane/envoy/service/ext_proc/v3"
	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/observability/metrics"
	"github.com/promptshield/promptshield/internal/scanner"
	"github.com/promptshield/promptshield/internal/shared/redact"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// processStream contains the full streaming enforcement logic that was previously
// implemented directly in Server.Process. Splitting it into its own file keeps
// server.go readable while retaining identical behaviour.
func (s *Server) processStream(stream extproc.ExternalProcessor_ProcessServer) error {
	ctx := stream.Context()
	tracer := otel.Tracer("promptshield/enforcer")
	ctx, span := tracer.Start(ctx, "extproc_stream")
	defer span.End()

	// Extract tenant ID from request headers when they arrive
	var tenantID uuid.UUID
	var tenantScReq, tenantScResp *scanner.Scanner
	// Enrichment from headers
	var method, endpoint, toolID, lane, planHash, conversationID, requestID string
	var planStep int

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
	var didAudit bool

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

	// Sliding-window tails (per-stream)
	var tailReq []byte
	var tailResp []byte

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
			if err == io.EOF {
				// Normal stream end
				return nil
			}
			return err
		}

		switch x := req.Request.(type) {
		case *extproc.ProcessingRequest_RequestHeaders:
			// Extract tenant ID and enrichers from request headers
			if x.RequestHeaders != nil && x.RequestHeaders.Headers != nil {
				tenantID = s.extractTenantID(x.RequestHeaders.Headers.Headers)
				for _, hv := range x.RequestHeaders.Headers.Headers {
					if hv == nil {
						continue
					}
					k := strings.ToLower(hv.Key)
					v := string(hv.RawValue)
					if v == "" {
						v = hv.Value
					}
					switch k {
					case ":method":
						method = v
					case ":path":
						endpoint = v
					case "x-ps-tool-id":
						toolID = v
					case "x-ps-lane":
						lane = v
					case "x-ps-plan-hash":
						planHash = v
					case "x-ps-plan-step":
						if n, e := strconv.Atoi(strings.TrimSpace(v)); e == nil {
							planStep = n
						}
					case "x-ps-conversation-id":
						conversationID = v
					case "x-request-id":
						requestID = v
					}
				}

				// Load tenant-specific scanners
				var err error
				tenantScReq, tenantScResp, err = s.getTenantScanners(ctx, tenantID)
				if err != nil {
					slog.Error("Failed to load tenant scanners", "tenant_id", tenantID, "error", err)
					// Fallback to global scanners
					tenantScReq, tenantScResp = s.scReq, s.scResp
				}

				span.SetAttributes(attribute.String("ps.tenant_id", tenantID.String()))
				span.SetAttributes(
					attribute.String("http.method", method),
					attribute.String("http.route", endpoint),
					attribute.String("ps.tool_id", toolID),
					attribute.String("ps.lane", lane),
					attribute.String("ps.plan_hash", planHash),
					attribute.Int("ps.plan_step", planStep),
					attribute.String("ps.conversation_id", conversationID),
					attribute.String("ps.request_id", requestID),
				)
			}

			// If no tenant scanners loaded, use global as fallback
			if tenantScReq == nil || tenantScResp == nil {
				tenantScReq, tenantScResp = s.scReq, s.scResp
			}

			// Apply runtime context to tenant-scoped scanners (do not mutate shared globals)
			if tenantScReq != nil && tenantScReq != s.scReq {
				tenantScReq.SetRuntimeContext(map[string]string{
					"direction": "request",
					"endpoint":  endpoint,
					"method":    method,
					"lane":      lane,
					"tool_id":   toolID,
				})
			}
			if tenantScResp != nil && tenantScResp != s.scResp {
				tenantScResp.SetRuntimeContext(map[string]string{
					"direction": "response",
					"endpoint":  endpoint,
					"method":    method,
					"lane":      lane,
					"tool_id":   toolID,
				})
			}

			// Pass through request headers
			if err := stream.Send(&extproc.ProcessingResponse{Response: &extproc.ProcessingResponse_RequestHeaders{RequestHeaders: &extproc.HeadersResponse{Response: &extproc.CommonResponse{}}}}); err != nil {
				span.RecordError(err)
				return err
			}

		case *extproc.ProcessingRequest_ResponseHeaders:
			// Header-based leak detection & response header mutations
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
						metrics.ExtprocStreams.WithLabelValues(decision).Inc()
						metrics.ExtprocStreamDuration.WithLabelValues(decision).Observe(time.Since(start).Seconds())
						if s.telemetry != nil {
							s.telemetry.Collect("decision", map[string]any{"ts": time.Now().UTC().Unix(), "decision": decision, "rule_id": reason})
						}
						span.SetAttributes(attribute.String("decision", decision), attribute.String("reason", reason))
						// audit
						s.auditDecision("enforcer.decision", map[string]any{
							"tenant_id":       tenantID.String(),
							"endpoint":        endpoint,
							"method":          method,
							"tool_id":         toolID,
							"lane":            lane,
							"plan_hash":       planHash,
							"plan_step":       planStep,
							"conversation_id": conversationID,
							"request_id":      requestID,
							"decision":        decision,
							"reason":          reason,
							"latency_ms":      time.Since(start).Milliseconds(),
							"bytes_request":   total,
							"bytes_response":  totalResp,
							"rulepack_ids":    s.currentRulepackIDs(),
						})
						didAudit = true
						return sendImmediateResponse(stream, decision, reason)
					}
				}
			}
			// Inject decision marker headers (if absent)
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
			rh.Response.HeaderMutation.SetHeaders = append(rh.Response.HeaderMutation.SetHeaders, header("x-ps-extproc", "resp"))
			if err := stream.Send(&extproc.ProcessingResponse{Response: &extproc.ProcessingResponse_ResponseHeaders{ResponseHeaders: rh}}); err != nil {
				span.RecordError(err)
				return err
			}

		case *extproc.ProcessingRequest_RequestBody:
			body := x.RequestBody
			if body != nil && len(body.Body) > 0 {
				b := body.Body
				// inflight bytes accounting + global ceiling enforcement
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
					lastReqAdded = added
				}

				total += int64(len(b))
				metrics.ExtprocBytes.Add(float64(len(b)))
				if s.maxStreamBytes > 0 && total > s.maxStreamBytes {
					decision = "quarantine"
					reason = "body_limit"
					metrics.ExtprocStreams.WithLabelValues(decision).Inc()
					metrics.ExtprocStreamDuration.WithLabelValues(decision).Observe(time.Since(start).Seconds())
					if s.telemetry != nil {
						s.telemetry.Collect("decision", map[string]any{"ts": time.Now().UTC().Unix(), "decision": decision, "rule_id": reason})
					}
					// audit
					s.auditDecision("enforcer.decision", map[string]any{
						"tenant_id":       tenantID.String(),
						"endpoint":        endpoint,
						"method":          method,
						"tool_id":         toolID,
						"lane":            lane,
						"plan_hash":       planHash,
						"plan_step":       planStep,
						"conversation_id": conversationID,
						"request_id":      requestID,
						"decision":        decision,
						"reason":          reason,
						"latency_ms":      time.Since(start).Milliseconds(),
						"bytes_request":   total,
						"bytes_response":  totalResp,
						"rulepack_ids":    s.currentRulepackIDs(),
					})
					span.AddEvent("ps.decision.block", trace.WithAttributes(
						attribute.String("reason", reason),
						attribute.Int64("ps.bytes_request_total", total),
						attribute.Int64("ps.bytes_response_total", totalResp),
					))
					return sendImmediateResponse(stream, decision, reason)
				}
				span.SetAttributes(attribute.Int64("ps.bytes_request_total", total))

				// Build sliding window including previous tail
				var window []byte
				if len(tailReq) > 0 {
					window = append(window, tailReq...)
				}
				window = append(window, b...)
				if len(window) > s.windowLimit {
					window = window[len(window)-s.windowLimit:]
				}
				// Update tail for next chunk
				if len(window) > s.overlap {
					tailReq = append([]byte(nil), window[len(window)-s.overlap:]...)
				} else {
					tailReq = append([]byte(nil), window...)
				}

				ctxScan, cancel := context.WithTimeout(ctx, s.timeout)

				// Acquire read lock to prevent rule reloading during scan
				s.rulesMutex.RLock()
				scLocalReq := tenantScReq
				if scLocalReq == nil {
					scLocalReq = s.scReq
				}
				s.rulesMutex.RUnlock()
				res, scanErr := scLocalReq.ScanReader(ctxScan, bytes.NewReader(window), "extproc:request-window")

				cancel()
				if scanErr != nil && !errors.Is(scanErr, context.DeadlineExceeded) {
					_ = stream.Send(&extproc.ProcessingResponse{Response: &extproc.ProcessingResponse_RequestBody{RequestBody: &extproc.BodyResponse{Response: &extproc.CommonResponse{}}}})
					continue
				}
				if hasThresholdHit(res, s.failOn) {
					decision = "quarantine"
					reason = firstReason(res)
					for _, v := range res.Violations {
						metrics.ExtprocRuleHits.WithLabelValues(v.RuleID, strings.ToUpper(v.Severity)).Inc()
					}
					if s.enforcementMode == "enforce" || s.enforcementMode == "quarantine" {
						metrics.ExtprocStreams.WithLabelValues(decision).Inc()
						metrics.ExtprocStreamDuration.WithLabelValues(decision).Observe(time.Since(start).Seconds())
						if s.telemetry != nil {
							s.telemetry.Collect("decision", map[string]any{"ts": time.Now().UTC().Unix(), "decision": decision, "rule_id": reason})
						}
						span.SetAttributes(attribute.String("decision", decision), attribute.String("reason", reason), attribute.Int("ps.violations", len(res.Violations)))
						span.AddEvent("ps.decision.block", trace.WithAttributes(
							attribute.String("reason", reason),
							attribute.Int("ps.violations", len(res.Violations)),
							attribute.Int64("ps.bytes_request_total", total),
							attribute.Int64("ps.bytes_response_total", totalResp),
						))
						// audit
						s.auditDecision("enforcer.decision", map[string]any{
							"tenant_id":       tenantID.String(),
							"endpoint":        endpoint,
							"method":          method,
							"tool_id":         toolID,
							"lane":            lane,
							"plan_hash":       planHash,
							"plan_step":       planStep,
							"conversation_id": conversationID,
							"request_id":      requestID,
							"decision":        decision,
							"reason":          reason,
							"latency_ms":      time.Since(start).Milliseconds(),
							"bytes_request":   total,
							"bytes_response":  totalResp,
							"violations":      len(res.Violations),
							"rulepack_ids":    s.currentRulepackIDs(),
						})
						didAudit = true
						return sendImmediateResponseWithDetails(stream, decision, reason, &res)
					}
					// observe/redact: continue streaming without block
				}
			}
			// continue streaming request body
			if err := stream.Send(&extproc.ProcessingResponse{Response: &extproc.ProcessingResponse_RequestBody{RequestBody: &extproc.BodyResponse{Response: &extproc.CommonResponse{}}}}); err != nil {
				span.RecordError(err)
				return err
			}

		case *extproc.ProcessingRequest_ResponseBody:
			body := x.ResponseBody
			if body != nil && len(body.Body) > 0 {
				b := body.Body
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
				metrics.ExtprocBytes.Add(float64(len(b)))
				if s.maxStreamBytes > 0 && totalResp > s.maxStreamBytes {
					decision = "quarantine"
					reason = "body_limit"
					metrics.ExtprocStreams.WithLabelValues(decision).Inc()
					metrics.ExtprocStreamDuration.WithLabelValues(decision).Observe(time.Since(start).Seconds())
					if s.telemetry != nil {
						s.telemetry.Collect("decision", map[string]any{"ts": time.Now().UTC().Unix(), "decision": decision, "rule_id": reason})
					}
					// audit
					s.auditDecision("enforcer.decision", map[string]any{
						"tenant_id":       tenantID.String(),
						"endpoint":        endpoint,
						"method":          method,
						"tool_id":         toolID,
						"lane":            lane,
						"plan_hash":       planHash,
						"plan_step":       planStep,
						"conversation_id": conversationID,
						"request_id":      requestID,
						"decision":        decision,
						"reason":          reason,
						"latency_ms":      time.Since(start).Milliseconds(),
						"bytes_request":   total,
						"bytes_response":  totalResp,
						"rulepack_ids":    s.currentRulepackIDs(),
					})
					span.AddEvent("ps.decision.block", trace.WithAttributes(
						attribute.String("reason", reason),
						attribute.Int64("ps.bytes_request_total", total),
						attribute.Int64("ps.bytes_response_total", totalResp),
					))
					return sendImmediateResponse(stream, decision, reason)
				}
				span.SetAttributes(attribute.Int64("ps.bytes_response_total", totalResp))

				// Build sliding window on response side
				var rwindow []byte
				if len(tailResp) > 0 {
					rwindow = append(rwindow, tailResp...)
				}
				rwindow = append(rwindow, b...)
				if len(rwindow) > s.windowLimit {
					rwindow = rwindow[len(rwindow)-s.windowLimit:]
				}
				if len(rwindow) > s.overlap {
					tailResp = append([]byte(nil), rwindow[len(rwindow)-s.overlap:]...)
				} else {
					tailResp = append([]byte(nil), rwindow...)
				}

				// quick leak heuristic
				sb := string(b)
				if strings.Contains(sb, "api_key=") || strings.Contains(sb, "password") || strings.Contains(sb, "\"prompt\":\"api_key=") {
					decision = "quarantine"
					reason = "response-leak"
					metrics.ExtprocStreams.WithLabelValues(decision).Inc()
					metrics.ExtprocStreamDuration.WithLabelValues(decision).Observe(time.Since(start).Seconds())
					if s.telemetry != nil {
						s.telemetry.Collect("decision", map[string]any{"ts": time.Now().UTC().Unix(), "decision": decision, "rule_id": reason})
					}
					// audit
					s.auditDecision("enforcer.decision", map[string]any{
						"tenant_id":       tenantID.String(),
						"endpoint":        endpoint,
						"method":          method,
						"tool_id":         toolID,
						"lane":            lane,
						"plan_hash":       planHash,
						"plan_step":       planStep,
						"conversation_id": conversationID,
						"request_id":      requestID,
						"decision":        decision,
						"reason":          reason,
						"latency_ms":      time.Since(start).Milliseconds(),
						"bytes_request":   total,
						"bytes_response":  totalResp,
						"rulepack_ids":    s.currentRulepackIDs(),
					})
					span.AddEvent("ps.decision.block", trace.WithAttributes(
						attribute.String("reason", reason),
						attribute.Int64("ps.bytes_request_total", total),
						attribute.Int64("ps.bytes_response_total", totalResp),
					))
					return sendImmediateResponse(stream, decision, reason)
				}

				ctxScan, cancel := context.WithTimeout(ctx, s.timeout)

				// Acquire read lock to prevent rule reloading during scan
				s.rulesMutex.RLock()
				scLocalResp := tenantScResp
				if scLocalResp == nil {
					scLocalResp = s.scResp
				}
				s.rulesMutex.RUnlock()
				res, scanErr := scLocalResp.ScanReader(ctxScan, bytes.NewReader(rwindow), "extproc:response-window")

				cancel()
				if scanErr != nil && !errors.Is(scanErr, context.DeadlineExceeded) {
					_ = stream.Send(&extproc.ProcessingResponse{Response: &extproc.ProcessingResponse_ResponseBody{ResponseBody: &extproc.BodyResponse{Response: &extproc.CommonResponse{}}}})
					continue
				}

				// Response-side decision logic (deny / redact / replace)
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
						metrics.ExtprocStreams.WithLabelValues(decision).Inc()
						metrics.ExtprocStreamDuration.WithLabelValues(decision).Observe(time.Since(start).Seconds())
						if s.telemetry != nil {
							s.telemetry.Collect("decision", map[string]any{"ts": time.Now().UTC().Unix(), "decision": decision, "rule_id": reason})
						}
						span.SetAttributes(attribute.String("decision", decision), attribute.String("reason", reason), attribute.Int("ps.violations", len(res.Violations)))
						span.AddEvent("ps.decision.block", trace.WithAttributes(
							attribute.String("reason", reason),
							attribute.Int("ps.violations", len(res.Violations)),
							attribute.Int64("ps.bytes_request_total", total),
							attribute.Int64("ps.bytes_response_total", totalResp),
						))
						return sendImmediateResponseWithDetails(stream, decision, reason, &res)
					case "redact":
						doRedact = true
					}
				}

				// replacement beats redaction when allowed
				if doReplace && s.enforcementMode != "observe" && os.Getenv("PS_ENFORCER_REPLACEMENT_MUTATION") != "0" && strings.ToLower(os.Getenv("PS_ENFORCER_REPLACEMENT_MUTATION")) != "false" {
					metrics.ExtprocStreams.WithLabelValues("replace").Inc()
					metrics.ExtprocStreamDuration.WithLabelValues("replace").Observe(time.Since(start).Seconds())
					if s.telemetry != nil {
						s.telemetry.Collect("decision", map[string]any{"ts": time.Now().UTC().Unix(), "decision": "replace", "rule_id": reason})
					}
					span.SetAttributes(attribute.String("decision", "replace"), attribute.String("reason", reason))
					span.AddEvent("ps.decision.replace", trace.WithAttributes(
						attribute.String("reason", reason),
						attribute.Int64("ps.bytes_request_total", total),
						attribute.Int64("ps.bytes_response_total", totalResp),
					))
					return sendImmediateReplacementResponse(stream, replaceBody, reason)
				}
				if doRedact && os.Getenv("PS_ENFORCER_REDACTION_MUTATION") != "0" && strings.ToLower(os.Getenv("PS_ENFORCER_REDACTION_MUTATION")) != "false" {
					// Assurance mode
					assurance := strings.ToLower(strings.TrimSpace(os.Getenv("PS_REDACTION_ASSURANCE")))
					if assurance == "" {
						assurance = "conservative"
					}
					if assurance == "observe" {
						// Do not mutate in observe mode
						if err := stream.Send(&extproc.ProcessingResponse{Response: &extproc.ProcessingResponse_ResponseBody{ResponseBody: &extproc.BodyResponse{Response: &extproc.CommonResponse{}}}}); err != nil {
							span.RecordError(err)
							return err
						}
						continue
					}
					redacted := redact.Redact(string(b))
					// Post-redaction self-scan
					ctxScan2, cancel2 := context.WithTimeout(ctx, s.timeout)
					mutWindow := append([]byte(nil), tailResp...)
					mutWindow = append(mutWindow, []byte(redacted)...)
					if len(mutWindow) > s.windowLimit {
						mutWindow = mutWindow[len(mutWindow)-s.windowLimit:]
					}
					res2, scanErr2 := scLocalResp.ScanReader(ctxScan2, bytes.NewReader(mutWindow), "extproc:response-window:redacted")
					cancel2()
					unclean := false
					if scanErr2 != nil && !errors.Is(scanErr2, context.DeadlineExceeded) {
						unclean = true
					}
					if len(res2.Violations) > 0 {
						if assurance == "conservative" {
							unclean = true
						} else {
							// best_effort: block only if threshold met
							if hasThresholdHit(res2, s.failOn) {
								unclean = true
							}
						}
					}
					if unclean {
						decision = "quarantine"
						reason = "redaction_not_clean"
						metrics.ExtprocStreams.WithLabelValues(decision).Inc()
						metrics.ExtprocStreamDuration.WithLabelValues(decision).Observe(time.Since(start).Seconds())
						if s.telemetry != nil {
							s.telemetry.Collect("decision", map[string]any{"ts": time.Now().UTC().Unix(), "decision": decision, "rule_id": reason})
						}
						span.SetAttributes(attribute.String("decision", decision), attribute.String("reason", reason))
						return sendImmediateResponse(stream, decision, reason)
					}
					// Send mutated body chunk
					rb := &extproc.BodyResponse{Response: &extproc.CommonResponse{}}
					rb.Response.BodyMutation = &extproc.BodyMutation{Mutation: &extproc.BodyMutation_Body{Body: []byte(redacted)}}
					metrics.ExtprocRedactions.Inc()
					if err := stream.Send(&extproc.ProcessingResponse{Response: &extproc.ProcessingResponse_ResponseBody{ResponseBody: rb}}); err != nil {
						span.RecordError(err)
						return err
					}
					continue
				}
			}
			// continue streaming response body
			if err := stream.Send(&extproc.ProcessingResponse{Response: &extproc.ProcessingResponse_ResponseBody{ResponseBody: &extproc.BodyResponse{Response: &extproc.CommonResponse{}}}}); err != nil {
				span.RecordError(err)
				return err
			}

		case *extproc.ProcessingRequest_RequestTrailers:
			if err := stream.Send(&extproc.ProcessingResponse{Response: &extproc.ProcessingResponse_RequestTrailers{RequestTrailers: &extproc.TrailersResponse{}}}); err != nil {
				span.RecordError(err)
				return err
			}
			metrics.ExtprocStreams.WithLabelValues(decision).Inc()
			metrics.ExtprocStreamDuration.WithLabelValues(decision).Observe(time.Since(start).Seconds())
			if s.telemetry != nil {
				s.telemetry.Collect("decision", map[string]any{"ts": time.Now().UTC().Unix(), "decision": decision, "rule_id": reason})
			}
			if !didAudit {
				// final allow/observe path audit
				s.auditDecision("enforcer.decision", map[string]any{
					"tenant_id":       tenantID.String(),
					"endpoint":        endpoint,
					"method":          method,
					"tool_id":         toolID,
					"lane":            lane,
					"plan_hash":       planHash,
					"plan_step":       planStep,
					"conversation_id": conversationID,
					"request_id":      requestID,
					"decision":        decision,
					"reason":          reason,
					"latency_ms":      time.Since(start).Milliseconds(),
					"bytes_request":   total,
					"bytes_response":  totalResp,
					"rulepack_ids":    s.currentRulepackIDs(),
				})
				didAudit = true
			}
			return nil

		case *extproc.ProcessingRequest_ResponseTrailers:
			if err := stream.Send(&extproc.ProcessingResponse{Response: &extproc.ProcessingResponse_ResponseTrailers{ResponseTrailers: &extproc.TrailersResponse{}}}); err != nil {
				span.RecordError(err)
				return err
			}
			metrics.ExtprocStreams.WithLabelValues(decision).Inc()
			metrics.ExtprocStreamDuration.WithLabelValues(decision).Observe(time.Since(start).Seconds())
			if s.telemetry != nil {
				s.telemetry.Collect("decision", map[string]any{"ts": time.Now().UTC().Unix(), "decision": decision, "rule_id": reason})
			}
			if !didAudit {
				s.auditDecision("enforcer.decision", map[string]any{
					"tenant_id":       tenantID.String(),
					"endpoint":        endpoint,
					"method":          method,
					"tool_id":         toolID,
					"lane":            lane,
					"plan_hash":       planHash,
					"plan_step":       planStep,
					"conversation_id": conversationID,
					"request_id":      requestID,
					"decision":        decision,
					"reason":          reason,
					"latency_ms":      time.Since(start).Milliseconds(),
					"bytes_request":   total,
					"bytes_response":  totalResp,
					"rulepack_ids":    s.currentRulepackIDs(),
				})
				didAudit = true
			}
			return nil

		default:
			continue // ignore unknown message types
		}
	}
}

// extractTenantID extracts tenant ID from HTTP headers in gRPC metadata
func (s *Server) extractTenantID(headers []*corev3.HeaderValue) uuid.UUID {
	for _, header := range headers {
		if header.Key == "x-ps-tenant-id" || header.Key == "X-PS-Tenant-ID" {
			if tenantID, err := uuid.Parse(string(header.RawValue)); err == nil {
				return tenantID
			}
		}
	}
	return uuid.Nil // No tenant ID found
}

// getTenantScanners loads tenant-specific scanners with their rulepacks
func (s *Server) getTenantScanners(ctx context.Context, tenantID uuid.UUID) (*scanner.Scanner, *scanner.Scanner, error) {
	if tenantID == uuid.Nil {
		// No tenant ID, use global scanners
		return s.scReq, s.scResp, nil
	}

	if s.rulepackRepo == nil {
		// No database configured, use global scanners
		return s.scReq, s.scResp, nil
	}

	// Load tenant-specific rulepacks from database
	packs, err := LoadRulesFromDatabase(ctx, s.rulepackRepo, tenantID)
	if err != nil {
		return nil, nil, err
	}

	if len(packs) == 0 {
		// No tenant-specific rules, use global scanners
		return s.scReq, s.scResp, nil
	}

	// Create tenant-specific scanners
	tenantScReq := scanner.ScanEngineCstor(0)
	tenantScResp := scanner.ScanEngineCstor(0)

	// Load tenant rulepacks into scanners
	tenantScReq.LoadRulePacks(packs)
	tenantScResp.LoadRulePacks(packs)

	slog.Debug("Loaded tenant-specific scanners", "tenant_id", tenantID, "rulepacks", len(packs))
	return tenantScReq, tenantScResp, nil
}
