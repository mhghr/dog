package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DedupStore struct {
	pool *pgxpool.Pool
}

func NewDedupStore(pool *pgxpool.Pool) *DedupStore {
	return &DedupStore{pool: pool}
}

func (s *DedupStore) Exists(ctx context.Context, eventID string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM telemetry_event_dedup WHERE event_id = $1)`,
		eventID,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check dedup: %w", err)
	}
	return exists, nil
}

func (s *DedupStore) MarkProcessed(ctx context.Context, eventID string) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO telemetry_event_dedup (event_id, processed_at) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		eventID, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("mark processed: %w", err)
	}
	return nil
}

func (s *DedupStore) CleanupOld(ctx context.Context, olderThan time.Duration) (int64, error) {
	tag, err := s.pool.Exec(ctx,
		`DELETE FROM telemetry_event_dedup WHERE processed_at < $1`,
		time.Now().Add(-olderThan),
	)
	if err != nil {
		return 0, fmt.Errorf("cleanup dedup: %w", err)
	}
	return tag.RowsAffected(), nil
}

func (s *DedupStore) InsertDLQEvent(ctx context.Context, eventID, eventType, errorReason string, retryCount int, payloadJSON []byte) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO telemetry_dlq_events (event_id, type, error_reason, retry_count, first_failed_at, last_failed_at, payload)
		 VALUES ($1, $2, $3, $4, NOW(), NOW(), $5)`,
		eventID, eventType, errorReason, retryCount, payloadJSON,
	)
	if err != nil {
		return fmt.Errorf("insert dlq event: %w", err)
	}
	return nil
}
