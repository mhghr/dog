// Package repository defines persistence ports. Implementations live in
// internal/postgres; consumers depend only on these interfaces.
package repository

import (
	"context"
	"time"

	"monitoring-platform/packages/shared/domain"
)

type ResultRepository interface {
	// InsertAndUpdateMonitor atomically stores the result (idempotent on
	// job_id) and updates the monitor status. Returns false on duplicates.
	InsertAndUpdateMonitor(ctx context.Context, result *domain.ProbeResult) (bool, error)
	ListByMonitor(ctx context.Context, monitorID string, limit, offset int) ([]domain.ProbeResult, int, error)
	LatestAttribute(ctx context.Context, monitorID, key string) (string, error)
	Series(ctx context.Context, monitorID string, from, to time.Time, stepSeconds int) (domain.MetricSeries, error)
	// SeriesByProbe returns time-bucketed series per probe location for a
	// monitor, used by the resource monitoring dashboard.
	SeriesByProbe(ctx context.Context, monitorID string, from, to time.Time, stepSeconds int) ([]domain.ProbeSeries, error)
	// LatestResultsByProbe returns the most recent result per probe location.
	LatestResultsByProbe(ctx context.Context, monitorID string) ([]domain.ProbeResult, error)
	DashboardSummary(ctx context.Context) (domain.DashboardSummary, error)
}

type LocationRepository interface {
	List(ctx context.Context) ([]domain.ProbeLocation, error)
	GetByCode(ctx context.Context, code string) (domain.ProbeLocation, error)
	Create(ctx context.Context, location *domain.ProbeLocation) error
}

type StatusPageRepository interface {
	Create(ctx context.Context, page *domain.StatusPage) error
	Update(ctx context.Context, page *domain.StatusPage) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context) ([]domain.StatusPage, error)
	GetByID(ctx context.Context, id string) (domain.StatusPage, error)
	PublicBySlug(ctx context.Context, slug string) (domain.PublicStatusPage, error)
}

type OrganizationRepository interface {
	Create(ctx context.Context, org *domain.Organization) error
	GetBySlug(ctx context.Context, slug string) (domain.Organization, error)
}

type ProbeAssignmentRepository interface {
	Create(ctx context.Context, assign *domain.ProbeAssignment) error
	ListByMonitor(ctx context.Context, monitorID string) ([]domain.ProbeAssignment, error)
	ListByProbe(ctx context.Context, probeID string) ([]domain.ProbeAssignment, error)
	Delete(ctx context.Context, monitorID, probeID string) error
}

type MonitorJobRepository interface {
	Create(ctx context.Context, job *domain.MonitorJob) error
	GetByID(ctx context.Context, id string) (domain.MonitorJob, error)
	ClaimPending(ctx context.Context, probeID string, limit int) ([]domain.MonitorJob, error)
	StartJob(ctx context.Context, jobID, probeID string) error
	FinishJob(ctx context.Context, jobID string, status domain.JobStatus, errMsg string) error
	ListByMonitor(ctx context.Context, monitorID string, limit, offset int) ([]domain.MonitorJob, int, error)
}

type ResourceHealthRepository interface {
	UpsertState(ctx context.Context, state *domain.ResourceHealthState) error
	GetState(ctx context.Context, resourceID string) (domain.ResourceHealthState, error)
	ListByWorkspace(ctx context.Context, workspaceID string) ([]domain.ResourceHealthState, error)
}
