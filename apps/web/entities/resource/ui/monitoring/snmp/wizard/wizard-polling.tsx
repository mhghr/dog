"use client";

import { Input } from "@/shared/ui/input";
import { Label } from "@/shared/ui/label";
import { cn } from "@/shared/utils/cn";

const INTERVAL_PRESETS = [
  { seconds: 30, label: "30s" },
  { seconds: 60, label: "1m" },
  { seconds: 300, label: "5m" },
];

export interface SnmpExecutionState {
  intervalSeconds: number;
  timeoutMillis: number;
  retries: number;
  discoveryIntervalSeconds: number;
}

export function SnmpPollingStep({
  execution,
  isFa,
  onChange,
}: {
  execution: SnmpExecutionState;
  isFa: boolean;
  onChange: (execution: SnmpExecutionState) => void;
}) {
  const t = (en: string, fa: string) => (isFa ? fa : en);
  const isPreset = INTERVAL_PRESETS.some((p) => p.seconds === execution.intervalSeconds);

  return (
    <div className="flex flex-col gap-5">
      <div>
        <h3 className="text-sm font-semibold text-foreground">{t("Polling Settings", "تنظیمات Polling")}</h3>
        <p className="mt-0.5 text-xs text-muted-foreground">
          {t(
            "Polling is asynchronous — a slow or unreachable device never blocks the collector.",
            "Polling غیرهمزمان است — دستگاه کند یا در دسترس نبوده هرگز کلکتور را مسدود نمی‌کند.",
          )}
        </p>
      </div>

      <div className="grid grid-cols-1 gap-5 sm:grid-cols-2">
        <div className="flex flex-col gap-2">
          <Label className="text-xs font-medium text-muted-foreground">{t("Polling interval", "بازه Polling")}</Label>
          <div className="flex flex-wrap items-center gap-1.5">
            {INTERVAL_PRESETS.map((preset) => (
              <button
                key={preset.seconds}
                type="button"
                onClick={() => onChange({ ...execution, intervalSeconds: preset.seconds })}
                className={cn(
                  "rounded-lg border px-3 py-1.5 text-xs font-medium transition-colors",
                  execution.intervalSeconds === preset.seconds
                    ? "border-primary/60 bg-primary/10 text-primary"
                    : "border-border/60 text-muted-foreground hover:border-primary/40",
                )}
              >
                {preset.label}
              </button>
            ))}
            <div className="flex items-center gap-1">
              <button
                type="button"
                onClick={() => onChange({ ...execution, intervalSeconds: isPreset ? 120 : execution.intervalSeconds })}
                className={cn(
                  "rounded-lg border px-3 py-1.5 text-xs font-medium transition-colors",
                  !isPreset ? "border-primary/60 bg-primary/10 text-primary" : "border-border/60 text-muted-foreground",
                )}
              >
                {t("Custom", "سفارشی")}
              </button>
              {!isPreset && (
                <Input
                  type="number"
                  value={execution.intervalSeconds}
                  min={10}
                  max={3600}
                  className="h-8 w-20 text-xs"
                  dir="ltr"
                  onChange={(e) => onChange({ ...execution, intervalSeconds: Number(e.target.value) })}
                />
              )}
            </div>
          </div>
        </div>

        <div className="grid grid-cols-3 gap-3">
          <div className="flex flex-col gap-1.5">
            <Label className="text-xs font-medium text-muted-foreground">{t("Timeout (ms)", "تایم‌اوت (میلی‌ثانیه)")}</Label>
            <Input type="number" value={execution.timeoutMillis} min={100} step={100} className="h-10" dir="ltr" onChange={(e) => onChange({ ...execution, timeoutMillis: Number(e.target.value) })} />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label className="text-xs font-medium text-muted-foreground">{t("Retries", "تلاش مجدد")}</Label>
            <Input type="number" value={execution.retries} min={0} max={5} className="h-10" dir="ltr" onChange={(e) => onChange({ ...execution, retries: Number(e.target.value) })} />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label className="text-xs font-medium text-muted-foreground">{t("Discovery refresh", "بازه کشف")}</Label>
            <Input type="number" value={Math.round(execution.discoveryIntervalSeconds / 3600)} min={1} className="h-10" dir="ltr" onChange={(e) => onChange({ ...execution, discoveryIntervalSeconds: Number(e.target.value) * 3600 })} />
          </div>
        </div>
      </div>
    </div>
  );
}
