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
	"github.com/promptshield/promptshield/internal/shared/types"
	pkg "github.com/promptshield/promptshield/pkg/types"
)

func checkHandlerVersioned(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqToken := os.Getenv("PS_ENFORCER_AUTH_TOKEN")
		if reqToken != "" {
			if !httpAuthOK(r, reqToken) {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte("unauthorized"))
				return
			}
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
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		// Stream request body directly; avoid temp files

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
		
		// Use event-driven scanner manager if available for real-time enforcement
		if opt.ScannerManager != nil && opt.ScannerManager.HasActivePolicies() {
			logger.Info("Using event-driven policy scanner for real-time enforcement")
			res, err = opt.ScannerManager.ScanReader(ctx, r.Body, "http:v1:check:event-driven")
		} else {
			// Fallback to static file-based scanner
			logger.Info("Using static file-based scanner fallback")
			sc := scanner.ScanEngineCstor(0)
			// Align scanner limits with runtime config defaults (env-backed for now)
			if v := os.Getenv("PS_ENFORCER_MAX_STREAM_BYTES"); v != "" {
				if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
					sc.SetMaxStreamBytes(n)
				}
			}
			// Quarantine behavior consistent with runtime defaults
			sc.SetQuarantineOnTimeout(true)
			sc.SetQuarantineOnError(true)
			// Async scan gating is handled at handler level; L3 gating occurs in scanner via license entitlements
			rulepack := os.Getenv("PS_ENFORCER_RULEPACK")
			if rulepack != "" {
				if packs, e := rules.LoadPacks(rulepack); e == nil {
					sc.LoadRulePacks(packs)
				}
			}
			// Stream scan directly from the request body
			res, err = sc.ScanReader(ctx, r.Body, "http:v1:check:static")
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
		resp := map[string]any{"decision": decision, "reason": reason, "violations": total, "request_id": r.Header.Get("X-Request-ID")}
		_ = json.NewEncoder(w).Encode(resp)

		// Publish decision event
		if opt.Events != nil {
			opt.Events.Publish(Event{Type: "decision", Data: resp})
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
				ctx, cancel := context.WithTimeout(r.Context(), time.Duration(timeoutMs)*time.Millisecond)
				res := runScanLine(ctx, line)
				cancel()
				b, _ := json.Marshal(res)
				_, _ = bw.Write(b)
				_ = bw.WriteByte('\n')
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
		ctx, cancel := context.WithTimeout(r.Context(), time.Duration(timeoutMs)*time.Millisecond)
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
	
	// Initialize semantic analyzer if enabled
	if os.Getenv("PS_SEMANTIC_ENABLED") == "true" {
		provider := os.Getenv("PS_SEMANTIC_PROVIDER")
		if provider == "openai" {
			apiKey := os.Getenv("OPENAI_API_KEY")
			if apiKey != "" {
				analyzer := semopenai.New(semopenai.Options{
					APIKey:         apiKey,
					MaxConcurrency: 2,
					CacheSize:      1000,
					CacheTTL:       15 * time.Minute,
					RequestsPerSecond: 10,
					BurstSize:      20,
				})
				sc.SetSemanticAnalyzer(analyzer)
			}
		}
	}
	
	rulepack := os.Getenv("PS_ENFORCER_RULEPACK")
	if rulepack != "" {
		if packs, e := rules.LoadPacks(rulepack); e == nil {
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
