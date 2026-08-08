package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"monitoring-platform/packages/shared/domain"
)

type StatusPageRepository struct {
	pool *pgxpool.Pool
}

func NewStatusPageRepository(pool *pgxpool.Pool) *StatusPageRepository {
	return &StatusPageRepository{pool: pool}
}

func isUniqueViolation(err error) bool {
	var pgError *pgconn.PgError
	return errors.As(err, &pgError) && pgError.Code == "23505"
}

func (r *StatusPageRepository) Create(ctx context.Context, page *domain.StatusPage) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin status page create: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	err = tx.QueryRow(ctx, `
		INSERT INTO status_pages (slug, name, description, enabled)
		VALUES ($1, $2, $3, $4)
		RETURNING id::text, created_at, updated_at`,
		page.Slug, page.Name, page.Description, page.Enabled,
	).Scan(&page.ID, &page.CreatedAt, &page.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.ErrDuplicate
		}
		return fmt.Errorf("insert status page: %w", err)
	}

	if err := insertComponents(ctx, tx, page.ID, page.Components); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit status page create: %w", err)
	}
	return nil
}

func (r *StatusPageRepository) Update(ctx context.Context, page *domain.StatusPage) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin status page update: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	commandTag, err := tx.Exec(ctx, `
		UPDATE status_pages
		SET slug = $2, name = $3, description = $4, enabled = $5, updated_at = NOW()
		WHERE id = $1`,
		page.ID, page.Slug, page.Name, page.Description, page.Enabled,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.ErrDuplicate
		}
		return fmt.Errorf("update status page: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	if _, err := tx.Exec(ctx, `DELETE FROM status_page_components WHERE status_page_id = $1`, page.ID); err != nil {
		return fmt.Errorf("clear status page components: %w", err)
	}

	if err := insertComponents(ctx, tx, page.ID, page.Components); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit status page update: %w", err)
	}
	return nil
}

func insertComponents(ctx context.Context, tx pgx.Tx, pageID string, components []domain.StatusPageComponent) error {
	for _, component := range components {
		_, err := tx.Exec(ctx, `
			INSERT INTO status_page_components (status_page_id, monitor_id, display_name, sort_order)
			VALUES ($1, $2, $3, $4)`,
			pageID, component.MonitorID, component.DisplayName, component.SortOrder,
		)
		if err != nil {
			var pgError *pgconn.PgError
			if errors.As(err, &pgError) && pgError.Code == "23503" {
				return domain.ErrNotFound
			}
			return fmt.Errorf("insert status page component: %w", err)
		}
	}
	return nil
}

func (r *StatusPageRepository) Delete(ctx context.Context, id string) error {
	commandTag, err := r.pool.Exec(ctx, `DELETE FROM status_pages WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete status page: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *StatusPageRepository) List(ctx context.Context) ([]domain.StatusPage, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT sp.id::text, sp.slug, sp.name, sp.description, sp.enabled, sp.created_at, sp.updated_at,
		       COALESCE(c.component_count, 0)
		FROM status_pages sp
		LEFT JOIN (
			SELECT status_page_id, COUNT(*) AS component_count
			FROM status_page_components
			GROUP BY status_page_id
		) c ON c.status_page_id = sp.id
		ORDER BY sp.created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list status pages: %w", err)
	}
	defer rows.Close()

	pages := make([]domain.StatusPage, 0)
	for rows.Next() {
		var page domain.StatusPage
		var componentCount int
		if err := rows.Scan(
			&page.ID, &page.Slug, &page.Name, &page.Description, &page.Enabled,
			&page.CreatedAt, &page.UpdatedAt, &componentCount,
		); err != nil {
			return nil, fmt.Errorf("scan status page: %w", err)
		}
		page.Components = make([]domain.StatusPageComponent, componentCount)
		pages = append(pages, page)
	}

	return pages, rows.Err()
}

func (r *StatusPageRepository) GetByID(ctx context.Context, id string) (domain.StatusPage, error) {
	var page domain.StatusPage

	err := r.pool.QueryRow(ctx, `
		SELECT id::text, slug, name, description, enabled, created_at, updated_at
		FROM status_pages
		WHERE id = $1`, id,
	).Scan(&page.ID, &page.Slug, &page.Name, &page.Description, &page.Enabled, &page.CreatedAt, &page.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.StatusPage{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.StatusPage{}, fmt.Errorf("get status page: %w", err)
	}

	rows, err := r.pool.Query(ctx, `
		SELECT c.id::text, c.monitor_id::text, m.name, c.display_name, c.sort_order
		FROM status_page_components c
		JOIN monitors m ON m.id = c.monitor_id
		WHERE c.status_page_id = $1
		ORDER BY c.sort_order, c.id`, id)
	if err != nil {
		return domain.StatusPage{}, fmt.Errorf("list status page components: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var component domain.StatusPageComponent
		if err := rows.Scan(
			&component.ID, &component.MonitorID, &component.MonitorName,
			&component.DisplayName, &component.SortOrder,
		); err != nil {
			return domain.StatusPage{}, fmt.Errorf("scan status page component: %w", err)
		}
		page.Components = append(page.Components, component)
	}

	return page, rows.Err()
}

// PublicBySlug returns the public projection of an enabled status page:
// component display names, live status and uptime percentages over the
// 24h/7d/30d windows computed from probe_results.
func (r *StatusPageRepository) PublicBySlug(ctx context.Context, slug string) (domain.PublicStatusPage, error) {
	var page domain.PublicStatusPage
	var pageID string

	err := r.pool.QueryRow(ctx, `
		SELECT id::text, name, description
		FROM status_pages
		WHERE slug = $1 AND enabled = TRUE`, slug,
	).Scan(&pageID, &page.Name, &page.Description)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.PublicStatusPage{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.PublicStatusPage{}, fmt.Errorf("get public status page: %w", err)
	}

	rows, err := r.pool.Query(ctx, `
		SELECT
			COALESCE(NULLIF(c.display_name, ''), m.name) AS name,
			m.last_status,
			ROUND(AVG(CASE WHEN r.success THEN 100.0 ELSE 0.0 END)
				FILTER (WHERE r.started_at >= NOW() - INTERVAL '24 hours')::numeric, 2)::float8 AS uptime_24h,
			ROUND(AVG(CASE WHEN r.success THEN 100.0 ELSE 0.0 END)
				FILTER (WHERE r.started_at >= NOW() - INTERVAL '7 days')::numeric, 2)::float8 AS uptime_7d,
			ROUND(AVG(CASE WHEN r.success THEN 100.0 ELSE 0.0 END)::numeric, 2)::float8 AS uptime_30d
		FROM status_page_components c
		JOIN monitors m ON m.id = c.monitor_id
		LEFT JOIN probe_results r
			ON r.monitor_id = m.id AND r.started_at >= NOW() - INTERVAL '30 days'
		WHERE c.status_page_id = $1
		GROUP BY c.id, m.id, m.name, m.last_status, c.display_name, c.sort_order
		ORDER BY c.sort_order, c.id`, pageID)
	if err != nil {
		return domain.PublicStatusPage{}, fmt.Errorf("list public status components: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var component domain.PublicStatusComponent
		if err := rows.Scan(
			&component.Name, &component.Status,
			&component.Uptime24h, &component.Uptime7d, &component.Uptime30d,
		); err != nil {
			return domain.PublicStatusPage{}, fmt.Errorf("scan public status component: %w", err)
		}
		page.Components = append(page.Components, component)
	}
	if err := rows.Err(); err != nil {
		return domain.PublicStatusPage{}, err
	}

	page.Status = domain.ComputeOverallStatus(page.Components)
	return page, nil
}
