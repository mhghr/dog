import type { MonitorTypeDef } from "@/entities/resource/model/types";
import type {
  HealthRuleDef,
  LocalizedText,
  MonitoringTypeSchema,
  SchemaField,
} from "../monitoring-schema";
import { pingSchema } from "./ping";
import { httpSchema } from "./http";
import { tcpSchema } from "./tcp";
import { dnsSchema } from "./dns";
import { tlsSchema } from "./tls";
import { snmpSchema } from "./snmp";

const explicitSchemas: Record<string, MonitoringTypeSchema> = {
  ping: pingSchema,
  http: httpSchema,
  tcp: tcpSchema,
  dns: dnsSchema,
  tls: tlsSchema,
  snmp: snmpSchema,
};

// Fallback labels so a type without an explicit schema still renders readable
// field/rule names instead of raw snake_case keys.
const FIELD_LABELS: Record<string, LocalizedText> = {
  host: { en: "Host", fa: "میزبان" },
  port: { en: "Port", fa: "پورت" },
  domain: { en: "Domain", fa: "دامنه" },
  url: { en: "URL", fa: "آدرس" },
  timeout_ms: { en: "Timeout", fa: "تایم‌اوت" },
  ip_version: { en: "IP version", fa: "نسخه IP" },
  verify_tls: { en: "Verify TLS", fa: "اعتبارسنجی TLS" },
  verify_chain: { en: "Verify chain", fa: "اعتبارسنجی زنجیره" },
  verify_hostname: { en: "Verify hostname", fa: "اعتبارسنجی نام میزبان" },
  server_name: { en: "Server name (SNI)", fa: "نام سرور (SNI)" },
  record_type: { en: "Record type", fa: "نوع رکورد" },
  resolver: { en: "Resolver", fa: "Resolver" },
  nameserver: { en: "Resolver", fa: "Resolver" },
  server: { en: "Resolver", fa: "Resolver" },
  count: { en: "Packet count", fa: "تعداد بسته" },
  packet_size: { en: "Packet size", fa: "اندازه بسته" },
  method: { en: "Method", fa: "متد" },
  expected_status: { en: "Expected status", fa: "کد مورد انتظار" },
  follow_redirects: { en: "Follow redirects", fa: "دنبال کردن تغییر مسیر" },
  max_redirects: { en: "Max redirects", fa: "حداکثر تغییر مسیر" },
};

const RULE_LABELS: Record<string, LocalizedText> = {
  reachability: { en: "Availability", fa: "دسترس‌پذیری" },
  response_time_ms: { en: "Response time", fa: "زمان پاسخ" },
  connect_time_ms: { en: "Connect time", fa: "زمان اتصال" },
  handshake_time_ms: { en: "Handshake time", fa: "زمان دست‌داد" },
  certificate_expiry_days: { en: "Certificate expiry", fa: "انقضای گواهی" },
  days_remaining: { en: "Days remaining", fa: "روز باقی‌مانده" },
  latency_ms: { en: "Latency", fa: "تأخیر" },
  packet_loss: { en: "Packet loss", fa: "افت بسته" },
  jitter_ms: { en: "Jitter", fa: "نوسان" },
};

function humanize(key: string): LocalizedText {
  const words = key.replace(/_/g, " ");
  return { en: words, fa: words };
}

function fieldFor(
  key: string,
  prop: { type?: string; default?: unknown; enum?: string[]; minimum?: number; maximum?: number },
): SchemaField {
  const isBoolean = prop.type === "boolean";
  const isSelect = prop.type === "string" && Array.isArray(prop.enum) && prop.enum.length > 0;
  const isNumber = prop.type === "integer" || prop.type === "number";
  const advanced = key === "ip_version";

  return {
    key,
    widget: isBoolean ? "switch" : isSelect ? "select" : isNumber ? "number" : "text",
    label: FIELD_LABELS[key] ?? humanize(key),
    section: advanced ? "advanced" : "configuration",
    ...(isSelect ? { options: prop.enum as string[] } : {}),
    ...(isNumber ? { min: prop.minimum, max: prop.maximum } : {}),
    ...(prop.default !== undefined ? { defaultValue: prop.default as string | number | boolean } : {}),
  };
}

function ruleFor(
  key: string,
  param: { warning_threshold?: number; critical_threshold?: number },
): HealthRuleDef {
  const isBoolean =
    /reachability|match|valid|resolved|available|connected|resolved|assertion/i.test(key);
  return {
    key,
    label: RULE_LABELS[key] ?? humanize(key),
    direction: isBoolean ? "boolean" : "higher_is_worse",
    defaultEnabled: true,
    ...(isBoolean
      ? {}
      : {
          defaults: {
            warning: param.warning_threshold ?? 0,
            critical: param.critical_threshold ?? 0,
          },
        }),
  };
}

/**
 * Returns the monitoring type schema for a type definition. Types with an
 * explicit schema use it; any other type (SMTP, NTP, domain expiration, future
 * types) falls back to a schema derived from the DB `configuration_schema` and
 * `health_parameters`, so new monitoring types get the standard layout without
 * a new form.
 */
export function getMonitoringSchema(type: MonitorTypeDef): MonitoringTypeSchema {
  const explicit = explicitSchemas[type.executor_key];
  if (explicit) {
    return explicit;
  }

  const configSchema = (type.config_schema ?? {}) as {
    properties?: Record<string, { type?: string; default?: unknown; enum?: string[]; minimum?: number; maximum?: number }>;
  };
  const healthParameters = (type.health_parameters ?? {}) as Record<
    string,
    { warning_threshold?: number; critical_threshold?: number }
  >;

  return {
    type: type.executor_key || type.slug,
    title: { en: type.name, fa: type.name },
    description: { en: type.description ?? "", fa: type.description ?? "" },
    target: {
      label: { en: "Target", fa: "هدف" },
      widget: "readonly",
    },
    execution: {
      minimumIntervalSeconds: 10,
      defaultIntervalSeconds: 60,
      defaultTimeoutMillis: 5000,
      defaultRetries: 1,
    },
    configFields: Object.entries(configSchema.properties ?? {}).map(([key, prop]) =>
      fieldFor(key, prop ?? {}),
    ),
    healthRules: Object.entries(healthParameters).map(([key, param]) =>
      ruleFor(key, param ?? {}),
    ),
  };
}

export const MONITORING_SCHEMAS = explicitSchemas;
