"use client";

import { useEffect, useState } from "react";
import { toast } from "sonner";

import { Button } from "@/shared/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/shared/ui/card";
import { Input } from "@/shared/ui/input";
import { Label } from "@/shared/ui/label";
import { Switch } from "@/shared/ui/switch";
import {
  useCreateResourceMonitor,
  useUpdateResourceMonitor,
} from "@/entities/resource/hooks/use-resource";
import type { MonitorTypeDef } from "@/entities/resource/model/types";
import type { MonitorV2, MonitorV2Input } from "@/entities/resource/model/monitor-v2";

interface MonitorConfigProps {
  resourceId: string;
  type: MonitorTypeDef;
  monitor: MonitorV2 | undefined;
  isFa: boolean;
}

interface SchemaProperty {
  type?: string;
  default?: number | string | boolean;
  enum?: string[];
  title?: string;
  minimum?: number;
  maximum?: number;
}

type HealthParamDef = {
  default_profile?: string;
  warning_threshold?: number;
  critical_threshold?: number;
  unit?: string;
  description?: string;
  [key: string]: unknown;
};

export function MonitorConfig({ resourceId, type, monitor, isFa }: MonitorConfigProps) {
  const createMonitor = useCreateResourceMonitor(resourceId);
  const updateMonitor = useUpdateResourceMonitor(resourceId);

  const schema = (type.config_schema ?? {}) as {
    properties?: Record<string, SchemaProperty>;
  };
  const healthParams = (type.health_parameters ?? {}) as Record<string, HealthParamDef>;

  // Execution settings from schema
  const [enabled, setEnabled] = useState(monitor?.enabled ?? false);
  const [intervalSeconds, setIntervalSeconds] = useState(
    monitor?.interval_seconds ?? 60,
  );
  const [timeoutMillis, setTimeoutMillis] = useState(monitor?.timeout_millis ?? 5000);
  const [retries, setRetries] = useState(monitor?.retries ?? 1);

  // Dynamic schema field values (stored in configuration)
  const [fieldValues, setFieldValues] = useState<Record<string, number | string | boolean>>(
    () => {
      const base = (monitor?.configuration ?? {}) as Record<string, unknown>;
      const values: Record<string, number | string | boolean> = {};
      const props = schema.properties ?? {};
      for (const [key, prop] of Object.entries(props)) {
        const existing = base[key];
        if (typeof existing === "number" || typeof existing === "string" || typeof existing === "boolean") {
          values[key] = existing;
        } else if (typeof prop.default !== "undefined") {
          values[key] = prop.default as number | string | boolean;
        } else if (prop.type === "number" || prop.type === "integer") {
          values[key] = 0;
        } else if (prop.type === "boolean") {
          values[key] = false;
        } else {
          values[key] = "";
        }
      }
      return values;
    },
  );

  // Health rules from health_parameters, with warning/critical thresholds
  const [healthRules, setHealthRules] = useState<Record<string, { warning?: number; critical?: number }>>(
    () => {
      const base = (monitor?.configuration ?? {}) as { health_rules?: Record<string, { warning?: number; critical?: number }> };
      const rules: Record<string, { warning?: number; critical?: number }> = {};
      for (const [key, def] of Object.entries(healthParams)) {
        const existing = base.health_rules?.[key];
        rules[key] = {
          warning:
            existing?.warning ?? def.warning_threshold,
          critical:
            existing?.critical ?? def.critical_threshold,
        };
      }
      return rules;
    },
  );

  const [pending, setPending] = useState(false);
  const isEditing = Boolean(monitor);

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault();
    setPending(true);
    try {
      const configuration: Record<string, unknown> = {
        ...fieldValues,
        health_rules: healthRules,
      };

      const base: MonitorV2Input = {
        monitor_type_id: type.id,
        name: monitor?.name ?? type.name,
        enabled,
        interval_seconds: intervalSeconds,
        timeout_millis: timeoutMillis,
        retries,
        configuration,
        severity: monitor?.severity ?? "warning",
      };

      if (monitor) {
        await updateMonitor.mutateAsync({ id: monitor.id, ...base });
      } else {
        await createMonitor.mutateAsync(base);
      }
      toast.success(isFa ? "ذخیره شد" : "Saved");
    } catch {
      toast.error(isFa ? "خطا در ذخیره" : "Failed to save");
    } finally {
      setPending(false);
    }
  };

  const setField = (key: string, value: number | string | boolean) => {
    setFieldValues((prev) => ({ ...prev, [key]: value }));
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle>{type.name}</CardTitle>
        <CardDescription>{type.description}</CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit} className="flex flex-col gap-5">
          {/* Execution settings */}
          <div>
            <h3 className="mb-3 text-sm font-semibold">
              {isFa ? "تنظیمات اجرایی" : "Execution settings"}
            </h3>
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
              <div className="flex flex-col gap-1.5">
                <Label>{isFa ? "بازه (ثانیه)" : "Interval (s)"}</Label>
                <Input
                  type="number"
                  min={10}
                  value={intervalSeconds}
                  onChange={(e) => setIntervalSeconds(Number(e.target.value))}
                />
              </div>
              <div className="flex flex-col gap-1.5">
                <Label>{isFa ? "تایم‌اوت (ms)" : "Timeout (ms)"}</Label>
                <Input
                  type="number"
                  min={100}
                  max={60000}
                  value={timeoutMillis}
                  onChange={(e) => setTimeoutMillis(Number(e.target.value))}
                />
              </div>
              <div className="flex flex-col gap-1.5">
                <Label>{isFa ? "تلاش مجدد" : "Retries"}</Label>
                <Input
                  type="number"
                  min={0}
                  max={5}
                  value={retries}
                  onChange={(e) => setRetries(Number(e.target.value))}
                />
              </div>
            </div>

            {/* Dynamic schema fields */}
            {Object.entries(schema.properties ?? {}).length > 0 ? (
              <div className="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
                {Object.entries(schema.properties ?? {}).map(([key, prop]) => {
                  if (prop.type === "boolean") {
                    return (
                      <div key={key} className="flex items-center justify-between rounded-lg border border-border p-3">
                        <Label>{prop.title ?? key}</Label>
                        <Switch
                          checked={Boolean(fieldValues[key])}
                          onCheckedChange={(v) => setField(key, v)}
                        />
                      </div>
                    );
                  }
                  const isNumber = prop.type === "number" || prop.type === "integer";
                  return (
                    <div key={key} className="flex flex-col gap-1.5">
                      <Label>{prop.title ?? key}</Label>
                      <Input
                        type={isNumber ? "number" : "text"}
                        min={prop.minimum}
                        max={prop.maximum}
                        value={String(fieldValues[key] ?? "")}
                        onChange={(e) =>
                          setField(
                            key,
                            isNumber ? Number(e.target.value) : e.target.value,
                          )
                        }
                        dir="ltr"
                      />
                    </div>
                  );
                })}
              </div>
            ) : null}
          </div>

          {/* Health rules */}
          {Object.keys(healthRules).length > 0 ? (
            <div>
              <h3 className="mb-3 text-sm font-semibold">
                {isFa ? "قوانین سلامت" : "Health rules"}
              </h3>
              <div className="flex flex-col gap-3">
                {Object.entries(healthRules).map(([key, rule]) => {
                  const def = healthParams[key];
                  return (
                    <div key={key} className="rounded-lg border border-border p-3">
                      <div className="mb-2 flex items-center justify-between">
                        <p className="text-sm font-medium">{key}</p>
                        {def?.unit ? (
                          <span className="text-xs text-muted-foreground">{def.unit}</span>
                        ) : null}
                      </div>
                      <div className="grid grid-cols-2 gap-3">
                        <div className="flex flex-col gap-1.5">
                          <Label className="text-xs text-muted-foreground">
                            {isFa ? "هشدار (Warning)" : "Warning"}
                          </Label>
                          <Input
                            type="number"
                            value={rule.warning ?? ""}
                            placeholder="—"
                            onChange={(e) =>
                              setHealthRules((prev) => ({
                                ...prev,
                                [key]: {
                                  ...prev[key],
                                  warning:
                                    e.target.value === "" ? undefined : Number(e.target.value),
                                },
                              }))
                            }
                            dir="ltr"
                          />
                        </div>
                        <div className="flex flex-col gap-1.5">
                          <Label className="text-xs text-muted-foreground">
                            {isFa ? "بحرانی (Critical)" : "Critical"}
                          </Label>
                          <Input
                            type="number"
                            value={rule.critical ?? ""}
                            placeholder="—"
                            onChange={(e) =>
                              setHealthRules((prev) => ({
                                ...prev,
                                [key]: {
                                  ...prev[key],
                                  critical:
                                    e.target.value === "" ? undefined : Number(e.target.value),
                                },
                              }))
                            }
                            dir="ltr"
                          />
                        </div>
                      </div>
                    </div>
                  );
                })}
              </div>
            </div>
          ) : null}

          <div className="flex items-center justify-between rounded-lg border border-border p-3">
            <Label>{isFa ? "فعال" : "Enabled"}</Label>
            <Switch checked={enabled} onCheckedChange={setEnabled} />
          </div>

          <div className="flex justify-end">
            <Button type="submit" disabled={pending} className="min-w-28">
              {pending
                ? isFa
                  ? "در حال ذخیره..."
                  : "Saving..."
                : isEditing
                  ? isFa
                    ? "به‌روزرسانی"
                    : "Update"
                  : isFa
                    ? "فعال‌سازی"
                    : "Enable"}
            </Button>
          </div>
        </form>
      </CardContent>
    </Card>
  );
}
