"use client";

import { useTranslations } from "next-intl";
import { Activity, Radio, Zap, Gauge as GaugeIcon } from "lucide-react";
import dynamic from "next/dynamic";

import { useMonitoring } from "@/entities/probe/hooks/use-monitoring";

const WorldMonitoringMap = dynamic(() => import("@/widgets/monitoring-map/world-monitoring-map").then((m) => m.WorldMonitoringMap), { ssr: false });
import { Skeleton } from "@/shared/ui/skeleton";
import { cn } from "@/shared/utils/cn";

function StatCard({
  icon: Icon,
  label,
  value,
  color,
}: {
  icon: React.ElementType;
  label: string;
  value: string | number;
  color: string;
}) {
  return (
    <div className="flex items-center gap-3 rounded-xl border border-border/70 bg-card/60 px-4 py-3">
      <span className={cn("grid size-9 place-items-center rounded-lg", color)}>
        <Icon className="size-4" aria-hidden />
      </span>
      <div>
        <p className="text-xs text-muted-foreground">{label}</p>
        <p className="text-lg font-semibold tabular-nums">{value}</p>
      </div>
    </div>
  );
}

function StatsSkeleton() {
  return (
    <div className="mb-4 grid grid-cols-2 gap-3 lg:grid-cols-4">
      {Array.from({ length: 4 }).map((_, i) => (
        <Skeleton key={i} className="h-[72px] rounded-xl" />
      ))}
    </div>
  );
}

export default function DashboardPage() {
  const t = useTranslations("monitoring");
  const { stats, loading } = useMonitoring();

  return (
    <div className="-mx-1">
      {loading ? <StatsSkeleton /> : (
        <div className="mb-4 grid grid-cols-2 gap-3 lg:grid-cols-4">
          <StatCard
            icon={Radio}
            label={t("totalProbes")}
            value={stats.totalProbes}
            color="bg-primary/10 text-primary"
          />
          <StatCard
            icon={Activity}
            label={t("onlineProbes")}
            value={stats.onlineProbes}
            color="bg-success/10 text-success"
          />
          <StatCard
            icon={GaugeIcon}
            label={t("avgLatency")}
            value={`${stats.avgLatency}ms`}
            color="bg-info/10 text-info"
          />
          <StatCard
            icon={Zap}
            label={t("activeConnections")}
            value={stats.activeConnections}
            color="bg-warning/10 text-warning"
          />
        </div>
      )}

      <div className="flex justify-center">
        <WorldMonitoringMap className="aspect-[2/1] w-full max-w-[calc(200dvh_-_26rem)]" />
      </div>
    </div>
  );
}
