import { PingConfigFields } from "@/features/monitor-management/ui/probe-config-fields";
import { Broadcast } from "@/shared/ui/icons";
import type { MonitorTypeDefinition } from "@/plugins/monitoring/core/definition";
import { PingMonitorSummary } from "@/plugins/monitoring/ping/summary";
import { PingMonitorConfiguration } from "@/plugins/monitoring/ping/configuration";

export const pingMonitorDefinition = {
  type: "ping", group: "network", icon: Broadcast,
  defaultIntervalSeconds: 60, minimumIntervalSeconds: 10,
  defaultValues: {
    ping_packet_count: 4,
    ping_packet_interval_millis: 200,
    ping_warning_latency_millis: 200,
    ping_critical_latency_millis: 500,
    ping_warning_packet_loss_percent: 5,
    ping_critical_packet_loss_percent: 20,
    ping_warning_jitter_millis: 50,
    ping_critical_jitter_millis: 100,
  },
  ConfigFields: PingConfigFields,
  Summary: PingMonitorSummary,
  Configuration: PingMonitorConfiguration,
  apiFieldMap: {
    "config.packet_count": "ping_packet_count",
    "config.packet_interval_millis": "ping_packet_interval_millis",
    "config.warning_latency_millis": "ping_warning_latency_millis",
    "config.critical_latency_millis": "ping_critical_latency_millis",
    "config.warning_packet_loss_percent": "ping_warning_packet_loss_percent",
    "config.critical_packet_loss_percent": "ping_critical_packet_loss_percent",
    "config.warning_jitter_millis": "ping_warning_jitter_millis",
    "config.critical_jitter_millis": "ping_critical_jitter_millis",
  },
} satisfies MonitorTypeDefinition;
