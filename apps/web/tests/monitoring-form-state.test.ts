import { describe, expect, it } from "vitest";

import { buildMonitorConfiguration } from "@/entities/resource/ui/settings/form-state";
import type { HealthRulesState } from "@/entities/resource/ui/settings/components/HealthRulesBuilder";
import type { ExecutionSettingsValues } from "@/entities/resource/ui/settings/components/ExecutionSettingsSection";

const execution: ExecutionSettingsValues = {
  intervalSeconds: 60,
  timeoutMillis: 5000,
  retries: 2,
  retryDelayMs: 500,
};

describe("buildMonitorConfiguration", () => {
  it("merges config fields with health rules and retry delay", () => {
    const healthRules: HealthRulesState = {
      latency_ms: { warning: 200, critical: 500 },
      packet_loss: { warning: 5, critical: 20 },
    };

    const configuration = buildMonitorConfiguration(
      { count: 4, packet_size: 56 },
      healthRules,
      execution,
    );

    expect(configuration).toEqual({
      count: 4,
      packet_size: 56,
      health_rules: {
        latency_ms: { warning: 200, critical: 500 },
        packet_loss: { warning: 5, critical: 20 },
      },
      retry_delay_ms: 500,
    });
  });

  it("serializes boolean rules with zero thresholds", () => {
    const configuration = buildMonitorConfiguration(
      {},
      { reachability: { warning: 0, critical: 0 } },
      execution,
    );

    expect(configuration.health_rules).toEqual({
      reachability: { warning: 0, critical: 0 },
    });
  });

  it("stores execution retry values at the monitor level", () => {
    const configuration = buildMonitorConfiguration({}, {}, execution);
    // retry_delay_ms lives in config; retries/interval/timeout are monitor-level
    // fields handled by the form, so they must not leak into configuration.
    expect(configuration.retry_delay_ms).toBe(500);
    expect(configuration.interval_seconds).toBeUndefined();
    expect(configuration.timeout_millis).toBeUndefined();
    expect(configuration.retries).toBeUndefined();
  });
});
