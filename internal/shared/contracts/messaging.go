package contracts

import (
	"context"
	"time"

	"github.com/promptshield/promptshield/internal/shared/events"
	"github.com/promptshield/promptshield/internal/shared/types"
)

// EventPublisher defines the interface for publishing domain events
type EventPublisher interface {
	// Publish publishes a single event
	Publish(ctx context.Context, event events.Event) error
	
	// PublishBatch publishes multiple events atomically
	PublishBatch(ctx context.Context, events []events.Event) error
	
	// PublishAsync publishes an event asynchronously
	PublishAsync(ctx context.Context, event events.Event) error
	
	// Close gracefully shuts down the publisher
	Close() error
}

// EventSubscriber defines the interface for subscribing to domain events
type EventSubscriber interface {
	// Subscribe subscribes to events of specific types
	Subscribe(ctx context.Context, eventTypes []string, handler EventHandler) error
	
	// SubscribeAll subscribes to all events
	SubscribeAll(ctx context.Context, handler EventHandler) error
	
	// Unsubscribe stops subscription for specific event types
	Unsubscribe(eventTypes []string) error
	
	// Close gracefully shuts down the subscriber
	Close() error
}

// EventHandler handles incoming events
type EventHandler func(ctx context.Context, event events.Event) error

// MessageQueue defines the interface for asynchronous message processing
type MessageQueue interface {
	// Enqueue adds a message to the queue
	Enqueue(ctx context.Context, message *types.QueueMessage) error
	
	// Dequeue retrieves and removes a message from the queue
	Dequeue(ctx context.Context, timeout time.Duration) (*types.QueueMessage, error)
	
	// Peek looks at the next message without removing it
	Peek(ctx context.Context) (*types.QueueMessage, error)
	
	// Ack acknowledges successful processing of a message
	Ack(ctx context.Context, messageID string) error
	
	// Nack negatively acknowledges a message (for retry)
	Nack(ctx context.Context, messageID string) error
	
	// GetQueueStats returns queue statistics
	GetQueueStats(ctx context.Context) (*types.QueueStats, error)
	
	// Purge removes all messages from the queue
	Purge(ctx context.Context) error
	
	// Close gracefully shuts down the queue
	Close() error
}

// StreamProcessor defines the interface for stream processing
type StreamProcessor interface {
	// ProcessStream processes a stream of events
	ProcessStream(ctx context.Context, stream <-chan events.Event, handler StreamHandler) error
	
	// CreateStream creates a new event stream
	CreateStream(ctx context.Context, streamName string, config *types.StreamConfig) error
	
	// DeleteStream deletes an event stream
	DeleteStream(ctx context.Context, streamName string) error
	
	// GetStreams returns list of available streams
	GetStreams(ctx context.Context) ([]string, error)
	
	// Subscribe subscribes to a stream
	Subscribe(ctx context.Context, streamName string, consumerGroup string, handler StreamHandler) error
}

// StreamHandler handles streaming events
type StreamHandler func(ctx context.Context, event events.Event) error

// NotificationService defines the interface for sending notifications
type NotificationService interface {
	// SendEmail sends an email notification
	SendEmail(ctx context.Context, notification *types.EmailNotification) error
	
	// SendSMS sends an SMS notification
	SendSMS(ctx context.Context, notification *types.SMSNotification) error
	
	// SendWebhook sends a webhook notification
	SendWebhook(ctx context.Context, notification *types.WebhookNotification) error
	
	// SendSlack sends a Slack notification
	SendSlack(ctx context.Context, notification *types.SlackNotification) error
	
	// GetDeliveryStatus gets the delivery status of a notification
	GetDeliveryStatus(ctx context.Context, notificationID string) (*types.DeliveryStatus, error)
}

// RuleUpdatePublisher defines the interface for publishing rule updates
// From internal/infrastructure/messaging/nats/publisher.go
type RuleUpdatePublisher interface {
	// PublishRuleUpdate publishes a rule pack update event
	PublishRuleUpdate(ctx context.Context, update *types.RuleUpdateEvent) error
	
	// Close gracefully shuts down the publisher
	Close() error
}

// RuleUpdateSubscriber defines the interface for subscribing to rule updates
type RuleUpdateSubscriber interface {
	// Subscribe subscribes to rule update events for a tenant
	Subscribe(ctx context.Context, tenantID string, handler RuleUpdateHandler) error
	
	// Unsubscribe stops subscription for a tenant
	Unsubscribe(tenantID string) error
	
	// Close gracefully shuts down the subscriber
	Close() error
}

// RuleUpdateHandler handles rule update events
type RuleUpdateHandler func(ctx context.Context, update *types.RuleUpdateEvent) error