"use client";

import { useLocale, useTranslations } from "next-intl";

import { RelativeTime } from "@/components/common/relative-time";
import { MonitorActions } from "@/components/monitors/monitor-actions";
import { MonitorTypeLabel } from "@/components/monitors/monitor-type-label";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Link } from "@/i18n/navigation";
import { formatDuration, formatInterval } from "@/lib/formatters";
import { STATUS_STYLES } from "@/lib/monitor-meta";
import type { Monitor } from "@/types/monitor";

export function MonitorTableSkeleton() {
  return (
    <div className="flex flex-col gap-2 rounded-xl border border-border glass p-4">
      {Array.from({ length: 6 }).map((_, index) => (
        <Skeleton key={index} className="h-12 w-full" />
      ))}
    </div>
  );
}

export function MonitorTable({ monitors }: { monitors: Monitor[] }) {
  const t = useTranslations("monitors");
  const tCommon = useTranslations("common");
  const locale = useLocale();

  return (
    <div className="overflow-hidden rounded-xl border border-border/60 glass dark:border-primary/5">
      <div className="overflow-x-auto">
        <Table>
          <TableHeader>
            <TableRow className="border-border/60 hover:bg-transparent">
              <TableHead className="max-w-48">{t("name")}</TableHead>
              <TableHead className="hidden max-w-56 md:table-cell">{t("target")}</TableHead>
              <TableHead>{t("type")}</TableHead>
              <TableHead>{t("status")}</TableHead>
              <TableHead className="hidden sm:table-cell">{t("lastCheck")}</TableHead>
              <TableHead className="hidden lg:table-cell">{t("duration")}</TableHead>
              <TableHead className="hidden lg:table-cell">{t("interval")}</TableHead>
              <TableHead>
                <span className="sr-only">{tCommon("actions")}</span>
              </TableHead>
            </TableRow>
          </TableHeader>

          <TableBody>
            {monitors.map((monitor) => (
              <TableRow key={monitor.id} className="border-border/50 transition-colors hover:bg-muted/30">
                <TableCell className="max-w-48 font-medium">
                  <Link
                    href={`/app/monitors/${monitor.id}`}
                    className="block truncate hover:text-primary hover:underline"
                  >
                    {monitor.name}
                  </Link>
                </TableCell>
                <TableCell
                  dir="ltr"
                  className="hidden max-w-56 truncate font-mono text-xs text-muted-foreground md:table-cell"
                >
                  {monitor.target}
                </TableCell>
                <TableCell>
                  <MonitorTypeLabel type={monitor.type} />
                </TableCell>
                <TableCell>
                  {monitor.last_result?.error_code && monitor.last_status === "down" ? (
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <span tabIndex={0}>
                          <span
                            className={`inline-block size-2.5 rounded-full ${STATUS_STYLES[monitor.last_status].dot}`}
                            aria-hidden
                          />
                        </span>
                      </TooltipTrigger>
                      <TooltipContent dir="ltr" className="font-mono text-xs">
                        {monitor.last_result.error_code}
                      </TooltipContent>
                    </Tooltip>
                  ) : (
                    <span
                      className={`inline-block size-2.5 rounded-full ${STATUS_STYLES[monitor.last_status].dot}`}
                      aria-hidden
                    />
                  )}
                </TableCell>
                <TableCell className="hidden text-sm text-muted-foreground sm:table-cell">
                  {monitor.last_checked_at ? (
                    <RelativeTime value={monitor.last_checked_at} />
                  ) : (
                    tCommon("never")
                  )}
                </TableCell>
                <TableCell className="hidden text-sm text-muted-foreground lg:table-cell tabular-nums" dir="ltr">
                  {monitor.last_result
                    ? formatDuration(monitor.last_result.duration_millis, locale)
                    : "—"}
                </TableCell>
                <TableCell
                  dir="ltr"
                  className="hidden text-sm text-muted-foreground lg:table-cell"
                >
                  {formatInterval(monitor.interval_seconds, locale)}
                </TableCell>
                <TableCell className="text-end">
                  <MonitorActions monitor={monitor} />
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
    </div>
  );
}
