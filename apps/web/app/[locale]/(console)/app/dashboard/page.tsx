"use client";

import { useTranslations } from "next-intl";
import { Activity, Radio, Zap, Gauge as GaugeIcon } from "lucide-react";
import { WorldMonitoringMap } from "@/components/monitoring/world-monitoring-map";
import { useMonitoring } from "@/hooks/use-monitoring";
import { Badge } from "@/components/ui/badge";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";
import type { Probe } from "@/types/monitoring";

const STATUS_DOT: Record<string, string> = {
  online: "bg-success shadow-[0_0_6px_var(--success)]",
  warning: "bg-warning shadow-[0_0_6px_var(--warning)]",
  offline: "bg-destructive shadow-[0_0_6px_var(--destructive)]",
};

const STATUS_BADGE: Record<string, string> = {
  online: "bg-success/15 text-success border-success/30",
  warning: "bg-warning/15 text-warning border-warning/30",
  offline: "bg-destructive/15 text-destructive border-destructive/30",
};

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

function ProbeListItem({ probe }: { probe: Probe }) {
  return (
    <div className="flex items-center gap-3 rounded-lg border border-border/50 bg-card/40 px-3 py-2.5 transition-colors hover:bg-accent/30">
      <span className={cn("size-2 shrink-0 rounded-full", STATUS_DOT[probe.status] ?? STATUS_DOT.offline)} />
      <div className="min-w-0 flex-1">
        <p className="truncate text-sm font-medium">{probe.name}</p>
        <p className="text-xs text-muted-foreground">
          {probe.city}, {probe.country}
        </p>
      </div>
      <Badge variant="outline" className={cn("pointer-events-none", STATUS_BADGE[probe.status] ?? STATUS_BADGE.offline)}>
        {probe.status}
      </Badge>
    </div>
  );
}

function DashboardSkeleton() {
  return (
    <div className="-mx-1">
      <div className="mb-4 grid grid-cols-2 gap-3 lg:grid-cols-4">
        {Array.from({ length: 4 }).map((_, i) => (
          <Skeleton key={i} className="h-[72px] rounded-xl" />
        ))}
      </div>
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-[1fr_280px]">
        <Skeleton className="aspect-[2/1] w-full rounded-xl" />
        <Skeleton className="h-[400px] rounded-xl" />
      </div>
    </div>
  );
}

export default function DashboardPage() {
  const t = useTranslations("monitoring");
  const { probes, stats, loading } = useMonitoring();

  if (loading) return <DashboardSkeleton />;

  return (
    <div className="-mx-1">
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

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-[1fr_280px]">
        <WorldMonitoringMap />

        <div className="rounded-xl border border-border bg-card">
          <div className="border-b border-border/70 px-4 py-3">
            <h3 className="text-sm font-semibold">{t("probes")}</h3>
          </div>
          <ScrollArea className="h-[400px]">
            {probes.length === 0 ? (
              <div className="flex items-center justify-center py-12">
                <p className="text-sm text-muted-foreground">{t("noProbes")}</p>
              </div>
            ) : (
              <div className="flex flex-col gap-1.5 p-2">
                {probes.map((probe) => (
                  <ProbeListItem key={probe.id} probe={probe} />
                ))}
              </div>
            )}
          </ScrollArea>
        </div>
      </div>
    </div>
  );
}
