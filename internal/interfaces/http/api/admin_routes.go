package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

// mountAdminLifecycle registers the drain and shutdown hooks.
func mountAdminLifecycle(r chi.Router, opt Options) {
	// Admin endpoints (simple token auth)
	r.Group(func(a chi.Router) {
		a.Use(adminAuth(opt))
		a.Post("/admin/drain", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusAccepted)
			if opt.OnDrain != nil {
				go func() { _ = opt.OnDrain(r.Context()) }()
			}
		})
		a.Post("/admin/shutdown", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusAccepted)
			if opt.OnShutdown != nil {
				// optional delay in seconds
				delay := 0 * time.Second
				if v := r.URL.Query().Get("delay"); v != "" {
					if n, err := strconv.Atoi(v); err == nil && n >= 0 {
						delay = time.Duration(n) * time.Second
					}
				}
				go func() { _ = opt.OnShutdown(r.Context(), delay) }()
			}
		})
	})
}
