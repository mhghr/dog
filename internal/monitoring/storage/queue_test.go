package storage

import (
	"testing"
	"time"

	"monitoring-platform/internal/domain"
)

func sampleBatch(agentID string) domain.MetricBatch {
	return domain.MetricBatch{
		AgentID: agentID,
		Samples: []domain.MetricSample{
			{Name: "system.cpu.utilization", Value: 12.5, Timestamp: time.Now()},
		},
	}
}

func TestEnqueueDequeueRoundTrip(t *testing.T) {
	q, err := NewQueue(t.TempDir(), 100)
	if err != nil {
		t.Fatal(err)
	}
	defer q.Close()

	batch := sampleBatch("ag_1")
	if err := q.Enqueue(batch); err != nil {
		t.Fatal(err)
	}
	items, err := q.Dequeue(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Data.AgentID != "ag_1" {
		t.Errorf("AgentID = %q, want ag_1", items[0].Data.AgentID)
	}
	if items[0].Data.Samples[0].Value != 12.5 {
		t.Errorf("sample value = %v, want 12.5", items[0].Data.Samples[0].Value)
	}
}

func TestDequeueFIFOOrder(t *testing.T) {
	q, err := NewQueue(t.TempDir(), 100)
	if err != nil {
		t.Fatal(err)
	}
	defer q.Close()

	for _, id := range []string{"ag_1", "ag_2", "ag_3"} {
		if err := q.Enqueue(sampleBatch(id)); err != nil {
			t.Fatal(err)
		}
	}

	items, err := q.Dequeue(3)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}
	for i, want := range []string{"ag_1", "ag_2", "ag_3"} {
		if items[i].Data.AgentID != want {
			t.Errorf("item %d AgentID = %q, want %q", i, items[i].Data.AgentID, want)
		}
	}
}

func TestAckRemovesItem(t *testing.T) {
	q, err := NewQueue(t.TempDir(), 100)
	if err != nil {
		t.Fatal(err)
	}
	defer q.Close()

	for _, id := range []string{"ag_1", "ag_2"} {
		if err := q.Enqueue(sampleBatch(id)); err != nil {
			t.Fatal(err)
		}
	}

	items, err := q.Dequeue(2)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	if err := q.Ack(items[0].ID); err != nil {
		t.Fatal(err)
	}
	if got := q.Size(); got != 1 {
		t.Errorf("Size() = %d, want 1", got)
	}
}

func TestEmptyQueue(t *testing.T) {
	q, err := NewQueue(t.TempDir(), 100)
	if err != nil {
		t.Fatal(err)
	}
	defer q.Close()

	items, err := q.Dequeue(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Errorf("Dequeue on empty queue returned %d items, want 0", len(items))
	}
	if got := q.Size(); got != 0 {
		t.Errorf("Size() = %d, want 0", got)
	}
}

func TestEvictionBoundsSize(t *testing.T) {
	q, err := NewQueue(t.TempDir(), 0)
	if err != nil {
		t.Fatal(err)
	}
	defer q.Close()

	for i := 0; i < 10; i++ {
		batch := sampleBatch("ag_evict")
		for j := 0; j < 200; j++ {
			batch.Samples = append(batch.Samples, domain.MetricSample{
				Name: "system.cpu.utilization", Value: 1.0, Timestamp: time.Now(),
			})
		}
		if err := q.Enqueue(batch); err != nil {
			t.Fatal(err)
		}
	}

	if got := q.Size(); got != 1 {
		t.Errorf("Size() = %d, want 1 (only most recent survives with zero cap)", got)
	}
}
