package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"monitoring-platform/packages/shared/domain"
)

type MonitorJobRepository struct {
	pool *pgxpool.Pool
}

func NewMonitorJobRepository(pool *pgxpool.Pool) *MonitorJobRepository {
	return &MonitorJobRepository{pool: pool}
}

func (r *MonitorJobRepository) Create(ctx context.Context, job *domain.MonitorJob) error {
	err := r.pool.QueryRow(ctx, `
		INSERT INTO monitor_jobs (monitor_id, probe_id, scheduled_at, status, attempt)
		VALUES ($1::uuid, $2, $3, $4::job_status, $5)
		RETURNING id::text, created_at`,
		job.MonitorID, job.ProbeID, job.ScheduledAt, string(job.Status), job.Attempt,
	).Scan(&job.ID, &job.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert monitor job: %w", err)
	}
	return nil
}

func (r *MonitorJobRepository) GetByID(ctx context.Context, id string) (domain.MonitorJob, error) {
	return r.scanJob(r.pool.QueryRow(ctx, jobSelect+` FROM monitor_jobs WHERE id = $1::uuid`, id))
}

func (r *MonitorJobRepository) ClaimPending(ctx context.Context, probeID string, limit int) ([]domain.MonitorJob, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin claim transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, jobSelect+` FROM monitor_jobs
		WHERE status = 'pending' AND (probe_id = $1::uuid OR probe_id IS NULL)
		ORDER BY scheduled_at LIMIT $2
		FOR UPDATE SKIP LOCKED`, probeID, limit)
	if err != nil {
		return nil, fmt.Errorf("claim pending jobs: %w", err)
	}

	jobs, err := r.scanJobRows(rows)
	rows.Close()
	if err != nil {
		return nil, err
	}

	for i := range jobs {
		if _, err := tx.Exec(ctx, `
			UPDATE monitor_jobs SET status = 'running', probe_id = $1::uuid, started_at = NOW()
			WHERE id = $2::uuid`,
			probeID, jobs[i].ID); err != nil {
			return nil, fmt.Errorf("start claimed job: %w", err)
		}
		jobs[i].Status = domain.JobRunning
		jobs[i].ProbeID = &probeID
		now := time.Now()
		jobs[i].StartedAt = &now
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit claim: %w", err)
	}
	return jobs, nil
}

func (r *MonitorJobRepository) StartJob(ctx context.Context, jobID, probeID string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE monitor_jobs SET status = 'running', probe_id = $2::uuid, started_at = NOW()
		WHERE id = $1::uuid AND status = 'pending'`, jobID, probeID)
	if err != nil {
		return fmt.Errorf("start job: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *MonitorJobRepository) FinishJob(ctx context.Context, jobID string, status domain.JobStatus, errMsg string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE monitor_jobs
		SET status = $2::job_status, finished_at = NOW(), error_message = $3
		WHERE id = $1::uuid`, jobID, string(status), errMsg)
	if err != nil {
		return fmt.Errorf("finish job: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *MonitorJobRepository) ListByMonitor(ctx context.Context, monitorID string, limit, offset int) ([]domain.MonitorJob, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM monitor_jobs WHERE monitor_id = $1::uuid`, monitorID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count monitor jobs: %w", err)
	}

	rows, err := r.pool.Query(ctx, jobSelect+` FROM monitor_jobs
		WHERE monitor_id = $1::uuid
		ORDER BY scheduled_at DESC LIMIT $2 OFFSET $3`, monitorID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list monitor jobs: %w", err)
	}
	defer rows.Close()

	jobs, err := r.scanJobRows(rows)
	return jobs, total, err
}

const jobSelect = `
	SELECT id::text, monitor_id::text, probe_id::text, scheduled_at, started_at,
	       finished_at, status::text, attempt, COALESCE(error_message,''), created_at`

func (r *MonitorJobRepository) scanJob(row pgx.Row) (domain.MonitorJob, error) {
	var job domain.MonitorJob
	var status string
	err := row.Scan(&job.ID, &job.MonitorID, &job.ProbeID, &job.ScheduledAt,
		&job.StartedAt, &job.FinishedAt, &status, &job.Attempt,
		&job.ErrorMessage, &job.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.MonitorJob{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.MonitorJob{}, err
	}
	job.Status = domain.JobStatus(status)
	return job, nil
}

func (r *MonitorJobRepository) scanJobRows(rows pgx.Rows) ([]domain.MonitorJob, error) {
	var jobs []domain.MonitorJob
	for rows.Next() {
		job, err := r.scanJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
}
