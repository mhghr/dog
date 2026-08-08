package repository

import (
	"context"

	"monitoring-platform/packages/shared/domain"
)

type PluginRepository interface {
	List(ctx context.Context) ([]domain.Plugin, error)
	ListByType(ctx context.Context, pluginType domain.PluginType) ([]domain.Plugin, error)
	ListByCategory(ctx context.Context, category string) ([]domain.Plugin, error)
	GetBySlug(ctx context.Context, slug string) (domain.Plugin, error)
	Create(ctx context.Context, plugin *domain.Plugin) error
	Update(ctx context.Context, plugin *domain.Plugin) error
	SetEnabled(ctx context.Context, slug string, enabled bool) error
}
