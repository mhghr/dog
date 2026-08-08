import { HTTPConfigFields } from "@/features/monitor-management/ui/probe-config-fields";
import { Globe } from "@/shared/ui/icons";
import type { MonitorTypeDefinition } from "@/plugins/monitoring/core/definition";

export const httpMonitorDefinition = {
  type: "http",
  group: "web",
  icon: Globe,
  defaultIntervalSeconds: 60,
  minimumIntervalSeconds: 10,
  defaultValues: {
    http_method: "GET",
    http_follow_redirects: true,
    http_verify_tls: true,
    http_expected_status_codes: "200",
  },
  ConfigFields: HTTPConfigFields,
  apiFieldMap: {
    "config.method": "http_method",
    "config.expected_status_codes": "http_expected_status_codes",
  },
} satisfies MonitorTypeDefinition;
