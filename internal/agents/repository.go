package agents

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) CreateEnrollmentToken(ctx context.Context, params CreateTokenParams) (*EnrollmentToken, error) {
	hash := sha256.Sum256([]byte(params.Token))

	var token EnrollmentToken
	err := r.pool.QueryRow(ctx, `
		INSERT INTO probe_agent_enrollment_tokens (
			token_hash, token_label, requested_location_id, expires_at, created_by
		) VALUES ($1, $2, $3, $4, $5)
		RETURNING id, token_hash, requested_location_id, expires_at, used_at, created_by, created_at
	`, hash[:], params.TokenLabel, params.LocationID, params.ExpiresAt, params.CreatedBy).Scan(
		&token.ID, &token.TokenHash, &token.RequestedLocationID,
		&token.ExpiresAt, &token.UsedAt, &token.CreatedBy, &token.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &token, nil
}

func (r *Repository) ConsumeEnrollmentToken(ctx context.Context, rawToken string) (*EnrollmentToken, error) {
	hash := sha256.Sum256([]byte(rawToken))

	var token EnrollmentToken
	err := r.pool.QueryRow(ctx, `
		UPDATE probe_agent_enrollment_tokens
		SET used_at = NOW()
		WHERE token_hash = $1
		  AND used_at IS NULL
		  AND expires_at > NOW()
		RETURNING id, token_hash, requested_location_id, expires_at, used_at, created_by, created_at
	`, hash[:]).Scan(
		&token.ID, &token.TokenHash, &token.RequestedLocationID,
		&token.ExpiresAt, &token.UsedAt, &token.CreatedBy, &token.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrInvalidToken
		}
		return nil, err
	}
	return &token, nil
}

func (r *Repository) CreateAgentWithToken(ctx context.Context, rawToken string, params CreateAgentParams) (*ProbeAgent, error) {
	hash := sha256.Sum256([]byte(rawToken))

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var requestedLocationID uuid.UUID
	var tokenID uuid.UUID
	err = tx.QueryRow(ctx, `
		UPDATE probe_agent_enrollment_tokens
		SET used_at = NOW()
		WHERE token_hash = $1
		  AND used_at IS NULL
		  AND expires_at > NOW()
		RETURNING requested_location_id, id
	`, hash[:]).Scan(&requestedLocationID, &tokenID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrInvalidToken
		}
		return nil, err
	}

	var agent ProbeAgent
	err = tx.QueryRow(ctx, `
		INSERT INTO probe_agents (
			location_id, name, hostname, machine_fingerprint, public_key,
			version, operating_system, architecture, public_ip, private_ips,
			capabilities, max_concurrency, status, agent_secret, enrollment_token_id,
			latitude, longitude, city, country
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, 'pending', $13, $14, $15, $16, $17, $18)
		RETURNING id, location_id, name, hostname, machine_fingerprint,
			public_key, COALESCE(certificate_serial, ''), COALESCE(agent_gateway_cert, ''), agent_secret,
			version, operating_system,
			architecture, COALESCE(public_ip::text, ''), private_ips, capabilities,
			max_concurrency, status, approved_by, approved_at,
			last_seen_at, revoked_at, enrollment_token_id, created_at, updated_at,
			latitude, longitude, city, country
	`,
		requestedLocationID, params.Name, params.Hostname, params.MachineFingerprint, params.PublicKey,
		params.Version, params.OperatingSystem, params.Architecture, params.PublicIP, params.PrivateIPs,
		params.Capabilities, params.MaxConcurrency, params.AgentSecret, tokenID,
		params.Latitude, params.Longitude, params.City, params.Country,
	).Scan(
		&agent.ID, &agent.LocationID, &agent.Name, &agent.Hostname, &agent.MachineFingerprint,
		&agent.PublicKey, &agent.CertificateSerial, &agent.GatewayCert, &agent.AgentSecret,
		&agent.Version, &agent.OperatingSystem,
		&agent.Architecture, &agent.PublicIP, &agent.PrivateIPs, &agent.Capabilities,
		&agent.MaxConcurrency, &agent.Status, &agent.ApprovedBy, &agent.ApprovedAt,
		&agent.LastSeenAt, &agent.RevokedAt, &agent.EnrollmentTokenID, &agent.CreatedAt, &agent.UpdatedAt,
		&agent.Latitude, &agent.Longitude, &agent.City, &agent.Country,
	)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return &agent, nil
}

func (r *Repository) CreateAgent(ctx context.Context, params CreateAgentParams) (*ProbeAgent, error) {
	var agent ProbeAgent
	err := r.pool.QueryRow(ctx, `
		INSERT INTO probe_agents (
			location_id, name, hostname, machine_fingerprint, public_key,
			version, operating_system, architecture, public_ip, private_ips,
			capabilities, max_concurrency, status, agent_secret,
			latitude, longitude, city, country
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, 'pending', $13, $14, $15, $16, $17)
		RETURNING id, location_id, name, hostname, machine_fingerprint,
			public_key, COALESCE(certificate_serial, ''), COALESCE(agent_gateway_cert, ''), agent_secret,
			version, operating_system,
			architecture, COALESCE(public_ip::text, ''), private_ips, capabilities,
			max_concurrency, status, approved_by, approved_at,
			last_seen_at, revoked_at, enrollment_token_id, created_at, updated_at,
			latitude, longitude, city, country
	`,
		params.LocationID, params.Name, params.Hostname, params.MachineFingerprint, params.PublicKey,
		params.Version, params.OperatingSystem, params.Architecture, params.PublicIP, params.PrivateIPs,
		params.Capabilities, params.MaxConcurrency, params.AgentSecret,
		params.Latitude, params.Longitude, params.City, params.Country,
	).Scan(
		&agent.ID, &agent.LocationID, &agent.Name, &agent.Hostname, &agent.MachineFingerprint,
		&agent.PublicKey, &agent.CertificateSerial, &agent.GatewayCert, &agent.AgentSecret,
		&agent.Version, &agent.OperatingSystem,
		&agent.Architecture, &agent.PublicIP, &agent.PrivateIPs, &agent.Capabilities,
		&agent.MaxConcurrency, &agent.Status, &agent.ApprovedBy, &agent.ApprovedAt,
		&agent.LastSeenAt, &agent.RevokedAt, &agent.EnrollmentTokenID, &agent.CreatedAt, &agent.UpdatedAt,
		&agent.Latitude, &agent.Longitude, &agent.City, &agent.Country,
	)
	if err != nil {
		return nil, err
	}
	return &agent, nil
}

func (r *Repository) GetAgent(ctx context.Context, id uuid.UUID) (*ProbeAgent, error) {
	var agent ProbeAgent
	err := r.pool.QueryRow(ctx, `
		SELECT id, location_id, name, hostname, machine_fingerprint,
			public_key, COALESCE(certificate_serial, ''), COALESCE(agent_gateway_cert, ''), COALESCE(agent_secret, ''),
			version, operating_system,
			architecture, COALESCE(public_ip::text, ''), private_ips, capabilities,
			max_concurrency, status, approved_by, approved_at,
			last_seen_at, revoked_at, enrollment_token_id, created_at, updated_at,
			latitude, longitude, city, country
		FROM probe_agents WHERE id = $1
	`, id).Scan(
		&agent.ID, &agent.LocationID, &agent.Name, &agent.Hostname, &agent.MachineFingerprint,
		&agent.PublicKey, &agent.CertificateSerial, &agent.GatewayCert, &agent.AgentSecret,
		&agent.Version, &agent.OperatingSystem,
		&agent.Architecture, &agent.PublicIP, &agent.PrivateIPs, &agent.Capabilities,
		&agent.MaxConcurrency, &agent.Status, &agent.ApprovedBy, &agent.ApprovedAt,
		&agent.LastSeenAt, &agent.RevokedAt, &agent.EnrollmentTokenID, &agent.CreatedAt, &agent.UpdatedAt,
		&agent.Latitude, &agent.Longitude, &agent.City, &agent.Country,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrAgentNotFound
		}
		return nil, err
	}
	return &agent, nil
}

func (r *Repository) GetAgentByIDAndSecret(ctx context.Context, id uuid.UUID, secret string) (*ProbeAgent, error) {
	var agent ProbeAgent
	err := r.pool.QueryRow(ctx, `
		SELECT id, location_id, name, hostname, machine_fingerprint,
			public_key, COALESCE(certificate_serial, ''), COALESCE(agent_gateway_cert, ''), COALESCE(agent_secret, ''),
			version, operating_system,
			architecture, COALESCE(public_ip::text, ''), private_ips, capabilities,
			max_concurrency, status, approved_by, approved_at,
			last_seen_at, revoked_at, enrollment_token_id, created_at, updated_at,
			latitude, longitude, city, country
		FROM probe_agents WHERE id = $1 AND agent_secret = $2
	`, id, secret).Scan(
		&agent.ID, &agent.LocationID, &agent.Name, &agent.Hostname, &agent.MachineFingerprint,
		&agent.PublicKey, &agent.CertificateSerial, &agent.GatewayCert, &agent.AgentSecret,
		&agent.Version, &agent.OperatingSystem,
		&agent.Architecture, &agent.PublicIP, &agent.PrivateIPs, &agent.Capabilities,
		&agent.MaxConcurrency, &agent.Status, &agent.ApprovedBy, &agent.ApprovedAt,
		&agent.LastSeenAt, &agent.RevokedAt, &agent.EnrollmentTokenID, &agent.CreatedAt, &agent.UpdatedAt,
		&agent.Latitude, &agent.Longitude, &agent.City, &agent.Country,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrAgentNotFound
		}
		return nil, err
	}
	return &agent, nil
}

func (r *Repository) ListAgents(ctx context.Context, params ListAgentsParams) ([]ProbeAgent, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, location_id, name, hostname, machine_fingerprint,
			public_key, COALESCE(certificate_serial, ''), COALESCE(agent_gateway_cert, ''), COALESCE(agent_secret, ''),
			version, operating_system,
			architecture, COALESCE(public_ip::text, ''), private_ips, capabilities,
			max_concurrency, status, approved_by, approved_at,
			last_seen_at, revoked_at, enrollment_token_id, created_at, updated_at,
			latitude, longitude, city, country
		FROM probe_agents
		WHERE ($1::probe_agent_status IS NULL OR status = $1)
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, params.Status, params.Limit, params.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var agents []ProbeAgent
	for rows.Next() {
		var a ProbeAgent
		if err := rows.Scan(
			&a.ID, &a.LocationID, &a.Name, &a.Hostname, &a.MachineFingerprint,
			&a.PublicKey, &a.CertificateSerial, &a.GatewayCert, &a.AgentSecret,
			&a.Version, &a.OperatingSystem,
			&a.Architecture, &a.PublicIP, &a.PrivateIPs, &a.Capabilities,
			&a.MaxConcurrency, &a.Status, &a.ApprovedBy, &a.ApprovedAt,
			&a.LastSeenAt, &a.RevokedAt, &a.EnrollmentTokenID, &a.CreatedAt, &a.UpdatedAt,
			&a.Latitude, &a.Longitude, &a.City, &a.Country,
		); err != nil {
			return nil, err
		}
		agents = append(agents, a)
	}
	return agents, rows.Err()
}

func (r *Repository) UpdateAgentStatus(ctx context.Context, id uuid.UUID, status AgentStatus, opts StatusUpdateOpts) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE probe_agents
		SET status = $2,
		    approved_by = COALESCE($3, approved_by),
		    approved_at = COALESCE($4, approved_at),
		    revoked_at = CASE WHEN $2 = 'revoked' THEN NOW() ELSE revoked_at END,
		    updated_at = NOW()
		WHERE id = $1
	`, id, status, opts.ApprovedBy, opts.ApprovedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrAgentNotFound
	}
	return nil
}

func (r *Repository) SetAgentCertificate(ctx context.Context, id uuid.UUID, serial string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE probe_agents
		SET certificate_serial = $2, updated_at = NOW()
		WHERE id = $1
	`, id, serial)
	return err
}

func (r *Repository) SetAgentGatewayCert(ctx context.Context, id uuid.UUID, certPEM string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE probe_agents
		SET agent_gateway_cert = $2, updated_at = NOW()
		WHERE id = $1
	`, id, certPEM)
	return err
}

func (r *Repository) AgentHeartbeat(ctx context.Context, id uuid.UUID, publicIP string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE probe_agents
		SET last_seen_at = NOW(),
		    public_ip = COALESCE($2::inet, public_ip),
		    status = CASE
		        WHEN status = 'offline' THEN 'active'
		        ELSE status
		    END,
		    updated_at = NOW()
		WHERE id = $1
	`, id, publicIP)
	return err
}

func (r *Repository) MarkOfflineAgents(ctx context.Context, maxSilence time.Duration) (int64, error) {
	tag, err := r.pool.Exec(ctx, `
		UPDATE probe_agents
		SET status = 'offline', updated_at = NOW()
		WHERE status = 'active'
		  AND last_seen_at < NOW() - $1::interval
	`, maxSilence.String())
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func (r *Repository) AuditLog(ctx context.Context, entry AuditEntry) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO probe_agent_audit_log (
			agent_id, actor_user_id, action, previous_state, next_state, remote_ip
		) VALUES ($1, $2, $3, $4, $5, $6)
	`, entry.AgentID, entry.ActorUserID, entry.Action, entry.PreviousState, entry.NextState, entry.RemoteIP)
	return err
}

type CreateTokenParams struct {
	Token      string
	TokenLabel string
	LocationID *uuid.UUID
	ExpiresAt  time.Time
	CreatedBy  uuid.UUID
}

type CreateAgentParams struct {
	LocationID         uuid.UUID
	Name               string
	Hostname           string
	MachineFingerprint string
	PublicKey          string
	Version            string
	OperatingSystem    string
	Architecture       string
	PublicIP           string
	PrivateIPs         []string
	Capabilities       []string
	MaxConcurrency     int32
	AgentSecret        string
	Latitude           *float64
	Longitude          *float64
	City               string
	Country            string
}

type ListAgentsParams struct {
	Status *AgentStatus
	Limit  int32
	Offset int32
}

func (r *Repository) UpdateCapacity(ctx context.Context, agentID uuid.UUID, runningJobs int32, spoolBytes int64) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE probe_agents
		SET running_jobs = $2,
		    spool_bytes = $3,
		    updated_at = NOW()
		WHERE id = $1
	`, agentID, runningJobs, spoolBytes)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrAgentNotFound
	}
	return nil
}

func (r *Repository) GetActiveAgentsForLocation(ctx context.Context, locationID uuid.UUID) ([]ProbeAgent, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, location_id, name, hostname, machine_fingerprint,
			public_key, COALESCE(certificate_serial, ''), COALESCE(agent_gateway_cert, ''), COALESCE(agent_secret, ''),
			version, operating_system,
			architecture, COALESCE(public_ip::text, ''), private_ips, capabilities,
			max_concurrency, COALESCE(running_jobs, 0), COALESCE(spool_bytes, 0),
			status, approved_by, approved_at,
			last_seen_at, revoked_at, enrollment_token_id, created_at, updated_at,
			latitude, longitude, city, country
		FROM probe_agents
		WHERE location_id = $1
		  AND status IN ('active', 'draining', 'updating')
		ORDER BY running_jobs ASC
	`, locationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var agents []ProbeAgent
	for rows.Next() {
		var a ProbeAgent
		if err := rows.Scan(
			&a.ID, &a.LocationID, &a.Name, &a.Hostname, &a.MachineFingerprint,
			&a.PublicKey, &a.CertificateSerial, &a.GatewayCert, &a.AgentSecret,
			&a.Version, &a.OperatingSystem,
			&a.Architecture, &a.PublicIP, &a.PrivateIPs, &a.Capabilities,
			&a.MaxConcurrency, &a.RunningJobs, &a.SpoolBytes,
			&a.Status, &a.ApprovedBy, &a.ApprovedAt,
			&a.LastSeenAt, &a.RevokedAt, &a.EnrollmentTokenID, &a.CreatedAt, &a.UpdatedAt,
			&a.Latitude, &a.Longitude, &a.City, &a.Country,
		); err != nil {
			return nil, err
		}
		agents = append(agents, a)
	}
	return agents, rows.Err()
}

type StatusUpdateOpts struct {
	ApprovedBy *uuid.UUID
	ApprovedAt *time.Time
}

type UnusedTokenInfo struct {
	ID                  uuid.UUID
	TokenLabel          string
	RequestedLocationID *uuid.UUID
	ExpiresAt           time.Time
	CreatedAt           time.Time
}

func (r *Repository) ListUnusedTokens(ctx context.Context) ([]UnusedTokenInfo, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, token_label, requested_location_id, expires_at, created_at
		FROM probe_agent_enrollment_tokens
		WHERE used_at IS NULL AND expires_at > NOW()
		ORDER BY created_at DESC
		LIMIT 100
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []UnusedTokenInfo
	for rows.Next() {
		var t UnusedTokenInfo
		if err := rows.Scan(&t.ID, &t.TokenLabel, &t.RequestedLocationID, &t.ExpiresAt, &t.CreatedAt); err != nil {
			return nil, err
		}
		tokens = append(tokens, t)
	}
	return tokens, rows.Err()
}
