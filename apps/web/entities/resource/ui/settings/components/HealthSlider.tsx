"use client";

import { Slider as SliderPrimitive, Direction } from "radix-ui";

import type { HealthRuleDirection } from "../monitoring-schema";
import { localizeUnit } from "../monitoring-schema";

export interface ThresholdValue {
  warning?: number;
  critical?: number;
}

const HEALTH_GREEN = "#7BD88F";
const HEALTH_AMBER = "#FFC95C";
const HEALTH_RED = "#FF8A8A";
const HEALTH_GREEN_TEXT = "#178A45";
const HEALTH_AMBER_TEXT = "#B07600";
const HEALTH_RED_TEXT = "#D32F2F";

interface HealthSliderProps {
  value: ThresholdValue;
  max: number;
  step: number;
  unit?: string;
  direction: HealthRuleDirection;
  showReadout?: boolean;
  isFa?: boolean;
  onValueChange: (warning: number, critical: number) => void;
}

// Dual-thumb threshold slider shared by every monitoring type's health rules.
// The two handles are kept at least one step apart so they never collapse onto
// each other, and the readout flips for LOWER_IS_WORSE metrics (e.g. days
// until certificate expiry) where fewer is worse.
export function HealthSlider({
  value,
  max,
  step,
  unit,
  direction,
  showReadout,
  isFa = false,
  onValueChange,
}: HealthSliderProps) {
  const t = (en: string, fa: string) => (isFa ? fa : en);
  const healthy = t("Healthy", "سالم");
  const warning = t("Warning", "هشدار");
  const critical = t("Critical", "بحرانی");
  const unitLabel = localizeUnit(unit, isFa);
  const unitSuffix = unitLabel ? ` ${unitLabel}` : "";
  const rawWarning = value.warning ?? 0;
  const rawCritical = value.critical ?? 0;
  let warn = Math.min(rawWarning, rawCritical);
  let crit = Math.max(rawWarning, rawCritical);

  const minGap = Math.max(step, 1);
  if (crit - warn < minGap) {
    const gap = minGap - (crit - warn);
    if (crit + gap <= max) {
      crit += gap;
    } else if (warn - gap >= 0) {
      warn -= gap;
    } else {
      crit = Math.min(max, warn + minGap);
    }
  }

  const wp = max > 0 ? (warn / max) * 100 : 0;
  const cp = max > 0 ? (crit / max) * 100 : 0;
  const fmt = (v: number) => (Number.isInteger(v) ? String(v) : v.toFixed(1));
  const lowerIsWorse = direction === "lower_is_worse";

  return (
    <div className={showReadout ? "w-full space-y-2.5" : "w-full"}>
      <Direction.Provider dir="ltr">
        <SliderPrimitive.Root
          value={[warn, crit]}
          min={0}
          max={max}
          step={step}
          onValueChange={(v) => {
            const [a, b] = v;
            onValueChange(Math.min(a, b), Math.max(a, b));
          }}
          className="relative flex w-full touch-none items-center select-none py-0.5"
        >
          <SliderPrimitive.Track className="relative h-3.5 grow rounded-full bg-muted">
            <div className="absolute inset-x-1 top-1/2 h-2 -translate-y-1/2 overflow-hidden rounded-full">
              <div
                className="absolute inset-y-0 left-0 rounded-l-full transition-[width] duration-200 ease-[cubic-bezier(0.23,1,0.32,1)]"
                style={{ width: `${wp}%`, background: HEALTH_GREEN }}
              />
              <div
                className="absolute inset-y-0 transition-[left,width] duration-200 ease-[cubic-bezier(0.23,1,0.32,1)]"
                style={{ left: `${wp}%`, width: `${cp - wp}%`, background: HEALTH_AMBER }}
              />
              <div
                className="absolute inset-y-0 right-0 rounded-r-full transition-[width] duration-200 ease-[cubic-bezier(0.23,1,0.32,1)]"
                style={{ width: `${100 - cp}%`, background: HEALTH_RED }}
              />
            </div>
          </SliderPrimitive.Track>
          <SliderPrimitive.Thumb
            aria-label="Warning threshold"
            className="block size-3.5 rounded-full border-[1.5px] border-white bg-white shadow-[0_1px_3px_rgba(0,0,0,0.2),0_0_0_2.5px_#7BD88F] transition-[box-shadow,scale] duration-150 ease-out hover:shadow-[0_1px_3px_rgba(0,0,0,0.25),0_0_0_4px_#7BD88F] active:scale-[0.96] focus-visible:shadow-[0_1px_3px_rgba(0,0,0,0.25),0_0_0_4px_#7BD88F] focus-visible:outline-none"
          />
          <SliderPrimitive.Thumb
            aria-label="Critical threshold"
            className="block size-3.5 rounded-full border-[1.5px] border-white bg-white shadow-[0_1px_3px_rgba(0,0,0,0.2),0_0_0_2.5px_#FF8A8A] transition-[box-shadow,scale] duration-150 ease-out hover:shadow-[0_1px_3px_rgba(0,0,0,0.25),0_0_0_4px_#FF8A8A] active:scale-[0.96] focus-visible:shadow-[0_1px_3px_rgba(0,0,0,0.25),0_0_0_4px_#FF8A8A] focus-visible:outline-none"
          />
        </SliderPrimitive.Root>
      </Direction.Provider>

      {showReadout && (
        <div className="flex flex-wrap items-center justify-between gap-x-3 text-[13px] font-semibold">
          {lowerIsWorse ? (
            <>
              <span className="tabular-nums" style={{ color: HEALTH_RED_TEXT }}>
                {critical} &lt; {fmt(crit)}{unit ? unitSuffix : ""}
              </span>
              <span className="tabular-nums" style={{ color: HEALTH_AMBER_TEXT }}>
                {warning} {fmt(crit)} – {fmt(warn)}{unit ? unitSuffix : ""}
              </span>
              <span className="tabular-nums" style={{ color: HEALTH_GREEN_TEXT }}>
                {healthy} &gt; {fmt(warn)}{unit ? unitSuffix : ""}
              </span>
            </>
          ) : (
            <>
              <span className="tabular-nums" style={{ color: HEALTH_GREEN_TEXT }}>
                {healthy} &lt; {fmt(warn)}{unit ? unitSuffix : ""}
              </span>
              <span className="tabular-nums" style={{ color: HEALTH_AMBER_TEXT }}>
                {warning} {fmt(warn)} – {fmt(crit)}{unit ? unitSuffix : ""}
              </span>
              <span className="tabular-nums" style={{ color: HEALTH_RED_TEXT }}>
                {critical} &gt; {fmt(crit)}{unit ? unitSuffix : ""}
              </span>
            </>
          )}
        </div>
      )}
    </div>
  );
}
