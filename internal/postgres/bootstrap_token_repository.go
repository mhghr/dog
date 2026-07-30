package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"monitoring-platform/internal/domain"
)

type BootstrapTokenRepository struct {
	pool *pgxpool.Pool
}

func NewBootstrapTokenRepository(pool *pgxpool.Pool) *BootstrapTokenRepository {
	return &BootstrapTokenRepository{pool: pool}
}

func (r *BootstrapTokenRepository) Create(ctx context.Context, token *domain.BootstrapToken) error {
	query := `INSERT INTO monitoring_bootstrap_tokens (id, tenant_id, token_hash, description, expires_at, created_by)
	           VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := r.pool.Exec(ctx, query, token.ID, token.TenantID, token.TokenHash, token.Description, token.ExpiresAt, token.CreatedBy)
	if err != nil {
		return fmt.Errorf("insert bootstrap token: %w", err)
	}
	return nil
}

func (r *BootstrapTokenRepository) GetByTokenHash(ctx context.Context, hash string) (domain.BootstrapToken, error) {
	query := `SELECT id::text, tenant_id::text, token_hash, description, expires_at, used_at, revoked_at, created_by::text, created_at
	           FROM monitoring_bootstrap_tokens WHERE token_hash = $1`
	row := r.pool.QueryRow(ctx, query, hash)

	var t domain.BootstrapToken
	var usedAt, revokedAt *time.Time
	var createdBy *string

	err := row.Scan(&t.ID, &t.TenantID, &t.TokenHash, &t.Description, &t.ExpiresAt, &usedAt, &revokedAt, &createdBy, &t.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return t, domain.ErrNotFound
		}
		return t, fmt.Errorf("get bootstrap token: %w", err)
	}

	t.UsedAt = usedAt
	t.RevokedAt = revokedAt
	t.CreatedBy = createdBy
	return t, nil
}

func (r *BootstrapTokenRepository) MarkUsed(ctx context.Context, tokenID string) error {
	now := time.Now()
	query := `UPDATE monitoring_bootstrap_tokens SET used_at = $1 WHERE id = $2 AND used_at IS NULL`
	tag, err := r.pool.Exec(ctx, query, now, tokenID)
	if err != nil {
		return fmt.Errorf("mark token used: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("token already used or not found")
	}
	return nil
}

func (r *BootstrapTokenRepository) MarkRevoked(ctx context.Context, tokenID string) error {
	now := time.Now()
	query := `UPDATE monitoring_bootstrap_tokens SET revoked_at = $1 WHERE id = $2 AND revoked_at IS NULL`
	tag, err := r.pool.Exec(ctx, query, now, tokenID)
	if err != nil {
		return fmt.Errorf("mark token revoked: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("token already revoked or not found")
	}
	return nil
}

func (r *BootstrapTokenRepository) DeleteExpired(ctx context.Context) (int64, error) {
	query := `DELETE FROM monitoring_bootstrap_tokens WHERE expires_at < NOW() AND used_at IS NULL`
	tag, err := r.pool.Exec(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("delete expired tokens: %w", err)
	}
	return tag.RowsAffected(), nil
}

func (r *BootstrapTokenRepository) ListByTenant(ctx context.Context, tenantID string, limit, offset int) ([]domain.BootstrapToken, int, error) {
	countQuery := `SELECT COUNT(*) FROM monitoring_bootstrap_tokens WHERE tenant_id = $1`
	var total int
	err := r.pool.QueryRow(ctx, countQuery, tenantID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count bootstrap tokens: %w", err)
	}

	query := `SELECT id::text, tenant_id::text, token_hash, description, expires_at, used_at, revoked_at, created_by::text, created_at
	           FROM monitoring_bootstrap_tokens WHERE tenant_id = $1
	           ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	rows, err := r.pool.Query(ctx, query, tenantID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list bootstrap tokens: %w", err)
	}
	defer rows.Close()

	var tokens []domain.BootstrapToken
	for rows.Next() {
		var t domain.BootstrapToken
		var usedAt, revokedAt *time.Time
		var createdBy *string
		if err := rows.Scan(&t.ID, &t.TenantID, &t.TokenHash, &t.Description, &t.ExpiresAt, &usedAt, &revokedAt, &createdBy, &t.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan bootstrap token: %w", err)
		}
		t.UsedAt = usedAt
		t.RevokedAt = revokedAt
		t.CreatedBy = createdBy
		tokens = append(tokens, t)
	}

	if tokens == nil {
		tokens = []domain.BootstrapToken{}
	}

	return tokens, total, nil
}
