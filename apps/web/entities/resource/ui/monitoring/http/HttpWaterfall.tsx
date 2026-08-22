"use client";

import { useMemo } from "react";

import { Card, CardContent, CardHeader, CardTitle } from "@/shared/ui/card";
import { hexToRgba } from "@/shared/ui/charts/chart-config";
import {
  waterfallPhasesOf,
  type HttpWaterfallPhase,
} from "./http-metrics";
import type { ProbeResult } from "@/entities/monitor/model/result";

// Renders the timing waterfall of a single HTTP check: DNS → Connect → TLS →
// TTFB → Download, each drawn as a segment on the same time axis so the phase
// that dominates the total is obvious. Duration is the pooled phase total;
// the gap between the phases and the full response time is left implicit.
export function HttpWaterfall({
  result,
  isFa,
}: {
  result: ProbeResult | null;
  isFa: boolean;
}) {
  const t = (en: string, fa: string) => (isFa ? fa : en);

  const { phases, total } = useMemo(() => {
    const phases = result ? waterfallPhasesOf(result) : [];
    const total = result?.metrics?.response_time_ms ?? result?.duration_millis ?? 0;
    return { phases, total: typeof total === "number" ? total : 0 };
  }, [result]);

  if (!result) {
    return (
      <Card variant="bordered" className="h-full shadow-subtle">
        <CardHeader className="px-5 pt-4">
          <CardTitle className="text-sm font-semibold text-foreground">
            {t("Timing Breakdown", "شکستن زمان‌بندی")}
          </CardTitle>
        </CardHeader>
        <CardContent className="px-4 pb-4">
          <p className="py-8 text-center text-sm text-muted-foreground">
            {t("Click a point on the chart to inspect one check", "برای بررسی یک چک، روی نقطه‌ای در نمودار کلیک کنید")}
          </p>
        </CardContent>
      </Card>
    );
  }

  const maxMs = Math.max(total, ...phases.map((p) => p.ms ?? 0), 1);

  return (
    <Card variant="bordered" className="h-full shadow-subtle">
      <CardHeader className="px-5 pt-4">
        <div className="flex items-center justify-between gap-3">
          <CardTitle className="text-sm font-semibold text-foreground">
            {t("Timing Breakdown", "شکستن زمان‌بندی")}
          </CardTitle>
          <span className="text-xs tabular-nums text-muted-foreground" dir="ltr">
            {t("Total", "مجموع")} <span className="font-semibold text-foreground">{Math.round(total)} ms</span>
          </span>
        </div>
      </CardHeader>
      <CardContent className="flex flex-col gap-2 px-4 pb-4">
        <PhaseRow phase={{ key: "total", label: { en: "Total", fa: "مجموع" }, ms: total, color: "#8B5CF6" }} maxMs={maxMs} isFa={isFa} />
        <div className="my-1 border-t border-border/40" />
        {phases.map((phase) => (
          <PhaseRow key={phase.key} phase={phase} maxMs={maxMs} isFa={isFa} />
        ))}
      </CardContent>
    </Card>
  );
}

function PhaseRow({ phase, maxMs, isFa }: { phase: HttpWaterfallPhase; maxMs: number; isFa: boolean }) {
  const pct = phase.ms != null ? Math.max((phase.ms / maxMs) * 100, 0.6) : 0;
  const label = phase.ms != null ? `${Math.round(phase.ms)} ms` : isFa ? "—" : "—";
  return (
    <div className="flex items-center gap-2.5">
      <span className="w-20 shrink-0 truncate text-xs text-muted-foreground">
        {isFa ? phase.label.fa : phase.label.en}
      </span>
      <div className="h-3 min-w-0 flex-1 overflow-hidden rounded-full bg-muted/40">
        {phase.ms != null && (
          <div
            className="h-full rounded-full transition-[width] duration-500 ease-out"
            style={{
              width: `${pct}%`,
              background: `linear-gradient(to right, ${hexToRgba(phase.color, 0.55)}, ${phase.color})`,
              boxShadow: `0 0 10px -1px ${hexToRgba(phase.color, 0.7)}`,
            }}
          />
        )}
      </div>
      <span className="w-16 shrink-0 text-right text-xs tabular-nums text-foreground" dir="ltr">
        {label}
      </span>
    </div>
  );
}
