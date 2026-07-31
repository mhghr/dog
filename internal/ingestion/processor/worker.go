package processor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"monitoring-platform/internal/domain"
	"monitoring-platform/internal/ingestion/messagebus"
)

// WorkerConfig configures the metric processor worker pool.
type WorkerConfig struct {
	NATSURL     string
	VMURL       string
	Subject     string
	Queue       string
	Durable     string
	Workers     int
	HTTPTimeout time.Duration
}

// WorkerPool consumes metric batches from the message bus and writes them to
// VictoriaMetrics. It creates its own MessageBus connection.
type WorkerPool struct {
	cfg     WorkerConfig
	logger  *slog.Logger
	client  *http.Client
	bus     messagebus.MessageBus
	mu      sync.Mutex
	started bool
}

// NewWorkerPool creates a worker pool. It connects to the message bus eagerly.
func NewWorkerPool(cfg WorkerConfig, logger *slog.Logger) (*WorkerPool, error) {
	bus, err := messagebus.NewNATSBus(messagebus.NATSConfig{
		URL:       cfg.NATSURL,
		Reconnect: true,
		MaxReconn: 10,
	}, logger)
	if err != nil {
		return nil, fmt.Errorf("connect to message bus: %w", err)
	}

	if cfg.Workers < 1 {
		cfg.Workers = 1
	}
	if cfg.HTTPTimeout == 0 {
		cfg.HTTPTimeout = 30 * time.Second
	}
	if cfg.Subject == "" {
		cfg.Subject = "metrics.>"
	}
	if cfg.Queue == "" {
		cfg.Queue = "metric-processors"
	}
	if cfg.Durable == "" {
		cfg.Durable = "metric-processor"
	}

	return &WorkerPool{
		cfg:    cfg,
		logger: logger,
		client: &http.Client{Timeout: cfg.HTTPTimeout},
		bus:    bus,
	}, nil
}

// Bus returns the underlying message bus connection.
func (wp *WorkerPool) Bus() messagebus.MessageBus {
	return wp.bus
}

// Start subscribes the worker pool to the metrics subject. One subscription is
// created; NATS queue groups distribute messages across pool members.
func (wp *WorkerPool) Start(ctx context.Context) error {
	wp.mu.Lock()
	defer wp.mu.Unlock()
	if wp.started {
		return fmt.Errorf("worker pool already started")
	}
	wp.started = true

	for i := 0; i < wp.cfg.Workers; i++ {
		workerID := i
		go wp.run(ctx, workerID)
	}

	return nil
}

func (wp *WorkerPool) run(ctx context.Context, workerID int) {
	logger := wp.logger.With("worker_id", workerID)

	err := wp.bus.Subscribe(ctx, messagebus.SubscribeOptions{
		Subject:    wp.cfg.Subject,
		Queue:      wp.cfg.Queue,
		Durable:    wp.cfg.Durable, // shared by all workers in the group
		DeliverNew: true,
	}, func(ctx context.Context, msg messagebus.Message) error {
		var batch domain.MetricBatch
		if err := json.Unmarshal(msg.Data, &batch); err != nil {
			wp.logger.Error("failed to unmarshal metric batch", "error", err, "subject", msg.Subject)
			return nil // don't redeliver malformed messages
		}

		vmReq := ConvertToVM(batch)
		if err := wp.writeToVM(ctx, vmReq); err != nil {
			return fmt.Errorf("write to VM: %w", err) // redeliver
		}

		logger.Debug("metrics written",
			"agent_id", batch.AgentID,
			"samples", len(batch.Samples),
		)
		return nil
	})
	if err != nil {
		logger.Error("failed to subscribe", "error", err)
		return
	}

	<-ctx.Done()
}

// writeToVM posts the samples to VictoriaMetrics' Prometheus import endpoint.
func (wp *WorkerPool) writeToVM(ctx context.Context, req VMWriteRequest) error {
	text := req.ToPrometheusText()
	if len(text) == 0 {
		return nil
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, wp.cfg.VMURL+"/api/v1/import/prometheus", bytes.NewReader([]byte(text)))
	if err != nil {
		return fmt.Errorf("create VM request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "text/plain")

	resp, err := wp.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("VM request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("VM returned status %d", resp.StatusCode)
	}
	return nil
}

// Close closes the message bus connection.
func (wp *WorkerPool) Close() error {
	return wp.bus.Close()
}
