"use client";

import { useState } from "react";
import { useLocale } from "next-intl";

import { CaretLeft, Monitor as MonitorIcon } from "@/shared/ui/icons";
import { Badge } from "@/shared/ui/badge";
import { Button } from "@/shared/ui/button";
import { Card, CardContent } from "@/shared/ui/card";
import { Skeleton } from "@/shared/ui/skeleton";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/shared/ui/tabs";
import {
  useDeleteResourceMonitor,
  useMonitorTypes,
  useResource,
  useResourceMonitorResults,
  useResourceMonitors,
  type Monitor,
} from "@/entities/resource/hooks/use-resource";
import { Link, usePathname } from "@/i18n/navigation";
import { cn } from "@/shared/utils/cn";
import type { MonitorTypeDef } from "@/entities/resource/model/types";

import { statusTone, MonitorTypeIcon } from "./components/monitor-type-icon";
import { MonitorConfig } from "./components/monitor-config";

export function ResourceDetailScreen({ resourceId }: { resourceId: string }) {
  const locale = useLocale();
  const pathname = usePathname();
  const wsMatch = pathname.match(/^\/console\/w\/([^/]+)/);
  const base = wsMatch ? `/console/w/${wsMatch[1]}` : "/app";
  const isFa = locale === "fa";
  const resourceQuery = useResource(resourceId);
  const monitorsQuery = useResourceMonitors(resourceId);

  const resource = resourceQuery.data;
  const monitors = monitorsQuery.data?.items ?? [];

  return (
    <div className="space-y-6">
      {resourceQuery.isPending ? (
        <Skeleton className="h-10 rounded-lg" />
      ) : resourceQuery.isError || !resource ? (
        <p className="text-sm text-destructive">
          {isFa ? "منبع یافت نشد" : "Resource not found"}
        </p>
      ) : (
        <div className="flex items-center gap-3 pb-4">
          <Link
            href={`${base}/resources`}
            aria-label={isFa ? "بازگشت به منابع" : "Back to resources"}
            className="inline-flex size-9 shrink-0 items-center justify-center rounded-lg border border-border/60 text-muted-foreground transition-colors hover:border-border hover:text-foreground"
          >
            <CaretLeft className="size-4" />
          </Link>
          <div className="min-w-0 flex-1">
            <h1 className="truncate text-xl font-semibold tracking-tight">
              {resource.name}
              <span className="mx-2 font-normal text-muted-foreground/60">·</span>
              <span className="text-base font-normal text-muted-foreground" dir="ltr">
                {resource.target || "—"}
              </span>
            </h1>
          </div>
          <Badge variant="outline" className={cn("shrink-0", statusTone(resource.status))}>
            {resource.status}
          </Badge>
        </div>
      )}

      <div className="mb-1 border-b border-border/60" />

      <Tabs defaultValue="monitoring" className="mt-1">
        <TabsList variant="line">
          <TabsTrigger value="monitoring">{isFa ? "مانیتورینگ" : "Monitoring"}</TabsTrigger>
          <TabsTrigger value="settings">{isFa ? "تنظیمات" : "Settings"}</TabsTrigger>
        </TabsList>

        <TabsContent value="monitoring" className="mt-4">
          {monitorsQuery.isPending ? (
            <div className="space-y-2">
              <Skeleton className="h-16 rounded-xl" />
              <Skeleton className="h-16 rounded-xl" />
            </div>
          ) : monitors.length === 0 ? (
            <Card className="border-border/70">
              <CardContent className="py-12 text-center">
                <div className="mx-auto mb-3 flex size-12 items-center justify-center rounded-full bg-muted/50">
                  <MonitorIcon className="size-6 text-muted-foreground" />
                </div>
                <p className="font-medium">{isFa ? "مانیتوری فعال نیست" : "No monitors active"}</p>
                <p className="mt-1 text-sm text-muted-foreground">
                  {isFa ? "از تب تنظیمات یک مانیتور فعال کنید" : "Enable a monitor from the Settings tab"}
                </p>
              </CardContent>
            </Card>
          ) : (
            <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
              {monitors.map((monitor) => (
                <MonitorRow key={monitor.id} resourceId={resourceId} monitor={monitor} />
              ))}
            </div>
          )}
        </TabsContent>

        <TabsContent value="settings" className="mt-4">
          <SettingsPanel resourceId={resourceId} monitors={monitors} />
        </TabsContent>
      </Tabs>
    </div>
  );
}

function MonitorRow({ resourceId, monitor }: { resourceId: string; monitor: Monitor }) {
  const locale = useLocale();
  const isFa = locale === "fa";
  const deleteMonitor = useDeleteResourceMonitor(resourceId);
  const { data: latestResult, isPending: resultPending } = useResourceMonitorResults(resourceId, monitor.id);

  const isPing = monitor.name.toLowerCase().includes("ping") ||
                 (latestResult?.metrics && "avg_rtt_ms" in latestResult.metrics);

  return (
    <Card className={cn("border-border/70", !monitor.enabled && "opacity-70")}>
      <CardContent className="flex items-start justify-between gap-3 p-4">
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            <p className="truncate text-sm font-semibold">{monitor.name}</p>
            <Badge variant="outline" className={statusTone(monitor.last_status)}>
              {monitor.last_status}
            </Badge>
            {!monitor.enabled && (
              <Badge variant="outline" className="border-muted bg-muted/40 text-muted-foreground">
                {isFa ? "متوقف" : "Paused"}
              </Badge>
            )}
          </div>
          <p className="mt-0.5 text-xs text-muted-foreground">
            {isFa
              ? `هر ${monitor.interval_seconds} ثانیه`
              : `Every ${monitor.interval_seconds} seconds`}
          </p>

          {isPing && latestResult && latestResult.metrics && (
            <div className="mt-2 grid grid-cols-3 gap-1.5">
              {latestResult.metrics.avg_rtt_ms != null && (
                <MetricPill
                  label={isFa ? "RTT" : "RTT"}
                  value={`${Math.round(Number(latestResult.metrics.avg_rtt_ms))}ms`}
                />
              )}
              {latestResult.metrics.packet_loss_percent != null && (
                <MetricPill
                  label={isFa ? "Packet Loss" : "Loss"}
                  value={`${Number(latestResult.metrics.packet_loss_percent).toFixed(1)}%`}
                  warn={Number(latestResult.metrics.packet_loss_percent) > 0}
                />
              )}
              {latestResult.metrics.min_rtt_ms != null && latestResult.metrics.max_rtt_ms != null && (
                <MetricPill
                  label={isFa ? "Min/Max" : "Min/Max"}
                  value={`${Math.round(Number(latestResult.metrics.min_rtt_ms))}/${Math.round(Number(latestResult.metrics.max_rtt_ms))}ms`}
                />
              )}
              {latestResult.metrics.stddev_rtt_ms != null && (
                <MetricPill
                  label={isFa ? "Jitter" : "Jitter"}
                  value={`${Number(latestResult.metrics.stddev_rtt_ms).toFixed(1)}ms`}
                />
              )}
              {latestResult.duration_millis != null && (
                <MetricPill
                  label={isFa ? "زمان" : "Duration"}
                  value={`${latestResult.duration_millis}ms`}
                />
              )}
            </div>
          )}
        </div>
        <Button
          variant="ghost"
          size="icon"
          onClick={() => deleteMonitor.mutate(monitor.id)}
          aria-label={isFa ? "حذف" : "Delete"}
          className="text-muted-foreground hover:text-destructive shrink-0"
        >
          <svg viewBox="0 0 256 256" fill="currentColor" className="size-4" aria-hidden><path d="M216 48h-40v-8a24 24 0 0 0-24-24h-48a24 24 0 0 0-24 24v8H40a8 8 0 0 0 0 16h8v144a16 16 0 0 0 16 16h128a16 16 0 0 0 16-16V64h8a8 8 0 0 0 0-16ZM96 40a8 8 0 0 1 8-8h48a8 8 0 0 1 8 8v8H96Zm96 168H64V64h128Zm-80-104v64a8 8 0 0 1-16 0v-64a8 8 0 0 1 16 0Zm48 0v64a8 8 0 0 1-16 0v-64a8 8 0 0 1 16 0Z"/></svg>
        </Button>
      </CardContent>
    </Card>
  );
}

function MetricPill({ label, value, warn }: { label: string; value: string; warn?: boolean }) {
  return (
    <div className={cn(
      "rounded-md px-2 py-1 text-xs",
      warn ? "bg-destructive/10 text-destructive" : "bg-muted/50 text-muted-foreground"
    )}>
      <span className="font-medium tabular-nums">{value}</span>
      <span className="ml-1 opacity-60">{label}</span>
    </div>
  );
}

function SettingsPanel({
  resourceId,
  monitors,
}: {
  resourceId: string;
  monitors: Monitor[];
}) {
  const locale = useLocale();
  const isFa = locale === "fa";
  const monitorTypesQuery = useMonitorTypes();
  const [selectedTypeId, setSelectedTypeId] = useState<string | null>(null);

  const types = monitorTypesQuery.data?.items ?? [];
  const selectedType = types.find((t) => t.id === selectedTypeId) ?? null;
  const selectedMonitor = selectedType
    ? monitors.find((m) => m.monitor_type_id === selectedType.id)
    : undefined;

  return (
    <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
      <div className="lg:col-span-2">
        {!selectedType ? (
          <Card className="flex h-full items-center border-dashed border-border/60">
            <CardContent className="flex w-full flex-col items-center justify-center py-16 text-center">
              <div className="mb-3 flex size-10 items-center justify-center rounded-full bg-muted/50">
                <MonitorIcon className="size-5 text-muted-foreground" />
              </div>
              <p className="font-medium text-sm">
                {isFa ? "یک نوع مانیتور را انتخاب کنید" : "Select a monitor type"}
              </p>
              <p className="mt-1 text-xs text-muted-foreground">
                {isFa ? "از کارت‌های سمت چپ یک گزینه را برای تنظیم انتخاب کنید" : "Choose a monitor type from the cards"}
              </p>
            </CardContent>
          </Card>
        ) : (
          <MonitorConfig
            key={selectedType.id}
            resourceId={resourceId}
            type={selectedType}
            monitor={selectedMonitor}
            isFa={isFa}
          />
        )}
      </div>

      <div>
        <TypeSelector types={types} monitors={monitors} selectedTypeId={selectedTypeId}
          onSelect={setSelectedTypeId} isPending={monitorTypesQuery.isPending} />
      </div>
    </div>
  );
}

function TypeSelector({
  types,
  monitors,
  selectedTypeId,
  onSelect,
  isPending,
}: {
  types: MonitorTypeDef[];
  monitors: Monitor[];
  selectedTypeId: string | null;
  onSelect: (id: string) => void;
  isPending: boolean;
}) {
  if (isPending) {
    return (
       <div className="overflow-hidden rounded-xl border border-border/60 bg-border/60">
        <div className="grid grid-cols-4 gap-px">
          {Array.from({ length: 8 }).map((_, i) => (
            <Skeleton key={i} className="aspect-square w-full rounded-xl" />
          ))}
        </div>
      </div>
    );
  }

  return (
    <div className="overflow-hidden rounded-xl border border-border/60 bg-border/60">
      <div className="grid grid-cols-4 gap-px">
        {types.map((type) => {
          const monitor = monitors.find((m) => m.monitor_type_id === type.id);
          const isActive = monitor?.enabled ?? false;
          const isSelected = selectedTypeId === type.id;

          return (
            <button
              key={type.id}
              type="button"
              onClick={() => onSelect(type.id)}
              aria-pressed={isSelected}
              className={cn(
                "flex aspect-square flex-col items-center justify-center gap-1.5 bg-card p-1 transition-all duration-150",
                isSelected
                  ? "bg-accent shadow-[inset_0_0_0_2px_var(--primary)_/_30%]"
                  : "hover:bg-muted/40",
                isActive && !isSelected && "bg-emerald-50/30 dark:bg-emerald-950/10",
              )}
            >
              <span
                className={cn(
                  "flex size-14 items-center justify-center rounded-full transition-colors",
                  isSelected
                    ? "bg-primary text-primary-foreground"
                    : isActive
                      ? "bg-emerald-100 text-emerald-600 dark:bg-emerald-900/40 dark:text-emerald-400"
                      : "bg-muted text-muted-foreground",
                )}
              >
                <MonitorTypeIcon type={type.name} className="size-7" />
              </span>
              <span
                className={cn(
                  "text-[9px] font-medium leading-none",
                  isSelected ? "text-foreground" : "text-muted-foreground",
                )}
              >
                {type.name}
              </span>
            </button>
          );
        })}
      </div>
    </div>
  );
}
