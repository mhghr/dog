"use client";

import { useTranslations } from "next-intl";

import { getMonitorDefinition } from "@/features/monitors/core/registry";
import { belongsToSameNode, monitorHost } from "@/features/monitors/core/target";
import { useMonitors } from "@/hooks/use-monitors";
import { Link } from "@/i18n/navigation";
import { cn } from "@/lib/utils";
import type { Monitor } from "@/types/monitor";

export function NodeMonitorTabs({ currentMonitor }: { currentMonitor: Monitor }) {
  const tTypes = useTranslations("types");
  const host = monitorHost(currentMonitor.target, currentMonitor.type);
  const monitorsQuery = useMonitors({ page: 1, pageSize: 100, search: host });

  const related = (monitorsQuery.data?.items ?? []).filter((monitor) =>
    belongsToSameNode(currentMonitor, monitor),
  );
  const monitors = related.some((monitor) => monitor.id === currentMonitor.id)
    ? related
    : [currentMonitor, ...related];

  return (
    <nav aria-label={tTypes(currentMonitor.type)} className="-mb-px overflow-x-auto border-b border-border/70">
      <div className="flex min-w-max items-end gap-1">
        {monitors.map((monitor) => {
          const definition = getMonitorDefinition(monitor.type);
          const Icon = definition.icon;
          const active = monitor.id === currentMonitor.id;

          return (
            <Link
              key={monitor.id}
              href={`/app/nodes/${monitor.id}`}
              aria-current={active ? "page" : undefined}
              title={monitor.name}
              className={cn(
                "relative flex h-11 items-center gap-2 whitespace-nowrap px-3 text-sm transition-colors",
                "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring",
                active ? "font-medium text-foreground" : "text-muted-foreground hover:text-foreground",
              )}
            >
              <Icon className={cn("size-4", active && "text-primary")} aria-hidden />
              <span>{tTypes(monitor.type)}</span>
              <span
                className={cn(
                  "absolute inset-x-2 bottom-0 h-0.5 bg-primary transition-opacity",
                  active ? "opacity-100" : "opacity-0",
                )}
                aria-hidden
              />
            </Link>
          );
        })}
      </div>
    </nav>
  );
}
