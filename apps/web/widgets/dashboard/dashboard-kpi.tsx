"use client";

import type { ComponentType, ReactNode } from "react";
import {
  Activity,
  AlertTriangle,
  ArrowDownCircle,
  ArrowUpCircle,
  LayoutDashboard,
} from "lucide-react";
import { useTranslations } from "next-intl";

import { Card, CardContent } from "@/shared/ui/card";
import { cn } from "@/shared/utils/cn";
import type { DashboardSummary } from "@/entities/dashboard/model/types";

function RadialGauge({
  value,
  size = 64,
  strokeWidth = 6,
}: {
  value: number;
  size?: number;
  strokeWidth?: number;
}) {
  const r = (size - strokeWidth) / 2;
  const c = 2 * Math.PI * r;
  const pct = Math.max(0, Math.min(100, value));
  const color =
    pct >= 99.5 ? "var(--success)" : pct >= 97 ? "var(--warning)" : "var(--destructive)";

  return (
    <div className="relative shrink-0" style={{ width: size, height: size }}>
      <svg width={size} height={size} className="-rotate-90" role="img" aria-label={`${pct.toFixed(1)}% availability`}>
        <circle
          cx={size / 2}
          cy={size / 2}
          r={r}
          fill="none"
          stroke="var(--muted)"
          strokeWidth={strokeWidth}
        />
        <circle
          cx={size / 2}
          cy={size / 2}
          r={r}
          fill="none"
          stroke={color}
          strokeWidth={strokeWidth}
          strokeLinecap="round"
          strokeDasharray={c}
          strokeDashoffset={c * (1 - pct / 100)}
          className="transition-[stroke-dashoffset] duration-700 ease-[cubic-bezier(0.23,1,0.32,1)]"
        />
      </svg>
      <span
        className="absolute inset-0 grid place-items-center text-sm font-semibold tabular-nums"
        dir="ltr"
        style={{ color }}
      >
        {pct.toFixed(1)}%
      </span>
    </div>
  );
}

function ChecksBar({ successful, failed }: { successful: number; failed: number }) {
  const total = successful + failed;
  const pct = total > 0 ? (successful / total) * 100 : 0;

  return (
    <div className="flex min-w-0 flex-1 flex-col gap-2">
      <div className="flex h-2 w-full overflow-hidden rounded-full bg-muted">
        <div
          className="bg-success transition-[width] duration-700 ease-[cubic-bezier(0.23,1,0.32,1)]"
          style={{ width: `${pct}%` }}
        />
        <div
          className="bg-destructive transition-[width] duration-700 ease-[cubic-bezier(0.23,1,0.32,1)]"
          style={{ width: `${100 - pct}%` }}
        />
      </div>
      <div className="flex items-center justify-between text-xs tabular-nums">
        <span className="font-medium text-success" dir="ltr">
          {successful} ok
        </span>
        <span className="font-medium text-destructive" dir="ltr">
          {failed} fail
        </span>
      </div>
    </div>
  );
}

interface KpiCardProps {
  title: string;
  value?: string | number;
  icon: ComponentType<{ className?: string; "aria-hidden"?: boolean }>;
  accent: string;
  delay: number;
  children?: ReactNode;
}

function KpiCard({ title, value, icon: Icon, accent, delay, children }: KpiCardProps) {
  return (
    <Card
      variant="bordered"
      className={cn(
        "group/card animate-in fade-in slide-in-from-bottom-2 shadow-subtle transition-all duration-[250ms] ease-[cubic-bezier(0.23,1,0.32,1)] hover:-translate-y-px hover:shadow-md",
      )}
      style={{ animationDelay: `${delay}ms`, animationFillMode: "backwards" }}
    >
      <CardContent className="flex h-full flex-col gap-3 p-4">
        <div className="flex items-center justify-between gap-2">
          <p className="truncate text-[11px] font-semibold uppercase tracking-[0.07em] text-muted-foreground">
            {title}
          </p>
          <span
            className={cn(
              "grid size-8 shrink-0 place-items-center rounded-lg transition-transform duration-[250ms] ease-[cubic-bezier(0.23,1,0.32,1)] group-hover/card:scale-110",
              accent,
            )}
          >
            <Icon className="size-4" aria-hidden />
          </span>
        </div>
        {value !== undefined ? (
          <div className="flex flex-1 items-center justify-between gap-3">
            <span className="text-3xl font-semibold tracking-tight tabular-nums" dir="ltr">
              {value}
            </span>
            {children}
          </div>
        ) : (
          <div className="flex flex-1 items-center justify-between gap-3">{children}</div>
        )}
      </CardContent>
    </Card>
  );
}

function KpiSkeleton({ delay }: { delay: number }) {
  return (
    <Card
      variant="bordered"
      className="animate-pulse shadow-subtle"
      style={{ animationDelay: `${delay}ms` }}
    >
      <CardContent className="flex h-full flex-col gap-3 p-4">
        <div className="h-3 w-20 rounded-full bg-muted" />
        <div className="h-8 w-16 rounded-lg bg-muted/80" />
      </CardContent>
    </Card>
  );
}

export function DashboardKpis({ summary }: { summary?: DashboardSummary }) {
  const t = useTranslations("dashboard");
  const counts = summary?.status_counts ?? {};

  if (!summary) {
    return (
      <div className="grid grid-cols-2 gap-3 md:grid-cols-3 xl:grid-cols-5">
        {[0, 90, 180, 270, 360].map((d) => (
          <KpiSkeleton key={d} delay={d} />
        ))}
      </div>
    );
  }

  const checks = summary.checks_24h;
  const checkTotal = checks.successful + checks.failed;

  return (
    <div className="grid grid-cols-2 gap-3 md:grid-cols-3 xl:grid-cols-5">
      <KpiCard
        title={t("totalMonitors")}
        value={summary.total_monitors}
        icon={LayoutDashboard}
        accent="bg-primary/10 text-primary"
        delay={0}
      />
      <KpiCard
        title={t("statusUp")}
        value={counts.up ?? 0}
        icon={ArrowUpCircle}
        accent="bg-success/10 text-success"
        delay={90}
      />
      <KpiCard
        title={t("statusDown")}
        value={counts.down ?? 0}
        icon={ArrowDownCircle}
        accent="bg-destructive/10 text-destructive"
        delay={180}
      />
      <KpiCard
        title={t("availability24h")}
        icon={Activity}
        accent="bg-info/10 text-info"
        delay={270}
      >
        <RadialGauge value={summary.availability_24h ?? 0} />
      </KpiCard>
      <KpiCard
        title={t("checks24h")}
        value={checkTotal}
        icon={AlertTriangle}
        accent="bg-warning/10 text-warning"
        delay={360}
      >
        <ChecksBar successful={checks.successful} failed={checks.failed} />
      </KpiCard>
    </div>
  );
}
