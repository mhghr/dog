"use client";

import { useState } from "react";
import { Check, Loader2, PlugZap, XCircle } from "lucide-react";

import { Button } from "@/shared/ui/button";
import { cn } from "@/shared/utils/cn";
import { resourcesApi, type SnmpTaskResponse } from "@/entities/resource/api/resource.api";
import { apiErrorMessage } from "@/shared/api/error-message";

const STATE_LABEL: Record<string, { en: string; fa: string }> = {
  success: { en: "Connected", fa: "متصل" },
  device_unreachable: { en: "Device unreachable", fa: "دستگاه در دسترس نیست" },
  snmp_timeout: { en: "SNMP timeout", fa: "تایم‌اوت SNMP" },
  authentication_failed: { en: "Authentication failed", fa: "احراز هویت ناموفق" },
  authorization_failed: { en: "Authorization failed", fa: "مجوز ناموفق" },
  invalid_oid: { en: "Invalid OID", fa: "OID نامعتبر" },
  unsupported_oid: { en: "Unsupported OID", fa: "OID پشتیبانی نشده" },
  invalid_config: { en: "Invalid configuration", fa: "پیکربندی نامعتبر" },
  no_response: { en: "No response", fa: "پاسخی دریافت نشد" },
};

function ResultRow({ label, value, ok }: { label: string; value: string; ok: boolean }) {
  return (
    <div className="flex items-center gap-2 rounded-lg border border-border/40 px-3 py-2">
      {ok ? <Check className="size-3.5 shrink-0 text-emerald-500" /> : <XCircle className="size-3.5 shrink-0 text-destructive" />}
      <span className="min-w-0 flex-1 truncate text-xs text-muted-foreground" dir="auto">
        {label}
      </span>
      <span className={cn("text-xs tabular-nums", ok ? "text-foreground" : "text-destructive")} dir="ltr">
        {value}
      </span>
    </div>
  );
}

export function SnmpTestStep({
  resourceId,
  monitorId,
  isFa,
  result,
  onResult,
  pollTask,
}: {
  resourceId: string;
  monitorId?: string;
  isFa: boolean;
  result: SnmpTaskResponse | null;
  onResult: (result: SnmpTaskResponse | null) => void;
  pollTask: (taskId: string, onUpdate: (t: SnmpTaskResponse) => void) => Promise<SnmpTaskResponse>;
}) {
  const t = (en: string, fa: string) => (isFa ? fa : en);
  const [running, setRunning] = useState(false);

  const run = async () => {
    if (!monitorId) return;
    setRunning(true);
    onResult(null);
    try {
      const { task_id } = await resourcesApi.snmpTestConnection(resourceId, monitorId);
      const task = await pollTask(task_id, (update) => onResult(update));
      onResult(task);
    } catch (err) {
      const msg = apiErrorMessage(err, isFa);
      onResult({
        task_id: "",
        kind: "test",
        status: "failed",
        created_at: new Date().toISOString(),
        error: msg.description ?? msg.title,
      });
    } finally {
      setRunning(false);
    }
  };

  const r = result?.result;
  const steps = r?.steps ?? [];

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h3 className="text-sm font-semibold text-foreground">{t("Test SNMP Connection", "تست اتصال SNMP")}</h3>
          <p className="mt-0.5 text-xs text-muted-foreground">
            {t(
              "A real SNMP GET is sent from the Dog SNMP collector to the device.",
              "یک درخواست واقعی SNMP GET از کلکتور SNMP دوج به دستگاه ارسال می‌شود.",
            )}
          </p>
        </div>
        <Button type="button" size="sm" disabled={!monitorId || running} onClick={run}>
          {running ? <Loader2 className="size-4 animate-spin" /> : <PlugZap className="size-4" />}
          {running ? t("Testing...", "در حال تست...") : t("Test SNMP Connection", "تست اتصال SNMP")}
        </Button>
      </div>

      {result?.status === "running" || result?.status === "pending" ? (
        <div className="flex items-center gap-2 rounded-lg border border-border/40 px-3 py-2 text-xs text-muted-foreground">
          <Loader2 className="size-3.5 animate-spin text-primary" />
          {t("Collector is testing the device...", "کلکتور در حال تست دستگاه است...")}
        </div>
      ) : null}

      {r && (
        <div className="flex flex-col gap-2">
          {steps.length > 0 && (
            <div className="flex flex-col gap-1.5">
              {steps.map((step, index) => (
                <ResultRow key={index} label={step} value="" ok={!step.toLowerCase().includes("failed")} />
              ))}
            </div>
          )}

          <div className="mt-1 flex flex-wrap items-center gap-2 rounded-lg border border-border/40 px-3 py-2">
            <span className="text-xs font-semibold text-foreground">
              {STATE_LABEL[r.state] ? (isFa ? STATE_LABEL[r.state].fa : STATE_LABEL[r.state].en) : r.state}
            </span>
            <span className="text-[10px] text-muted-foreground" dir="ltr">{r.state}</span>
            <span className="ms-auto text-[10px] tabular-nums text-muted-foreground" dir="ltr">
              {r.duration_millis != null ? `${r.duration_millis} ms` : ""}
            </span>
          </div>

          {r.ok && (
            <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
              {r.sys_name && (
                <div className="rounded-lg border border-border/40 px-3 py-2">
                  <span className="text-[10px] text-muted-foreground">{t("Device name", "نام دستگاه")}</span>
                  <p className="truncate text-sm font-medium text-foreground" dir="ltr">{r.sys_name}</p>
                </div>
              )}
              {r.sys_descr && (
                <div className="rounded-lg border border-border/40 px-3 py-2">
                  <span className="text-[10px] text-muted-foreground">{t("Description", "توضیحات")}</span>
                  <p className="truncate text-sm font-medium text-foreground" dir="auto">{r.sys_descr}</p>
                </div>
              )}
              {r.sys_object_id && (
                <div className="rounded-lg border border-border/40 px-3 py-2">
                  <span className="text-[10px] text-muted-foreground">sysObjectID</span>
                  <p className="truncate text-sm font-medium text-foreground" dir="ltr">{r.sys_object_id}</p>
                </div>
              )}
              {r.uptime && (
                <div className="rounded-lg border border-border/40 px-3 py-2">
                  <span className="text-[10px] text-muted-foreground">{t("Uptime (s)", "آپ‌تایم (ثانیه)")}</span>
                  <p className="truncate text-sm font-medium text-foreground" dir="ltr">{r.uptime}</p>
                </div>
              )}
            </div>
          )}

          {!r.ok && r.detail && (
            <p className="rounded-lg border border-destructive/20 bg-destructive/5 px-3 py-2 text-xs text-destructive" dir="auto">
              {r.detail}
            </p>
          )}
        </div>
      )}
    </div>
  );
}
