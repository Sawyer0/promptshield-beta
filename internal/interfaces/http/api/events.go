package api

import (
	"bytes"
	"encoding/json"
	"sync"
	"time"
)

// Event represents a server-sent or internal event.
type Event struct {
	Type string         `json:"type"`
	Data map[string]any `json:"data"`
	Time time.Time      `json:"time"`
}

// SSE formats the event for Server-Sent Events transport.
func (e Event) SSE() []byte {
	if e.Time.IsZero() {
		e.Time = time.Now().UTC()
	}
	payload := map[string]any{
		"type": e.Type,
		"time": e.Time.Format(time.RFC3339Nano),
		"data": e.Data,
	}
	b, _ := json.Marshal(payload)
	var buf bytes.Buffer
	buf.WriteString("event: ")
	if e.Type == "" {
		buf.WriteString("message")
	} else {
		buf.WriteString(e.Type)
	}
	buf.WriteByte('\n')
	buf.WriteString("data: ")
	buf.Write(b)
	buf.WriteString("\n\n")
	return buf.Bytes()
}

// EventHub is a minimal in-memory broadcaster for events.
type EventHub struct {
	mu          sync.RWMutex
	subscribers map[chan *Event]struct{}
}

func NewEventHub() *EventHub {
	return &EventHub{subscribers: make(map[chan *Event]struct{})}
}

// Subscribe registers a listener and returns a channel to receive events.
// The channel is buffered to avoid blocking; callers must drain it promptly.
func (h *EventHub) Subscribe(cancel <-chan struct{}) chan *Event {
	ch := make(chan *Event, 16)
	h.mu.Lock()
	h.subscribers[ch] = struct{}{}
	h.mu.Unlock()
	go func() {
		<-cancel
		h.Unsubscribe(ch)
	}()
	return ch
}

// Unsubscribe removes and closes a listener channel.
func (h *EventHub) Unsubscribe(ch chan *Event) {
	h.mu.Lock()
	if _, ok := h.subscribers[ch]; ok {
		delete(h.subscribers, ch)
		close(ch)
	}
	h.mu.Unlock()
}

// Publish sends an event to all listeners (best-effort, non-blocking).
func (h *EventHub) Publish(e Event) {
	h.mu.RLock()
	for ch := range h.subscribers {
		select {
		case ch <- &e:
		default:
			// drop if receiver is slow
		}
	}
	h.mu.RUnlock()
}
