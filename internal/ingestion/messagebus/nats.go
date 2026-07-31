package messagebus

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

// NATSConfig holds connection parameters for NATS.
type NATSConfig struct {
	URL       string
	Reconnect bool
	MaxReconn int
}

// NATSBus implements MessageBus using NATS JetStream.
type NATSBus struct {
	conn   *nats.Conn
	js     nats.JetStreamContext
	subs   []*nats.Subscription
	mu     sync.Mutex
	logger *slog.Logger
	done   chan struct{}
}

// NewNATSBus creates a new NATS JetStream message bus.
func NewNATSBus(cfg NATSConfig, logger *slog.Logger) (*NATSBus, error) {
	opts := []nats.Option{
		nats.Name("monitoring-agent-gateway"),
		nats.Timeout(10 * time.Second),
		nats.ReconnectWait(2 * time.Second),
		nats.MaxReconnects(cfg.MaxReconn),
	}

	conn, err := nats.Connect(cfg.URL, opts...)
	if err != nil {
		return nil, fmt.Errorf("connect to NATS: %w", err)
	}

	js, err := conn.JetStream()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("jetstream context: %w", err)
	}

	bus := &NATSBus{
		conn:   conn,
		js:     js,
		logger: logger,
		done:   make(chan struct{}),
	}

	if err := bus.ensureStreams(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("ensure streams: %w", err)
	}

	return bus, nil
}

// ensureStreams creates the required JetStream streams if they don't exist.
func (b *NATSBus) ensureStreams() error {
	streams := []struct {
		name     string
		subjects []string
	}{
		{
			name:     "METRICS",
			subjects: []string{"metrics.>"},
		},
		{
			name:     "METRICS_DLQ",
			subjects: []string{"metrics.dlq.>"},
		},
	}

	for _, s := range streams {
		_, err := b.js.StreamInfo(s.name)
		if err != nil {
			_, err = b.js.AddStream(&nats.StreamConfig{
				Name:      s.name,
				Subjects:  s.subjects,
				Storage:   nats.FileStorage,
				Retention: nats.InterestPolicy,
				MaxAge:    72 * time.Hour,
				MaxBytes:  10 * 1024 * 1024 * 1024,
			})
			if err != nil {
				return fmt.Errorf("create stream %s: %w", s.name, err)
			}
			b.logger.Info("created stream", "name", s.name)
		}
	}

	return nil
}

// Publish sends a message to NATS JetStream.
func (b *NATSBus) Publish(ctx context.Context, opts PublishOptions) error {
	msg := &nats.Msg{
		Subject: opts.Subject,
		Data:    opts.Data,
		Header:  nats.Header{},
	}
	for k, v := range opts.Headers {
		msg.Header.Set(k, v)
	}

	_, err := b.js.PublishMsg(msg, nats.Context(ctx))
	if err != nil {
		return fmt.Errorf("publish to %s: %w", opts.Subject, err)
	}
	return nil
}

// Subscribe registers a JetStream push consumer with the given subscription options.
func (b *NATSBus) Subscribe(ctx context.Context, opts SubscribeOptions, handler MessageHandler) error {
	subOpts := []nats.SubOpt{
		nats.Durable(opts.Durable),
		nats.ManualAck(),
		nats.MaxDeliver(10),
		nats.AckWait(60 * time.Second),
	}

	if opts.DeliverNew {
		subOpts = append(subOpts, nats.DeliverNew())
	} else {
		subOpts = append(subOpts, nats.DeliverAll())
	}

	if opts.Queue != "" {
		subOpts = append(subOpts, nats.BindStream("METRICS"))
	}

	sub, err := b.js.QueueSubscribe(opts.Subject, opts.Queue, func(msg *nats.Msg) {
		busMsg := Message{
			ID:        uuid.NewString(),
			Subject:   msg.Subject,
			Data:      msg.Data,
			Timestamp: time.Now(),
			Headers:   make(map[string]string),
		}
		for k := range msg.Header {
			busMsg.Headers[k] = msg.Header.Get(k)
		}

		if err := handler(ctx, busMsg); err != nil {
			b.logger.Error("message handler failed",
				"subject", msg.Subject,
				"error", err,
			)
			msg.Nak()
			return
		}
		msg.Ack()
	}, subOpts...)
	if err != nil {
		return fmt.Errorf("subscribe to %s: %w", opts.Subject, err)
	}

	b.mu.Lock()
	b.subs = append(b.subs, sub)
	b.mu.Unlock()

	b.logger.Info("subscribed",
		"subject", opts.Subject,
		"queue", opts.Queue,
		"durable", opts.Durable,
	)

	return nil
}

// Ack acknowledges a message (NATS handles this via msg.Ack() in Subscribe, so this is a no-op).
func (b *NATSBus) Ack(ctx context.Context, msg Message) error {
	return nil
}

// Nack negatively acknowledges a message (NATS handles this via msg.Nak() in Subscribe).
func (b *NATSBus) Nack(ctx context.Context, msg Message) error {
	return nil
}

// Close unsubscribes all consumers and closes the NATS connection.
func (b *NATSBus) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	for _, sub := range b.subs {
		sub.Unsubscribe()
	}
	b.conn.Close()
	close(b.done)
	return nil
}
