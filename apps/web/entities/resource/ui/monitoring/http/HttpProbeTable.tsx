"use client";

import { useMemo } from "react";

import { Card, CardContent, CardHeader, CardTitle } from "@/shared/ui/card";
import { cn } from "@/shared/utils/cn";
import type { ProbeAggregateMetrics } from "@/entities/resource/api/resource.api";
import { statusLabelOf } from "./http-metrics";

interface HttpProbeTableProps {
  probes: ProbeAggregateMetrics[];
  selectedProbeId: string | null;
  isFa: boolean;
  onSelect: (probeId: string | null) => void;
}

function pct(v: number | null): string {
  return v == null ? "—" : `${Math.round(v)}%`;
}

function ms(v: number | null): string {
  return v == null ? "—" : `${Math.round(v)}`;
}

// Per-probe performance table over the selected range. Rows are selectable:
// picking a probe filters the timeline to its region and drives the detail
// cards below.
export function HttpProbeTable({
  probes,
  selectedProbeId,
  isFa,
  onSelect,
}: HttpProbeTableProps) {
  const t = (en: string, fa: string) => (isFa ? fa : en);

  const rows = useMemo(
    () =>
      [...probes].sort((a, b) => {
        const na = a.location || a.probe_name;
        const nb = b.location || b.probe_name;
        return na.localeCompare(nb);
      }),
    [probes],
  );

  return (
    <Card variant="bordered" className="shadow-subtle">
      <CardHeader className="px-5 pt-4">
        <CardTitle className="text-sm font-semibold text-foreground">
          {t("Probe Performance", "عملکرد پراب‌ها")}
        </CardTitle>
      </CardHeader>
      <CardContent className="px-3 pb-3">
        {rows.length === 0 ? (
          <p className="py-8 text-center text-sm text-muted-foreground">
            {t("No probes recorded in this range", "در این بازه پرابی ثبت نشده است")}
          </p>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full min-w-[560px] border-collapse text-xs">
              <thead>
                <tr className="text-start text-[11px] uppercase tracking-wide text-muted-foreground">
                  <Th className="text-start ps-2">{t("Location", "موقعیت")}</Th>
                  <Th className="text-center">{t("Status", "وضعیت")}</Th>
                  <Th className="text-center">{t("Availability", "دسترس‌پذیری")}</Th>
                  <Th className="text-center">{t("Avg", "میانگین")}</Th>
                  <Th className="text-center">{t("P95", "P95")}</Th>
                  <Th className="text-center">{t("Checks", "بررسی‌ها")}</Th>
                  <Th className="text-end pe-2">{t("Last Check", "آخرین بررسی")}</Th>
                </tr>
              </thead>
              <tbody>
                {rows.map((probe) => {
                  const name = probe.location || probe.probe_name || probe.probe_id;
                  const active = probe.probe_id === selectedProbeId;
                  const failed = !probe.last_success;
                  return (
                    <tr
                      key={probe.probe_id}
                      role="button"
                      tabIndex={0}
                      onClick={() => onSelect(active ? null : probe.probe_id)}
                      onKeyDown={(e) => {
                        if (e.key === "Enter" || e.key === " ") {
                          e.preventDefault();
                          onSelect(active ? null : probe.probe_id);
                        }
                      }}
                      className={cn(
                        "cursor-pointer border-t border-border/40 transition-colors hover:bg-accent/50 focus-visible:bg-accent/50 focus-visible:outline-none",
                        active && "bg-primary/[0.06]",
                      )}
                    >
                      <td className="px-2 py-2.5">
                        <span className="flex items-center gap-2">
                          <span
                            className={cn(
                              "size-1.5 shrink-0 rounded-full",
                              failed ? "bg-destructive shadow-[0_0_6px_1px_rgba(244,63,94,0.5)]" : "bg-success shadow-[0_0_6px_1px_rgba(16,185,129,0.5)]",
                            )}
                            aria-hidden
                          />
                          <span className="font-medium text-foreground">{name}</span>
                        </span>
                      </td>
                      <td className="px-2 text-center">
                        {probe.last_status_code != null ? (
                          <span
                            className={cn(
                              "inline-flex rounded-md px-1.5 py-0.5 font-mono tabular-nums",
                              failed ? "bg-destructive/12 text-destructive" : "bg-success/12 text-success",
                            )}
                            dir="ltr"
                          >
                            {statusLabelOf(probe.last_status_code, null)}
                          </span>
                        ) : (
                          <span className="text-muted-foreground">{t("—", "—")}</span>
                        )}
                      </td>
                      <td className={cn("px-2 text-center tabular-nums", probe.availability != null && probe.availability < 95 ? "text-warning" : "text-foreground")}>
                        {pct(probe.availability)}
                      </td>
                      <td className="px-2 text-center tabular-nums text-foreground" dir="ltr">{ms(probe.avg_response_time_ms)}</td>
                      <td className="px-2 text-center tabular-nums text-muted-foreground" dir="ltr">{ms(probe.p95_response_time_ms)}</td>
                      <td className="px-2 text-center tabular-nums text-muted-foreground">{probe.checks.total_requests}</td>
                      <td className="px-2 text-end tabular-nums text-muted-foreground" dir="ltr">
                        {probe.last_checked_at ? formatRelative(probe.last_checked_at, isFa) : "—"}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </CardContent>
    </Card>
  );
}

function Th({ children, className }: { children: React.ReactNode; className?: string }) {
  return <th className={cn("px-2 py-2 font-semibold", className)}>{children}</th>;
}

function formatRelative(iso: string, isFa: boolean): string {
  const then = new Date(iso).getTime();
  const diff = Math.max(0, Date.now() - then);
  const sec = Math.floor(diff / 1000);
  const min = Math.floor(sec / 60);
  const hour = Math.floor(min / 60);
  const day = Math.floor(hour / 24);
  if (day > 0) return isFa ? `${day} روز پیش` : `${day}d ago`;
  if (hour > 0) return isFa ? `${hour} ساعت پیش` : `${hour}h ago`;
  if (min > 0) return isFa ? `${min} دقیقه پیش` : `${min}m ago`;
  return isFa ? "لحظاتی پیش" : "just now";
}
