package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
)

// liveEventSubject is the NATS subject live events are published to. Every API
// replica subscribes to it so each instance can serve the same SSE stream to
// its connected browsers (multi-instance SSE).
const liveEventSubject = "events.live"

// originHeader carries the publishing instance id so a relay can drop its own
// echoed events instead of double-delivering to local subscribers.
const originHeader = "dog-event-origin"

// DistributedPublisher fans a live event out to the shared NATS bus AND the
// local in-process Bus. Local subscribers see the event immediately; the NATS
// relay on other replicas redelivers it to their local buses.
type DistributedPublisher struct {
	local *Bus
	nc    *nats.Conn
	origin string
	logger *slog.Logger
}

// NewDistributedPublisher builds a publisher that broadcasts to NATS and to the
// local bus. origin uniquely identifies this instance (used for echo drop).
func NewDistributedPublisher(local *Bus, nc *nats.Conn, origin string, logger *slog.Logger) *DistributedPublisher {
	return &DistributedPublisher{local: local, nc: nc, origin: origin, logger: logger}
}

// Publish delivers to the local bus immediately and best-effort to NATS. A
// failed NATS publish is logged, never fatal: ingestion must not stall because
// the live stream is momentarily unavailable (PostgreSQL remains the source of
// truth).
func (d *DistributedPublisher) Publish(event Event) {
	d.local.Publish(event)

	msg, err := encodeLiveEvent(event, d.origin)
	if err != nil {
		d.logger.Error("marshal live event", "name", event.Name, "error", err)
		return
	}
	if err := d.nc.PublishMsg(msg); err != nil {
		d.logger.Warn("publish live event to NATS", "name", event.Name, "error", err)
	}
}

// encodeLiveEvent serializes an event for the wire (subject + payload + origin
// header). Split out of Publish so the wire format is unit-testable without a
// live NATS connection.
func encodeLiveEvent(event Event, origin string) (*nats.Msg, error) {
	payload, err := json.Marshal(event)
	if err != nil {
		return nil, err
	}
	msg := &nats.Msg{
		Subject: liveEventSubject,
		Data:    payload,
		Header:  nats.Header{},
	}
	msg.Header.Set(originHeader, origin)
	return msg, nil
}

// NATSRelay subscribes to the shared live-event subject and forwards events
// originating from OTHER instances into the local Bus. Events echoed by this
// very instance are dropped so local subscribers never see a copy twice.
type NATSRelay struct {
	nc     *nats.Conn
	local  *Bus
	origin string
	logger *slog.Logger
}

// NewNATSRelay builds a relay for the given instance identity.
func NewNATSRelay(local *Bus, nc *nats.Conn, origin string, logger *slog.Logger) *NATSRelay {
	return &NATSRelay{nc: nc, local: local, origin: origin, logger: logger}
}

// Start subscribes to the live-event subject and blocks until the context is
// cancelled or the connection is closed. Call it in a goroutine.
func (r *NATSRelay) Start(ctx context.Context) error {
	sub, err := r.nc.Subscribe(liveEventSubject, func(msg *nats.Msg) {
		if shouldForward(r.origin, msg.Header.Get(originHeader)) {
			var event Event
			if err := json.Unmarshal(msg.Data, &event); err != nil {
				r.logger.Warn("decode live event", "error", err)
				return
			}
			r.local.Publish(event)
		}
	})
	if err != nil {
		return fmt.Errorf("subscribe live events: %w", err)
	}
	defer sub.Unsubscribe()

	<-ctx.Done()
	return nil
}

// shouldForward reports whether an event carrying the given origin should be
// delivered locally. Events from the same instance are echoes of the local
// publish and must be dropped; events with an unknown origin are forwarded so
// a mixed-version fleet keeps working during rollout.
func shouldForward(selfOrigin, eventOrigin string) bool {
	return eventOrigin != selfOrigin
}

// WaitForNATSConnect blocks until the NATS connection is ready or the context
// is cancelled, so a relay can be started safely at boot.
func WaitForNATSConnect(ctx context.Context, nc *nats.Conn) error {
	for {
		if nc.IsConnected() {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}
}

// CleanNATSURL strips the URL scheme so instance origins are stable and usable
// in header values (no "://").
func CleanNATSURL(url string) string {
	url = strings.TrimPrefix(url, "nats://")
	url = strings.TrimPrefix(url, "tls://")
	return url
}
