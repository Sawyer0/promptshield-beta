package enforcerhttp

import (
	"context"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/promptshield/promptshield/internal/encoding/jsonx"
	"github.com/promptshield/promptshield/internal/license"
	"github.com/promptshield/promptshield/internal/scanner"
	"go.opentelemetry.io/otel/trace"
)

// checkHandlerConfig carries the state the /check handler needs from the mux.
type checkHandlerConfig struct {
	// scanners hands out *scanner.Scanner values preloaded with the active RulePacks.
	scanners *sync.Pool
	// hasRules reports whether any RulePack was loaded at startup.
	hasRules bool
	// requireRules mirrors PS_REQUIRE_RULEPACK_AT_STARTUP.
	requireRules bool
}

// newCheckHandler returns the POST /check handler used directly and by Envoy
// ext_authz (HTTP service). It streams the request body through the scanner
// and answers with a decision in headers and a small JSON body.
//
// Status code follows the enforcement mode: observe (default) always answers
// 200 so traffic keeps flowing; enforce or quarantine answers 403 when the
// decision is not allow.
func newCheckHandler(cfg checkHandlerConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if reqToken := os.Getenv("PS_ENFORCER_AUTH_TOKEN"); reqToken != "" && !HttpAuthOK(r, reqToken) {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("unauthorized"))
			recordRequest(http.StatusUnauthorized)
			return
		}
		if !license.IsLicensed() {
			w.Header().Set("X-PromptShield-License", "EVALUATION")
			if !license.AllowEvalRequest() {
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte("Rate limit exceeded in evaluation mode"))
				recordRequest(http.StatusTooManyRequests)
				return
			}
		} else {
			w.Header().Set("X-PromptShield-License", "LICENSED")
		}

		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		start := time.Now()

		maxBytes := int64(1 << 20)
		if v := os.Getenv("PS_ENFORCER_MAX_BODY_BYTES"); v != "" {
			if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
				maxBytes = n
			}
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
		defer r.Body.Close()

		sc := cfg.scanners.Get().(*scanner.Scanner)
		defer cfg.scanners.Put(sc)

		res, err := sc.ScanReader(ctx, r.Body, "http:request")
		if err != nil {
			code := http.StatusInternalServerError
			body := map[string]any{"code": "INTERNAL", "message": "scan failed", "details": map[string]any{"error": err.Error()}}
			if strings.Contains(strings.ToLower(err.Error()), "request body too large") {
				code = http.StatusBadRequest
				body = map[string]any{"code": "INVALID_ARGUMENT", "message": "request body too large or invalid", "details": map[string]any{"max_bytes": maxBytes}}
			}
			w.Header().Set("content-type", "application/json")
			w.WriteHeader(code)
			_ = jsonx.NewEncoder(w).Encode(body)
			recordRequest(code)
			return
		}

		decision, reason := "allow", "no_signals"
		anyDeny, anyQuarantine := false, false
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
		total := len(res.Violations)
		switch {
		case anyDeny:
			decision, reason = "deny", firstNonEmpty(firstRule, "response_action")
		case anyQuarantine || total > 0:
			decision, reason = "quarantine", firstNonEmpty(firstRule, "signals_detected")
		}

		reqID := middleware.GetReqID(r.Context())
		if reqID == "" {
			reqID = generateRequestID()
		}

		mode := strings.ToLower(strings.TrimSpace(os.Getenv("PS_ENFORCER_MODE")))
		if mode == "" {
			mode = strings.ToLower(strings.TrimSpace(os.Getenv("PS_ENFORCER_ENFORCEMENT_MODE")))
		}
		statusCode := http.StatusOK
		if (mode == "enforce" || mode == "quarantine") && decision != "allow" {
			statusCode = http.StatusForbidden
		}

		if spanCtx := trace.SpanFromContext(r.Context()).SpanContext(); spanCtx.IsValid() {
			w.Header().Set("x-ps-trace-id", spanCtx.TraceID().String())
		}
		w.Header().Set("x-ps-request-id", reqID)
		w.Header().Set("x-ps-decision", decision)
		w.Header().Set("x-ps-reason", reason)
		w.Header().Set("content-type", "application/json")
		w.WriteHeader(statusCode)
		_ = jsonx.NewEncoder(w).Encode(map[string]any{"decision": decision, "reason": reason, "violations": total, "request_id": reqID})

		if !metricsEnabled() {
			return
		}
		enforcerDecisions.WithLabelValues(decision).Inc()
		recordRequest(statusCode)
		enforcerReqDuration.WithLabelValues("/check", decision).Observe(time.Since(start).Seconds())
		switch {
		case !cfg.hasRules:
			policyBypass.WithLabelValues("no_rules").Inc()
		case !cfg.requireRules:
			policyBypass.WithLabelValues("config").Inc()
		}
	}
}

func recordRequest(code int) {
	if metricsEnabled() {
		enforcerRequests.WithLabelValues("/check", strconv.Itoa(code)).Inc()
	}
}
