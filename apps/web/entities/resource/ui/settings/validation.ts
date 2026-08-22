import type { LocalizedText } from "./monitoring-schema";

export type FieldErrors = Record<string, LocalizedText>;

export interface ValidationValues {
  /** probe executor key, e.g. "http", "ping" */
  type: string;
  /** resource target the monitor checks */
  target: string;
  config: Record<string, unknown>;
  intervalSeconds: number;
  timeoutMillis: number;
  retries: number;
}

const HOSTNAME_PATTERN = /^(?=.{1,253}$)[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$/;
const IP_PATTERN = /^(\d{1,3}\.){3}\d{1,3}$|^[0-9a-fA-F:]+$/;

function asNumber(value: unknown): number | undefined {
  if (typeof value === "number" && !Number.isNaN(value)) return value;
  if (typeof value === "string" && value.trim() !== "") {
    const n = Number(value);
    return Number.isNaN(n) ? undefined : n;
  }
  return undefined;
}

export function isValidPort(value: unknown): boolean {
  const port = asNumber(value);
  return port != null && Number.isInteger(port) && port >= 1 && port <= 65535;
}

export function isValidHostname(value: string): boolean {
  const trimmed = value.trim();
  if (!trimmed) return false;
  if (IP_PATTERN.test(trimmed)) return true;
  return HOSTNAME_PATTERN.test(trimmed);
}

export function isValidUrl(value: string): boolean {
  try {
    const parsed = new URL(value);
    return parsed.protocol === "http:" || parsed.protocol === "https:";
  } catch {
    return false;
  }
}

export function isValidResolver(value: string): boolean {
  const trimmed = value.trim();
  if (!trimmed) return true; // empty = system resolver
  const [host, port] = trimmed.split(":");
  if (!isValidHostname(host)) return false;
  if (port !== undefined && !isValidPort(port)) return false;
  return true;
}

export function parseStatusCodes(value: unknown): number[] | null {
  if (typeof value !== "string" || value.trim() === "") return null;
  const codes = value
    .split(",")
    .map((part) => Number(part.trim()))
    .filter((n) => !Number.isNaN(n) && Number.isInteger(n));
  return codes.length > 0 ? codes : null;
}

/**
 * Per-monitoring-type validation run before save. Returns a map of
 * configuration/execution field keys to localized error messages. An empty
 * result means the configuration is valid.
 */
export function validateMonitoringConfig(values: ValidationValues): FieldErrors {
  const errors: FieldErrors = {};

  // Common execution constraints.
  if (values.intervalSeconds < 3 || values.intervalSeconds > 120) {
    errors.interval_seconds = { en: "Interval must be between 3 and 120 seconds", fa: "بازه باید بین ۳ تا ۱۲۰ ثانیه باشد" };
  }
  if (values.timeoutMillis < 100 || values.timeoutMillis > 600000) {
    errors.timeout_millis = { en: "Timeout must be between 100 and 600000 ms", fa: "تایم‌اوت باید بین ۱۰۰ تا ۶۰۰۰۰۰ میلی‌ثانیه باشد" };
  }
  if (values.retries < 0 || values.retries > 10) {
    errors.retries = { en: "Retries must be between 0 and 10", fa: "تعداد تلاش مجدد باید بین ۰ تا ۱۰ باشد" };
  }
  if (!values.target.trim()) {
    errors.target = { en: "Target is required", fa: "هدف الزامی است" };
  }

  const type = values.type;
  switch (type) {
    case "ping": {
      const count = asNumber(values.config.count);
      if (count == null || count < 1 || count > 20) {
        errors.count = { en: "Packet count must be between 1 and 20", fa: "تعداد بسته باید بین ۱ تا ۲۰ باشد" };
      }
      break;
    }
    case "http": {
      const method = values.config.method;
      const methods = ["GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"];
      if (typeof method !== "string" || !methods.includes(method.toUpperCase())) {
        errors.method = { en: "Invalid HTTP method", fa: "متد HTTP نامعتبر است" };
      }
      if (parseStatusCodes(values.config.expected_status_codes) === null) {
        errors.expected_status_codes = {
          en: "Expected status codes must be comma-separated integers",
          fa: "کدهای وضعیت مورد انتظار باید اعداد صحیح جدا شده با ویرگول باشند",
        };
      }
      const rawUrl = typeof values.config.url === "string" ? values.config.url.trim() : "";
      if (rawUrl === "") {
        errors.url = { en: "Address is required", fa: "آدرس الزامی است" };
      } else if (!isValidUrl(rawUrl) && !isValidHostname(rawUrl)) {
        errors.url = { en: "Enter a valid URL or hostname", fa: "آدرس یا نام میزبان معتبر وارد کنید" };
      }
      break;
    }
    case "tcp": {
      if (!isValidPort(values.config.port)) {
        errors.port = { en: "Port must be between 1 and 65535", fa: "پورت باید بین ۱ تا ۶۵۵۳۵ باشد" };
      }
      break;
    }
    case "dns": {
      const recordTypes = ["A", "AAAA", "CNAME", "MX", "TXT", "NS"];
      const recordType = String(values.config.record_type ?? "A").toUpperCase();
      if (!recordTypes.includes(recordType)) {
        errors.record_type = { en: "Invalid DNS record type", fa: "نوع رکورد DNS نامعتبر است" };
      }
      if (typeof values.config.resolver === "string" && !isValidResolver(values.config.resolver)) {
        errors.resolver = { en: "Invalid resolver (host[:port])", fa: "Resolver نامعتبر است (host[:port])" };
      }
      break;
    }
    case "tls": {
      if (!isValidPort(values.config.port)) {
        errors.port = { en: "Port must be between 1 and 65535", fa: "پورت باید بین ۱ تا ۶۵۵۳۵ باشد" };
      }
      if (
        typeof values.config.server_name === "string" &&
        values.config.server_name.trim() !== "" &&
        !isValidHostname(values.config.server_name)
      ) {
        errors.server_name = { en: "Invalid hostname", fa: "نام میزبان نامعتبر است" };
      }
      break;
    }
    case "snmp": {
      const host = typeof values.config.host === "string" ? values.config.host.trim() : "";
      if (host === "") {
        errors.host = { en: "Device address is required", fa: "آدرس دستگاه الزامی است" };
      } else if (!isValidHostname(host)) {
        errors.host = { en: "Enter a valid IP or hostname", fa: "آدرس IP یا نام میزبان معتبر وارد کنید" };
      }
      if (!isValidPort(values.config.port)) {
        errors.port = { en: "Port must be between 1 and 65535", fa: "پورت باید بین ۱ تا ۶۵۵۳۵ باشد" };
      }
      const version = String(values.config.version ?? "2c");
      if (!["1", "2c", "3"].includes(version)) {
        errors.version = { en: "Invalid SNMP version", fa: "نسخه SNMP نامعتبر است" };
      }
      if (version === "1" || version === "2c") {
        if (typeof values.config.community !== "string" || values.config.community.trim() === "") {
          errors.community = { en: "Community string is required", fa: "رشته Community الزامی است" };
        }
      }
      if (version === "3") {
        if (typeof values.config.username !== "string" || values.config.username.trim() === "") {
          errors.username = { en: "SNMPv3 username is required", fa: "نام کاربری SNMPv3 الزامی است" };
        }
        const level = String(values.config.security_level ?? "noAuthNoPriv");
        if (level === "authNoPriv" || level === "authPriv") {
          if (typeof values.config.authentication_secret !== "string" || values.config.authentication_secret.trim() === "") {
            errors.authentication_secret = { en: "Authentication secret is required", fa: "رمز احراز هویت الزامی است" };
          }
        }
        if (level === "authPriv") {
          if (typeof values.config.privacy_secret !== "string" || values.config.privacy_secret.trim() === "") {
            errors.privacy_secret = { en: "Privacy secret is required", fa: "رمز رمزنگاری الزامی است" };
          }
        }
      }
      break;
    }
    default:
      break;
  }

  return errors;
}

export function hasErrors(errors: FieldErrors): boolean {
  return Object.keys(errors).length > 0;
}
