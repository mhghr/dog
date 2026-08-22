"use client";

import { useMemo } from "react";
import { Checkbox } from "@/shared/ui/checkbox";
import { Button } from "@/shared/ui/button";
import { Badge } from "@/shared/ui/badge";
import { cn } from "@/shared/utils/cn";
import type { SnmpDiscovery } from "@/entities/resource/api/resource.api";

const SYSTEM_METRICS = [
  { key: "cpu", en: "CPU Usage", fa: "مصرف پردازنده" },
  { key: "memory", en: "Memory Usage", fa: "مصرف حافظه" },
  { key: "uptime", en: "Uptime", fa: "آپ‌تایم" },
  { key: "temperature", en: "Temperature", fa: "دما" },
  { key: "fan", en: "Fan Status", fa: "وضعیت فن" },
  { key: "power", en: "Power Supply", fa: "منبع تغذیه" },
  { key: "hardware", en: "Hardware Health", fa: "سلامت سخت‌افزار" },
];

function operStatusLabel(status: number | undefined): string {
  switch (status) {
    case 1: return "up";
    case 2: return "down";
    default: return String(status ?? "—");
  }
}

export function SnmpMetricsStep({
  discovery,
  isFa,
  selectedIds,
  onSelected,
}: {
  discovery: SnmpDiscovery | null;
  isFa: boolean;
  selectedIds: Set<number>;
  onSelected: (ids: Set<number>) => void;
}) {
  const t = (en: string, fa: string) => (isFa ? fa : en);
  const interfaces = discovery?.interfaces ?? [];

  const availableSensors = useMemo(() => {
    const types = new Set((discovery?.sensors ?? []).map((s) => s.sensor_type));
    return {
      temperature: types.has("temperature"),
      fan: types.has("fan"),
      power: types.has("power"),
    };
  }, [discovery]);

  const toggle = (ifIndex: number) => {
    const next = new Set(selectedIds);
    if (next.has(ifIndex)) next.delete(ifIndex);
    else next.add(ifIndex);
    onSelected(next);
  };

  const selectAllUp = () => {
    const next = new Set<number>();
    for (const inf of interfaces) {
      const lower = `${inf.if_name ?? ""} ${inf.if_descr ?? ""}`.toLowerCase();
      if (inf.if_oper_status === 1 && !lower.includes("loopback")) next.add(inf.if_index);
    }
    onSelected(next);
  };
  const selectAll = () => onSelected(new Set(interfaces.map((i) => i.if_index)));
  const clearAll = () => onSelected(new Set());

  return (
    <div className="flex flex-col gap-5">
      {/* System metrics */}
      <div>
        <h3 className="text-sm font-semibold text-foreground">{t("System Metrics", "متریک‌های سیستم")}</h3>
        <p className="mt-0.5 text-xs text-muted-foreground">
          {t("Only metrics the device actually supports can be enabled.", "فقط متریک‌هایی که دستگاه واقعاً پشتیبانی می‌کند قابل فعال‌سازی هستند.")}
        </p>
        <div className="mt-2 grid grid-cols-1 gap-1.5 sm:grid-cols-2">
          {SYSTEM_METRICS.map((metric) => {
            const unsupported =
              (metric.key === "temperature" && !availableSensors.temperature) ||
              (metric.key === "fan" && !availableSensors.fan) ||
              (metric.key === "power" && !availableSensors.power);
            return (
              <div
                key={metric.key}
                className={cn(
                  "flex items-center gap-2 rounded-lg border border-border/40 px-3 py-2",
                  unsupported && "opacity-50",
                )}
              >
                <Checkbox checked={!unsupported} disabled={unsupported} />
                <span className="text-sm text-foreground">{isFa ? metric.fa : metric.en}</span>
                {unsupported && (
                  <Badge variant="outline" className="ms-auto px-1.5 text-[9px] text-muted-foreground">
                    {t("Unsupported", "پشتیبانی نشده")}
                  </Badge>
                )}
              </div>
            );
          })}
        </div>
      </div>

      {/* Interfaces */}
      <div>
        <div className="flex flex-wrap items-center justify-between gap-2">
          <div>
            <h3 className="text-sm font-semibold text-foreground">{t("Interfaces", "اینترفیس‌ها")}</h3>
            <p className="mt-0.5 text-xs text-muted-foreground">
              {t("Up interfaces are selected by default.", "اینترفیس‌های Up به‌صورت پیش‌فرض انتخاب شده‌اند.")}
            </p>
          </div>
          <div className="flex items-center gap-1.5">
            <Button type="button" size="sm" variant="outline" className="h-7 text-xs" onClick={selectAllUp}>
              {t("Select All Up", "انتخاب همه Up")}
            </Button>
            <Button type="button" size="sm" variant="outline" className="h-7 text-xs" onClick={selectAll}>
              {t("Select All", "انتخاب همه")}
            </Button>
            <Button type="button" size="sm" variant="outline" className="h-7 text-xs" onClick={clearAll}>
              {t("Clear All", "پاک کردن همه")}
            </Button>
          </div>
        </div>

        <div className="mt-2 max-h-80 overflow-y-auto rounded-lg border border-border/40">
          <table className="w-full border-collapse text-left">
            <thead className="sticky top-0 bg-card">
              <tr className="border-b border-border/60 text-[11px] uppercase tracking-wide text-muted-foreground">
                <th className="py-2 pe-2 ps-3 font-semibold">{t("Sel", "انتخاب")}</th>
                <th className="py-2 pe-2 font-semibold">{t("Interface", "اینترفیس")}</th>
                <th className="hidden py-2 pe-2 font-semibold sm:table-cell">{t("Alias", "Alias")}</th>
                <th className="py-2 pe-2 font-semibold">{t("Admin", "Admin")}</th>
                <th className="py-2 pe-2 font-semibold">{t("Oper", "Oper")}</th>
                <th className="hidden py-2 pe-2 text-right font-semibold sm:table-cell">{t("Speed", "سرعت")}</th>
              </tr>
            </thead>
            <tbody>
              {interfaces.map((inf) => {
                const selected = selectedIds.has(inf.if_index);
                const isLoopback = `${inf.if_name ?? ""} ${inf.if_descr ?? ""}`.toLowerCase().includes("loopback");
                return (
                  <tr key={inf.if_index} className="cursor-pointer border-b border-border/40 transition-colors last:border-0 hover:bg-muted/30" onClick={() => toggle(inf.if_index)}>
                    <td className="py-2 pe-2 ps-3">
                      <Checkbox checked={selected} onCheckedChange={() => toggle(inf.if_index)} />
                    </td>
                    <td className="py-2 pe-2">
                      <div className="flex min-w-0 items-center gap-1.5">
                        <span className="truncate text-sm text-foreground" dir="ltr">{inf.if_name || `if${inf.if_index}`}</span>
                        {isLoopback && <Badge variant="outline" className="px-1 text-[9px] text-muted-foreground">loopback</Badge>}
                      </div>
                    </td>
                    <td className="hidden truncate py-2 pe-2 text-xs text-muted-foreground sm:table-cell" dir="ltr">{inf.if_alias || "—"}</td>
                    <td className="py-2 pe-2 text-xs" dir="ltr">{inf.if_admin_status ?? "—"}</td>
                    <td className="py-2 pe-2">
                      <span className={cn("text-xs font-medium", inf.if_oper_status === 1 ? "text-emerald-500" : inf.if_oper_status === 2 ? "text-destructive" : "text-muted-foreground")} dir="ltr">
                        {operStatusLabel(inf.if_oper_status)}
                      </span>
                    </td>
                    <td className="hidden py-2 pe-2 text-right text-xs tabular-nums text-muted-foreground sm:table-cell" dir="ltr">
                      {inf.if_speed_bps ? `${(inf.if_speed_bps / 1_000_000_000).toFixed(1)} Gbps` : "—"}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
