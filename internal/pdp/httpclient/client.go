package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/promptshield/promptshield/internal/pdp"
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
		http: &http.Client{Timeout: to},
	}
}

func (c *Client) Evaluate(ctx context.Context, req pdp.Request) (pdp.Response, error) {
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
		return pdp.Response{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 500 {
		return pdp.Response{}, fmt.Errorf("pdp upstream %d", resp.StatusCode)
	}
	var wire struct {
		Decision    string            `json:"decision"`
		Obligations []pdp.Obligation  `json:"obligations"`
		Reason      string            `json:"reason"`
		Risk        float64           `json:"risk"`
		TTLMs       int64             `json:"ttlMs"`
		Cacheable   bool              `json:"cacheable"`
		Provider    string            `json:"provider"`
		Raw         json.RawMessage   `json:"raw"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&wire); err != nil {
		return pdp.Response{}, err
	}
	d := pdp.Decision(wire.Decision)
	switch d {
	case pdp.Permit, pdp.Deny, pdp.Indeterminate, pdp.NotApplicable:
	default:
		return pdp.Response{}, fmt.Errorf("invalid decision: %s", wire.Decision)
	}
	return pdp.Response{
		Decision:    d,
		Obligations: wire.Obligations,
		Reason:      wire.Reason,
		Risk:        wire.Risk,
		Cacheable:   wire.Cacheable,
		Provider:    wire.Provider,
		Raw:         wire.Raw,
		TTL:         time.Duration(wire.TTLMs) * time.Millisecond,
	}, nil
}
