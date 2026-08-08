import { TLSConfigFields } from "@/features/monitor-management/ui/probe-config-fields";
import { ShieldCheck } from "@/shared/ui/icons";
import type { MonitorTypeDefinition } from "@/plugins/monitoring/core/definition";

export const tlsMonitorDefinition = {
  type: "tls", group: "web", icon: ShieldCheck,
  defaultIntervalSeconds: 3600, minimumIntervalSeconds: 300,
  defaultValues: { tls_port: 443, tls_verify_chain: true, tls_verify_hostname: true, tls_min_version: "1.2", tls_warning_days: 30, tls_critical_days: 7 },
  ConfigFields: TLSConfigFields, apiFieldMap: {},
} satisfies MonitorTypeDefinition;
