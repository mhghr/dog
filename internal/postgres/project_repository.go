package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"monitoring-platform/internal/domain"
)

type ProjectRepository struct {
	pool *pgxpool.Pool
}

func NewProjectRepository(pool *pgxpool.Pool) *ProjectRepository {
	return &ProjectRepository{pool: pool}
}

func (r *ProjectRepository) Create(ctx context.Context, project *domain.Project) error {
	err := r.pool.QueryRow(ctx, `
		INSERT INTO projects (organization_id, name)
		VALUES ($1, $2)
		RETURNING id::text, created_at, updated_at`,
		project.OrganizationID, project.Name,
	).Scan(&project.ID, &project.CreatedAt, &project.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.ErrDuplicate
		}
		return fmt.Errorf("insert project: %w", err)
	}
	return nil
}

func (r *ProjectRepository) ListByOrganization(ctx context.Context, orgID string) ([]domain.Project, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, organization_id::text, name, created_at, updated_at
		FROM projects
		WHERE organization_id = $1
		ORDER BY created_at`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	defer rows.Close()

	projects := make([]domain.Project, 0)
	for rows.Next() {
		var project domain.Project
		if err := rows.Scan(
			&project.ID, &project.OrganizationID, &project.Name,
			&project.CreatedAt, &project.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan project: %w", err)
		}
		projects = append(projects, project)
	}
	return projects, rows.Err()
}
