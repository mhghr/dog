"use client";

import { useMemo, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { ArrowLeft, ArrowRight, Check } from "lucide-react";

import { Button } from "@/shared/ui/button";
import { Input } from "@/shared/ui/input";
import { Label } from "@/shared/ui/label";
import { cn } from "@/shared/utils/cn";
import { useCreateResourceMonitor, useUpdateResourceMonitor } from "@/entities/resource/hooks/use-resource";
import { resourcesApi, type SnmpDiscovery, type SnmpTaskResponse } from "@/entities/resource/api/resource.api";
import type { Resource } from "@/entities/resource/model/types";
import type { MonitorTypeDef } from "@/entities/resource/model/types";
import type { Monitor } from "@/entities/resource/hooks/types";
import { HealthRulesBuilder, type HealthRuleThreshold, type HealthRulesState } from "../../../settings/components/HealthRulesBuilder";
import type { HealthRuleDef } from "../../../settings/monitoring-schema";
import { SnmpConfigForm } from "./wizard-config";
import { SnmpTestStep } from "./wizard-test";
import { SnmpDiscoveryStep } from "./wizard-discovery";
import { SnmpMetricsStep } from "./wizard-metrics";
import { SnmpPollingStep } from "./wizard-polling";
import { SnmpReviewStep } from "./wizard-review";

const STEP_LABELS: Array<{ en: string; fa: string }> = [
  { en: "SNMP Monitoring", fa: "مانیتورینگ SNMP" },
  { en: "Configuration", fa: "پیکربندی" },
  { en: "Test Connection", fa: "تست اتصال" },
  { en: "Discovery", fa: "کشف دستگاه" },
  { en: "Metrics", fa: "متریک‌ها" },
  { en: "Polling", fa: "زمان‌بندی" },
  { en: "Health Rules", fa: "قوانین سلامت" },
  { en: "Review", fa: "بررسی" },
];

const SNMP_HEALTH_RULES: HealthRuleDef[] = [
  { key: "reachability", label: { en: "Device reachable", fa: "دسترس‌پذیری دستگاه" }, direction: "boolean", defaultEnabled: true },
  { key: "device_health", label: { en: "Hardware health", fa: "سلامت سخت‌افزار" }, direction: "boolean", defaultEnabled: true },
  { key: "cpu_percent", label: { en: "CPU utilization", fa: "مصرف پردازنده" }, unit: "%", direction: "higher_is_worse", defaultEnabled: true, defaults: { warning: 80, critical: 95 } },
  { key: "memory_percent", label: { en: "Memory utilization", fa: "مصرف حافظه" }, unit: "%", direction: "higher_is_worse", defaultEnabled: true, defaults: { warning: 80, critical: 95 } },
  { key: "temperature_celsius", label: { en: "Temperature", fa: "دما" }, unit: "°C", direction: "higher_is_worse", defaultEnabled: false, defaults: { warning: 60, critical: 75 } },
  { key: "interface_oper_status", label: { en: "Interface down", fa: "اینترفیس قطع" }, direction: "boolean", defaultEnabled: true },
  { key: "interface_utilization_percent", label: { en: "Interface utilization", fa: "مصرف پهنای باند" }, unit: "%", direction: "higher_is_worse", defaultEnabled: false, defaults: { warning: 80, critical: 95 } },
];

const defaultHealthRules = (): HealthRulesState => {
  const state: HealthRulesState = {};
  for (const rule of SNMP_HEALTH_RULES) {
    if (rule.direction !== "boolean") {
      state[rule.key] = { warning: rule.defaults?.warning ?? 0, critical: rule.defaults?.critical ?? 0 };
    }
  }
  return state;
};

function pollTask(
  taskId: string,
  onUpdate: (task: SnmpTaskResponse) => void,
): Promise<SnmpTaskResponse> {
  return new Promise((resolve, reject) => {
    const started = Date.now();
    const tick = async () => {
      try {
        const task = await resourcesApi.snmpGetTask(taskId);
        onUpdate(task);
        if (task.status === "success" || task.status === "failed") {
          resolve(task);
          return;
        }
        if (Date.now() - started > 90_000) {
          reject(new Error("task timeout"));
          return;
        }
        setTimeout(tick, 1200);
      } catch (err) {
        reject(err);
      }
    };
    void tick();
  });
}

export function SnmpWizard({
  resourceId,
  resource,
  type,
  isFa,
  onDone,
}: {
  resourceId: string;
  resource: Resource | undefined;
  type: MonitorTypeDef;
  isFa: boolean;
  onDone: () => void;
}) {
  const t = (en: string, fa: string) => (isFa ? fa : en);
  const queryClient = useQueryClient();
  const create = useCreateResourceMonitor(resourceId);
  const update = useUpdateResourceMonitor(resourceId);

  const [step, setStep] = useState(0);
  const [monitorName, setMonitorName] = useState("SNMP Monitoring");
  const [monitor, setMonitor] = useState<Monitor | null>(null);
  const [config, setConfig] = useState<Record<string, unknown>>(() => {
    const target = resource?.target ?? "";
    return {
      host: target,
      port: 161,
      version: "3",
      community: "",
      username: "",
      security_level: "authPriv",
      authentication_protocol: "SHA-256",
      authentication_secret: "",
      privacy_protocol: "AES-256",
      privacy_secret: "",
      timeout_seconds: 3,
      retries: 1,
      max_repetitions: 10,
      discovery_interval_seconds: 86400,
    };
  });
  const [execution, setExecution] = useState({ intervalSeconds: 60, timeoutMillis: 5000, retries: 1, discoveryIntervalSeconds: 86400 });
  const [healthRules, setHealthRules] = useState<HealthRulesState>(defaultHealthRules);
  const [discovery, setDiscovery] = useState<SnmpDiscovery | null>(null);
  const [testResult, setTestResult] = useState<SnmpTaskResponse | null>(null);
  const [selectedIds, setSelectedIds] = useState<Set<number>>(new Set());
  const [creating, setCreating] = useState(false);

  // Keep host bound to the resource target (never re-entered by the user).
  const setField = (key: string, value: unknown) => setConfig((prev) => ({ ...prev, [key]: value }));

  const saveMonitor = async (finalize: boolean) => {
    const configuration: Record<string, unknown> = { ...config, health_rules: healthRules };
    if (resource?.target) {
      configuration.host = resource.target;
    }
    if (discovery) {
      configuration.discovery = discovery;
      configuration.interfaces = discovery.interfaces.map((inf) => ({
        if_index: inf.if_index,
        if_name: inf.if_name,
        if_descr: inf.if_descr,
        if_alias: inf.if_alias,
        monitor: selectedIds.has(inf.if_index),
        ignore: !selectedIds.has(inf.if_index),
      }));
      configuration.monitored_interface_ids = Array.from(selectedIds).sort((a, b) => a - b);
    }

    const input = {
      monitor_type_id: type.id,
      name: monitorName.trim() || "SNMP Monitoring",
      enabled: finalize,
      interval_seconds: execution.intervalSeconds,
      timeout_millis: execution.timeoutMillis,
      retries: execution.retries,
      configuration,
      severity: "warning" as const,
    };

    if (monitor) {
      const updated = await update.mutateAsync({ id: monitor.id, ...input });
      setMonitor(updated);
      return updated;
    }
    const created = await create.mutateAsync(input);
    setMonitor(created);
    return created;
  };

  const next = async () => {
    // Step 1 → 2: nothing persisted yet. Step 2 → 3: create the (disabled)
    // monitor so test/discovery run against the stored encrypted config.
    if (step === 1) {
      if (!monitor) {
        try {
          setCreating(true);
          await saveMonitor(false);
          await queryClient.invalidateQueries({ queryKey: ["resources", resourceId, "monitors"] });
        } catch (err) {
          toast.error(t("Failed to save monitor", "ذخیره مونیتور ناموفق بود"));
          return;
        } finally {
          setCreating(false);
        }
      }
    }
    setStep((s) => Math.min(s + 1, STEP_LABELS.length - 1));
  };

  const back = () => setStep((s) => Math.max(s - 1, 0));

  const finish = async () => {
    try {
      setCreating(true);
      await saveMonitor(true);
      await queryClient.invalidateQueries({ queryKey: ["resources", resourceId, "monitors"] });
      toast.success(t("SNMP monitor created", "مونیتور SNMP ایجاد شد"));
      onDone();
    } catch (err) {
      toast.error(t("Failed to create monitor", "ایجاد مونیتور ناموفق بود"));
    } finally {
      setCreating(false);
    }
  };

  const canNext = useMemo(() => {
    switch (step) {
      case 0:
        return monitorName.trim().length >= 2;
      case 1:
        return Boolean(monitor) || creating;
      case 2:
        return testResult?.status === "success";
      case 3:
        return Boolean(discovery) && selectedIds.size > 0;
      case 4:
        return selectedIds.size > 0;
      case 5:
      case 6:
        return true;
      case 7:
        return Boolean(monitor);
      default:
        return false;
    }
  }, [step, monitorName, monitor, creating, testResult, discovery, selectedIds]);

  const isLast = step === STEP_LABELS.length - 1;

  return (
    <div className="flex flex-col gap-5 rounded-2xl border border-border/50 bg-card p-5 shadow-subtle">
      {/* Progress stepper */}
      <div className="flex flex-wrap items-center gap-1.5">
        {STEP_LABELS.map((label, index) => {
          const done = index < step;
          const active = index === step;
          return (
            <button
              key={index}
              type="button"
              onClick={() => index < step && setStep(index)}
              className={cn(
                "flex items-center gap-1.5 rounded-full border px-2.5 py-1 text-[11px] font-medium transition-colors",
                active
                  ? "border-primary/60 bg-primary/10 text-primary"
                  : done
                    ? "border-emerald-500/30 bg-emerald-500/5 text-emerald-500"
                    : "border-border/60 text-muted-foreground",
              )}
            >
              <span
                className={cn(
                  "grid size-4 shrink-0 place-items-center rounded-full text-[9px]",
                  done ? "bg-emerald-500 text-white" : active ? "bg-primary text-white" : "bg-muted text-muted-foreground",
                )}
              >
                {done ? <Check className="size-2.5" /> : index + 1}
              </span>
              {isFa ? label.fa : label.en}
            </button>
          );
        })}
      </div>

      {/* Step content */}
      <div className="min-h-[360px]">
        {step === 0 && (
          <SnmpInfoStep
            resource={resource}
            isFa={isFa}
            monitorName={monitorName}
            onNameChange={setMonitorName}
          />
        )}
        {step === 1 && (
          <SnmpConfigForm
            config={config}
            isFa={isFa}
            resourceId={resourceId}
            monitorId={monitor?.id}
            onChange={setField}
            credentialConfigured={Boolean(monitor?.configuration?.credential_reference)}
          />
        )}
        {step === 2 && (
          <SnmpTestStep
            resourceId={resourceId}
            monitorId={monitor?.id}
            isFa={isFa}
            result={testResult}
            onResult={setTestResult}
            pollTask={pollTask}
          />
        )}
        {step === 3 && (
          <SnmpDiscoveryStep
            resourceId={resourceId}
            monitorId={monitor?.id}
            isFa={isFa}
            discovery={discovery}
            onDiscovery={setDiscovery}
            onSelected={setSelectedIds}
            pollTask={pollTask}
          />
        )}
        {step === 4 && (
          <SnmpMetricsStep
            discovery={discovery}
            isFa={isFa}
            selectedIds={selectedIds}
            onSelected={setSelectedIds}
          />
        )}
        {step === 5 && (
          <SnmpPollingStep
            execution={execution}
            isFa={isFa}
            onChange={setExecution}
          />
        )}
        {step === 6 && (
          <div className="flex flex-col gap-3">
            <h3 className="text-sm font-semibold text-foreground">
              {t("Health Rules", "قوانین سلامت")}
            </h3>
            <p className="text-xs text-muted-foreground">
              {t(
                "Default thresholds for this network device. Only monitoring-enabled interfaces are covered by the Interface Down rule.",
                "آستانه‌های پیش‌فرض این دستگاه. قانون قطع اینترفیس فقط اینترفیس‌های فعال‌شده را پوشش می‌دهد.",
              )}
            </p>
            <HealthRulesBuilder
              rules={SNMP_HEALTH_RULES.filter((r) => r.direction !== "boolean")}
              state={healthRules}
              isFa={isFa}
              onChange={(key: string, next: HealthRuleThreshold) => setHealthRules((prev) => ({ ...prev, [key]: next }))}
            />
          </div>
        )}
        {step === 7 && (
          <SnmpReviewStep
            resource={resource}
            isFa={isFa}
            monitorName={monitorName}
            config={config}
            execution={execution}
            discovery={discovery}
            selectedIds={selectedIds}
            healthRules={healthRules}
            credentialConfigured={Boolean(monitor?.configuration?.credential_reference)}
          />
        )}
      </div>

      {/* Navigation */}
      <div className="flex items-center justify-between border-t border-border/50 pt-4">
        <Button type="button" variant="ghost" size="sm" disabled={step === 0} onClick={back}>
          <ArrowLeft className="size-4 rtl:rotate-180" />
          {t("Back", "قبلی")}
        </Button>
        {isLast ? (
          <Button type="button" size="sm" disabled={creating || !canNext} onClick={finish}>
            {creating ? t("Creating...", "در حال ایجاد...") : t("Create SNMP Monitor", "ایجاد مونیتور SNMP")}
          </Button>
        ) : (
          <Button type="button" size="sm" disabled={!canNext || creating} onClick={next}>
            {t("Next", "بعدی")}
            <ArrowRight className="size-4 rtl:rotate-180" />
          </Button>
        )}
      </div>
    </div>
  );
}

// Step 1 — read-only resource info + monitor name.
function SnmpInfoStep({
  resource,
  isFa,
  monitorName,
  onNameChange,
}: {
  resource: Resource | undefined;
  isFa: boolean;
  monitorName: string;
  onNameChange: (name: string) => void;
}) {
  const t = (en: string, fa: string) => (isFa ? fa : en);
  const rows: Array<{ label: string; value: string }> = [
    { label: t("Resource name", "نام منبع"), value: resource?.name ?? "—" },
    { label: t("Device type", "نوع دستگاه"), value: resource?.type_name ?? "—" },
    { label: t("IP address / Hostname", "آدرس IP / نام میزبان"), value: resource?.target ?? "—" },
    { label: t("Location", "موقعیت"), value: resource?.metadata?.location ? String(resource.metadata.location) : "—" },
  ];

  return (
    <div className="flex flex-col gap-4">
      <div>
        <h3 className="text-base font-semibold text-foreground">{t("SNMP Monitoring", "مانیتورینگ SNMP")}</h3>
        <p className="mt-0.5 text-xs text-muted-foreground">
          {t(
            "The SNMP collector connects directly to the device's SNMP agent. No agent is installed on your device.",
            "کلکتور SNMP مستقیماً به SNMP Agent دستگاه متصل می‌شود. هیچ Agent روی دستگاه نصب نمی‌شود.",
          )}
        </p>
      </div>

      <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
        {rows.map((row) => (
          <div key={row.label} className="flex flex-col gap-0.5 rounded-lg border border-border/40 px-3 py-2">
            <span className="text-[10px] text-muted-foreground">{row.label}</span>
            <span className="truncate text-sm font-medium text-foreground" dir="ltr">
              {row.value}
            </span>
          </div>
        ))}
      </div>

      <div className="flex flex-col gap-1.5">
        <Label className="text-xs font-medium text-muted-foreground">
          {t("Monitor name", "نام مونیتور")}
        </Label>
        <Input
          value={monitorName}
          onChange={(e) => onNameChange(e.target.value)}
          placeholder="SNMP Monitoring"
          className="h-10 max-w-sm"
          dir="auto"
        />
      </div>
    </div>
  );
}
