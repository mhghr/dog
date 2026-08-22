package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"monitoring-platform/packages/shared/domain"
	"monitoring-platform/packages/shared/repository"
	"monitoring-platform/packages/shared/security"
	snmplib "monitoring-platform/packages/shared/snmp"
	"monitoring-platform/packages/shared/snmp/trap"
)

// NewSNMPTrapReceiver builds a UDP-162 trap receiver whose events are bound to
// resources and persisted through the given dependencies.
func NewSNMPTrapReceiver(deps Deps, addr string) *trap.Receiver {
	h := &Handler{deps: deps}
	return trap.NewReceiver(addr, h.snmpTrapSink, trap.StandardNormalizer{Vendors: trap.DefaultVendors()})
}

// SNMPSecretMask is sent by the UI when a secret field is left unchanged on
// update. The API keeps the previously stored encrypted value.
const SNMPSecretMask = "••••••••"

// snmpSecretKeys are the monitor-configuration keys whose values must never be
// stored, logged or returned in plaintext.
var snmpSecretKeys = []string{"community", "authentication_secret", "privacy_secret"}

// loadSnmpMonitor loads a monitor, verifies ownership and that it is an SNMP
// (network-device collector) monitor. On failure it writes the error response.
func (h *Handler) loadSnmpMonitor(w http.ResponseWriter, r *http.Request, monitorID string) (domain.Monitor, bool) {
	if _, err := uuid.Parse(monitorID); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_id", "monitor id must be a valid UUID", nil)
		return domain.Monitor{}, false
	}
	if !h.monitorBelongsToOrg(w, r, monitorID) {
		return domain.Monitor{}, false
	}
	monitor, err := h.deps.MonitorRepo.GetByID(r.Context(), monitorID)
	if err != nil {
		writeDomainError(w, r, err)
		return domain.Monitor{}, false
	}
	if monitor.Type != domain.MonitorSNMP {
		writeError(w, r, http.StatusBadRequest, "not_snmp", "monitor is not an SNMP monitor", nil)
		return domain.Monitor{}, false
	}
	return monitor, true
}

// snmpDeviceConfigFromMonitor converts the monitor configuration into a
// snmpApplyTask applies a completed discovery task to the monitor: persists
// the discovery cache, seeds the interface policy rows and updates the monitor
// configuration so routine polls reuse the cached identity.
func (h *Handler) snmpApplyTask(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskID")
	if _, err := uuid.Parse(taskID); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_id", "task id must be a valid UUID", nil)
		return
	}
	task, err := h.deps.SNMP.GetTask(r.Context(), taskID)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	if !h.resourceBelongsToOrg(w, r, task.ResourceID) {
		return
	}
	if task.Kind != domain.SNMPTaskDiscovery {
		writeError(w, r, http.StatusBadRequest, "not_discovery", "task is not a discovery task", nil)
		return
	}
	if task.Status != domain.SNMPTaskSuccess || len(task.Result) == 0 {
		writeError(w, r, http.StatusConflict, "task_not_ready", "discovery task did not succeed", nil)
		return
	}

	var result domain.SNMPTaskResult
	if err := json.Unmarshal(task.Result, &result); err != nil || len(result.Discovery) == 0 {
		writeError(w, r, http.StatusConflict, "no_result", "discovery result is empty", nil)
		return
	}

	var discovery domain.SNMPDiscoveryResult
	if err := json.Unmarshal(result.Discovery, &discovery); err != nil {
		writeError(w, r, http.StatusInternalServerError, "bad_result", "discovery result could not be decoded", nil)
		return
	}

	if err := h.persistDiscovery(r.Context(), task.MonitorID, &discovery); err != nil {
		writeDomainError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":              true,
		"interfaces":      len(discovery.Interfaces),
		"sensors":         len(discovery.Sensors),
		"vendor":          discovery.Device.Vendor,
		"model":           discovery.Device.Model,
		"sys_name":        discovery.Device.SysName,
		"discovered_at":   discovery.DiscoveredAt,
	})
}

// persistDiscovery stores the discovery cache, seeds interface policy rows and
// updates the monitor configuration.
func (h *Handler) persistDiscovery(ctx context.Context, monitorID string, discovery *domain.SNMPDiscoveryResult) error {
	if err := h.deps.SNMP.UpsertDiscovery(ctx, monitorID, discovery); err != nil {
		return err
	}

	rows := make([]domain.SNMPInterfaceRow, 0, len(discovery.Interfaces))
	for _, inf := range discovery.Interfaces {
		rows = append(rows, domain.SNMPInterfaceRow{
			MonitorID: monitorID,
			IfIndex:   inf.IfIndex,
			IfName:    inf.IfName,
			IfDescr:   inf.IfDescr,
			IfAlias:   inf.IfAlias,
			Monitor:   !defaultIgnoredInterface(inf),
			Ignore:    defaultIgnoredInterface(inf),
		})
	}
	if len(rows) > 0 {
		if err := h.deps.SNMP.BulkUpsertInterfaces(ctx, monitorID, rows); err != nil {
			return err
		}
	}

	monitor, err := h.deps.MonitorRepo.GetByID(ctx, monitorID)
	if err != nil {
		return err
	}
	monitor.Configuration["discovery"] = discovery
	return h.deps.MonitorRepo.Update(ctx, &monitor)
}

func defaultIgnoredInterface(inf domain.SNMPInterfaceInfo) bool {
	if inf.IfIndex == 0 {
		return true
	}
	return false
}

func (h *Handler) snmpGetDiscovery(w http.ResponseWriter, r *http.Request) {
	monitor, ok := h.loadSnmpMonitor(w, r, chi.URLParam(r, "monitorID"))
	if !ok {
		return
	}
	discovery, err := h.deps.SNMP.GetDiscovery(r.Context(), monitor.ID)
	if errors.Is(err, domain.ErrNotFound) {
		writeJSON(w, http.StatusOK, map[string]any{"discovery": nil})
		return
	}
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"discovery": discovery})
}

func (h *Handler) snmpListInterfaces(w http.ResponseWriter, r *http.Request) {
	monitor, ok := h.loadSnmpMonitor(w, r, chi.URLParam(r, "monitorID"))
	if !ok {
		return
	}
	rows, err := h.deps.SNMP.ListInterfaces(r.Context(), monitor.ID)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	if rows == nil {
		rows = []domain.SNMPInterfaceRow{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": rows})
}

func (h *Handler) snmpUpdateInterface(w http.ResponseWriter, r *http.Request) {
	monitor, ok := h.loadSnmpMonitor(w, r, chi.URLParam(r, "monitorID"))
	if !ok {
		return
	}

	ifIndex, err := strconv.Atoi(chi.URLParam(r, "ifIndex"))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_index", "ifIndex must be an integer", nil)
		return
	}

	var input struct {
		DisplayName         *string  `json:"display_name"`
		Ignore              *bool    `json:"ignore"`
		Monitor             *bool    `json:"monitor"`
		UtilizationWarning  *float64 `json:"utilization_warning"`
		UtilizationCritical *float64 `json:"utilization_critical"`
		OperDownCritical    *bool    `json:"oper_down_critical"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_json", "invalid body", nil)
		return
	}

	// Load the current row (may not exist yet) then apply the patch.
	existing, err := h.deps.SNMP.ListInterfaces(r.Context(), monitor.ID)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	var row *domain.SNMPInterfaceRow
	for i := range existing {
		if existing[i].IfIndex == ifIndex {
			row = &existing[i]
			break
		}
	}
	if row == nil {
		row = &domain.SNMPInterfaceRow{MonitorID: monitor.ID, IfIndex: ifIndex, Monitor: true}
	}
	if input.DisplayName != nil {
		row.DisplayName = *input.DisplayName
	}
	if input.Ignore != nil {
		row.Ignore = *input.Ignore
	}
	if input.Monitor != nil {
		row.Monitor = *input.Monitor
	}
	if input.UtilizationWarning != nil {
		row.UtilizationWarning = input.UtilizationWarning
	}
	if input.UtilizationCritical != nil {
		row.UtilizationCritical = input.UtilizationCritical
	}
	if input.OperDownCritical != nil {
		row.OperDownCritical = input.OperDownCritical
	}

	if err := h.deps.SNMP.UpsertInterface(r.Context(), row); err != nil {
		writeDomainError(w, r, err)
		return
	}

	// Mirror the per-interface policy into the monitor configuration so the
	// probe honours it on the next poll.
	cfgInterfaces, _ := monitor.Configuration["interfaces"].([]any)
	updated := upsertConfigInterface(cfgInterfaces, row)
	monitor.Configuration["interfaces"] = updated
	if err := h.deps.MonitorRepo.Update(r.Context(), &monitor); err != nil {
		h.deps.Logger.Error("update monitor interface policy failed", "error", err)
	}

	writeJSON(w, http.StatusOK, row)
}

func upsertConfigInterface(items []any, row *domain.SNMPInterfaceRow) []any {
	if items == nil {
		items = []any{}
	}
	replaced := false
	for i, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if intOf(m["if_index"]) == row.IfIndex {
			items[i] = configInterfaceEntry(row)
			replaced = true
			break
		}
	}
	if !replaced {
		items = append(items, configInterfaceEntry(row))
	}
	return items
}

func configInterfaceEntry(row *domain.SNMPInterfaceRow) map[string]any {
	entry := map[string]any{
		"if_index": row.IfIndex,
		"if_name":  row.IfName,
		"monitor":  row.Monitor,
		"ignore":   row.Ignore,
	}
	if row.DisplayName != "" {
		entry["display_name"] = row.DisplayName
	}
	if row.UtilizationWarning != nil {
		entry["utilization_warning"] = *row.UtilizationWarning
	}
	if row.UtilizationCritical != nil {
		entry["utilization_critical"] = *row.UtilizationCritical
	}
	if row.OperDownCritical != nil {
		entry["oper_down_critical"] = *row.OperDownCritical
	}
	return entry
}

func intOf(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	case json.Number:
		parsed, _ := n.Int64()
		return int(parsed)
	default:
		return -1
	}
}

func (h *Handler) snmpListEvents(w http.ResponseWriter, r *http.Request) {
	monitor, ok := h.loadSnmpMonitor(w, r, chi.URLParam(r, "monitorID"))
	if !ok {
		return
	}
	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	events, err := h.deps.SNMP.ListEvents(r.Context(), repository.SNMPEventFilter{
		ResourceID: monitor.ResourceID,
		MonitorID:  monitor.ID,
		Limit:      limit,
	})
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	if events == nil {
		events = []domain.SNMPEvent{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": events})
}

// ── Trap receiver wiring ───────────────────────────────────────────────────

// snmpTrapSink is a Handler that resolves the source device to a resource and
// persists the normalized event. It is wired into the API process when the
// SNMP trap receiver is enabled.
func (h *Handler) snmpTrapSink(event domain.SNMPEvent) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Resolve the device address to a resource whose monitor targets match.
	resourceID, monitorID, found := h.resolveSnmpSource(ctx, event.Source)
	if !found {
		h.deps.Logger.Debug("snmp trap from unknown device", "source", event.Source)
		return
	}
	event.ResourceID = resourceID
	event.MonitorID = monitorID
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now().UTC()
	}
	if err := h.deps.SNMP.InsertEvent(ctx, &event); err != nil {
		h.deps.Logger.Error("persist snmp trap failed", "error", err, "source", event.Source)
	}
	h.deps.Logger.Info("snmp trap recorded",
		"event_type", event.EventType,
		"resource_id", resourceID,
		"monitor_id", monitorID,
		"source", event.Source,
	)
}

// resolveSnmpSource finds the SNMP monitor whose target host matches the trap
// source address. It avoids leaking tenant data across the event stream.
func (h *Handler) resolveSnmpSource(ctx context.Context, source string) (resourceID, monitorID string, found bool) {
	if source == "" {
		return "", "", false
	}
	// Cheap, bounded scan: SNMP monitors whose resource target equals the
	// source address. Targets are indexed by the monitors repo; the scan is
	// acceptable for the receiver's expected event volume.
	monitors, err := h.deps.MonitorRepo.ListSnmpMonitorsByTarget(ctx, source)
	if err != nil || len(monitors) == 0 {
		return "", "", false
	}
	m := monitors[0]
	return m.ResourceID, m.ID, true
}

// encryptSNMPConfigSecrets encrypts the plaintext secret keys in an SNMP
// monitor configuration. Values equal to SNMPSecretMask are left untouched so
// unchanged secrets survive an update. Returns the updated configuration.
func (h *Handler) encryptSNMPConfigSecrets(config map[string]any, masterKey string) {
	if config == nil || masterKey == "" {
		return
	}
	for _, key := range snmpSecretKeys {
		value, ok := config[key].(string)
		if !ok || value == "" || value == SNMPSecretMask {
			continue
		}
		encrypted, err := security.EncryptSecret(masterKey, value)
		if err != nil {
			continue
		}
		config[key] = encrypted
	}
}

// sanitizeErr strips sensitive material from errors before logging.
func sanitizeErr(err error) string {
	if err == nil {
		return ""
	}
	return fmt.Sprintf("%v", snmplib.SanitizeError(err))
}
