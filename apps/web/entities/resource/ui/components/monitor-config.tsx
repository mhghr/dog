"use client";

import { useState } from "react";
import { toast } from "sonner";
import { Clock, SlidersHorizontal, Heart, CircleCheck, AlertTriangle, CircleAlert } from "lucide-react";

import { Button } from "@/shared/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/shared/ui/card";
import { Input } from "@/shared/ui/input";
import { Label } from "@/shared/ui/label";
import { Switch } from "@/shared/ui/switch";
import {
  useCreateResourceMonitor,
  useUpdateResourceMonitor,
} from "@/entities/resource/hooks/use-resource";
import type { MonitorTypeDef } from "@/entities/resource/model/types";
import type { Monitor, MonitorInput } from "@/entities/resource/hooks/types";
import { MonitorTypeIcon } from "./monitor-type-icon";
import { cn } from "@/shared/utils/cn";

interface MonitorConfigProps {
  resourceId: string;
  type: MonitorTypeDef;
  monitor: Monitor | undefined;
  isFa: boolean;
}

export function MonitorConfig({ resourceId, type, monitor, isFa }: MonitorConfigProps) {
  const createMonitor = useCreateResourceMonitor(resourceId);
  const updateMonitor = useUpdateResourceMonitor(resourceId);

  const schema = (type.config_schema ?? {}) as { properties?: Record<string, { type?: string; default?: number | string | boolean; title?: string; minimum?: number; maximum?: number }> };
  const healthParams = (type.health_parameters ?? {}) as Record<string, { default_profile?: string; warning_threshold?: number; critical_threshold?: number; unit?: string }>;

  const [enabled, setEnabled] = useState(monitor?.enabled ?? false);
  const [intervalSeconds, setIntervalSeconds] = useState(monitor?.interval_seconds ?? 60);
  const [timeoutMillis, setTimeoutMillis] = useState(monitor?.timeout_millis ?? 5000);
  const [retries, setRetries] = useState(monitor?.retries ?? 1);

  const [fieldValues, setFieldValues] = useState<Record<string, number | string | boolean>>(() => {
    const base = (monitor?.configuration ?? {}) as Record<string, unknown>;
    const values: Record<string, number | string | boolean> = {};
    for (const [key, prop] of Object.entries(schema.properties ?? {})) {
      const existing = base[key];
      if (typeof existing === "number" || typeof existing === "string" || typeof existing === "boolean") {
        values[key] = existing;
      } else if (typeof prop.default !== "undefined") {
        values[key] = prop.default as number | string | boolean;
      } else {
        values[key] = (prop.type === "number" || prop.type === "integer") ? 0 : (prop.type === "boolean" ? false : "");
      }
    }
    return values;
  });

  const [healthRules, setHealthRules] = useState<Record<string, { warning?: number; critical?: number }>>(() => {
    const base = (monitor?.configuration ?? {}) as { health_rules?: Record<string, { warning?: number; critical?: number }> };
    const rules: Record<string, { warning?: number; critical?: number }> = {};
    for (const [key, def] of Object.entries(healthParams)) {
      const existing = base.health_rules?.[key];
      rules[key] = { warning: existing?.warning ?? def.warning_threshold, critical: existing?.critical ?? def.critical_threshold };
    }
    return rules;
  });

  const [pending, setPending] = useState(false);
  const isEditing = Boolean(monitor);

  const metricLabels: Record<string, string> = {
    latency_ms: isFa ? "تأخیر" : "Latency",
    packet_loss: isFa ? "Packet Loss" : "Packet Loss",
    packet_loss_percent: isFa ? "درصد Packet Loss" : "Packet Loss %",
    jitter_ms: isFa ? "Jitter" : "Jitter",
    response_time_ms: isFa ? "زمان پاسخ" : "Response time",
    rtt_ms: isFa ? "RTT" : "RTT",
    status_code: isFa ? "کد وضعیت" : "Status code",
    cpu_percent: isFa ? "CPU" : "CPU",
    memory_percent: isFa ? "حافظه" : "Memory",
    disk_percent: isFa ? "دیسک" : "Disk",
    days_remaining: isFa ? "روزهای باقی‌مانده" : "Days remaining",
    tls_days_remaining: isFa ? "روزهای TLS" : "TLS days",
    connect_time_ms: isFa ? "زمان اتصال" : "Connect time",
    resolved: isFa ? "وضعیت" : "Resolved",
    connected: isFa ? "وضعیت" : "Connected",
    valid: isFa ? "معتبر" : "Valid",
    container_count: isFa ? "تعداد کانتینر" : "Containers",
    pod_count: isFa ? "تعداد Pod" : "Pods",
    restart_count: isFa ? "راه‌اندازی مجدد" : "Restarts",
    connections_active: isFa ? "اتصالات فعال" : "Active connections",
    query_latency_ms: isFa ? "زمان کوئری" : "Query latency",
    if_oper_status: isFa ? "وضعیت اینترفیس" : "Interface status",
    if_in_octets: isFa ? "ورودی" : "In",
    if_out_octets: isFa ? "خروجی" : "Out",
  };

  const labelName = (key: string) => metricLabels[key] ?? key.replace(/_/g, " ");

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault();
    setPending(true);
    try {
      const configuration: Record<string, unknown> = { ...fieldValues, health_rules: healthRules };
      const base: MonitorInput = {
        monitor_type_id: type.id, name: monitor?.name ?? type.name, enabled,
        interval_seconds: intervalSeconds, timeout_millis: timeoutMillis, retries,
        configuration, severity: monitor?.severity ?? "warning",
      };
      if (monitor) { await updateMonitor.mutateAsync({ id: monitor.id, ...base }); }
      else { await createMonitor.mutateAsync(base); }
      toast.success(isFa ? "ذخیره شد" : "Saved");
    } catch {
      toast.error(isFa ? "خطا در ذخیره" : "Failed to save");
    } finally { setPending(false); }
  };

  const setField = (key: string, value: number | string | boolean) => setFieldValues(p => ({ ...p, [key]: value }));

  return (
    <Card variant="bordered" className="h-full">
      <CardHeader className="border-b border-border/60 pb-3.5">
        <div className="flex items-center justify-between gap-3">
          <div className="flex min-w-0 items-center gap-3">
            <span className="flex size-9 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary">
              <MonitorTypeIcon type={type.name} className="size-4.5" />
            </span>
            <div className="min-w-0">
              <CardTitle className="truncate text-sm">{type.name}</CardTitle>
              <CardDescription className="mt-0.5 truncate text-xs">{type.description}</CardDescription>
            </div>
          </div>
          <div className="flex shrink-0 items-center gap-2">
            <span className="text-[11px] text-muted-foreground">{isFa ? "فعال" : "On"}</span>
            <Switch checked={enabled} onCheckedChange={setEnabled} />
          </div>
        </div>
      </CardHeader>
      <CardContent className="pt-5">
        <form onSubmit={handleSubmit} className="flex flex-col gap-5">
          {/* Execution Settings */}
          <section>
            <div className="mb-3 flex items-center gap-2.5">
              <span className="flex size-7 shrink-0 items-center justify-center rounded-lg bg-primary/8 text-primary">
                <Clock className="size-3.5" />
              </span>
              <h3 className="text-[13px] font-semibold">{isFa ? "تنظیمات اجرایی" : "Execution"}</h3>
            </div>
            <div className="grid grid-cols-3 gap-3">
              <NumberInput label={isFa ? "بازه (s)" : "Interval (s)"} value={intervalSeconds} onChange={setIntervalSeconds} min={10} suffix="s" />
              <NumberInput label={isFa ? "تایم‌اوت (ms)" : "Timeout (ms)"} value={timeoutMillis} onChange={setTimeoutMillis} min={100} max={60000} suffix="ms" />
              <NumberInput label={isFa ? "تلاش مجدد" : "Retries"} value={retries} onChange={setRetries} min={0} max={5} suffix="×" />
            </div>

            {Object.keys(schema.properties ?? {}).length > 0 ? (
              <div className="mt-3 flex flex-wrap gap-3">
                {Object.entries(schema.properties ?? {}).map(([key, prop]) => (
                  prop.type === "boolean" ? (
                    <div key={key} className="flex items-center gap-2 pt-1">
                      <Label className="text-xs">{prop.title ?? key}</Label>
                      <Switch checked={Boolean(fieldValues[key])} onCheckedChange={v => setField(key, v)} />
                    </div>
                  ) : (
                    <NumberInput
                      key={key}
                      label={prop.title ?? key}
                      value={Number(fieldValues[key] ?? 0)}
                      onChange={v => setField(key, v)}
                      min={prop.minimum}
                      max={prop.maximum}
                      dir="ltr"
                    />
                  )
                ))}
              </div>
            ) : null}
          </section>

          {/* Probe Config Fields */}
          {Object.keys(schema.properties ?? {}).length > 2 ? (
            <section>
              <div className="mb-3 flex items-center gap-2.5">
                <span className="flex size-7 shrink-0 items-center justify-center rounded-lg bg-primary/8 text-primary">
                  <SlidersHorizontal className="size-3.5" />
                </span>
                <h3 className="text-[13px] font-semibold">{isFa ? "تنظیمات پروب" : "Probe config"}</h3>
              </div>
              {Object.keys(schema.properties ?? {}).length > 0 ? (
                <div className="flex flex-wrap gap-3">
                  {Object.entries(schema.properties ?? {}).map(([key, prop]) => (
                    prop.type !== "boolean" ? (
                      <NumberInput
                        key={key}
                        label={prop.title ?? key}
                        value={Number(fieldValues[key] ?? 0)}
                        onChange={v => setField(key, v)}
                        min={prop.minimum}
                        max={prop.maximum}
                        dir="ltr"
                      />
                    ) : null
                  ))}
                </div>
              ) : null}
            </section>
          ) : null}

          {/* Health Rules */}
          {Object.keys(healthRules).length > 0 ? (
            <section>
              <div className="mb-3 flex items-center gap-2.5">
                <span className="flex size-7 shrink-0 items-center justify-center rounded-lg bg-amber-500/10 text-amber-600">
                  <Heart className="size-3.5" />
                </span>
                <h3 className="text-[13px] font-semibold">{isFa ? "قوانین سلامت" : "Health Rules"}</h3>
              </div>
              <div className="flex flex-col gap-4">
                {Object.entries(healthRules).map(([key, rule]) => {
                  const def = healthParams[key];
                  return (
                    <div key={key} className="rounded-xl border border-border/50 bg-card/40 p-4">
                      <div className="mb-3 flex items-center justify-between">
                        <p className="text-sm font-semibold">{labelName(key)}</p>
                        {def?.unit ? <span className="text-[11px] text-muted-foreground">{def.unit}</span> : null}
                      </div>
                      <div className="grid grid-cols-2 gap-3">
                        <div className={cn(
                          "flex flex-col gap-1.5 rounded-lg border p-3 transition-all focus-within:ring-[3px]",
                          "border-amber-500/30 bg-amber-500/[0.04] focus-within:ring-amber-500/15",
                        )}>
                          <Label className="flex items-center gap-1.5 text-[11px] font-medium text-muted-foreground">
                            <span className="size-1.5 rounded-full bg-amber-500" />
                            {isFa ? "هشدار" : "Warning"}
                          </Label>
                          <Input type="number" value={rule.warning ?? ""} placeholder="—"
                            onChange={e => setHealthRules(p => ({ ...p, [key]: { ...p[key], warning: e.target.value === "" ? undefined : Number(e.target.value) } }))}
                            dir="ltr" className="border-0 bg-transparent p-0 text-sm font-medium shadow-none focus-visible:ring-0" />
                        </div>
                        <div className={cn(
                          "flex flex-col gap-1.5 rounded-lg border p-3 transition-all focus-within:ring-[3px]",
                          "border-red-500/30 bg-red-500/[0.04] focus-within:ring-red-500/15",
                        )}>
                          <Label className="flex items-center gap-1.5 text-[11px] font-medium text-muted-foreground">
                            <span className="size-1.5 rounded-full bg-red-500" />
                            {isFa ? "بحرانی" : "Critical"}
                          </Label>
                          <Input type="number" value={rule.critical ?? ""} placeholder="—"
                            onChange={e => setHealthRules(p => ({ ...p, [key]: { ...p[key], critical: e.target.value === "" ? undefined : Number(e.target.value) } }))}
                            dir="ltr" className="border-0 bg-transparent p-0 text-sm font-medium shadow-none focus-visible:ring-0" />
                        </div>
                      </div>
                    </div>
                  );
                })}
              </div>

              <div className="mt-4 flex flex-wrap items-center gap-3 text-[11px]">
                <span className={cn("flex items-center gap-1.5", isFa ? "ms-1" : "me-1")}>
                  <CircleCheck className="size-3 text-emerald-500" />
                  <span className="text-muted-foreground">{isFa ? "سالم (> هشدار)" : "Healthy (< warning)"}</span>
                </span>
                <span className="flex items-center gap-1.5">
                  <AlertTriangle className="size-3 text-amber-500" />
                  <span className="text-muted-foreground">{isFa ? "هشدار" : "Warning"}</span>
                </span>
                <span className="flex items-center gap-1.5">
                  <CircleAlert className="size-3 text-red-500" />
                  <span className="text-muted-foreground">{isFa ? "بحرانی" : "Critical"}</span>
                </span>
              </div>
            </section>
          ) : null}

          {/* Save */}
          <div className="flex justify-end border-t border-border/60 pt-4">
            <Button type="submit" disabled={pending} size="lg" className="min-w-32 shadow-sm">
              {pending ? (isFa ? "در حال ذخیره..." : "Saving...") : isFa ? "ذخیره" : "Save"}
            </Button>
          </div>
        </form>
      </CardContent>
    </Card>
  );
}

function NumberInput({
  label,
  value,
  onChange,
  min,
  max,
  suffix,
  dir,
}: {
  label: string;
  value: number;
  onChange: (v: number) => void;
  min?: number;
  max?: number;
  suffix?: string;
  dir?: "ltr" | "rtl";
}) {
  return (
    <div className="flex flex-col gap-1.5">
      <Label className="text-[11px] font-medium text-muted-foreground">{label}</Label>
      <div className="relative">
        <Input
          type="number"
          min={min}
          max={max}
          value={value}
          onChange={e => onChange(Number(e.target.value))}
          dir={dir ?? "ltr"}
          className={cn(suffix ? "pr-10" : "", "font-medium")}
        />
        {suffix ? (
          <span className="pointer-events-none absolute inset-y-0 end-0 flex items-center pe-3 text-[11px] font-medium text-muted-foreground">
            {suffix}
          </span>
        ) : null}
      </div>
    </div>
  );
}
