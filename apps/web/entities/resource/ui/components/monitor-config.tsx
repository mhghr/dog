"use client";

import { useState } from "react";
import { toast } from "sonner";
import { Activity, Clock, Info } from "lucide-react";
import { Slider as SliderPrimitive, Direction } from "radix-ui";

import { Button } from "@/shared/ui/button";
import { Input } from "@/shared/ui/input";
import { Label } from "@/shared/ui/label";
import { Switch } from "@/shared/ui/switch";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/shared/ui/tooltip";
import { useCreateResourceMonitor, useUpdateResourceMonitor } from "@/entities/resource/hooks/use-resource";
import type { MonitorTypeDef } from "@/entities/resource/model/types";
import type { Monitor, MonitorInput } from "@/entities/resource/hooks/types";
import { apiErrorMessage } from "@/shared/api/error-message";
import { MonitorTypeIcon } from "./monitor-type-icon";

interface Props { resourceId: string; type: MonitorTypeDef; monitor: Monitor | undefined; isFa: boolean; }

const METRIC_LABELS: Record<string, { en: string; fa: string }> = {
  latency_ms: { en: "Latency", fa: "تأخیر" },
  rtt_ms: { en: "RTT", fa: "تأخیر رفت‌وبرگشت" },
  packet_loss: { en: "Packet Loss", fa: "افت بسته" },
  jitter_ms: { en: "Jitter", fa: "نوسان" },
  cpu_percent: { en: "CPU", fa: "پردازنده" },
  memory_percent: { en: "Memory", fa: "حافظه" },
  disk_percent: { en: "Disk", fa: "دیسک" },
  days_remaining: { en: "Days", fa: "روز" },
  response_time_ms: { en: "Response time", fa: "زمان پاسخ" },
};

const METRIC_UNITS: Record<string, string> = {
  latency_ms: "ms", rtt_ms: "ms", jitter_ms: "ms", response_time_ms: "ms",
  packet_loss: "%", cpu_percent: "%", memory_percent: "%", disk_percent: "%",
  days_remaining: "day",
};

const METRIC_HINTS: Record<string, { en: string; fa: string }> = {
  latency_ms: { en: "Round-trip delay", fa: "تأخیر رفت‌وبرگشت بسته‌ها" },
  packet_loss: { en: "Percentage of lost packets", fa: "درصد بسته‌های از دست رفته" },
  jitter_ms: { en: "Delay variation", fa: "نوسان در تأخیر" },
  cpu_percent: { en: "CPU usage", fa: "میزان استفاده از پردازنده" },
  memory_percent: { en: "Memory usage", fa: "میزان استفاده از حافظه" },
  disk_percent: { en: "Disk usage", fa: "میزان استفاده از دیسک" },
  days_remaining: { en: "Days until expiry", fa: "روز تا انقضا" },
};

const FIELD_TITLE_LABELS: Record<string, { en: string; fa: string }> = {
  count: { en: "Packet count", fa: "تعداد بسته" },
  packet_size: { en: "Packet size", fa: "اندازه بسته" },
  host: { en: "Host", fa: "میزبان" },
  target: { en: "Target", fa: "هدف" },
  url: { en: "URL", fa: "آدرس" },
  domain: { en: "Domain", fa: "دامنه" },
  port: { en: "Port", fa: "پورت" },
  method: { en: "Method", fa: "متد" },
  timeout_ms: { en: "Timeout (ms)", fa: "تایم‌اوت (میلی‌ثانیه)" },
  interval_ms: { en: "Interval (ms)", fa: "فاصله (میلی‌ثانیه)" },
  expected_status: { en: "Expected status", fa: "کد وضعیت مورد انتظار" },
  follow_redirects: { en: "Follow redirects", fa: "دنبال کردن تغییر مسیر" },
  verify_ssl: { en: "Verify SSL", fa: "اعتبارسنجی SSL" },
};

const TYPE_NAME_FA: Record<string, string> = {
  Ping: "پینگ", "HTTP Check": "بررسی HTTP", "TCP Port": "پورت TCP",
  "DNS Resolution": "تفکیک DNS", "SSL Certificate": "گواهی SSL",
  "Domain Expiry": "انقضای دامنه", "Host Metrics": "متریک‌های میزبان",
};

const INPUT_HINTS: Record<string, { en: string; fa: string }> = {
  interval: { en: "Seconds between checks", fa: "فاصله زمانی بین بررسی‌ها به ثانیه" },
  timeout: { en: "Max wait for response (ms)", fa: "حداکثر انتظار برای پاسخ (میلی‌ثانیه)" },
  retries: { en: "Retry attempts on failure", fa: "تلاش مجدد در صورت خطا" },
};

function tLabel(isFa: boolean, en: string, fa: string): string { return isFa ? fa : en; }
function tTypeName(isFa: boolean, name: string): string { return isFa && TYPE_NAME_FA[name] ? TYPE_NAME_FA[name] : name; }

function tFieldTitle(isFa: boolean, key: string, title?: string): string {
  if (title && FIELD_TITLE_LABELS[title]) return isFa ? FIELD_TITLE_LABELS[title].fa : FIELD_TITLE_LABELS[title].en;
  if (FIELD_TITLE_LABELS[key]) return isFa ? FIELD_TITLE_LABELS[key].fa : FIELD_TITLE_LABELS[key].en;
  return title ?? key.replace(/_/g, " ");
}

function tMetricLabel(isFa: boolean, key: string): string {
  if (METRIC_LABELS[key]) return isFa ? METRIC_LABELS[key].fa : METRIC_LABELS[key].en;
  return key.replace(/_/g, " ");
}

function tHint(isFa: boolean, key: string): string | undefined { const h = INPUT_HINTS[key]; return h ? (isFa ? h.fa : h.en) : undefined; }
function tMetricHint(isFa: boolean, key: string): string | undefined { const h = METRIC_HINTS[key]; return h ? (isFa ? h.fa : h.en) : undefined; }

function InfoTip({ hint }: { hint: string }) {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span className="inline-flex cursor-help text-muted-foreground/40 hover:text-muted-foreground">
          <Info className="size-3.5" />
        </span>
      </TooltipTrigger>
      <TooltipContent side="top" className="max-w-56 text-xs">{hint}</TooltipContent>
    </Tooltip>
  );
}

// ── Health slider colors ──
const HEALTH_GREEN = "#7BD88F";
const HEALTH_AMBER = "#FFC95C";
const HEALTH_RED   = "#FF8A8A";
const HEALTH_GREEN_TEXT = "#178A45";
const HEALTH_AMBER_TEXT = "#B07600";
const HEALTH_RED_TEXT   = "#D32F2F";

function HealthSlider({ isFa, value, max, step, unit, showReadout, onValueChange }: {
  isFa: boolean; value: { warning?: number; critical?: number };
  max: number; step: number; unit?: string; showReadout?: boolean;
  onValueChange: (w: number, c: number) => void;
}) {
  const w = value.warning ?? 0, c = value.critical ?? 0;
  const warn = Math.min(w, c), crit = Math.max(w, c);
  const wp = max > 0 ? (warn / max) * 100 : 0;
  const cp = max > 0 ? (crit / max) * 100 : 0;
  const fmt = (v: number) => (Number.isInteger(v) ? String(v) : v.toFixed(1));

  return (
    <div className={showReadout ? "w-full space-y-2.5" : "w-full"}>
      <Direction.Provider dir="ltr">
        <SliderPrimitive.Root
          value={[warn, crit]} min={0} max={max} step={step}
          onValueChange={(v) => { const [a, b] = v; onValueChange(Math.min(a, b), Math.max(a, b)); }}
          className="relative flex w-full touch-none items-center select-none py-0.5"
        >
          <SliderPrimitive.Track className="relative h-3.5 grow rounded-full bg-zinc-300 dark:bg-zinc-300">
            <div className="absolute inset-x-1 top-1/2 h-2 -translate-y-1/2 overflow-hidden rounded-full">
              <div className="absolute inset-y-0 left-0 rounded-l-full transition-[width] duration-200 ease-[cubic-bezier(0.23,1,0.32,1)]" style={{ width: `${wp}%`, background: HEALTH_GREEN }} />
              <div className="absolute inset-y-0 transition-[left,width] duration-200 ease-[cubic-bezier(0.23,1,0.32,1)]" style={{ left: `${wp}%`, width: `${cp - wp}%`, background: HEALTH_AMBER }} />
              <div className="absolute inset-y-0 right-0 rounded-r-full transition-[width] duration-200 ease-[cubic-bezier(0.23,1,0.32,1)]" style={{ width: `${100 - cp}%`, background: HEALTH_RED }} />
            </div>
          </SliderPrimitive.Track>
          <SliderPrimitive.Thumb
            aria-label={tLabel(isFa, "Warning threshold", "آستانه هشدار")}
            className="block size-3.5 rounded-full border-[1.5px] border-white bg-white shadow-[0_1px_3px_rgba(0,0,0,0.2),0_0_0_2.5px_#7BD88F] transition-all duration-150 ease-out hover:shadow-[0_1px_3px_rgba(0,0,0,0.25),0_0_0_4px_#7BD88F] active:scale-90 focus-visible:shadow-[0_1px_3px_rgba(0,0,0,0.25),0_0_0_4px_#7BD88F] focus-visible:outline-none"
          />
          <SliderPrimitive.Thumb
            aria-label={tLabel(isFa, "Critical threshold", "آستانه بحرانی")}
            className="block size-3.5 rounded-full border-[1.5px] border-white bg-white shadow-[0_1px_3px_rgba(0,0,0,0.2),0_0_0_2.5px_#FF8A8A] transition-all duration-150 ease-out hover:shadow-[0_1px_3px_rgba(0,0,0,0.25),0_0_0_4px_#FF8A8A] active:scale-90 focus-visible:shadow-[0_1px_3px_rgba(0,0,0,0.25),0_0_0_4px_#FF8A8A] focus-visible:outline-none"
          />
        </SliderPrimitive.Root>
      </Direction.Provider>

      {showReadout && (
        <div className="flex items-center justify-between text-[13px] font-semibold">
          <span className="tabular-nums" style={{ color: HEALTH_GREEN_TEXT }}>{tLabel(isFa, "Healthy", "سالم")} &lt; {fmt(warn)}{unit ? ` ${unit}` : ""}</span>
          <span className="tabular-nums" style={{ color: HEALTH_AMBER_TEXT }}>{tLabel(isFa, "Warning", "هشدار")} {fmt(warn)} – {fmt(crit)}{unit ? ` ${unit}` : ""}</span>
          <span className="tabular-nums" style={{ color: HEALTH_RED_TEXT }}>{tLabel(isFa, "Critical", "بحرانی")} &gt; {fmt(crit)}{unit ? ` ${unit}` : ""}</span>
        </div>
      )}
    </div>
  );
}

// ── Main ──
export function MonitorConfig({ resourceId, type, monitor, isFa }: Props) {
  const create = useCreateResourceMonitor(resourceId);
  const update = useUpdateResourceMonitor(resourceId);

  const schema = (type.config_schema ?? {}) as {
    properties?: Record<string, { type?: string; default?: number | string | boolean; title?: string; minimum?: number; maximum?: number }>;
  };
  const hp = (type.health_parameters ?? {}) as Record<string, {
    default_profile?: string; warning_threshold?: number; critical_threshold?: number; unit?: string;
  }>;

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
      toast.success(tLabel(isFa, "Saved", "ذخیره شد"), {
        description: tLabel(isFa, "Monitor settings saved", "تنظیمات مانیتور ذخیره شد"),
      });
    } catch (err) {
      const msg = apiErrorMessage(err, isFa);
      toast.error(msg.title, { description: msg.description });
    }
    finally { setPending(false); }
  };

  const resetDefaults = () => {
    setIv(60); setTo(5000); setRt(1);
    const f: Record<string, number | string | boolean> = {};
    for (const [k, p] of Object.entries(schema.properties ?? {})) f[k] = p.default ?? (p.type === "boolean" ? false : 0);
    setFields(f);
    const r: Record<string, { warning?: number; critical?: number }> = {};
    for (const [k, d] of Object.entries(hp)) r[k] = { warning: d.warning_threshold, critical: d.critical_threshold };
    setRules(r);
  };

  return (
    <div className="flex h-full flex-col rounded-xl border border-border/40 bg-white dark:bg-zinc-900">
      {/* Header row */}
      <div className="flex items-center justify-between px-12 pb-3.5 pt-5">
        <div className="flex min-w-0 items-center gap-3">
          <span className="flex size-9 shrink-0 items-center justify-center rounded-xl bg-primary/10 text-primary">
            <MonitorTypeIcon type={type.name} className="size-4" />
          </span>
          <div className="min-w-0">
            <h2 className="truncate text-[16px] font-semibold leading-snug tracking-tight text-foreground/80">
              {tTypeName(isFa, type.name)} {tLabel(isFa, "Settings", "تنظیمات")}
            </h2>
            <p className="text-[13px] leading-snug text-muted-foreground">
              {isFa ? "تنظیمات اجرایی و قوانین سلامت" : type.description}
            </p>
          </div>
        </div>
        <div className="flex shrink-0 items-center gap-2">
          <span className="text-[13px] text-muted-foreground">{tLabel(isFa, "On", "فعال")}</span>
          <Switch checked={enabled} onCheckedChange={setEnabled} />
        </div>
      </div>

      <div className="mx-12 h-px bg-border/50" />

      <div className="flex-1 overflow-auto px-12">
        <form onSubmit={submit} className="flex flex-col gap-8 pb-6 pt-7">

          {/* ── Execution ── */}
          <section>
            <h3 className="mb-3 flex items-center gap-2 px-1 text-sm font-semibold text-foreground/70">
              <span className="flex size-6 items-center justify-center rounded-md bg-primary/10 text-primary"><Clock className="size-3.5" /></span>
              {tLabel(isFa, "Execution Settings", "تنظیمات اجرا")}
            </h3>
            <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-5">
              <NumInput label={tLabel(isFa, "Interval (s)", "بازه (ثانیه)")} hint={tHint(isFa, "interval")} v={iv} onChange={setIv} min={10} />
              <NumInput label={tLabel(isFa, "Timeout (ms)", "تایم‌اوت (میلی‌ثانیه)")} hint={tHint(isFa, "timeout")} v={to} onChange={setTo} min={100} max={60000} />
              <NumInput label={tLabel(isFa, "Retries", "تلاش مجدد")} hint={tHint(isFa, "retries")} v={rt} onChange={setRt} min={0} max={5} />
              {Object.entries(schema.properties ?? {}).map(([k, p]) =>
                p.type === "boolean" ? (
                  <div key={k} className="flex items-center gap-2 pt-1">
                    <Label className="text-[13px] text-muted-foreground">{tFieldTitle(isFa, k, p.title)}</Label>
                    <Switch checked={Boolean(fields[k])} onCheckedChange={(v) => setFields((pr) => ({ ...pr, [k]: v }))} />
                  </div>
                ) : (
                  <NumInput key={k} label={tFieldTitle(isFa, k, p.title)} hint={tHint(isFa, k)} v={Number(fields[k] ?? 0)} onChange={(v) => setFields((pr) => ({ ...pr, [k]: v }))} min={p.minimum} max={p.maximum} />
                ),
              )}
            </div>
          </section>

          {/* ── Health Rules ── */}
          {Object.keys(rules).length > 0 && (
            <section>
              <h3 className="mb-3 flex items-center gap-2 px-1 text-sm font-semibold text-foreground/70">
                <span className="flex size-6 items-center justify-center rounded-md bg-rose-500/10 text-rose-500"><Activity className="size-3.5" /></span>
                {tLabel(isFa, "Health Rules", "قوانین سلامت")}
              </h3>
              <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-5">
                {Object.entries(rules).map(([k, r]) => {
                  const d = hp[k];
                  const defBase = Math.max(d?.warning_threshold ?? 0, d?.critical_threshold ?? 0);
                  const sliderMax = defBase > 0 && defBase <= 20 ? Math.ceil(defBase * 3) : defBase <= 100 ? 100 : Math.max(Math.ceil(defBase * 2.5), 100);
                  const step = sliderMax > 500 ? 5 : sliderMax > 100 ? 1 : 0.5;
                  const unit = hp[k]?.unit || METRIC_UNITS[k];
                  return (
                    <div key={k} className="col-span-2 space-y-2">
                      <div className="flex items-center gap-1.5 px-1">
                        <span className="text-sm font-medium">{tMetricLabel(isFa, k)}</span>
                        {unit && <span className="text-[12px] text-muted-foreground">({unit})</span>}
                        {tMetricHint(isFa, k) && <InfoTip hint={tMetricHint(isFa, k)!} />}
                      </div>
                      <div className="flex min-h-24 items-center px-4 py-4">
                        <div className="flex w-full flex-1 items-center">
                          <HealthSlider isFa={isFa} value={r} max={sliderMax} step={step} unit={unit} showReadout
                            onValueChange={(w, c) => setRules((p) => ({ ...p, [k]: { warning: w, critical: c } }))}
                          />
                        </div>
                      </div>
                    </div>
                  );
                })}
              </div>
            </section>
          )}

          {/* ── Actions ── */}
          <div className="flex items-center justify-end gap-2 pt-1">
            <Button type="button" variant="outline" size="sm" disabled={pending} onClick={resetDefaults}>
              {tLabel(isFa, "Defaults", "پیش‌فرض")}
            </Button>
            <Button type="submit" size="sm" disabled={pending}>
              {pending ? tLabel(isFa, "Saving...", "در حال ذخیره...") : tLabel(isFa, "Save", "ذخیره")}
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
}

function NumInput({ label, hint, v, onChange, min, max }: {
  label: string; hint?: string; v: number; onChange: (v: number) => void; min?: number; max?: number;
}) {
  return (
    <div className="flex flex-col gap-1.5">
      <div className="flex items-center gap-1">
        <Label className="text-[11px] text-muted-foreground">{label}</Label>
        {hint && <InfoTip hint={hint} />}
      </div>
      <Input type="number" min={min} max={max} value={v} onChange={(e) => onChange(Number(e.target.value))} dir="ltr" />
    </div>
  );
}
