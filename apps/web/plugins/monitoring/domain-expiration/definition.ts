import { DomainExpirationConfigFields } from "@/features/monitor-management/ui/probe-config-fields";
import { CalendarCheck } from "@/shared/ui/icons";
import type { MonitorTypeDefinition } from "@/plugins/monitoring/core/definition";

export const domainExpirationMonitorDefinition = {
  type: "domain_expiration", group: "domain_email", icon: CalendarCheck,
  defaultIntervalSeconds: 43200, minimumIntervalSeconds: 3600,
  defaultValues: { domain_warning_days: 45, domain_critical_days: 15, domain_check_nameservers: false },
  ConfigFields: DomainExpirationConfigFields, apiFieldMap: {},
} satisfies MonitorTypeDefinition;
