package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"monitoring-platform/internal/domain"
)

type RefreshTokenRepository struct {
	pool *pgxpool.Pool
}

func NewRefreshTokenRepository(pool *pgxpool.Pool) *RefreshTokenRepository {
	return &RefreshTokenRepository{pool: pool}
}

func (r *RefreshTokenRepository) Create(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO auth_refresh_tokens (user_id, token_hash, expires_at)
		VALUES ($1::uuid, $2, $3)`,
		userID, tokenHash, expiresAt)
	if err != nil {
		return fmt.Errorf("create refresh token: %w", err)
	}

	return nil
}

func (r *RefreshTokenRepository) Rotate(ctx context.Context, oldHash, newHash string, expiresAt time.Time) (string, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("begin token rotation: %w", err)
	}
	defer tx.Rollback(ctx)

	var (
		tokenID        string
		userID         string
		tokenExpiresAt time.Time
		revokedAt      *time.Time
	)

	err = tx.QueryRow(ctx, `
		SELECT id::text, user_id::text, expires_at, revoked_at
		FROM auth_refresh_tokens
		WHERE token_hash = $1
		FOR UPDATE`, oldHash,
	).Scan(&tokenID, &userID, &tokenExpiresAt, &revokedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return "", domain.ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("load refresh token: %w", err)
	}

	// Replay of an already-rotated token: assume theft and revoke the
	// whole session family for this user.
	if revokedAt != nil {
		if _, err := tx.Exec(ctx, `
			UPDATE auth_refresh_tokens
			SET revoked_at = NOW()
			WHERE user_id = $1::uuid AND revoked_at IS NULL`, userID); err != nil {
			return "", fmt.Errorf("revoke token family: %w", err)
		}
		if err := tx.Commit(ctx); err != nil {
			return "", err
		}
		return "", domain.ErrTokenReused
	}

	if time.Now().After(tokenExpiresAt) {
		return "", domain.ErrTokenExpired
	}

	var newID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO auth_refresh_tokens (user_id, token_hash, expires_at)
		VALUES ($1::uuid, $2, $3)
		RETURNING id::text`,
		userID, newHash, expiresAt,
	).Scan(&newID); err != nil {
		return "", fmt.Errorf("insert rotated token: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE auth_refresh_tokens
		SET revoked_at = NOW(), replaced_by = $2::uuid
		WHERE id = $1::uuid`,
		tokenID, newID); err != nil {
		return "", fmt.Errorf("revoke old token: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("commit token rotation: %w", err)
	}

	return userID, nil
}

func (r *RefreshTokenRepository) Revoke(ctx context.Context, tokenHash string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE auth_refresh_tokens
		SET revoked_at = NOW()
		WHERE token_hash = $1 AND revoked_at IS NULL`, tokenHash)
	if err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}

	return nil
}

type OTPRepository struct {
	pool *pgxpool.Pool
}

func NewOTPRepository(pool *pgxpool.Pool) *OTPRepository {
	return &OTPRepository{pool: pool}
}

func (r *OTPRepository) Create(ctx context.Context, phone, codeHash string, expiresAt time.Time) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin otp create: %w", err)
	}
	defer tx.Rollback(ctx)

	// A new code invalidates all previous active codes for the phone.
	if _, err := tx.Exec(ctx, `
		UPDATE auth_otp_codes
		SET consumed_at = NOW()
		WHERE phone = $1 AND consumed_at IS NULL`, phone); err != nil {
		return fmt.Errorf("invalidate previous otp codes: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO auth_otp_codes (phone, code_hash, expires_at)
		VALUES ($1, $2, $3)`,
		phone, codeHash, expiresAt); err != nil {
		return fmt.Errorf("insert otp code: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *OTPRepository) CountSince(ctx context.Context, phone string, since time.Time) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM auth_otp_codes
		WHERE phone = $1 AND created_at > $2`,
		phone, since,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count otp codes: %w", err)
	}

	return count, nil
}

func (r *OTPRepository) LatestActive(ctx context.Context, phone string) (domain.OTPCode, error) {
	var code domain.OTPCode

	err := r.pool.QueryRow(ctx, `
		SELECT id::text, phone, code_hash, expires_at, attempts, created_at
		FROM auth_otp_codes
		WHERE phone = $1
		  AND consumed_at IS NULL
		  AND expires_at > NOW()
		ORDER BY created_at DESC
		LIMIT 1`, phone,
	).Scan(&code.ID, &code.Phone, &code.CodeHash, &code.ExpiresAt, &code.Attempts, &code.CreatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return domain.OTPCode{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.OTPCode{}, fmt.Errorf("load otp code: %w", err)
	}

	return code, nil
}

func (r *OTPRepository) RegisterFailure(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE auth_otp_codes
		SET attempts = attempts + 1
		WHERE id = $1::uuid`, id)
	if err != nil {
		return fmt.Errorf("register otp failure: %w", err)
	}

	return nil
}

func (r *OTPRepository) Consume(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE auth_otp_codes
		SET consumed_at = NOW()
		WHERE id = $1::uuid`, id)
	if err != nil {
		return fmt.Errorf("consume otp code: %w", err)
	}

	return nil
}
