package grpcenforcer

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"sync/atomic"
	"time"

	extproc "github.com/envoyproxy/go-control-plane/envoy/service/ext_proc/v3"
	"github.com/promptshield/promptshield/internal/observability/metrics"
	"github.com/promptshield/promptshield/internal/shared/redact"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

// processStream contains the full streaming enforcement logic that was previously
// implemented directly in Server.Process. Splitting it into its own file keeps
// server.go readable while retaining identical behaviour.
func (s *Server) processStream(stream extproc.ExternalProcessor_ProcessServer) error {
	ctx := stream.Context()
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

	// Reset sliding-window tails for this stream
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
					return sendImmediateResponse(stream, decision, reason)
				}

				// Build sliding window including previous tail
				var window []byte
				if len(s.tailReq) > 0 {
					window = append(window, s.tailReq...)
				}
				window = append(window, b...)
				if len(window) > s.windowLimit {
					window = window[len(window)-s.windowLimit:]
				}
				// Update tail for next chunk
				if len(window) > s.overlap {
					s.tailReq = append([]byte(nil), window[len(window)-s.overlap:]...)
				} else {
					s.tailReq = append([]byte(nil), window...)
				}

				ctxScan, cancel := context.WithTimeout(ctx, s.timeout)
				
				// Acquire read lock to prevent rule reloading during scan
				s.rulesMutex.RLock()
				res, scanErr := s.scReq.ScanReader(ctxScan, bytes.NewReader(window), "extproc:request-window")
				s.rulesMutex.RUnlock()
				
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
						span.SetAttributes(attribute.String("decision", decision), attribute.String("reason", reason))
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
					return sendImmediateResponse(stream, decision, reason)
				}

				// Build sliding window on response side
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
					return sendImmediateResponse(stream, decision, reason)
				}

				ctxScan, cancel := context.WithTimeout(ctx, s.timeout)
				
				// Acquire read lock to prevent rule reloading during scan
				s.rulesMutex.RLock()
				res, scanErr := s.scResp.ScanReader(ctxScan, bytes.NewReader(rwindow), "extproc:response-window")
				s.rulesMutex.RUnlock()
				
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
						span.SetAttributes(attribute.String("decision", decision), attribute.String("reason", reason))
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
					return sendImmediateReplacementResponse(stream, replaceBody, reason)
				}
				if doRedact && os.Getenv("PS_ENFORCER_REDACTION_MUTATION") != "0" && strings.ToLower(os.Getenv("PS_ENFORCER_REDACTION_MUTATION")) != "false" {
					redacted := redact.Redact(string(b))
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
			return nil

		default:
			continue // ignore unknown message types
		}
	}
}
