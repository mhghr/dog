"use client";

import { Card, CardContent, CardHeader, CardTitle } from "@/shared/ui/card";
import { StatusBadge } from "@/design-system/components";
import { formatRelativeTime } from "@/shared/utils/formatters";
import { healthTone, healthLabel } from "./probe-health-ui";
import type { HttpProbeHealth } from "./http-metrics";

interface HttpAvailabilityHistoryProps {
  probeHealth: HttpProbeHealth[];
  isFa: boolean;
}

// Recent checks across all probes — one row per probe's latest check.
export function HttpAvailabilityHistory({ probeHealth, isFa }: HttpAvailabilityHistoryProps) {
  const t = (en: string, fa: string) => (isFa ? fa : en);

  const rows = [...probeHealth].sort((a, b) =>
    String(b.lastCheckedAt ?? "").localeCompare(String(a.lastCheckedAt ?? "")),
  );

  if (rows.length === 0) return null;

  return (
    <Card variant="bordered" className="shadow-subtle">
      <CardHeader className="px-5 pt-4">
        <CardTitle className="text-sm font-semibold text-foreground">
          {t("Availability History", "تاریخچه دسترس‌پذیری")}
        </CardTitle>
      </CardHeader>
      <CardContent className="overflow-x-auto px-4 pb-4">
        <table className="w-full min-w-[560px] border-collapse text-left">
          <thead>
            <tr className="border-b border-border/60 text-[11px] font-semibold uppercase tracking-[0.05em] text-muted-foreground">
              <th className="px-1 pb-2.5 font-semibold">{t("Probe", "پراب")}</th>
              <th className="px-1 pb-2.5 font-semibold">{t("Status", "وضعیت")}</th>
              <th className="px-1 pb-2.5 text-right font-semibold">{t("Latency", "تأخیر")}</th>
              <th className="px-1 pb-2.5 text-right font-semibold">{t("HTTP code", "کد HTTP")}</th>
              <th className="px-1 pb-2.5 text-right font-semibold">{t("Error", "خطا")}</th>
              <th className="px-1 pb-2.5 text-right font-semibold">{t("Time", "زمان")}</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((stat) => (
              <tr key={stat.probeId} className="border-b border-border/40 last:border-0">
                <td className="px-1 py-2.5 text-sm font-medium" dir="auto">{stat.location}</td>
                <td className="px-1 py-2.5">
                  <StatusBadge tone={healthTone(stat.health)} label={healthLabel(stat.health, isFa)} />
                </td>
                <td className="px-1 py-2.5 text-right tabular-nums text-[13px] text-muted-foreground" dir="ltr">
                  {stat.responseTimeMs == null ? "—" : `${Math.round(stat.responseTimeMs)} ms`}
                </td>
                <td className="px-1 py-2.5 text-right tabular-nums text-[13px] text-muted-foreground" dir="ltr">
                  {stat.statusCode ?? "—"}
                </td>
                <td className="px-1 py-2.5 text-right text-xs text-muted-foreground" dir="ltr">
                  {stat.errorCode ?? "—"}
                </td>
                <td className="px-1 py-2.5 text-right text-xs text-muted-foreground">
                  {formatRelativeTime(stat.lastCheckedAt, "en")}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </CardContent>
    </Card>
  );
}
