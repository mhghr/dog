package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"monitoring-platform/packages/shared/domain"
)

type PluginRepository struct {
	pool *pgxpool.Pool
}

func NewPluginRepository(pool *pgxpool.Pool) *PluginRepository {
	return &PluginRepository{pool: pool}
}

func (r *PluginRepository) List(ctx context.Context) ([]domain.Plugin, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, name, slug, description, type::text, version, icon, category,
		       configuration_schema, enabled, created_at, updated_at
		FROM plugins ORDER BY category, name`)
	if err != nil {
		return nil, fmt.Errorf("list plugins: %w", err)
	}
	defer rows.Close()

	return scanPlugins(rows)
}

func (r *PluginRepository) ListByType(ctx context.Context, pluginType domain.PluginType) ([]domain.Plugin, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, name, slug, description, type::text, version, icon, category,
		       configuration_schema, enabled, created_at, updated_at
		FROM plugins WHERE type = $1::plugin_type ORDER BY name`, string(pluginType))
	if err != nil {
		return nil, fmt.Errorf("list plugins by type: %w", err)
	}
	defer rows.Close()

	return scanPlugins(rows)
}

func (r *PluginRepository) ListByCategory(ctx context.Context, category string) ([]domain.Plugin, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, name, slug, description, type::text, version, icon, category,
		       configuration_schema, enabled, created_at, updated_at
		FROM plugins WHERE category = $1 ORDER BY name`, category)
	if err != nil {
		return nil, fmt.Errorf("list plugins by category: %w", err)
	}
	defer rows.Close()

	return scanPlugins(rows)
}

func (r *PluginRepository) GetBySlug(ctx context.Context, slug string) (domain.Plugin, error) {
	return r.scanPlugin(r.pool.QueryRow(ctx, `
		SELECT id::text, name, slug, description, type::text, version, icon, category,
		       configuration_schema, enabled, created_at, updated_at
		FROM plugins WHERE slug = $1`, slug))
}

func (r *PluginRepository) Create(ctx context.Context, plugin *domain.Plugin) error {
	configJSON := plugin.ConfigurationSchema
	if len(configJSON) == 0 {
		configJSON = json.RawMessage("{}")
	}

	err := r.pool.QueryRow(ctx, `
		INSERT INTO plugins (name, slug, description, type, version, icon, category, configuration_schema, enabled)
		VALUES ($1, $2, $3, $4::plugin_type, $5, $6, $7, $8::jsonb, $9)
		RETURNING id::text, created_at, updated_at`,
		plugin.Name, plugin.Slug, plugin.Description, string(plugin.Type),
		plugin.Version, plugin.Icon, plugin.Category, configJSON, plugin.Enabled,
	).Scan(&plugin.ID, &plugin.CreatedAt, &plugin.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.ErrDuplicate
		}
		return fmt.Errorf("insert plugin: %w", err)
	}
	return nil
}

func (r *PluginRepository) Update(ctx context.Context, plugin *domain.Plugin) error {
	configJSON := plugin.ConfigurationSchema
	if len(configJSON) == 0 {
		configJSON = json.RawMessage("{}")
	}

	row := r.pool.QueryRow(ctx, `
		UPDATE plugins
		SET name = $2, description = $3, type = $4::plugin_type, version = $5,
		    icon = $6, category = $7, configuration_schema = $8::jsonb, enabled = $9, updated_at = NOW()
		WHERE id = $1::uuid
		RETURNING updated_at`,
		plugin.ID, plugin.Name, plugin.Description, string(plugin.Type),
		plugin.Version, plugin.Icon, plugin.Category, configJSON, plugin.Enabled,
	)
	if err := row.Scan(&plugin.UpdatedAt); errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	} else if err != nil {
		return fmt.Errorf("update plugin: %w", err)
	}
	return nil
}

func (r *PluginRepository) SetEnabled(ctx context.Context, slug string, enabled bool) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE plugins SET enabled = $2, updated_at = NOW() WHERE slug = $1`, slug, enabled)
	if err != nil {
		return fmt.Errorf("set plugin enabled: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *PluginRepository) scanPlugin(row pgx.Row) (domain.Plugin, error) {
	var p domain.Plugin
	var pluginType string
	err := row.Scan(&p.ID, &p.Name, &p.Slug, &p.Description, &pluginType, &p.Version,
		&p.Icon, &p.Category, &p.ConfigurationSchema, &p.Enabled, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Plugin{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Plugin{}, err
	}
	p.Type = domain.PluginType(pluginType)
	return p, nil
}

func scanPlugins(rows pgx.Rows) ([]domain.Plugin, error) {
	var plugins []domain.Plugin
	for rows.Next() {
		var p domain.Plugin
		var pluginType string
		if err := rows.Scan(&p.ID, &p.Name, &p.Slug, &p.Description, &pluginType, &p.Version,
			&p.Icon, &p.Category, &p.ConfigurationSchema, &p.Enabled, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan plugin: %w", err)
		}
		p.Type = domain.PluginType(pluginType)
		plugins = append(plugins, p)
	}
	return plugins, rows.Err()
}
