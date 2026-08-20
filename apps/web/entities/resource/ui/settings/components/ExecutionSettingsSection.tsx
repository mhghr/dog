"use client";

import { Input } from "@/shared/ui/input";
import { Label } from "@/shared/ui/label";
import { localizeUnit } from "../monitoring-schema";

export interface ExecutionSettingsValues {
  intervalSeconds: number;
  timeoutMillis: number;
  retries: number;
  retryDelayMs: number;
}

interface ExecutionSettingsSectionProps {
  value: ExecutionSettingsValues;
  isFa: boolean;
  onChange: (next: ExecutionSettingsValues) => void;
}

// Allowed check-interval range in seconds.
export const INTERVAL_MIN_SECONDS = 3;
export const INTERVAL_MAX_SECONDS = 120;

// Shared, monitoring-type-agnostic execution settings used by every
// monitoring type: check interval and timeout. Units are shown in parentheses
// inside the labels, localized.
export function ExecutionSettingsSection({
  value,
  isFa,
  onChange,
}: ExecutionSettingsSectionProps) {
  const t = (en: string, fa: string) => (isFa ? fa : en);

  const setInterval = (next: number) => {
    const clamped = Number.isNaN(next) ? INTERVAL_MIN_SECONDS : next;
    onChange({
      ...value,
      intervalSeconds: Math.min(INTERVAL_MAX_SECONDS, Math.max(INTERVAL_MIN_SECONDS, clamped)),
    });
  };

  return (
    <>
      {/* Check interval */}
      <div className="flex flex-col gap-2">
        <Label className="text-xs font-medium text-muted-foreground">
          {t("Check interval", "بازه زمانی")} ({localizeUnit("s", isFa)})
        </Label>
        <Input
          type="number"
          value={value.intervalSeconds}
          min={INTERVAL_MIN_SECONDS}
          max={INTERVAL_MAX_SECONDS}
          className="h-10 w-full"
          dir="ltr"
          onChange={(e) => setInterval(Number(e.target.value))}
        />
      </div>

      {/* Timeout */}
      <div className="flex flex-col gap-2">
        <Label className="text-xs font-medium text-muted-foreground">
          {t("Timeout", "تایم‌اوت")} ({localizeUnit("ms", isFa)})
        </Label>
        <Input
          type="number"
          value={value.timeoutMillis}
          min={100}
          max={600000}
          className="h-10 w-full"
          dir="ltr"
          onChange={(e) => onChange({ ...value, timeoutMillis: Number(e.target.value) })}
        />
      </div>
    </>
  );
}
