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
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/license"
	"github.com/promptshield/promptshield/internal/observability/metrics"
	"github.com/promptshield/promptshield/internal/rules"
	"github.com/promptshield/promptshield/internal/scanner"
	semdeberta "github.com/promptshield/promptshield/internal/semantic/deberta"
	semfake "github.com/promptshield/promptshield/internal/semantic/fake"
	ckeys "github.com/promptshield/promptshield/internal/shared/contextkeys"
	"github.com/promptshield/promptshield/internal/shared/types"
	pkg "github.com/promptshield/promptshield/pkg/types"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// conversation state (minimal) for drift signals
var convStore sync.Map // map[string]*convState

type convState struct {
	LastText  string
	UpdatedAt time.Time
}

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

		// PDP gate for message scanning/sending
		failClosed := !strings.EqualFold(strings.TrimSpace(os.Getenv("PS_PDP_FAIL_OPEN_CHECK")), "true")
if ok, reason := authorizePDP(r, "message.send", "message", "", map[string]any{"endpoint": r.URL.Path, "method": r.Method}, failClosed); !ok {
			writeErrorJSON(w, http.StatusForbidden, "PDP_DENY", "not authorized: "+reason, nil, r)
			return
		}

		// tenant in context for downstream
		ctx, cancel := context.WithTimeout(ctxSpan, 2*time.Second)
		defer cancel()
		ctx = context.WithValue(ctx, ckeys.TenantID, tenantID)

		// Enforce body limits
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

		// Endpoint context for assignment-scoped rulepacks
		endpoint := strings.TrimSpace(r.Header.Get("X-PS-Endpoint"))
		if endpoint == "" {
			endpoint = strings.TrimSpace(r.URL.Query().Get("endpoint"))
		}
		ctx = context.WithValue(ctx, ckeys.EndpointPath, endpoint)

		// Modes: NDJSON (aggregate=false), JSON array (aggregate), or single
		aggregate := true
		if a := r.URL.Query().Get("aggregate"); a != "" {
			aggregate = a != "false"
		}
		ct := r.Header.Get("content-type")
		if strings.HasPrefix(ct, "application/x-ndjson") && !aggregate {
			w.Header().Set("content-type", "application/x-ndjson")
			bw := bufio.NewWriter(w)
			defer bw.Flush()
			s := bufio.NewScanner(r.Body)
			buf := make([]byte, 0, 1024*1024)
			s.Buffer(buf, 10*1024*1024)
			for s.Scan() {
				line := s.Bytes()
				base := ctx
				ctxLine, cancelLine := context.WithTimeout(base, 2*time.Second)
res := runScanLine(ctxLine, line, strings.TrimSpace(r.Header.Get("X-PS-Conversation-ID")), opt, endpoint, r.Method)
				cancelLine()
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
			metrics.ScanRequestDuration.WithLabelValues("ndjson").Observe(2.0)
			return
		}

		// Read entire body for JSON array or single
		body, _ := io.ReadAll(r.Body)
		// Try JSON array aggregate mode
		var rawArray []json.RawMessage
		if len(body) > 0 && body[0] == '[' && json.Unmarshal(body, &rawArray) == nil && len(rawArray) > 0 {
			var (
				decisions       []map[string]any
				totalViolations int
			)
for _, item := range rawArray {
				res := runScanLine(ctx, item, strings.TrimSpace(r.Header.Get("X-PS-Conversation-ID")), opt, endpoint, r.Method)
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
			metrics.ScanRequestDuration.WithLabelValues("aggregate").Observe(2.0)
			return
		}

		// Single payload enforcement
		logger := slog.With("component", "api-check")
		var res pkg.ScanResult
		var err error
		// Prefer assignment-scoped packs when repositories are configured
		if opt.AssignmentRepository != nil && opt.RulepackService != nil {
			logger.Info("Using assignment-scoped rulepacks", "endpoint", endpoint)
			sc := scanner.ScanEngineCstor(0)
			sc.SetBaseContext(ctx)
			if os.Getenv("PS_SEMANTIC_ENABLED") == "true" {
				if v := strings.ToLower(strings.TrimSpace(os.Getenv("PS_FAKE_L3"))); v == "1" || v == "true" || v == "yes" {
					var d time.Duration
					if ms := strings.TrimSpace(os.Getenv("PS_FAKE_L3_DELAY_MS")); ms != "" {
						if n, err := strconv.Atoi(ms); err == nil && n > 0 { d = time.Duration(n) * time.Millisecond }
					}
					sc.SetSemanticAnalyzer(semfake.Analyzer{Delay: d})
				} else if ep := strings.TrimSpace(os.Getenv("PS_DEBERTA_ENDPOINT")); ep != "" {
					analyzer := semdeberta.New(semdeberta.Options{Endpoint: ep, APIKey: strings.TrimSpace(os.Getenv("HF_TOKEN"))})
					sc.SetSemanticAnalyzer(analyzer)
				}
			}
packs, perr := resolveApplicableRulepacks(ctx, opt, endpoint, r.Method)
			if perr == nil && len(packs) > 0 {
				sc.LoadRulePacks(packs)
			} else if rp := os.Getenv("PS_ENFORCER_RULEPACK"); rp != "" {
				if packs, e := rules.LoadPacks(rp); e == nil { sc.LoadRulePacks(packs) }
			}
			res, err = sc.ScanReader(ctx, bytes.NewReader(body), "http:v1:check:assignment")
		} else if opt.ScannerManager != nil && opt.ScannerManager.HasActivePolicies() {
			logger.Info("Using database-loaded rulepack scanner")
			res, err = opt.ScannerManager.ScanReader(ctx, bytes.NewReader(body), "http:v1:check:database")
		} else {
			logger.Info("No active rulepacks - allowing request (fail-open)")
			res = pkg.ScanResult{Violations: []pkg.Violation{}, ScanInfo: pkg.ScanInfo{ShouldBlock: false, BlockReason: "no_rulepacks_loaded", TotalViolations: 0, ScanStatus: "success"}}
		}
		if err != nil {
			msg := err.Error()
			code := http.StatusInternalServerError
			if strings.Contains(strings.ToLower(msg), "request body too large") { code = http.StatusBadRequest }
			errCode := "INTERNAL"; if code == http.StatusBadRequest { errCode = "INVALID_ARGUMENT" }
			writeError(w, code, errCode, "scan failed", map[string]any{"error": msg})
			return
		}
		decision := "allow"; reason := "no_signals"; total := len(res.Violations)
		anyQuarantine := false; anyDeny := false; firstRule := ""
		for _, v := range res.Violations {
			if firstRule == "" { firstRule = v.RuleID }
			switch v.ResponseAction { case "deny", "block": anyDeny = true; case "quarantine": anyQuarantine = true }
		}
		if anyDeny { decision = "deny"; reason = firstNonEmpty(firstRule, "response_action") } else if anyQuarantine { decision = "quarantine"; reason = firstNonEmpty(firstRule, "signals_detected") } else {
			finalScore, _, _ := computeFinalScore(&res)
			if finalScore >= parseFloatEnv("PS_BLOCK_THRESHOLD", 0.75) { decision = "quarantine"; reason = "policy_bridge_threshold" }
		}
		if id := r.Header.Get("X-Request-ID"); id != "" { w.Header().Set("x-ps-request-id", id) }
		w.Header().Set("x-ps-decision", decision); w.Header().Set("x-ps-reason", reason); w.Header().Set("content-type", "application/json")
		statusCode := http.StatusOK; if decision != "allow" { statusCode = http.StatusForbidden }
		w.WriteHeader(statusCode)
		_ = json.NewEncoder(w).Encode(decisionResponse{Decision: decision, Reason: reason, Violations: total, RequestID: r.Header.Get("X-Request-ID")})
		respMap := map[string]any{"decision": decision, "reason": reason, "violations": total, "request_id": r.Header.Get("X-Request-ID")}
		// Span + publish + audit
		ev := "ps.decision.allow"; if decision == "quarantine" || decision == "deny" { ev = "ps.decision.block" }
		span.SetAttributes(attribute.String("decision", decision), attribute.String("reason", reason), attribute.Int("ps.violations", total))
		span.AddEvent(ev, trace.WithAttributes(attribute.String("reason", reason), attribute.Int("ps.violations", total)))
		if opt.Events != nil { opt.Events.Publish(Event{Type: "decision", Data: respMap}) }
		if opt.AuditLogger != nil { _ = opt.AuditLogger.LogWithContext(ctx, types.AuditEvent{Action: "request.decision", ObjectType: "request", ObjectID: uuid.New(), Metadata: map[string]any{"path": r.URL.Path, "status": statusCode, "decision": decision, "reason": reason, "violations": total}, Timestamp: time.Now().UTC()}) }
		if store := getUsageStoreFromCtx(r.Context()); store != nil { var bytesN int64; if cl := r.Header.Get("Content-Length"); cl != "" { if n, err := strconv.ParseInt(cl, 10, 64); err == nil { bytesN = n } }; _ = store.Record(r.Context(), r.Header.Get("x-tenant-id"), r.URL.Path, decision, bytesN, time.Now()) }
	}
}

func runScanLine(ctx context.Context, data []byte, convID string, opt Options, endpoint, method string) map[string]any {
	// Opportunistic prune of conversation store based on TTL
	pruneConversationStore(convTTL())
sc := scanner.ScanEngineCstor(0)
	sc.SetBaseContext(ctx)
	// Attempt assignment-based rulepack loading
	loadedFromAssignments := false
	if opt.AssignmentRepository != nil && opt.RulepackService != nil {
if packs, err := resolveApplicableRulepacks(ctx, opt, endpoint, method); err == nil && len(packs) > 0 {
			sc.LoadRulePacks(packs)
			loadedFromAssignments = true
		}
	}
	// Inject conversation signals (simple drift/privilege jump heuristics)
	setConversationSignals(sc, convID, string(data))
	// Initialize semantic analyzer (DeBERTa for /scan path)
	if os.Getenv("PS_SEMANTIC_ENABLED") == "true" {
		// Optional deterministic fake analyzer for tests
		if v := strings.ToLower(strings.TrimSpace(os.Getenv("PS_FAKE_L3"))); v == "1" || v == "true" || v == "yes" {
			var d time.Duration
			if ms := strings.TrimSpace(os.Getenv("PS_FAKE_L3_DELAY_MS")); ms != "" {
				if n, err := strconv.Atoi(ms); err == nil && n > 0 { d = time.Duration(n) * time.Millisecond }
			}
			sc.SetSemanticAnalyzer(semfake.Analyzer{Delay: d})
		} else if ep := strings.TrimSpace(os.Getenv("PS_DEBERTA_ENDPOINT")); ep != "" {
			analyzer := semdeberta.New(semdeberta.Options{Endpoint: ep, APIKey: strings.TrimSpace(os.Getenv("HF_TOKEN"))})
			sc.SetSemanticAnalyzer(analyzer)
		}
	}
// Fallback to file-based packs when no assignment matched
	if !loadedFromAssignments {
		if rp := os.Getenv("PS_ENFORCER_RULEPACK"); rp != "" {
			if packs, e := rules.LoadPacks(rp); e == nil {
				sc.LoadRulePacks(packs)
			}
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
		} else if anyQuarantine {
			decision = "quarantine"
			reason = firstNonEmpty(firstRule, "signals_detected")
		} else {
			finalScore, _, _ := computeFinalScore(&res)
			threshold := parseFloatEnv("PS_BLOCK_THRESHOLD", 0.75)
			if finalScore >= threshold {
				decision = "quarantine"
				reason = "policy_bridge_threshold"
			}
		}
	}
	// Policy bridge: weighted score override (no user-facing rationale)
	finalScore, riskScore, patternScore := computeFinalScore(&res)
	alpha := parseFloatEnv("PS_ALPHA", 0.7)
	beta := parseFloatEnv("PS_BETA", 0.3)
	_ = alpha; _ = beta // already applied in computeFinalScore
	threshold := parseFloatEnv("PS_BLOCK_THRESHOLD", 0.75)
	if decision == "allow" && finalScore >= threshold {
		decision = "quarantine"
		reason = "policy_bridge_threshold"
	}
	// Internal probe rationale (short, ephemeral, logged only)
	rationale := buildInternalRationale(convID, riskScore, patternScore, finalScore, &res)
	if logger := slog.With("component", "api-scan"); true {
		logger.Info("security_probe",
			"conv_id", convID,
			"risk", fmtFloat(riskScore),
			"pattern", fmtFloat(patternScore),
			"final", fmtFloat(finalScore),
			"rationale", rationale,
		)
	}
	// Update conversation state
	updateConversation(convID, string(data))
	return map[string]any{"decision": decision, "reason": reason, "violations": total}
}

// setConversationSignals computes simple drift flags and sets runtime context gating
func setConversationSignals(sc *scanner.Scanner, convID, text string) {
	ctx := map[string]string{"conv_drift": "low", "conv_priv_jump": "false"}
	if convID != "" {
		if v, ok := convStore.Load(convID); ok {
			prev := v.(*convState)
			if driftHigh(prev.LastText, text) {
				ctx["conv_drift"] = "high"
			}
			// Heuristic privileged jump: if text contains common privileged tokens and prior didn't
			if privilegedTokens(text) && !privilegedTokens(prev.LastText) {
				ctx["conv_priv_jump"] = "true"
			}
		}
	}
	sc.SetRuntimeContext(ctx)
}

func updateConversation(convID, text string) {
	if convID == "" { return }
	convStore.Store(convID, &convState{LastText: truncate(text, 800), UpdatedAt: time.Now()})
}

func driftHigh(prev, curr string) bool {
	// simple token Jaccard distance > 0.8 → high drift
	pt := tokenize(prev); ct := tokenize(curr)
	if len(pt) == 0 || len(ct) == 0 { return false }
	set := make(map[string]struct{}, len(pt))
	for _, t := range pt { set[t] = struct{}{} }
	inter := 0
	for _, t := range ct { if _, ok := set[t]; ok { inter++ } }
	union := len(pt) + len(ct) - inter
	if union == 0 { return false }
	d := 1.0 - float64(inter)/float64(union)
	return d > 0.8
}

func tokenize(s string) []string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "\n", " ")
	parts := strings.Fields(s)
	if len(parts) > 128 { parts = parts[:128] }
	return parts
}

func privilegedTokens(s string) bool {
	ls := strings.ToLower(s)
	return strings.Contains(ls, "export all pii") || strings.Contains(ls, "dump database") || strings.Contains(ls, "system prompt")
}

func truncate(s string, n int) string {
	if len(s) <= n { return s }
	return s[:n]
}

func parseFloatEnv(k string, def float64) float64 {
	v := strings.TrimSpace(os.Getenv(k))
	if v == "" { return def }
	if f, err := strconv.ParseFloat(v, 64); err == nil { return f }
	return def
}

func fmtFloat(f float64) string { return strconv.FormatFloat(f, 'f', 3, 64) }

func convTTL() time.Duration {
	v := strings.TrimSpace(os.Getenv("PS_CONV_TTL"))
	if v == "" {
		return 15 * time.Minute
	}
	d, err := time.ParseDuration(v)
	if err != nil { return 15 * time.Minute }
	return d
}

func pruneConversationStore(ttl time.Duration) {
	if ttl <= 0 { return }
	cut := time.Now().Add(-ttl)
	count := 0
	convStore.Range(func(key, value any) bool {
		cs, ok := value.(*convState)
		if !ok { convStore.Delete(key); return true }
		if cs.UpdatedAt.Before(cut) {
			convStore.Delete(key)
		}
		count++
		// Bound work per call
		return count < 500
	})
}

// computeFinalScore calculates final score = α·risk + β·pattern
// risk = max confidence among L3 violations
// pattern = 1.0 if any L2 matches; 0.5 if only L1; else 0.0
func computeFinalScore(r *pkg.ScanResult) (final, risk, pattern float64) {
	alpha := parseFloatEnv("PS_ALPHA", 0.7)
	beta := parseFloatEnv("PS_BETA", 0.3)
	var hasL2, hasL1 bool
	for _, v := range r.Violations {
		if v.Level == 3 && v.Confidence > risk { risk = v.Confidence }
		if v.Level == 2 { hasL2 = true }
		if v.Level == 1 { hasL1 = true }
	}
	if hasL2 { pattern = 1.0 } else if hasL1 { pattern = 0.5 } else { pattern = 0.0 }
	final = alpha*risk + beta*pattern
	return
}

// buildInternalRationale creates a compact probe rationale for logs only
func buildInternalRationale(convID string, risk, pattern, final float64, r *pkg.ScanResult) string {
	ruleCount := len(r.Violations)
	return "risk=" + fmtFloat(risk) + ",pattern=" + fmtFloat(pattern) + ",final=" + fmtFloat(final) + ",rules=" + strconv.Itoa(ruleCount)
}

// stringFromAny converts interface{} to string best-effort
func stringFromAny(v any) string {
	s, _ := v.(string)
	return s
}
