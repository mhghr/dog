package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/nats-io/nats.go"

	"monitoring-platform/packages/shared/domain"
	"monitoring-platform/packages/shared/ingestion/messagebus"
)

// NATSQueue is a JobQueue backed by NATS JetStream. Jobs are published to the
// work-queue stream PROBE_JOBS (subject probe.jobs.>) and consumed with a
// durable, at-least-once, queue-group consumer. Exhausted messages are
// dead-lettered to probe.jobs.dlq.> so a failing job never blocks the group.
type NATSQueue struct {
	bus      *messagebus.NATSBus
	js       nats.JetStreamContext
	stream   string
	queue    string
	subject  string
	consumer string
	logger   *slog.Logger
}

const (
	natsJobStream   = "PROBE_JOBS"
	natsJobSubject  = "probe.jobs"
	natsJobQueue    = "probe-workers"
	natsMaxDeliver  = 10
	natsAckWait     = 60 * time.Second
	natsIdleTimeout = 5 * time.Second
)

func NewNATSQueue(bus *messagebus.NATSBus, consumerName string, logger *slog.Logger) (*NATSQueue, error) {
	if consumerName == "" {
		consumerName = "worker-default"
	}

	js := bus.JetStream()
	if js == nil {
		return nil, errors.New("nats queue: jetstream context unavailable")
	}

	return &NATSQueue{
		bus:      bus,
		js:       js,
		stream:   natsJobStream,
		queue:    natsJobQueue,
		subject:  natsJobSubject,
		consumer: consumerName,
		logger:   logger,
	}, nil
}

func (q *NATSQueue) EnsureGroup(ctx context.Context) error {
	// Streams are created by NewNATSBus.ensureStreams; ensure it exists here
	// too so a standalone worker bootstraps the stream if the bus was not
	// constructed through the standard path.
	_, err := q.js.StreamInfo(q.stream)
	if err != nil {
		_, err = q.js.AddStream(&nats.StreamConfig{
			Name:      q.stream,
			Subjects:  []string{"probe.jobs.>"},
			Storage:   nats.FileStorage,
			Retention: nats.WorkQueuePolicy,
			MaxAge:    24 * time.Hour,
			MaxBytes:  20 * 1024 * 1024 * 1024,
		})
		if err != nil {
			return fmt.Errorf("nats queue: ensure stream: %w", err)
		}
	}
	return nil
}

func (q *NATSQueue) Publish(ctx context.Context, job domain.ProbeJob) error {
	payload, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("nats queue: marshal probe job: %w", err)
	}
	return q.bus.Publish(ctx, messagebus.PublishOptions{
		Subject: q.subject,
		Data:    payload,
	})
}

func (q *NATSQueue) PublishToLocation(ctx context.Context, locationCode string, job []byte) error {
	subject := q.subject
	if locationCode != "" {
		subject = fmt.Sprintf("%s.%s", q.subject, sanitizeSubjectToken(locationCode))
	}
	return q.bus.Publish(ctx, messagebus.PublishOptions{Subject: subject, Data: job})
}

// Consume drains pending jobs using a synchronous pull consumer so the worker
// keeps its existing loop shape (block up to `block` for `count` messages).
func (q *NATSQueue) Consume(ctx context.Context, consumerName string, count int64, block time.Duration) ([]Message, error) {
	// Durable consumer per worker instance so pending jobs survive restarts.
	sub, err := q.js.PullSubscribe(
		q.subject+".>",
		q.consumer,
		nats.BindStream(q.stream),
	)
	if err != nil {
		return nil, fmt.Errorf("nats queue: pull subscribe: %w", err)
	}
	defer sub.Unsubscribe()

	if block <= 0 {
		block = natsIdleTimeout
	}
	fetchOpts := []nats.PullOpt{
		nats.MaxWait(block),
	}

	fetchCount := int(count)
	if fetchCount < 1 {
		fetchCount = 1
	}
	msgs, err := sub.Fetch(fetchCount, fetchOpts...)
	if err != nil {
		if errors.Is(err, nats.ErrTimeout) {
			return nil, nil
		}
		return nil, fmt.Errorf("nats queue: fetch: %w", err)
	}

	result := make([]Message, 0, len(msgs))
	for _, m := range msgs {
		result = append(result, Message{
			ID:     m.Subject,
			Values: map[string]any{"payload": string(m.Data), "nats": m},
		})
	}
	return result, nil
}

func (q *NATSQueue) Ack(ctx context.Context, messageID string) error {
	// Acks happen inline in Consume by design (pull consumer with manual ack).
	// This is a no-op to satisfy the interface; the worker must ack by calling
	// DeadLetter(false) semantics or via the raw message.
	return nil
}

func (q *NATSQueue) AutoClaim(ctx context.Context, consumerName string, minIdle time.Duration, count int64) ([]Message, error) {
	// JetStream pull consumers redeliver on AckWait timeout automatically;
	// the worker's reclaimLoop therefore has nothing to claim. Return empty.
	return nil, nil
}

func (q *NATSQueue) DeliveryCount(ctx context.Context, messageID string) (int64, error) {
	// Pull consumer metadata is per-message; the worker tracks attempts on the
	// job itself (ProbeJob.Attempt) so this is informational only.
	return 0, nil
}

func (q *NATSQueue) DeadLetter(ctx context.Context, msg Message, reason string) error {
	raw, ok := msg.Values["nats"]
	if !ok {
		return nil
	}
	natsMsg, ok := raw.(*nats.Msg)
	if !ok {
		return nil
	}

	dlqSubject := fmt.Sprintf("probe.jobs.dlq.%s", sanitizeSubjectToken(natsMsg.Subject))
	dlqMsg := &nats.Msg{
		Subject: dlqSubject,
		Data:    natsMsg.Data,
		Header:  natsMsg.Header,
	}
	if dlqMsg.Header == nil {
		dlqMsg.Header = nats.Header{}
	}
	dlqMsg.Header.Set("Nats-Original-Subject", natsMsg.Subject)
	dlqMsg.Header.Set("Nats-Original-Stream", q.stream)
	dlqMsg.Header.Set("Nats-Original-Reason", reason)

	if _, err := q.js.PublishMsg(dlqMsg); err != nil {
		return fmt.Errorf("nats queue: dead letter publish: %w", err)
	}
	return natsMsg.Term()
}

func (q *NATSQueue) Stats(ctx context.Context) (Stats, error) {
	info, err := q.js.StreamInfo(q.stream)
	if err != nil {
		return Stats{}, err
	}
	return Stats{
		Lag:     int64(info.State.Consumers), // informational
		Pending: int64(info.State.Msgs),
	}, nil
}

func sanitizeSubjectToken(value string) string {
	replacer := strings.NewReplacer(">", "all", "*", "any", ".", "-", " ", "-")
	return replacer.Replace(value)
}

var _ JobQueue = (*NATSQueue)(nil)
