import { TCPConfigFields } from "@/features/monitor-management/ui/probe-config-fields";
import { PlugsConnected } from "@/shared/ui/icons";
import type { MonitorTypeDefinition } from "@/plugins/monitoring/core/definition";

export const tcpMonitorDefinition = {
  type: "tcp", group: "network", icon: PlugsConnected,
  defaultIntervalSeconds: 60, minimumIntervalSeconds: 10,
  defaultValues: {}, ConfigFields: TCPConfigFields,
  apiFieldMap: { "config.port": "tcp_port" },
} satisfies MonitorTypeDefinition;
