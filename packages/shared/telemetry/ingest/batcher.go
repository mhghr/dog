package ingest

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

var ErrVMFlushFailed = errors.New("victoriametrics flush failed")

type BatchWriterConfig struct {
	BatchSize     int
	FlushInterval time.Duration
	HTTPTimeout   time.Duration
}

type pendingBatch struct {
	eventIDs []string
	lines    []string
}

type BatchWriter struct {
	vmURL  string
	cfg    BatchWriterConfig
	client *http.Client
	mu     sync.Mutex
	pending pendingBatch
}

func NewBatchWriter(vmURL string, cfg BatchWriterConfig) *BatchWriter {
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 1000
	}
	if cfg.HTTPTimeout < time.Millisecond {
		cfg.HTTPTimeout = 30 * time.Second
	}
	return &BatchWriter{
		vmURL:  strings.TrimRight(vmURL, "/"),
		cfg:    cfg,
		client: &http.Client{Timeout: cfg.HTTPTimeout},
	}
}

func (bw *BatchWriter) Add(eventID string, lines []string) ([]string, error) {
	bw.mu.Lock()
	bw.pending.eventIDs = append(bw.pending.eventIDs, eventID)
	bw.pending.lines = append(bw.pending.lines, lines...)
	shouldFlush := len(bw.pending.eventIDs) >= bw.cfg.BatchSize
	bw.mu.Unlock()

	if shouldFlush {
		return bw.Flush(context.Background())
	}
	return nil, nil
}

func (bw *BatchWriter) Flush(ctx context.Context) ([]string, error) {
	bw.mu.Lock()
	if len(bw.pending.eventIDs) == 0 {
		bw.mu.Unlock()
		return nil, nil
	}
	batch := bw.pending
	bw.pending = pendingBatch{}
	bw.mu.Unlock()

	if err := bw.writeToVM(ctx, batch.lines); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrVMFlushFailed, err)
	}
	return batch.eventIDs, nil
}

func (bw *BatchWriter) writeToVM(ctx context.Context, lines []string) error {
	if len(lines) == 0 {
		return nil
	}
	body := strings.Join(lines, "\n")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		bw.vmURL+"/api/v1/import/prometheus",
		bytes.NewReader([]byte(body)))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "text/plain")

	resp, err := bw.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("VM returned %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

func (bw *BatchWriter) Shutdown(ctx context.Context) error {
	_, err := bw.Flush(ctx)
	return err
}
