// SNMP monitor configuration accessors. Mirrors the monitor configuration
// shape produced by the settings schema (flat keys + health_rules).

export interface SnmpThresholds {
  cpuWarning: number;
  cpuCritical: number;
  memWarning: number;
  memCritical: number;
  utilizationWarning: number;
  utilizationCritical: number;
  temperatureWarning: number;
  temperatureCritical: number;
}

export interface SnmpInterfaceSetting {
  if_index: number;
  if_name?: string;
  display_name?: string;
  ignore?: boolean;
  monitor?: boolean;
  utilization_warning?: number | null;
  utilization_critical?: number | null;
  oper_down_critical?: boolean | null;
}

export interface SnmpConfig {
  host: string;
  port: number;
  version: string;
  thresholds: SnmpThresholds;
  interfaces: SnmpInterfaceSetting[];
}

const DEFAULTS: SnmpThresholds = {
  cpuWarning: 80,
  cpuCritical: 95,
  memWarning: 80,
  memCritical: 95,
  utilizationWarning: 80,
  utilizationCritical: 95,
  temperatureWarning: 60,
  temperatureCritical: 75,
};

function asNumber(value: unknown): number | undefined {
  if (typeof value === "number" && !Number.isNaN(value)) return value;
  if (typeof value === "string" && value.trim() !== "") {
    const n = Number(value);
    return Number.isNaN(n) ? undefined : n;
  }
  return undefined;
}

function ruleOf(
  healthRules: Record<string, unknown> | undefined,
  key: string,
  fallback: { warning: number; critical: number },
): { warning: number; critical: number } {
  const rule = (healthRules?.[key] ?? {}) as { warning?: unknown; critical?: unknown };
  return {
    warning: asNumber(rule.warning) ?? fallback.warning,
    critical: asNumber(rule.critical) ?? fallback.critical,
  };
}

export function readSnmpConfig(configuration: Record<string, unknown>): SnmpConfig {
  const healthRules = configuration.health_rules as Record<string, unknown> | undefined;
  const cpu = ruleOf(healthRules, "cpu_percent", { warning: 80, critical: 95 });
  const mem = ruleOf(healthRules, "memory_percent", { warning: 80, critical: 95 });
  const util = ruleOf(healthRules, "interface_utilization_percent", { warning: 80, critical: 95 });
  const temp = ruleOf(healthRules, "temperature_celsius", { warning: 60, critical: 75 });

  const interfaces: SnmpInterfaceSetting[] = Array.isArray(configuration.interfaces)
    ? (configuration.interfaces as SnmpInterfaceSetting[])
    : [];

  return {
    host: typeof configuration.host === "string" ? configuration.host : "",
    port: asNumber(configuration.port) ?? 161,
    version: typeof configuration.version === "string" ? configuration.version : "3",
    thresholds: {
      ...DEFAULTS,
      cpuWarning: cpu.warning,
      cpuCritical: cpu.critical,
      memWarning: mem.warning,
      memCritical: mem.critical,
      utilizationWarning: util.warning,
      utilizationCritical: util.critical,
      temperatureWarning: temp.warning,
      temperatureCritical: temp.critical,
    },
    interfaces,
  };
}

export function interfaceDisplayName(
  ifIndex: number,
  ifName: string | undefined,
  settings: SnmpInterfaceSetting[],
): string {
  const setting = settings.find((s) => s.if_index === ifIndex);
  if (setting?.display_name) return setting.display_name;
  if (ifName) return ifName;
  return `if${ifIndex}`;
}
