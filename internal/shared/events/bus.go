package events

import (
	"context"
	"log/slog"
	"sync"
)

// Handler represents an event handler function
type Handler func(ctx context.Context, event Event) error

// EventBus provides in-process event publishing and subscription
type EventBus struct {
	mu        sync.RWMutex
	handlers  map[string][]Handler
	logger    *slog.Logger
}

// NewEventBus creates a new in-process event bus
func NewEventBus() *EventBus {
	return &EventBus{
		handlers: make(map[string][]Handler),
		logger:   slog.With("component", "eventbus"),
	}
}

// Subscribe registers a handler for a specific event type
func (bus *EventBus) Subscribe(eventType string, handler Handler) {
	bus.mu.Lock()
	defer bus.mu.Unlock()
	
	bus.handlers[eventType] = append(bus.handlers[eventType], handler)
	bus.logger.Debug("Event handler subscribed", "event_type", eventType)
}

// Publish sends an event to all registered handlers for its type
func (bus *EventBus) Publish(ctx context.Context, event Event) error {
	bus.mu.RLock()
	handlers := make([]Handler, len(bus.handlers[event.EventType()]))
	copy(handlers, bus.handlers[event.EventType()])
	bus.mu.RUnlock()
	
	if len(handlers) == 0 {
		bus.logger.Debug("No handlers for event", "event_type", event.EventType(), "event_id", event.EventID())
		return nil
	}
	
	bus.logger.Debug("Publishing event", "event_type", event.EventType(), "event_id", event.EventID(), "handlers", len(handlers))
	
	// Execute handlers concurrently but don't wait for them to complete
	// This ensures policy activation doesn't block API responses
	for _, handler := range handlers {
		go func(h Handler) {
			if err := h(ctx, event); err != nil {
				bus.logger.Error("Event handler failed", "event_type", event.EventType(), "event_id", event.EventID(), "error", err)
			}
		}(handler)
	}
	
	return nil
}

// PublishSync publishes an event and waits for all handlers to complete
func (bus *EventBus) PublishSync(ctx context.Context, event Event) error {
	bus.mu.RLock()
	handlers := make([]Handler, len(bus.handlers[event.EventType()]))
	copy(handlers, bus.handlers[event.EventType()])
	bus.mu.RUnlock()
	
	if len(handlers) == 0 {
		return nil
	}
	
	bus.logger.Debug("Publishing event synchronously", "event_type", event.EventType(), "event_id", event.EventID(), "handlers", len(handlers))
	
	// Execute handlers synchronously
	for _, handler := range handlers {
		if err := handler(ctx, event); err != nil {
			bus.logger.Error("Event handler failed", "event_type", event.EventType(), "event_id", event.EventID(), "error", err)
			return err
		}
	}
	
	return nil
}

// UnsubscribeAll removes all handlers for an event type
func (bus *EventBus) UnsubscribeAll(eventType string) {
	bus.mu.Lock()
	defer bus.mu.Unlock()
	
	delete(bus.handlers, eventType)
	bus.logger.Debug("All handlers unsubscribed", "event_type", eventType)
}

// GetSubscriberCount returns the number of subscribers for an event type
func (bus *EventBus) GetSubscriberCount(eventType string) int {
	bus.mu.RLock()
	defer bus.mu.RUnlock()
	
	return len(bus.handlers[eventType])
}

// Global event bus instance for the application
var globalEventBus *EventBus
var globalEventBusOnce sync.Once

// GlobalEventBus returns the global event bus instance
func GlobalEventBus() *EventBus {
	globalEventBusOnce.Do(func() {
		globalEventBus = NewEventBus()
	})
	return globalEventBus
}