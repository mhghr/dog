package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"monitoring-platform/packages/shared/domain"
)

type ResourceRepository struct {
	pool *pgxpool.Pool
}

func NewResourceRepository(pool *pgxpool.Pool) *ResourceRepository {
	return &ResourceRepository{pool: pool}
}

const resourceColumns = `
	r.id::text,
	r.organization_id::text,
	r.workspace_id::text,
	r.resource_type_id::text,
	r.created_by::text,
	r.name,
	r.description,
	r.target,
	r.status,
	r.metadata,
	r.created_at,
	r.updated_at
`

func (r *ResourceRepository) Create(ctx context.Context, res *domain.Resource) error {
	metadataJSON, _ := json.Marshal(res.Metadata)
	if metadataJSON == nil {
		metadataJSON = []byte("{}")
	}

	err := r.pool.QueryRow(ctx, `
		INSERT INTO resources (organization_id, workspace_id, resource_type_id, created_by,
		                       name, description, target, status, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::jsonb)
		RETURNING `+resourceColumns,
		res.OrganizationID, res.WorkspaceID, res.ResourceTypeID, res.CreatedBy,
		res.Name, res.Description, res.Target, res.Status, metadataJSON,
	).Scan(&res.ID, &res.OrganizationID, &res.WorkspaceID, &res.ResourceTypeID, &res.CreatedBy,
		&res.Name, &res.Description, &res.Target, &res.Status, &metadataJSON,
		&res.CreatedAt, &res.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create resource: %w", err)
	}

	json.Unmarshal(metadataJSON, &res.Metadata)
	return nil
}

func (r *ResourceRepository) GetByID(ctx context.Context, id string) (domain.Resource, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT `+resourceColumns+`,
		       rt.name AS type_name, rt.category AS type_category, rt.icon AS type_icon
		FROM resources r
		JOIN resource_types rt ON rt.id = r.resource_type_id
		WHERE r.id = $1::uuid`, id)

	res, err := scanResource(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Resource{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Resource{}, err
	}
	return res, nil
}

func (r *ResourceRepository) List(ctx context.Context, filter domain.ResourceListFilter) ([]domain.Resource, int, error) {
	whereClause, args := buildResourceListFilter(filter)

	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM resources r`+whereClause, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count resources: %w", err)
	}

	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	args = append(args, pageSize, (page-1)*pageSize)
	limitArg, offsetArg := len(args)-1, len(args)

	query := fmt.Sprintf(`
		SELECT `+resourceColumns+`,
		       rt.name AS type_name, rt.category AS type_category, rt.icon AS type_icon,
		       COALESCE(mc.monitors_count, 0),
		       COALESCE(rhs.state::text, 'unknown'),
		       COALESCE(rhs.score, 0)
		FROM resources r
		JOIN resource_types rt ON rt.id = r.resource_type_id
		LEFT JOIN (
			SELECT resource_id, COUNT(*)::int AS monitors_count
			FROM monitors
			WHERE resource_id IS NOT NULL
			GROUP BY resource_id
		) mc ON mc.resource_id = r.id
		LEFT JOIN resource_health_state rhs ON rhs.resource_id = r.id
		%s
		ORDER BY r.created_at DESC
		LIMIT $%d OFFSET $%d`, whereClause, limitArg, offsetArg)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list resources: %w", err)
	}
	defer rows.Close()

	resources := make([]domain.Resource, 0)
	for rows.Next() {
		res, err := scanResourceWithStats(rows)
		if err != nil {
			return nil, 0, err
		}
		resources = append(resources, res)
	}
	return resources, total, rows.Err()
}

// buildResourceListFilter turns a resource list filter into a WHERE clause and
// its bind arguments. The clause always includes the organization predicate.
func buildResourceListFilter(filter domain.ResourceListFilter) (string, []any) {
	where := []string{"r.organization_id = $1::uuid"}
	args := []any{filter.OrganizationID}

	if filter.WorkspaceID != "" {
		args = append(args, filter.WorkspaceID)
		where = append(where, fmt.Sprintf("r.workspace_id = $%d::uuid", len(args)))
	}
	if filter.ResourceTypeID != "" {
		args = append(args, filter.ResourceTypeID)
		where = append(where, fmt.Sprintf("r.resource_type_id = $%d::uuid", len(args)))
	}
	if filter.Status != "" {
		args = append(args, filter.Status)
		where = append(where, fmt.Sprintf("r.status = $%d", len(args)))
	}
	if search := strings.TrimSpace(filter.Search); search != "" {
		args = append(args, "%"+escapeLike(search)+"%")
		where = append(where, fmt.Sprintf("(r.name ILIKE $%d OR r.description ILIKE $%d OR r.target ILIKE $%d)", len(args), len(args), len(args)))
	}
	if len(filter.Tags) > 0 {
		for key, value := range filter.Tags {
			args = append(args, key, value)
			n := len(args)
			where = append(where, fmt.Sprintf(`EXISTS (
				SELECT 1 FROM resource_tags rt
				JOIN tags t ON t.id = rt.tag_id
				WHERE rt.resource_id = r.id AND t.key = $%d AND t.value = $%d
			)`, n-1, n))
		}
	}

	return " WHERE " + strings.Join(where, " AND "), args
}

func (r *ResourceRepository) Update(ctx context.Context, res *domain.Resource) error {
	metadataJSON, _ := json.Marshal(res.Metadata)
	if metadataJSON == nil {
		metadataJSON = []byte("{}")
	}

	tag, err := r.pool.Exec(ctx, `
		UPDATE resources SET
			name=$2, description=$3, target=$4, status=$5, metadata=$6::jsonb, updated_at=NOW()
		WHERE id=$1::uuid`,
		res.ID, res.Name, res.Description, res.Target, res.Status, metadataJSON)
	if err != nil {
		return fmt.Errorf("update resource: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *ResourceRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM resources WHERE id=$1::uuid`, id)
	if err != nil {
		return fmt.Errorf("delete resource: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *ResourceRepository) ListTypes(ctx context.Context) ([]domain.ResourceType, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, name, category, slug, icon, capabilities, configuration_schema, created_at
		FROM resource_types ORDER BY category, name`)
	if err != nil {
		return nil, fmt.Errorf("list resource types: %w", err)
	}
	defer rows.Close()

	types := make([]domain.ResourceType, 0)
	for rows.Next() {
		var t domain.ResourceType
		var schema []byte
		if err := rows.Scan(&t.ID, &t.Name, &t.Category, &t.Slug, &t.Icon, &t.Capabilities, &schema, &t.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan resource type: %w", err)
		}
		t.ConfigurationSchema = schema
		types = append(types, t)
	}
	return types, rows.Err()
}

func (r *ResourceRepository) ListMonitorTypes(ctx context.Context) ([]domain.MonitorTypeDef, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, name, slug, category, execution_type, executor_key, description, icon,
		       enabled, metric_keys, configuration_schema, default_configuration, metric_schema,
		       health_parameters, supported_resource_types, created_at, updated_at
		FROM monitor_types ORDER BY category, name`)
	if err != nil {
		return nil, fmt.Errorf("list monitor types: %w", err)
	}
	defer rows.Close()

	types := make([]domain.MonitorTypeDef, 0)
	for rows.Next() {
		var t domain.MonitorTypeDef
		var enabled bool
		if err := rows.Scan(&t.ID, &t.Name, &t.Slug, &t.Category, &t.ExecutionType, &t.ExecutorKey,
			&t.Description, &t.Icon, &enabled, &t.MetricKeys,
			&t.ConfigSchema, &t.DefaultConfiguration, &t.MetricSchema,
			&t.HealthParameters, &t.SupportedResourceTypes, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan monitor type: %w", err)
		}
		t.Enabled = enabled
		types = append(types, t)
	}
	return types, rows.Err()
}

// ── Tags ──────────────────────────────────────────────────────

func (r *ResourceRepository) AttachTag(ctx context.Context, resourceID, key, value string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var tagID string
	err = tx.QueryRow(ctx, `
		INSERT INTO tags (organization_id, key, value)
		SELECT organization_id, $2, $3 FROM resources WHERE id = $1::uuid
		ON CONFLICT (organization_id, key, value) DO UPDATE SET key = EXCLUDED.key
		RETURNING id::text`, resourceID, key, value).Scan(&tagID)
	if err != nil {
		return fmt.Errorf("upsert tag: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO resource_tags (resource_id, tag_id)
		VALUES ($1::uuid, $2::uuid) ON CONFLICT DO NOTHING`, resourceID, tagID)
	if err != nil {
		return fmt.Errorf("attach tag: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *ResourceRepository) RemoveTag(ctx context.Context, resourceID, tagID string) error {
	tag, err := r.pool.Exec(ctx, `
		DELETE FROM resource_tags WHERE resource_id = $1::uuid AND tag_id = $2::uuid`,
		resourceID, tagID)
	if err != nil {
		return fmt.Errorf("remove tag: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *ResourceRepository) ListTags(ctx context.Context, resourceID string) ([]domain.Tag, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT t.id::text, t.organization_id::text, t.key, t.value
		FROM resource_tags rt
		JOIN tags t ON t.id = rt.tag_id
		WHERE rt.resource_id = $1::uuid
		ORDER BY t.key, t.value`, resourceID)
	if err != nil {
		return nil, fmt.Errorf("list tags: %w", err)
	}
	defer rows.Close()

	return scanTags(rows)
}

func (r *ResourceRepository) ListAllTags(ctx context.Context, orgID string) ([]domain.Tag, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, organization_id::text, key, value
		FROM tags WHERE organization_id = $1::uuid ORDER BY key, value`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list all tags: %w", err)
	}
	defer rows.Close()

	return scanTags(rows)
}

// ── Workspaces ────────────────────────────────────────────────

func (r *ResourceRepository) CreateWorkspace(ctx context.Context, ws *domain.Workspace) error {
	settingsJSON, _ := json.Marshal(ws.Settings)
	if settingsJSON == nil {
		settingsJSON = []byte("{}")
	}

	err := r.pool.QueryRow(ctx, `
		INSERT INTO workspaces (organization_id, name, slug, description, plan, settings)
		VALUES ($1, $2, $3, $4, $5, $6::jsonb)
		RETURNING id::text, created_at, updated_at`,
		ws.OrganizationID, ws.Name, ws.Slug, ws.Description, string(ws.Plan), settingsJSON,
	).Scan(&ws.ID, &ws.CreatedAt, &ws.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.ErrDuplicate
		}
		return fmt.Errorf("create workspace: %w", err)
	}
	return nil
}

func (r *ResourceRepository) ListWorkspaces(ctx context.Context, orgID string) ([]domain.Workspace, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, organization_id::text, name, slug, description, plan, settings, created_at, updated_at
		FROM workspaces WHERE organization_id = $1::uuid ORDER BY created_at`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list workspaces: %w", err)
	}
	defer rows.Close()

	workspaces := make([]domain.Workspace, 0)
	for rows.Next() {
		var ws domain.Workspace
		var plan string
		var settings []byte
		if err := rows.Scan(&ws.ID, &ws.OrganizationID, &ws.Name, &ws.Slug, &ws.Description,
			&plan, &settings, &ws.CreatedAt, &ws.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan workspace: %w", err)
		}
		ws.Plan = domain.WorkspacePlan(plan)
		json.Unmarshal(settings, &ws.Settings)
		workspaces = append(workspaces, ws)
	}
	return workspaces, rows.Err()
}

// ── helpers ───────────────────────────────────────────────────

func scanResource(row pgx.Row) (domain.Resource, error) {
	var (
		res          domain.Resource
		metadataJSON []byte
	)
	err := row.Scan(&res.ID, &res.OrganizationID, &res.WorkspaceID, &res.ResourceTypeID, &res.CreatedBy,
		&res.Name, &res.Description, &res.Target, &res.Status, &metadataJSON,
		&res.CreatedAt, &res.UpdatedAt,
		&res.TypeName, &res.TypeCategory, &res.TypeIcon)
	if err != nil {
		return res, err
	}
	json.Unmarshal(metadataJSON, &res.Metadata)
	return res, nil
}

func scanResourceWithStats(row pgx.Row) (domain.Resource, error) {
	var (
		res          domain.Resource
		metadataJSON []byte
		healthState  string
	)
	err := row.Scan(&res.ID, &res.OrganizationID, &res.WorkspaceID, &res.ResourceTypeID, &res.CreatedBy,
		&res.Name, &res.Description, &res.Target, &res.Status, &metadataJSON,
		&res.CreatedAt, &res.UpdatedAt,
		&res.TypeName, &res.TypeCategory, &res.TypeIcon,
		&res.MonitorsCount, &healthState, &res.HealthScore)
	if err != nil {
		return res, err
	}
	json.Unmarshal(metadataJSON, &res.Metadata)
	res.HealthStatus = healthState
	return res, nil
}

func scanTags(rows pgx.Rows) ([]domain.Tag, error) {
	tags := make([]domain.Tag, 0)
	for rows.Next() {
		var t domain.Tag
		if err := rows.Scan(&t.ID, &t.OrganizationID, &t.Key, &t.Value); err != nil {
			return nil, fmt.Errorf("scan tag: %w", err)
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}
