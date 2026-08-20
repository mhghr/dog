// Package queue defines the probe-job queue abstraction. Jobs flow
// Scheduler → Queue → Worker; the queue may be backed by Redis Streams (the
// legacy default) or NATS JetStream (the enterprise execution backbone).
package queue

import (
	"context"
	"time"

	"monitoring-platform/packages/shared/domain"
)

// Message is a queue-agnostic probe job message. Values holds the raw job
// payload plus any broker-specific metadata (e.g. original message ID).
type Message struct {
	ID     string
	Values map[string]any
}

// Stats reports backlog (not yet delivered) and pending (delivered but
// unacked) counts for the worker consumer.
type Stats struct {
	Lag     int64 `json:"lag"`
	Pending int64 `json:"pending"`
}

// JobQueue is the contract between scheduler, worker and the message broker.
// Implementations must support at-least-once delivery, durable consumers and
// dead-lettering so a crashed worker never loses a job.
type JobQueue interface {
	// EnsureGroup creates the stream/consumer group if it does not exist.
	EnsureGroup(ctx context.Context) error

	// Publish enqueues a probe job on the shared stream.
	Publish(ctx context.Context, job domain.ProbeJob) error

	// PublishToLocation enqueues a job payload on the location-scoped stream.
	PublishToLocation(ctx context.Context, locationCode string, job []byte) error

	// Consume returns up to count undelivered jobs for the consumer, blocking
	// up to block duration.
	Consume(ctx context.Context, consumerName string, count int64, block time.Duration) ([]Message, error)

	// Ack acknowledges a message as processed.
	Ack(ctx context.Context, messageID string) error

	// AutoClaim recovers messages abandoned by crashed consumers.
	AutoClaim(ctx context.Context, consumerName string, minIdle time.Duration, count int64) ([]Message, error)

	// DeliveryCount reports how many times a pending message was delivered.
	DeliveryCount(ctx context.Context, messageID string) (int64, error)

	// DeadLetter moves a poison message to the dead-letter stream and acks it.
	DeadLetter(ctx context.Context, msg Message, reason string) error

	// Stats reports queue backlog.
	Stats(ctx context.Context) (Stats, error)
}
