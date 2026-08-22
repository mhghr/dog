"use client";

import { useMemo, useState } from "react";
import { useLocale } from "next-intl";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { Skeleton } from "@/shared/ui/skeleton";
import { resourcesApi, type SnmpInterfaceRow } from "@/entities/resource/api/resource.api";
import {
  useResourceMonitorMetrics,
  type MetricsRange,
} from "@/entities/resource/hooks/use-resource";
import type { Monitor } from "@/entities/resource/hooks/types";
import type { ProbeSeries } from "@/entities/resource/api/resource.api";
import { readSnmpConfig } from "./snmp-config";
import {
  summarizeSnmp,
  toSnmpInterfaces,
  toSnmpSensors,
  toSnmpDevice,
  sparkOf,
} from "./snmp-metrics";
import { SnmpKpiGrid } from "./SnmpKpiGrid";
import { SnmpInterfacesCard, type SnmpInterfaceTableRow } from "./SnmpInterfacesCard";
import { SnmpHardwareCard } from "./SnmpHardwareCard";
import { SnmpEventsCard } from "./SnmpEventsCard";
import { SnmpDiagnosticsCard } from "./SnmpDiagnosticsCard";
import { PingTimeRangeSelector } from "../ping/PingTimeRangeSelector";

function pooledAvg(series: ProbeSeries[] | undefined): number | null {
  const values = (series ?? []).flatMap((s) => s.points.map((p) => p.value)).filter((v) => v != null);
  if (values.length === 0) return null;
  return values.reduce((a, b) => a + b, 0) / values.length;
}

export function SnmpMonitoringView({
  resourceId,
  monitor,
}: {
  resourceId: string;
  monitor: Monitor;
}) {
  const locale = useLocale();
  const isFa = locale === "fa";
  const t = (en: string, fa: string) => (isFa ? fa : en);
  const queryClient = useQueryClient();
  const [range, setRange] = useState<MetricsRange>("1h");

  const config = useMemo(() => readSnmpConfig(monitor.configuration), [monitor.configuration]);

  const metricsQuery = useResourceMonitorMetrics(resourceId, monitor.id, range);
  const statusQuery = useResourceMonitorMetrics(resourceId, monitor.id, range, "status");
  const cpuQuery = useResourceMonitorMetrics(resourceId, monitor.id, range, "device.cpu_percent");
  const memQuery = useResourceMonitorMetrics(resourceId, monitor.id, range, "device.memory_percent");

  const latest = metricsQuery.data?.latest ?? [];
  const summary = useMemo(() => summarizeSnmp(latest), [latest]);
  const interfaces = useMemo(() => toSnmpInterfaces(latest), [latest]);
  const sensors = useMemo(() => toSnmpSensors(latest), [latest]);
  const device = useMemo(() => toSnmpDevice(latest), [latest]);

  const cpuSpark = useMemo(() => sparkOf(cpuQuery.data?.series), [cpuQuery.data?.series]);
  const memSpark = useMemo(() => sparkOf(memQuery.data?.series), [memQuery.data?.series]);
  const statusSpark = useMemo(() => sparkOf(statusQuery.data?.series, (v) => v * 100), [statusQuery.data?.series]);
  const cpuAvg = useMemo(() => pooledAvg(cpuQuery.data?.series), [cpuQuery.data?.series]);
  const memAvg = useMemo(() => pooledAvg(memQuery.data?.series), [memQuery.data?.series]);

  const interfacesQuery = useQuery({
    queryKey: ["resources", resourceId, "snmp", monitor.id, "interfaces"],
    queryFn: () => resourcesApi.snmpListInterfaces(resourceId, monitor.id),
    enabled: Boolean(resourceId && monitor.id),
    staleTime: 15_000,
    refetchInterval: 60_000,
  });

  const eventsQuery = useQuery({
    queryKey: ["resources", resourceId, "snmp", monitor.id, "events"],
    queryFn: () => resourcesApi.snmpListEvents(resourceId, monitor.id, 30),
    enabled: Boolean(resourceId && monitor.id),
    staleTime: 30_000,
    refetchInterval: 60_000,
  });

  const rows: SnmpInterfaceTableRow[] = useMemo(() => {
    const settings = interfacesQuery.data?.items ?? [];
    return interfaces.map((snapshot) => ({
      snapshot,
      setting: settings.find((s: SnmpInterfaceRow) => s.if_index === snapshot.if_index),
    }));
  }, [interfaces, interfacesQuery.data]);

  const toggleMonitor = async (row: SnmpInterfaceTableRow, next: boolean) => {
    try {
      await resourcesApi.snmpUpdateInterface(resourceId, monitor.id, row.snapshot.if_index, {
        monitor: next,
        ignore: !next,
      });
      await queryClient.invalidateQueries({ queryKey: ["resources", resourceId, "snmp", monitor.id, "interfaces"] });
      await queryClient.invalidateQueries({ queryKey: ["resources", resourceId, "monitors"] });
      toast.success(next ? t("Interface monitored", "اینترفیس مانیتور شد") : t("Interface ignored", "اینترفیس نادیده گرفته شد"));
    } catch {
      toast.error(t("Failed to update interface", "به‌روزرسانی اینترفیس ناموفق بود"));
    }
  };

  const hasData = latest.length > 0;
  const isLoading = metricsQuery.isPending || statusQuery.isPending;
  const isError = metricsQuery.isError || statusQuery.isError;

  return (
    <section className="flex flex-col gap-6">
      {/* Header row: time range on the left, monitor name + device on the right. */}
      <div className="flex flex-wrap items-center justify-between gap-3">
        <PingTimeRangeSelector range={range} onChange={setRange} />
        <div className="flex min-w-0 items-center gap-2">
          <h2 className="truncate text-base font-semibold tracking-tight text-foreground">
            {monitor.name}
          </h2>
          {device.sys_name && (
            <span className="truncate rounded-md bg-muted/50 px-2 py-1 font-mono text-xs text-muted-foreground" dir="ltr">
              {device.sys_name}
            </span>
          )}
        </div>
      </div>

      {isError && !isLoading ? (
        <div className="flex flex-col items-center justify-center gap-2 rounded-xl border border-border/60 bg-card px-6 py-16 text-sm text-muted-foreground shadow-subtle">
          <span>{t("Unable to load data", "خطا در دریافت داده")}</span>
        </div>
      ) : isLoading ? (
        <div className="space-y-4">
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-6">
            {Array.from({ length: 6 }).map((_, i) => (
              <Skeleton key={i} className="h-36 rounded-xl" />
            ))}
          </div>
          <Skeleton className="h-72 rounded-xl" />
          <Skeleton className="h-64 rounded-xl" />
        </div>
      ) : !hasData ? (
        <div className="flex flex-col items-center justify-center gap-2 rounded-xl border border-border/60 bg-card px-6 py-16 shadow-subtle">
          <p className="text-sm font-medium text-foreground/80">
            {t("No monitoring data yet", "هنوز داده مانیتورینگ وجود ندارد")}
          </p>
          <p className="text-xs text-muted-foreground">
            {t(
              "The collector is active but has not produced results. Run discovery and wait for the first poll.",
              "کلکتور فعال است اما هنوز نتیجه‌ای تولید نکرده. Discovery را اجرا و منتظر اولین Poll بمانید.",
            )}
          </p>
        </div>
      ) : (
        <>
          <SnmpKpiGrid
            summary={summary}
            thresholds={config.thresholds}
            rangeLabel={range}
            cpuSpark={cpuSpark}
            memSpark={memSpark}
            statusSpark={statusSpark}
            cpuAvg={cpuAvg}
            memAvg={memAvg}
            isFa={isFa}
          />

          {/* Interfaces table full-width. */}
          <SnmpInterfacesCard
            resourceId={resourceId}
            monitorId={monitor.id}
            range={range}
            rows={rows}
            isFa={isFa}
            isLoading={interfacesQuery.isPending}
            onToggleMonitor={toggleMonitor}
          />

          {/* Hardware health + SNMP events side by side. */}
          <div className="grid grid-cols-1 gap-3 lg:grid-cols-2">
            <SnmpHardwareCard sensors={sensors} isFa={isFa} />
            <SnmpEventsCard events={eventsQuery.data?.items ?? []} isFa={isFa} isLoading={eventsQuery.isPending} />
          </div>

          {/* Advanced diagnostics — hidden in the main flow. */}
          <SnmpDiagnosticsCard resourceId={resourceId} monitorId={monitor.id} isFa={isFa} />
        </>
      )}
    </section>
  );
}
