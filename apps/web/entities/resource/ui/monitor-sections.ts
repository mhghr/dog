// partitionMonitorsByType groups monitors by executor type for the monitoring
// sections. Each entry keeps the monitors of one executor type. Kept in a
// dependency-free module so it can be unit-tested in isolation.
import {
  isDnsMonitor,
  isHttpMonitor,
  isPingMonitor,
  isSnmpMonitor,
  isTcpMonitor,
  isTlsMonitor,
} from "@/entities/resource/hooks/resource-query";
import type { Monitor } from "@/entities/resource/hooks/types";
import type { MonitorTypeDef } from "@/entities/resource/model/types";

export function partitionMonitorsByType(
  monitors: Monitor[],
  types: MonitorTypeDef[],
): Array<{ type: string; monitors: Monitor[] }> {
  return [
    { type: "ping", monitors: monitors.filter((m) => isPingMonitor(m, types)) },
    { type: "http", monitors: monitors.filter((m) => isHttpMonitor(m, types)) },
    { type: "tcp", monitors: monitors.filter((m) => isTcpMonitor(m, types)) },
    { type: "dns", monitors: monitors.filter((m) => isDnsMonitor(m, types)) },
    { type: "tls", monitors: monitors.filter((m) => isTlsMonitor(m, types)) },
    { type: "snmp", monitors: monitors.filter((m) => isSnmpMonitor(m, types)) },
  ];
}
