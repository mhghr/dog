// Package repository defines persistence ports. Implementations live in
// internal/postgres; consumers depend only on these interfaces.
package repository

import (
	"context"
	"time"

	"monitoring-platform/internal/domain"
)

type MonitorRepository interface {
	Create(ctx context.Context, monitor *domain.Monitor) error
	GetByID(ctx context.Context, id string) (domain.Monitor, error)
	List(ctx context.Context, filter domain.MonitorListFilter) ([]domain.MonitorWithLastResult, int, error)
	Update(ctx context.Context, monitor *domain.Monitor) error
	Delete(ctx context.Context, id string) error
	SetEnabled(ctx context.Context, id string, enabled bool) (domain.Monitor, error)

	// ClaimDue selects due monitors with FOR UPDATE SKIP LOCKED, invokes
	// publish for each, and advances next_run_at only for published ones.
	ClaimDue(ctx context.Context, limit int, publish func(domain.Monitor) error) (int, error)
}

type ResultRepository interface {
	// InsertAndUpdateMonitor atomically stores the result (idempotent on
	// job_id) and updates the monitor status. Returns false on duplicates.
	InsertAndUpdateMonitor(ctx context.Context, result *domain.ProbeResult) (bool, error)
	ListByMonitor(ctx context.Context, monitorID string, limit, offset int) ([]domain.ProbeResult, int, error)
	LatestAttribute(ctx context.Context, monitorID, key string) (string, error)
	Series(ctx context.Context, monitorID string, from, to time.Time, stepSeconds int) (domain.MetricSeries, error)
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

type ProjectRepository interface {
	Create(ctx context.Context, project *domain.Project) error
	ListByOrganization(ctx context.Context, orgID string) ([]domain.Project, error)
}
