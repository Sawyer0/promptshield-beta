package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	pg "github.com/promptshield/promptshield/internal/infrastructure/persistence/postgres"
)

type presetSpec struct {
	ID               string         `json:"id"`
	Name             string         `json:"name"`
	Description      string         `json:"description"`
	AllowedToolQuery string         `json:"allowed_tool_query,omitempty"`
	ArgContracts     []string       `json:"arg_contracts,omitempty"`
	RiskRules        map[string]any `json:"risk_rules,omitempty"`
	Patterns         map[string]any `json:"patterns,omitempty"`
}

func registerPresetHandlers(r chi.Router, opt Options) {
	// Static preset catalog (capability-based)
	presets := []presetSpec{
		{
			ID:               "agent_safe_defaults",
			Name:             "Agent Safe Defaults",
			Description:      "Allow read + GETs only; enable action selector, context minimization, plan-then-execute.",
			AllowedToolQuery: "read AND (network_get OR file-io_read) AND NOT (write OR email-send OR payment OR code-exec)",
			Patterns: map[string]any{
				"action_selector":      map[string]any{"enabled": true, "mode": "enforce"},
				"context_minimization": map[string]any{"enabled": true, "strip_point": "after_tool_selection", "mask_token": "<USER_TEXT>"},
				"plan_then_execute":    map[string]any{"enabled": true, "max_steps": 8, "drift_policy": "block"},
			},
		},
		{
			ID:               "rag_only",
			Name:             "RAG Only",
			Description:      "Read-only retrieval: file-io(read) + network(GET); map-reduce enabled.",
			AllowedToolQuery: "(file-io_read OR network_get) AND NOT (write OR code-exec OR shell OR db-write OR email-send OR payment)",
			Patterns: map[string]any{
				"context_minimization": map[string]any{"enabled": true, "strip_point": "after_tool_selection"},
				"plan_then_execute":    map[string]any{"enabled": true},
				"map_reduce":           map[string]any{"enabled": true, "map_output": "score", "reduce_type": "non_llm"},
			},
		},
		{
			ID:               "update_by_id",
			Name:             "Update-By-ID",
			Description:      "Writes allowed only with id+enum args; plan locked.",
			AllowedToolQuery: "db-write",
			ArgContracts:     []string{"ids_only_updates"},
			Patterns: map[string]any{
				"plan_then_execute":    map[string]any{"enabled": true},
				"context_minimization": map[string]any{"enabled": true},
			},
		},
		{
			ID:          "no_secrets_outbound",
			Name:        "No-Secrets Outbound",
			Description: "Redact secrets in outbound responses; independent of actions.",
			RiskRules:   map[string]any{"outbound_redaction": true},
		},
		{
			ID:          "approval_irreversible",
			Name:        "Human-Approval for Irreversible",
			Description: "Irreversible side-effects require human approval.",
			RiskRules:   map[string]any{"approval_side_effect": "irreversible"},
			Patterns:    map[string]any{"plan_then_execute": map[string]any{"enabled": true}},
		},
		{
			ID:               "web_research_strict",
			Name:             "Web Research (Strict)",
			Description:      "GET-only to allowlisted domains; plan locked.",
			AllowedToolQuery: "network_get AND NOT (write OR payment OR email-send)",
			RiskRules:        map[string]any{"domain_allowlist": []string{"example.com"}},
			Patterns:         map[string]any{"context_minimization": map[string]any{"enabled": true}, "plan_then_execute": map[string]any{"enabled": true}},
		},
		{
			ID:               "customer_support_draft_only",
			Name:             "Customer-Support Draft-Only",
			Description:      "Allow email draft only; free-text limited; plan locked.",
			AllowedToolQuery: "email-send AND side:reversible",
			ArgContracts:     []string{"free_text_limited_500", "enum_only_templates"},
			Patterns:         map[string]any{"plan_then_execute": map[string]any{"enabled": true}},
		},
		{
			ID:               "finance_guardrails",
			Name:             "Finance Guardrails",
			Description:      "Payments limited by amount/currency; vendor allowlist.",
			AllowedToolQuery: "payment",
			RiskRules:        map[string]any{"amount_limit": 1000, "currency_enum": []string{"USD", "EUR"}, "vendor_allowlist": []string{}},
			Patterns:         map[string]any{"plan_then_execute": map[string]any{"enabled": true, "drift_policy": "block"}, "context_minimization": map[string]any{"enabled": true}},
		},
		{
			ID:               "dual_lane_quarantine_search",
			Name:             "Dual-Lane (Quarantine Search)",
			Description:      "Quarantined lane has no write tools; handles-only bridge.",
			AllowedToolQuery: "network_get OR file-io_read",
			Patterns:         map[string]any{"dual_llm": map[string]any{"enabled": true, "quarantined_tools_disabled": true, "bridge_handles_only": true}},
		},
	}

	// List presets
	r.Get("/api/presets", func(w http.ResponseWriter, r *http.Request) {
		out := make([]map[string]any, 0, len(presets))
		for _, p := range presets {
			out = append(out, map[string]any{"id": p.ID, "name": p.Name, "description": p.Description})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": out})
	})

	// Get preset details
	r.Get("/api/presets/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimSpace(chi.URLParam(r, "id"))
		for _, p := range presets {
			if p.ID == id {
				_ = json.NewEncoder(w).Encode(p)
				return
			}
		}
		writeErrorJSON(w, http.StatusNotFound, "NOT_FOUND", "preset not found", nil, r)
	})

	// Preview tools matched by preset's allowed_tool_query
	r.Get("/api/presets/{id}/preview", func(w http.ResponseWriter, r *http.Request) {
		if opt.DB == nil {
			writeErrorJSON(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "database not configured", nil, r)
			return
		}
		id := strings.TrimSpace(chi.URLParam(r, "id"))
		var spec *presetSpec
		for i := range presets {
			if presets[i].ID == id {
				spec = &presets[i]
				break
			}
		}
		if spec == nil {
			writeErrorJSON(w, http.StatusNotFound, "NOT_FOUND", "preset not found", nil, r)
			return
		}
		q := strings.TrimSpace(spec.AllowedToolQuery)
		if q == "" {
			_ = json.NewEncoder(w).Encode(map[string]any{"matched": []any{}, "total": 0})
			return
		}
		tenantID, ok := requireTenantID(w, r)
		if !ok {
			return
		}
		repo := pg.Tools(opt.DB)
		items, _, err := repo.List(r.Context(), tenantID, 0, 1000)
		if err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil, r)
			return
		}
		var matched []any
		for _, t := range items {
			adapter := &pgToolLike{CapabilityTags: t.CapabilityTags, DataDomains: t.DataDomains, SideEffect: t.SideEffect, AuthScope: t.AuthScope}
			if matchToolByQuery(adapter, q) {
				matched = append(matched, t)
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"matched": matched, "total": len(matched)})
	})
}
