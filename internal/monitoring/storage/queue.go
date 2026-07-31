package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"monitoring-platform/internal/domain"

	_ "modernc.org/sqlite"
)

// QueueItem is a single buffered metric batch.
type QueueItem struct {
	ID        string
	Data      domain.MetricBatch
	CreatedAt time.Time
	Attempts  int
}

// Queue is a disk-backed FIFO of metric batches.
type Queue struct {
	db      *sql.DB
	maxSize int64
}

// NewQueue opens (or creates) the spool database under dataDir.
func NewQueue(dataDir string, maxSizeMB int) (*Queue, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("create spool dir: %w", err)
	}

	dbPath := filepath.Join(dataDir, "spool.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open spool db: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping spool db: %w", err)
	}

	if err := migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate spool db: %w", err)
	}

	return &Queue{
		db:      db,
		maxSize: int64(maxSizeMB) * 1024 * 1024,
	}, nil
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS metric_spool (
			id TEXT PRIMARY KEY,
			payload BLOB NOT NULL,
			created_at INTEGER NOT NULL,
			attempts INTEGER NOT NULL DEFAULT 0
		);
		CREATE INDEX IF NOT EXISTS idx_spool_created ON metric_spool(created_at);
	`)
	return err
}

// Enqueue stores a batch, evicting the oldest entry if over the size cap.
func (q *Queue) Enqueue(batch domain.MetricBatch) error {
	payload, err := json.Marshal(batch)
	if err != nil {
		return fmt.Errorf("marshal batch: %w", err)
	}

	if q.sizeBytes() >= q.maxSize {
		if oldest, err := q.oldestID(); err == nil {
			q.delete(oldest)
		}
	}

	id := fmt.Sprintf("%s-%d", batch.AgentID, time.Now().UnixNano())
	_, err = q.db.Exec(
		`INSERT INTO metric_spool (id, payload, created_at) VALUES (?, ?, ?)`,
		id, payload, time.Now().Unix(),
	)
	if err != nil {
		return fmt.Errorf("insert spool item: %w", err)
	}
	return nil
}

// Dequeue returns up to limit oldest items.
func (q *Queue) Dequeue(limit int) ([]QueueItem, error) {
	rows, err := q.db.Query(
		`SELECT id, payload, created_at, attempts FROM metric_spool ORDER BY created_at ASC LIMIT ?`, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("dequeue: %w", err)
	}
	defer rows.Close()

	var items []QueueItem
	for rows.Next() {
		var item QueueItem
		var payload []byte
		var createdAt int64
		if err := rows.Scan(&item.ID, &payload, &createdAt, &item.Attempts); err != nil {
			return nil, fmt.Errorf("scan spool row: %w", err)
		}
		if err := json.Unmarshal(payload, &item.Data); err != nil {
			continue // skip corrupt rows
		}
		item.CreatedAt = time.Unix(createdAt, 0)
		items = append(items, item)
	}
	return items, nil
}

// Ack removes an item by ID.
func (q *Queue) Ack(id string) error {
	_, err := q.db.Exec(`DELETE FROM metric_spool WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("ack spool item: %w", err)
	}
	return nil
}

// Size returns the number of buffered batches.
func (q *Queue) Size() int {
	var count int
	_ = q.db.QueryRow(`SELECT COUNT(*) FROM metric_spool`).Scan(&count)
	return count
}

// Close closes the database.
func (q *Queue) Close() error {
	return q.db.Close()
}

func (q *Queue) sizeBytes() int64 {
	var total int64
	_ = q.db.QueryRow(`SELECT COALESCE(SUM(LENGTH(payload)), 0) FROM metric_spool`).Scan(&total)
	return total
}

func (q *Queue) oldestID() (string, error) {
	var id string
	err := q.db.QueryRow(`SELECT id FROM metric_spool ORDER BY created_at ASC LIMIT 1`).Scan(&id)
	return id, err
}

func (q *Queue) delete(id string) {
	_, _ = q.db.Exec(`DELETE FROM metric_spool WHERE id = ?`, id)
}
