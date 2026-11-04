package api

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/promptshield/promptshield/internal/application/decisions"
)

const maxDecisionLimit = 500

// RecordDecision stores a decision in the log and broadcasts it as a
// "decision" event. Both /check and /v1/check report through here.
func RecordDecision(opt Options, rec decisions.Record) {
	if opt.Decisions != nil {
		opt.Decisions.Add(rec)
	}
	if opt.Events != nil {
		opt.Events.Publish(Event{Type: "decision", Data: rec.EventData(), Time: rec.Time})
	}
}

// mountDecisions registers the decision history endpoints. Callers apply
// admin auth.
//
//	GET /decisions?limit=&decision=  recent decisions, newest first
//	GET /decisions/stats             lifetime counts and latency percentiles
//	GET /decisions/{requestID}       one decision
func mountDecisions(r chi.Router, opt Options) {
	r.Get("/decisions", func(w http.ResponseWriter, req *http.Request) {
		q, err := parseDecisionQuery(req)
		if err != nil {
			writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT", err.Error(), nil, req)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"decisions": opt.Decisions.Recent(q)}, req)
	})
	r.Get("/decisions/stats", func(w http.ResponseWriter, req *http.Request) {
		writeJSON(w, http.StatusOK, opt.Decisions.Stats(), req)
	})
	r.Get("/decisions/{requestID}", func(w http.ResponseWriter, req *http.Request) {
		// Router-generated ids look like "host/abc-000001", so clients send
		// the slash escaped; chi leaves the parameter in its raw form.
		id, err := url.PathUnescape(chi.URLParam(req, "requestID"))
		if err != nil {
			writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT", "invalid request id", nil, req)
			return
		}
		rec, ok := opt.Decisions.Get(id)
		if !ok {
			writeErrorJSON(w, http.StatusNotFound, "NOT_FOUND", "decision not found or no longer retained", nil, req)
			return
		}
		writeJSON(w, http.StatusOK, rec, req)
	})
}

type queryError string

func (e queryError) Error() string { return string(e) }

func parseDecisionQuery(req *http.Request) (decisions.Query, error) {
	q := decisions.Query{Limit: 100}
	if v := strings.TrimSpace(req.URL.Query().Get("limit")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > maxDecisionLimit {
			return q, queryError("limit must be an integer between 1 and 500")
		}
		q.Limit = n
	}
	switch d := strings.TrimSpace(req.URL.Query().Get("decision")); d {
	case "", decisions.Allow, decisions.Deny, decisions.Quarantine:
		q.Decision = d
	default:
		return q, queryError("decision must be allow, deny, or quarantine")
	}
	return q, nil
}
