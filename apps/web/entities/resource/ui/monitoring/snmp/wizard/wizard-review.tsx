"use client";

import { Badge } from "@/shared/ui/badge";
import type { Resource } from "@/entities/resource/model/types";
import type { SnmpDiscovery } from "@/entities/resource/api/resource.api";
import type { SnmpExecutionState } from "./wizard-polling";
import type { HealthRulesState } from "../../../settings/components/HealthRulesBuilder";

function Row({ label, value, ltr }: { label: string; value: string; ltr?: boolean }) {
  return (
    <div className="flex items-center justify-between gap-3 rounded-lg border border-border/40 px-3 py-2">
      <span className="text-xs text-muted-foreground">{label}</span>
      <span className="text-xs font-semibold tabular-nums text-foreground" dir={ltr ? "ltr" : "auto"}>
        {value}
      </span>
    </div>
  );
}

export function SnmpReviewStep({
  resource,
  isFa,
  monitorName,
  config,
  execution,
  discovery,
  selectedIds,
  healthRules,
  credentialConfigured,
}: {
  resource: Resource | undefined;
  isFa: boolean;
  monitorName: string;
  config: Record<string, unknown>;
  execution: SnmpExecutionState;
  discovery: SnmpDiscovery | null;
  selectedIds: Set<number>;
  healthRules: HealthRulesState;
  credentialConfigured: boolean;
}) {
  const t = (en: string, fa: string) => (isFa ? fa : en);
  const version = String(config.version ?? "3");

  return (
    <div className="flex flex-col gap-4">
      <h3 className="text-sm font-semibold text-foreground">{t("Review & Create", "بررسی و ایجاد")}</h3>

      <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
        <Row label={t("Resource", "منبع")} value={resource?.name ?? "—"} />
        <Row label={t("IP / Hostname", "IP / نام میزبان")} value={resource?.target ?? "—"} ltr />
        <Row label={t("Monitor name", "نام مونیتور")} value={monitorName} />
        <Row label={t("SNMP version", "نسخه SNMP")} value={version === "3" ? "SNMPv3" : "SNMPv2c"} ltr />
        <Row label={t("Port", "پورت")} value={String(config.port ?? 161)} ltr />
        <Row label={t("Polling interval", "بازه Polling")} value={`${execution.intervalSeconds}s`} ltr />
        <Row label={t("Discovery interval", "بازه کشف")} value={`${Math.round(execution.discoveryIntervalSeconds / 3600)}h`} ltr />
        <Row label={t("Credentials", "اعتبارنامه")} value={credentialConfigured ? t("Configured", "پیکربندی شده") : t("Pending", "در انتظار")} />
      </div>

      <div className="grid grid-cols-1 gap-2 sm:grid-cols-3">
        <Row label={t("System metrics", "متریک‌های سیستم")} value={String(7)} ltr />
        <Row label={t("Monitored interfaces", "اینترفیس‌های مانیتورشده")} value={String(selectedIds.size)} ltr />
        <Row label={t("Hardware sensors", "سنسورهای سخت‌افزار")} value={String(discovery?.sensors.length ?? 0)} ltr />
      </div>

      <div>
        <span className="text-xs font-semibold text-muted-foreground">{t("Health rules", "قوانین سلامت")}</span>
        <div className="mt-1.5 flex flex-wrap gap-1.5">
          {Object.entries(healthRules).map(([key, rule]) => (
            <Badge key={key} variant="outline" className="gap-1 px-2 py-0.5 text-[10px]">
              <span dir="ltr">{key}</span>
              <span className="text-muted-foreground">
                {rule.warning}/{rule.critical}
              </span>
            </Badge>
          ))}
        </div>
      </div>

      <div className="rounded-lg border border-primary/20 bg-primary/5 px-3 py-2 text-[11px] text-muted-foreground">
        {t(
          "Creating the monitor enables scheduled polling. The SNMP collector connects directly to the device's SNMP agent over UDP/161 — no agent is installed on your device.",
          "با ایجاد مونیتور، Polling زمان‌بندی‌شده فعال می‌شود. کلکتور SNMP مستقیماً از طریق UDP/161 به SNMP Agent دستگاه متصل می‌شود — هیچ Agent روی دستگاه نصب نمی‌شود.",
        )}
      </div>
    </div>
  );
}
