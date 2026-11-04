package rulesets

import (
	"time"

	"github.com/promptshield/promptshield/internal/rules"
)

// Snapshot is a display-safe view of the active packs. It reports how many
// keywords and patterns each rule has, never the match terms themselves.
type Snapshot struct {
	Source     string    `json:"source"`
	LoadedAt   time.Time `json:"loaded_at"`
	Generation int       `json:"generation"`
	RuleCount  int       `json:"rule_count"`
	Packs      []Pack    `json:"packs"`
}

// Pack summarises one RulePack.
type Pack struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Rules       []Rule `json:"rules"`
}

// Rule summarises one rule.
type Rule struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Level    int    `json:"level"`
	Severity string `json:"severity"`
	Category string `json:"category"`
	Action   string `json:"action"`
	Enabled  bool   `json:"enabled"`
	Keywords int    `json:"keywords"`
	Patterns int    `json:"patterns"`
}

func summarizePack(p rules.RulePack) Pack {
	pack := Pack{
		Name:        p.Metadata.Name,
		Version:     p.Metadata.Version,
		Description: p.Metadata.Description,
		Rules:       make([]Rule, 0, len(p.Rules)),
	}
	for _, r := range p.Rules {
		action := ""
		if r.Response != nil {
			action = r.Response.Action
		}
		pack.Rules = append(pack.Rules, Rule{
			ID:       r.ID,
			Name:     r.Name,
			Level:    r.Level,
			Severity: r.Severity,
			Category: r.Category,
			Action:   action,
			Enabled:  r.Enabled == nil || *r.Enabled,
			Keywords: len(r.Keywords),
			Patterns: len(r.Patterns),
		})
	}
	return pack
}
