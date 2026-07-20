"use client";

import { useLocale, useTranslations } from "next-intl";

import { ErrorState } from "@/components/common/error-state";
import { RelativeTime } from "@/components/common/relative-time";
import { MonitorStatusBadge } from "@/components/monitors/monitor-status-badge";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { useDashboardSummary } from "@/hooks/use-dashboard-summary";
import { Link } from "@/i18n/navigation";
import { formatDuration, formatNumber, formatPercent } from "@/lib/formatters";
import { STATUS_STYLES } from "@/lib/monitor-meta";
import { cn } from "@/lib/utils";
import type { AppIcon } from "@/lib/icons";
import {
  CalendarCheck,
  EnvelopeSimple,
  Pulse,
  ShieldWarning,
  Timer,
  Warning,
} from "@/lib/icons";
import type { MonitorStatus } from "@/types/monitor";

function StatCard({
  label,
  value,
  status,
  size = "sm",
}: {
  label: string;
  value: string;
  status?: MonitorStatus;
  size?: "lg" | "sm";
}) {
  return (
    <div className="stat-card">
      <span className="flex items-center gap-2 text-sm text-muted-foreground">
        {status ? (
          <span
            className={cn("size-2 rounded-full", STATUS_STYLES[status].dot)}
            aria-hidden
          />
        ) : null}
        {label}
      </span>
      <span
        className={cn(
          "font-semibold tabular-nums tracking-tight",
          size === "lg" ? "text-3xl" : "text-xl",
        )}
        dir="ltr"
      >
        {value}
      </span>
    </div>
  );
}

function AttentionCard({
  icon: Icon,
  label,
  count,
}: {
  icon: AppIcon;
  label: string;
  count: number;
}) {
  const hasAttention = count > 0;
  return (
    <div
      className={cn(
        "flex items-center gap-3 rounded-lg border px-3 py-2.5 transition-colors",
        hasAttention
          ? "border-warning/30 bg-warning/5"
          : "border-border/60 bg-muted/30",
      )}
    >
      <Icon
        className={cn(
          "size-4 shrink-0",
          hasAttention ? "text-warning" : "text-muted-foreground/50",
        )}
        aria-hidden
      />
      <span className="flex-1 text-sm">{label}</span>
      <span
        className={cn(
          "text-sm font-semibold tabular-nums",
          hasAttention ? "text-warning" : "text-muted-foreground",
        )}
      >
        {count}
      </span>
    </div>
  );
}

export default function DashboardPage() {
  const t = useTranslations("dashboard");
  const tTypes = useTranslations("types");
  const tStatus = useTranslations("status");
  const locale = useLocale();

  const summaryQuery = useDashboardSummary();

  if (summaryQuery.isPending) {
    return (
      <div className="space-y-4">
        <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
          {Array.from({ length: 4 }).map((_, index) => (
            <Skeleton key={index} className="h-[104px] rounded-xl" />
          ))}
        </div>
        <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
          {Array.from({ length: 4 }).map((_, index) => (
            <Skeleton key={index} className="h-[88px] rounded-xl" />
          ))}
        </div>
      </div>
    );
  }

  if (summaryQuery.isError) {
    return <ErrorState onRetry={() => void summaryQuery.refetch()} />;
  }

  const summary = summaryQuery.data;

  return (
    <div className="space-y-4">
      {/* Primary KPI row */}
      <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
        <StatCard
          size="lg"
          label={t("totalMonitors")}
          value={formatNumber(summary.total_monitors, locale)}
        />
        <StatCard
          size="lg"
          label={t("availability24h")}
          value={formatPercent(summary.availability_24h, locale)}
        />
        <StatCard
          size="lg"
          label={tStatus("up")}
          status="up"
          value={formatNumber(summary.status_counts.up, locale)}
        />
        <StatCard
          size="lg"
          label={tStatus("down")}
          status="down"
          value={formatNumber(summary.status_counts.down, locale)}
        />
      </div>

      {/* Secondary KPI row */}
      <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
        <StatCard
          label={tStatus("paused")}
          status="paused"
          value={formatNumber(summary.status_counts.paused, locale)}
        />
        <StatCard
          label={tStatus("unknown")}
          status="unknown"
          value={formatNumber(summary.status_counts.unknown, locale)}
        />
        <StatCard
          label={t("successful")}
          value={formatNumber(summary.checks_24h.successful, locale)}
        />
        <StatCard
          label={t("failed")}
          value={formatNumber(summary.checks_24h.failed, locale)}
        />
      </div>

      {/* Detail panels */}
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        {/* Recent failures */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-base">
              <span className="grid size-7 place-items-center rounded-lg bg-destructive/10 text-destructive">
                <Warning className="size-3.5" aria-hidden />
              </span>
              {t("recentFailures")}
            </CardTitle>
          </CardHeader>
          <CardContent>
            {summary.recent_failures.length === 0 ? (
              <p className="py-8 text-center text-sm text-muted-foreground">
                {t("noFailures")}
              </p>
            ) : (
              <ul className="-mx-(--card-spacing) flex flex-col divide-y divide-border">
                {summary.recent_failures.map((failure, index) => (
                  <li
                    key={`${failure.monitor_id}-${index}`}
                    className="flex items-center gap-3 px-(--card-spacing) py-3 transition-colors hover:bg-muted/40"
                  >
                    <MonitorStatusBadge status="down" />
                    <div className="min-w-0 flex-1">
                      <Link
                        href={`/app/monitors/${failure.monitor_id}`}
                        className="block truncate text-sm font-medium hover:text-primary hover:underline"
                      >
                        {failure.monitor_name}
                      </Link>
                      <p
                        dir="ltr"
                        className="truncate text-start font-mono text-xs text-muted-foreground"
                      >
                        {failure.error_code ?? "—"}
                      </p>
                    </div>
                    <span className="shrink-0 text-xs text-muted-foreground">
                      <RelativeTime value={failure.started_at} />
                    </span>
                  </li>
                ))}
              </ul>
            )}
          </CardContent>
        </Card>

        <div className="flex flex-col gap-4">
          {/* Slowest monitors */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2 text-base">
                <span className="grid size-7 place-items-center rounded-lg bg-warning/10 text-warning">
                  <Timer className="size-3.5" aria-hidden />
                </span>
                {t("slowestMonitors")}
              </CardTitle>
            </CardHeader>
            <CardContent>
              {summary.slowest_monitors.length === 0 ? (
                <p className="py-8 text-center text-sm text-muted-foreground">
                  {t("noSlowMonitors")}
                </p>
              ) : (
                <ul className="-mx-(--card-spacing) flex flex-col divide-y divide-border">
                  {summary.slowest_monitors.map((slow) => (
                    <li key={slow.monitor_id} className="flex items-center gap-3 px-(--card-spacing) py-3 transition-colors hover:bg-muted/40">
                      <Pulse className="size-4 shrink-0 text-muted-foreground" aria-hidden />
                      <div className="min-w-0 flex-1">
                        <Link
                          href={`/app/monitors/${slow.monitor_id}`}
                          className="block truncate text-sm font-medium hover:text-primary hover:underline"
                        >
                          {slow.monitor_name}
                        </Link>
                        <p className="text-xs text-muted-foreground">
                          {tTypes(slow.monitor_type as Parameters<typeof tTypes>[0])}
                        </p>
                      </div>
                      <span className="shrink-0 text-sm font-medium tabular-nums" dir="ltr">
                        {formatDuration(slow.duration_millis, locale)}
                      </span>
                    </li>
                  ))}
                </ul>
              )}
            </CardContent>
          </Card>

          {/* Attention required */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2 text-base">
                <span className="grid size-7 place-items-center rounded-lg bg-info/10 text-info">
                  <ShieldWarning className="size-3.5" aria-hidden />
                </span>
                {t("attentionTitle")}
              </CardTitle>
            </CardHeader>
            <CardContent className="grid grid-cols-1 gap-2 sm:grid-cols-2">
              <AttentionCard
                icon={ShieldWarning}
                label={t("certificatesExpiring")}
                count={summary.attention_required.certificates_expiring_30d}
              />
              <AttentionCard
                icon={CalendarCheck}
                label={t("domainsExpiring")}
                count={summary.attention_required.domains_expiring_45d}
              />
              <AttentionCard
                icon={EnvelopeSimple}
                label={t("smtpFailures")}
                count={summary.attention_required.smtp_starttls_failures}
              />
              <AttentionCard
                icon={Timer}
                label={t("ntpHighOffset")}
                count={summary.attention_required.ntp_high_offset}
              />
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}
