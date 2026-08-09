"use client";

import { useState } from "react";
import { toast } from "sonner";

import { Info } from "lucide-react";
import { Button } from "@/shared/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/shared/ui/card";
import { Input } from "@/shared/ui/input";
import { Label } from "@/shared/ui/label";
import { Switch } from "@/shared/ui/switch";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/shared/ui/tooltip";
import {
  useCreateResourceMonitor,
  useUpdateResourceMonitor,
} from "@/entities/resource/hooks/use-resource";
import type { MonitorTypeDef } from "@/entities/resource/model/types";
import type { Monitor, MonitorInput } from "@/entities/resource/hooks/types";
import { MonitorTypeIcon } from "./monitor-type-icon";

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
      <CardHeader className="border-b border-border/60 pb-3">
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
            <span className="text-xs text-muted-foreground">{isFa ? "فعال" : "On"}</span>
            <Switch checked={enabled} onCheckedChange={setEnabled} />
          </div>
        </div>
      </CardHeader>
      <CardContent className="pt-4">
        <form onSubmit={handleSubmit} className="flex flex-col gap-5">
          {/* Execution Settings */}
          <section>
            <div className="mb-3 flex items-center gap-2">
              <span className="size-1.5 rounded-full bg-primary/60" />
              <h3 className="text-[13px] font-semibold">{isFa ? "تنظیمات اجرایی" : "Execution"}</h3>
            </div>
            <div className="flex flex-wrap gap-3">
              <div className="flex flex-col gap-1">
                <LabelWithTip tip={isFa ? "فاصله بین هر بار اجرای مانیتور" : "Seconds between each execution"}>
                  {isFa ? "بازه (s)" : "Interval (s)"}
                </LabelWithTip>
                <Input type="number" min={10} value={intervalSeconds} onChange={e => setIntervalSeconds(Number(e.target.value))} />
              </div>
              <div className="flex flex-col gap-1">
                <LabelWithTip tip={isFa ? "حداکثر زمان انتظار برای پاسخ" : "Max wait time for a response"}>
                  {isFa ? "تایم‌اوت (ms)" : "Timeout (ms)"}
                </LabelWithTip>
                <Input type="number" min={100} max={60000} value={timeoutMillis} onChange={e => setTimeoutMillis(Number(e.target.value))} />
              </div>
              <div className="flex flex-col gap-1">
                <LabelWithTip tip={isFa ? "تعداد تلاش مجدد در صورت خطا" : "Retry attempts on failure"}>
                  {isFa ? "تلاش مجدد" : "Retries"}
                </LabelWithTip>
                <Input type="number" min={0} max={5} value={retries} onChange={e => setRetries(Number(e.target.value))} />
              </div>
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
                    <div key={key} className="flex flex-col gap-1">
                      <LabelWithTip tip={prop.title ?? key}>
                        {prop.title ?? key}
                      </LabelWithTip>
                      <Input type={(prop.type === "number" || prop.type === "integer") ? "number" : "text"}
                        min={prop.minimum} max={prop.maximum}
                        value={String(fieldValues[key] ?? "")}
                        onChange={e => setField(key, (prop.type === "number" || prop.type === "integer") ? Number(e.target.value) : e.target.value)}
                        dir="ltr" />
                    </div>
                  )
                ))}
              </div>
            ) : null}
          </section>

          {/* Health Rules */}
          {Object.keys(healthRules).length > 0 ? (
            <section>
              <div className="mb-3 flex items-center gap-2">
                <span className="size-1.5 rounded-full bg-amber-500/60" />
                <h3 className="text-sm font-semibold">{isFa ? "قوانین سلامت" : "Health Rules"}</h3>
              </div>
              <div className="flex flex-col gap-4">
                {Object.entries(healthRules).map(([key, rule]) => {
                  const def = healthParams[key];
                  return (
                    <div key={key}>
                      <div className="mb-2 flex items-center justify-between">
                        <p className="text-sm font-medium">{labelName(key)}</p>
                        {def?.unit ? <span className="text-xs text-muted-foreground">{def.unit}</span> : null}
                      </div>
                      <div className="grid grid-cols-2 gap-3">
                        <div className="flex flex-col gap-1">
                          <Label className="text-xs text-muted-foreground">
                            <span className="inline-block size-1.5 rounded-full bg-amber-500 align-middle mr-1.5" />
                            {isFa ? "هشدار" : "Warning"}
                          </Label>
                          <Input type="number" value={rule.warning ?? ""} placeholder="—"
                            onChange={e => setHealthRules(p => ({ ...p, [key]: { ...p[key], warning: e.target.value === "" ? undefined : Number(e.target.value) } }))}
                            dir="ltr" />
                        </div>
                        <div className="flex flex-col gap-1">
                          <Label className="text-xs text-muted-foreground">
                            <span className="inline-block size-1.5 rounded-full bg-red-500 align-middle mr-1.5" />
                            {isFa ? "بحرانی" : "Critical"}
                          </Label>
                          <Input type="number" value={rule.critical ?? ""} placeholder="—"
                            onChange={e => setHealthRules(p => ({ ...p, [key]: { ...p[key], critical: e.target.value === "" ? undefined : Number(e.target.value) } }))}
                            dir="ltr" />
                        </div>
                      </div>
                    </div>
                  );
                })}
              </div>
            </section>
          ) : null}

          {/* Enable toggle + Save */}
          <div className="flex justify-end border-t border-border/60 pt-4">
            <Button type="submit" disabled={pending} className="min-w-28">
              {pending ? (isFa ? "در حال ذخیره..." : "Saving...") : isFa ? "ذخیره" : "Save"}
            </Button>
          </div>
        </form>
      </CardContent>
    </Card>
  );
}

function LabelWithTip({ children, tip }: { children: React.ReactNode; tip: string }) {
  return (
    <div className="flex items-center gap-1">
      <span className="text-xs text-muted-foreground">{children}</span>
      <Tooltip>
        <TooltipTrigger asChild>
          <span className="inline-flex cursor-help text-muted-foreground/50 transition-colors hover:text-muted-foreground">
            <Info className="size-3" />
          </span>
        </TooltipTrigger>
        <TooltipContent side="top" className="max-w-56 text-xs">
          {tip}
        </TooltipContent>
      </Tooltip>
    </div>
  );
}
