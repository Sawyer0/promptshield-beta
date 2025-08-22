package api

import (
	"net/http"

	"github.com/promptshield/promptshield/internal/application/services"
	"github.com/promptshield/promptshield/internal/testutil/mocks"
)

// testRouter creates a router with mock dependencies for testing LLM Gateway behavior.
//
//lint:ignore U1000 used only in tests
func testRouter() http.Handler {
	mockRepo := &mocks.MockRulepackRepository{}
	rulepackService := services.RulepackServiceCstor(mockRepo, nil)

	return NewMux(Options{
		RulepackService: rulepackService,
	})
}

// testRouterWithOptions creates a router with mock dependencies and custom options for testing.
//
//lint:ignore U1000 used only in tests
func testRouterWithOptions(opts Options) http.Handler {
	// Ensure required dependencies are set
	if opts.RulepackService == nil {
		mockRepo := &mocks.MockRulepackRepository{}
		opts.RulepackService = services.RulepackServiceCstor(mockRepo, nil)
	}

	return NewMux(opts)
}
