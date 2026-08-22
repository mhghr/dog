// Monitoring type schema model.
//
// This is the single contract that drives the standard monitoring settings
// layout (Configuration / Execution / Health Rules / Probe Locations /
// Advanced) for every monitoring type — Ping, HTTP, TCP, DNS, TLS and future
// types. The frontend never hardcodes a form: each type declares its schema
// here (or falls back to a generic schema derived from the DB
// `configuration_schema` / `health_parameters`), and the shared renderers
// turn the schema into UI.

export type SectionId =
  | "configuration"
  | "execution"
  | "health_rules"
  | "probe_locations"
  | "advanced";

export interface LocalizedText {
  en: string;
  fa: string;
}

const UNIT_LABELS: Record<string, LocalizedText> = {
  ms: { en: "ms", fa: "میلی‌ثانیه" },
  s: { en: "s", fa: "ثانیه" },
  d: { en: "days", fa: "روز" },
  days: { en: "days", fa: "روز" },
  day: { en: "days", fa: "روز" },
  "%": { en: "%", fa: "درصد" },
};

/** Localizes a unit label ("ms" → "میلی‌ثانیه" in Persian). */
export function localizeUnit(unit: string | undefined, isFa: boolean): string {
  if (!unit) return "";
  const label = UNIT_LABELS[unit];
  if (!label) return unit;
  return isFa ? label.fa : label.en;
}

export type FieldWidget =
  | "text"
  | "number"
  | "select"
  | "switch"
  | "textarea"
  | "keyvalue"
  | "password";

export interface SchemaField {
  /** configuration key, e.g. "port", "ip_version" */
  key: string;
  widget: FieldWidget;
  label: LocalizedText;
  help?: LocalizedText;
  /** which section renders the field */
  section: "configuration" | "advanced";
  options?: string[];
  min?: number;
  max?: number;
  step?: number;
  defaultValue?: string | number | boolean;
  required?: boolean;
  placeholder?: string;
  /** show the field only when another config field matches */
  visibleWhen?: { field: string; equals?: unknown; equalsAny?: unknown[] };
}

export type HealthRuleDirection = "higher_is_worse" | "lower_is_worse" | "boolean";

export interface HealthRuleDef {
  /** health_rules key stored in monitor configuration */
  key: string;
  label: LocalizedText;
  unit?: string;
  direction: HealthRuleDirection;
  description?: LocalizedText;
  defaultEnabled: boolean;
  /** default warning/critical thresholds (single source, mirrors catalog) */
  defaults?: { warning: number; critical: number };
}

export interface ExecutionDefaults {
  minimumIntervalSeconds: number;
  defaultIntervalSeconds: number;
  defaultTimeoutMillis: number;
  defaultRetries: number;
}

export interface TargetDef {
  label: LocalizedText;
  /** readonly shows the resource target; "text" is an editable config field */
  widget: "readonly" | "text";
  /** config key the editable value is stored under (e.g. "url") */
  key?: string;
  placeholder?: string;
}

export interface MonitoringTypeSchema {
  /** probe executor key, e.g. "http", "ping", "tls" */
  type: string;
  title: LocalizedText;
  description: LocalizedText;
  execution: ExecutionDefaults;
  configFields: SchemaField[];
  healthRules: HealthRuleDef[];
  /** how the target/address is rendered; omit to hide it entirely */
  target?: TargetDef;
}

export const INTERVAL_PRESETS = [
  { seconds: 10, label: "10s" },
  { seconds: 30, label: "30s" },
  { seconds: 60, label: "1m" },
  { seconds: 300, label: "5m" },
] as const;
