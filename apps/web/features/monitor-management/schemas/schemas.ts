import { z } from "zod";

import { getMonitorDefinition } from "@/plugins/monitoring/core/registry";
import { MONITOR_TYPE_VALUES, type CreateMonitorInput, type Monitor, type MonitorType } from "@/entities/monitor/model/types";

type Translator = (key: string, values?: Record<string, string | number>) => string;

const optionalInt = (min: number, max: number) =>
  z.preprocess(
    (value) =>
      value === "" || value === null || value === undefined ||
      (typeof value === "number" && Number.isNaN(value))
        ? undefined
        : value,
    z.coerce.number().int().min(min).max(max).optional(),
  );

const optionalText = z.preprocess(
  (value) => (typeof value === "string" && value.trim() === "" ? undefined : value),
  z.string().trim().optional(),
);

export function createMonitorFormSchema(t: Translator) {
  return z
    .object({
      name: z
        .string()
        .trim()
        .min(2, t("nameMin"))
        .max(200, t("nameMax")),
      type: z.enum(MONITOR_TYPE_VALUES),
      target: z.string().trim().min(1, t("targetRequired")),
      interval_seconds: z.coerce.number().int().min(10).max(604800),
      timeout_millis: z.coerce.number().int().min(100, t("timeoutRange")).max(60000, t("timeoutRange")),
      retries: z.coerce.number().int().min(0, t("retriesRange")).max(5, t("retriesRange")),
      enabled: z.boolean(),
      warning_duration_millis: optionalInt(1, 60000),
      critical_duration_millis: optionalInt(1, 60000),

      http_method: optionalText,
      http_expected_status_codes: optionalText,
      http_body_contains: optionalText,
      http_body: optionalText,
      http_headers: optionalText,
      http_follow_redirects: z.boolean().optional(),
      http_max_redirects: optionalInt(0, 20),
      http_verify_tls: z.boolean().optional(),

      tcp_port: optionalInt(1, 65535),

      dns_server: optionalText,
      dns_record_type: optionalText,
      dns_expected_values: optionalText,

      ping_packet_count: optionalInt(1, 20),
      ping_packet_interval_millis: optionalInt(10, 10000),
      ping_warning_latency_millis: optionalInt(1, 60000),
      ping_critical_latency_millis: optionalInt(1, 60000),
      ping_warning_packet_loss_percent: optionalInt(0, 100),
      ping_critical_packet_loss_percent: optionalInt(0, 100),
      ping_warning_jitter_millis: optionalInt(0, 60000),
      ping_critical_jitter_millis: optionalInt(0, 60000),

      tls_port: optionalInt(1, 65535),
      tls_server_name: optionalText,
      tls_verify_chain: z.boolean().optional(),
      tls_verify_hostname: z.boolean().optional(),
      tls_min_version: optionalText,
      tls_warning_days: optionalInt(1, 3650),
      tls_critical_days: optionalInt(1, 3650),
      tls_expected_issuer: optionalText,
      tls_expected_fingerprint: optionalText,

      domain_warning_days: optionalInt(1, 3650),
      domain_critical_days: optionalInt(1, 3650),
      domain_check_nameservers: z.boolean().optional(),
      domain_expected_registrar: optionalText,
      domain_expected_nameservers: optionalText,

      smtp_port: optionalInt(1, 65535),
      smtp_mode: optionalText,
      smtp_ehlo_domain: optionalText,
      smtp_require_starttls: z.boolean().optional(),
      smtp_verify_tls: z.boolean().optional(),
      smtp_expected_banner: optionalText,
      smtp_expected_capabilities: optionalText,

      ntp_port: optionalInt(1, 65535),
      ntp_version: optionalInt(3, 4),
      ntp_max_offset_millis: optionalInt(1, 600000),
      ntp_max_round_trip_millis: optionalInt(1, 600000),
      ntp_stratum_min: optionalInt(1, 16),
      ntp_stratum_max: optionalInt(1, 16),
      ntp_warning_offset_millis: optionalInt(1, 600000),
      ntp_warning_round_trip_millis: optionalInt(1, 600000),
    })
    .superRefine((value, context) => {
      const target = value.target.trim();

      if (
        value.type === "http" &&
        !target.startsWith("http://") &&
        !target.startsWith("https://")
      ) {
        context.addIssue({
          code: "custom",
          path: ["target"],
          message: t("httpTargetInvalid"),
        });
      }

      if (value.type !== "http" && target.includes("/")) {
        context.addIssue({
          code: "custom",
          path: ["target"],
          message: t("hostTargetInvalid"),
        });
      }

      if (value.type === "tcp" && !target.includes(":") && !value.tcp_port) {
        context.addIssue({
          code: "custom",
          path: ["tcp_port"],
          message: t("tcpPortRequired"),
        });
      }

      const minInterval = getMonitorDefinition(value.type).minimumIntervalSeconds;
      if (value.interval_seconds < minInterval) {
        context.addIssue({
          code: "custom",
          path: ["interval_seconds"],
          message: t("intervalMin", { seconds: minInterval }),
        });
      }
    });
}

export type MonitorFormValues = z.infer<ReturnType<typeof createMonitorFormSchema>>;

export function defaultFormValues(type: MonitorType = "http"): MonitorFormValues {
  const definition = getMonitorDefinition(type);
  return {
    name: "",
    type,
    target: "",
    interval_seconds: definition.defaultIntervalSeconds,
    timeout_millis: 5000,
    retries: 1,
    enabled: true,

    ...definition.defaultValues,
  } as MonitorFormValues;
}

function parseCsv(raw?: string): string[] {
  if (!raw) return [];
  return raw
    .split(/[,\n]/)
    .map((item) => item.trim())
    .filter(Boolean);
}

function parseStatusCodes(raw?: string): number[] {
  return parseCsv(raw)
    .map((item) => Number.parseInt(item, 10))
    .filter((code) => Number.isInteger(code) && code >= 100 && code <= 599);
}

function parseHeaders(raw?: string): Record<string, string> {
  const headers: Record<string, string> = {};
  for (const line of (raw ?? "").split("\n")) {
    const separator = line.indexOf(":");
    if (separator <= 0) continue;
    const name = line.slice(0, separator).trim();
    const value = line.slice(separator + 1).trim();
    if (name) headers[name] = value;
  }
  return headers;
}

function joinCsv(arr: unknown[]): string {
  return arr.map(String).join(", ");
}

function formatHeaders(obj: Record<string, unknown>): string {
  return Object.entries(obj)
    .map(([name, value]) => `${name}: ${String(value)}`)
    .join("\n");
}

type FieldDef = {
  configKey: string;
  formKey: string;
  transform?: "upper" | "codes" | "csv" | "headers";
  reverse?: "csv" | "csv-list" | "headers";
  optional?: true;
};

const TYPE_FIELDS: Record<string, FieldDef[]> = {
  http: [
    { configKey: "method", formKey: "http_method", transform: "upper" },
    { configKey: "expected_status_codes", formKey: "http_expected_status_codes", transform: "codes", reverse: "csv" },
    { configKey: "body_contains", formKey: "http_body_contains" },
    { configKey: "body", formKey: "http_body" },
    { configKey: "headers", formKey: "http_headers", transform: "headers", reverse: "headers" },
    { configKey: "follow_redirects", formKey: "http_follow_redirects", optional: true },
    { configKey: "max_redirects", formKey: "http_max_redirects" },
    { configKey: "verify_tls", formKey: "http_verify_tls", optional: true },
  ],
  tcp: [
    { configKey: "port", formKey: "tcp_port" },
  ],
  dns: [
    { configKey: "server", formKey: "dns_server" },
    { configKey: "record_type", formKey: "dns_record_type", transform: "upper" },
    { configKey: "expected_values", formKey: "dns_expected_values", transform: "csv", reverse: "csv" },
  ],
  ping: [
    { configKey: "packet_count", formKey: "ping_packet_count" },
    { configKey: "packet_interval_millis", formKey: "ping_packet_interval_millis" },
    { configKey: "warning_latency_millis", formKey: "ping_warning_latency_millis" },
    { configKey: "critical_latency_millis", formKey: "ping_critical_latency_millis" },
    { configKey: "warning_packet_loss_percent", formKey: "ping_warning_packet_loss_percent" },
    { configKey: "critical_packet_loss_percent", formKey: "ping_critical_packet_loss_percent" },
    { configKey: "warning_jitter_millis", formKey: "ping_warning_jitter_millis" },
    { configKey: "critical_jitter_millis", formKey: "ping_critical_jitter_millis" },
  ],
  tls: [
    { configKey: "port", formKey: "tls_port" },
    { configKey: "server_name", formKey: "tls_server_name" },
    { configKey: "verify_chain", formKey: "tls_verify_chain", optional: true },
    { configKey: "verify_hostname", formKey: "tls_verify_hostname", optional: true },
    { configKey: "minimum_tls_version", formKey: "tls_min_version" },
    { configKey: "warning_days", formKey: "tls_warning_days" },
    { configKey: "critical_days", formKey: "tls_critical_days" },
    { configKey: "expected_issuer_contains", formKey: "tls_expected_issuer" },
    { configKey: "expected_fingerprint_sha256", formKey: "tls_expected_fingerprint" },
  ],
  domain_expiration: [
    { configKey: "warning_days", formKey: "domain_warning_days" },
    { configKey: "critical_days", formKey: "domain_critical_days" },
    { configKey: "check_nameservers", formKey: "domain_check_nameservers", optional: true },
    { configKey: "expected_registrar_contains", formKey: "domain_expected_registrar" },
    { configKey: "expected_nameservers", formKey: "domain_expected_nameservers", transform: "csv", reverse: "csv" },
  ],
  smtp: [
    { configKey: "port", formKey: "smtp_port" },
    { configKey: "mode", formKey: "smtp_mode" },
    { configKey: "ehlo_domain", formKey: "smtp_ehlo_domain" },
    { configKey: "require_starttls", formKey: "smtp_require_starttls", optional: true },
    { configKey: "verify_tls", formKey: "smtp_verify_tls", optional: true },
    { configKey: "expected_banner_contains", formKey: "smtp_expected_banner" },
    { configKey: "expected_capabilities", formKey: "smtp_expected_capabilities", transform: "csv", reverse: "csv" },
  ],
  ntp: [
    { configKey: "port", formKey: "ntp_port" },
    { configKey: "version", formKey: "ntp_version" },
    { configKey: "max_offset_millis", formKey: "ntp_max_offset_millis" },
    { configKey: "max_round_trip_millis", formKey: "ntp_max_round_trip_millis" },
    { configKey: "allowed_stratum_min", formKey: "ntp_stratum_min" },
    { configKey: "allowed_stratum_max", formKey: "ntp_stratum_max" },
    { configKey: "warning_offset_millis", formKey: "ntp_warning_offset_millis" },
    { configKey: "warning_round_trip_millis", formKey: "ntp_warning_round_trip_millis" },
  ],
};

function isEmptyValue(value: unknown): boolean {
  return value === undefined || value === null || value === "" ||
    (Array.isArray(value) && value.length === 0) ||
    (typeof value === "object" && !Array.isArray(value) && Object.keys(value as object).length === 0);
}

function applyTransform(value: unknown, transform: FieldDef["transform"]): unknown {
  switch (transform) {
    case "upper": return typeof value === "string" ? value.toUpperCase() : value;
    case "codes": return parseStatusCodes(value as string);
    case "csv": return parseCsv(value as string);
    case "headers": return parseHeaders(value as string);
    default: return value;
  }
}

export function buildProbeConfig(values: MonitorFormValues): Record<string, unknown> {
  const config: Record<string, unknown> = {};

  const set = (key: string, value: unknown) => {
    if (isEmptyValue(value)) return;
    config[key] = value;
  };

  set("warning_duration_millis", values.warning_duration_millis);
  set("critical_duration_millis", values.critical_duration_millis);

  const fields = TYPE_FIELDS[values.type] ?? [];
  for (const f of fields) {
    const raw = (values as Record<string, unknown>)[f.formKey];
    if (f.optional && raw === undefined) continue;
    const transformed = f.transform ? applyTransform(raw, f.transform) : raw;
    set(f.configKey, transformed);
  }

  return config;
}

export function buildMonitorPayload(values: MonitorFormValues): CreateMonitorInput {
  return {
    name: values.name.trim(),
    type: values.type,
    target: values.target.trim(),
    interval_seconds: values.interval_seconds,
    timeout_millis: values.timeout_millis,
    retries: values.retries,
    enabled: values.enabled,
    config: buildProbeConfig(values),
  };
}

function applyReverseTransform(raw: unknown, f: FieldDef): unknown | undefined {
  if (f.reverse === "csv") {
    const arr = Array.isArray(raw) ? (raw as unknown[]).map(String) : [];
    return arr.length > 0 ? arr.join(", ") : undefined;
  }
  if (f.reverse === "headers") {
    if (raw && typeof raw === "object" && !Array.isArray(raw)) {
      return formatHeaders(raw as Record<string, unknown>);
    }
    return undefined;
  }
  if (f.transform === "codes") {
    const arr = Array.isArray(raw) ? (raw as number[]).map(String) : [];
    return arr.length > 0 ? arr.join(", ") : undefined;
  }
  return raw;
}

export function monitorToFormValues(monitor: Monitor): MonitorFormValues {
  const values = defaultFormValues(monitor.type);
  const config = monitor.config ?? {};

  const bool = (key: string) =>
    typeof config[key] === "boolean" ? (config[key] as boolean) : undefined;
  const num = (key: string) =>
    typeof config[key] === "number" ? (config[key] as number) : undefined;

  values.name = monitor.name;
  values.type = monitor.type;
  values.target = monitor.target;
  values.interval_seconds = monitor.interval_seconds;
  values.timeout_millis = monitor.timeout_millis;
  values.retries = monitor.retries;
  values.enabled = monitor.enabled;
  values.warning_duration_millis = num("warning_duration_millis");
  values.critical_duration_millis = num("critical_duration_millis");

  const fields = TYPE_FIELDS[monitor.type] ?? [];
  for (const f of fields) {
    const raw = config[f.configKey];
    if (raw == null) continue;

    if (f.optional && typeof raw === "boolean") {
      (values as Record<string, unknown>)[f.formKey] = bool(f.configKey) ?? (values as Record<string, unknown>)[f.formKey];
      continue;
    }

    const result = applyReverseTransform(raw, f);
    if (result !== undefined) {
      (values as Record<string, unknown>)[f.formKey] = result;
    }
  }

  return values;
}
