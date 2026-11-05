package api

import (
	"net/http"
	"strings"
)

// eventStreamHandler serves GET /events as Server-Sent Events from opt.Events.
func eventStreamHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var flusher http.Flusher
		if f, ok := w.(http.Flusher); ok {
			flusher = f
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		ch := opt.Events.Subscribe(r.Context().Done())
		defer opt.Events.Unsubscribe(ch)
		// Optional type filtering
		var filters map[string]struct{}
		if v := strings.TrimSpace(r.URL.Query().Get("types")); v != "" {
			filters = make(map[string]struct{})
			for _, t := range strings.Split(v, ",") {
				if t = strings.TrimSpace(t); t != "" {
					filters[t] = struct{}{}
				}
			}
		}
		// initial
		_, _ = w.Write([]byte("event: ready\ndata: {\"status\":\"ok\"}\n\n"))
		if flusher != nil {
			flusher.Flush()
		}
		notify := r.Context().Done()
		for {
			select {
			case <-notify:
				return
			case ev := <-ch:
				if ev == nil {
					return
				}
				if len(filters) > 0 {
					if _, ok := filters[ev.Type]; !ok {
						continue
					}
				}
				_, _ = w.Write(ev.SSE())
				if flusher != nil {
					flusher.Flush()
				}
			}
		}
	}
}
