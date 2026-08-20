"use client";

import { useQuery } from "@tanstack/react-query";
import { BellOff } from "lucide-react";

import { Card, CardContent, CardHeader, CardTitle } from "@/shared/ui/card";
import { Skeleton } from "@/shared/ui/skeleton";
import { alertApi } from "@/entities/alert/api/alert.api";
import type { AlertPolicy } from "@/entities/alert/model/types";

const SEVERITY_STYLES: Record<AlertPolicy["severity"], string> = {
  info: "bg-muted/60 text-muted-foreground",
  warning: "bg-warning/12 text-warning",
  critical: "bg-destructive/12 text-destructive",
};

function conditionSummary(policy: AlertPolicy, isFa: boolean): string {
  const c = policy.conditions;
  const t = (en: string, fa: string) => (isFa ? fa : en);
  const parts: string[] = [];
  if (c.consecutive_failures != null && c.consecutive_failures > 0) {
    parts.push(t(`${c.consecutive_failures} consecutive failures`, `${c.consecutive_failures} شکست متوالی`));
  }
  if (c.high_latency_ms != null && c.high_latency_ms > 0) {
    parts.push(t(`latency > ${c.high_latency_ms}ms`, `تأخیر بیشتر از ${c.high_latency_ms} میلی‌ثانیه`));
  }
  if (c.packet_loss_percent != null && c.packet_loss_percent > 0) {
    parts.push(t(`packet loss > ${c.packet_loss_percent}%`, `افت بسته بیشتر از ${c.packet_loss_percent}٪`));
  }
  if (c.ssl_expiring_days != null) {
    parts.push(t(`certificate expiring within ${c.ssl_expiring_days} days`, `انقضای گواهی تا ${c.ssl_expiring_days} روز`));
  }
  if (c.dns_mismatch) parts.push(t("DNS answer mismatch", "عدم تطابق پاسخ DNS"));
  if (c.smtp_starttls_fail) parts.push(t("SMTP STARTTLS failure", "خطای STARTTLS"));
  if (c.ntp_offset_ms != null) parts.push(t(`NTP offset > ${c.ntp_offset_ms}ms`, `انحراف NTP بیشتر از ${c.ntp_offset_ms} میلی‌ثانیه`));
  return parts.length > 0 ? parts.join(" · ") : t("Monitor failure", "شکست مانیتور");
}

interface HttpAlertRulesProps {
  isFa: boolean;
}

// Active alert rules for this monitor. Enables operators to see configured
// conditions at a glance; an empty/errored state renders gracefully.
export function HttpAlertRules({ isFa }: HttpAlertRulesProps) {
  const t = (en: string, fa: string) => (isFa ? fa : en);
  const policiesQuery = useQuery({
    queryKey: ["alerting", "policies"],
    queryFn: () => alertApi.listPolicies(),
    staleTime: 60_000,
  });

  const policies = (policiesQuery.data?.items ?? []).filter((p) => p.enabled);

  return (
    <Card variant="bordered" className="shadow-subtle">
      <CardHeader className="px-5 pt-4">
        <CardTitle className="text-sm font-semibold text-foreground">
          {t("Alert Rules", "قوانین هشدار")}
        </CardTitle>
      </CardHeader>
      <CardContent className="px-4 pb-4">
        {policiesQuery.isPending ? (
          <Skeleton className="h-20 w-full rounded-lg" />
        ) : policies.length === 0 ? (
          <div className="flex items-center justify-center gap-2 rounded-lg border border-dashed border-border py-8 text-sm text-muted-foreground">
            <BellOff className="size-4" />
            {t("No active alert rules", "قانون هشدار فعالی وجود ندارد")}
          </div>
        ) : (
          <div className="flex flex-col gap-2">
            {policies.map((policy) => (
              <div
                key={policy.id}
                className="flex flex-wrap items-center gap-x-3 gap-y-1.5 rounded-lg border border-border/50 px-3.5 py-2.5"
              >
                <span
                  className={`rounded-full px-2 py-0.5 text-[11px] font-medium ${SEVERITY_STYLES[policy.severity]}`}
                >
                  {policy.severity}
                </span>
                <span className="text-sm font-medium">{policy.name}</span>
                <span className="min-w-0 flex-1 text-xs text-muted-foreground">
                  {conditionSummary(policy, isFa)}
                </span>
                {policy.scope?.monitor_ids?.length ? (
                  <span className="text-[11px] text-muted-foreground">
                    {t("Scoped", "محدود به")}: {policy.scope.monitor_ids.length}
                  </span>
                ) : (
                  <span className="text-[11px] text-muted-foreground">{t("All probes", "همه پراب‌ها")}</span>
                )}
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
