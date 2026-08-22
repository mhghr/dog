"use client";

import { useState } from "react";
import { Check, Loader2, Radar } from "lucide-react";
import { toast } from "sonner";

import { Button } from "@/shared/ui/button";
import { cn } from "@/shared/utils/cn";
import { resourcesApi, type SnmpDiscovery, type SnmpTaskResponse } from "@/entities/resource/api/resource.api";
import { apiErrorMessage } from "@/shared/api/error-message";

const DISCOVERY_STAGES = [
  { key: "system", en: "System information", fa: "اطلاعات سیستم" },
  { key: "interfaces", en: "Interfaces", fa: "اینترفیس‌ها" },
  { key: "cpu", en: "CPU", fa: "پردازنده" },
  { key: "memory", en: "Memory", fa: "حافظه" },
  { key: "temperature", en: "Temperature", fa: "دما" },
  { key: "fan", en: "Fan", fa: "فن" },
  { key: "power", en: "Power supply", fa: "منبع تغذیه" },
  { key: "hardware", en: "Hardware sensors", fa: "سنسورهای سخت‌افزار" },
];

export function SnmpDiscoveryStep({
  resourceId,
  monitorId,
  isFa,
  discovery,
  onDiscovery,
  onSelected,
  pollTask,
}: {
  resourceId: string;
  monitorId?: string;
  isFa: boolean;
  discovery: SnmpDiscovery | null;
  onDiscovery: (discovery: SnmpDiscovery) => void;
  onSelected: (ids: Set<number>) => void;
  pollTask: (taskId: string, onUpdate: (t: SnmpTaskResponse) => void) => Promise<SnmpTaskResponse>;
}) {
  const t = (en: string, fa: string) => (isFa ? fa : en);
  const [running, setRunning] = useState(false);
  const [stageIndex, setStageIndex] = useState(0);

  const run = async () => {
    if (!monitorId) return;
    setRunning(true);
    setStageIndex(0);
    try {
      const { task_id } = await resourcesApi.snmpDiscover(resourceId, monitorId);
      const interval = setInterval(() => setStageIndex((i) => Math.min(i + 1, DISCOVERY_STAGES.length - 1)), 700);
      const task = await pollTask(task_id, () => {});
      clearInterval(interval);

      if (task.status === "success" && task.result?.ok) {
        // Apply the discovery to the monitor (persist cache + interface rows).
        await resourcesApi.snmpApplyTask(task_id);
        const raw = task.result.discovery;
        const parsed: SnmpDiscovery =
          typeof raw === "string" ? JSON.parse(raw) : (raw as SnmpDiscovery);
        onDiscovery(parsed);
        // Default selection: up interfaces that are not loopbacks.
        const selected = new Set<number>();
        for (const inf of parsed.interfaces) {
          const lower = `${inf.if_name ?? ""} ${inf.if_descr ?? ""}`.toLowerCase();
          if (inf.if_oper_status === 1 && !lower.includes("loopback")) {
            selected.add(inf.if_index);
          }
        }
        onSelected(selected);
      } else {
        throw new Error(task.error || task.result?.detail || "discovery failed");
      }
    } catch (err) {
      const msg = apiErrorMessage(err, isFa);
      toast.error(msg.title, { description: msg.description });
    } finally {
      setRunning(false);
    }
  };

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h3 className="text-sm font-semibold text-foreground">{t("Discovery", "کشف دستگاه")}</h3>
          <p className="mt-0.5 text-xs text-muted-foreground">
            {t(
              "Walks the device MIB once and caches the result. Routine polls only fetch the needed OIDs.",
              "MIB دستگاه یک‌بار Walk و نتیجه کش می‌شود. Pollهای معمولی فقط OIDهای لازم را می‌گیرند.",
            )}
          </p>
        </div>
        <Button type="button" size="sm" disabled={!monitorId || running} onClick={run}>
          {running ? <Loader2 className="size-4 animate-spin" /> : <Radar className="size-4" />}
          {running ? t("Discovering...", "در حال کشف...") : t("Run Discovery", "اجرای کشف")}
        </Button>
      </div>

      {running && (
        <div className="flex flex-col gap-2 rounded-lg border border-border/40 p-3">
          {DISCOVERY_STAGES.map((stage, index) => {
            const active = index === stageIndex;
            const done = index < stageIndex;
            return (
              <div key={stage.key} className="flex items-center gap-2 text-xs">
                {done ? (
                  <Check className="size-3.5 text-emerald-500" />
                ) : active ? (
                  <Loader2 className="size-3.5 animate-spin text-primary" />
                ) : (
                  <span className="size-3.5 rounded-full border border-border/50" />
                )}
                <span className={cn(done ? "text-muted-foreground line-through" : active ? "text-foreground" : "text-muted-foreground/60")}>
                  {isFa ? stage.fa : stage.en}
                </span>
              </div>
            );
          })}
        </div>
      )}

      {discovery && (
        <div className="flex flex-col gap-3">
          <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
            <Count label={t("Interfaces", "اینترفیس‌ها")} value={discovery.interfaces.length} />
            <Count label={t("Temp sensors", "سنسور دما")} value={discovery.sensors.filter((s) => s.sensor_type === "temperature").length} />
            <Count label={t("Fan sensors", "سنسور فن")} value={discovery.sensors.filter((s) => s.sensor_type === "fan").length} />
            <Count label={t("Power sensors", "سنسور برق")} value={discovery.sensors.filter((s) => s.sensor_type === "power").length} />
          </div>
          <div className="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
            <span className="font-semibold text-foreground" dir="auto">
              {discovery.device.sys_name || "—"}
            </span>
            <span className="font-mono" dir="ltr">
              {discovery.device.vendor} {discovery.device.model}
            </span>
            {!discovery.hardware_ok && (
              <span className="rounded-full border border-amber-500/30 bg-amber-500/10 px-2 py-0.5 text-[10px] text-amber-500">
                {t("Partial discovery — some MIBs unsupported", "کشف ناقص — برخی MIBها پشتیبانی نمی‌شوند")}
              </span>
            )}
          </div>
        </div>
      )}
    </div>
  );
}

function Count({ label, value }: { label: string; value: number }) {
  return (
    <div className="flex flex-col gap-0.5 rounded-lg border border-border/40 px-3 py-2">
      <span className="text-xl font-bold tabular-nums text-foreground" dir="ltr">
        {value}
      </span>
      <span className="text-[10px] text-muted-foreground">{label}</span>
    </div>
  );
}
