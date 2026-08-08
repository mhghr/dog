"use client";

import { useMemo } from "react";
import { useTranslations } from "next-intl";

import { groupMonitorsByNode, monitorBadgeTone } from "@/features/monitor-management/ui/monitor-grid";
import { useConsoleBase } from "@/widgets/console-shell/use-console-base";
import { Skeleton } from "@/shared/ui/skeleton";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/shared/ui/table";
import { getMonitorDefinition } from "@/plugins/monitoring/core/registry";
import { Link } from "@/i18n/navigation";
import { cn } from "@/shared/utils/cn";
import type { Monitor } from "@/entities/monitor/model/types";

export function MonitorTableSkeleton() {
  return (
    <div className="flex flex-col gap-2 rounded-xl border border-border p-4">
      {Array.from({ length: 6 }).map((_, index) => <Skeleton key={index} className="h-12 w-full" />)}
    </div>
  );
}

export function MonitorTable({ monitors }: { monitors: Monitor[] }) {
  const t = useTranslations("monitors");
  const tTypes = useTranslations("types");
  const base = useConsoleBase();
  const nodes = useMemo(() => groupMonitorsByNode(monitors), [monitors]);

  return (
    <div className="overflow-hidden rounded-xl border border-border/60 bg-card/55">
      <div className="overflow-x-auto">
        <Table>
          <TableHeader>
            <TableRow className="border-border/60 hover:bg-transparent">
              <TableHead>{t("name")}</TableHead>
              <TableHead>{t("target")}</TableHead>
              <TableHead>{t("activeMonitors")}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {nodes.map((node) => (
              <TableRow key={node.key} className="border-border/50 hover:bg-muted/25">
                <TableCell className="max-w-56 font-medium">
                  <Link href={`${base}/nodes/${node.monitors[0].id}`} className="block truncate rounded-sm hover:text-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring">
                    {node.name}
                  </Link>
                </TableCell>
                <TableCell dir="ltr" className="max-w-64 truncate text-start font-mono text-xs text-muted-foreground">
                  {node.target}
                </TableCell>
                <TableCell>
                  <div className="flex min-w-max flex-wrap gap-1.5">
                    {node.monitors.map((monitor) => {
                      const definition = getMonitorDefinition(monitor.type);
                      const Icon = definition.icon;
                      return (
                        <Link
                          key={monitor.id}
                          href={`${base}/nodes/${monitor.id}`}
                          className={cn("inline-flex h-7 items-center gap-1.5 whitespace-nowrap rounded-md border px-2 text-xs font-medium outline-none hover:brightness-110 focus-visible:ring-2 focus-visible:ring-ring", monitorBadgeTone(monitor))}
                        >
                          <Icon className="size-3.5" aria-hidden />
                          {tTypes(monitor.type)}
                        </Link>
                      );
                    })}
                  </div>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
    </div>
  );
}
