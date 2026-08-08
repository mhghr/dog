package spool

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"monitoring-platform/packages/shared/domain"

	_ "modernc.org/sqlite"
)

type StoredResult struct {
	ResultID      string
	JobID         string
	LeaseID       string
	MonitorID     string
	Result        *domain.ProbeResult
	Attempts      int
	NextAttemptAt time.Time
	CreatedAt     time.Time
}

type Spool struct {
	db *sql.DB
}

func New(dataDir string) (*Spool, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("create spool directory: %w", err)
	}

	dbPath := filepath.Join(dataDir, "spool.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open spool database: %w", err)
	}

	db.SetMaxOpenConns(1)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping spool database: %w", err)
	}

	if err := migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate spool database: %w", err)
	}

	return &Spool{db: db}, nil
}

func migrate(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS spool_results (
		result_id TEXT PRIMARY KEY,
		job_id TEXT NOT NULL,
		lease_id TEXT NOT NULL,
		monitor_id TEXT NOT NULL,
		payload BLOB NOT NULL,
		attempts INTEGER NOT NULL DEFAULT 0,
		next_attempt_at INTEGER NOT NULL,
		created_at INTEGER NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_spool_next_attempt ON spool_results(next_attempt_at);
	`
	_, err := db.Exec(query)
	return err
}

func (s *Spool) Store(leaseID string, result *domain.ProbeResult) error {
	payload, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("marshal result: %w", err)
	}

	now := time.Now().UTC().Unix()

	_, err = s.db.Exec(
		`INSERT INTO spool_results (result_id, job_id, lease_id, monitor_id, payload, attempts, next_attempt_at, created_at)
		 VALUES (?, ?, ?, ?, ?, 0, ?, ?)`,
		result.ID, result.JobID, leaseID, result.MonitorID, payload, now, now,
	)
	if err != nil {
		return fmt.Errorf("insert spool result: %w", err)
	}

	return nil
}

func (s *Spool) Pending(limit int) ([]StoredResult, error) {
	now := time.Now().UTC().Unix()

	rows, err := s.db.Query(
		`SELECT result_id, job_id, lease_id, monitor_id, payload, attempts, next_attempt_at, created_at
		 FROM spool_results
		 WHERE next_attempt_at <= ?
		 ORDER BY next_attempt_at ASC
		 LIMIT ?`,
		now, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("query pending spool results: %w", err)
	}
	defer rows.Close()

	var results []StoredResult

	for rows.Next() {
		var sr StoredResult
		var nextAttemptAt, createdAt int64
		var payload []byte

		if err := rows.Scan(&sr.ResultID, &sr.JobID, &sr.LeaseID, &sr.MonitorID, &payload, &sr.Attempts, &nextAttemptAt, &createdAt); err != nil {
			return nil, fmt.Errorf("scan spool row: %w", err)
		}

		var result domain.ProbeResult
		if err := json.Unmarshal(payload, &result); err != nil {
			return nil, fmt.Errorf("unmarshal spool payload: %w", err)
		}
		sr.Result = &result
		sr.NextAttemptAt = time.Unix(nextAttemptAt, 0).UTC()
		sr.CreatedAt = time.Unix(createdAt, 0).UTC()

		results = append(results, sr)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate spool rows: %w", err)
	}

	return results, nil
}

func (s *Spool) Ack(resultIDs []string) error {
	if len(resultIDs) == 0 {
		return nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin ack transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`DELETE FROM spool_results WHERE result_id = ?`)
	if err != nil {
		return fmt.Errorf("prepare ack statement: %w", err)
	}
	defer stmt.Close()

	for _, id := range resultIDs {
		if _, err := stmt.Exec(id); err != nil {
			return fmt.Errorf("ack result %s: %w", id, err)
		}
	}

	return tx.Commit()
}

func (s *Spool) MarkFailed(resultID string, nextAttempt time.Time) error {
	_, err := s.db.Exec(
		`UPDATE spool_results
		 SET attempts = attempts + 1,
		     next_attempt_at = ?
		 WHERE result_id = ?`,
		nextAttempt.UTC().Unix(), resultID,
	)
	if err != nil {
		return fmt.Errorf("mark spool result failed: %w", err)
	}

	return nil
}

func (s *Spool) Stats() (totalCount int, totalBytes int64, oldestAge time.Duration) {
	now := time.Now().UTC().Unix()

	var oldestUnix int64
	err := s.db.QueryRow(
		`SELECT COUNT(*), COALESCE(SUM(LENGTH(payload)), 0), COALESCE(MIN(created_at), ?)
		 FROM spool_results`,
		now,
	).Scan(&totalCount, &totalBytes, &oldestUnix)
	if err != nil {
		return 0, 0, 0
	}

	if totalCount > 0 {
		oldestAge = time.Since(time.Unix(oldestUnix, 0).UTC())
	}

	return
}

func (s *Spool) Close() error {
	return s.db.Close()
}
