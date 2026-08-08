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

type SNMPRepository struct {
	pool *pgxpool.Pool
}

func NewSNMPRepository(pool *pgxpool.Pool) *SNMPRepository {
	return &SNMPRepository{pool: pool}
}

func (r *SNMPRepository) CreateCredential(ctx context.Context, cred *domain.SNMPCredential) error {
	err := r.pool.QueryRow(ctx, `
		INSERT INTO snmp_credentials (
			workspace_id, name, description, version, community, username,
			authentication_protocol, authentication_passphrase,
			privacy_protocol, privacy_passphrase, security_level, context_name,
			encrypted_config
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13::jsonb)
		RETURNING id::text, created_at, updated_at`,
		cred.WorkspaceID, cred.Name, cred.Description, cred.Version, cred.Community,
		cred.Username, cred.AuthenticationProtocol, cred.AuthenticationPassphrase,
		cred.PrivacyProtocol, cred.PrivacyPassphrase, string(cred.SecurityLevel),
		cred.ContextName, cred.EncryptedConfig,
	).Scan(&cred.ID, &cred.CreatedAt, &cred.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert snmp credential: %w", err)
	}
	return nil
}

func (r *SNMPRepository) GetCredential(ctx context.Context, id string) (domain.SNMPCredential, error) {
	return r.scanCredential(r.pool.QueryRow(ctx, `
		SELECT id::text, workspace_id::text, name, description, version, community, username,
		       authentication_protocol, authentication_passphrase,
		       privacy_protocol, privacy_passphrase, security_level, context_name,
		       encrypted_config, created_at, updated_at
		FROM snmp_credentials WHERE id = $1::uuid`, id))
}

func (r *SNMPRepository) ListCredentials(ctx context.Context, workspaceID string) ([]domain.SNMPCredential, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, workspace_id::text, name, description, version, community, username,
		       authentication_protocol, authentication_passphrase,
		       privacy_protocol, privacy_passphrase, security_level, context_name,
		       encrypted_config, created_at, updated_at
		FROM snmp_credentials WHERE workspace_id = $1::uuid ORDER BY name`, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list snmp credentials: %w", err)
	}
	defer rows.Close()

	return r.scanCredentialRows(rows)
}

func (r *SNMPRepository) UpdateCredential(ctx context.Context, cred *domain.SNMPCredential) error {
	row := r.pool.QueryRow(ctx, `
		UPDATE snmp_credentials
		SET name = $2, description = $3, version = $4, community = $5, username = $6,
		    authentication_protocol = $7, authentication_passphrase = $8,
		    privacy_protocol = $9, privacy_passphrase = $10, security_level = $11,
		    context_name = $12, encrypted_config = $13::jsonb, updated_at = NOW()
		WHERE id = $1::uuid
		RETURNING updated_at`,
		cred.ID, cred.Name, cred.Description, cred.Version, cred.Community,
		cred.Username, cred.AuthenticationProtocol, cred.AuthenticationPassphrase,
		cred.PrivacyProtocol, cred.PrivacyPassphrase, string(cred.SecurityLevel),
		cred.ContextName, cred.EncryptedConfig,
	)
	if err := row.Scan(&cred.UpdatedAt); errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	} else if err != nil {
		return fmt.Errorf("update snmp credential: %w", err)
	}
	return nil
}

func (r *SNMPRepository) DeleteCredential(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM snmp_credentials WHERE id = $1::uuid`, id)
	if err != nil {
		return fmt.Errorf("delete snmp credential: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *SNMPRepository) CreateDevice(ctx context.Context, dev *domain.SNMPDevice) error {
	oidsJSON := dev.OIDs
	if len(oidsJSON) == 0 {
		oidsJSON = json.RawMessage("[]")
	}

	err := r.pool.QueryRow(ctx, `
		INSERT INTO snmp_devices (resource_id, credential_id, transport, port, max_repetitions, timeout_seconds, retries, oids)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, $7, $8::jsonb)
		RETURNING id::text, created_at, updated_at`,
		dev.ResourceID, dev.CredentialID, dev.Transport, dev.Port,
		dev.MaxRepetitions, dev.TimeoutSeconds, dev.Retries, oidsJSON,
	).Scan(&dev.ID, &dev.CreatedAt, &dev.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.ErrDuplicate
		}
		return fmt.Errorf("insert snmp device: %w", err)
	}
	return nil
}

func (r *SNMPRepository) GetDevice(ctx context.Context, id string) (domain.SNMPDevice, error) {
	return r.scanDevice(r.pool.QueryRow(ctx, `
		SELECT id::text, resource_id::text, credential_id::text, transport, port,
		       max_repetitions, timeout_seconds, retries, oids, created_at, updated_at
		FROM snmp_devices WHERE id = $1::uuid`, id))
}

func (r *SNMPRepository) ListDevicesByResource(ctx context.Context, resourceID string) ([]domain.SNMPDevice, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, resource_id::text, credential_id::text, transport, port,
		       max_repetitions, timeout_seconds, retries, oids, created_at, updated_at
		FROM snmp_devices WHERE resource_id = $1::uuid ORDER BY created_at`, resourceID)
	if err != nil {
		return nil, fmt.Errorf("list snmp devices: %w", err)
	}
	defer rows.Close()

	var devices []domain.SNMPDevice
	for rows.Next() {
		dev, err := r.scanDeviceFromRows(rows)
		if err != nil {
			return nil, err
		}
		devices = append(devices, dev)
	}
	return devices, rows.Err()
}

func (r *SNMPRepository) UpdateDevice(ctx context.Context, dev *domain.SNMPDevice) error {
	oidsJSON := dev.OIDs
	if len(oidsJSON) == 0 {
		oidsJSON = json.RawMessage("[]")
	}

	row := r.pool.QueryRow(ctx, `
		UPDATE snmp_devices
		SET credential_id = $2::uuid, transport = $3, port = $4, max_repetitions = $5,
		    timeout_seconds = $6, retries = $7, oids = $8::jsonb, updated_at = NOW()
		WHERE id = $1::uuid
		RETURNING updated_at`,
		dev.ID, dev.CredentialID, dev.Transport, dev.Port,
		dev.MaxRepetitions, dev.TimeoutSeconds, dev.Retries, oidsJSON,
	)
	if err := row.Scan(&dev.UpdatedAt); errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	} else if err != nil {
		return fmt.Errorf("update snmp device: %w", err)
	}
	return nil
}

func (r *SNMPRepository) DeleteDevice(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM snmp_devices WHERE id = $1::uuid`, id)
	if err != nil {
		return fmt.Errorf("delete snmp device: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *SNMPRepository) scanCredential(row pgx.Row) (domain.SNMPCredential, error) {
	var c domain.SNMPCredential
	var secLevel string
	err := row.Scan(&c.ID, &c.WorkspaceID, &c.Name, &c.Description, &c.Version,
		&c.Community, &c.Username, &c.AuthenticationProtocol, &c.AuthenticationPassphrase,
		&c.PrivacyProtocol, &c.PrivacyPassphrase, &secLevel, &c.ContextName,
		&c.EncryptedConfig, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.SNMPCredential{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.SNMPCredential{}, err
	}
	c.SecurityLevel = domain.SNMPSecurityLevel(secLevel)
	return c, nil
}

func (r *SNMPRepository) scanDevice(row pgx.Row) (domain.SNMPDevice, error) {
	var d domain.SNMPDevice
	err := row.Scan(&d.ID, &d.ResourceID, &d.CredentialID, &d.Transport,
		&d.Port, &d.MaxRepetitions, &d.TimeoutSeconds, &d.Retries,
		&d.OIDs, &d.CreatedAt, &d.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.SNMPDevice{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.SNMPDevice{}, err
	}
	return d, nil
}

func (r *SNMPRepository) scanCredentialRows(rows pgx.Rows) ([]domain.SNMPCredential, error) {
	var creds []domain.SNMPCredential
	for rows.Next() {
		c, err := r.scanCredentialFromRows(rows)
		if err != nil {
			return nil, err
		}
		creds = append(creds, c)
	}
	return creds, rows.Err()
}

func (r *SNMPRepository) scanCredentialFromRows(rows pgx.Rows) (domain.SNMPCredential, error) {
	var c domain.SNMPCredential
	var secLevel string
	if err := rows.Scan(&c.ID, &c.WorkspaceID, &c.Name, &c.Description, &c.Version,
		&c.Community, &c.Username, &c.AuthenticationProtocol, &c.AuthenticationPassphrase,
		&c.PrivacyProtocol, &c.PrivacyPassphrase, &secLevel, &c.ContextName,
		&c.EncryptedConfig, &c.CreatedAt, &c.UpdatedAt); err != nil {
		return domain.SNMPCredential{}, fmt.Errorf("scan snmp credential: %w", err)
	}
	c.SecurityLevel = domain.SNMPSecurityLevel(secLevel)
	return c, nil
}

func (r *SNMPRepository) scanDeviceFromRows(rows pgx.Rows) (domain.SNMPDevice, error) {
	var d domain.SNMPDevice
	if err := rows.Scan(&d.ID, &d.ResourceID, &d.CredentialID, &d.Transport,
		&d.Port, &d.MaxRepetitions, &d.TimeoutSeconds, &d.Retries,
		&d.OIDs, &d.CreatedAt, &d.UpdatedAt); err != nil {
		return domain.SNMPDevice{}, fmt.Errorf("scan snmp device: %w", err)
	}
	return d, nil
}
