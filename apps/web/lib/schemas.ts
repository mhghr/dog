import { z } from "zod";

import { getMonitorDefinition } from "@/features/monitors/core/registry";
import { MONITOR_TYPE_VALUES, type CreateMonitorInput, type Monitor, type MonitorType } from "@/types/monitor";

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
  if (!raw) {
    return [];
  }

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
    if (separator <= 0) {
      continue;
    }

    const name = line.slice(0, separator).trim();
    const value = line.slice(separator + 1).trim();
    if (name) {
      headers[name] = value;
    }
  }

  return headers;
}

// buildProbeConfig converts flat form values into the per-type config object
// expected by the API.
export function buildProbeConfig(values: MonitorFormValues): Record<string, unknown> {
  const config: Record<string, unknown> = {};

  const set = (key: string, value: unknown) => {
    if (
      value === undefined ||
      value === null ||
      value === "" ||
      (Array.isArray(value) && value.length === 0) ||
      (typeof value === "object" &&
        !Array.isArray(value) &&
        Object.keys(value as object).length === 0)
    ) {
      return;
    }
    config[key] = value;
  };

  switch (values.type) {
    case "http": {
      set("method", values.http_method?.toUpperCase());
      const codes = parseStatusCodes(values.http_expected_status_codes);
      if (codes.length > 0) set("expected_status_codes", codes);
      set("body_contains", values.http_body_contains);
      set("body", values.http_body);
      set("headers", parseHeaders(values.http_headers));
      if (values.http_follow_redirects !== undefined) {
        set("follow_redirects", values.http_follow_redirects);
      }
      set("max_redirects", values.http_max_redirects);
      if (values.http_verify_tls !== undefined) {
        set("verify_tls", values.http_verify_tls);
      }
      break;
    }
    case "tcp": {
      set("port", values.tcp_port);
      break;
    }
    case "dns": {
      set("server", values.dns_server);
      set("record_type", values.dns_record_type?.toUpperCase());
      const expected = parseCsv(values.dns_expected_values);
      if (expected.length > 0) set("expected_values", expected);
      break;
    }
    case "ping": {
      set("packet_count", values.ping_packet_count);
      set("packet_interval_millis", values.ping_packet_interval_millis);
      break;
    }
    case "tls": {
      set("port", values.tls_port);
      set("server_name", values.tls_server_name);
      if (values.tls_verify_chain !== undefined) set("verify_chain", values.tls_verify_chain);
      if (values.tls_verify_hostname !== undefined) {
        set("verify_hostname", values.tls_verify_hostname);
      }
      set("minimum_tls_version", values.tls_min_version);
      set("warning_days", values.tls_warning_days);
      set("critical_days", values.tls_critical_days);
      set("expected_issuer_contains", values.tls_expected_issuer);
      set("expected_fingerprint_sha256", values.tls_expected_fingerprint);
      break;
    }
    case "domain_expiration": {
      set("warning_days", values.domain_warning_days);
      set("critical_days", values.domain_critical_days);
      if (values.domain_check_nameservers !== undefined) {
        set("check_nameservers", values.domain_check_nameservers);
      }
      set("expected_registrar_contains", values.domain_expected_registrar);
      const nameservers = parseCsv(values.domain_expected_nameservers);
      if (nameservers.length > 0) set("expected_nameservers", nameservers);
      break;
    }
    case "smtp": {
      set("port", values.smtp_port);
      set("mode", values.smtp_mode);
      set("ehlo_domain", values.smtp_ehlo_domain);
      if (values.smtp_require_starttls !== undefined) {
        set("require_starttls", values.smtp_require_starttls);
      }
      if (values.smtp_verify_tls !== undefined) set("verify_tls", values.smtp_verify_tls);
      set("expected_banner_contains", values.smtp_expected_banner);
      const capabilities = parseCsv(values.smtp_expected_capabilities);
      if (capabilities.length > 0) set("expected_capabilities", capabilities);
      break;
    }
    case "ntp": {
      set("port", values.ntp_port);
      set("version", values.ntp_version);
      set("max_offset_millis", values.ntp_max_offset_millis);
      set("max_round_trip_millis", values.ntp_max_round_trip_millis);
      set("allowed_stratum_min", values.ntp_stratum_min);
      set("allowed_stratum_max", values.ntp_stratum_max);
      break;
    }
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

// monitorToFormValues flattens a stored monitor back into form values for the
// edit screen.
export function monitorToFormValues(monitor: Monitor): MonitorFormValues {
  const values = defaultFormValues(monitor.type);
  const config = monitor.config ?? {};

  const str = (key: string) =>
    typeof config[key] === "string" ? (config[key] as string) : undefined;
  const num = (key: string) =>
    typeof config[key] === "number" ? (config[key] as number) : undefined;
  const bool = (key: string) =>
    typeof config[key] === "boolean" ? (config[key] as boolean) : undefined;
  const list = (key: string) =>
    Array.isArray(config[key]) ? (config[key] as unknown[]).map(String) : [];

  values.name = monitor.name;
  values.type = monitor.type;
  values.target = monitor.target;
  values.interval_seconds = monitor.interval_seconds;
  values.timeout_millis = monitor.timeout_millis;
  values.retries = monitor.retries;
  values.enabled = monitor.enabled;

  switch (monitor.type) {
    case "http": {
      values.http_method = str("method") ?? values.http_method;
      const codes = list("expected_status_codes");
      if (codes.length > 0) values.http_expected_status_codes = codes.join(", ");
      values.http_body_contains = str("body_contains");
      values.http_body = str("body");
      const headers = config["headers"];
      if (headers && typeof headers === "object" && !Array.isArray(headers)) {
        values.http_headers = Object.entries(headers as Record<string, unknown>)
          .map(([name, value]) => `${name}: ${String(value)}`)
          .join("\n");
      }
      values.http_follow_redirects = bool("follow_redirects") ?? values.http_follow_redirects;
      values.http_max_redirects = num("max_redirects");
      values.http_verify_tls = bool("verify_tls") ?? values.http_verify_tls;
      break;
    }
    case "tcp": {
      values.tcp_port = num("port");
      break;
    }
    case "dns": {
      values.dns_server = str("server") ?? values.dns_server;
      values.dns_record_type = str("record_type") ?? values.dns_record_type;
      const expected = list("expected_values");
      if (expected.length > 0) values.dns_expected_values = expected.join(", ");
      break;
    }
    case "ping": {
      values.ping_packet_count = num("packet_count") ?? values.ping_packet_count;
      values.ping_packet_interval_millis = num("packet_interval_millis");
      break;
    }
    case "tls": {
      values.tls_port = num("port") ?? values.tls_port;
      values.tls_server_name = str("server_name");
      values.tls_verify_chain = bool("verify_chain") ?? values.tls_verify_chain;
      values.tls_verify_hostname = bool("verify_hostname") ?? values.tls_verify_hostname;
      values.tls_min_version = str("minimum_tls_version") ?? values.tls_min_version;
      values.tls_warning_days = num("warning_days") ?? values.tls_warning_days;
      values.tls_critical_days = num("critical_days") ?? values.tls_critical_days;
      values.tls_expected_issuer = str("expected_issuer_contains");
      values.tls_expected_fingerprint = str("expected_fingerprint_sha256");
      break;
    }
    case "domain_expiration": {
      values.domain_warning_days = num("warning_days") ?? values.domain_warning_days;
      values.domain_critical_days = num("critical_days") ?? values.domain_critical_days;
      values.domain_check_nameservers =
        bool("check_nameservers") ?? values.domain_check_nameservers;
      values.domain_expected_registrar = str("expected_registrar_contains");
      const nameservers = list("expected_nameservers");
      if (nameservers.length > 0) {
        values.domain_expected_nameservers = nameservers.join(", ");
      }
      break;
    }
    case "smtp": {
      values.smtp_port = num("port") ?? values.smtp_port;
      values.smtp_mode = str("mode") ?? values.smtp_mode;
      values.smtp_ehlo_domain = str("ehlo_domain") ?? values.smtp_ehlo_domain;
      values.smtp_require_starttls =
        bool("require_starttls") ?? values.smtp_require_starttls;
      values.smtp_verify_tls = bool("verify_tls") ?? values.smtp_verify_tls;
      values.smtp_expected_banner = str("expected_banner_contains");
      const capabilities = list("expected_capabilities");
      if (capabilities.length > 0) {
        values.smtp_expected_capabilities = capabilities.join(", ");
      }
      break;
    }
    case "ntp": {
      values.ntp_port = num("port") ?? values.ntp_port;
      values.ntp_version = num("version") ?? values.ntp_version;
      values.ntp_max_offset_millis = num("max_offset_millis") ?? values.ntp_max_offset_millis;
      values.ntp_max_round_trip_millis =
        num("max_round_trip_millis") ?? values.ntp_max_round_trip_millis;
      values.ntp_stratum_min = num("allowed_stratum_min");
      values.ntp_stratum_max = num("allowed_stratum_max");
      break;
    }
  }

  return values;
}
