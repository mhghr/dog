"use client";

import { AlertTriangle, Info, XCircle } from "lucide-react";

import { Card, CardContent, CardHeader, CardTitle } from "@/shared/ui/card";
import { Skeleton } from "@/shared/ui/skeleton";
import { cn } from "@/shared/utils/cn";
import { formatRelativeTime } from "@/shared/utils/formatters";
import type { SnmpEvent } from "@/entities/resource/api/resource.api";

function severityIcon(severity: string) {
  switch (severity) {
    case "critical":
      return <XCircle className="size-4 text-destructive" />;
    case "warning":
      return <AlertTriangle className="size-4 text-warning" />;
    default:
      return <Info className="size-4 text-sky-500" />;
  }
}

export function SnmpEventsCard({
  events,
  isFa,
  isLoading,
}: {
  events: SnmpEvent[];
  isFa: boolean;
  isLoading: boolean;
}) {
  const t = (en: string, fa: string) => (isFa ? fa : en);

  return (
    <Card variant="bordered" className="h-full shadow-subtle">
      <CardHeader className="px-5 pt-4">
        <CardTitle className="text-sm font-semibold text-foreground">
          {t("SNMP Events", "رویدادهای SNMP")}
        </CardTitle>
      </CardHeader>
      <CardContent className="px-4 pb-4 pt-1">
        {isLoading ? (
          <Skeleton className="h-40 w-full rounded-lg" />
        ) : events.length === 0 ? (
          <p className="py-8 text-center text-sm text-muted-foreground">
            {t("No SNMP events recorded", "رویداد SNMP ثبت نشده است")}
          </p>
        ) : (
          <ul className="flex max-h-80 flex-col gap-1.5 overflow-y-auto pe-1">
            {events.map((event) => (
              <li
                key={event.id}
                className="flex items-start gap-2.5 rounded-lg border border-border/40 px-3 py-2"
              >
                <span className="mt-0.5 shrink-0">{severityIcon(event.severity)}</span>
                <div className="min-w-0 flex-1">
                  <div className="flex items-center justify-between gap-2">
                    <span
                      className="truncate text-xs font-semibold text-foreground"
                      dir="ltr"
                    >
                      {event.event_type}
                    </span>
                    <span className="shrink-0 text-[10px] text-muted-foreground">
                      {formatRelativeTime(event.created_at, isFa ? "fa" : "en")}
                    </span>
                  </div>
                  <p className="mt-0.5 truncate text-xs text-muted-foreground" dir="auto">
                    {event.summary}
                  </p>
                  {event.if_index != null && event.if_index > 0 && (
                    <p className="mt-0.5 text-[10px] tabular-nums text-muted-foreground/70" dir="ltr">
                      ifIndex {event.if_index}
                      {event.if_name ? ` · ${event.if_name}` : ""}
                    </p>
                  )}
                </div>
              </li>
            ))}
          </ul>
        )}
      </CardContent>
    </Card>
  );
}
