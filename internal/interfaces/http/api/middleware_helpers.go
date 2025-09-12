package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// MiddlewareHelpers provides common middleware patterns to reduce code duplication

// AdminGroup creates a router group with admin authentication
func AdminGroup(r chi.Router, opt Options, fn func(chi.Router)) {
	r.Group(func(a chi.Router) {
		a.Use(adminAuth(opt))
		fn(a)
	})
}

// RateLimitedGroup creates a router group with rate limiting
func RateLimitedGroup(r chi.Router, opt Options, fn func(chi.Router)) {
	r.Group(func(g chi.Router) {
		if opt.QuotaStore != nil {
			g.Use(tenantQuota(opt))
		}
		fn(g)
	})
}

// StandardMiddlewareChain applies the standard middleware chain to a router
func StandardMiddlewareChain(r chi.Router, opt Options) {
	applyStandardMiddleware(r, opt)
}

// ConditionalMiddleware applies middleware only if condition is true
func ConditionalMiddleware(condition bool, mw func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	if condition {
		return mw
	}
	return func(next http.Handler) http.Handler {
		return next // no-op
	}
}

// ChainMiddleware chains multiple middleware functions
func ChainMiddleware(middlewares ...func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			next = middlewares[i](next)
		}
		return next
	}
}
