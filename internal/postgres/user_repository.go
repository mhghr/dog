package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"monitoring-platform/internal/domain"
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

const userColumns = `
	id::text,
	COALESCE(google_id, ''),
	COALESCE(email, ''),
	COALESCE(phone, ''),
	name,
	avatar_url,
	COALESCE(organization_id::text, ''),
	created_at,
	updated_at,
	last_login_at
`

func scanUser(row rowScanner) (domain.User, error) {
	var user domain.User

	if err := row.Scan(
		&user.ID, &user.GoogleID, &user.Email, &user.Phone,
		&user.Name, &user.AvatarURL, &user.OrganizationID,
		&user.CreatedAt, &user.UpdatedAt, &user.LastLoginAt,
	); err != nil {
		return domain.User{}, err
	}

	return user, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (domain.User, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+userColumns+` FROM users WHERE id = $1::uuid`, id)

	user, err := scanUser(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.User{}, fmt.Errorf("get user: %w", err)
	}

	return user, nil
}

func (r *UserRepository) UpsertGoogleUser(ctx context.Context, googleID, email, name, avatarURL string) (domain.User, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.User{}, fmt.Errorf("begin upsert google user: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Existing Google identity.
	row := tx.QueryRow(ctx, `
		UPDATE users
		SET
			email = COALESCE(NULLIF($2, ''), email),
			name = CASE WHEN $3 <> '' THEN $3 ELSE name END,
			avatar_url = CASE WHEN $4 <> '' THEN $4 ELSE avatar_url END,
			last_login_at = NOW(),
			updated_at = NOW()
		WHERE google_id = $1
		RETURNING `+userColumns,
		googleID, email, name, avatarURL)

	user, err := scanUser(row)
	if err == nil {
		return user, tx.Commit(ctx)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, fmt.Errorf("update google user: %w", err)
	}

	// 2. Link Google identity to an existing email account.
	if email != "" {
		row = tx.QueryRow(ctx, `
			UPDATE users
			SET
				google_id = $1,
				name = CASE WHEN $3 <> '' THEN $3 ELSE name END,
				avatar_url = CASE WHEN $4 <> '' THEN $4 ELSE avatar_url END,
				last_login_at = NOW(),
				updated_at = NOW()
			WHERE email = $2 AND google_id IS NULL
			RETURNING `+userColumns,
			googleID, email, name, avatarURL)

		user, err = scanUser(row)
		if err == nil {
			return user, tx.Commit(ctx)
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, fmt.Errorf("link google user: %w", err)
		}
	}

	// 3. First login: provision the account.
	row = tx.QueryRow(ctx, `
		INSERT INTO users (google_id, email, name, avatar_url, last_login_at)
		VALUES ($1, NULLIF($2, ''), $3, $4, NOW())
		RETURNING `+userColumns,
		googleID, email, name, avatarURL)

	user, err = scanUser(row)
	if err != nil {
		return domain.User{}, fmt.Errorf("insert google user: %w", err)
	}

	return user, tx.Commit(ctx)
}

func (r *UserRepository) UpsertPhoneUser(ctx context.Context, phone string) (domain.User, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO users (phone, last_login_at)
		VALUES ($1, NOW())
		ON CONFLICT (phone) WHERE phone IS NOT NULL
		DO UPDATE SET last_login_at = NOW(), updated_at = NOW()
		RETURNING `+userColumns,
		phone)

	user, err := scanUser(row)
	if err != nil {
		return domain.User{}, fmt.Errorf("upsert phone user: %w", err)
	}

	return user, nil
}
