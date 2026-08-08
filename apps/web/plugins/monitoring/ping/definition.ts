import { PingConfigFields } from "@/features/monitor-management/ui/probe-config-fields";
import { Broadcast } from "@/shared/ui/icons";
import type { MonitorTypeDefinition } from "@/plugins/monitoring/core/definition";
import { PingMonitorSummary } from "@/plugins/monitoring/ping/summary";
import { PingMonitorConfiguration } from "@/plugins/monitoring/ping/configuration";

export const pingMonitorDefinition = {
  type: "ping", group: "network", icon: Broadcast,
  defaultIntervalSeconds: 60, minimumIntervalSeconds: 10,
  defaultValues: { ping_packet_count: 4, warning_duration_millis: 200, critical_duration_millis: 500, ping_warning_packet_loss_percent: 5, ping_critical_packet_loss_percent: 20, ping_warning_jitter_millis: 50, ping_critical_jitter_millis: 100 }, ConfigFields: PingConfigFields,
  Summary: PingMonitorSummary,
  Configuration: PingMonitorConfiguration,
  apiFieldMap: {},
} satisfies MonitorTypeDefinition;
