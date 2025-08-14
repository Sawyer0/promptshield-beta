package scanner

// keywordRule represents a simple built-in keyword detector used for fast paths.
type keywordRule struct {
	id       string
	message  string
	severity string
	// match on lowercase line content for a simple MVP
	keyword string
}

func defaultKeywordRules() []keywordRule {
	return []keywordRule{
		{id: "pii:api_key", message: "Potential API key reference", severity: "WARNING", keyword: "api_key"},
		{id: "secrets:password", message: "Possible password reference", severity: "WARNING", keyword: "password"},
		{id: "injection:ignore-previous", message: "Prompt injection attempt phrase detected", severity: "HIGH", keyword: "ignore previous instructions"},
		{id: "injection:system-prompt", message: "Attempt to reveal system prompt", severity: "HIGH", keyword: "system prompt"},
	}
}

// SetBuiltinKeywordsEnabled enables or disables built-in keyword rules.
func (s *Scanner) SetBuiltinKeywordsEnabled(enable bool) {
	if s == nil {
		return
	}
	if enable {
		s.keywordRules = defaultKeywordRules()
	} else {
		s.keywordRules = nil
	}
}
