import type { MonitorTypeDefinition, MonitorTypeGroupKey } from "@/plugins/monitoring/core/definition";
import { dnsMonitorDefinition } from "@/plugins/monitoring/dns/definition";
import { domainExpirationMonitorDefinition } from "@/plugins/monitoring/domain-expiration/definition";
import { httpMonitorDefinition } from "@/plugins/monitoring/http/definition";
import { ntpMonitorDefinition } from "@/plugins/monitoring/ntp/definition";
import { pingMonitorDefinition } from "@/plugins/monitoring/ping/definition";
import { smtpMonitorDefinition } from "@/plugins/monitoring/smtp/definition";
import { tcpMonitorDefinition } from "@/plugins/monitoring/tcp/definition";
import { tlsMonitorDefinition } from "@/plugins/monitoring/tls/definition";
import type { MonitorType } from "@/entities/monitor/model/types";
import type { MonitorFormValues } from "@/features/monitor-management/schemas/schemas";

const definitions = [
  httpMonitorDefinition, tcpMonitorDefinition, dnsMonitorDefinition,
  pingMonitorDefinition, tlsMonitorDefinition, domainExpirationMonitorDefinition,
  smtpMonitorDefinition, ntpMonitorDefinition,
] as const satisfies readonly MonitorTypeDefinition[];

export const MONITOR_TYPES = definitions.map((definition) => definition.type) as MonitorType[];

export const MONITOR_REGISTRY = Object.fromEntries(
  definitions.map((definition) => [definition.type, definition]),
) as Record<MonitorType, MonitorTypeDefinition>;

export const MONITOR_TYPE_GROUPS: ReadonlyArray<{ key: MonitorTypeGroupKey; types: MonitorType[] }> = (
  ["web", "network", "domain_email"] as const
).map((key) => ({ key, types: definitions.filter((definition) => definition.group === key).map((definition) => definition.type) }));

export function getMonitorDefinition(type: MonitorType): MonitorTypeDefinition {
  return MONITOR_REGISTRY[type];
}

const COMMON_API_FIELD_MAP: Readonly<Record<string, keyof MonitorFormValues>> = {
  name: "name", target: "target", type: "type",
  interval_seconds: "interval_seconds", timeout_millis: "timeout_millis",
  retries: "retries",
};

export function getMonitorFormField(
  type: MonitorType,
  apiField: string,
): keyof MonitorFormValues | null {
  return COMMON_API_FIELD_MAP[apiField] ?? MONITOR_REGISTRY[type].apiFieldMap[apiField] ?? null;
}
