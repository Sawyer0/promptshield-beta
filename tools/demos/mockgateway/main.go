package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"gopkg.in/yaml.v3"
)

// JSON Schema for the DSL rulepack
const dslSchemaJSON = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://promptshield.local/schemas/rules.json",
  "type": "object",
  "additionalProperties": false,
  "required": ["rules"],
  "properties": {
    "rules": {
      "type": "array",
      "minItems": 1,
      "items": {
        "type": "object",
        "additionalProperties": false,
        "required": ["id", "category", "severity", "message"],
        "properties": {
          "id": {"type": "string", "minLength": 1},
          "category": {"type": "string", "minLength": 1},
          "severity": {"type": "string", "enum": ["low","medium","high","critical"]},
          "message": {"type": "string", "minLength": 1},
          "patterns": {
            "type": "array",
            "minItems": 0,
            "items": {"type": "string", "minLength": 1}
          },
          "keywords": {
            "type": "array",
            "minItems": 0,
            "items": {"type": "string", "minLength": 1}
          },
          "match": {
            "type": "object",
            "additionalProperties": false,
            "properties": {
              "type": {"type": "string", "enum": ["any","all"]},
              "caseSensitive": {"type": "boolean"},
              "wordBoundary": {"type": "boolean"}
            },
            "default": {"type": "any", "caseSensitive": false, "wordBoundary": false}
          },
          "scope": {"type": "string", "enum": ["outbound","inbound","both"], "default": "both"},
          "action": {"type": "string", "enum": ["block","flag","redact","allow"], "default": "block"},
          "redact": {
            "type": "object",
            "additionalProperties": false,
            "properties": {
              "fields": {"type": "array", "items": {"type": "string"}},
              "replacement": {"type": "string"}
            }
          },
          "when": {
            "type": "object",
            "additionalProperties": false,
            "properties": {
              "pathMatches": {"type": "array", "items": {"type": "string"}},
              "modelMatches": {"type": "array", "items": {"type": "string"}},
              "headers": {"type": "object", "additionalProperties": {"type": "string"}}
            }
          },
          "llm": {
            "type": "object",
            "additionalProperties": false,
            "properties": {
              "classifier": {"type": "string"},
              "label": {"type": "string", "enum": ["unsafe","safe"]},
              "threshold": {"type": "number", "minimum": 0, "maximum": 1}
            }
          }
        }
      }
    }
  }
}`

var ruleStorePath string

func getRuleStorePath() string {
	if ruleStorePath != "" {
		return ruleStorePath
	}
	p := os.Getenv("RULESTORE_PATH")
	if strings.TrimSpace(p) == "" {
		p = "./rules.json"
	}
	ruleStorePath = p
	return ruleStorePath
}

func persistRules(jsonBytes []byte) {
	path := getRuleStorePath()
	_ = os.MkdirAll(strings.TrimSuffix(path, "/rules.json"), 0o755)
	_ = os.WriteFile(path, jsonBytes, 0o644)
}

func loadPersistedRules() {
	path := getRuleStorePath()
	b, err := os.ReadFile(path)
	if err != nil {
		return
	}
	if err := ensureSchemaCompiled(); err != nil {
		log.Printf("schema init failed: %v", err)
		return
	}
	var js interface{}
	if err := json.Unmarshal(b, &js); err != nil {
		log.Printf("invalid persisted rules json: %v", err)
		return
	}
	if err := dslSchema.Validate(js); err != nil {
		log.Printf("persisted rules schema invalid: %v", err)
		return
	}
	var req ruleSetRequest
	if err := json.Unmarshal(b, &req); err != nil {
		log.Printf("persisted rules unmarshal: %v", err)
		return
	}
	compiled, err := compileRules(req)
	if err != nil {
		log.Printf("persisted rules compile: %v", err)
		return
	}
	mu.Lock()
	dynamicRules = compiled
	mu.Unlock()
	log.Printf("Loaded %d rules from %s", len(compiled), path)
}

// Chat payload types
type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model     string        `json:"model"`
	Messages  []chatMessage `json:"messages"`
	MaxTokens int           `json:"max_tokens,omitempty"`
	Stream    bool          `json:"stream,omitempty"`
}

// Decision payload types
type violation struct {
	ID       string `json:"id"`
	Category string `json:"category"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

type gatewayDecision struct {
	Decision   string      `json:"decision"`
	Blocked    bool        `json:"blocked"`
	Violations []violation `json:"violations,omitempty"`
	Stage      string      `json:"stage,omitempty"`
	Escalation []string    `json:"escalation,omitempty"`
	LLM        *llmEval    `json:"llm,omitempty"`
}

// State
var (
	mu            sync.Mutex
	totalRequests int
	blockedCount  int
	allowedCount  int
	rulesBlocks   int
	llmBlocks     int
	lastDec       *gatewayDecision
	lastUpdated   time.Time
	dynamicRules  []compiledRule
	dslSchema     *jsonschema.Schema
)

// LLM types
type llmEval struct {
	Label      string  `json:"label"`
	Category   string  `json:"category,omitempty"`
	Confidence float64 `json:"confidence,omitempty"`
	Raw        string  `json:"raw,omitempty"`
}

type llmRule struct {
	classifier string
	desired    string
	threshold  float64
}

// Rule structs
type ruleSpec struct {
	ID       string   `json:"id" yaml:"id"`
	Category string   `json:"category" yaml:"category"`
	Severity string   `json:"severity" yaml:"severity"`
	Message  string   `json:"message" yaml:"message"`
	Patterns []string `json:"patterns" yaml:"patterns"`
	Keywords []string `json:"keywords" yaml:"keywords"`
	Match    struct {
		Type          string `json:"type" yaml:"type"`
		CaseSensitive bool   `json:"caseSensitive" yaml:"caseSensitive"`
		WordBoundary  bool   `json:"wordBoundary" yaml:"wordBoundary"`
	} `json:"match" yaml:"match"`
	Scope  string `json:"scope" yaml:"scope"`
	Action string `json:"action" yaml:"action"`
	Redact struct {
		Fields      []string `json:"fields" yaml:"fields"`
		Replacement string   `json:"replacement" yaml:"replacement"`
	} `json:"redact" yaml:"redact"`
	When struct {
		PathMatches  []string          `json:"pathMatches" yaml:"pathMatches"`
		ModelMatches []string          `json:"modelMatches" yaml:"modelMatches"`
		Headers      map[string]string `json:"headers" yaml:"headers"`
	} `json:"when" yaml:"when"`
	LLM struct {
		Classifier string  `json:"classifier" yaml:"classifier"`
		Label      string  `json:"label" yaml:"label"`
		Threshold  float64 `json:"threshold" yaml:"threshold"`
	} `json:"llm" yaml:"llm"`
}

type ruleSetRequest struct {
	Rules []ruleSpec `json:"rules" yaml:"rules"`
}

type whenCompiled struct {
	pathRegex  []*regexp.Regexp
	modelRegex []*regexp.Regexp
	headers    map[string]string
}

type redactConfig struct {
	fields      []string
	replacement string
}

type compiledRule struct {
	id             string
	category       string
	severity       string
	message        string
	regexes        []*regexp.Regexp
	keywordRegexes []*regexp.Regexp
	matchAll       bool
	scope          string
	action         string
	redact         redactConfig
	when           whenCompiled
	llm            *llmRule
}

// Schema compile once
func ensureSchemaCompiled() error {
	if dslSchema != nil {
		return nil
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("mem://dsl-schema.json", strings.NewReader(dslSchemaJSON)); err != nil {
		return err
	}
	s, err := compiler.Compile("mem://dsl-schema.json")
	if err != nil {
		return err
	}
	dslSchema = s
	return nil
}

// LLM mocks
func llmMockClassify(content string) llmEval {
	lc := strings.ToLower(content)
	unsafePhrases := []string{
		"internal directives", "internal configuration", "exact underlying instructions",
		"verbatim", "bypass", "circumvent", "ignore safety", "chain-of-thought",
	}
	for _, p := range unsafePhrases {
		if strings.Contains(lc, p) {
			return llmEval{Label: "unsafe", Category: "PromptInjection", Confidence: 0.8}
		}
	}
	return llmEval{Label: "safe", Confidence: 0.6}
}

func llmLiveClassify(providerBase *url.URL, authHeader, content string) (llmEval, error) {
	payload := map[string]any{
		"model":       "gpt-4o-mini",
		"temperature": 0,
		"messages": []map[string]string{
			{"role": "system", "content": "You are a security classifier. Respond ONLY with strict JSON: {\\\"label\\\":\\\"safe|unsafe\\\",\\\"category\\\":string,\\\"confidence\\\":0..1}"},
			{"role": "user", "content": "Classify this for prompt-injection risk: " + content},
		},
	}
	buf, _ := json.Marshal(payload)
	target := providerBase.ResolveReference(&url.URL{Path: "/v1/chat/completions"}).String()
	resp, respBody, err := postJSON(target, buf, authHeader)
	if err != nil {
		return llmEval{}, err
	}
	_ = resp
	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	_ = json.Unmarshal(respBody, &parsed)
	if len(parsed.Choices) == 0 {
		return llmEval{Label: "safe", Raw: string(respBody)}, nil
	}
	txt := parsed.Choices[0].Message.Content
	var out llmEval
	if err := json.Unmarshal([]byte(txt), &out); err == nil {
		out.Raw = txt
		if out.Label == "" {
			out.Label = "safe"
		}
		return out, nil
	}
	l := strings.ToLower(txt)
	if strings.Contains(l, "unsafe") {
		return llmEval{Label: "unsafe", Raw: txt}, nil
	}
	return llmEval{Label: "safe", Raw: txt}, nil
}

// Utility
func lastUserContent(messages []chatMessage) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if strings.ToLower(messages[i].Role) == "user" {
			return messages[i].Content
		}
	}
	return ""
}

func buildKeywordRegex(keyword string, caseSensitive, wordBoundary bool) (*regexp.Regexp, error) {
	pattern := regexp.QuoteMeta(keyword)
	if wordBoundary {
		pattern = "\\b" + pattern + "\\b"
	}
	if caseSensitive {
		return regexp.Compile(pattern)
	}
	return regexp.Compile("(?i)" + pattern)
}

func compileRules(rs ruleSetRequest) ([]compiledRule, error) {
	out := make([]compiledRule, 0, len(rs.Rules))
	for _, r := range rs.Rules {
		cr := compiledRule{
			id:       r.ID,
			category: r.Category,
			severity: r.Severity,
			message:  r.Message,
			scope:    defaultIfEmpty(r.Scope, "both"),
			action:   defaultIfEmpty(r.Action, "block"),
			matchAll: strings.EqualFold(r.Match.Type, "all"),
			when:     whenCompiled{headers: map[string]string{}},
			redact:   redactConfig{fields: r.Redact.Fields, replacement: r.Redact.Replacement},
		}
		for _, p := range r.Patterns {
			rx, err := regexp.Compile(p)
			if err != nil {
				return nil, fmt.Errorf("compile pattern %q: %w", p, err)
			}
			cr.regexes = append(cr.regexes, rx)
		}
		for _, kw := range r.Keywords {
			rkx, err := buildKeywordRegex(kw, r.Match.CaseSensitive, r.Match.WordBoundary)
			if err != nil {
				return nil, fmt.Errorf("compile keyword %q: %w", kw, err)
			}
			cr.keywordRegexes = append(cr.keywordRegexes, rkx)
		}
		for _, pr := range r.When.PathMatches {
			rx, err := regexp.Compile(pr)
			if err != nil {
				return nil, fmt.Errorf("pathMatches %q: %w", pr, err)
			}
			cr.when.pathRegex = append(cr.when.pathRegex, rx)
		}
		for _, mr := range r.When.ModelMatches {
			rx, err := regexp.Compile(mr)
			if err != nil {
				return nil, fmt.Errorf("modelMatches %q: %w", mr, err)
			}
			cr.when.modelRegex = append(cr.when.modelRegex, rx)
		}
		for k, v := range r.When.Headers {
			cr.when.headers[strings.ToLower(k)] = v
		}
		if r.LLM.Classifier != "" {
			cr.llm = &llmRule{classifier: r.LLM.Classifier, desired: defaultIfEmpty(r.LLM.Label, "unsafe"), threshold: r.LLM.Threshold}
		}
		out = append(out, cr)
	}
	return out, nil
}

func defaultIfEmpty(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func (cr *compiledRule) appliesToRequest(r *http.Request, req chatRequest) bool {
	// when.filters
	if len(cr.when.pathRegex) > 0 {
		ok := false
		for _, rx := range cr.when.pathRegex {
			if rx.MatchString(r.URL.Path) {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}
	if len(cr.when.modelRegex) > 0 {
		ok := false
		for _, rx := range cr.when.modelRegex {
			if rx.MatchString(req.Model) {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}
	for hk, hv := range cr.when.headers {
		if r.Header.Get(hk) != hv {
			return false
		}
	}
	return true
}

func (cr *compiledRule) matchesText(text string) bool {
	// regex patterns OR keyword patterns
	regexHit := false
	for _, rx := range cr.regexes {
		if rx.MatchString(text) {
			regexHit = true
			break
		}
	}
	keywordHit := false
	if len(cr.keywordRegexes) > 0 {
		if cr.matchAll {
			ok := true
			for _, rk := range cr.keywordRegexes {
				if !rk.MatchString(text) {
					ok = false
					break
				}
			}
			keywordHit = ok
		} else {
			for _, rk := range cr.keywordRegexes {
				if rk.MatchString(text) {
					keywordHit = true
					break
				}
			}
		}
	}
	return regexHit || keywordHit
}

// Inbound redaction (very small JSONPath-like: dot-notation with [] for arrays)
func applyRedactions(body []byte, fields []string, replacement string) ([]byte, error) {
	if len(fields) == 0 {
		return body, nil
	}
	var v interface{}
	if err := json.Unmarshal(body, &v); err != nil {
		return body, err
	}
	for _, path := range fields {
		v = redactSet(v, path, replacement)
	}
	return json.Marshal(v)
}

func redactSet(node interface{}, path string, replacement string) interface{} {
	segments := strings.Split(path, ".")
	return redactWalk(node, segments, replacement)
}

func redactWalk(node interface{}, segs []string, replacement string) interface{} {
	if len(segs) == 0 {
		return node
	}
	head, tail := segs[0], segs[1:]
	sliceAll := strings.HasSuffix(head, "[]")
	key := strings.TrimSuffix(head, "[]")
	switch n := node.(type) {
	case map[string]interface{}:
		child, ok := n[key]
		if !ok {
			return node
		}
		if sliceAll {
			arr, _ := child.([]interface{})
			for i := range arr {
				arr[i] = redactWalk(arr[i], tail, replacement)
			}
			n[key] = arr
			return n
		}
		if len(tail) == 0 {
			n[key] = replacement
			return n
		}
		n[key] = redactWalk(child, tail, replacement)
		return n
	case []interface{}:
		// apply to each element
		for i := range n {
			n[i] = redactWalk(n[i], segs, replacement)
		}
		return n
	default:
		return node
	}
}

// Extract assistant text (best-effort) for inbound checks
func extractAssistantText(body []byte) string {
	var j map[string]interface{}
	if err := json.Unmarshal(body, &j); err != nil {
		return string(body)
	}
	choices, _ := j["choices"].([]interface{})
	if len(choices) == 0 {
		return string(body)
	}
	first, _ := choices[0].(map[string]interface{})
	msg, _ := first["message"].(map[string]interface{})
	if msg == nil {
		return string(body)
	}
	if c, ok := msg["content"].(string); ok {
		return c
	}
	return string(body)
}

func postJSON(target string, body []byte, authHeader string) (*http.Response, []byte, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest(http.MethodPost, target, bytes.NewReader(body))
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, err := io.ReadAll(resp.Body)
	return resp, respBody, err
}

func evaluateOutbound(r *http.Request, req chatRequest, content string) (violations []violation, action string, redact redactConfig) {
	mu.Lock()
	rules := make([]compiledRule, len(dynamicRules))
	copy(rules, dynamicRules)
	mu.Unlock()
	for _, cr := range rules {
		if cr.scope == "inbound" {
			continue
		}
		if !cr.appliesToRequest(r, req) {
			continue
		}
		if cr.matchesText(content) {
			violations = append(violations, violation{ID: cr.id, Category: cr.category, Severity: cr.severity, Message: cr.message})
			if cr.action == "redact" {
				redact = cr.redact
			}
			if cr.action == "block" {
				action = "block"
				break
			}
			if cr.action == "flag" && action == "" {
				action = "flag"
			}
		}
	}
	if action == "" {
		action = "allow"
	}
	return
}

func evaluateInbound(r *http.Request, req chatRequest, respBody []byte) (violations []violation, action string, newBody []byte) {
	text := extractAssistantText(respBody)
	mu.Lock()
	rules := make([]compiledRule, len(dynamicRules))
	copy(rules, dynamicRules)
	mu.Unlock()
	var redactFields []string
	var redactRepl string
	for _, cr := range rules {
		if cr.scope == "outbound" {
			continue
		}
		if !cr.appliesToRequest(r, req) {
			continue
		}
		if cr.matchesText(text) {
			violations = append(violations, violation{ID: cr.id, Category: cr.category, Severity: cr.severity, Message: cr.message})
			if cr.action == "redact" {
				redactFields = append(redactFields, cr.redact.fields...)
				if cr.redact.replacement != "" {
					redactRepl = cr.redact.replacement
				}
			}
			if cr.action == "block" {
				action = "block"
				break
			}
			if cr.action == "flag" && action == "" {
				action = "flag"
			}
		}
	}
	if len(redactFields) > 0 {
		if redactRepl == "" {
			redactRepl = "[REDACTED]"
		}
		if b, err := applyRedactions(respBody, redactFields, redactRepl); err == nil {
			respBody = b
		}
	}
	if action == "" {
		action = "allow"
	}
	return violations, action, respBody
}

func chatHandler(providerBase *url.URL) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		var req chatRequest
		if err := json.Unmarshal(body, &req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		content := lastUserContent(req.Messages)

		// Require rulepack or enabled LLM
		mu.Lock()
		hasRules := len(dynamicRules) > 0
		mu.Unlock()
		llmEnabled := strings.EqualFold(strings.TrimSpace(os.Getenv("LLM_EVAL_ENABLED")), "1")
		if !hasRules && !llmEnabled {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusPreconditionRequired)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{"type": "no_rules_loaded", "message": "No rulepack loaded. Upload your DSL rulepack to /v1/rules before sending requests."}})
			return
		}

		// Outbound evaluation
		outViol, outAction, _ := evaluateOutbound(r, req, content)
		if outAction == "block" && len(outViol) > 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnprocessableEntity)
			dec := &gatewayDecision{Decision: "block", Blocked: true, Violations: outViol, Stage: "rules", Escalation: []string{"rules"}}
			_ = json.NewEncoder(w).Encode(dec)
			mu.Lock()
			totalRequests++
			blockedCount++
			rulesBlocks++
			lastDec = dec
			lastUpdated = time.Now()
			mu.Unlock()
			return
		}

		// Forward to provider
		target := providerBase.ResolveReference(&url.URL{Path: "/v1/chat/completions"}).String()
		authHeader := r.Header.Get("Authorization")
		resp, respBody, err := postJSON(target, body, authHeader)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		// Inbound evaluation
		inViol, inAction, newBody := evaluateInbound(r, req, respBody)
		respBody = newBody

		// LLM evaluation (post-rules), only if enabled and not already blocked
		if llmEnabled && inAction != "block" && outAction != "block" {
			// If any rule includes LLM config, use content (user prompt) for classification
			mu.Lock()
			rules := make([]compiledRule, len(dynamicRules))
			copy(rules, dynamicRules)
			mu.Unlock()
			needLLM := false
			var llmCfg *llmRule
			for _, cr := range rules {
				if cr.llm != nil {
					needLLM = true
					llmCfg = cr.llm
					break
				}
			}
			if needLLM {
				var eval llmEval
				if strings.EqualFold(strings.TrimSpace(os.Getenv("LLM_EVAL_MODE")), "live") && authHeader != "" {
					if e, err := llmLiveClassify(providerBase, authHeader, content); err == nil {
						eval = e
					} else {
						eval = llmMockClassify(content)
					}
				} else {
					eval = llmMockClassify(content)
				}
				if strings.EqualFold(eval.Label, defaultIfEmpty(llmCfg.desired, "unsafe")) && (llmCfg.threshold == 0 || eval.Confidence >= llmCfg.threshold) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusUnprocessableEntity)
					dec := &gatewayDecision{Decision: "block", Blocked: true, Stage: "llm", Escalation: []string{"rules", "llm"}, LLM: &eval}
					_ = json.NewEncoder(w).Encode(dec)
					mu.Lock()
					totalRequests++
					blockedCount++
					llmBlocks++
					lastDec = dec
					lastUpdated = time.Now()
					mu.Unlock()
					return
				}
			}
		}

		// Decide final outcome based on inbound action (block/flag/redact/allow)
		if inAction == "block" && len(inViol) > 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnprocessableEntity)
			dec := &gatewayDecision{Decision: "block", Blocked: true, Violations: inViol, Stage: "rules", Escalation: []string{"rules"}}
			_ = json.NewEncoder(w).Encode(dec)
			mu.Lock()
			totalRequests++
			blockedCount++
			rulesBlocks++
			lastDec = dec
			lastUpdated = time.Now()
			mu.Unlock()
			return
		}

		// Return (possibly redacted) provider response; record metrics and last decision
		for k, vv := range resp.Header {
			for _, v := range vv {
				w.Header().Add(k, v)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		_, _ = w.Write(respBody)
		mu.Lock()
		totalRequests++
		if inAction == "flag" || outAction == "flag" {
			lastDec = &gatewayDecision{Decision: "allow", Blocked: false, Stage: "allow"}
		} else {
			lastDec = &gatewayDecision{Decision: "allow", Blocked: false, Stage: "allow"}
		}
		allowedCount++
		lastUpdated = time.Now()
		mu.Unlock()
	}
}

func health(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }

func metricsHandler(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()
	payload := map[string]any{
		"requests":      totalRequests,
		"blocked":       blockedCount,
		"allowed":       allowedCount,
		"blocksByStage": map[string]int{"rules": rulesBlocks, "llm": llmBlocks},
		"updatedAt":     lastUpdated.Format(time.RFC3339),
	}
	if lastDec != nil {
		payload["lastDecision"] = lastDec
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(payload)
}

// YAML helpers
func toJSONFriendly(i interface{}) interface{} {
	switch x := i.(type) {
	case map[string]interface{}:
		m := make(map[string]interface{}, len(x))
		for k, v := range x {
			m[k] = toJSONFriendly(v)
		}
		return m
	case map[interface{}]interface{}:
		m := make(map[string]interface{}, len(x))
		for k, v := range x {
			ks := ""
			switch kt := k.(type) {
			case string:
				ks = kt
			default:
				ks = fmt.Sprint(kt)
			}
			m[ks] = toJSONFriendly(v)
		}
		return m
	case []interface{}:
		a := make([]interface{}, len(x))
		for i := range x {
			a[i] = toJSONFriendly(x[i])
		}
		return a
	default:
		return x
	}
}

func rulesHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		mu.Lock()
		defer mu.Unlock()
		payload := map[string]any{"count": len(dynamicRules)}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(payload)
	case http.MethodPost:
		if err := ensureSchemaCompiled(); err != nil {
			http.Error(w, "schema init: "+err.Error(), http.StatusInternalServerError)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		ct := r.Header.Get("Content-Type")
		var jsonBytes []byte
		if strings.Contains(ct, "yaml") || strings.Contains(ct, "yml") {
			var y interface{}
			if err := yaml.Unmarshal(body, &y); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			canon := toJSONFriendly(y)
			jsonBytes, err = json.Marshal(canon)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		} else {
			jsonBytes = body
		}
		var js interface{}
		if err := json.Unmarshal(jsonBytes, &js); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{"type": "invalid_json", "message": err.Error()}})
			return
		}
		if err := dslSchema.Validate(js); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{"type": "schema_validation_failed", "message": err.Error()}})
			return
		}
		var req ruleSetRequest
		if err := json.Unmarshal(jsonBytes, &req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		compiled, err := compileRules(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		mu.Lock()
		dynamicRules = compiled
		mu.Unlock()
		persistRules(jsonBytes)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok", "rules": len(compiled)})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func main() {
	port := os.Getenv("MOCK_GATEWAY_PORT")
	if port == "" {
		port = "8080"
	}
	provider := os.Getenv("PROVIDER_URL")
	if provider == "" {
		provider = "http://127.0.0.1:18080"
	}
	providerURL, err := url.Parse(provider)
	if err != nil {
		log.Fatalf("invalid provider url: %v", err)
	}

	loadPersistedRules()

	http.HandleFunc("/v1/chat/completions", chatHandler(providerURL))
	http.HandleFunc("/v1/metrics", metricsHandler)
	http.HandleFunc("/v1/rules", rulesHandler)
	http.HandleFunc("/health", health)

	addr := ":" + port
	log.Printf("Mock gateway listening on %s (provider %s)", addr, providerURL)
	log.Fatal(http.ListenAndServe(addr, nil))
}
