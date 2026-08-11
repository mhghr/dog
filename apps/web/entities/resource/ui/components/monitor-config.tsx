"use client";

import { useState } from "react";
import { toast } from "sonner";
import { Clock, Heart } from "lucide-react";

import { Button } from "@/shared/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/shared/ui/card";
import { Input } from "@/shared/ui/input";
import { Label } from "@/shared/ui/label";
import { Switch } from "@/shared/ui/switch";
import { useCreateResourceMonitor, useUpdateResourceMonitor } from "@/entities/resource/hooks/use-resource";
import type { MonitorTypeDef } from "@/entities/resource/model/types";
import type { Monitor, MonitorInput } from "@/entities/resource/hooks/types";
import { MonitorTypeIcon } from "./monitor-type-icon";
import { cn } from "@/shared/utils/cn";

interface Props {
  resourceId: string;
  type: MonitorTypeDef;
  monitor: Monitor | undefined;
  isFa: boolean;
}

const METRIC_LABELS: Record<string, { en: string; fa: string }> = {
  latency_ms: { en: "Latency", fa: "تأخیر" },
  rtt_ms: { en: "RTT", fa: "تأخیر رفت‌وبرگشت" },
  packet_loss: { en: "Packet Loss", fa: "افت بسته" },
  packet_loss_percent: { en: "Packet Loss %", fa: "درصد افت بسته" },
  jitter_ms: { en: "Jitter", fa: "نوسان" },
  response_time_ms: { en: "Response time", fa: "زمان پاسخ" },
  status_code: { en: "Status code", fa: "کد وضعیت" },
  cpu_percent: { en: "CPU", fa: "پردازنده" },
  memory_percent: { en: "Memory", fa: "حافظه" },
  disk_percent: { en: "Disk", fa: "دیسک" },
  days_remaining: { en: "Days", fa: "روز" },
  tls_days_remaining: { en: "TLS days", fa: "روز TLS" },
  tls_certificate_days: { en: "Certificate days", fa: "روزهای گواهی" },
  uptime_percent: { en: "Uptime %", fa: "درصد سرویس‌دهی" },
  error_rate: { en: "Error rate", fa: "نرخ خطا" },
  throughput: { en: "Throughput", fa: "توان عملیاتی" },
  connections: { en: "Connections", fa: "اتصالات" },
};

const FIELD_TITLE_LABELS: Record<string, { en: string; fa: string }> = {
  count: { en: "Packet count", fa: "تعداد بسته" },
  packet_count: { en: "Packet count", fa: "تعداد بسته" },
  packet_size: { en: "Packet size", fa: "اندازه بسته" },
  packet_interval_ms: { en: "Packet interval", fa: "فاصله بسته" },
  host: { en: "Host", fa: "میزبان" },
  target: { en: "Target", fa: "هدف" },
  url: { en: "URL", fa: "آدرس" },
  domain: { en: "Domain", fa: "دامنه" },
  port: { en: "Port", fa: "پورت" },
  method: { en: "Method", fa: "متد" },
  headers: { en: "Headers", fa: "هدرها" },
  body: { en: "Body", fa: "بدنه" },
  timeout_ms: { en: "Timeout (ms)", fa: "تایم‌اوت (میلی‌ثانیه)" },
  timeout: { en: "Timeout", fa: "تایم‌اوت" },
  interval_ms: { en: "Interval (ms)", fa: "فاصله (میلی‌ثانیه)" },
  expected_status: { en: "Expected status", fa: "کد وضعیت مورد انتظار" },
  follow_redirects: { en: "Follow redirects", fa: "دنبال کردن تغییر مسیر" },
  verify_ssl: { en: "Verify SSL", fa: "اعتبارسنجی SSL" },
  verify_tls: { en: "Verify TLS", fa: "اعتبارسنجی TLS" },
  verify_hostname: { en: "Verify hostname", fa: "اعتبارسنجی نام میزبان" },
  record_type: { en: "Record type", fa: "نوع رکورد" },
  nameserver: { en: "Name server", fa: "سرور نام" },
  resolver: { en: "Resolver", fa: "تحلیل‌گر" },
  expected_value: { en: "Expected value", fa: "مقدار مورد انتظار" },
  collectors: { en: "Collectors", fa: "گردآورنده‌ها" },
  disk_paths: { en: "Disk paths", fa: "مسیرهای دیسک" },
  network_interfaces: { en: "Network interfaces", fa: "واسط‌های شبکه" },
  container_names: { en: "Container names", fa: "نام کانتینرها" },
  include_stopped: { en: "Include stopped", fa: "شامل متوقف‌شده" },
  namespaces: { en: "Namespaces", fa: "فضاهای نام" },
  collect_pod_metrics: { en: "Collect pod metrics", fa: "جمع‌آوری متریک پاد" },
  engine: { en: "Engine", fa: "موتور" },
  metrics: { en: "Metrics", fa: "متریک‌ها" },
  version: { en: "Version", fa: "نسخه" },
  community: { en: "Community", fa: "انجمن" },
  oids: { en: "OIDs", fa: "OIDها" },
  ip_version: { en: "IP version", fa: "نسخه IP" },
  protocol: { en: "Protocol", fa: "پروتکل" },
};

function tLabel(isFa: boolean, en: string, fa: string): string {
  return isFa ? fa : en;
}

function tFieldTitle(isFa: boolean, key: string, title?: string): string {
  if (title && FIELD_TITLE_LABELS[title]) {
    return isFa ? FIELD_TITLE_LABELS[title].fa : FIELD_TITLE_LABELS[title].en;
  }
  if (FIELD_TITLE_LABELS[key]) {
    return isFa ? FIELD_TITLE_LABELS[key].fa : FIELD_TITLE_LABELS[key].en;
  }
  return title ?? key.replace(/_/g, " ");
}

function tMetricLabel(isFa: boolean, key: string): string {
  if (METRIC_LABELS[key]) {
    return isFa ? METRIC_LABELS[key].fa : METRIC_LABELS[key].en;
  }
  return key.replace(/_/g, " ");
}

export function MonitorConfig({ resourceId, type, monitor, isFa }: Props) {
  const create = useCreateResourceMonitor(resourceId);
  const update = useUpdateResourceMonitor(resourceId);

  const schema = (type.config_schema ?? {}) as { properties?: Record<string, { type?: string; default?: number | string | boolean; title?: string; minimum?: number; maximum?: number }> };
  const hp = (type.health_parameters ?? {}) as Record<string, { default_profile?: string; warning_threshold?: number; critical_threshold?: number; unit?: string }>;

  const [enabled, setEnabled] = useState(monitor?.enabled ?? false);
  const [iv, setIv] = useState(monitor?.interval_seconds ?? 60);
  const [to, setTo] = useState(monitor?.timeout_millis ?? 5000);
  const [rt, setRt] = useState(monitor?.retries ?? 1);
  const [fields, setFields] = useState<Record<string, number | string | boolean>>(() => {
    const base = (monitor?.configuration ?? {}) as Record<string, unknown>;
    const vals: Record<string, number | string | boolean> = {};
    for (const [k, p] of Object.entries(schema.properties ?? {})) {
      const e = base[k];
      vals[k] = (typeof e === "number" || typeof e === "string" || typeof e === "boolean") ? e : p.default ?? (p.type === "boolean" ? false : 0);
    }
    return vals;
  });
  const [rules, setRules] = useState<Record<string, { warning?: number; critical?: number }>>(() => {
    const base = (monitor?.configuration ?? {}) as { health_rules?: Record<string, { warning?: number; critical?: number }> };
    const r: Record<string, { warning?: number; critical?: number }> = {};
    for (const [k, d] of Object.entries(hp)) {
      const e = base.health_rules?.[k];
      r[k] = { warning: e?.warning ?? d.warning_threshold, critical: e?.critical ?? d.critical_threshold };
    }
    return r;
  });
  const [pending, setPending] = useState(false);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault(); setPending(true);
    try {
      const cfg = { ...fields, health_rules: rules };
      const base: MonitorInput = {
        monitor_type_id: type.id, name: monitor?.name ?? type.name, enabled,
        interval_seconds: iv, timeout_millis: to, retries: rt, configuration: cfg,
        severity: monitor?.severity ?? "warning",
      };
      if (monitor) await update.mutateAsync({ id: monitor.id, ...base });
      else await create.mutateAsync(base);
      toast.success(tLabel(isFa, "Saved", "ذخیره شد"));
    } catch { toast.error(tLabel(isFa, "Error", "خطا")); }
    finally { setPending(false); }
  };

  return (
    <Card className="rounded-2xl border-2 border-border h-full">
      <CardHeader className="border-b-2 border-border pb-3.5">
        <div className="flex items-center justify-between gap-3">
          <div className="flex min-w-0 items-center gap-3">
            <span className="flex size-10 shrink-0 items-center justify-center rounded-xl bg-primary text-primary-foreground">
              <MonitorTypeIcon type={type.name} className="size-5" />
            </span>
            <div className="min-w-0">
              <CardTitle className="truncate text-sm font-extrabold">{type.name} {tLabel(isFa, "Settings", "تنظیمات")}</CardTitle>
              <CardDescription className="mt-0.5 text-xs font-semibold">{type.description}</CardDescription>
            </div>
          </div>
          <div className="flex shrink-0 items-center gap-2">
            <span className="text-xs font-bold text-muted-foreground">{tLabel(isFa, "On", "فعال")}</span>
            <Switch checked={enabled} onCheckedChange={setEnabled} />
          </div>
        </div>
      </CardHeader>
      <CardContent className="pt-5">
        <form onSubmit={submit} className="flex flex-col gap-5">
          <section>
            <div className="mb-3 flex items-center gap-2.5">
              <span className="flex size-8 shrink-0 items-center justify-center rounded-lg bg-blue-600 text-white"><Clock className="size-4" /></span>
              <h3 className="text-sm font-extrabold">{tLabel(isFa, "Execution", "اجرا")}</h3>
            </div>
            <div className="grid grid-cols-3 gap-3">
              <NumInput label={tLabel(isFa, "Interval (s)", "بازه (ثانیه)")} v={iv} onChange={setIv} min={10} />
              <NumInput label={tLabel(isFa, "Timeout (ms)", "تایم‌اوت (میلی‌ثانیه)")} v={to} onChange={setTo} min={100} max={60000} />
              <NumInput label={tLabel(isFa, "Retries", "تلاش مجدد")} v={rt} onChange={setRt} min={0} max={5} />
            </div>
            {Object.keys(schema.properties ?? {}).length > 0 && (
              <div className="mt-3 flex flex-wrap gap-3">
                {Object.entries(schema.properties ?? {}).map(([k, p]) => p.type === "boolean" ? (
                  <div key={k} className="flex items-center gap-2 pt-1">
                    <Label className="text-xs font-bold">{tFieldTitle(isFa, k, p.title)}</Label>
                    <Switch checked={Boolean(fields[k])} onCheckedChange={(v) => setFields((pr) => ({ ...pr, [k]: v }))} />
                  </div>
                ) : (
                  <NumInput key={k} label={tFieldTitle(isFa, k, p.title)} v={Number(fields[k] ?? 0)} onChange={(v) => setFields((pr) => ({ ...pr, [k]: v }))} min={p.minimum} max={p.maximum} />
                ))}
              </div>
            )}
            {Object.keys(schema.properties ?? {}).length > 2 && (
              <div className="mt-3 flex flex-wrap gap-3">
                {Object.entries(schema.properties ?? {}).map(([k, p]) => p.type !== "boolean" && (
                  <NumInput key={`c-${k}`} label={tFieldTitle(isFa, k, p.title)} v={Number(fields[k] ?? 0)} onChange={(v) => setFields((pr) => ({ ...pr, [k]: v }))} min={p.minimum} max={p.maximum} />
                ))}
              </div>
            )}
          </section>

          {Object.keys(rules).length > 0 && (
            <section>
              <div className="mb-4 flex items-center gap-2.5">
                <span className="flex size-8 shrink-0 items-center justify-center rounded-lg bg-rose-600 text-white"><Heart className="size-4" /></span>
                <div>
                  <h3 className="text-sm font-extrabold">{tLabel(isFa, "Health Rules", "قوانین سلامت")}</h3>
                  <p className="text-[11px] font-semibold text-muted-foreground">{tLabel(isFa, "Define health state conditions", "تعریف شرایط هر سطح")}</p>
                </div>
              </div>

              <div className="flex flex-col gap-4">
                {Object.entries(rules).map(([k, r]) => (
                  <div key={k} className="flex items-start gap-6">
                    <div className="w-28 shrink-0 pt-2">
                      <p className="text-sm font-bold">{tMetricLabel(isFa, k)}</p>
                      {hp[k]?.unit && <p className="text-[11px] text-muted-foreground">{hp[k].unit}</p>}
                    </div>
                    <div className="flex flex-1 items-center gap-3">
                      <div className="flex flex-1 items-center gap-2 rounded-lg bg-amber-500/5 px-3 py-2">
                        <span className="size-2 shrink-0 rounded-full bg-amber-500" />
                        <span className="text-xs font-semibold text-amber-700 dark:text-amber-400">{tLabel(isFa, "Warning", "هشدار")}</span>
                        <Input type="number" value={r.warning ?? ""} placeholder="—" dir="ltr"
                          onChange={(e) => setRules((p) => ({ ...p, [k]: { ...p[k], warning: e.target.value === "" ? undefined : Number(e.target.value) } }))}
                          className="ml-auto h-8 w-24 border-0 bg-transparent text-sm font-bold shadow-none focus-visible:ring-0" />
                      </div>
                      <div className="flex flex-1 items-center gap-2 rounded-lg bg-red-500/5 px-3 py-2">
                        <span className="size-2 shrink-0 rounded-full bg-red-500" />
                        <span className="text-xs font-semibold text-red-700 dark:text-red-400">{tLabel(isFa, "Critical", "بحرانی")}</span>
                        <Input type="number" value={r.critical ?? ""} placeholder="—" dir="ltr"
                          onChange={(e) => setRules((p) => ({ ...p, [k]: { ...p[k], critical: e.target.value === "" ? undefined : Number(e.target.value) } }))}
                          className="ml-auto h-8 w-24 border-0 bg-transparent text-sm font-bold shadow-none focus-visible:ring-0" />
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </section>
          )}

          <div className="flex justify-end border-t-2 border-border pt-4">
            <Button type="submit" disabled={pending} size="lg" className="min-w-32 font-extrabold">
              {pending ? tLabel(isFa, "Saving...", "در حال ذخیره...") : tLabel(isFa, "Save", "ذخیره")}
            </Button>
          </div>
        </form>
      </CardContent>
    </Card>
  );
}

function NumInput({ label, v, onChange, min, max }: { label: string; v: number; onChange: (v: number) => void; min?: number; max?: number }) {
  return (
    <div className="flex flex-col gap-1">
      <Label className="text-[11px] font-extrabold text-muted-foreground">{label}</Label>
      <Input type="number" min={min} max={max} value={v} onChange={(e) => onChange(Number(e.target.value))} dir="ltr" className="font-bold" />
    </div>
  );
}
