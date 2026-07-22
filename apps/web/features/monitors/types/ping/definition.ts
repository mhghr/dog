import { PingConfigFields } from "@/components/monitors/probe-config-fields";
import { Broadcast } from "@/lib/icons";
import type { MonitorTypeDefinition } from "@/features/monitors/core/definition";
import { PingMonitorSummary } from "@/features/monitors/types/ping/summary";
import { PingMonitorConfiguration } from "@/features/monitors/types/ping/configuration";

export const pingMonitorDefinition = {
  type: "ping", group: "network", icon: Broadcast,
  defaultIntervalSeconds: 60, minimumIntervalSeconds: 10,
  defaultValues: { ping_packet_count: 4, ping_warning_latency_millis: 200, ping_critical_latency_millis: 500 }, ConfigFields: PingConfigFields,
  Summary: PingMonitorSummary,
  Configuration: PingMonitorConfiguration,
  apiFieldMap: {},
} satisfies MonitorTypeDefinition;
