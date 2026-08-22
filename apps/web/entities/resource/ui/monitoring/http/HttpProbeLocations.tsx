"use client";

import { useState } from "react";

import { Card, CardContent, CardHeader, CardTitle } from "@/shared/ui/card";
import { StatusBadge } from "@/design-system/components";
import { Sheet, SheetContent, SheetHeader, SheetTitle } from "@/shared/ui/sheet";
import { Skeleton } from "@/shared/ui/skeleton";
import { formatRelativeTime } from "@/shared/utils/formatters";
import { healthLabel, healthTone } from "./probe-health-ui";
import type { HttpProbeHealth } from "./http-metrics";

interface HttpProbeLocationsProps {
  probeHealth: HttpProbeHealth[];
  isFa: boolean;
  isLoading: boolean;
}

// Probe names with a detail drawer per probe. No map tab — clicking a probe
// name opens the per-probe performance breakdown.
export function HttpProbeLocations({ probeHealth, isFa, isLoading }: HttpProbeLocationsProps) {
  const t = (en: string, fa: string) => (isFa ? fa : en);
  const [selected, setSelected] = useState<HttpProbeHealth | null>(null);

  return (
    <Card variant="bordered" className="h-full shadow-subtle">
      <CardHeader className="px-5 pt-4">
        <CardTitle className="text-sm font-semibold text-foreground">
          {t("Probe Locations", "موقعیت پراب‌ها")}
        </CardTitle>
      </CardHeader>

      <CardContent className="px-4 pb-4 pt-1">
        {isLoading ? (
          <Skeleton className="h-40 w-full rounded-lg" />
        ) : probeHealth.length === 0 ? (
          <p className="py-8 text-center text-sm text-muted-foreground">
            {t("No probe has recent data", "هیچ پرابی داده اخیر ندارد")}
          </p>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full border-collapse text-left">
              <thead>
                <tr className="border-b border-border/60 text-[11px] font-semibold uppercase tracking-[0.05em] text-muted-foreground">
                  <th className="py-2 pe-3 text-start font-semibold">{t("Probe", "پراب")}</th>
                  <th className="py-2 pe-3 font-semibold">{t("Status", "وضعیت")}</th>
                  <th className="py-2 pe-3 text-right font-semibold">{t("Response", "پاسخ")}</th>
                  <th className="py-2 pe-3 text-right font-semibold">{t("DNS", "DNS")}</th>
                  <th className="py-2 pe-3 text-right font-semibold">{t("TCP", "TCP")}</th>
                  <th className="py-2 pe-3 text-right font-semibold">{t("TLS", "TLS")}</th>
                  <th className="py-2 pe-3 text-right font-semibold">{t("TTFB", "TTFB")}</th>
                  <th className="py-2 pe-3 text-right font-semibold">{t("HTTP", "HTTP")}</th>
                  <th className="py-2 pe-3 text-right font-semibold">{t("Size", "حجم")}</th>
                  <th className="py-2 pe-3 text-right font-semibold">{t("Uptime", "آپ‌تایم")}</th>
                  <th className="py-2 text-right font-semibold">{t("Last check", "آخرین بررسی")}</th>
                </tr>
              </thead>
              <tbody>
                {probeHealth.map((stat) => (
                  <HttpProbeRow key={stat.probeId} stat={stat} isFa={isFa} onSelect={setSelected} />
                ))}
              </tbody>
            </table>
          </div>
        )}
      </CardContent>

      <ProbeDetailSheet
        probe={selected}
        isFa={isFa}
        onOpenChange={(open) => {
          if (!open) setSelected(null);
        }}
      />
    </Card>
  );
}

function BreakdownRow({ label, value, unit = "ms" }: { label: string; value: number | null; unit?: string }) {
  return (
    <div className="flex items-center justify-between gap-2 text-xs">
      <span className="text-muted-foreground">{label}</span>
      <span className="font-medium tabular-nums text-foreground" dir="ltr">
        {value == null ? "—" : `${Math.round(value)} ${unit}`}
      </span>
    </div>
  );
}

function msCell(value: number | null): string {
  return value == null ? "—" : `${Math.round(value)} ms`;
}

function kbCell(value: number | null): string {
  return value == null ? "—" : `${Math.round(value / 1024)} KB`;
}

function percentCell(value: number | null): string {
  return value == null ? "—" : `${Math.round(value)}%`;
}

const numericCellClass =
  "py-2.5 pe-3 text-right tabular-nums text-[13px] text-muted-foreground";

function HttpProbeRow({
  stat,
  isFa,
  onSelect,
}: {
  stat: HttpProbeHealth;
  isFa: boolean;
  onSelect: (probe: HttpProbeHealth) => void;
}) {
  return (
    <tr
      onClick={() => onSelect(stat)}
      className="cursor-pointer border-b border-border/40 transition-colors last:border-0 hover:bg-muted/30"
    >
      <td className="py-2.5 pe-3 text-sm font-medium" dir="auto">{stat.location}</td>
      <td className="py-2.5 pe-3">
        <StatusBadge tone={healthTone(stat.health)} label={healthLabel(stat.health, isFa)} />
      </td>
      <td className={`${numericCellClass} ltr`} dir="ltr">{msCell(stat.responseTimeMs)}</td>
      <td className={`${numericCellClass} ltr`} dir="ltr">{msCell(stat.breakdown.dns)}</td>
      <td className={`${numericCellClass} ltr`} dir="ltr">{msCell(stat.breakdown.connect)}</td>
      <td className={`${numericCellClass} ltr`} dir="ltr">{msCell(stat.breakdown.tls)}</td>
      <td className={`${numericCellClass} ltr`} dir="ltr">{msCell(stat.breakdown.ttfb)}</td>
      <td className={`${numericCellClass} ltr`} dir="ltr">{stat.statusCode ?? "—"}</td>
      <td className={`${numericCellClass} ltr`} dir="ltr">{kbCell(stat.responseSize)}</td>
      <td className={`${numericCellClass} ltr`} dir="ltr">{percentCell(stat.availability)}</td>
      <td className="py-2.5 text-right text-xs text-muted-foreground">
        {stat.lastCheckedAt ? formatRelativeTime(stat.lastCheckedAt, isFa ? "fa" : "en") : "—"}
      </td>
    </tr>
  );
}

function ProbeDetailSheet({
  probe,
  isFa,
  onOpenChange,
}: {
  probe: HttpProbeHealth | null;
  isFa: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const t = (en: string, fa: string) => (isFa ? fa : en);

  return (
    <Sheet open={probe != null} onOpenChange={onOpenChange}>
      <SheetContent side="right" className="w-full sm:max-w-md overflow-y-auto">
        <SheetHeader>
          <SheetTitle className="flex items-center gap-2">
            {probe?.location ?? ""}
            {probe && (
              <StatusBadge tone={healthTone(probe.health)} label={healthLabel(probe.health, isFa)} />
            )}
          </SheetTitle>
        </SheetHeader>

        {probe && (
          <div className="mt-4 flex flex-col gap-5 px-4">
            <section className="flex flex-col gap-2">
              <h4 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                {t("Performance", "عملکرد")}
              </h4>
              <div className="rounded-lg border border-border/50 p-3">
                <BreakdownRow label={t("Response time", "زمان پاسخ")} value={probe.responseTimeMs} />
                <BreakdownRow label={t("DNS", "DNS")} value={probe.breakdown.dns} />
                <BreakdownRow label={t("TCP connect", "اتصال TCP")} value={probe.breakdown.connect} />
                <BreakdownRow label={t("TLS", "TLS")} value={probe.breakdown.tls} />
                <BreakdownRow label={t("TTFB", "TTFB")} value={probe.breakdown.ttfb} />
                <BreakdownRow label={t("Download", "دانلود")} value={probe.breakdown.download} />
              </div>
            </section>

            <section className="flex flex-col gap-2">
              <h4 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                {t("HTTP Response", "پاسخ HTTP")}
              </h4>
              <div className="rounded-lg border border-border/50 p-3">
                <BreakdownRow label={t("Status", "وضعیت")} value={probe.statusCode} unit="" />
                <BreakdownRow
                  label={t("Response size", "اندازه پاسخ")}
                  value={probe.responseSize == null ? null : probe.responseSize / 1024}
                  unit="KB"
                />
              </div>
            </section>

            {probe.errorMessage && (
              <section className="flex flex-col gap-2">
                <h4 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                  {t("Error", "خطا")}
                </h4>
                <div className="rounded-lg border border-destructive/25 bg-destructive/5 p-3 text-xs text-destructive">
                  <p className="font-medium">{probe.errorCode ?? "error"}</p>
                  <p className="mt-1 text-muted-foreground">{probe.errorMessage}</p>
                </div>
              </section>
            )}

            <section className="flex flex-col gap-2">
              <h4 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                {t("Last check", "آخرین بررسی")}
              </h4>
              <p className="text-xs text-muted-foreground">
                {probe.lastCheckedAt ? formatRelativeTime(probe.lastCheckedAt, isFa ? "fa" : "en") : "—"}
              </p>
            </section>
          </div>
        )}
      </SheetContent>
    </Sheet>
  );
}
