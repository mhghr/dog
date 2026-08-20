import { HTTPConfigFields } from "@/features/monitor-management/ui/probe-config-fields";
import { Globe } from "@/shared/ui/icons";
import type { MonitorTypeDefinition } from "@/plugins/monitoring/core/definition";
import { HttpMonitorSummary } from "@/plugins/monitoring/http/summary";
import { HttpMonitorConfiguration } from "@/plugins/monitoring/http/configuration";

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
  Summary: HttpMonitorSummary,
  Configuration: HttpMonitorConfiguration,
  apiFieldMap: {
    "config.method": "http_method",
    "config.follow_redirects": "http_follow_redirects",
    "config.verify_tls": "http_verify_tls",
    "config.max_redirects": "http_max_redirects",
    "config.expected_status_codes": "http_expected_status_codes",
    "config.body_contains": "http_body_contains",
    "config.request_body": "http_body",
    "config.headers": "http_headers",
  },
} satisfies MonitorTypeDefinition;
