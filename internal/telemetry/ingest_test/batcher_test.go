package ingest_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"monitoring-platform/internal/telemetry/ingest"
)

func TestBatchWriter_FlushesOnSize(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(204)
	}))
	defer srv.Close()

	bw := ingest.NewBatchWriter(srv.URL, ingest.BatchWriterConfig{
		BatchSize:     2,
		FlushInterval: 10_000,
		HTTPTimeout:   5_000,
	})
	defer bw.Shutdown(context.Background())

	ids, err := bw.Add("event-1", []string{"metric_1 1 123"})
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if ids != nil {
		t.Fatal("should not flush yet")
	}

	ids, err = bw.Add("event-2", []string{"metric_2 1 123"})
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if ids == nil {
		t.Fatal("should flush on batch size")
	}
	if len(ids) != 2 {
		t.Fatalf("expected 2 IDs, got %d", len(ids))
	}
}

func TestBatchWriter_ManualFlush(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(204)
	}))
	defer srv.Close()

	bw := ingest.NewBatchWriter(srv.URL, ingest.BatchWriterConfig{
		BatchSize:     100,
		FlushInterval: 10_000,
		HTTPTimeout:   5_000,
	})
	defer bw.Shutdown(context.Background())

	_, _ = bw.Add("event-1", []string{"metric_1 1 123"})
	_, _ = bw.Add("event-2", []string{"metric_2 1 123"})

	ids, err := bw.Flush(context.Background())
	if err != nil {
		t.Fatalf("flush: %v", err)
	}
	if len(ids) != 2 {
		t.Fatalf("expected 2 IDs, got %d", len(ids))
	}
}

func TestBatchWriter_FlushReturnsErrorOnVMFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()

	bw := ingest.NewBatchWriter(srv.URL, ingest.BatchWriterConfig{
		BatchSize:     1,
		FlushInterval: 10_000,
		HTTPTimeout:   5_000,
	})
	defer bw.Shutdown(context.Background())

	ids, err := bw.Add("event-1", []string{"metric_1 1 123"})
	if err == nil {
		t.Fatal("expected error on VM failure")
	}
	if ids != nil {
		t.Fatal("should not return IDs on failure")
	}
}

func TestBatchWriter_FlushEmptyBatch(t *testing.T) {
	bw := ingest.NewBatchWriter("http://localhost:9999", ingest.BatchWriterConfig{
		BatchSize:     100,
		FlushInterval: 10_000,
		HTTPTimeout:   5_000,
	})
	defer bw.Shutdown(context.Background())

	ids, err := bw.Flush(context.Background())
	if err != nil {
		t.Fatalf("unexpected error on empty flush: %v", err)
	}
	if ids != nil {
		t.Fatal("empty flush should return nil IDs")
	}
}
