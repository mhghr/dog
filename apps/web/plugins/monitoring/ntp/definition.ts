import { NTPConfigFields } from "@/features/monitor-management/ui/probe-config-fields";
import { Clock } from "@/shared/ui/icons";
import type { MonitorTypeDefinition } from "@/plugins/monitoring/core/definition";

export const ntpMonitorDefinition = {
  type: "ntp", group: "network", icon: Clock,
  defaultIntervalSeconds: 300, minimumIntervalSeconds: 60,
  defaultValues: { ntp_port: 123, ntp_version: 4, ntp_max_offset_millis: 1000, ntp_max_round_trip_millis: 2000 },
  ConfigFields: NTPConfigFields, apiFieldMap: { "config.version": "ntp_version" },
} satisfies MonitorTypeDefinition;
