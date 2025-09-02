package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/domain"
	pg "github.com/promptshield/promptshield/internal/infrastructure/persistence/postgres"
)

// registerToolHandlers mounts /api/tools CRUD endpoints backed by Postgres.
func registerToolHandlers(r chi.Router, opt Options) {
	r.Route("/api/tools", func(tr chi.Router) {
		if opt.DB == nil {
			tr.Get("/", func(w http.ResponseWriter, r *http.Request) {
				writeErrorJSON(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "database not configured", nil, r)
			})
			tr.Post("/", func(w http.ResponseWriter, r *http.Request) {
				writeErrorJSON(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "database not configured", nil, r)
			})
			tr.Get("/{id}", func(w http.ResponseWriter, r *http.Request) {
				writeErrorJSON(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "database not configured", nil, r)
			})
			tr.Put("/{id}", func(w http.ResponseWriter, r *http.Request) {
				writeErrorJSON(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "NOT_IMPLEMENTED", nil, r)
			})
			tr.Delete("/{id}", func(w http.ResponseWriter, r *http.Request) {
				writeErrorJSON(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "database not configured", nil, r)
			})
			return
		}

		repo := pg.Tools(opt.DB)

		// List
		tr.Get("/", func(w http.ResponseWriter, r *http.Request) {
			tenantStr := r.Header.Get("X-PS-Tenant-ID")
			if tenantStr == "" {
				writeErrorJSON(w, http.StatusBadRequest, "MISSING_TENANT", "X-PS-Tenant-ID required", nil, r)
				return
			}
			tenantID, err := uuid.Parse(tenantStr)
			if err != nil {
				writeErrorJSON(w, http.StatusBadRequest, "INVALID_TENANT", "bad tenant id", nil, r)
				return
			}
			// pagination
			off, lim := 0, 50
			if v := strings.TrimSpace(r.URL.Query().Get("offset")); v != "" {
				if n, e := strconv.Atoi(v); e == nil && n >= 0 {
					off = n
				}
			}
			if v := strings.TrimSpace(r.URL.Query().Get("limit")); v != "" {
				if n, e := strconv.Atoi(v); e == nil && n > 0 && n <= 200 {
					lim = n
				}
			}
			items, total, err := repo.List(r.Context(), tenantID, off, lim)
			if err != nil {
				writeErrorJSON(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil, r)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"data": items, "total": total, "offset": off, "limit": lim})
		})

		// Preview matched tools by boolean query over tags/traits
		tr.Get("/preview", func(w http.ResponseWriter, r *http.Request) {
			tenantStr := r.Header.Get("X-PS-Tenant-ID")
			if tenantStr == "" {
				writeErrorJSON(w, http.StatusBadRequest, "MISSING_TENANT", "X-PS-Tenant-ID required", nil, r)
				return
			}
			tenantID, err := uuid.Parse(tenantStr)
			if err != nil {
				writeErrorJSON(w, http.StatusBadRequest, "INVALID_TENANT", "bad tenant id", nil, r)
				return
			}
			q := strings.TrimSpace(r.URL.Query().Get("query"))
			if q == "" {
				_ = json.NewEncoder(w).Encode(map[string]any{"matched": []any{}, "total": 0})
				return
			}
			// Fetch up to a reasonable cap per request for preview
			items, _, err := repo.List(r.Context(), tenantID, 0, 1000)
			if err != nil {
				writeErrorJSON(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil, r)
				return
			}
			var matched []any
			for _, t := range items {
				if matchToolByQuery(t, q) {
					matched = append(matched, t)
				}
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"matched": matched, "total": len(matched)})
		})

		// Create
		tr.Post("/", func(w http.ResponseWriter, r *http.Request) {
			tenantStr := r.Header.Get("X-PS-Tenant-ID")
			if tenantStr == "" {
				writeErrorJSON(w, http.StatusBadRequest, "MISSING_TENANT", "X-PS-Tenant-ID required", nil, r)
				return
			}
			tenantID, err := uuid.Parse(tenantStr)
			if err != nil {
				writeErrorJSON(w, http.StatusBadRequest, "INVALID_TENANT", "bad tenant id", nil, r)
				return
			}
			var body struct {
				ToolID         string          `json:"tool_id"`
				Name           string          `json:"name"`
				Description    string          `json:"description"`
				CapabilityTags []string        `json:"capability_tags"`
				DataDomains    []string        `json:"data_domains"`
				SideEffect     string          `json:"side_effect"`
				AuthScope      string          `json:"auth_scope"`
				ArgSchema      json.RawMessage `json:"arg_schema"`
				RiskScore      *int            `json:"risk_score"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT", "bad json", nil, r)
				return
			}
			if strings.TrimSpace(body.ToolID) == "" || strings.TrimSpace(body.Name) == "" {
				writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT", "tool_id and name required", nil, r)
				return
			}
			t := &struct {
				ID string `json:"id"`
			}{}
			dt := &domain.Tool{
				TenantID:       tenantID,
				ToolID:         body.ToolID,
				Name:           body.Name,
				Description:    body.Description,
				CapabilityTags: body.CapabilityTags,
				DataDomains:    body.DataDomains,
				SideEffect:     body.SideEffect,
				AuthScope:      body.AuthScope,
				ArgSchema:      body.ArgSchema,
				RiskScore:      body.RiskScore,
			}
			if err := repo.Create(r.Context(), dt); err != nil {
				writeErrorJSON(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil, r)
				return
			}
			t.ID = dt.ID.String()
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(t)
		})

		// Get
		tr.Get("/{id}", func(w http.ResponseWriter, r *http.Request) {
			tenantStr := r.Header.Get("X-PS-Tenant-ID")
			if tenantStr == "" {
				writeErrorJSON(w, http.StatusBadRequest, "MISSING_TENANT", "X-PS-Tenant-ID required", nil, r)
				return
			}
			tenantID, err := uuid.Parse(tenantStr)
			if err != nil {
				writeErrorJSON(w, http.StatusBadRequest, "INVALID_TENANT", "bad tenant id", nil, r)
				return
			}
			id, err := uuid.Parse(chi.URLParam(r, "id"))
			if err != nil {
				writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT", "bad id", nil, r)
				return
			}
			tool, err := repo.Get(r.Context(), tenantID, id)
			if err != nil {
				writeErrorJSON(w, http.StatusNotFound, "NOT_FOUND", "tool not found", nil, r)
				return
			}
			_ = json.NewEncoder(w).Encode(tool)
		})

		// Update
		tr.Put("/{id}", func(w http.ResponseWriter, r *http.Request) {
			tenantStr := r.Header.Get("X-PS-Tenant-ID")
			if tenantStr == "" {
				writeErrorJSON(w, http.StatusBadRequest, "MISSING_TENANT", "X-PS-Tenant-ID required", nil, r)
				return
			}
			tenantID, err := uuid.Parse(tenantStr)
			if err != nil {
				writeErrorJSON(w, http.StatusBadRequest, "INVALID_TENANT", "bad tenant id", nil, r)
				return
			}
			id, err := uuid.Parse(chi.URLParam(r, "id"))
			if err != nil {
				writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT", "bad id", nil, r)
				return
			}
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT", "bad json", nil, r)
				return
			}
			// fetch current
			tool, err := repo.Get(r.Context(), tenantID, id)
			if err != nil {
				writeErrorJSON(w, http.StatusNotFound, "NOT_FOUND", "tool not found", nil, r)
				return
			}
			// apply updates
			if v, ok := body["tool_id"].(string); ok {
				tool.ToolID = v
			}
			if v, ok := body["name"].(string); ok {
				tool.Name = v
			}
			if v, ok := body["description"].(string); ok {
				tool.Description = v
			}
			if v, ok := body["capability_tags"].([]any); ok {
				tool.CapabilityTags = toStringSliceUnsafe(v)
			}
			if v, ok := body["data_domains"].([]any); ok {
				tool.DataDomains = toStringSliceUnsafe(v)
			}
			if v, ok := body["side_effect"].(string); ok {
				tool.SideEffect = v
			}
			if v, ok := body["auth_scope"].(string); ok {
				tool.AuthScope = v
			}
			if v, ok := body["arg_schema"]; ok {
				b, _ := json.Marshal(v)
				tool.ArgSchema = b
			}
			if v, ok := body["risk_score"].(float64); ok {
				n := int(v)
				tool.RiskScore = &n
			}
			if err := repo.Update(r.Context(), tool); err != nil {
				writeErrorJSON(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil, r)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		})

		// Delete
		tr.Delete("/{id}", func(w http.ResponseWriter, r *http.Request) {
			tenantStr := r.Header.Get("X-PS-Tenant-ID")
			if tenantStr == "" {
				writeErrorJSON(w, http.StatusBadRequest, "MISSING_TENANT", "X-PS-Tenant-ID required", nil, r)
				return
			}
			tenantID, err := uuid.Parse(tenantStr)
			if err != nil {
				writeErrorJSON(w, http.StatusBadRequest, "INVALID_TENANT", "bad tenant id", nil, r)
				return
			}
			id, err := uuid.Parse(chi.URLParam(r, "id"))
			if err != nil {
				writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT", "bad id", nil, r)
				return
			}
			if err := repo.Delete(r.Context(), tenantID, id); err != nil {
				writeErrorJSON(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil, r)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		})
	})
}

// Helpers
func toStringSliceUnsafe(v []any) []string {
	out := make([]string, 0, len(v))
	for _, x := range v {
		if s, ok := x.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// Boolean expression evaluator for tool tags/traits
func matchToolByQuery(t *domain.Tool, query string) bool {
	q := strings.TrimSpace(strings.ToLower(strings.ReplaceAll(query, "-", "_")))
	if q == "" {
		return true
	}
	tagset := map[string]struct{}{}
	for _, s := range t.CapabilityTags {
		tagset[strings.ToLower(strings.ReplaceAll(s, "-", "_"))] = struct{}{}
	}
	for _, s := range t.DataDomains {
		tagset["domain:"+strings.ToLower(strings.ReplaceAll(s, "-", "_"))] = struct{}{}
	}
	tagset["side:"+strings.ToLower(strings.ReplaceAll(t.SideEffect, "-", "_"))] = struct{}{}
	tagset["auth:"+strings.ToLower(strings.ReplaceAll(t.AuthScope, "-", "_"))] = struct{}{}

	evalSym := func(sym string) bool {
		_, ok := tagset[sym]
		return ok
	}
	type token struct{ kind, val string }
	var toks []token
	i := 0
	for i < len(q) {
		switch q[i] {
		case ' ':
			i++
		case '(':
			toks = append(toks, token{kind: "(", val: "("})
			i++
		case ')':
			toks = append(toks, token{kind: ")", val: ")"})
			i++
		default:
			j := i
			for j < len(q) && !strings.ContainsRune(" ()", rune(q[j])) {
				j++
			}
			word := q[i:j]
			up := strings.ToUpper(word)
			if up == "AND" || up == "OR" || up == "NOT" {
				toks = append(toks, token{kind: up, val: up})
			} else {
				toks = append(toks, token{kind: "SYM", val: word})
			}
			i = j
		}
	}
	var pos int
	var parseExpr func() bool
	var parseTerm func() bool
	var parseFactor func() bool
	parseExpr = func() bool {
		v := parseTerm()
		for pos < len(toks) && toks[pos].kind == "OR" {
			pos++
			v = v || parseTerm()
		}
		return v
	}
	parseTerm = func() bool {
		v := parseFactor()
		for pos < len(toks) && toks[pos].kind == "AND" {
			pos++
			v = v && parseFactor()
		}
		return v
	}
	parseFactor = func() bool {
		if pos < len(toks) && toks[pos].kind == "NOT" {
			pos++
			return !parseFactor()
		}
		if pos < len(toks) && toks[pos].kind == "(" {
			pos++
			v := parseExpr()
			if pos < len(toks) && toks[pos].kind == ")" {
				pos++
			}
			return v
		}
		if pos < len(toks) && toks[pos].kind == "SYM" {
			v := evalSym(toks[pos].val)
			pos++
			return v
		}
		return false
	}
	return parseExpr()
}
