package spool

import (
	"context"
	"log/slog"
	"math"
	"math/rand"
	"sync"
	"time"

	"monitoring-platform/internal/domain"
)

type BatcherConfig struct {
	BatchSize     int
	MaxBatchBytes int
	FlushInterval time.Duration
	MaxRetries    int
	BackoffBase   time.Duration
	BackoffMax    time.Duration
	JitterFactor  float64
}

func DefaultBatcherConfig() BatcherConfig {
	return BatcherConfig{
		BatchSize:     250,
		MaxBatchBytes: 1048576,
		FlushInterval: 250 * time.Millisecond,
		MaxRetries:    5,
		BackoffBase:   time.Second,
		BackoffMax:    30 * time.Second,
		JitterFactor:  0.2,
	}
}

type ResultSender interface {
	SendBatch(ctx context.Context, results []*domain.ProbeResult) ([]string, error)
}

type Batcher struct {
	spool  *Spool
	sender ResultSender
	config BatcherConfig
	logger *slog.Logger

	flushCh    chan struct{}
	closeOnce  sync.Once
	closed     chan struct{}
}

func NewBatcher(spool *Spool, sender ResultSender, config BatcherConfig, logger *slog.Logger) *Batcher {
	if config.BatchSize <= 0 {
		config.BatchSize = 250
	}
	if config.MaxBatchBytes <= 0 {
		config.MaxBatchBytes = 1048576
	}
	if config.FlushInterval <= 0 {
		config.FlushInterval = 250 * time.Millisecond
	}
	if config.MaxRetries <= 0 {
		config.MaxRetries = 5
	}
	if config.BackoffBase <= 0 {
		config.BackoffBase = time.Second
	}
	if config.BackoffMax <= 0 {
		config.BackoffMax = 30 * time.Second
	}
	if config.JitterFactor <= 0 {
		config.JitterFactor = 0.2
	}

	return &Batcher{
		spool:   spool,
		sender:  sender,
		config:  config,
		logger:  logger,
		flushCh: make(chan struct{}, 1),
		closed:  make(chan struct{}),
	}
}

func (b *Batcher) Run(ctx context.Context) {
	defer close(b.closed)

	ticker := time.NewTicker(b.config.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			b.flush()
			return
		case <-ticker.C:
			b.flush()
		case <-b.flushCh:
			b.flush()
		}
	}
}

func (b *Batcher) Flush() {
	select {
	case b.flushCh <- struct{}{}:
	default:
	}
}

func (b *Batcher) flush() {
	for {
		pending, err := b.spool.Pending(b.config.BatchSize)
		if err != nil {
			b.logger.Error("read pending spool results failed", "error", err)
			return
		}

		if len(pending) == 0 {
			return
		}

		results := make([]*domain.ProbeResult, 0, len(pending))
		for _, sr := range pending {
			results = append(results, sr.Result)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		storedIDs, err := b.sender.SendBatch(ctx, results)
		cancel()

		if err != nil {
			b.logger.Error("send batch failed", "error", err, "count", len(results))
			for _, sr := range pending {
				b.markFailed(sr)
			}
			return
		}

		storedSet := make(map[string]bool, len(storedIDs))
		for _, id := range storedIDs {
			storedSet[id] = true
		}

		var idsToAck []string
		for _, sr := range pending {
			if storedSet[sr.ResultID] {
				idsToAck = append(idsToAck, sr.ResultID)
			} else {
				b.markFailed(sr)
			}
		}

		if len(idsToAck) > 0 {
			if err := b.spool.Ack(idsToAck); err != nil {
				b.logger.Error("ack spool results failed", "error", err)
				return
			}
		}

		b.logger.Debug("batch processed", "sent", len(results), "acked", len(idsToAck))

		if len(pending) < b.config.BatchSize {
			return
		}
	}
}

func (b *Batcher) markFailed(sr StoredResult) {
	if sr.Attempts >= b.config.MaxRetries {
		b.logger.Warn("spool result exceeded max retries, discarding",
			"result_id", sr.ResultID,
			"attempts", sr.Attempts,
		)
		if err := b.spool.Ack([]string{sr.ResultID}); err != nil {
			b.logger.Error("discard spool result failed", "result_id", sr.ResultID, "error", err)
		}
		return
	}

	backoff := b.calculateBackoff(sr.Attempts + 1)
	nextAttempt := time.Now().Add(backoff)
	if err := b.spool.MarkFailed(sr.ResultID, nextAttempt); err != nil {
		b.logger.Error("mark spool result failed", "result_id", sr.ResultID, "error", err)
	}
}

func (b *Batcher) calculateBackoff(attempt int) time.Duration {
	backoff := float64(b.config.BackoffBase) * math.Pow(2, float64(attempt-1))

	if backoff > float64(b.config.BackoffMax) {
		backoff = float64(b.config.BackoffMax)
	}

	jitter := 1 - b.config.JitterFactor + (2 * b.config.JitterFactor * rand.Float64())
	backoff *= jitter

	return time.Duration(backoff)
}
