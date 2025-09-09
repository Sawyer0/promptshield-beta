package errors

import "errors"

// ErrFailOnThreshold indicates that the scan found violations at or above the
// configured fail-on severity threshold.
var ErrFailOnThreshold = errors.New("fail on threshold")

// Semantic provider wiring errors (opt-in L3 engine)
var (
	// ErrOpenAIAPIKeyMissing indicates OPENAI_API_KEY/PS_OPENAI_API_KEY was not provided.
	ErrOpenAIAPIKeyMissing = errors.New("openai api key missing")

	// ErrSemanticAnalyzerNotConfigured indicates L3 rules are present but no semantic analyzer was wired.
	ErrSemanticAnalyzerNotConfigured = errors.New("semantic analyzer not configured")

	// ErrStreamLimitExceeded indicates the streaming byte limit was exceeded during scanning.
	ErrStreamLimitExceeded = errors.New("stream byte limit exceeded")
)
