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

type LocationRepository struct {
	pool *pgxpool.Pool
}

func NewLocationRepository(pool *pgxpool.Pool) *LocationRepository {
	return &LocationRepository{pool: pool}
}

func (r *LocationRepository) List(ctx context.Context) ([]domain.ProbeLocation, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, name, code, enabled, created_at
		FROM probe_locations
		ORDER BY created_at`)
	if err != nil {
		return nil, fmt.Errorf("list probe locations: %w", err)
	}
	defer rows.Close()

	locations := make([]domain.ProbeLocation, 0)
	for rows.Next() {
		var location domain.ProbeLocation
		if err := rows.Scan(&location.ID, &location.Name, &location.Code, &location.Enabled, &location.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan probe location: %w", err)
		}
		locations = append(locations, location)
	}

	return locations, rows.Err()
}

func (r *LocationRepository) GetByCode(ctx context.Context, code string) (domain.ProbeLocation, error) {
	var location domain.ProbeLocation

	err := r.pool.QueryRow(ctx, `
		SELECT id::text, name, code, enabled, created_at
		FROM probe_locations
		WHERE code = $1`, code,
	).Scan(&location.ID, &location.Name, &location.Code, &location.Enabled, &location.CreatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ProbeLocation{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.ProbeLocation{}, fmt.Errorf("get probe location: %w", err)
	}

	return location, nil
}

func (r *LocationRepository) Create(ctx context.Context, location *domain.ProbeLocation) error {
	err := r.pool.QueryRow(ctx, `
		INSERT INTO probe_locations (name, code, enabled)
		VALUES ($1, $2, $3)
		RETURNING id::text, created_at`,
		location.Name, location.Code, location.Enabled,
	).Scan(&location.ID, &location.CreatedAt)
	if err != nil {
		var pgError *pgconn.PgError
		if errors.As(err, &pgError) && pgError.Code == "23505" {
			return domain.ErrDuplicate
		}
		return fmt.Errorf("insert probe location: %w", err)
	}
	return nil
}
