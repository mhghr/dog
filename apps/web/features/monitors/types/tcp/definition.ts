import { TCPConfigFields } from "@/components/monitors/probe-config-fields";
import { PlugsConnected } from "@/lib/icons";
import type { MonitorTypeDefinition } from "@/features/monitors/core/definition";

export const tcpMonitorDefinition = {
  type: "tcp", group: "network", icon: PlugsConnected,
  defaultIntervalSeconds: 60, minimumIntervalSeconds: 10,
  defaultValues: {}, ConfigFields: TCPConfigFields,
  apiFieldMap: { "config.port": "tcp_port" },
} satisfies MonitorTypeDefinition;
