import { DomainExpirationConfigFields } from "@/components/monitors/probe-config-fields";
import { CalendarCheck } from "@/lib/icons";
import type { MonitorTypeDefinition } from "@/features/monitors/core/definition";

export const domainExpirationMonitorDefinition = {
  type: "domain_expiration", group: "domain_email", icon: CalendarCheck,
  defaultIntervalSeconds: 43200, minimumIntervalSeconds: 3600,
  defaultValues: { domain_warning_days: 45, domain_critical_days: 15, domain_check_nameservers: false },
  ConfigFields: DomainExpirationConfigFields, apiFieldMap: {},
} satisfies MonitorTypeDefinition;
