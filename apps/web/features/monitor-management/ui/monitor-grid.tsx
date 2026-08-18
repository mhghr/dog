"use client";

import { useMemo } from "react";
import { useTranslations } from "next-intl";

import { Skeleton } from "@/shared/ui/skeleton";
import { getMonitorDefinition } from "@/plugins/monitoring/core/registry";
import { monitorHost } from "@/plugins/monitoring/core/target";
import { Link } from "@/i18n/navigation";
import { useConsoleBase } from "@/widgets/console-shell/use-console-base";
import { cn } from "@/shared/utils/cn";
import type { Monitor } from "@/entities/monitor/model/types";

export interface NodeGroup {
  key: string;
  name: string;
  target: string;
  monitors: Monitor[];
}

export function monitorBadgeTone(monitor: Monitor) {
  if (monitor.last_status === "down") {
    return "border-destructive/25 bg-destructive/10 text-destructive";
  }
  const metrics = monitor.last_result?.metrics ?? {};
  const duration = monitor.type === "ping" ? metrics.rtt_ms : monitor.last_result?.duration_millis;
  const criticalChecks = [
    [duration, monitor.config.critical_duration_millis],
    [metrics.packet_loss_percent, monitor.config.critical_packet_loss_percent],
    [metrics.jitter_ms, monitor.config.critical_jitter_millis],
    [Math.abs(metrics.offset_ms ?? 0), monitor.config.max_offset_millis],
    [metrics.round_trip_ms, monitor.config.max_round_trip_millis],
  ];
  const warningChecks = [
    [duration, monitor.config.warning_duration_millis],
    [metrics.packet_loss_percent, monitor.config.warning_packet_loss_percent],
    [metrics.jitter_ms, monitor.config.warning_jitter_millis],
    [Math.abs(metrics.offset_ms ?? 0), monitor.config.warning_offset_millis],
    [metrics.round_trip_ms, monitor.config.warning_round_trip_millis],
  ];
  const daysRemaining = metrics.days_remaining;
  if (typeof daysRemaining === "number" && typeof monitor.config.critical_days === "number" && daysRemaining <= monitor.config.critical_days) return "border-destructive/25 bg-destructive/10 text-destructive";
  if (typeof daysRemaining === "number" && typeof monitor.config.warning_days === "number" && daysRemaining <= monitor.config.warning_days) return "border-warning/25 bg-warning/10 text-warning";
  if (criticalChecks.some(([value, threshold]) => typeof value === "number" && typeof threshold === "number" && value >= threshold)) {
    return "border-destructive/25 bg-destructive/10 text-destructive";
  }
  if (warningChecks.some(([value, threshold]) => typeof value === "number" && typeof threshold === "number" && value >= threshold)) {
    return "border-warning/25 bg-warning/10 text-warning";
  }
  if (monitor.last_status === "unknown" || monitor.last_status === "paused") {
    return "border-warning/25 bg-warning/10 text-warning";
  }
  return "border-success/25 bg-success/10 text-success";
}

export function groupMonitorsByNode(monitors: Monitor[]): NodeGroup[] {
  const groups = new Map<string, NodeGroup>();

  for (const monitor of monitors) {
    const host = monitorHost(monitor.target, monitor.type);
    const key = host.toLocaleLowerCase();
    const existing = groups.get(key);

    if (existing) existing.monitors.push(monitor);
    else groups.set(key, { key, name: monitor.name, target: host, monitors: [monitor] });
  }

  return [...groups.values()];
}

export function MonitorGridSkeleton() {
  return (
    <div className="grid min-w-0 gap-3 sm:grid-cols-2 xl:grid-cols-3">
      {Array.from({ length: 6 }).map((_, index) => (
        <Skeleton key={index} className="h-36 rounded-xl" />
      ))}
    </div>
  );
}

export function MonitorGrid({ monitors }: { monitors: Monitor[] }) {
  const tTypes = useTranslations("types");
  const base = useConsoleBase();

  const nodes = useMemo(() => groupMonitorsByNode(monitors), [monitors]);

  return (
    <div className="grid min-w-0 gap-3 sm:grid-cols-2 xl:grid-cols-3">
      {nodes.map((node) => (
        <article
          key={node.key}
          className="group relative min-w-0 overflow-hidden rounded-xl border border-border/70 bg-card transition-[border-color,box-shadow] duration-200 hover:border-border hover:shadow-card-hover"
        >
          <Link
            href={`${base}/monitors/${(node.monitors.find((m) => m.enabled) ?? node.monitors[0]).id}`}
            className="block min-w-0 px-4 py-3.5 outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring"
          >
            <div className="flex items-center gap-2">
              <span className="flex size-8 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                  <rect x="2" y="2" width="20" height="8" rx="2" ry="2" />
                  <rect x="2" y="14" width="20" height="8" rx="2" ry="2" />
                  <line x1="6" y1="6" x2="6.01" y2="6" />
                  <line x1="6" y1="18" x2="6.01" y2="18" />
                </svg>
              </span>
              <div className="min-w-0">
                <h2 className="truncate text-[15px] font-semibold tracking-tight text-foreground">
                  {node.name}
                </h2>
                <p dir="ltr" className="truncate text-start font-mono text-[11px] text-muted-foreground/70">
                  {node.target}
                </p>
              </div>
            </div>
          </Link>

          <div className="border-t border-border/50 px-4 py-2.5">
            <div className="flex min-w-0 flex-wrap gap-1.5">
              {node.monitors.map((monitor) => {
                const definition = getMonitorDefinition(monitor.type);
                const Icon = definition.icon;

                return (
                  <Link
                    key={monitor.id}
                    href={`${base}/monitors/${monitor.id}`}
                    title={monitor.name}
                    className={cn(
                      "inline-flex h-6 items-center gap-1.5 whitespace-nowrap rounded-md border px-2 text-[11px] font-medium outline-none transition-[filter] hover:brightness-110 active:translate-y-px focus-visible:ring-2 focus-visible:ring-ring",
                      monitorBadgeTone(monitor),
                    )}
                  >
                    <Icon className="size-3" aria-hidden />
                    <span>{tTypes(monitor.type)}</span>
                  </Link>
                );
              })}
            </div>
          </div>
        </article>
      ))}
    </div>
  );
}
