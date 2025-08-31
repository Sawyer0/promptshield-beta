package composite

import (
    "context"
    "strings"
    "github.com/promptshield/promptshield/internal/rules"
)

type Analyzer interface { Analyze(ctx context.Context, input string, cfg rules.Semantic) (bool, float64, error) }

type Composite struct {
    omni   Analyzer // may be nil
    custom Analyzer // may be nil
}

func New(omni Analyzer, custom Analyzer) *Composite { return &Composite{omni: omni, custom: custom} }

func (c *Composite) Analyze(ctx context.Context, input string, cfg rules.Semantic) (bool, float64, error) {
    engine := strings.ToLower(strings.TrimSpace(cfg.Engine))
    if engine == "" { engine = "omni" }

    switch engine {
    case "omni":
        if c.omni == nil { return false, 0, nil }
        return c.omni.Analyze(ctx, input, cfg)
    case "custom":
        if c.custom == nil { return false, 0, nil }
        return c.custom.Analyze(ctx, input, cfg)
    case "custom+omni":
        var customOK bool; var customConf float64; var e error
        if c.custom != nil {
            customOK, customConf, e = c.custom.Analyze(ctx, input, cfg)
            if e != nil && !cfg.FallbackOnError {
                // Respect fallback_on_error for custom engine
                return false, 0, e
            }
        }
        var omniOK bool; var omniConf float64; var e2 error
        if c.omni != nil {
            omniOK, omniConf, e2 = c.omni.Analyze(ctx, input, cfg)
            // ignore omni errors, treat as no signal
            _ = e2
        }
        mode := strings.ToLower(strings.TrimSpace(cfg.CombineMode))
        if mode == "" { mode = "or" }
        switch mode {
        case "and":
            return (customOK && omniOK), max(customConf, omniConf), nil
        case "weighted":
            wOmni := cfg.WeightOmni; if wOmni <= 0 { wOmni = 0.6 }
            wCustom := cfg.WeightCustom; if wCustom <= 0 { wCustom = 0.4 }
            score := (wOmni*omniConf + wCustom*customConf)
            thr := cfg.ConfidenceThreshold; if thr == 0 { thr = 0.7 }
            return score >= thr, score, nil
        default: // or
            return (customOK || omniOK), max(customConf, omniConf), nil
        }
    default:
        // default to omni behavior
        if c.omni == nil { return false, 0, nil }
        return c.omni.Analyze(ctx, input, cfg)
    }
}

func max(a, b float64) float64 { if a > b { return a }; return b }

