package messagebus

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
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

type streamDef struct {
	name      string
	subjects  []string
	retention nats.RetentionPolicy
	maxAge    time.Duration
	maxBytes  int64
}

// ensureStreams creates the required JetStream streams if they don't exist.
func (b *NATSBus) ensureStreams() error {
	streams := []streamDef{
		{
			name:      "METRICS",
			subjects:  []string{"metrics.>"},
			retention: nats.InterestPolicy,
			maxAge:    72 * time.Hour,
			maxBytes:  10 * 1024 * 1024 * 1024,
		},
		{
			name:      "METRICS_DLQ",
			subjects:  []string{"metrics.dlq.>"},
			retention: nats.InterestPolicy,
			maxAge:    72 * time.Hour,
			maxBytes:  10 * 1024 * 1024 * 1024,
		},
		{
			name:      "TELEMETRY_EVENTS",
			subjects:  []string{"telemetry.probe.>", "telemetry.metrics.>"},
			retention: nats.LimitsPolicy,
			maxAge:    24 * time.Hour,
			maxBytes:  50 * 1024 * 1024 * 1024,
		},
		{
			name:      "TELEMETRY_DLQ",
			subjects:  []string{"telemetry.dlq.>"},
			retention: nats.LimitsPolicy,
			maxAge:    24 * time.Hour,
			maxBytes:  50 * 1024 * 1024 * 1024,
		},
		{
			// Enterprise execution backbone: scheduler → worker probe jobs.
			name:      "PROBE_JOBS",
			subjects:  []string{"probe.jobs.>"},
			retention: nats.WorkQueuePolicy,
			maxAge:    24 * time.Hour,
			maxBytes:  20 * 1024 * 1024 * 1024,
		},
		{
			name:      "PROBE_JOBS_DLQ",
			subjects:  []string{"probe.jobs.dlq.>"},
			retention: nats.LimitsPolicy,
			maxAge:    24 * time.Hour,
			maxBytes:  20 * 1024 * 1024 * 1024,
		},
	}

	for _, s := range streams {
		_, err := b.js.StreamInfo(s.name)
		if err != nil {
			_, err = b.js.AddStream(&nats.StreamConfig{
				Name:      s.name,
				Subjects:  s.subjects,
				Storage:   nats.FileStorage,
				Retention: s.retention,
				MaxAge:    s.maxAge,
				MaxBytes:  s.maxBytes,
				Replicas:  getReplicas(),
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

// maxDeliverAttempts is the maximum number of deliveries before a message is
// routed to the dead-letter stream. It must match nats.MaxDeliver below.
const maxDeliverAttempts = 10

// Subscribe registers a JetStream push consumer with the given subscription options.
func (b *NATSBus) Subscribe(ctx context.Context, opts SubscribeOptions, handler MessageHandler) error {
	subOpts := []nats.SubOpt{
		nats.Durable(opts.Durable),
		nats.ManualAck(),
		nats.MaxDeliver(maxDeliverAttempts),
		nats.AckWait(60 * time.Second),
	}

	if opts.DeliverNew {
		subOpts = append(subOpts, nats.DeliverNew())
	} else {
		subOpts = append(subOpts, nats.DeliverAll())
	}

	stream := opts.Stream
	if stream == "" {
		stream = "METRICS"
	}

	var sub *nats.Subscription
	var err error

	if opts.Queue != "" {
		subOpts = append(subOpts, nats.BindStream(stream))
		sub, err = b.js.QueueSubscribe(opts.Subject, opts.Queue, b.makeHandler(ctx, handler), subOpts...)
	} else {
		sub, err = b.js.Subscribe(opts.Subject, b.makeHandler(ctx, handler), subOpts...)
	}

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

// makeHandler wraps a MessageHandler into a NATS message handler.
func (b *NATSBus) makeHandler(ctx context.Context, handler MessageHandler) nats.MsgHandler {
	return func(msg *nats.Msg) {
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

			if isExhausted(msg) {
				b.routeToDLQ(msg)
				return
			}
			msg.Nak()
			return
		}
		msg.Ack()
	}
}

// isExhausted reports whether a message has hit the maximum number of
// redeliveries and should be dead-lettered instead of redelivered again.
func isExhausted(msg *nats.Msg) bool {
	md, err := msg.Metadata()
	if err != nil {
		return false
	}
	return md.NumDelivered >= maxDeliverAttempts
}

// routeToDLQ republishes an exhausted message to the dead-letter stream and
// terminates it so JetStream stops redelivering it. The raw payload and
// headers are preserved for later analysis/repair.
func (b *NATSBus) routeToDLQ(msg *nats.Msg) {
	var dlqSubject, originalStream string

	if strings.HasPrefix(msg.Subject, "metrics.") {
		dlqSubject = fmt.Sprintf("metrics.dlq.%s", sanitizeSubject(msg.Subject))
		originalStream = "METRICS"
	} else if strings.HasPrefix(msg.Subject, "telemetry.") {
		dlqSubject = fmt.Sprintf("telemetry.dlq.%s", sanitizeTelemetrySubject(msg.Subject))
		originalStream = "TELEMETRY_EVENTS"
	} else {
		b.logger.Error("unhandled subject prefix for DLQ routing",
			"subject", msg.Subject,
		)
		msg.Term()
		return
	}

	dlqMsg := &nats.Msg{
		Subject: dlqSubject,
		Data:    msg.Data,
		Header:  msg.Header,
	}
	if dlqMsg.Header == nil {
		dlqMsg.Header = nats.Header{}
	}
	dlqMsg.Header.Set("Nats-Original-Subject", msg.Subject)
	dlqMsg.Header.Set("Nats-Original-Stream", originalStream)

	if _, err := b.js.PublishMsg(dlqMsg); err != nil {
		b.logger.Error("failed to route message to DLQ",
			"subject", msg.Subject,
			"dlq_subject", dlqSubject,
			"error", err,
		)
		msg.Nak()
		return
	}

	b.logger.Warn("routed message to DLQ",
		"subject", msg.Subject,
		"dlq_subject", dlqSubject,
		"deliveries", deliveriesOf(msg),
	)
	msg.Term()
}

func deliveriesOf(msg *nats.Msg) uint64 {
	md, err := msg.Metadata()
	if err != nil {
		return 0
	}
	return md.NumDelivered
}

// Ack acknowledges a message (NATS handles this via msg.Ack() in Subscribe, so this is a no-op).
func (b *NATSBus) Ack(ctx context.Context, msg Message) error {
	return nil
}

// sanitizeSubject maps a wildcard subscription subject to a concrete dead-letter
// subject under metrics.dlq.> so exhausted messages stay inside the DLQ stream.
func sanitizeSubject(subject string) string {
	replacer := strings.NewReplacer(">", "all", "*", "any", ".", "-")
	return replacer.Replace(strings.TrimPrefix(subject, "metrics."))
}

func sanitizeTelemetrySubject(subject string) string {
	replacer := strings.NewReplacer(">", "all", "*", "any", ".", "-")
	return replacer.Replace(strings.TrimPrefix(subject, "telemetry."))
}

func getReplicas() int {
	if r := os.Getenv("NATS_REPLICAS"); r != "" {
		if n, err := strconv.Atoi(r); err == nil && n > 0 {
			return n
		}
	}
	return 1
}

// Nack negatively acknowledges a message (NATS handles this via msg.Nak() in Subscribe).
func (b *NATSBus) Nack(ctx context.Context, msg Message) error {
	return nil
}

// JetStream returns the underlying JetStream context for advanced usage
// (e.g. synchronous workers that need pull-based consumption).
func (b *NATSBus) JetStream() nats.JetStreamContext {
	return b.js
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
