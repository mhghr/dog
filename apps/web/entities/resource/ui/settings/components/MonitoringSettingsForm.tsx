"use client";

import { useMemo, useState } from "react";
import { toast } from "sonner";

import { Button } from "@/shared/ui/button";
import { Input } from "@/shared/ui/input";
import { useCreateResourceMonitor, useUpdateResourceMonitor } from "@/entities/resource/hooks/use-resource";
import type { MonitorTypeDef } from "@/entities/resource/model/types";
import type { Monitor, MonitorInput } from "@/entities/resource/hooks/types";
import { apiErrorMessage } from "@/shared/api/error-message";
import { getMonitoringSchema } from "../schemas";
import { buildMonitorConfiguration } from "../form-state";
import type { SchemaField } from "../monitoring-schema";
import { validateMonitoringConfig, hasErrors, type FieldErrors } from "../validation";
import { MonitoringTypeHeader } from "./MonitoringTypeHeader";
import { SettingsSection } from "./SettingsSection";
import { ExecutionSettingsSection, type ExecutionSettingsValues, INTERVAL_MIN_SECONDS, INTERVAL_MAX_SECONDS } from "./ExecutionSettingsSection";
import { HealthRulesBuilder, type HealthRulesState } from "./HealthRulesBuilder";
import { SchemaFieldRenderer, SchemaToggleRow } from "./SchemaField";
import { SnmpConnectionPanel } from "./SnmpConnectionPanel";

interface MonitoringSettingsFormProps {
  resourceId: string;
  type: MonitorTypeDef;
  monitor: Monitor | undefined;
  target?: string;
  isFa: boolean;
}

function visible(field: SchemaField, config: Record<string, unknown>): boolean {
  if (!field.visibleWhen) return true;
  const { field: dep, equals, equalsAny } = field.visibleWhen;
  const current = config[dep];
  if (equals !== undefined) return current === equals;
  if (equalsAny) return equalsAny.includes(current);
  return true;
}

function asNumber(value: unknown): number | undefined {
  if (typeof value === "number" && !Number.isNaN(value)) return value;
  if (typeof value === "string" && value.trim() !== "") {
    const n = Number(value);
    return Number.isNaN(n) ? undefined : n;
  }
  return undefined;
}

// Pick the message for the current locale so the form stores plain strings.
function localizeErrors(errors: FieldErrors, isFa: boolean): Record<string, string> {
  const result: Record<string, string> = {};
  for (const [key, message] of Object.entries(errors)) {
    result[key] = isFa ? message.fa : message.en;
  }
  return result;
}

// Standard, schema-driven monitoring settings form used by every monitoring
// type. The layout (Configuration → Execution → Health Rules → Probe
// Locations → Advanced) is fixed; the per-type schema decides which fields and
// rules appear.
export function MonitoringSettingsForm({
  resourceId,
  type,
  monitor,
  target = "",
  isFa,
}: MonitoringSettingsFormProps) {
  const t = (en: string, fa: string) => (isFa ? fa : en);
  const schema = useMemo(() => getMonitoringSchema(type), [type]);
  // Boolean rules (availability, assertions) are always-on platform behavior
  // with no thresholds, so they are not configurable in the form.
  const configurableRules = schema.healthRules.filter((rule) => rule.direction !== "boolean");

  const create = useCreateResourceMonitor(resourceId);
  const update = useUpdateResourceMonitor(resourceId);

  const savedConfig = (monitor?.configuration ?? {}) as Record<string, unknown>;
  const savedRules = (savedConfig.health_rules ?? {}) as Record<string, { warning?: number; critical?: number }>;

  const [enabled, setEnabled] = useState(monitor?.enabled ?? false);
  const [config, setConfig] = useState<Record<string, unknown>>(() => {
    const base: Record<string, unknown> = {};
    for (const field of schema.configFields) {
      const saved = savedConfig[field.key];
      base[field.key] = saved !== undefined ? saved : field.defaultValue;
    }
    if (schema.target?.widget === "text" && schema.target.key) {
      base[schema.target.key] = savedConfig[schema.target.key] ?? "";
    }
    return base;
  });

  const [healthRules, setHealthRules] = useState<HealthRulesState>(() => {
    const base: HealthRulesState = {};
    for (const rule of configurableRules) {
      const saved = savedRules[rule.key];
      base[rule.key] = {
        warning: saved?.warning ?? rule.defaults?.warning ?? 0,
        critical: saved?.critical ?? rule.defaults?.critical ?? 0,
      };
    }
    return base;
  });

  const [execution, setExecution] = useState<ExecutionSettingsValues>(() => {
    const defaultInterval = monitor?.interval_seconds ?? schema.execution.defaultIntervalSeconds;
    return {
      intervalSeconds: Math.min(INTERVAL_MAX_SECONDS, Math.max(INTERVAL_MIN_SECONDS, defaultInterval)),
      timeoutMillis: monitor?.timeout_millis ?? schema.execution.defaultTimeoutMillis,
      retries: monitor?.retries ?? schema.execution.defaultRetries,
      retryDelayMs: asNumber(savedConfig.retry_delay_ms) ?? 500,
    };
  });

  const [errors, setErrors] = useState<Record<string, string>>({});
  const [pending, setPending] = useState(false);

  const setField = (key: string, value: unknown) => setConfig((prev) => ({ ...prev, [key]: value }));

  // All config fields (configuration + advanced) render inside Execution
  // Settings, filtered only by conditional visibility. Switches are separated
  // from the input grid so every input box stays uniform.
  const visibleFields = schema.configFields.filter((field) => visible(field, config));
  const inputFields = visibleFields.filter((field) => field.widget !== "switch");
  const switchFields = visibleFields.filter((field) => field.widget === "switch");

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();

    const validation = validateMonitoringConfig({
      type: schema.type,
      target,
      config,
      intervalSeconds: execution.intervalSeconds,
      timeoutMillis: execution.timeoutMillis,
      retries: execution.retries,
    });

    if (hasErrors(validation)) {
      setErrors(localizeErrors(validation, isFa));
      toast.error(t("Please fix the highlighted fields", "لطفاً فیلدهای مشخص‌شده را اصلاح کنید"));
      return;
    }
    setErrors({});
    setPending(true);

    try {
      const configuration = buildMonitorConfiguration(config, healthRules, execution);

      const input: MonitorInput = {
        monitor_type_id: type.id,
        name: monitor?.name ?? type.name,
        enabled,
        interval_seconds: execution.intervalSeconds,
        timeout_millis: execution.timeoutMillis,
        retries: execution.retries,
        configuration,
        severity: monitor?.severity ?? "warning",
      };

      if (monitor) {
        await update.mutateAsync({ id: monitor.id, ...input });
      } else {
        await create.mutateAsync(input);
      }

      toast.success(t("Saved", "ذخیره شد"), {
        description: t("Monitoring settings saved", "تنظیمات مانیتورینگ ذخیره شد"),
      });
    } catch (err) {
      const msg = apiErrorMessage(err, isFa);
      toast.error(msg.title, { description: msg.description });
    } finally {
      setPending(false);
    }
  };

  const resetDefaults = () => {
    setEnabled(monitor?.enabled ?? false);
    setExecution({
      intervalSeconds: schema.execution.defaultIntervalSeconds,
      timeoutMillis: schema.execution.defaultTimeoutMillis,
      retries: schema.execution.defaultRetries,
      retryDelayMs: 500,
    });
    const base: Record<string, unknown> = {};
    for (const field of schema.configFields) {
      base[field.key] = field.defaultValue;
    }
    if (schema.target?.widget === "text" && schema.target.key) {
      base[schema.target.key] = "";
    }
    setConfig(base);
    const rules: HealthRulesState = {};
    for (const rule of schema.healthRules) {
      rules[rule.key] = {
        warning: rule.defaults?.warning ?? 0,
        critical: rule.defaults?.critical ?? 0,
      };
    }
    setHealthRules(rules);
    setErrors({});
  };

  return (
    <form onSubmit={submit} noValidate className="flex flex-col gap-4">
      <div className="overflow-hidden rounded-2xl border border-border/50 bg-card shadow-subtle">
        {/* Header */}
        <MonitoringTypeHeader
          schema={schema}
          isFa={isFa}
          enabled={enabled}
          onToggle={setEnabled}
        />

        {/* Execution Settings (includes the monitoring type configuration) */}
        {(visibleFields.length > 0 || schema.target) && (
          <SettingsSection title={t("Execution Settings", "تنظیمات اجرا")}>
            <div className="grid grid-cols-1 gap-x-4 gap-y-5 sm:grid-cols-2 lg:grid-cols-3">
              {/* Interval and timeout sit first so all boxes align in rows. */}
              <ExecutionSettingsSection
                value={execution}
                isFa={isFa}
                onChange={setExecution}
              />

              {schema.target && (() => {
                const targetDef = schema.target;
                const targetKey = targetDef.widget === "text" ? targetDef.key : undefined;
                return (
                  <div className="flex flex-col gap-2">
                    <span className="text-xs font-medium text-muted-foreground">
                      {isFa ? targetDef.label.fa : targetDef.label.en}
                      {targetKey && <span className="text-destructive"> *</span>}
                    </span>
                    {targetKey ? (
                      <>
                        <Input
                          type="text"
                          value={typeof config[targetKey] === "string" ? (config[targetKey] as string) : ""}
                          placeholder={targetDef.placeholder}
                          className="h-10"
                          dir="ltr"
                          data-invalid={errors[targetKey] ? "true" : undefined}
                          onChange={(e) => setField(targetKey, e.target.value)}
                        />
                        {errors[targetKey] && (
                          <p className="text-xs text-destructive">{errors[targetKey]}</p>
                        )}
                      </>
                    ) : (
                      <code className="rounded-md bg-muted/50 px-2.5 py-2 text-xs text-foreground" dir="ltr">
                        {target || t("—", "—")}
                      </code>
                    )}
                  </div>
                );
              })()}
              {inputFields.map((field) => (
                <div
                  key={field.key}
                  className={field.widget === "keyvalue" ? "sm:col-span-2 lg:col-span-3" : ""}
                >
                  <SchemaFieldRenderer
                    field={field}
                    value={config[field.key]}
                    isFa={isFa}
                    error={errors[field.key]}
                    onChange={(value) => setField(field.key, value)}
                  />
                </div>
              ))}
            </div>

            {/* Toggle fields render as aligned rows, never mixed into the
                input grid, so every input box stays the same size. */}
            {switchFields.length > 0 && (
              <div className="mt-4 grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-3">
                {switchFields.map((field) => (
                  <SchemaToggleRow
                    key={field.key}
                    field={field}
                    value={config[field.key]}
                    isFa={isFa}
                    onChange={(value) => setField(field.key, value)}
                  />
                ))}
              </div>
            )}
          </SettingsSection>
        )}

        {/* Health Rules */}
        {configurableRules.length > 0 && (
          <SettingsSection title={t("Health Rules", "قوانین سلامت")}>
            <HealthRulesBuilder
              rules={configurableRules}
              state={healthRules}
              isFa={isFa}
              onChange={(key, next) => setHealthRules((prev) => ({ ...prev, [key]: next }))}
            />
          </SettingsSection>
        )}

        {/* SNMP collector: connection test + discovery + interface policy.
            Requires a saved monitor because discovery runs against the stored
            (encrypted) credentials. */}
        {schema.type === "snmp" && monitor?.id && (
          <SettingsSection title={t("Connection & Discovery", "اتصال و کشف")}>
            <SnmpConnectionPanel resourceId={resourceId} monitorId={monitor.id} isFa={isFa} />
          </SettingsSection>
        )}

        {/* Actions: inside the card, at the bottom, behind a divider line. */}
        <div className="flex items-center justify-end gap-2 border-t border-border/50 bg-muted/15 px-7 py-4">
          <Button type="button" variant="outline" size="sm" disabled={pending} onClick={resetDefaults}>
            {t("Defaults", "پیش‌فرض")}
          </Button>
          <Button type="submit" size="sm" disabled={pending}>
            {pending ? t("Saving...", "در حال ذخیره...") : t("Save", "ذخیره")}
          </Button>
        </div>
      </div>
    </form>
  );
}
