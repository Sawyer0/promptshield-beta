//go:build opa_inprocess

package pdp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/open-policy-agent/opa/rego"
)

type inprocessClient struct{
	query rego.PreparedEvalQuery
	timeout time.Duration
}

func NewInprocessClient(cfg InprocessConfig) (Client, error) {
	if cfg.EntryPoint == "" {
		return nil, fmt.Errorf("entrypoint required for inprocess mode")
	}
	paths := []string{}
	if cfg.BundlePath != "" { paths = append(paths, cfg.BundlePath) }
	if cfg.PolicyPath != "" { paths = append(paths, cfg.PolicyPath) }
	if cfg.DataPath != "" { paths = append(paths, cfg.DataPath) }
	if len(paths) == 0 {
		return nil, fmt.Errorf("no policy/data paths provided")
	}
	r := rego.New(
		rego.Query(fmt.Sprintf("data.%s", cfg.EntryPoint)),
		rego.Load(paths, nil),
	)
	pq, err := r.PrepareForEval(context.Background())
	if err != nil {
		return nil, err
	}
	to := cfg.Timeout
	if to <= 0 { to = 2 * time.Second }
	return &inprocessClient{query: pq, timeout: to}, nil
}

func (c *inprocessClient) Evaluate(ctx context.Context, req Request) (Response, error) {
	in := map[string]any{
		"subject": req.Subject,
		"action": req.Action,
		"resource": req.Resource,
		"environment": req.Environment,
	}
	ctx2, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	rs, err := c.query.Eval(ctx2, rego.EvalInput(in))
	if err != nil { return Response{}, err }
	if len(rs) == 0 || len(rs[0].Expressions) == 0 {
		return Response{Decision: NotApplicable, Cacheable: true, TTL: 1500 * time.Millisecond, Provider: "opa"}, nil
	}
	var out map[string]any
	b, _ := json.Marshal(rs[0].Expressions[0].Value)
	_ = json.Unmarshal(b, &out)
	resp := Response{Decision: NotApplicable, Provider: "opa", Cacheable: true}
	if s, ok := out["decision"].(string); ok { resp.Decision = Decision(s) }
	if s, ok := out["reason"].(string); ok { resp.Reason = s }
	if f, ok := out["risk"].(float64); ok { resp.Risk = f }
	if n, ok := out["ttlMs"].(float64); ok && n > 0 { resp.TTL = time.Duration(int64(n)) * time.Millisecond }
	if cb, ok := out["cacheable"].(bool); ok { resp.Cacheable = cb }
	if arr, ok := out["obligations"].([]any); ok {
		for _, it := range arr {
			if m, ok := it.(map[string]any); ok {
				ob := Obligation{}
				if t, ok := m["type"].(string); ok { ob.Type = t }
				if k, ok := m["key"].(string); ok { ob.Key = k }
				ob.Value = m["value"]
				resp.Obligations = append(resp.Obligations, ob)
			}
		}
	}
	return resp, nil
}
