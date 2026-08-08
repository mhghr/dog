package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"monitoring-platform/packages/shared/domain"
)

type MonitorTypeParameterRepository struct {
	pool *pgxpool.Pool
}

func NewMonitorTypeParameterRepository(pool *pgxpool.Pool) *MonitorTypeParameterRepository {
	return &MonitorTypeParameterRepository{pool: pool}
}

func (r *MonitorTypeParameterRepository) ListByMonitorType(ctx context.Context, monitorType string) ([]domain.MonitorTypeParameter, error) {
	return []domain.MonitorTypeParameter{}, nil
}
