package api

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/license"
	"github.com/promptshield/promptshield/internal/observability/metrics"
	"github.com/promptshield/promptshield/internal/rules"
	"github.com/promptshield/promptshield/internal/scanner"
	semopenai "github.com/promptshield/promptshield/internal/semantic/openai"
	ckeys "github.com/promptshield/promptshield/internal/shared/contextkeys"
	"github.com/promptshield/promptshield/internal/shared/types"
	pkg "github.com/promptshield/promptshield/pkg/types"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type decisionResponse struct {
	Decision   string `json:"decision"`
	Reason     string `json:"reason"`
	Violations int    `json:"violations"`
	RequestID  string `json:"request_id"`
}

func checkHandlerVersioned(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tracer := otel.Tracer("promptshield/http")
		ctxSpan, span := tracer.Start(r.Context(), "http_check",
			trace.WithAttributes(
				attribute.String("http.route", "/check"),
				attribute.String("ps.tenant_id", r.Header.Get("X-PS-Tenant-ID")),
				attribute.String("ps.request_id", r.Header.Get("X-Request-ID")),
			),
		)
		defer span.End()

		reqToken := os.Getenv("PS_ENFORCER_AUTH_TOKEN")
		if reqToken != "" {
			if !httpAuthOK(r, reqToken) {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte("unauthorized"))
				return
			}
		}
		// Require tenant id header for isolation
		tenantID := strings.TrimSpace(r.Header.Get("X-PS-Tenant-ID"))
		if _, ok := validateTenantIDString(w, tenantID); !ok {
			return
		}
		if !license.IsLicensed() {
			w.Header().Set("X-PromptShield-License", "EVALUATION")
			if !license.AllowEvalRequest() {
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte("Rate limit exceeded in evaluation mode"))
				return
			}
		} else {
			w.Header().Set("X-PromptShield-License", "LICENSED")
		}
		ctx, cancel := context.WithTimeout(ctxSpan, 2*time.Second)
		defer cancel()
		// Inject tenant into context for scanner manager and BYOK
		ctx = context.WithValue(ctx, ckeys.TenantID, tenantID)

		maxBytes := int64(1 << 20)
		if v := os.Getenv("PS_ENFORCER_MAX_BODY_BYTES"); v != "" {
			if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
				maxBytes = n
			}
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
		if r.Body != nil {
			defer r.Body.Close()
		}

		logger := slog.With("component", "api-check")
		var res pkg.ScanResult
		var err error

		// Use scanner manager for all enforcement - no fallback
		if opt.ScannerManager != nil && opt.ScannerManager.HasActivePolicies() {
			logger.Info("Using database-loaded rulepack scanner")
			res, err = opt.ScannerManager.ScanReader(ctx, r.Body, "http:v1:check:database")
		} else {
			logger.Info("No active rulepacks - allowing request (fail-open)")
			res = pkg.ScanResult{
				Violations: []pkg.Violation{},
				ScanInfo: pkg.ScanInfo{
					ShouldBlock:     false,
					BlockReason:     "no_rulepacks_loaded",
					TotalViolations: 0,
					ScanStatus:      "success",
				},
			}
		}
		if err != nil {
			// Map body-size errors to 400; otherwise 500
			msg := err.Error()
			code := http.StatusInternalServerError
			if strings.Contains(strings.ToLower(msg), "request body too large") {
				code = http.StatusBadRequest
			}
			errCode := "INTERNAL"
			if code == http.StatusBadRequest {
				errCode = "INVALID_ARGUMENT"
			}
			writeError(w, code, errCode, "scan failed", map[string]any{"error": msg})
			return
		}
		decision := "allow"
		reason := "no_signals"
		total := len(res.Violations)
		anyQuarantine := false
		anyDeny := false
		firstRule := ""
		for _, v := range res.Violations {
			if firstRule == "" {
				firstRule = v.RuleID
			}
			a := v.ResponseAction
			switch a {
			case "deny", "block":
				anyDeny = true
			case "quarantine":
				anyQuarantine = true
			}
		}
		if anyDeny {
			decision = "deny"
			reason = firstNonEmpty(firstRule, "response_action")
		} else if anyQuarantine || total > 0 {
			decision = "quarantine"
			reason = firstNonEmpty(firstRule, "signals_detected")
		}
		// Attach request id if available
		if id := r.Header.Get("X-Request-ID"); id != "" {
			w.Header().Set("x-ps-request-id", id)
		}
		w.Header().Set("x-ps-decision", decision)
		w.Header().Set("x-ps-reason", reason)
		w.Header().Set("content-type", "application/json")
		statusCode := http.StatusOK
		if decision != "allow" {
			statusCode = http.StatusForbidden
		}
		w.WriteHeader(statusCode)
		respObj := decisionResponse{Decision: decision, Reason: reason, Violations: total, RequestID: r.Header.Get("X-Request-ID")}
		_ = json.NewEncoder(w).Encode(respObj)
		respMap := map[string]any{"decision": decision, "reason": reason, "violations": total, "request_id": r.Header.Get("X-Request-ID")}

		// span decision event
		ev := "ps.decision.allow"
		if decision == "quarantine" || decision == "deny" {
			ev = "ps.decision.block"
		}
		span.SetAttributes(
			attribute.String("decision", decision),
			attribute.String("reason", reason),
			attribute.Int("ps.violations", total),
		)
		span.AddEvent(ev, trace.WithAttributes(
			attribute.String("reason", reason),
			attribute.Int("ps.violations", total),
		))

		// Publish decision event
		if opt.Events != nil {
			opt.Events.Publish(Event{Type: "decision", Data: respMap})
		}
		// Audit durable trail (best-effort)
		if opt.AuditLogger != nil {
			_ = opt.AuditLogger.LogWithContext(ctx, types.AuditEvent{
				Action:     "request.decision",
				ObjectType: "request",
				ObjectID:   uuid.New(),
				Metadata: map[string]interface{}{
					"path":       r.URL.Path,
					"status":     statusCode,
					"decision":   decision,
					"reason":     reason,
					"violations": total,
				},
				Timestamp: time.Now().UTC(),
			})
		}

		// usage accounting (best-effort)
		if store := getUsageStoreFromCtx(r.Context()); store != nil {
			tenant := r.Header.Get("x-tenant-id")
			route := r.URL.Path
			var bytesN int64
			if cl := r.Header.Get("Content-Length"); cl != "" {
				if n, err := strconv.ParseInt(cl, 10, 64); err == nil {
					bytesN = n
				}
			}
			_ = store.Record(r.Context(), tenant, route, decision, bytesN, time.Now())
		}
	}
}

func scanHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tracer := otel.Tracer("promptshield/http")
		ctxSpan, span := tracer.Start(r.Context(), "http_scan",
			trace.WithAttributes(
				attribute.String("http.route", "/scan"),
				attribute.String("ps.tenant_id", r.Header.Get("X-PS-Tenant-ID")),
				attribute.String("ps.request_id", r.Header.Get("X-Request-ID")),
			),
		)
		defer span.End()

		// Resolve runtime config from store (fallback to env defaults)
		store := opt.ConfigStore
		if store == nil {
			store = NewRuntimeConfigStoreFromEnv()
		}
		cfg := store.Get()
		timeoutMs := cfg.RequestTimeoutMs
		if timeoutMs <= 0 {
			timeoutMs = 2000
		}
		maxBytes := cfg.MaxStreamBytes
		if maxBytes <= 0 {
			maxBytes = 5_000_000
		}

		// Aggregate or line-by-line NDJSON
		aggregate := true
		if a := r.URL.Query().Get("aggregate"); a != "" {
			aggregate = a != "false"
		}

		// Enforce body limits
		r.Body = http.MaxBytesReader(w, r.Body, maxBytes)

		ct := r.Header.Get("content-type")
		modeLabel := "aggregate"
		if strings.HasPrefix(ct, "application/x-ndjson") && !aggregate {
			modeLabel = "ndjson"
			w.Header().Set("content-type", "application/x-ndjson")
			bw := bufio.NewWriter(w)
			defer bw.Flush()
			s := bufio.NewScanner(r.Body)
			buf := make([]byte, 0, 1024*1024)
			s.Buffer(buf, 10*1024*1024)
			for s.Scan() {
				line := s.Bytes()
				// attach tenant id into context for BYOK
				base := r.Context()
				if t := strings.TrimSpace(r.Header.Get("X-PS-Tenant-ID")); t != "" {
					base = context.WithValue(base, ckeys.TenantID, t)
				}
				ctx, cancel := context.WithTimeout(base, time.Duration(timeoutMs)*time.Millisecond)
				res := runScanLine(ctx, line)
				cancel()
				b, _ := json.Marshal(res)
				_, _ = bw.Write(b)
				_ = bw.WriteByte('\n')
				// span per-line event
				dn := stringFromAny(res["decision"])
				rn := stringFromAny(res["reason"])
				vn := intFromAny(res["violations"])
				ev := "ps.decision.allow"
				if dn == "quarantine" || dn == "deny" {
					ev = "ps.decision.block"
				}
				span.AddEvent(ev, trace.WithAttributes(
					attribute.String("decision", dn),
					attribute.String("reason", rn),
					attribute.Int("ps.violations", vn),
				))
				if opt.Events != nil {
					opt.Events.Publish(Event{Type: "decision", Data: res})
				}
				if opt.AuditLogger != nil {
					_ = opt.AuditLogger.LogWithContext(ctx, types.AuditEvent{
						Action:     "scan.decision",
						ObjectType: "request",
						ObjectID:   uuid.New(),
						Metadata:   res,
						Timestamp:  time.Now().UTC(),
					})
				}
				metrics.ScanEventsTotal.WithLabelValues(r.URL.Path).Inc()
			}
			metrics.ScanRequestDuration.WithLabelValues(modeLabel).Observe(float64(timeoutMs) / 1000.0)
			return
		}

		body, _ := io.ReadAll(r.Body)
		ctx, cancel := context.WithTimeout(ctxSpan, time.Duration(timeoutMs)*time.Millisecond)
		defer cancel()
		// Try JSON array for aggregate decisions
		var (
			rawArray        []json.RawMessage
			decisions       []map[string]any
			totalViolations int
		)
		if len(body) > 0 && body[0] == '[' && json.Unmarshal(body, &rawArray) == nil && len(rawArray) > 0 {
			for _, item := range rawArray {
				res := runScanLine(ctx, item)
				decisions = append(decisions, res)
				totalViolations += intFromAny(res["violations"])
				// span per-item decision event
				dn := stringFromAny(res["decision"])
				rn := stringFromAny(res["reason"])
				vn := intFromAny(res["violations"])
				ev := "ps.decision.allow"
				if dn == "quarantine" || dn == "deny" {
					ev = "ps.decision.block"
				}
				span.AddEvent(ev, trace.WithAttributes(
					attribute.String("decision", dn),
					attribute.String("reason", rn),
					attribute.Int("ps.violations", vn),
				))
				if opt.Events != nil {
					opt.Events.Publish(Event{Type: "decision", Data: res})
				}
				if opt.AuditLogger != nil {
					_ = opt.AuditLogger.LogWithContext(ctx, types.AuditEvent{
						Action:     "scan.decision",
						ObjectType: "request",
						ObjectID:   uuid.New(),
						Metadata:   res,
						Timestamp:  time.Now().UTC(),
					})
				}
			}
			report := map[string]any{
				"decisions": decisions,
				"summary": map[string]any{
					"total":      len(decisions),
					"violations": totalViolations,
				},
			}
			_ = json.NewEncoder(w).Encode(report)
			metrics.ScanRequestDuration.WithLabelValues(modeLabel).Observe(float64(timeoutMs) / 1000.0)
			return
		}
		// Fallback: single record
		res := runScanLine(ctx, body)
		// record span decision event
		dn := stringFromAny(res["decision"])
		rn := stringFromAny(res["reason"])
		vn := intFromAny(res["violations"])
		ev := "ps.decision.allow"
		if dn == "quarantine" || dn == "deny" {
			ev = "ps.decision.block"
		}
		span.SetAttributes(attribute.Int("ps.violations", vn))
		span.AddEvent(ev, trace.WithAttributes(
			attribute.String("decision", dn),
			attribute.String("reason", rn),
			attribute.Int("ps.violations", vn),
		))

		report := map[string]any{
			"decisions": []any{res},
			"summary": map[string]any{
				"total":      1,
				"violations": intFromAny(res["violations"]),
			},
		}
		_ = json.NewEncoder(w).Encode(report)
		if opt.Events != nil {
			opt.Events.Publish(Event{Type: "decision", Data: res})
		}
		if opt.AuditLogger != nil {
			_ = opt.AuditLogger.LogWithContext(ctx, types.AuditEvent{
				Action:     "scan.decision",
				ObjectType: "request",
				ObjectID:   uuid.New(),
				Metadata:   res,
				Timestamp:  time.Now().UTC(),
			})
		}
		metrics.ScanRequestDuration.WithLabelValues(modeLabel).Observe(float64(timeoutMs) / 1000.0)
	}
}

func runScanLine(ctx context.Context, data []byte) map[string]any {
	sc := scanner.ScanEngineCstor(0)
	sc.SetBaseContext(ctx)
	// Initialize semantic analyzer (omni only for /scan path)
	if os.Getenv("PS_SEMANTIC_ENABLED") == "true" {
		if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
			analyzer := semopenai.New(semopenai.Options{
				APIKey:            apiKey,
				MaxConcurrency:    2,
				CacheSize:         1000,
				CacheTTL:          15 * time.Minute,
				RequestsPerSecond: 10,
				BurstSize:         20,
			})
			sc.SetSemanticAnalyzer(analyzer)
		}
	}
	if rp := os.Getenv("PS_ENFORCER_RULEPACK"); rp != "" {
		if packs, e := rules.LoadPacks(rp); e == nil {
			sc.LoadRulePacks(packs)
		}
	}
	res, err := sc.ScanReader(ctx, bytes.NewReader(data), "http:v1:scan-line")
	decision := "allow"
	reason := "no_signals"
	total := 0
	if err == nil {
		total = len(res.Violations)
		anyQuarantine := false
		anyDeny := false
		firstRule := ""
		for _, v := range res.Violations {
			if firstRule == "" {
				firstRule = v.RuleID
			}
			switch v.ResponseAction {
			case "deny", "block":
				anyDeny = true
			case "quarantine":
				anyQuarantine = true
			}
		}
		if anyDeny {
			decision = "deny"
			reason = firstNonEmpty(firstRule, "response_action")
		} else if anyQuarantine || total > 0 {
			decision = "quarantine"
			reason = firstNonEmpty(firstRule, "signals_detected")
		}
	}
	return map[string]any{"decision": decision, "reason": reason, "violations": total}
}

// stringFromAny converts interface{} to string best-effort
func stringFromAny(v any) string {
	s, _ := v.(string)
	return s
}
