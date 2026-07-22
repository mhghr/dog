"use client";

import { useMemo } from "react";
import { useTranslations } from "next-intl";

import { Skeleton } from "@/components/ui/skeleton";
import { getMonitorDefinition } from "@/features/monitors/core/registry";
import { monitorHost } from "@/features/monitors/core/target";
import { Link } from "@/i18n/navigation";
import { cn } from "@/lib/utils";
import type { Monitor } from "@/types/monitor";

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
  const duration = monitor.type === "ping"
    ? monitor.last_result?.metrics?.avg_rtt_ms
    : monitor.last_result?.duration_millis;
  const warning = monitor.config.warning_latency_millis;
  const critical = monitor.config.critical_latency_millis;
  if (typeof duration === "number" && typeof critical === "number" && duration >= critical) {
    return "border-destructive/25 bg-destructive/10 text-destructive";
  }
  if (typeof duration === "number" && typeof warning === "number" && duration >= warning) {
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

  const nodes = useMemo(() => groupMonitorsByNode(monitors), [monitors]);

  return (
    <div className="grid min-w-0 gap-3 sm:grid-cols-2 xl:grid-cols-3">
      {nodes.map((node) => (
        <article
          key={node.key}
          className="group min-w-0 overflow-hidden rounded-xl border border-border/70 bg-card/70 shadow-sm shadow-foreground/[0.02] transition-[border-color,background-color,transform] duration-150 hover:-translate-y-0.5 hover:border-primary/30 hover:bg-card"
        >
          <Link
            href={`/app/monitors/${node.monitors[0].id}`}
            className="block min-w-0 px-4 py-3.5 outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring"
          >
            <h2 className="truncate text-[15px] font-semibold tracking-tight transition-colors group-hover:text-primary">
              {node.name}
            </h2>
            <p dir="ltr" className="mt-1 truncate text-start font-mono text-xs text-muted-foreground">
              {node.target}
            </p>
          </Link>

          <div className="border-t border-border/60 px-4 py-3">
            <div className="flex min-w-0 flex-wrap gap-2">
              {node.monitors.map((monitor) => {
                const definition = getMonitorDefinition(monitor.type);
                const Icon = definition.icon;

                return (
                  <Link
                    key={monitor.id}
                    href={`/app/monitors/${monitor.id}`}
                    title={monitor.name}
                    className={cn(
                      "inline-flex h-7 items-center gap-1.5 whitespace-nowrap rounded-md border px-2 text-xs font-medium outline-none transition-[filter,transform] hover:brightness-110 active:translate-y-px focus-visible:ring-2 focus-visible:ring-ring disabled:pointer-events-none disabled:opacity-50",
                      monitorBadgeTone(monitor),
                    )}
                  >
                    <Icon className="size-3.5" aria-hidden />
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
