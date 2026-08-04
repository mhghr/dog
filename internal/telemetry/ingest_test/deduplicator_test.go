package ingest_test

import (
	"context"
	"testing"

	"monitoring-platform/internal/telemetry/ingest"
)

type mockDedupStore struct {
	events map[string]bool
}

func newMockDedupStore() *mockDedupStore {
	return &mockDedupStore{events: make(map[string]bool)}
}

func (m *mockDedupStore) Exists(ctx context.Context, eventID string) (bool, error) {
	return m.events[eventID], nil
}

func (m *mockDedupStore) MarkProcessed(ctx context.Context, eventID string) error {
	m.events[eventID] = true
	return nil
}

func TestDeduplicator_FirstEvent_NotDuplicate(t *testing.T) {
	store := newMockDedupStore()
	d := ingest.NewDeduplicator(store, 100, 5*60)
	ctx := context.Background()

	dup, err := d.IsDuplicate(ctx, "event-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dup {
		t.Fatal("first event should not be duplicate")
	}
}

func TestDeduplicator_SecondEvent_IsDuplicate(t *testing.T) {
	store := newMockDedupStore()
	d := ingest.NewDeduplicator(store, 100, 5*60)
	ctx := context.Background()

	if err := d.MarkProcessed(ctx, "event-1", store); err != nil {
		t.Fatalf("mark processed: %v", err)
	}
	dup, err := d.IsDuplicate(ctx, "event-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !dup {
		t.Fatal("second event should be duplicate")
	}
}

func TestDeduplicator_MemoryCacheHit(t *testing.T) {
	store := newMockDedupStore()
	d := ingest.NewDeduplicator(store, 100, 5*60)
	ctx := context.Background()

	if err := d.MarkProcessed(ctx, "event-1", store); err != nil {
		t.Fatalf("mark processed: %v", err)
	}
	dup, err := d.IsDuplicate(ctx, "event-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !dup {
		t.Fatal("event in memory cache should be duplicate")
	}
}

func TestDeduplicator_MarkProcessedIdempotent(t *testing.T) {
	store := newMockDedupStore()
	d := ingest.NewDeduplicator(store, 100, 5*60)
	ctx := context.Background()

	if err := d.MarkProcessed(ctx, "event-1", store); err != nil {
		t.Fatalf("first mark: %v", err)
	}
	if err := d.MarkProcessed(ctx, "event-1", store); err != nil {
		t.Fatalf("second mark: %v", err)
	}
}
