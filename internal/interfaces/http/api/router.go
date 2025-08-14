package api

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	appscan "github.com/promptshield/promptshield/internal/application/scan"
	"github.com/promptshield/promptshield/internal/license"
	"github.com/promptshield/promptshield/internal/rules"
	"github.com/promptshield/promptshield/internal/scanner"
	"github.com/promptshield/promptshield/internal/version"
	"gopkg.in/yaml.v3"
)

func httpAuthOK(r *http.Request, want string) bool {
	if want == "" {
		return true
	}
	if v := r.Header.Get("Authorization"); v != "" {
		if len(v) >= 7 && (strings.HasPrefix(v, "Bearer ") || strings.HasPrefix(v, "bearer ")) {
			if v[7:] == want {
				return true
			}
		}
		if v == want {
			return true
		}
	}
	if v := r.Header.Get("X-PS-Token"); v != "" {
		if v == want {
			return true
		}
	}
	return false
}



// Options configures the API mux.
type Options struct {
	AdminToken         string
	AllowInsecureAdmin bool
	ConfigStore        *RuntimeConfigStore
	RulepackManager    *RulepackManager
}

type apiError struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

func writeError(w http.ResponseWriter, status int, code, msg string, details map[string]any) {
	w.Header().Set("content-type", "application/json")
	w.Header().Set("X-PS-API-Version", "1")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(apiError{Code: code, Message: msg, Details: details})
}

func versionHeader(v string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-PS-API-Version", v)
			next.ServeHTTP(w, r)
		})
	}
}

func adminAuth(opt Options) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if opt.AdminToken == "" && !opt.AllowInsecureAdmin {
				writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "admin token required", nil)
				return
			}
			tok := r.Header.Get("Authorization")
			if strings.HasPrefix(strings.ToLower(tok), "bearer ") {
				tok = tok[7:]
			}
			if tok == "" {
				tok = r.Header.Get("X-PS-Admin-Token")
			}
			if tok != opt.AdminToken {
				writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid admin token", nil)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RuntimeConfig reflects tunables for the enforcer.
type RuntimeConfig struct {
	EnforcementMode    string  `json:"enforcement_mode"`
	FailOn             string  `json:"fail_on"`
	RedactionEnabled   bool    `json:"redaction_enabled"`
	MaxStreamBytes     int64   `json:"max_stream_bytes"`
	StreamWindow       int     `json:"stream_window"`
	StreamOverlap      int     `json:"stream_overlap"`
	RPS                float64 `json:"rps"`
	RPSBurst           int     `json:"rps_burst"`
	InflightLimitBytes int64   `json:"inflight_limit_bytes"`
	InflightBackoffMs  int     `json:"inflight_backoff_ms"`
	PerRuleTimeoutMs   int     `json:"per_rule_timeout_ms"`
	RequestTimeoutMs   int     `json:"request_timeout_ms"`
	ResponseTimeoutMs  int     `json:"response_timeout_ms"`
}

type RuntimeConfigStore struct {
	mu  sync.RWMutex
	cfg RuntimeConfig
}

func NewRuntimeConfigStoreFromEnv() *RuntimeConfigStore {
	cfg := RuntimeConfig{
		EnforcementMode:  strings.ToLower(strings.TrimSpace(os.Getenv("PS_ENFORCER_MODE"))),
		FailOn:          strings.ToUpper(strings.TrimSpace(os.Getenv("PS_ENFORCER_FAIL_ON"))),
		RedactionEnabled: !(strings.ToLower(strings.TrimSpace(os.Getenv("PS_ENFORCER_REDACTION_MUTATION"))) == "0" || strings.ToLower(strings.TrimSpace(os.Getenv("PS_ENFORCER_REDACTION_MUTATION"))) == "false"),
		MaxStreamBytes:   parseInt64Default(os.Getenv("PS_ENFORCER_MAX_STREAM_BYTES"), 5_000_000),
		StreamWindow:     parseIntDefault(os.Getenv("PS_ENFORCER_STREAM_WINDOW"), 64*1024),
		StreamOverlap:    parseIntDefault(os.Getenv("PS_ENFORCER_STREAM_OVERLAP"), 4096),
		RPS:              parseFloatDefault(os.Getenv("PS_ENFORCER_RPS"), 0),
		RPSBurst:         parseIntDefault(os.Getenv("PS_ENFORCER_RPS_BURST"), 1),
		InflightLimitBytes: parseInt64Default(os.Getenv("PS_ENFORCER_INFLIGHT_LIMIT_BYTES"), 0),
		InflightBackoffMs:  parseIntDefault(os.Getenv("PS_ENFORCER_INFLIGHT_BACKOFF_MS"), 0),
		PerRuleTimeoutMs:   0,
		RequestTimeoutMs:   300,
		ResponseTimeoutMs:  300,
	}
	if cfg.EnforcementMode == "" {
		cfg.EnforcementMode = "observe"
	}
	if cfg.FailOn == "" {
		cfg.FailOn = "HIGH"
	}
	return &RuntimeConfigStore{cfg: cfg}
}

func (s *RuntimeConfigStore) Get() RuntimeConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg
}

func (s *RuntimeConfigStore) Update(p RuntimeConfig) RuntimeConfig {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := s.cfg
	if p.EnforcementMode != "" {
		c.EnforcementMode = p.EnforcementMode
	}
	if p.FailOn != "" {
		c.FailOn = p.FailOn
	}
	if p.MaxStreamBytes != 0 {
		c.MaxStreamBytes = p.MaxStreamBytes
	}
	if p.StreamWindow != 0 {
		c.StreamWindow = p.StreamWindow
	}
	if p.StreamOverlap != 0 {
		c.StreamOverlap = p.StreamOverlap
	}
	if p.RPS != 0 {
		c.RPS = p.RPS
	}
	if p.RPSBurst != 0 {
		c.RPSBurst = p.RPSBurst
	}
	if p.InflightLimitBytes != 0 {
		c.InflightLimitBytes = p.InflightLimitBytes
	}
	if p.InflightBackoffMs != 0 {
		c.InflightBackoffMs = p.InflightBackoffMs
	}
	if p.PerRuleTimeoutMs != 0 {
		c.PerRuleTimeoutMs = p.PerRuleTimeoutMs
	}
	if p.RequestTimeoutMs != 0 {
		c.RequestTimeoutMs = p.RequestTimeoutMs
	}
	if p.ResponseTimeoutMs != 0 {
		c.ResponseTimeoutMs = p.ResponseTimeoutMs
	}
	c.RedactionEnabled = p.RedactionEnabled || (!p.RedactionEnabled && s.cfg.RedactionEnabled && p != (RuntimeConfig{}))
	s.cfg = c
	return s.cfg
}

func (s *RuntimeConfigStore) Reset() RuntimeConfig {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cfg = NewRuntimeConfigStoreFromEnv().cfg
	return s.cfg
}

// Rulepack Manager

type RulepackMeta struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Version string `json:"version"`
	Source  string `json:"source"`
	Active  bool   `json:"active"`
}

type LoadedPack struct {
	Pack rules.RulePack
}

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
	// simplistic: pick first pack as active
	if len(packs) > 0 {
		p := packs[0]
		id := p.Metadata.Name
		m.packs[id] = LoadedPack{Pack: p}
		m.activeID = id
		return RulepackMeta{ID: id, Name: p.Metadata.Name, Version: p.Metadata.Version, Source: p.SourcePath, Active: true}, nil
	}
	return RulepackMeta{}, toErr([]error{errNotFound("rulepack")})
}

// Router

func NewMux(opt Options) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Timeout(10 * time.Second))
	r.Use(versionHeader("1"))

	if opt.ConfigStore == nil {
		opt.ConfigStore = NewRuntimeConfigStoreFromEnv()
	}
	if opt.RulepackManager == nil {
		opt.RulepackManager = NewRulepackManager()
	}
	// Accept bearer/ps-token for enforcement endpoints

	// Health & info
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if os.Getenv("PS_ENFORCER_RULEPACK") == "" {
			if _, err := os.Stat("/rules/basic-security.yaml"); err != nil {
				if _, err := os.Stat("rules/basic-security.yaml"); err != nil {
					http.Error(w, "not ready", http.StatusServiceUnavailable)
					return
				}
			}
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready"))
	})
	// Version
	r.Get("/version", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"version":    version.Version,
			"commit":     version.Commit,
			"build_date": version.BuildDate,
		})
	})

	// Rulepacks
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
				body, _ := io.ReadAll(r.Body)
				defer r.Body.Close()
				meta, err := opt.RulepackManager.Upload(body, activate)
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
				var req struct{ ID string `json:"id"` }
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

	// Config
	r.Route("/config", func(rr chi.Router) {
		rr.Get("/", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(opt.ConfigStore.Get())
		})
		rr.Group(func(a chi.Router) {
			a.Use(adminAuth(opt))
			a.Put("/", func(w http.ResponseWriter, r *http.Request) {
				var p RuntimeConfig
				_ = json.NewDecoder(r.Body).Decode(&p)
				cfg := opt.ConfigStore.Update(p)
				_ = json.NewEncoder(w).Encode(cfg)
			})
			a.Post("/reset", func(w http.ResponseWriter, r *http.Request) {
				cfg := opt.ConfigStore.Reset()
				_ = json.NewEncoder(w).Encode(cfg)
			})
		})
	})

	// Admin
	r.Group(func(a chi.Router) {
		a.Use(adminAuth(opt))
		a.Post("/admin/drain", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusAccepted)
		})
		a.Post("/admin/shutdown", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusAccepted)
		})
	})

	// License
	r.Get("/license", func(w http.ResponseWriter, r *http.Request) {
		l := license.Info()
		_ = json.NewEncoder(w).Encode(map[string]any{
			"org":          l.Organization,
			"tier":         l.Tier,
			"expires_at":   l.ExpiresAt,
			"licensed":     license.IsLicensed(),
		})
	})
	r.Group(func(a chi.Router) {
		a.Use(adminAuth(opt))
		a.Post("/license", func(w http.ResponseWriter, r *http.Request) {
			key := r.FormValue("key")
			if key == "" {
				var body struct{ Key string `json:"key"` }
				_ = json.NewDecoder(r.Body).Decode(&body)
				key = body.Key
			}
			if key == "" {
				writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "missing key", nil)
				return
			}
			_ = os.Setenv("PROMPTSHIELD_LICENSE_KEY", key)
			w.WriteHeader(http.StatusNoContent)
		})
	})

	// Decision endpoints
	r.Post("/check", checkHandlerVersioned())
	r.Post("/scan", scanHandler(opt))

	// Observability stubs
	r.Get("/stats", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok"})
	})
	r.Get("/events", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		_, _ = w.Write([]byte("event: ready\ndata: {\"status\":\"ok\"}\n\n"))
	})

	return r
}

func checkHandlerVersioned() http.HandlerFunc {
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

		tmp, err := os.CreateTemp("", "ps-check-*.txt")
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		defer func() { _ = os.Remove(tmp.Name()) }()

		maxBytes := int64(1 << 20)
		if v := os.Getenv("PS_ENFORCER_MAX_BODY_BYTES"); v != "" {
			if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
				maxBytes = n
			}
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
		if r.Body != nil {
			defer r.Body.Close()
			if _, err := tmp.ReadFrom(r.Body); err != nil {
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}
		}
		_ = tmp.Close()

		sc := scanner.New(0)
		rulepack := os.Getenv("PS_ENFORCER_RULEPACK")
		if rulepack == "" {
			if _, err := os.Stat("/rules/basic-security.yaml"); err == nil {
				rulepack = "/rules/basic-security.yaml"
			} else if _, err := os.Stat("rules/basic-security.yaml"); err == nil {
				rulepack = "rules/basic-security.yaml"
			}
		}
		if rulepack != "" {
			if packs, e := rules.LoadPacks(rulepack); e == nil {
				sc.LoadRulePacks(packs)
			}
		}
		svc := appscan.NewService(sc)
		res, err := svc.Scan(ctx, []string{tmp.Name()}, appscan.Options{Workers: 1, PendingWindow: 32})
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		decision := "allow"
		reason := "no_signals"
		total := 0
		anyQuarantine := false
		anyDeny := false
		firstRule := ""
		for _, rr := range res {
			total += len(rr.Violations)
			for _, v := range rr.Violations {
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
		}
		if anyDeny {
			decision = "deny"
			reason = firstNonEmpty(firstRule, "response_action")
		} else if anyQuarantine || total > 0 {
			decision = "quarantine"
			reason = firstNonEmpty(firstRule, "signals_detected")
		}
		w.Header().Set("x-ps-decision", decision)
		w.Header().Set("x-ps-reason", reason)
		w.Header().Set("content-type", "application/json")
		statusCode := http.StatusOK
		if decision != "allow" {
			statusCode = http.StatusForbidden
		}
		w.WriteHeader(statusCode)
		_ = json.NewEncoder(w).Encode(map[string]any{"decision": decision, "reason": reason, "violations": total})
	}
}

func scanHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		aggregate := true
		if a := r.URL.Query().Get("aggregate"); a != "" {
			aggregate = a != "false"
		}
		ct := r.Header.Get("content-type")
		if strings.HasPrefix(ct, "application/x-ndjson") && !aggregate {
			bw := bufio.NewWriter(w)
			defer bw.Flush()
			s := bufio.NewScanner(r.Body)
			buf := make([]byte, 0, 1024*1024)
			s.Buffer(buf, 10*1024*1024)
			for s.Scan() {
				line := s.Bytes()
				ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
				res := runScanLine(ctx, line)
				cancel()
				b, _ := json.Marshal(res)
				_, _ = bw.Write(b)
				_ = bw.WriteByte('\n')
			}
			return
		}
		// aggregate path
		body, _ := io.ReadAll(r.Body)
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		res := runScanLine(ctx, body)
		cancel()
		_ = json.NewEncoder(w).Encode(res)
	}
}

func runScanLine(ctx context.Context, data []byte) map[string]any {
	tmp, _ := os.CreateTemp("", "ps-scan-*.txt")
	_, _ = tmp.Write(data)
	_ = tmp.Close()
	defer os.Remove(tmp.Name())
	sc := scanner.New(0)
	rulepack := os.Getenv("PS_ENFORCER_RULEPACK")
	if rulepack == "" {
		if _, err := os.Stat("/rules/basic-security.yaml"); err == nil {
			rulepack = "/rules/basic-security.yaml"
		} else if _, err := os.Stat("rules/basic-security.yaml"); err == nil {
			rulepack = "rules/basic-security.yaml"
		}
	}
	if rulepack != "" {
		if packs, e := rules.LoadPacks(rulepack); e == nil {
			sc.LoadRulePacks(packs)
		}
	}
	svc := appscan.NewService(sc)
	results, err := svc.Scan(ctx, []string{tmp.Name()}, appscan.Options{Workers: 1, PendingWindow: 32})
	decision := "allow"
	reason := "no_signals"
	total := 0
	if err == nil {
		anyQuarantine := false
		anyDeny := false
		firstRule := ""
		for _, rr := range results {
			total += len(rr.Violations)
			for _, v := range rr.Violations {
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

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// Helpers

func parseIntDefault(s string, def int) int {
	if s == "" {
		return def
	}
	if n, err := strconv.Atoi(s); err == nil {
		return n
	}
	return def
}

func parseInt64Default(s string, def int64) int64 {
	if s == "" {
		return def
	}
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		return n
	}
	return def
}

func parseFloatDefault(s string, def float64) float64 {
	if s == "" {
		return def
	}
	if n, err := strconv.ParseFloat(s, 64); err == nil {
		return n
	}
	return def
}

// yamlUnmarshal is a local indirection to avoid importing yaml here if not necessary
var yamlUnmarshal = func(data []byte, v any) error {
	return yaml.Unmarshal(data, v)
}

func errNotFound(what string) error { return &notFoundErr{s: what + " not found"} }

type notFoundErr struct{ s string }

func (e *notFoundErr) Error() string { return e.s }

func toErr(errs []error) error {
	if len(errs) == 0 {
		return nil
	}
	var b strings.Builder
	for i, e := range errs {
		if i > 0 {
			b.WriteString("; ")
		}
		b.WriteString(e.Error())
	}
	return errors.New(b.String())
}