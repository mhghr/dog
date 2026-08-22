"use client";

import { Card, CardContent, CardHeader, CardTitle } from "@/shared/ui/card";
import { Skeleton } from "@/shared/ui/skeleton";
import { cn } from "@/shared/utils/cn";
import type { PingSummary } from "./ping-metrics";

interface StatRow {
  label: string;
  value: string;
  tone?: "success" | "warning" | "destructive" | "muted";
}

const TONE_TEXT: Record<string, string> = {
  success: "text-success",
  warning: "text-warning",
  destructive: "text-destructive",
  muted: "text-foreground",
};

const TONE_GLOW: Record<string, string> = {
  success: "dark:neon-text-current dark:text-success",
  warning: "dark:neon-text-current dark:text-warning",
  destructive: "dark:neon-text-current dark:text-destructive",
  muted: "text-foreground",
};

export function PingCheckStatistics({
  summary,
  isLoading,
  hasData,
  isFa,
}: {
  summary: PingSummary | null;
  isLoading: boolean;
  hasData: boolean;
  isFa: boolean;
}) {
  const s = summary;
  const t = (en: string, fa: string) => (isFa ? fa : en);

  const checks: StatRow[] = [
    { label: t("Total", "مجموع"), value: String(s?.totalChecks ?? "—") },
    { label: t("Successful", "موفق"), value: String(s?.successChecks ?? "—"), tone: "success" },
    { label: t("Failed", "ناموفق"), value: String(s?.failedChecks ?? "—"), tone: s?.failedChecks ? "destructive" : "muted" },
    { label: t("Success rate", "نرخ موفقیت"), value: s?.availability == null ? "—" : `${Math.round(s.availability)}%` },
  ];

  const latency: StatRow[] = [
    { label: t("Average", "میانگین"), value: s?.latency == null ? "—" : `${Math.round(s.latency)} ms` },
    { label: t("Minimum", "حداقل"), value: s?.latencyMin == null ? "—" : `${Math.round(s.latencyMin)} ms` },
    { label: t("Maximum", "حداکثر"), value: s?.latencyMax == null ? "—" : `${Math.round(s.latencyMax)} ms` },
  ];

  const packets: StatRow[] = [
    { label: t("Sent", "ارسال‌شده"), value: String(s?.packetsSent ?? "—") },
    { label: t("Received", "دریافت‌شده"), value: String(s?.packetsReceived ?? "—") },
    { label: t("Lost", "از دست رفته"), value: String(s?.packetsLost ?? "—"), tone: s?.packetsLost ? "destructive" : "muted" },
    { label: t("Loss", "درصد اتلاف"), value: s?.packetLoss == null ? "—" : `${Math.round(s.packetLoss)}%`, tone: s?.packetLoss ? "warning" : "muted" },
  ];

  const jitter: StatRow[] = [
    { label: t("Average", "میانگین"), value: s?.jitter == null ? "—" : `${Math.round(s.jitter)} ms` },
    { label: t("Maximum", "حداکثر"), value: s?.jitterMax == null ? "—" : `${Math.round(s.jitterMax)} ms` },
  ];

  return (
    <Card
      variant="bordered"
      className="shadow-subtle transition-[border-color,box-shadow] duration-300 dark:hover:border-primary/40 dark:hover:shadow-glow"
    >
      <CardHeader className="px-5 pt-4">
        <CardTitle className="text-sm font-semibold text-foreground">
          {t("Check statistics", "آمار بررسی‌ها")}
        </CardTitle>
      </CardHeader>
      <CardContent className="px-5 pb-4">
        {isLoading ? (
          <Skeleton className="h-24 w-full rounded-lg" />
        ) : !hasData ? (
          <p className="py-4 text-sm text-muted-foreground">
            {t("No monitoring data yet", "هنوز داده‌ای برای مانیتورینگ ثبت نشده است")}
          </p>
        ) : (
          <div className="grid grid-cols-2 gap-x-6 gap-y-5 sm:grid-cols-4">
            <StatGroup title={t("Checks", "بررسی‌ها")} rows={checks} />
            <StatGroup title={t("Latency", "تاخیر")} rows={latency} />
            <StatGroup title={t("Packets", "بسته‌ها")} rows={packets} />
            <StatGroup title={t("Jitter", "نوسان")} rows={jitter} />
          </div>
        )}
      </CardContent>
    </Card>
  );
}

function StatGroup({ title, rows }: { title: string; rows: StatRow[] }) {
  return (
    <div>
      <p className="mb-2.5 text-[11px] font-semibold uppercase tracking-[0.06em] text-muted-foreground">
        {title}
      </p>
      <dl className="space-y-1.5">
        {rows.map((row) => (
          <div key={row.label} className="flex items-center justify-between gap-2 text-[13px]">
            <dt className="truncate text-muted-foreground">{row.label}</dt>
            <dd
              dir="ltr"
              className={cn(
                "shrink-0 font-medium tabular-nums",
                TONE_TEXT[row.tone ?? "muted"],
                row.tone && row.tone !== "muted" && TONE_GLOW[row.tone],
              )}
            >
              {row.value}
            </dd>
          </div>
        ))}
      </dl>
    </div>
  );
}
