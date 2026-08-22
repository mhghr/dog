package snmp

import (
	"context"
	"encoding/json"
	"time"

	"monitoring-platform/packages/shared/domain"
)

// ExecuteTask runs an on-demand SNMP operation (test connection or discovery)
// using the collector code shared by the worker and the inline fallback. The
// configuration is the monitor configuration with encrypted secrets; they are
// decrypted in-memory and never leave this function.
//
// This is the single execution entry point used for both the NATS worker path
// (production) and the inline fallback, guaranteeing identical behavior.
func ExecuteTask(ctx context.Context, kind domain.SNMPTaskKind, config map[string]any, key string, registry *Registry) (domain.SNMPTaskResult, error) {
	start := time.Now().UTC()

	data, err := json.Marshal(config)
	if err != nil {
		return domain.SNMPTaskResult{State: string(domain.SNMPStateInvalidConfig), Detail: "invalid monitor configuration"}, err
	}
	var cfg domain.SNMPDeviceConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return domain.SNMPTaskResult{State: string(domain.SNMPStateInvalidConfig), Detail: "invalid monitor configuration"}, err
	}
	if err := DecryptSecretValues(key, &cfg); err != nil {
		return domain.SNMPTaskResult{State: string(domain.SNMPStateInvalidConfig), Detail: "could not decrypt credentials"}, err
	}
	params, err := BuildParams(&cfg)
	if err != nil {
		return domain.SNMPTaskResult{State: string(domain.SNMPStateInvalidConfig), Detail: err.Error()}, err
	}

	result := domain.SNMPTaskResult{
		Kind:           string(kind),
		State:          string(domain.SNMPStateSuccess),
		DurationMillis: time.Since(start).Milliseconds(),
	}

	switch kind {
	case domain.SNMPTaskTest:
		test := TestDevice(ctx, params)
		result.OK = test.OK
		result.State = string(test.State)
		result.Detail = test.Detail
		result.SysName = test.SysName
		result.SysDescr = test.SysDescr
		result.SysObjectID = test.SysObjectID
		result.Uptime = test.Uptime
		result.Steps = test.Steps
		result.DurationMillis = time.Since(start).Milliseconds()

	case domain.SNMPTaskDiscovery:
		discovery, err := Discovery(ctx, params, registry)
		if err != nil {
			result.OK = false
			result.State = string(ResponseErrorState(err))
			result.Detail = SanitizeError(err)
			result.DurationMillis = time.Since(start).Milliseconds()
			return result, err
		}
		raw, _ := json.Marshal(discovery)
		result.OK = true
		result.State = string(domain.SNMPStateSuccess)
		result.Discovery = raw
		result.SysName = discovery.Device.SysName
		result.SysDescr = discovery.Device.SysDescr
		result.SysObjectID = discovery.Device.SysObjectID
		result.Uptime = discovery.Device.SysUpTime
		result.DurationMillis = time.Since(start).Milliseconds()

	default:
		return domain.SNMPTaskResult{State: string(domain.SNMPStateInvalidConfig), Detail: "unknown task kind"}, err
	}

	return result, nil
}
