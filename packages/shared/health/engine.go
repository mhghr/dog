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

func (e *Engine) EvaluateResult(ctx context.Context, result *domain.ProbeResult) error {
	monitorType := detectMonitorType(result)
	if monitorType == "" {
		return nil
	}

	catalogDefs, ok := AllParameters[monitorType]
	if !ok {
		return nil
	}

	rulesByKey := e.rulesByKey(ctx, result.MonitorID)

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

		newState := e.EvaluateParameter(ctx, result.MonitorID, catDef.Key, recentValues, &rule, catDef)
		e.logger.Debug("health parameter evaluated",
			"monitor_id", result.MonitorID,
			"parameter", catDef.Key,
			"state", newState,
			"value", value,
		)
	}

	return nil
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

func splitParamKey(key string) []string {
	var parts []string
	parts = append(parts, key)

	switch {
	case key == "ping.reachability":
		return []string{"ping.reachability", "reachability"}
	case key == "ping.packet_loss_percent":
		return []string{"packet_loss_percent", "packet_loss"}
	case key == "ping.rtt.avg_ms":
		return []string{"rtt_avg_ms", "avg_rtt_ms", "rtt_ms"}
	case key == "ping.rtt.min_ms":
		return []string{"rtt_min_ms", "min_rtt_ms"}
	case key == "ping.rtt.max_ms":
		return []string{"rtt_max_ms", "max_rtt_ms"}
	case key == "ping.jitter_ms":
		return []string{"jitter_ms", "jitter"}
	case key == "http.reachability":
		return []string{"http.reachability"}
	case key == "http.status_code":
		return []string{"status_code"}
	case key == "http.total_duration_ms":
		return []string{"total_duration_ms", "duration_ms"}
	case key == "http.ttfb_ms":
		return []string{"ttfb_ms", "time_to_first_byte_ms"}
	case key == "http.dns_duration_ms":
		return []string{"dns_duration_ms"}
	case key == "http.connect_duration_ms":
		return []string{"connect_duration_ms"}
	case key == "http.tls_duration_ms":
		return []string{"tls_duration_ms"}
	case key == "http.content_assertion":
		return []string{"content_assertion"}
	case key == "tcp.reachability":
		return []string{"tcp.reachability"}
	case key == "tcp.connect_duration_ms":
		return []string{"connect_duration_ms", "connection_time_ms"}
	case key == "tcp.banner_match":
		return []string{"banner_match"}
	case key == "dns.reachability":
		return []string{"dns.reachability"}
	case key == "dns.resolution_duration_ms":
		return []string{"resolution_duration_ms", "resolution_time_ms"}
	case key == "dns.expected_record_match":
		return []string{"expected_record_match", "record_match"}
	case key == "tls.certificate_valid":
		return []string{"certificate_valid", "cert_valid"}
	case key == "tls.hostname_match":
		return []string{"hostname_match"}
	case key == "tls.chain_trusted":
		return []string{"chain_trusted"}
	case key == "tls.days_remaining":
		return []string{"days_remaining", "cert_days_remaining"}
	case key == "tls.protocol_version":
		return []string{"protocol_version", "tls_version"}
	case key == "domain_expiration.days_remaining":
		return []string{"days_remaining", "domain_days_remaining"}
	case key == "domain_expiration.registrar_match":
		return []string{"registrar_match"}
	case key == "domain_expiration.nameserver_match":
		return []string{"nameserver_match"}
	case key == "smtp.reachability":
		return []string{"smtp.reachability"}
	case key == "smtp.banner_match":
		return []string{"banner_match"}
	case key == "smtp.starttls_available":
		return []string{"starttls_available", "tls_available"}
	case key == "smtp.handshake_duration_ms":
		return []string{"handshake_duration_ms", "handshake_time_ms"}
	case key == "ntp.reachability":
		return []string{"ntp.reachability"}
	case key == "ntp.offset_ms":
		return []string{"offset_ms", "offset"}
	case key == "ntp.round_trip_ms":
		return []string{"round_trip_ms", "rtt_ms"}
	case key == "ntp.jitter_ms":
		return []string{"jitter_ms", "jitter"}
	case key == "ntp.stratum":
		return []string{"stratum"}
	}

	return parts
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

func (e *Engine) EvaluateParameter(ctx context.Context, monitorID, paramKey string, recentValues []float64, rule *ParameterRule, catDef ParameterDefinition) HealthState {
	if rule == nil || !rule.Enabled || rule.Mode == ModeDisabled {
		return HealthUnknown
	}

	direction := catDef.Direction

	switch direction {
	case DirBooleanFailure:
		return evaluateBooleanFailure(recentValues, rule)
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

func evaluateHigherIsWorse(recentValues []float64, rule *ParameterRule, catDef ParameterDefinition, repo Repository, ctx context.Context, monitorID, paramKey string) HealthState {
	return evaluateDirectional(recentValues, rule, catDef, repo, ctx, monitorID, paramKey, compareValue)
}

func evaluateLowerIsWorse(recentValues []float64, rule *ParameterRule, catDef ParameterDefinition, repo Repository, ctx context.Context, monitorID, paramKey string) HealthState {
	return evaluateDirectional(recentValues, rule, catDef, repo, ctx, monitorID, paramKey, compareValueLower)
}

// evaluateDirectional applies threshold rules whose direction (higher vs lower
// is worse) is encoded in the compare function.
func evaluateDirectional(recentValues []float64, rule *ParameterRule, catDef ParameterDefinition, repo Repository, ctx context.Context, monitorID, paramKey string, compare func(value, threshold float64, op string) bool) HealthState {
	if len(recentValues) == 0 {
		return evaluateMissingData(rule, repo, ctx, monitorID, paramKey)
	}

	aggValue := aggregateValues(recentValues, rule.Aggregation)
	newState := HealthOK

	if rule.ErrorValue != nil {
		if compare(aggValue, *rule.ErrorValue, rule.ErrorOperator) {
			newState = HealthError
		}
	}

	if rule.WarningValue != nil && newState == HealthOK {
		if compare(aggValue, *rule.WarningValue, rule.WarningOperator) {
			newState = HealthWarning
		}
	}

	if rule.RecoveryValue != nil {
		previousState, err := repo.GetHealthState(ctx, monitorID, paramKey)
		if err == nil && previousState.CurrentState != HealthOK && previousState.CurrentState != HealthUnknown {
			if !compare(aggValue, *rule.RecoveryValue, rule.WarningOperator) {
				newState = HealthOK
			}
		}
	}

	return persistAndReturn(repo, ctx, monitorID, paramKey, newState, aggValue)
}

func evaluateBooleanFailure(recentValues []float64, rule *ParameterRule) HealthState {
	if len(recentValues) == 0 {
		return HealthUnknown
	}

	if recentValues[len(recentValues)-1] < 1.0 {
		return HealthError
	}
	return HealthOK
}

func evaluateEnumState(recentValues []float64, rule *ParameterRule, catDef ParameterDefinition, repo Repository, ctx context.Context, monitorID, paramKey string) HealthState {
	if len(recentValues) == 0 {
		return HealthUnknown
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

func evaluateRangeDeviation(recentValues []float64, rule *ParameterRule, catDef ParameterDefinition, repo Repository, ctx context.Context, monitorID, paramKey string) HealthState {
	if len(recentValues) < 2 {
		return HealthUnknown
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

func evaluateMissingData(rule *ParameterRule, repo Repository, ctx context.Context, monitorID, paramKey string) HealthState {
	if rule.MissingDataPolicy == "IGNORE" {
		return HealthUnknown
	}

	if rule.MissingDataPolicy == "ERROR" {
		return persistAndReturn(repo, ctx, monitorID, paramKey, HealthError, 0)
	}

	if rule.MissingDataPolicy == "WARNING" {
		return persistAndReturn(repo, ctx, monitorID, paramKey, HealthWarning, 0)
	}

	return HealthUnknown
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

func persistAndReturn(repo Repository, ctx context.Context, monitorID, paramKey string, newState HealthState, currentValue float64) HealthState {
	now := time.Now().UTC()

	existing, err := repo.GetHealthState(ctx, monitorID, paramKey)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return newState
	}

	var previousState *HealthState
	var stateChangedAt *time.Time

	if errors.Is(err, domain.ErrNotFound) {
		if newState != HealthUnknown {
			stateChangedAt = &now
		}
	} else {
		if existing.CurrentState != newState {
			previousState = &existing.CurrentState
			stateChangedAt = &now
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

	return newState
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

		state := e.EvaluateParameter(ctx, monitor.ID, catDef.Key, values, &rule, catDef)
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
