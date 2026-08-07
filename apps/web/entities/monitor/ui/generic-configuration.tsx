"use client";

import { useTranslations } from "next-intl";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import type { MonitorConfigurationProps } from "@/features/monitors/core/definition";

export function GenericMonitorConfiguration({ monitor }: MonitorConfigurationProps) {
  const t = useTranslations("monitorDetail");
  const tFields = useTranslations("monitors.fields");

  return (
    <Card>
      <CardHeader><CardTitle className="text-base">{t("configuration")}</CardTitle></CardHeader>
      <CardContent className="grid grid-cols-2 gap-4 sm:grid-cols-3">
        <div className="flex flex-col gap-1"><span className="text-xs text-muted-foreground">{tFields("intervalSeconds")}</span><span className="text-sm font-medium tabular-nums" dir="ltr">{monitor.interval_seconds}s</span></div>
        <div className="flex flex-col gap-1"><span className="text-xs text-muted-foreground">{tFields("timeoutMillis")}</span><span className="text-sm font-medium tabular-nums" dir="ltr">{monitor.timeout_millis}ms</span></div>
        <div className="flex flex-col gap-1"><span className="text-xs text-muted-foreground">{tFields("retries")}</span><span className="text-sm font-medium tabular-nums" dir="ltr">{monitor.retries}</span></div>
      </CardContent>
    </Card>
  );
}
