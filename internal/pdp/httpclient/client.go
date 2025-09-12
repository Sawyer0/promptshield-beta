package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/promptshield/promptshield/internal/observability/metrics"
	"github.com/promptshield/promptshield/internal/pdp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type Config struct {
	Endpoint string
	APIKey   string
	Timeout  time.Duration
}

type Client struct {
	cfg  Config
	http *http.Client
}

func New(cfg Config) *Client {
	to := cfg.Timeout
	if to <= 0 {
		to = 2 * time.Second
	}
	return &Client{
		cfg:  cfg,
		http: &http.Client{Timeout: to, Transport: otelhttp.NewTransport(http.DefaultTransport)},
	}
}

func (c *Client) Evaluate(ctx context.Context, req pdp.Request) (pdp.Response, error) {
	start := time.Now()
	if c.cfg.Endpoint == "" {
		return pdp.Response{}, errors.New("pdp endpoint not configured")
	}
	b, _ := json.Marshal(req)
	hreq, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.Endpoint, bytes.NewReader(b))
	hreq.Header.Set("Content-Type", "application/json")
	if c.cfg.APIKey != "" {
		hreq.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	}
resp, err := c.http.Do(hreq)
	if err != nil {
		if metrics.Enabled() { metrics.CacheOperations.WithLabelValues("error", "pdp").Inc() }
		return pdp.Response{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 500 {
		if metrics.Enabled() { metrics.CacheOperations.WithLabelValues("error", "pdp").Inc() }
		return pdp.Response{}, fmt.Errorf("pdp upstream %d", resp.StatusCode)
	}
	// Support both top-level and OPA-style nested {"result": {...}}
	var wireTop struct {
		Decision    string            `json:"decision"`
		Obligations []pdp.Obligation  `json:"obligations"`
		Reason      string            `json:"reason"`
		Risk        float64           `json:"risk"`
		TTLMs       int64             `json:"ttlMs"`
		Cacheable   bool              `json:"cacheable"`
		Provider    string            `json:"provider"`
		Raw         json.RawMessage   `json:"raw"`
	}
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&wireTop); err != nil {
		return pdp.Response{}, err
	}
	w := &wireTop
	d := pdp.Decision(w.Decision)
	switch d {
	case pdp.Permit, pdp.Deny, pdp.Indeterminate, pdp.NotApplicable:
	default:
		return pdp.Response{}, fmt.Errorf("invalid decision: %s", w.Decision)
	}
	respObj := pdp.Response{
		Decision:    d,
		Obligations: w.Obligations,
		Reason:      w.Reason,
		Risk:        w.Risk,
		Cacheable:   w.Cacheable,
		Provider:    w.Provider,
		Raw:         w.Raw,
		TTL:         time.Duration(w.TTLMs) * time.Millisecond,
	}
	if metrics.Enabled() { metrics.RecordDuration(metrics.TimeToFirstDecision, time.Since(start)) }
	return respObj, nil
}
