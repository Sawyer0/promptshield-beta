package api

import (
    "net/http"

    "github.com/promptshield/promptshield/internal/application/services"
    "github.com/promptshield/promptshield/internal/repository"
)

// testRouter creates a router with mock dependencies for testing LLM Gateway behavior.
//lint:ignore U1000 used by integration-style tests as a helper
func testRouter() http.Handler {
    factory, _ := repository.NewTestRepositoryFactory(nil, nil)
    rulepackService := services.RulepackServiceFromFactory(factory, nil)
    return NewMux(Options{RulepackService: rulepackService})
}

// testRouterWithOptions creates a router with mock dependencies and custom options for testing.
//lint:ignore U1000 kept for future tests and local debugging
func testRouterWithOptions(opts Options) http.Handler {
    if opts.RulepackService == nil {
        factory, _ := repository.NewTestRepositoryFactory(nil, nil)
        opts.RulepackService = services.RulepackServiceFromFactory(factory, nil)
    }
    return NewMux(opts)
}
