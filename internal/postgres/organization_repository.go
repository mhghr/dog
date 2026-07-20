package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"monitoring-platform/internal/domain"
)

type OrganizationRepository struct {
	pool *pgxpool.Pool
}

func NewOrganizationRepository(pool *pgxpool.Pool) *OrganizationRepository {
	return &OrganizationRepository{pool: pool}
}

func (r *OrganizationRepository) Create(ctx context.Context, org *domain.Organization) error {
	err := r.pool.QueryRow(ctx, `
		INSERT INTO organizations (name, slug)
		VALUES ($1, $2)
		RETURNING id::text, created_at, updated_at`,
		org.Name, org.Slug,
	).Scan(&org.ID, &org.CreatedAt, &org.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.ErrDuplicate
		}
		return fmt.Errorf("insert organization: %w", err)
	}
	return nil
}

func (r *OrganizationRepository) GetBySlug(ctx context.Context, slug string) (domain.Organization, error) {
	var org domain.Organization
	err := r.pool.QueryRow(ctx, `
		SELECT id::text, name, slug, created_at, updated_at
		FROM organizations
		WHERE slug = $1`, slug,
	).Scan(&org.ID, &org.Name, &org.Slug, &org.CreatedAt, &org.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Organization{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Organization{}, fmt.Errorf("get organization by slug: %w", err)
	}
	return org, nil
}
