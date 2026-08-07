import { DNSConfigFields } from "@/components/monitors/probe-config-fields";
import { TreeStructure } from "@/lib/icons";
import type { MonitorTypeDefinition } from "@/features/monitors/core/definition";

export const dnsMonitorDefinition = {
  type: "dns", group: "network", icon: TreeStructure,
  defaultIntervalSeconds: 60, minimumIntervalSeconds: 30,
  defaultValues: { dns_record_type: "A", dns_server: "1.1.1.1:53" },
  ConfigFields: DNSConfigFields,
  apiFieldMap: { "config.record_type": "dns_record_type", "config.server": "dns_server" },
} satisfies MonitorTypeDefinition;
