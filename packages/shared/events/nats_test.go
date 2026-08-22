package events

import (
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
)

func TestShouldForwardDropsSelfEcho(t *testing.T) {
	cases := []struct {
		name        string
		selfOrigin  string
		eventOrigin string
		want        bool
	}{
		{name: "self echo dropped", selfOrigin: "localhost-1", eventOrigin: "localhost-1", want: false},
		{name: "peer forwarded", selfOrigin: "localhost-1", eventOrigin: "localhost-2", want: true},
		{name: "unknown origin forwarded during rollout", selfOrigin: "localhost-1", eventOrigin: "", want: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldForward(tc.selfOrigin, tc.eventOrigin); got != tc.want {
				t.Fatalf("shouldForward(%q, %q) = %v, want %v", tc.selfOrigin, tc.eventOrigin, got, tc.want)
			}
		})
	}
}

func TestCleanNATSURL(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{in: "nats://localhost:4222", want: "localhost:4222"},
		{in: "tls://nats.example.com:4222", want: "nats.example.com:4222"},
		{in: "localhost:4222", want: "localhost:4222"},
	}
	for _, tc := range cases {
		if got := CleanNATSURL(tc.in); got != tc.want {
			t.Fatalf("CleanNATSURL(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestDistributedPublisherDegradesGracefully verifies that publishing through a
// DistributedPublisher does not panic or stall when the NATS connection is not
// connected (e.g. broker temporarily down). The local bus still receives the
// event — ingestion must never depend on the live stream being healthy.
func TestDistributedPublisherDegradesGracefully(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(discardWriter{}, nil))
	local := NewBus()

	// A connection that will never be connected.
	nc, err := nats.Connect("nats://127.0.0.1:1",
		nats.Timeout(200*time.Millisecond),
		nats.MaxReconnects(0),
		nats.DontRandomize(),
		nats.NoEcho(),
	)
	if err != nil {
		t.Skipf("unexpected connect error: %v", err)
	}
	defer nc.Close()

	publisher := NewDistributedPublisher(local, nc, "localhost-test", logger)

	sub, cancel := local.Subscribe(1)
	defer cancel()

	publisher.Publish(Event{Name: "probe-result", ID: "abc", Data: []byte(`{}`)})

	select {
	case event := <-sub:
		if event.Name != "probe-result" {
			t.Fatalf("unexpected event name %q", event.Name)
		}
	case <-time.After(time.Second):
		t.Fatal("local bus did not receive the event")
	}
}

func TestEncodeLiveEvent(t *testing.T) {
	msg, err := encodeLiveEvent(Event{Name: "probe-result", ID: "abc", Data: []byte(`{"ok":true}`)}, "localhost-1")
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}
	if msg.Subject != liveEventSubject {
		t.Fatalf("subject = %q, want %q", msg.Subject, liveEventSubject)
	}
	if got := msg.Header.Get(originHeader); got != "localhost-1" {
		t.Fatalf("origin = %q, want localhost-1", got)
	}
	var decoded Event
	if err := json.Unmarshal(msg.Data, &decoded); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if decoded.Name != "probe-result" || decoded.ID != "abc" {
		t.Fatalf("round-trip mismatch: %+v", decoded)
	}
}

type discardWriter struct{}

func (discardWriter) Write(p []byte) (int, error) { return len(p), nil }
