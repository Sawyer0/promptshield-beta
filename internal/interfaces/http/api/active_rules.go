package api

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/promptshield/promptshield/internal/application/rulesets"
)

// mountActiveRules registers endpoints for the RulePacks /check enforces.
// Callers apply admin auth.
//
//	GET  /rules         display-safe summary of the active packs
//	POST /rules/reload  re-read PS_ENFORCER_RULEPACK and swap it in
func mountActiveRules(r chi.Router, opt Options) {
	r.Get("/rules", func(w http.ResponseWriter, req *http.Request) {
		writeJSON(w, http.StatusOK, activeRulesSnapshot(opt), req)
	})
	r.Post("/rules/reload", func(w http.ResponseWriter, req *http.Request) {
		if opt.ActiveRules == nil {
			writeErrorJSON(w, http.StatusConflict, "RULES_NOT_CONFIGURED", "gateway has no reloadable rulepack set", nil, req)
			return
		}
		snap, err := opt.ActiveRules.Reload()
		switch {
		case errors.Is(err, rulesets.ErrNoSource):
			writeErrorJSON(w, http.StatusConflict, "RULES_NOT_CONFIGURED", "PS_ENFORCER_RULEPACK is not set", nil, req)
			return
		case err != nil:
			writeErrorJSON(w, http.StatusUnprocessableEntity, "RULEPACK_INVALID", "reload failed; previous rules stay active", map[string]any{"error": err.Error()}, req)
			return
		}
		if opt.Events != nil {
			opt.Events.Publish(Event{Type: "rules_reloaded", Data: map[string]any{
				"generation": snap.Generation,
				"rule_count": snap.RuleCount,
			}})
		}
		writeJSON(w, http.StatusOK, snap, req)
	})
}

func activeRulesSnapshot(opt Options) rulesets.Snapshot {
	if opt.ActiveRules == nil {
		return rulesets.Snapshot{Packs: []rulesets.Pack{}}
	}
	return opt.ActiveRules.Snapshot()
}
