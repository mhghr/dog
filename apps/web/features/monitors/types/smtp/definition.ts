import { SMTPConfigFields } from "@/components/monitors/probe-config-fields";
import { EnvelopeSimple } from "@/lib/icons";
import type { MonitorTypeDefinition } from "@/features/monitors/core/definition";

export const smtpMonitorDefinition = {
  type: "smtp", group: "domain_email", icon: EnvelopeSimple,
  defaultIntervalSeconds: 60, minimumIntervalSeconds: 30,
  defaultValues: { smtp_port: 587, smtp_mode: "starttls", smtp_ehlo_domain: "monitor.example.com", smtp_require_starttls: true, smtp_verify_tls: true },
  ConfigFields: SMTPConfigFields,
  apiFieldMap: { "config.mode": "smtp_mode", "config.ehlo_domain": "smtp_ehlo_domain" },
} satisfies MonitorTypeDefinition;
