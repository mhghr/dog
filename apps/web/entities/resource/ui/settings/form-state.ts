import type { HealthRulesState } from "./components/HealthRulesBuilder";
import type { ExecutionSettingsValues } from "./components/ExecutionSettingsSection";

// Pure builder for the monitor `configuration` payload produced by the shared
// settings form. Kept side-effect free so the save shape is unit-testable.
export function buildMonitorConfiguration(
  config: Record<string, unknown>,
  healthRules: HealthRulesState,
  execution: ExecutionSettingsValues,
): Record<string, unknown> {
  const rules: Record<string, { warning: number; critical: number }> = {};
  for (const [key, rule] of Object.entries(healthRules)) {
    rules[key] = { warning: rule.warning, critical: rule.critical };
  }

  return {
    ...config,
    health_rules: rules,
    retry_delay_ms: execution.retryDelayMs,
  };
}
