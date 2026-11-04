package api

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"
)

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
		EnforcementMode:    strings.ToLower(strings.TrimSpace(os.Getenv("PS_ENFORCER_MODE"))),
		FailOn:             strings.ToUpper(strings.TrimSpace(os.Getenv("PS_ENFORCER_FAIL_ON"))),
		RedactionEnabled:   !(strings.ToLower(strings.TrimSpace(os.Getenv("PS_ENFORCER_REDACTION_MUTATION"))) == "0" || strings.ToLower(strings.TrimSpace(os.Getenv("PS_ENFORCER_REDACTION_MUTATION"))) == "false"),
		MaxStreamBytes:     parseInt64Default(os.Getenv("PS_ENFORCER_MAX_STREAM_BYTES"), 5_000_000),
		StreamWindow:       parseIntDefault(os.Getenv("PS_ENFORCER_STREAM_WINDOW"), 64*1024),
		StreamOverlap:      parseIntDefault(os.Getenv("PS_ENFORCER_STREAM_OVERLAP"), 4096),
		RPS:                parseFloatDefault(os.Getenv("PS_ENFORCER_RPS"), 0),
		RPSBurst:           parseIntDefault(os.Getenv("PS_ENFORCER_RPS_BURST"), 1),
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

// Config routes
func mountConfig(r chi.Router, opt Options) {
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
}
