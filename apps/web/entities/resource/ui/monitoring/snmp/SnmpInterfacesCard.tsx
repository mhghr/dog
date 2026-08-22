"use client";

import { useMemo, useState } from "react";
import { ArrowDown, ArrowUp, Search } from "lucide-react";

import { Card, CardContent, CardHeader, CardTitle } from "@/shared/ui/card";
import { Badge } from "@/shared/ui/badge";
import { Input } from "@/shared/ui/input";
import { Button } from "@/shared/ui/button";
import { Switch } from "@/shared/ui/switch";
import { Skeleton } from "@/shared/ui/skeleton";
import { cn } from "@/shared/utils/cn";
import type { SnmpInterfaceSnapshot } from "./snmp-metrics";
import { operStatusLabel } from "./snmp-metrics";
import { interfaceDisplayName, type SnmpInterfaceSetting } from "./snmp-config";
import type { MetricsRange } from "@/entities/resource/hooks/use-resource";
import { SnmpInterfaceDetail } from "./SnmpInterfaceDetail";

export interface SnmpInterfaceTableRow {
  snapshot: SnmpInterfaceSnapshot;
  setting?: SnmpInterfaceSetting;
}

function formatBits(bps: number | undefined): string {
  if (bps == null) return "—";
  if (bps >= 1_000_000_000) return `${(bps / 1_000_000_000).toFixed(2)} Gbps`;
  if (bps >= 1_000_000) return `${(bps / 1_000_000).toFixed(2)} Mbps`;
  if (bps >= 1_000) return `${(bps / 1_000).toFixed(1)} Kbps`;
  return `${Math.round(bps)} bps`;
}

function formatSpeed(speed: number | undefined): string {
  if (speed == null || speed <= 0) return "—";
  if (speed >= 1_000_000_000) return `${(speed / 1_000_000_000).toFixed(1)} Gbps`;
  return `${(speed / 1_000_000).toFixed(0)} Mbps`;
}

type SortKey = "if_index" | "name" | "status" | "in" | "out" | "util" | "errors";

interface SnmpInterfacesCardProps {
  resourceId: string;
  monitorId: string;
  range: MetricsRange;
  rows: SnmpInterfaceTableRow[];
  isFa: boolean;
  isLoading: boolean;
  onToggleMonitor: (row: SnmpInterfaceTableRow, next: boolean) => void;
}

export function SnmpInterfacesCard({
  resourceId,
  monitorId,
  range,
  rows,
  isFa,
  isLoading,
  onToggleMonitor,
}: SnmpInterfacesCardProps) {
  const t = (en: string, fa: string) => (isFa ? fa : en);
  const [search, setSearch] = useState("");
  const [statusFilter, setStatusFilter] = useState<"all" | "up" | "down">("all");
  const [sortKey, setSortKey] = useState<SortKey>("if_index");
  const [sortDir, setSortDir] = useState<"asc" | "desc">("asc");
  const [selected, setSelected] = useState<SnmpInterfaceTableRow | null>(null);

  const filtered = useMemo(() => {
    let list = rows;
    const q = search.trim().toLowerCase();
    if (q) {
      list = list.filter((row) => {
        const name = interfaceDisplayName(row.snapshot.if_index, row.snapshot.if_name, row.setting ? [row.setting] : []);
        const alias = row.snapshot.if_alias ?? "";
        const descr = row.snapshot.if_descr ?? "";
        return `${name} ${alias} ${descr}`.toLowerCase().includes(q);
      });
    }
    if (statusFilter === "up") list = list.filter((r) => r.snapshot.if_oper_status === 1);
    if (statusFilter === "down") list = list.filter((r) => r.snapshot.if_oper_status === 2);

    const factor = sortDir === "asc" ? 1 : -1;
    return [...list].sort((a, b) => {
      switch (sortKey) {
        case "if_index":
          return (a.snapshot.if_index - b.snapshot.if_index) * factor;
        case "name":
          return (
            interfaceDisplayName(a.snapshot.if_index, a.snapshot.if_name, a.setting ? [a.setting] : []).localeCompare(
              interfaceDisplayName(b.snapshot.if_index, b.snapshot.if_name, b.setting ? [b.setting] : []),
            ) * factor
          );
        case "status":
          return ((a.snapshot.if_oper_status ?? 0) - (b.snapshot.if_oper_status ?? 0)) * factor;
        case "in":
          return ((a.snapshot.in_bps ?? 0) - (b.snapshot.in_bps ?? 0)) * factor;
        case "out":
          return ((a.snapshot.out_bps ?? 0) - (b.snapshot.out_bps ?? 0)) * factor;
        case "util":
          return ((a.snapshot.utilization_percent ?? 0) - (b.snapshot.utilization_percent ?? 0)) * factor;
        case "errors":
          return ((a.snapshot.if_in_errors ?? 0) - (b.snapshot.if_in_errors ?? 0)) * factor;
        default:
          return 0;
      }
    });
  }, [rows, search, statusFilter, sortKey, sortDir]);

  const header = (key: SortKey, label: string, className?: string) => (
    <th
      className={cn("cursor-pointer select-none whitespace-nowrap py-2 pe-2 font-semibold", className)}
      onClick={() => {
        if (sortKey === key) setSortDir((d) => (d === "asc" ? "desc" : "asc"));
        else {
          setSortKey(key);
          setSortDir("asc");
        }
      }}
    >
      <span className="inline-flex items-center gap-1">
        {label}
        {sortKey === key && (sortDir === "asc" ? <ArrowUp className="size-3" /> : <ArrowDown className="size-3" />)}
      </span>
    </th>
  );

  return (
    <Card variant="bordered" className="shadow-subtle">
      <CardHeader className="flex-row flex-wrap items-center justify-between gap-3 space-y-0 px-5 pt-4">
        <CardTitle className="text-sm font-semibold text-foreground">
          {t("Interfaces", "اینترفیس‌ها")}
          <span className="ms-2 text-xs font-normal text-muted-foreground">
            {rows.length} {t("interfaces", "اینترفیس")}
          </span>
        </CardTitle>
        <div className="flex flex-wrap items-center gap-2">
          <div className="relative">
            <Search className="absolute start-2.5 top-1/2 size-3.5 -translate-y-1/2 text-muted-foreground/60" />
            <Input
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder={t("Search interfaces", "جستجوی اینترفیس")}
              className="h-8 w-48 ps-8 text-xs"
              dir="auto"
            />
          </div>
          <div className="inline-flex items-center gap-0.5 rounded-lg border border-border/60 bg-muted/25 p-0.5">
            {(["all", "up", "down"] as const).map((f) => (
              <Button
                key={f}
                type="button"
                size="sm"
                variant="ghost"
                className={cn(
                  "h-6 px-2.5 text-xs",
                  statusFilter === f ? "bg-card text-foreground shadow-sm" : "text-muted-foreground",
                )}
                onClick={() => setStatusFilter(f)}
              >
                {f === "all" ? t("All", "همه") : f === "up" ? t("Up", "بالا") : t("Down", "پایین")}
              </Button>
            ))}
          </div>
        </div>
      </CardHeader>
      <CardContent className="px-4 pb-4 pt-1">
        {isLoading ? (
          <Skeleton className="h-56 w-full rounded-lg" />
        ) : rows.length === 0 ? (
          <p className="py-10 text-center text-sm text-muted-foreground">
            {t(
              "No interfaces discovered — run discovery from the settings",
              "اینترفیسی کشف نشده — از تنظیمات Discovery را اجرا کنید",
            )}
          </p>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full border-collapse text-left">
              <thead>
                <tr className="border-b border-border/60 text-[11px] uppercase tracking-[0.05em] text-muted-foreground">
                  {header("name", t("Interface", "اینترفیس"))}
                  {header("status", t("Status", "وضعیت"))}
                  {header("if_index", t("Speed", "سرعت"), "text-right")}
                  {header("in", t("Inbound", "ورودی"), "text-right")}
                  {header("out", t("Outbound", "خروجی"), "text-right")}
                  {header("util", t("Util", "مصرف"), "text-right")}
                  {header("errors", t("Errors", "خطا"), "text-right")}
                  <th className="py-2 pe-2 text-right font-semibold">{t("Discards", "افت")}</th>
                  <th className="py-2 font-semibold">{t("Monitor", "مانیتور")}</th>
                </tr>
              </thead>
              <tbody>
                {filtered.map((row) => {
                  const s = row.snapshot;
                  const name = interfaceDisplayName(s.if_index, s.if_name, row.setting ? [row.setting] : []);
                  const up = s.if_oper_status === 1;
                  const util = s.utilization_percent ?? 0;
                  const utilTone = util >= 95 ? "bg-rose-500" : util >= 80 ? "bg-amber-500" : "bg-emerald-500";
                  return (
                    <tr
                      key={s.if_index}
                      onClick={() => setSelected(row)}
                      className="cursor-pointer border-b border-border/40 transition-colors last:border-0 hover:bg-muted/30"
                    >
                      <td className="py-2.5 pe-2">
                        <div className="flex min-w-0 flex-col">
                          <span className="truncate text-sm font-medium text-foreground" dir="auto">
                            {name}
                          </span>
                          {s.if_alias && (
                            <span className="truncate text-[10px] text-muted-foreground" dir="auto">
                              {s.if_alias}
                            </span>
                          )}
                        </div>
                      </td>
                      <td className="py-2.5 pe-2">
                        <Badge
                          variant="outline"
                          className={cn(
                            "px-2 py-0.5 text-[10px]",
                            up ? "border-emerald-500/30 bg-emerald-500/10 text-emerald-500" : "border-destructive/30 bg-destructive/10 text-destructive",
                          )}
                        >
                          {operStatusLabel(s.if_oper_status)}
                        </Badge>
                      </td>
                      <td className="py-2.5 pe-2 text-right text-xs tabular-nums text-muted-foreground" dir="ltr">
                        {formatSpeed(s.if_speed_bps)}
                      </td>
                      <td className="py-2.5 pe-2 text-right text-xs tabular-nums text-foreground" dir="ltr">
                        <span className="text-emerald-500">↓</span> {formatBits(s.in_bps)}
                      </td>
                      <td className="py-2.5 pe-2 text-right text-xs tabular-nums text-foreground" dir="ltr">
                        <span className="text-sky-500">↑</span> {formatBits(s.out_bps)}
                      </td>
                      <td className="py-2.5 pe-2">
                        <div className="flex items-center justify-end gap-1.5">
                          <div className="h-1.5 w-16 overflow-hidden rounded-full bg-muted/40">
                            <div className={`h-full rounded-full ${utilTone}`} style={{ width: `${Math.min(100, util)}%` }} />
                          </div>
                          <span className="w-10 text-right text-[11px] tabular-nums text-muted-foreground" dir="ltr">
                            {util.toFixed(0)}%
                          </span>
                        </div>
                      </td>
                      <td className="py-2.5 pe-2 text-right text-xs tabular-nums text-muted-foreground" dir="ltr">
                        {((s.if_in_errors ?? 0) + (s.if_out_errors ?? 0)).toLocaleString()}
                      </td>
                      <td className="py-2.5 pe-2 text-right text-xs tabular-nums text-muted-foreground" dir="ltr">
                        {((s.if_in_discards ?? 0) + (s.if_out_discards ?? 0)).toLocaleString()}
                      </td>
                      <td className="py-2.5">
                        <Switch
                          checked={row.setting ? (row.setting.monitor ?? true) : true}
                          onCheckedChange={(next) => {
                            // Stop the click from opening the detail drawer.
                            setSelected(null);
                            onToggleMonitor(row, next);
                          }}
                        />
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </CardContent>

      {selected && (
        <SnmpInterfaceDetail
          resourceId={resourceId}
          monitorId={monitorId}
          range={range}
          row={selected}
          isFa={isFa}
          onOpenChange={(open) => {
            if (!open) setSelected(null);
          }}
        />
      )}
    </Card>
  );
}
