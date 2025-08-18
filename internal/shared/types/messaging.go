package types

import "time"

// QueueMessage represents a message in the queue
type QueueMessage struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Payload     []byte                 `json:"payload"`
	Headers     map[string]string      `json:"headers,omitempty"`
	Priority    int                    `json:"priority"`
	Attempts    int                    `json:"attempts"`
	MaxAttempts int                    `json:"max_attempts"`
	Delay       time.Duration          `json:"delay,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	VisibleAt   time.Time              `json:"visible_at"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// QueueStats represents queue statistics
type QueueStats struct {
	TotalMessages    int64     `json:"total_messages"`
	VisibleMessages  int64     `json:"visible_messages"`
	InFlightMessages int64     `json:"in_flight_messages"`
	DelayedMessages  int64     `json:"delayed_messages"`
	LastUpdated      time.Time `json:"last_updated"`
}

// StreamConfig configures a stream
type StreamConfig struct {
	MaxLength         int           `json:"max_length"`
	RetentionTime     time.Duration `json:"retention_time"`
	Partitions        int           `json:"partitions"`
	ReplicationFactor int           `json:"replication_factor"`
}

// EmailNotification represents an email notification
type EmailNotification struct {
	To          []string                   `json:"to"`
	CC          []string                   `json:"cc,omitempty"`
	BCC         []string                   `json:"bcc,omitempty"`
	Subject     string                     `json:"subject"`
	Body        string                     `json:"body"`
	IsHTML      bool                       `json:"is_html"`
	Attachments []NotificationAttachment   `json:"attachments,omitempty"`
	Priority    NotificationPriority       `json:"priority"`
	Metadata    map[string]interface{}     `json:"metadata,omitempty"`
}

// SMSNotification represents an SMS notification
type SMSNotification struct {
	To       []string                 `json:"to"`
	Message  string                   `json:"message"`
	Priority NotificationPriority     `json:"priority"`
	Metadata map[string]interface{}   `json:"metadata,omitempty"`
}

// WebhookNotification represents a webhook notification
type WebhookNotification struct {
	URL        string                 `json:"url"`
	Method     string                 `json:"method"`
	Headers    map[string]string      `json:"headers,omitempty"`
	Payload    []byte                 `json:"payload"`
	Timeout    time.Duration          `json:"timeout"`
	RetryCount int                    `json:"retry_count"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// SlackNotification represents a Slack notification
type SlackNotification struct {
	Channel     string                 `json:"channel"`
	Message     string                 `json:"message"`
	Username    string                 `json:"username,omitempty"`
	IconEmoji   string                 `json:"icon_emoji,omitempty"`
	Attachments []SlackAttachment      `json:"attachments,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// NotificationAttachment represents a file attachment
type NotificationAttachment struct {
	FileName    string `json:"file_name"`
	ContentType string `json:"content_type"`
	Data        []byte `json:"data"`
}

// SlackAttachment represents a Slack message attachment
type SlackAttachment struct {
	Title     string       `json:"title,omitempty"`
	Text      string       `json:"text,omitempty"`
	Color     string       `json:"color,omitempty"`
	Fields    []SlackField `json:"fields,omitempty"`
	Footer    string       `json:"footer,omitempty"`
	Timestamp int64        `json:"ts,omitempty"`
}

// SlackField represents a field in a Slack attachment
type SlackField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

// NotificationPriority represents notification priority levels
type NotificationPriority string

const (
	NotificationPriorityLow      NotificationPriority = "low"
	NotificationPriorityNormal   NotificationPriority = "normal"
	NotificationPriorityHigh     NotificationPriority = "high"
	NotificationPriorityCritical NotificationPriority = "critical"
)

// DeliveryStatus represents the delivery status of a notification
type DeliveryStatus struct {
	ID          string                 `json:"id"`
	Status      DeliveryStatusType     `json:"status"`
	SentAt      *time.Time             `json:"sent_at,omitempty"`
	DeliveredAt *time.Time             `json:"delivered_at,omitempty"`
	FailedAt    *time.Time             `json:"failed_at,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Attempts    int                    `json:"attempts"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// DeliveryStatusType represents delivery status types
type DeliveryStatusType string

const (
	DeliveryStatusPending   DeliveryStatusType = "pending"
	DeliveryStatusSent      DeliveryStatusType = "sent"
	DeliveryStatusDelivered DeliveryStatusType = "delivered"
	DeliveryStatusFailed    DeliveryStatusType = "failed"
	DeliveryStatusRetrying  DeliveryStatusType = "retrying"
)

// RuleUpdateEvent represents a rule pack update event
type RuleUpdateEvent struct {
	TenantID      string    `json:"tenant_id"`
	TargetScope   string    `json:"target_scope"`
	RulepackID    string    `json:"rulepack_id"`
	Version       int       `json:"version"`
	ContentSHA256 string    `json:"content_sha256"`
	UpdatedAt     time.Time `json:"updated_at"`
}