package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"monitoring-platform/packages/shared/domain"
	"monitoring-platform/packages/shared/repository"
)

// On-demand tasks ───────────────────────────────────────────────────────────

func (r *SNMPRepository) CreateTask(ctx context.Context, task *domain.SNMPTask) error {
	configJSON, err := json.Marshal(task.Config)
	if err != nil {
		return fmt.Errorf("marshal snmp task config: %w", err)
	}

	err = r.pool.QueryRow(ctx, `
		INSERT INTO snmp_tasks (workspace_id, resource_id, monitor_id, kind, config, status, created_at)
		VALUES ($1, $2::uuid, $3, $4, $5::jsonb, $6, $7)
		RETURNING id::text`,
		nullableString(task.WorkspaceID), task.ResourceID, nullableString(task.MonitorID),
		string(task.Kind), configJSON, string(domain.SNMPTaskPending), task.CreatedAt,
	).Scan(&task.TaskID)
	if err != nil {
		return fmt.Errorf("create snmp task: %w", err)
	}
	return nil
}

func (r *SNMPRepository) GetTask(ctx context.Context, taskID string) (domain.SNMPTask, error) {
	var task domain.SNMPTask
	var configJSON, resultJSON []byte

	err := r.pool.QueryRow(ctx, `
		SELECT id::text, COALESCE(workspace_id::text, ''), resource_id::text,
		       COALESCE(monitor_id::text, ''), kind, config, status, result, COALESCE(error, ''),
		       created_at, finished_at
		FROM snmp_tasks WHERE id = $1::uuid`, taskID).
		Scan(&task.TaskID, &task.WorkspaceID, &task.ResourceID, &task.MonitorID,
			&task.Kind, &configJSON, &task.Status, &resultJSON, &task.Error, &task.CreatedAt, &task.FinishedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.SNMPTask{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.SNMPTask{}, fmt.Errorf("get snmp task: %w", err)
	}
	if len(configJSON) > 0 {
		_ = json.Unmarshal(configJSON, &task.Config)
	}
	if len(resultJSON) > 0 {
		task.Result = resultJSON
	}
	return task, nil
}

func (r *SNMPRepository) SetTaskRunning(ctx context.Context, taskID string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE snmp_tasks SET status = 'running' WHERE id = $1::uuid`, taskID)
	if err != nil {
		return fmt.Errorf("mark snmp task running: %w", err)
	}
	return nil
}

func (r *SNMPRepository) FinishTask(ctx context.Context, taskID string, status domain.SNMPTaskStatus, result json.RawMessage, errorMsg string) error {
	var resultJSON any
	if len(result) > 0 {
		resultJSON = result
	}
	_, err := r.pool.Exec(ctx, `
		UPDATE snmp_tasks SET status = $2, result = $3, error = $4, finished_at = NOW()
		WHERE id = $1::uuid`,
		taskID, string(status), resultJSON, nullableString(errorMsg))
	if err != nil {
		return fmt.Errorf("finish snmp task: %w", err)
	}
	return nil
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

// Discovery cache ───────────────────────────────────────────────────────────

func (r *SNMPRepository) UpsertDiscovery(ctx context.Context, monitorID string, discovery *domain.SNMPDiscoveryResult) error {
	deviceJSON, err := json.Marshal(discovery.Device)
	if err != nil {
		return fmt.Errorf("marshal snmp device: %w", err)
	}
	interfacesJSON, err := json.Marshal(discovery.Interfaces)
	if err != nil {
		return fmt.Errorf("marshal snmp interfaces: %w", err)
	}
	sensorsJSON, err := json.Marshal(discovery.Sensors)
	if err != nil {
		return fmt.Errorf("marshal snmp sensors: %w", err)
	}

	_, err = r.pool.Exec(ctx, `
		INSERT INTO snmp_discovery (monitor_id, device, interfaces, sensors, discovered_at)
		VALUES ($1::uuid, $2::jsonb, $3::jsonb, $4::jsonb, $5)
		ON CONFLICT (monitor_id)
		DO UPDATE SET device = EXCLUDED.device, interfaces = EXCLUDED.interfaces,
		              sensors = EXCLUDED.sensors, discovered_at = EXCLUDED.discovered_at`,
		monitorID, deviceJSON, interfacesJSON, sensorsJSON, discovery.DiscoveredAt)
	if err != nil {
		return fmt.Errorf("upsert snmp discovery: %w", err)
	}
	return nil
}

func (r *SNMPRepository) GetDiscovery(ctx context.Context, monitorID string) (domain.SNMPDiscoveryResult, error) {
	var d domain.SNMPDiscoveryResult
	var deviceJSON, interfacesJSON, sensorsJSON []byte

	err := r.pool.QueryRow(ctx, `
		SELECT device, interfaces, sensors, discovered_at
		FROM snmp_discovery WHERE monitor_id = $1::uuid`, monitorID).
		Scan(&deviceJSON, &interfacesJSON, &sensorsJSON, &d.DiscoveredAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.SNMPDiscoveryResult{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.SNMPDiscoveryResult{}, fmt.Errorf("get snmp discovery: %w", err)
	}

	if err := json.Unmarshal(deviceJSON, &d.Device); err != nil {
		return domain.SNMPDiscoveryResult{}, fmt.Errorf("unmarshal snmp device: %w", err)
	}
	if err := json.Unmarshal(interfacesJSON, &d.Interfaces); err != nil {
		return domain.SNMPDiscoveryResult{}, fmt.Errorf("unmarshal snmp interfaces: %w", err)
	}
	if err := json.Unmarshal(sensorsJSON, &d.Sensors); err != nil {
		return domain.SNMPDiscoveryResult{}, fmt.Errorf("unmarshal snmp sensors: %w", err)
	}
	return d, nil
}

// Interface policy + last state ─────────────────────────────────────────────

func (r *SNMPRepository) UpsertInterface(ctx context.Context, row *domain.SNMPInterfaceRow) error {
	err := r.pool.QueryRow(ctx, `
		INSERT INTO snmp_interfaces (
			monitor_id, if_index, if_name, if_descr, if_alias, display_name,
			ignore, monitor, utilization_warning, utilization_critical,
			oper_down_critical, last_oper_status, last_in_bps, last_out_bps,
			last_utilization_percent, last_check_at
		) VALUES ($1::uuid, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		ON CONFLICT (monitor_id, if_index)
		DO UPDATE SET if_name = EXCLUDED.if_name, if_descr = EXCLUDED.if_descr,
		              if_alias = EXCLUDED.if_alias, display_name = EXCLUDED.display_name,
		              ignore = EXCLUDED.ignore, monitor = EXCLUDED.monitor,
		              utilization_warning = EXCLUDED.utilization_warning,
		              utilization_critical = EXCLUDED.utilization_critical,
		              oper_down_critical = EXCLUDED.oper_down_critical,
		              last_oper_status = EXCLUDED.last_oper_status,
		              last_in_bps = EXCLUDED.last_in_bps, last_out_bps = EXCLUDED.last_out_bps,
		              last_utilization_percent = EXCLUDED.last_utilization_percent,
		              last_check_at = EXCLUDED.last_check_at, updated_at = NOW()
		RETURNING id::text, created_at, updated_at`,
		row.MonitorID, row.IfIndex, row.IfName, row.IfDescr, row.IfAlias, row.DisplayName,
		row.Ignore, row.Monitor, row.UtilizationWarning, row.UtilizationCritical,
		row.OperDownCritical, row.LastOperStatus, row.LastInBps, row.LastOutBps,
		row.LastUtilizationPercent, row.LastCheckAt,
	).Scan(&row.ID, &row.CreatedAt, &row.UpdatedAt)
	if err != nil {
		return fmt.Errorf("upsert snmp interface: %w", err)
	}
	return nil
}

func (r *SNMPRepository) BulkUpsertInterfaces(ctx context.Context, monitorID string, rows []domain.SNMPInterfaceRow) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin interface upsert: %w", err)
	}
	defer tx.Rollback(ctx)

	for i := range rows {
		rows[i].MonitorID = monitorID
		if err := upsertInterfaceTx(ctx, tx, &rows[i]); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func upsertInterfaceTx(ctx context.Context, tx pgx.Tx, row *domain.SNMPInterfaceRow) error {
	err := tx.QueryRow(ctx, `
		INSERT INTO snmp_interfaces (
			monitor_id, if_index, if_name, if_descr, if_alias, display_name,
			ignore, monitor, utilization_warning, utilization_critical,
			oper_down_critical, last_oper_status, last_in_bps, last_out_bps,
			last_utilization_percent, last_check_at
		) VALUES ($1::uuid, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		ON CONFLICT (monitor_id, if_index)
		DO UPDATE SET if_name = EXCLUDED.if_name, if_descr = EXCLUDED.if_descr,
		              if_alias = EXCLUDED.if_alias, display_name = EXCLUDED.display_name,
		              ignore = EXCLUDED.ignore, monitor = EXCLUDED.monitor,
		              utilization_warning = EXCLUDED.utilization_warning,
		              utilization_critical = EXCLUDED.utilization_critical,
		              oper_down_critical = EXCLUDED.oper_down_critical,
		              last_oper_status = EXCLUDED.last_oper_status,
		              last_in_bps = EXCLUDED.last_in_bps, last_out_bps = EXCLUDED.last_out_bps,
		              last_utilization_percent = EXCLUDED.last_utilization_percent,
		              last_check_at = EXCLUDED.last_check_at, updated_at = NOW()
		RETURNING id::text, created_at, updated_at`,
		row.MonitorID, row.IfIndex, row.IfName, row.IfDescr, row.IfAlias, row.DisplayName,
		row.Ignore, row.Monitor, row.UtilizationWarning, row.UtilizationCritical,
		row.OperDownCritical, row.LastOperStatus, row.LastInBps, row.LastOutBps,
		row.LastUtilizationPercent, row.LastCheckAt,
	).Scan(&row.ID, &row.CreatedAt, &row.UpdatedAt)
	if err != nil {
		return fmt.Errorf("upsert snmp interface: %w", err)
	}
	return nil
}

func (r *SNMPRepository) ListInterfaces(ctx context.Context, monitorID string) ([]domain.SNMPInterfaceRow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, monitor_id::text, if_index, if_name, if_descr, if_alias,
		       display_name, ignore, monitor, utilization_warning, utilization_critical,
		       oper_down_critical, last_oper_status, last_in_bps, last_out_bps,
		       last_utilization_percent, last_check_at, created_at, updated_at
		FROM snmp_interfaces WHERE monitor_id = $1::uuid ORDER BY if_index`, monitorID)
	if err != nil {
		return nil, fmt.Errorf("list snmp interfaces: %w", err)
	}
	defer rows.Close()

	var result []domain.SNMPInterfaceRow
	for rows.Next() {
		var row domain.SNMPInterfaceRow
		if err := rows.Scan(&row.ID, &row.MonitorID, &row.IfIndex, &row.IfName, &row.IfDescr, &row.IfAlias,
			&row.DisplayName, &row.Ignore, &row.Monitor, &row.UtilizationWarning, &row.UtilizationCritical,
			&row.OperDownCritical, &row.LastOperStatus, &row.LastInBps, &row.LastOutBps,
			&row.LastUtilizationPercent, &row.LastCheckAt, &row.CreatedAt, &row.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan snmp interface: %w", err)
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// Event stream ──────────────────────────────────────────────────────────────

func (r *SNMPRepository) InsertEvent(ctx context.Context, event *domain.SNMPEvent) error {
	detailsJSON, err := json.Marshal(event.Details)
	if err != nil {
		return fmt.Errorf("marshal snmp event details: %w", err)
	}

	var workspaceID any
	if event.WorkspaceID != "" {
		workspaceID = event.WorkspaceID
	}
	var monitorID any
	if event.MonitorID != "" {
		monitorID = event.MonitorID
	}
	var probeID any
	if event.ProbeID != "" {
		probeID = event.ProbeID
	}
	var interfaceID any
	if event.InterfaceID != "" {
		interfaceID = event.InterfaceID
	}

	err = r.pool.QueryRow(ctx, `
		INSERT INTO snmp_events (
			workspace_id, resource_id, monitor_id, probe_id, kind, event_type,
			severity, source, summary, interface_id, if_index, if_name, details, created_at
		) VALUES ($1, $2::uuid, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13::jsonb, $14)
		RETURNING id::text`,
		workspaceID, event.ResourceID, monitorID, probeID, string(event.Kind), event.EventType,
		event.Severity, event.Source, event.Summary, interfaceID, event.IfIndex, event.IfName,
		detailsJSON, event.CreatedAt,
	).Scan(&event.ID)
	if err != nil {
		return fmt.Errorf("insert snmp event: %w", err)
	}
	return nil
}

func (r *SNMPRepository) ListEvents(ctx context.Context, filter repository.SNMPEventFilter) ([]domain.SNMPEvent, error) {
	limit := filter.Limit
	if limit <= 0 || limit > 500 {
		limit = 50
	}

	query := `
		SELECT id::text, COALESCE(workspace_id::text, ''), resource_id::text,
		       COALESCE(monitor_id::text, ''), COALESCE(probe_id::text, ''),
		       kind, event_type, severity, source, summary,
		       COALESCE(interface_id::text, ''), COALESCE(if_index, 0), COALESCE(if_name, ''),
		       details, created_at
		FROM snmp_events WHERE 1=1`
	args := []any{}
	if filter.ResourceID != "" {
		args = append(args, filter.ResourceID)
		query += fmt.Sprintf(" AND resource_id = $%d::uuid", len(args))
	}
	if filter.MonitorID != "" {
		args = append(args, filter.MonitorID)
		query += fmt.Sprintf(" AND monitor_id = $%d::uuid", len(args))
	}
	if filter.Kind != "" {
		args = append(args, filter.Kind)
		query += fmt.Sprintf(" AND kind = $%d", len(args))
	}
	if !filter.Since.IsZero() {
		args = append(args, filter.Since)
		query += fmt.Sprintf(" AND created_at >= $%d", len(args))
	}
	args = append(args, limit)
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d", len(args))

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list snmp events: %w", err)
	}
	defer rows.Close()

	var result []domain.SNMPEvent
	for rows.Next() {
		var event domain.SNMPEvent
		var detailsJSON []byte
		if err := rows.Scan(&event.ID, &event.WorkspaceID, &event.ResourceID,
			&event.MonitorID, &event.ProbeID, &event.Kind, &event.EventType, &event.Severity,
			&event.Source, &event.Summary, &event.InterfaceID, &event.IfIndex, &event.IfName,
			&detailsJSON, &event.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan snmp event: %w", err)
		}
		if err := json.Unmarshal(detailsJSON, &event.Details); err != nil {
			event.Details = nil
		}
		if event.Kind == "" {
			event.Kind = domain.SNMPEventTrap
		}
		result = append(result, event)
	}
	return result, rows.Err()
}
