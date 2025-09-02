package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	pg "github.com/promptshield/promptshield/internal/infrastructure/persistence/postgres"
	"github.com/promptshield/promptshield/internal/shared/types"
)

// agentEnforcementMiddleware enforces Action-Selector, Dual-LLM, ArgContracts and Plan-Then-Execute
// for requests that declare a tool via X-PS-Tool-ID. If the header is absent, it is a no-op.
func agentEnforcementMiddleware(opt Options) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			toolID := strings.TrimSpace(r.Header.Get("X-PS-Tool-ID"))
			if toolID == "" {
				next.ServeHTTP(w, r)
				return
			}
			if opt.DB == nil || opt.RulepackService == nil {
				setDecisionHeaders(w, "deny", "system", "authorization service not available", nil)
				writeErrorJSON(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "authorization service not available", nil, r)
				return
			}
			tenantStr := strings.TrimSpace(r.Header.Get("X-PS-Tenant-ID"))
			if tenantStr == "" {
				setDecisionHeaders(w, "deny", "tenant", "missing X-PS-Tenant-ID", nil)
				writeErrorJSON(w, http.StatusBadRequest, "MISSING_TENANT", "X-PS-Tenant-ID required", nil, r)
				return
			}
			tenantID, err := uuid.Parse(tenantStr)
			if err != nil {
				setDecisionHeaders(w, "deny", "tenant", "invalid tenant id", nil)
				writeErrorJSON(w, http.StatusBadRequest, "INVALID_TENANT", "bad tenant id", nil, r)
				return
			}

			// Gather plan context from headers (optional)
			lane := strings.TrimSpace(r.Header.Get("X-PS-Lane"))
			planHash := strings.TrimSpace(r.Header.Get("X-PS-Plan-Hash"))
			conversationID := strings.TrimSpace(r.Header.Get("X-PS-Conversation-ID"))
			stepIdx := 0
			if v := strings.TrimSpace(r.Header.Get("X-PS-Plan-Step")); v != "" {
				if n, e := strconv.Atoi(v); e == nil {
					stepIdx = n
				}
			}
			var plan json.RawMessage
			if v := strings.TrimSpace(r.Header.Get("X-PS-Plan")); v != "" {
				plan = json.RawMessage(v)
			}

			// Restore lane or plan from PlanState when not provided
			if opt.PlanState != nil && conversationID != "" {
				if lane == "" {
					if savedLane, _ := opt.PlanState.GetLane(r.Context(), tenantID, conversationID); savedLane != "" {
						lane = savedLane
					}
				}
				if len(plan) == 0 || planHash == "" {
					if p, h, _ := opt.PlanState.GetPlan(r.Context(), tenantID, conversationID); len(p) > 0 && h != "" {
						plan, planHash = p, h
					}
				}
			}

			// Read args from body when JSON
			var args json.RawMessage
			if r.Body != nil && (r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch) {
				bodyCopy, _ := io.ReadAll(r.Body)
				r.Body.Close()
				r.Body = io.NopCloser(bytes.NewReader(bodyCopy))
				// Try to parse JSON; if not JSON, pass empty args
				var js any
				if json.Unmarshal(bodyCopy, &js) == nil {
					args = json.RawMessage(bodyCopy)
				}
			}

			// Load patterns and preset
			patterns, preset := loadActivePatterns(r, opt, tenantID)

			// Load tool from registry
			repo := pg.Tools(opt.DB)
			tool, err := repo.GetByToolID(r.Context(), tenantID, toolID)
			if err != nil {
				setDecisionHeaders(w, "deny", "action_selector", "tool not registered", map[string]string{"X-PS-Reason-Detail": "tool_registry_miss"})
				logAgentAudit(r, opt, tenantID, toolID, "deny", "action_selector", map[string]any{"reason": "tool_registry_miss"})
				writeErrorJSON(w, http.StatusForbidden, "NOT_ALLOWED", "tool not registered", map[string]any{"reason": "tool_registry_miss"}, r)
				return
			}

			// Dual-LLM lane token (if configured, persist lane with TTL)
			laneTTL := 15 * time.Minute
			if patterns != nil && patterns.DualLLM != nil && patterns.DualLLM.Enabled && conversationID != "" && opt.PlanState != nil {
				_ = opt.PlanState.PutLane(r.Context(), tenantID, conversationID, lane, laneTTL)
			}

			// Dual-LLM
			if patterns != nil && patterns.DualLLM != nil && patterns.DualLLM.Enabled {
				l := strings.ToLower(strings.TrimSpace(lane))
				if l == "quarantined" {
					if patterns.DualLLM.QuarantinedToolsDisabled {
						setDecisionHeaders(w, "deny", "dual_llm", "tools disabled on quarantined lane", nil)
						logAgentAudit(r, opt, tenantID, toolID, "deny", "dual_llm", map[string]any{"lane": l})
						writeErrorJSON(w, http.StatusForbidden, "NOT_ALLOWED", "tools disabled on quarantined lane", map[string]any{"policy": "dual_llm"}, r)
						return
					}
					for _, c := range tool.CapabilityTags {
						c = strings.ToLower(c)
						if strings.Contains(c, "write") || strings.Contains(c, "post") || strings.Contains(c, "payment") || strings.Contains(c, "email-send") {
							setDecisionHeaders(w, "deny", "dual_llm", "write-like tools forbidden on quarantined lane", nil)
							logAgentAudit(r, opt, tenantID, toolID, "deny", "dual_llm", map[string]any{"lane": l, "capability": c})
							writeErrorJSON(w, http.StatusForbidden, "NOT_ALLOWED", "write-like tools forbidden on quarantined lane", map[string]any{"policy": "dual_llm"}, r)
							return
						}
					}
				}
			}

			// Action-Selector
			if patterns != nil && patterns.ActionSelector != nil && patterns.ActionSelector.Enabled {
				q := strings.TrimSpace(patterns.ActionSelector.AllowedToolQuery)
				if q != "" && !matchToolByQuery(tool, q) {
					setDecisionHeaders(w, "deny", "action_selector", "tool not allowed by policy", nil)
					logAgentAudit(r, opt, tenantID, toolID, "deny", "action_selector", map[string]any{"query": q})
					writeErrorJSON(w, http.StatusForbidden, "NOT_ALLOWED", "tool not allowed by policy", map[string]any{"policy": "action_selector"}, r)
					return
				}
			}

			// Arg contracts
			if preset != nil && len(preset.ArgContracts) > 0 && len(args) > 0 {
				var argsObj map[string]any
				_ = json.Unmarshal(args, &argsObj)
				adapter := &pgToolLike{CapabilityTags: tool.CapabilityTags, DataDomains: tool.DataDomains, SideEffect: tool.SideEffect, AuthScope: tool.AuthScope, ArgSchema: tool.ArgSchema}
				if ok, why := validateArgContracts(adapter, preset.ArgContracts, argsObj); !ok {
					setDecisionHeaders(w, "deny", "arg_contracts", why, nil)
					logAgentAudit(r, opt, tenantID, toolID, "deny", "arg_contracts", map[string]any{"why": why})
					writeErrorJSON(w, http.StatusForbidden, "NOT_ALLOWED", why, map[string]any{"policy": "arg_contracts"}, r)
					return
				}
			}

			// Risk rules (irreversible approval)
			if preset != nil && preset.RiskRules != nil {
				if v, ok := preset.RiskRules["approval_side_effect"].(string); ok {
					if strings.EqualFold(v, "irreversible") && strings.EqualFold(tool.SideEffect, "irreversible") {
						setDecisionHeaders(w, "quarantine", "risk_rules", "approval required for irreversible side-effect", nil)
						logAgentAudit(r, opt, tenantID, toolID, "quarantine", "risk_rules", map[string]any{"side_effect": tool.SideEffect})
						writeErrorJSON(w, http.StatusForbidden, "QUARANTINE", "approval required for irreversible side-effect", map[string]any{"policy": "risk_rules"}, r)
						return
					}
				}
			}

			// Plan-Then-Execute (persist plan on success for future steps)
			if patterns != nil && patterns.PlanThenExecute != nil && patterns.PlanThenExecute.Enabled {
				if len(plan) == 0 {
					setDecisionHeaders(w, "deny", "plan_then_execute", "plan required but missing", nil)
					logAgentAudit(r, opt, tenantID, toolID, "deny", "plan_then_execute", map[string]any{"step": stepIdx})
					writeErrorJSON(w, http.StatusForbidden, "NOT_ALLOWED", "plan required but missing", map[string]any{"policy": "plan_then_execute"}, r)
					return
				}
				if ok, why := verifyPlan(plan, planHash, stepIdx, toolID, patterns.PlanThenExecute); !ok {
					setDecisionHeaders(w, "deny", "plan_then_execute", why, nil)
					logAgentAudit(r, opt, tenantID, toolID, "deny", "plan_then_execute", map[string]any{"why": why})
					writeErrorJSON(w, http.StatusForbidden, "NOT_ALLOWED", why, map[string]any{"policy": "plan_then_execute"}, r)
					return
				}
				// On success, cache plan and hash for subsequent steps
				if opt.PlanState != nil && conversationID != "" {
					_ = opt.PlanState.PutPlan(r.Context(), tenantID, conversationID, plan, planHash, 30*time.Minute)
				}
			}

			// All checks passed
			extras := map[string]string{}
			if patterns != nil && patterns.ActionSelector != nil && patterns.ActionSelector.PerActionTimeoutMs > 0 {
				extras["X-PS-Timeout"] = strconv.Itoa(patterns.ActionSelector.PerActionTimeoutMs)
			}
			setDecisionHeaders(w, "allow", "none", "ok", extras)
			logAgentAudit(r, opt, tenantID, toolID, "allow", "none", map[string]any{"timeout_ms": patterns.ActionSelector.PerActionTimeoutMs})
			next.ServeHTTP(w, r)
		})
	}
}

func logAgentAudit(r *http.Request, opt Options, tenantID uuid.UUID, toolID string, decision string, policy string, meta map[string]any) {
	if opt.AuditLogger == nil {
		return
	}
	m := map[string]any{
		"decision": decision,
		"policy":   policy,
		"tool_id":  toolID,
		"endpoint": r.URL.Path,
		"method":   r.Method,
	}
	for k, v := range meta {
		m[k] = v
	}
	var act types.AuditAction
	if strings.EqualFold(decision, "allow") {
		act = types.AuditActionRequestAllowed
	} else {
		act = types.AuditActionRequestBlocked
	}
	opt.AuditLogger.LogWithContext(r.Context(), types.AuditEvent{
		TenantID:   &tenantID,
		ActorType:  types.ActorTypeSystem,
		Action:     string(act),
		ObjectType: string(types.ObjectTypeRequest),
		ObjectID:   uuid.New(),
		Metadata:   m,
		Timestamp:  time.Now(),
	})
}

// setDecisionHeaders standardizes decision headers on responses
func setDecisionHeaders(w http.ResponseWriter, decision, policy, reason string, extras map[string]string) {
	if w == nil {
		return
	}
	if decision != "" {
		w.Header().Set("X-PS-Decision", decision)
	}
	if policy != "" {
		w.Header().Set("X-PS-Policy", policy)
	}
	if reason != "" {
		w.Header().Set("X-PS-Reason", reason)
	}
	for k, v := range extras {
		if strings.TrimSpace(k) != "" && strings.TrimSpace(v) != "" {
			w.Header().Set(k, v)
		}
	}
}
