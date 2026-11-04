package errors

import "errors"

// ErrFailOnThreshold indicates that the scan found violations at or above the
// configured fail-on severity threshold.
var ErrFailOnThreshold = errors.New("fail on threshold")

// Semantic provider wiring errors (opt-in L3 engine)
var (
	// ErrSemanticProviderNotSet is returned when PS_SEMANTIC_PROVIDER is empty while
	// semantic analysis is enabled and L3 rules are present.
	ErrSemanticProviderNotSet = errors.New("semantic provider not set")

	// ErrUnsupportedProvider is returned when PS_SEMANTIC_PROVIDER has an unknown value.
	ErrUnsupportedProvider = errors.New("unsupported semantic provider")

	// ErrOpenAIAPIKeyMissing indicates OPENAI_API_KEY/PS_OPENAI_API_KEY was not provided.
	ErrOpenAIAPIKeyMissing = errors.New("openai api key missing")

	// ErrAnthropicAPIKeyMissing indicates ANTHROPIC_API_KEY/PS_ANTHROPIC_API_KEY was not provided.
	ErrAnthropicAPIKeyMissing = errors.New("anthropic api key missing")

	// ErrSemanticAnalyzerNotConfigured indicates L3 rules are present but no semantic analyzer was wired.
	ErrSemanticAnalyzerNotConfigured = errors.New("semantic analyzer not configured")

	// ErrStreamLimitExceeded indicates the streaming byte limit was exceeded during scanning.
	ErrStreamLimitExceeded = errors.New("stream byte limit exceeded")
)
