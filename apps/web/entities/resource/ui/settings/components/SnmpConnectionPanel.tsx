"use client";

import { useState } from "react";
import { Cable, RefreshCw, Search, Wifi } from "lucide-react";

import { Button } from "@/shared/ui/button";
import { Input } from "@/shared/ui/input";
import { Switch } from "@/shared/ui/switch";
import { Badge } from "@/shared/ui/badge";
import { Skeleton } from "@/shared/ui/skeleton";
import { cn } from "@/shared/utils/cn";
import { toast } from "sonner";
import {
  resourcesApi,
  type SnmpDiscovery,
  type SnmpInterfaceRow,
  type SnmpTaskResponse,
} from "@/entities/resource/api/resource.api";
import { apiErrorMessage } from "@/shared/api/error-message";

async function pollTask(
  taskId: string,
  setTask: (task: SnmpTaskResponse) => void,
): Promise<SnmpTaskResponse> {
  const started = Date.now();
  for (;;) {
    const task = await resourcesApi.snmpGetTask(taskId);
    setTask(task);
    if (task.status === "success" || task.status === "failed") return task;
    if (Date.now() - started > 90_000) throw new Error("task timeout");
    await new Promise((resolve) => setTimeout(resolve, 1200));
  }
}

// Connection & Discovery panel for SNMP network-device monitors. Test
// Connection runs a real SNMP GET against the saved device configuration;
// Discovery walks the MIB and lets the operator choose which interfaces to
// monitor, rename, or ignore. Secrets stay encrypted in the monitor config —
// they never travel through this panel.
export function SnmpConnectionPanel({
  resourceId,
  monitorId,
  isFa,
}: {
  resourceId: string;
  monitorId: string;
  isFa: boolean;
}) {
  const t = (en: string, fa: string) => (isFa ? fa : en);
  const [testing, setTesting] = useState(false);
  const [discovering, setDiscovering] = useState(false);
  const [testResult, setTestResult] = useState<{
    ok: boolean;
    state: string;
    detail: string;
    sys_name?: string;
  } | null>(null);
  const [discovery, setDiscovery] = useState<SnmpDiscovery | null>(null);
  const [rows, setRows] = useState<SnmpInterfaceRow[] | null>(null);
  const [savingIndex, setSavingIndex] = useState<number | null>(null);

  const runTest = async () => {
    setTesting(true);
    setTestResult(null);
    try {
      const { task_id } = await resourcesApi.snmpTestConnection(resourceId, monitorId);
      const task = await pollTask(task_id, () => {});
      const result = task.result;
      if (!result) throw new Error("no result");
      setTestResult({
        ok: result.ok,
        state: result.state,
        detail: result.detail ?? "",
        sys_name: result.sys_name,
      });
    } catch (err) {
      const msg = apiErrorMessage(err, isFa);
      setTestResult({ ok: false, state: "error", detail: msg.description ?? msg.title });
    } finally {
      setTesting(false);
    }
  };

  const runDiscovery = async () => {
    setDiscovering(true);
    try {
      const { task_id } = await resourcesApi.snmpDiscover(resourceId, monitorId);
      const task = await pollTask(task_id, () => {});
      if (task.status !== "success" || !task.result?.ok) {
        throw new Error(task.result?.detail || "discovery failed");
      }
      await resourcesApi.snmpApplyTask(task_id);
      const raw = task.result.discovery;
      const parsed: SnmpDiscovery = typeof raw === "string" ? JSON.parse(raw) : (raw as SnmpDiscovery);
      setDiscovery(parsed);
      const ifaces = await resourcesApi.snmpListInterfaces(resourceId, monitorId);
      setRows(ifaces.items);
      toast.success(t("Discovery complete", "کشف دستگاه انجام شد"), {
        description: t(
          `${parsed.interfaces.length} interfaces found`,
          `${parsed.interfaces.length} اینترفیس پیدا شد`,
        ),
      });
    } catch (err) {
      const msg = apiErrorMessage(err, isFa);
      toast.error(msg.title, { description: msg.description });
    } finally {
      setDiscovering(false);
    }
  };

  const toggleMonitor = async (row: SnmpInterfaceRow, next: boolean) => {
    setSavingIndex(row.if_index);
    try {
      const updated = await resourcesApi.snmpUpdateInterface(resourceId, monitorId, row.if_index, {
        monitor: next,
        ignore: !next,
      });
      setRows((prev) => (prev ?? []).map((r) => (r.if_index === row.if_index ? { ...r, ...updated } : r)));
    } catch {
      toast.error(t("Failed to update interface", "به‌روزرسانی اینترفیس ناموفق بود"));
    } finally {
      setSavingIndex(null);
    }
  };

  const updateDisplayName = async (row: SnmpInterfaceRow, displayName: string) => {
    setSavingIndex(row.if_index);
    try {
      const updated = await resourcesApi.snmpUpdateInterface(resourceId, monitorId, row.if_index, {
        display_name: displayName,
      });
      setRows((prev) => (prev ?? []).map((r) => (r.if_index === row.if_index ? { ...r, ...updated } : r)));
    } catch {
      toast.error(t("Failed to rename interface", "تغییر نام اینترفیس ناموفق بود"));
    } finally {
      setSavingIndex(null);
    }
  };

  return (
    <div className="flex flex-col gap-4">
      {/* Connection test */}
      <div className="rounded-xl border border-border/50 p-4">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div className="flex items-center gap-2">
            <Cable className="size-4 text-primary" />
            <span className="text-sm font-semibold text-foreground">
              {t("Connection", "اتصال")}
            </span>
          </div>
          <Button type="button" size="sm" variant="outline" disabled={testing} onClick={runTest}>
            {testing ? t("Testing...", "در حال تست...") : t("Test Connection", "تست اتصال")}
          </Button>
        </div>

        {testResult && (
          <div
            className={cn(
              "mt-3 flex flex-wrap items-center gap-2 rounded-lg border px-3 py-2 text-xs",
              testResult.ok
                ? "border-emerald-500/30 bg-emerald-500/5 text-emerald-600 dark:text-emerald-400"
                : "border-destructive/30 bg-destructive/5 text-destructive",
            )}
          >
            <span className="font-semibold">{testResult.ok ? t("Connected", "متصل") : t("Failed", "ناموفق")}</span>
            <span className="text-muted-foreground" dir="ltr">
              {testResult.state}
            </span>
            {testResult.sys_name && (
              <span className="font-mono text-muted-foreground" dir="ltr">
                · {testResult.sys_name}
              </span>
            )}
          </div>
        )}
      </div>

      {/* Discovery */}
      <div className="rounded-xl border border-border/50 p-4">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div className="flex items-center gap-2">
            <Search className="size-4 text-primary" />
            <span className="text-sm font-semibold text-foreground">
              {t("Discovery", "کشف دستگاه")}
            </span>
          </div>
          <Button type="button" size="sm" variant="outline" disabled={discovering} onClick={runDiscovery}>
            {discovering ? (
              <RefreshCw className="size-3.5 animate-spin" />
            ) : (
              <Wifi className="size-3.5" />
            )}
            {discovering ? t("Discovering...", "در حال کشف...") : t("Run Discovery", "اجرای کشف")}
          </Button>
        </div>

        {discovering ? (
          <Skeleton className="mt-3 h-40 w-full rounded-lg" />
        ) : rows === null ? (
          <p className="mt-3 text-xs text-muted-foreground">
            {t(
              "Discovery reads sysName, interfaces (IF-MIB) and sensors. Routines polls only fetch the needed OIDs.",
              "کشف دستگاه sysName، اینترفیس‌ها (IF-MIB) و سنسورها را می‌خواند. Pollهای عادی فقط OIDهای لازم را می‌گیرند.",
            )}
          </p>
        ) : rows.length === 0 ? (
          <p className="mt-3 text-xs text-muted-foreground">
            {t("No interfaces returned", "اینترفیسی برگردانده نشد")}
          </p>
        ) : (
          <ul className="mt-3 flex flex-col divide-y divide-border/40">
            {rows.map((row) => (
              <li key={row.if_index} className="flex flex-wrap items-center gap-3 py-2.5">
                <span className="min-w-0 flex-1">
                  <span className="flex items-center gap-2">
                    <span className="truncate text-sm font-medium text-foreground" dir="auto">
                      {row.display_name || row.if_name || `if${row.if_index}`}
                    </span>
                    <Badge
                      variant="outline"
                      className={cn(
                        "px-1.5 py-0 text-[9px]",
                        row.last_oper_status === 1
                          ? "border-emerald-500/30 text-emerald-500"
                          : "border-destructive/30 text-destructive",
                      )}
                    >
                      {row.last_oper_status === 1 ? "up" : row.last_oper_status === 2 ? "down" : "—"}
                    </Badge>
                  </span>
                  <span className="mt-0.5 flex flex-wrap items-center gap-1.5">
                    <Input
                      value={row.display_name ?? ""}
                      placeholder={row.if_name || `if${row.if_index}`}
                      className="h-7 w-44 text-xs"
                      dir="auto"
                      disabled={savingIndex === row.if_index}
                      onBlur={(e) => {
                        if (e.target.value !== (row.display_name ?? "")) {
                          void updateDisplayName(row, e.target.value);
                        }
                      }}
                    />
                    {row.if_alias && (
                      <span className="text-[10px] text-muted-foreground" dir="auto">
                        {row.if_alias}
                      </span>
                    )}
                  </span>
                </span>
                <span className="flex items-center gap-2">
                  <span className="text-[11px] text-muted-foreground">
                    {t("Monitor", "مانیتور")}
                  </span>
                  <Switch
                    checked={row.monitor}
                    disabled={savingIndex === row.if_index}
                    onCheckedChange={(next) => void toggleMonitor(row, next)}
                  />
                </span>
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  );
}
