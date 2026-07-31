package messagebus

import (
	"context"
	"time"
)

// Message represents a single message on the bus.
type Message struct {
	ID        string
	Subject   string
	Data      []byte
	Timestamp time.Time
	Headers   map[string]string
}

// PublishOptions describes a message to publish.
type PublishOptions struct {
	Subject string
	Data    []byte
	Headers map[string]string
}

// SubscribeOptions describes a subscription.
type SubscribeOptions struct {
	Subject    string
	Queue      string
	Durable    string
	DeliverNew bool
	// Stream is the JetStream stream to bind the consumer to. Empty means
	// "METRICS", the default for metric processing subscriptions.
	Stream string
}

// MessageHandler processes a received message.
type MessageHandler func(ctx context.Context, msg Message) error

// MessageBus is an abstraction over a pub/sub message broker (NATS, Kafka, etc.).
// Implementations must support:
// - At-least-once delivery semantics
// - Durable subscriptions
// - Queue groups for load-balanced consumers
type MessageBus interface {
	// Publish sends a message to the given subject.
	Publish(ctx context.Context, opts PublishOptions) error

	// Subscribe registers a handler on the given subject.
	// The handler receives messages delivered at-least-once.
	Subscribe(ctx context.Context, opts SubscribeOptions, handler MessageHandler) error

	// Ack acknowledges a message as successfully processed.
	Ack(ctx context.Context, msg Message) error

	// Nack negatively acknowledges a message (triggers redelivery).
	Nack(ctx context.Context, msg Message) error

	// Close terminates the connection and cleans up resources.
	Close() error
}
