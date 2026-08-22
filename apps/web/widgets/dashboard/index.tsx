"use client";

import { useTranslations } from "next-intl";
import { useLocale } from "next-intl";

import { useDashboardSummary } from "@/entities/dashboard/hooks/use-dashboard";
import { DashboardKpis } from "./dashboard-kpi";
import { DashboardCharts } from "./dashboard-charts";
import { DashboardFeeds } from "./dashboard-feeds";

const GRAIN =
  "url(\"data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='160' height='160'%3E%3Cfilter id='n'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='0.9' numOctaves='2' stitchTiles='stitch'/%3E%3C/filter%3E%3Crect width='100%25' height='100%25' filter='url(%23n)'/%3E%3C/svg%3E\")";

export function DashboardOverview() {
  const t = useTranslations("dashboard");
  const locale = useLocale();
  const query = useDashboardSummary();
  const summary = query.data;
  const isError = query.isError;

  const lastUpdated =
    query.dataUpdatedAt > 0
      ? new Date(query.dataUpdatedAt).toLocaleTimeString(locale, {
          hour: "2-digit",
          minute: "2-digit",
          second: "2-digit",
        })
      : null;

  return (
    <div className="relative">
      <div
        aria-hidden
        className="pointer-events-none fixed inset-0 z-0 opacity-[0.03] mix-blend-overlay"
        style={{ backgroundImage: GRAIN }}
      />
      <div className="relative z-[1] space-y-3">
        <div className="flex flex-wrap items-end justify-between gap-3">
          <div>
            <h1 className="text-xl font-semibold tracking-tight">{t("title")}</h1>
            <p className="mt-0.5 text-sm text-muted-foreground">{t("subtitle")}</p>
          </div>

          <div className="flex items-center gap-2">
            {isError ? (
              <span className="rounded-full border border-destructive/30 bg-destructive/10 px-3 py-1.5 text-xs font-medium text-destructive">
                {t("attentionTitle")}
              </span>
            ) : (
              <div className="flex items-center gap-2 rounded-full border border-border/40 bg-card px-3 py-1.5 text-xs text-muted-foreground shadow-subtle">
                <span className="relative flex size-2">
                  <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-success opacity-60" />
                  <span className="relative inline-flex size-2 rounded-full bg-success" />
                </span>
                <span className="font-medium text-foreground">{t("live")}</span>
                {lastUpdated ? (
                  <>
                    <span aria-hidden className="text-muted-foreground/50">
                      •
                    </span>
                    <span className="tabular-nums" dir="ltr">
                      {lastUpdated}
                    </span>
                  </>
                ) : null}
              </div>
            )}
          </div>
        </div>

        <DashboardKpis summary={summary} />
        <DashboardCharts summary={summary} />
        <DashboardFeeds summary={summary} />
      </div>
    </div>
  );
}
