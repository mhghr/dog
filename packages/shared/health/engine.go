package health

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"time"

	"monitoring-platform/packages/shared/domain"
)

type Engine struct {
	repo   Repository
	logger *slog.Logger
}

func NewEngine(repo Repository, logger *slog.Logger) *Engine {
	return &Engine{repo: repo, logger: logger}
}

// EvaluateResult evaluates a probe result against the parameter catalog for its
// monitor type, persisting health states. It returns an outcome for every
// parameter whose state transitioned, so callers can trigger notifications.
func (e *Engine) EvaluateResult(ctx context.Context, result *domain.ProbeResult) ([]EvaluateOutcome, error) {
	monitorType := detectMonitorType(result)
	if monitorType == "" {
		return nil, nil
	}

	catalogDefs, ok := AllParameters[monitorType]
	if !ok {
		return nil, nil
	}

	rulesByKey := e.rulesByKey(ctx, result.MonitorID)

	var outcomes []EvaluateOutcome

	for _, catDef := range catalogDefs {
		rule, hasRule := rulesByKey[catDef.Key]
		if !hasRule {
			rule = defaultRuleFromCatalog(catDef, result.MonitorID)
		}

		if !rule.Enabled {
			continue
		}

		value, found := extractParamValue(result, catDef.Key)
		var recentValues []float64
		if found {
			recentValues = []float64{value}
		}

		newState, outcome := e.EvaluateParameter(ctx, result.MonitorID, catDef.Key, recentValues, &rule, catDef)
		e.logger.Debug("health parameter evaluated",
			"monitor_id", result.MonitorID,
			"parameter", catDef.Key,
			"state", newState,
			"value", value,
		)

		if outcome != nil {
			outcomes = append(outcomes, *outcome)
		}
	}

	return outcomes, nil
}

func detectMonitorType(result *domain.ProbeResult) string {
	if mt, ok := result.Attributes["monitor_type"].(string); ok && mt != "" {
		return mt
	}
	if mt, ok := result.Metrics["monitor_type"].(string); ok && mt != "" {
		return mt
	}
	return ""
}

// rulesByKey loads the parameter rules for a monitor and indexes them by key.
func (e *Engine) rulesByKey(ctx context.Context, monitorID string) map[string]ParameterRule {
	rules, err := e.repo.ListParameterRules(ctx, monitorID)
	if err != nil {
		rules = nil
	}

	rulesByKey := make(map[string]ParameterRule, len(rules))
	for _, rule := range rules {
		rulesByKey[rule.ParameterKey] = rule
	}
	return rulesByKey
}

func defaultRuleFromCatalog(catDef ParameterDefinition, monitorID string) ParameterRule {
	return ParameterRule{
		MonitorID:         monitorID,
		ParameterKey:      catDef.Key,
		Mode:              ModeInheritDefault,
		Aggregation:       "avg",
		WindowType:        "checks",
		WindowValue:       3,
		WarningOperator:   "gte",
		WarningValue:      catDef.DefaultWarning,
		ErrorOperator:     "gte",
		ErrorValue:        catDef.DefaultError,
		RecoveryValue:     catDef.Recovery,
		MinimumSamples:    3,
		MissingDataPolicy: "IGNORE",
		MissedChecks:      3,
		CooldownSeconds:   0,
		Enabled:           true,
	}
}

func extractParamValue(result *domain.ProbeResult, key string) (float64, bool) {
	if v, ok := result.Metrics[key]; ok {
		return toFloat64(v)
	}
	components := splitParamKey(key)
	for _, part := range components {
		if v, ok := result.Metrics[part]; ok {
			return toFloat64(v)
		}
	}
	return 0, false
}

// paramKeyAliases maps a catalog parameter key to the metric keys a probe
// executor may actually emit for it. The first entry is always the canonical
// key itself.
var paramKeyAliases = map[string][]string{
	"ping.reachability":             {"ping.reachability", "reachability"},
	"ping.packet_loss_percent":      {"packet_loss_percent", "packet_loss"},
	"ping.rtt.avg_ms":               {"rtt_avg_ms", "avg_rtt_ms", "rtt_ms"},
	"ping.rtt.min_ms":               {"rtt_min_ms", "min_rtt_ms"},
	"ping.rtt.max_ms":               {"rtt_max_ms", "max_rtt_ms"},
	"ping.jitter_ms":                {"jitter_ms", "jitter"},
	"http.reachability":             {"http.reachability", "reachability"},
	"http.status_code":              {"status_code"},
	"http.response_time_ms":         {"response_time_ms", "total_duration_ms", "duration_ms"},
	"http.total_duration_ms":        {"total_duration_ms", "duration_ms", "response_time_ms"},
	"http.ttfb_ms":                  {"ttfb_ms", "time_to_first_byte_ms"},
	"http.dns_duration_ms":          {"dns_duration_ms"},
	"http.connect_duration_ms":      {"connect_duration_ms"},
	"http.tls_duration_ms":          {"tls_duration_ms"},
	"http.download_time_ms":         {"download_time_ms"},
	"http.response_size_bytes":      {"response_size_bytes"},
	"http.content_assertion":        {"content_assertion"},
	"tcp.reachability":              {"tcp.reachability", "reachability"},
	"tcp.connect_time_ms":           {"connect_time_ms", "connection_time_ms", "connect_duration_ms"},
	"dns.reachability":              {"dns.reachability", "reachability"},
	"dns.response_time_ms":          {"response_time_ms", "resolution_duration_ms", "resolution_time_ms", "dns_duration_ms"},
	"dns.answer_count":              {"answer_count"},
	"dns.expected_record_match":     {"expected_record_match", "record_match"},
	"tls.reachability":              {"reachability", "tls.reachability"},
	"ssl.reachability":              {"reachability", "ssl.reachability"},
	"tls.handshake_time_ms":         {"handshake_time_ms", "handshake_duration_ms"},
	"ssl.handshake_time_ms":         {"handshake_time_ms", "handshake_duration_ms"},
	"tls.certificate_expiry_days":   {"certificate_expiry_days", "days_remaining", "cert_days_remaining"},
	"ssl.certificate_expiry_days":   {"certificate_expiry_days", "days_remaining", "cert_days_remaining"},
	"ssl.certificate_valid":         {"certificate_valid", "cert_valid"},
	"tls.certificate_valid":         {"certificate_valid", "cert_valid"},
	"ssl.hostname_match":            {"hostname_match"},
	"tls.hostname_match":            {"hostname_match"},
	"ssl.chain_valid":               {"chain_valid", "chain_trusted"},
	"tls.chain_valid":               {"chain_valid", "chain_trusted"},
	"domain_expiration.days_remaining":     {"days_remaining", "domain_days_remaining"},
	"domain_expiration.registrar_match":    {"registrar_match"},
	"domain_expiration.nameserver_match":   {"nameserver_match"},
	"smtp.reachability":             {"smtp.reachability"},
	"smtp.banner_match":             {"banner_match"},
	"smtp.starttls_available":       {"starttls_available", "tls_available"},
	"smtp.handshake_duration_ms":    {"handshake_duration_ms", "handshake_time_ms"},
	"ntp.reachability":              {"ntp.reachability"},
	"ntp.offset_ms":                 {"offset_ms", "offset"},
	"ntp.round_trip_ms":             {"round_trip_ms", "rtt_ms"},
	"ntp.jitter_ms":                 {"jitter_ms", "jitter"},
	"ntp.stratum":                   {"stratum"},
	"snmp.reachability":             {"snmp.reachability", "reachability"},
	"snmp.device_health":            {"device_health"},
	"snmp.cpu_percent":              {"device.cpu_percent", "cpu_percent"},
	"snmp.memory_percent":           {"device.memory_percent", "memory_percent"},
	"snmp.temperature_celsius":      {"device.temperature_celsius", "temperature_celsius"},
	"snmp.uptime_seconds":           {"device.uptime_seconds", "uptime_seconds"},
	"snmp.interface_oper_status":    {"snmp.interface_oper_status"},
	"snmp.interface_utilization_percent": {"snmp.interface_utilization_percent"},
}

func splitParamKey(key string) []string {
	if aliases, ok := paramKeyAliases[key]; ok {
		return aliases
	}
	return []string{key}
}

func toFloat64(v any) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case int32:
		return float64(val), true
	case bool:
		if val {
			return 1.0, true
		}
		return 0.0, true
	default:
		return 0, false
	}
}

func (e *Engine) EvaluateParameter(ctx context.Context, monitorID, paramKey string, recentValues []float64, rule *ParameterRule, catDef ParameterDefinition) (HealthState, *EvaluateOutcome) {
	if rule == nil || !rule.Enabled || rule.Mode == ModeDisabled {
		return HealthUnknown, nil
	}

	direction := catDef.Direction

	switch direction {
	case DirBooleanFailure:
		return evaluateBooleanFailure(recentValues, rule, e.repo, ctx, monitorID, paramKey)
	case DirHigherIsWorse:
		return evaluateHigherIsWorse(recentValues, rule, catDef, e.repo, ctx, monitorID, paramKey)
	case DirLowerIsWorse:
		return evaluateLowerIsWorse(recentValues, rule, catDef, e.repo, ctx, monitorID, paramKey)
	case DirEnumState:
		return evaluateEnumState(recentValues, rule, catDef, e.repo, ctx, monitorID, paramKey)
	case DirRangeDeviation:
		return evaluateRangeDeviation(recentValues, rule, catDef, e.repo, ctx, monitorID, paramKey)
	default:
		return evaluateHigherIsWorse(recentValues, rule, catDef, e.repo, ctx, monitorID, paramKey)
	}
}

func evaluateHigherIsWorse(recentValues []float64, rule *ParameterRule, catDef ParameterDefinition, repo Repository, ctx context.Context, monitorID, paramKey string) (HealthState, *EvaluateOutcome) {
	return evaluateDirectional(recentValues, rule, catDef, repo, ctx, monitorID, paramKey, compareValue)
}

func evaluateLowerIsWorse(recentValues []float64, rule *ParameterRule, catDef ParameterDefinition, repo Repository, ctx context.Context, monitorID, paramKey string) (HealthState, *EvaluateOutcome) {
	return evaluateDirectional(recentValues, rule, catDef, repo, ctx, monitorID, paramKey, compareValueLower)
}

// evaluateDirectional applies threshold rules whose direction (higher vs lower
// is worse) is encoded in the compare function.
func evaluateDirectional(recentValues []float64, rule *ParameterRule, catDef ParameterDefinition, repo Repository, ctx context.Context, monitorID, paramKey string, compare func(value, threshold float64, op string) bool) (HealthState, *EvaluateOutcome) {
	if len(recentValues) == 0 {
		return evaluateMissingData(rule, repo, ctx, monitorID, paramKey)
	}

	aggValue := aggregateValues(recentValues, rule.Aggregation)

	return evaluateThresholds(aggValue, rule, repo, ctx, monitorID, paramKey, compare)
}

// evaluateThresholds applies error/warning/recovery thresholds against a single
// aggregated value and persists the resulting health state.
func evaluateThresholds(aggValue float64, rule *ParameterRule, repo Repository, ctx context.Context, monitorID, paramKey string, compare func(value, threshold float64, op string) bool) (HealthState, *EvaluateOutcome) {
	newState := thresholdState(aggValue, rule, compare)

	if rule.RecoveryValue != nil {
		applyRecovery(ctx, repo, monitorID, paramKey, aggValue, rule, compare, &newState)
	}

	return persistAndReturn(repo, ctx, monitorID, paramKey, newState, aggValue)
}

// thresholdState computes the state from error/warning thresholds, warning
// only applying when the error threshold has not already fired.
func thresholdState(aggValue float64, rule *ParameterRule, compare func(value, threshold float64, op string) bool) HealthState {
	if rule.ErrorValue != nil && compare(aggValue, *rule.ErrorValue, rule.ErrorOperator) {
		return HealthError
	}
	if rule.WarningValue != nil && compare(aggValue, *rule.WarningValue, rule.WarningOperator) {
		return HealthWarning
	}
	return HealthOK
}

// applyRecovery clears an error/warning state once the aggregated value drops
// back below the recovery threshold (higher-is-worse direction).
func applyRecovery(ctx context.Context, repo Repository, monitorID, paramKey string, aggValue float64, rule *ParameterRule, compare func(value, threshold float64, op string) bool, newState *HealthState) {
	previousState, err := repo.GetHealthState(ctx, monitorID, paramKey)
	if err != nil || previousState.CurrentState == HealthOK || previousState.CurrentState == HealthUnknown {
		return
	}
	if !compare(aggValue, *rule.RecoveryValue, rule.WarningOperator) {
		*newState = HealthOK
	}
}

func evaluateBooleanFailure(recentValues []float64, rule *ParameterRule, repo Repository, ctx context.Context, monitorID, paramKey string) (HealthState, *EvaluateOutcome) {
	if len(recentValues) == 0 {
		return HealthUnknown, nil
	}

	currentValue := recentValues[len(recentValues)-1]
	newState := HealthOK
	if currentValue < 1.0 {
		newState = HealthError
	}

	previous, err := repo.GetHealthState(ctx, monitorID, paramKey)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return newState, nil
	}

	oldState := HealthUnknown
	if err == nil {
		if previous.CurrentState == newState {
			return newState, nil
		}
		oldState = previous.CurrentState
	}

	return newState, &EvaluateOutcome{
		MonitorID:    monitorID,
		ParameterKey: paramKey,
		OldState:     oldState,
		NewState:     newState,
		CurrentValue: currentValue,
	}
}

func evaluateEnumState(recentValues []float64, rule *ParameterRule, catDef ParameterDefinition, repo Repository, ctx context.Context, monitorID, paramKey string) (HealthState, *EvaluateOutcome) {
	if len(recentValues) == 0 {
		return HealthUnknown, nil
	}

	aggValue := recentValues[len(recentValues)-1]
	newState := HealthOK

	previousState, err := repo.GetHealthState(ctx, monitorID, paramKey)
	if err == nil && previousState.CurrentState != HealthUnknown {
		prevVal := 0.0
		if previousState.CurrentValue != nil {
			prevVal = *previousState.CurrentValue
		}
		if math.Abs(aggValue-prevVal) > 0.001 {
			newState = HealthWarning
		}
	}

	return persistAndReturn(repo, ctx, monitorID, paramKey, newState, aggValue)
}

func evaluateRangeDeviation(recentValues []float64, rule *ParameterRule, catDef ParameterDefinition, repo Repository, ctx context.Context, monitorID, paramKey string) (HealthState, *EvaluateOutcome) {
	if len(recentValues) < 2 {
		return HealthUnknown, nil
	}

	mean := 0.0
	for _, v := range recentValues {
		mean += v
	}
	mean /= float64(len(recentValues))

	sumSq := 0.0
	for _, v := range recentValues {
		diff := v - mean
		sumSq += diff * diff
	}
	stddev := math.Sqrt(sumSq / float64(len(recentValues)))

	newState := HealthOK
	if rule.ErrorValue != nil && stddev > *rule.ErrorValue {
		newState = HealthError
	} else if rule.WarningValue != nil && stddev > *rule.WarningValue {
		newState = HealthWarning
	}

	return persistAndReturn(repo, ctx, monitorID, paramKey, newState, stddev)
}

func evaluateMissingData(rule *ParameterRule, repo Repository, ctx context.Context, monitorID, paramKey string) (HealthState, *EvaluateOutcome) {
	if rule.MissingDataPolicy == "IGNORE" {
		return HealthUnknown, nil
	}

	if rule.MissingDataPolicy == "ERROR" {
		return persistAndReturn(repo, ctx, monitorID, paramKey, HealthError, 0)
	}

	if rule.MissingDataPolicy == "WARNING" {
		return persistAndReturn(repo, ctx, monitorID, paramKey, HealthWarning, 0)
	}

	return HealthUnknown, nil
}

func aggregateValues(values []float64, agg string) float64 {
	if len(values) == 0 {
		return 0
	}

	switch agg {
	case "min":
		return minValue(values)
	case "max":
		return maxValue(values)
	case "sum":
		return sumValues(values)
	case "last":
		return values[len(values)-1]
	default:
		return sumValues(values) / float64(len(values))
	}
}

func minValue(values []float64) float64 {
	min := values[0]
	for _, v := range values[1:] {
		if v < min {
			min = v
		}
	}
	return min
}

func maxValue(values []float64) float64 {
	max := values[0]
	for _, v := range values[1:] {
		if v > max {
			max = v
		}
	}
	return max
}

func sumValues(values []float64) float64 {
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum
}

func compareValue(value, threshold float64, op string) bool {
	switch op {
	case "gt":
		return value > threshold
	case "lt":
		return value < threshold
	case "eq":
		return value == threshold
	case "ne":
		return value != threshold
	default:
		return value >= threshold
	}
}

func compareValueLower(value, threshold float64, op string) bool {
	switch op {
	case "gt":
		return value < threshold
	case "lt":
		return value > threshold
	case "eq":
		return value == threshold
	default:
		return value <= threshold
	}
}

func (s HealthState) valueOrZero() float64 {
	weights := map[HealthState]float64{
		HealthOK:      0,
		HealthWarning: 1,
		HealthError:   2,
		HealthUnknown: 3,
	}
	return weights[s]
}

func persistAndReturn(repo Repository, ctx context.Context, monitorID, paramKey string, newState HealthState, currentValue float64) (HealthState, *EvaluateOutcome) {
	now := time.Now().UTC()

	existing, err := repo.GetHealthState(ctx, monitorID, paramKey)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return newState, nil
	}

	var previousState *HealthState
	var stateChangedAt *time.Time
	changed := false

	if errors.Is(err, domain.ErrNotFound) {
		if newState != HealthUnknown {
			stateChangedAt = &now
			changed = true
		}
	} else {
		if existing.CurrentState != newState {
			previousState = &existing.CurrentState
			stateChangedAt = &now
			changed = true
		} else {
			previousState = existing.PreviousState
			stateChangedAt = existing.StateChangedAt
		}
	}

	_ = repo.UpsertHealthState(ctx, &ParameterHealthState{
		MonitorID:      monitorID,
		ParameterKey:   paramKey,
		CurrentState:   newState,
		CurrentValue:   &currentValue,
		EvaluatedAt:    &now,
		PreviousState:  previousState,
		StateChangedAt: stateChangedAt,
	})

	if !changed {
		return newState, nil
	}

	oldState := HealthUnknown
	if previousState != nil {
		oldState = *previousState
	}

	return newState, &EvaluateOutcome{
		MonitorID:    monitorID,
		ParameterKey: paramKey,
		OldState:     oldState,
		NewState:     newState,
		CurrentValue: currentValue,
	}
}

type MonitorHealth struct {
	MonitorID string
	WorstState HealthState
	Parameters map[string]HealthState
}

func (e *Engine) EvaluateMonitor(ctx context.Context, monitor *domain.Monitor, recentResults []domain.ProbeResult) (MonitorHealth, error) {
	monitorType := string(monitor.Type)
	catalogDefs, ok := AllParameters[monitorType]
	if !ok {
		return MonitorHealth{}, nil
	}

	rulesByKey := e.rulesByKey(ctx, monitor.ID)

	paramStates := make(map[string]HealthState)
	worstState := HealthOK

	for _, catDef := range catalogDefs {
		rule, hasRule := rulesByKey[catDef.Key]
		if !hasRule {
			rule = defaultRuleFromCatalog(catDef, monitor.ID)
		}

		if !rule.Enabled {
			continue
		}

		var values []float64
		for _, res := range recentResults {
			if v, ok := extractParamValue(&res, catDef.Key); ok {
				values = append(values, v)
			}
		}

		state, _ := e.EvaluateParameter(ctx, monitor.ID, catDef.Key, values, &rule, catDef)
		paramStates[catDef.Key] = state

		if state.valueOrZero() > worstState.valueOrZero() {
			worstState = state
		}
	}

	return MonitorHealth{
		MonitorID:   monitor.ID,
		WorstState:  worstState,
		Parameters:  paramStates,
	}, nil
}
