package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"sort"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	pg "github.com/promptshield/promptshield/internal/infrastructure/persistence/postgres"
	"github.com/promptshield/promptshield/internal/rules"
	"gopkg.in/yaml.v3"
)

// Agent authorization and pattern enforcement handlers
func registerAgentHandlers(r chi.Router, opt Options) {
	r.Route("/api/agent", func(ar chi.Router) {
		// Authorization for a tool action (Action-Selector, ArgContracts, RiskRules, Plan-Then-Execute)
		ar.Post("/authorize", func(w http.ResponseWriter, r *http.Request) {
			tenantStr := strings.TrimSpace(r.Header.Get("X-PS-Tenant-ID"))
			if tenantStr == "" || opt.RulepackService == nil || opt.DB == nil {
				writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT", "missing tenant or service not available", nil, r)
				return
			}
			tenantID, err := uuid.Parse(tenantStr)
			if err != nil {
				writeErrorJSON(w, http.StatusBadRequest, "INVALID_TENANT", "bad tenant id", nil, r)
				return
			}

			var req struct {
				ToolID    string          `json:"tool_id"`
				Args      json.RawMessage `json:"args"`
				StepIndex int             `json:"step_index"`
				Plan      json.RawMessage `json:"plan"`
				PlanHash  string          `json:"plan_hash"`
				Lane      string          `json:"lane"` // "privileged"|"quarantined"
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT", "bad json", nil, r)
				return
			}
			if strings.TrimSpace(req.ToolID) == "" {
				writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT", "tool_id required", nil, r)
				return
			}

			// Load active patterns (pick highest priority among active packs that define patterns)
			patterns, preset := loadActivePatterns(r, opt, tenantID)

			// Tools registry
			repo := pg.Tools(opt.DB)
			tool, err := repo.GetByToolID(r.Context(), tenantID, req.ToolID)
			if err != nil {
				writeErrorJSON(w, http.StatusNotFound, "NOT_FOUND", "tool not registered", nil, r)
				return
			}

			decision := map[string]any{"allow": true, "reason": "ok"}

			// Dual-LLM enforcement (lane-based)
			if patterns != nil && patterns.DualLLM != nil && patterns.DualLLM.Enabled {
				lane := strings.ToLower(strings.TrimSpace(req.Lane))
				if lane == "quarantined" {
					if patterns.DualLLM.QuarantinedToolsDisabled {
						decision["allow"] = false
						decision["reason"] = "dual_llm: tools disabled on quarantined lane"
						_ = json.NewEncoder(w).Encode(decision)
						return
					}
					// Otherwise, forbid write-like capability tags
					for _, c := range tool.CapabilityTags {
						c = strings.ToLower(c)
						if strings.Contains(c, "write") || strings.Contains(c, "post") || strings.Contains(c, "payment") || strings.Contains(c, "email-send") {
							decision["allow"] = false
							decision["reason"] = "dual_llm: write-like tools forbidden on quarantined lane"
							_ = json.NewEncoder(w).Encode(decision)
							return
						}
					}
				}
			}

			// Action-Selector enforcement using allowed_tool_query
			if patterns != nil && patterns.ActionSelector != nil && patterns.ActionSelector.Enabled {
				q := strings.TrimSpace(patterns.ActionSelector.AllowedToolQuery)
				if q != "" && !matchToolByQuery(tool, q) {
					decision["allow"] = false
					decision["reason"] = "action_selector: tool not allowed by preset query"
					_ = json.NewEncoder(w).Encode(decision)
					return
				}
				if patterns.ActionSelector.PerActionTimeoutMs > 0 {
					decision["timeout_ms"] = patterns.ActionSelector.PerActionTimeoutMs
				}
			}

			// Arg contracts enforcement (from preset)
			if preset != nil && len(preset.ArgContracts) > 0 {
				var argsObj map[string]any
				if len(req.Args) > 0 {
					_ = json.Unmarshal(req.Args, &argsObj)
				}
				// adapt tool to pgToolLike
				adapter := &pgToolLike{CapabilityTags: tool.CapabilityTags, DataDomains: tool.DataDomains, SideEffect: tool.SideEffect, AuthScope: tool.AuthScope, ArgSchema: tool.ArgSchema}
				if ok, why := validateArgContracts(adapter, preset.ArgContracts, argsObj); !ok {
					decision["allow"] = false
					decision["reason"] = why
					_ = json.NewEncoder(w).Encode(decision)
					return
				}
			}

			// Risk rules enforcement (irreversible requires approval)
			if preset != nil && preset.RiskRules != nil {
				if v, ok := preset.RiskRules["approval_side_effect"].(string); ok {
					if strings.EqualFold(v, "irreversible") && strings.EqualFold(tool.SideEffect, "irreversible") {
						decision["allow"] = false
						decision["reason"] = "approval_required: irreversible side-effect"
						decision["action"] = "quarantine"
						_ = json.NewEncoder(w).Encode(decision)
						return
					}
				}
				if allowlistRaw, ok := preset.RiskRules["domain_allowlist"].([]any); ok {
					// Best-effort URL validation when args include url
					var args map[string]any
					_ = json.Unmarshal(req.Args, &args)
					if u, _ := args["url"].(string); u != "" {
						allowed := false
						for _, x := range allowlistRaw {
							if s, ok := x.(string); ok && strings.Contains(strings.ToLower(u), strings.ToLower(s)) {
								allowed = true
								break
							}
						}
						if !allowed {
							decision["allow"] = false
							decision["reason"] = "domain not allowlisted"
							_ = json.NewEncoder(w).Encode(decision)
							return
						}
					}
				}
			}

			// Plan-Then-Execute enforcement
			if patterns != nil && patterns.PlanThenExecute != nil && patterns.PlanThenExecute.Enabled {
				// Verify step and tool id against plan if provided
				if len(req.Plan) > 0 {
					if ok, why := verifyPlan(req.Plan, req.PlanHash, req.StepIndex, req.ToolID, patterns.PlanThenExecute); !ok {
						decision["allow"] = false
						decision["reason"] = why
						_ = json.NewEncoder(w).Encode(decision)
						return
					}
				} else {
					// No plan when required
					if patterns.PlanThenExecute.MaxSteps > 0 {
						decision["allow"] = false
						decision["reason"] = "plan required but missing"
						_ = json.NewEncoder(w).Encode(decision)
						return
					}
				}
			}

			_ = json.NewEncoder(w).Encode(decision)
		})

		// Context-Minimization policy endpoint for clients to fetch masking instructions
		ar.Get("/context-policy", func(w http.ResponseWriter, r *http.Request) {
			tenantStr := strings.TrimSpace(r.Header.Get("X-PS-Tenant-ID"))
			if tenantStr == "" || opt.RulepackService == nil {
				writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT", "missing tenant or service not available", nil, r)
				return
			}
			tenantID, err := uuid.Parse(tenantStr)
			if err != nil {
				writeErrorJSON(w, http.StatusBadRequest, "INVALID_TENANT", "bad tenant id", nil, r)
				return
			}
			patterns, _ := loadActivePatterns(r, opt, tenantID)
			var resp map[string]any
			if patterns != nil && patterns.ContextMinimization != nil && patterns.ContextMinimization.Enabled {
				resp = map[string]any{
					"enabled":     true,
					"strip_point": firstNonEmpty(patterns.ContextMinimization.StripPoint, "after_tool_selection"),
					"step":        patterns.ContextMinimization.Step,
					"mask_token":  firstNonEmpty(patterns.ContextMinimization.MaskToken, "<USER_TEXT>"),
					"retain":      patterns.ContextMinimization.Retain,
				}
			} else {
				resp = map[string]any{"enabled": false}
			}
			_ = json.NewEncoder(w).Encode(resp)
		})
	})
}

// Utility: Load active patterns/preset for tenant by selecting pack with highest composition priority
func loadActivePatterns(r *http.Request, opt Options, tenantID uuid.UUID) (*rules.Patterns, *rules.Preset) {
	infos, err := opt.RulepackService.List(r.Context(), tenantID)
	if err != nil || len(infos) == 0 {
		return nil, nil
	}
	type packWithPrio struct {
		p    rules.RulePack
		prio int
	}
	var selected *packWithPrio
	for _, info := range infos {
		if !info.Active {
			continue
		}
		dsl, _, err := opt.RulepackService.GetActive(r.Context(), info.ID)
		if err != nil {
			continue
		}
		var rp rules.RulePack
		if err := yaml.Unmarshal(dsl, &rp); err != nil {
			continue
		}
		if rp.Patterns == nil && rp.Preset == nil {
			continue
		}
		prio := 0
		if rp.Composition != nil {
			prio = rp.Composition.Priority
		}
		if selected == nil || prio > selected.prio {
			selected = &packWithPrio{p: rp, prio: prio}
		}
	}
	if selected == nil {
		return nil, nil
	}
	return selected.p.Patterns, selected.p.Preset
}

// Validate argument contracts against tool schema and provided args
func validateArgContracts(tool *pgToolLike, contracts []string, args map[string]any) (bool, string) {
	// Decode ArgSchema: expect { params: [ { name, type, required, enum?:[], pattern?:string } ] }
	type param struct {
		Name     string   `json:"name"`
		Type     string   `json:"type"`
		Required bool     `json:"required"`
		Enum     []string `json:"enum"`
		Pattern  string   `json:"pattern"`
	}
	var schema struct {
		Params []param `json:"params"`
	}
	_ = json.Unmarshal(tool.ArgSchema, &schema)
	paramByName := map[string]param{}
	for _, p := range schema.Params {
		paramByName[strings.ToLower(p.Name)] = p
	}

	hasID := false
	for _, c := range contracts {
		switch strings.ToLower(c) {
		case "ids_only_updates", "ids-only updates", "ids_only":
			// Require an id param and forbid free-text fields without enum/regex
			for k, v := range args {
				key := strings.ToLower(k)
				p, ok := paramByName[key]
				if key == "id" {
					switch v.(type) {
					case string, float64, int, int64:
						hasID = true
					default:
						return false, "ids_only: id must be string or number"
					}
					continue
				}
				if !ok {
					// Unknown param: treat as free-text, block
					return false, "ids_only: unknown param " + key
				}
				t := strings.ToLower(p.Type)
				if t == "string" && len(p.Enum) == 0 && p.Pattern == "" {
					return false, "ids_only: free-text param not allowed: " + key
				}
			}
			if !hasID {
				return false, "ids_only: id param required"
			}
		case "free_text_limited_500":
			// Any string param values must be <= 500 characters
			var keys []string
			for k := range args {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				if s, ok := args[k].(string); ok {
					if len([]rune(s)) > 500 {
						return false, "free_text_limited_500: param too long: " + k
					}
				}
			}
		case "enum_only_templates", "enum-only actions":
			// All string params must be enums in schema
			for k := range args {
				key := strings.ToLower(k)
				if p, ok := paramByName[key]; ok {
					if strings.ToLower(p.Type) == "string" && len(p.Enum) == 0 {
						return false, "enum_only: non-enum string param: " + key
					}
				}
			}
		}
	}
	return true, ""
}

// pgToolLike is satisfied by *domain.Tool; we alias minimal fields here to avoid import cycles in helpers
type pgToolLike struct {
	CapabilityTags []string
	DataDomains    []string
	SideEffect     string
	AuthScope      string
	ArgSchema      json.RawMessage
}

// verifyPlan checks that the requested step/tool matches the plan and optional hash
func verifyPlan(plan json.RawMessage, planHash string, step int, toolID string, cfg *rules.PlanThenExecute) (bool, string) {
	// Optional hash check: SHA-256 hex of minified JSON
	if strings.TrimSpace(planHash) != "" {
		h := sha256.Sum256(minifyJSON(plan))
		if !strings.EqualFold(planHash, hex.EncodeToString(h[:])) {
			return false, "plan hash mismatch"
		}
	}
	// Basic shape: { steps: [ { tool_id: string } ] }
	var p struct {
		Steps []struct {
			ToolID string `json:"tool_id"`
		} `json:"steps"`
	}
	if err := json.Unmarshal(plan, &p); err != nil {
		return false, "bad plan json"
	}
	if cfg.MaxSteps > 0 && len(p.Steps) > cfg.MaxSteps {
		return false, "plan exceeds max steps"
	}
	if step < 0 || step >= len(p.Steps) {
		return false, "invalid step index"
	}
	if strings.TrimSpace(p.Steps[step].ToolID) != strings.TrimSpace(toolID) {
		return false, "step/tool mismatch"
	}
	return true, ""
}

func minifyJSON(b []byte) []byte {
	var m any
	if json.Unmarshal(b, &m) == nil {
		out, _ := json.Marshal(m)
		return out
	}
	return b
}
