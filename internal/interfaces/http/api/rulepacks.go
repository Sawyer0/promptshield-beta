package api

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/promptshield/promptshield/internal/rules"
)

type RulepackMeta struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Version string `json:"version"`
	Source  string `json:"source"`
	Active  bool   `json:"active"`
}

type LoadedPack struct{ Pack rules.RulePack }

type RulepackManager struct {
	mu       sync.RWMutex
	activeID string
	packs    map[string]LoadedPack
}

func NewRulepackManager() *RulepackManager {
	return &RulepackManager{packs: make(map[string]LoadedPack)}
}

func (m *RulepackManager) List() []RulepackMeta {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []RulepackMeta
	for id, lp := range m.packs {
		out = append(out, RulepackMeta{ID: id, Name: lp.Pack.Metadata.Name, Version: lp.Pack.Metadata.Version, Source: lp.Pack.SourcePath, Active: id == m.activeID})
	}
	return out
}

func (m *RulepackManager) Active() RulepackMeta {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.activeID == "" {
		return RulepackMeta{}
	}
	lp, ok := m.packs[m.activeID]
	if !ok {
		return RulepackMeta{}
	}
	return RulepackMeta{ID: m.activeID, Name: lp.Pack.Metadata.Name, Version: lp.Pack.Metadata.Version, Source: lp.Pack.SourcePath, Active: true}
}

func (m *RulepackManager) Validate(data []byte) (bool, []string, []string) {
	var p rules.RulePack
	if err := yamlUnmarshal(data, &p); err != nil {
		return false, nil, []string{err.Error()}
	}
	errs := rules.ValidatePack(p)
	if len(errs) > 0 {
		var es []string
		for _, e := range errs {
			es = append(es, e.Error())
		}
		return false, nil, es
	}
	return true, nil, nil
}

func (m *RulepackManager) Upload(data []byte, activate bool) (RulepackMeta, error) {
	var p rules.RulePack
	if err := yamlUnmarshal(data, &p); err != nil {
		return RulepackMeta{}, err
	}
	errs := rules.ValidatePack(p)
	if len(errs) > 0 {
		return RulepackMeta{}, toErr(errs)
	}
	id := p.Metadata.Name
	if id == "" {
		id = strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.packs[id] = LoadedPack{Pack: p}
	if activate {
		m.activeID = id
	}
	return RulepackMeta{ID: id, Name: p.Metadata.Name, Version: p.Metadata.Version, Source: p.SourcePath, Active: activate}, nil
}

func (m *RulepackManager) SetActive(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.packs[id]; !ok {
		return toErr([]error{errNotFound("rulepack")})
	}
	m.activeID = id
	return nil
}

func (m *RulepackManager) ReloadFrom(path string) (RulepackMeta, error) {
	packs, err := rules.LoadPacks(path)
	if err != nil {
		return RulepackMeta{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(packs) > 0 {
		p := packs[0]
		id := p.Metadata.Name
		m.packs[id] = LoadedPack{Pack: p}
		m.activeID = id
		return RulepackMeta{ID: id, Name: p.Metadata.Name, Version: p.Metadata.Version, Source: p.SourcePath, Active: true}, nil
	}
	return RulepackMeta{}, toErr([]error{errNotFound("rulepack")})
}

func mountRulepacks(r chi.Router, opt Options) {
	r.Route("/rulepacks", func(rr chi.Router) {
		rr.Get("/", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(opt.RulepackManager.List())
		})
		rr.Get("/active", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(opt.RulepackManager.Active())
		})
		rr.Group(func(a chi.Router) {
			a.Use(adminAuth(opt))
			a.Post("/validate", func(w http.ResponseWriter, r *http.Request) {
				body, _ := io.ReadAll(r.Body)
				defer r.Body.Close()
				valid, warnings, errors := opt.RulepackManager.Validate(body)
				_ = json.NewEncoder(w).Encode(map[string]any{"valid": valid, "warnings": warnings, "errors": errors})
			})
			a.Post("/", func(w http.ResponseWriter, r *http.Request) {
				activate := r.URL.Query().Get("activate") == "true"
				ct := r.Header.Get("content-type")
				var data []byte
				// Support multipart/form-data with file field
				if strings.HasPrefix(ct, "multipart/form-data") {
					if err := r.ParseMultipartForm(10 << 20); err != nil { // 10MB limit
						writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "invalid multipart form", map[string]any{"error": err.Error()})
						return
					}
					f, _, err := r.FormFile("file")
					if err != nil {
						writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "missing file", nil)
						return
					}
					defer f.Close()
					data, _ = io.ReadAll(f)
				} else {
					// raw body (application/x-yaml)
					body, _ := io.ReadAll(r.Body)
					defer r.Body.Close()
					data = body
				}
				meta, err := opt.RulepackManager.Upload(data, activate)
				if err != nil {
					writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", err.Error(), nil)
					return
				}
				w.WriteHeader(http.StatusCreated)
				_ = json.NewEncoder(w).Encode(meta)
			})
			a.Post("/reload", func(w http.ResponseWriter, r *http.Request) {
				path := r.URL.Query().Get("path")
				if path == "" {
					path = os.Getenv("PS_ENFORCER_RULEPACK")
				}
				if path == "" {
					writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "no path provided", nil)
					return
				}
				meta, err := opt.RulepackManager.ReloadFrom(path)
				if err != nil {
					writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", err.Error(), nil)
					return
				}
				_ = json.NewEncoder(w).Encode(meta)
			})
			a.Put("/active", func(w http.ResponseWriter, r *http.Request) {
				var req struct {
					ID string `json:"id"`
				}
				_ = json.NewDecoder(r.Body).Decode(&req)
				if req.ID == "" {
					writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "missing id", nil)
					return
				}
				if err := opt.RulepackManager.SetActive(req.ID); err != nil {
					writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", err.Error(), nil)
					return
				}
				_ = json.NewEncoder(w).Encode(opt.RulepackManager.Active())
			})
			a.Delete("/{id}", func(w http.ResponseWriter, r *http.Request) {
				id := chi.URLParam(r, "id")
				opt.RulepackManager.mu.Lock()
				delete(opt.RulepackManager.packs, id)
				if opt.RulepackManager.activeID == id {
					opt.RulepackManager.activeID = ""
				}
				opt.RulepackManager.mu.Unlock()
				w.WriteHeader(http.StatusNoContent)
			})
		})
	})
}
