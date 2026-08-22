"use client";

import { formatDistanceToNow } from "date-fns";
import { enUS, faIR } from "date-fns/locale";
import { AlertOctagon, CheckCircle2 } from "lucide-react";
import { useLocale, useTranslations } from "next-intl";

import { Card, CardContent, CardHeader, CardTitle } from "@/shared/ui/card";
import { cn } from "@/shared/utils/cn";
import type {
  AttentionRequired,
  DashboardSummary,
} from "@/entities/dashboard/model/types";

function PanelSkeleton() {
  return (
    <Card variant="bordered" className="animate-pulse shadow-subtle">
      <CardHeader>
        <div className="h-3 w-28 rounded-full bg-muted" />
      </CardHeader>
      <CardContent className="space-y-3">
        {[0, 1, 2, 3].map((i) => (
          <div key={i} className="h-4 w-full rounded-full bg-muted/50" />
        ))}
      </CardContent>
    </Card>
  );
}

function RecentFailures({ failures }: { failures: DashboardSummary["recent_failures"] }) {
  const t = useTranslations("dashboard");
  const locale = useLocale();

  if (failures.length === 0) {
    return (
      <div className="flex flex-col items-center gap-2 py-8 text-center">
        <CheckCircle2 className="size-8 text-success/70" aria-hidden />
        <p className="text-sm text-muted-foreground">{t("noFailures")}</p>
      </div>
    );
  }

  return (
    <div className="divide-y divide-border/40">
      {failures.map((f, i) => (
        <div key={`${f.monitor_id}-${f.started_at}-${i}`} className="flex items-center gap-3 py-2.5">
          <span className="relative flex size-2 shrink-0">
            <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-destructive opacity-60" />
            <span className="relative inline-flex size-2 rounded-full bg-destructive" />
          </span>
          <div className="min-w-0 flex-1">
            <p className="truncate text-sm font-medium">{f.monitor_name}</p>
            <p className="truncate text-xs text-muted-foreground" dir="ltr">
              {f.error_code ?? f.monitor_type}
            </p>
          </div>
          <span className="shrink-0 text-xs text-muted-foreground">
            {formatDistanceToNow(new Date(f.started_at), {
              addSuffix: true,
              locale: locale === "fa" ? faIR : enUS,
            })}
          </span>
        </div>
      ))}
    </div>
  );
}

function SlowestMonitors({ monitors }: { monitors: DashboardSummary["slowest_monitors"] }) {
  const t = useTranslations("dashboard");

  if (monitors.length === 0) {
    return (
      <div className="py-8 text-center text-sm text-muted-foreground">{t("noSlowMonitors")}</div>
    );
  }

  const max = Math.max(...monitors.map((m) => m.duration_millis), 1);

  return (
    <div className="space-y-3.5">
      {monitors.map((m) => (
        <div key={m.monitor_id} className="space-y-1.5">
          <div className="flex items-baseline justify-between gap-3 text-sm">
            <span className="truncate font-medium">{m.monitor_name}</span>
            <span className="shrink-0 tabular-nums text-muted-foreground" dir="ltr">
              {m.duration_millis}ms
            </span>
          </div>
          <div className="h-1.5 w-full overflow-hidden rounded-full bg-muted">
            <div
              className="h-full rounded-full bg-gradient-to-r from-primary/70 to-primary transition-[width] duration-700 ease-[cubic-bezier(0.23,1,0.32,1)]"
              style={{ width: `${Math.max((m.duration_millis / max) * 100, 6)}%` }}
            />
          </div>
        </div>
      ))}
    </div>
  );
}

function AttentionList({ attention }: { attention: AttentionRequired }) {
  const t = useTranslations("dashboard");

  const items = [
    { label: t("certificatesExpiring"), value: attention.certificates_expiring_30d },
    { label: t("domainsExpiring"), value: attention.domains_expiring_45d },
    { label: t("smtpFailures"), value: attention.smtp_starttls_failures },
    { label: t("ntpHighOffset"), value: attention.ntp_high_offset },
  ];

  const any = items.some((i) => i.value > 0);

  if (!any) {
    return (
      <div className="flex flex-col items-center gap-2 py-8 text-center">
        <CheckCircle2 className="size-8 text-success/70" aria-hidden />
        <p className="text-sm text-muted-foreground">{t("allGood")}</p>
      </div>
    );
  }

  return (
    <div className="grid grid-cols-2 gap-2.5">
      {items.map((item) => {
        const active = item.value > 0;
        return (
          <div
            key={item.label}
            className={cn(
              "rounded-xl border p-3 transition-colors duration-[250ms]",
              active
                ? "border-destructive/30 bg-destructive/5"
                : "border-border/40 bg-muted/30",
            )}
          >
            <p className="truncate text-[11px] leading-4 text-muted-foreground">{item.label}</p>
            <p
              className={cn(
                "mt-1 text-xl font-semibold tabular-nums",
                active ? "text-destructive" : "text-muted-foreground",
              )}
              dir="ltr"
            >
              {item.value}
            </p>
          </div>
        );
      })}
    </div>
  );
}

export function DashboardFeeds({ summary }: { summary?: DashboardSummary }) {
  const t = useTranslations("dashboard");

  if (!summary) {
    return (
      <div className="grid grid-cols-1 gap-3 lg:grid-cols-3">
        <PanelSkeleton />
        <PanelSkeleton />
        <PanelSkeleton />
      </div>
    );
  }

  return (
    <div className="grid grid-cols-1 gap-3 lg:grid-cols-3">
      <Card
        variant="bordered"
        className="animate-in fade-in slide-in-from-bottom-2 shadow-subtle transition-all duration-[250ms] ease-[cubic-bezier(0.23,1,0.32,1)] hover:shadow-md"
        style={{ animationDelay: "720ms", animationFillMode: "backwards" }}
      >
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <AlertOctagon className="size-4 text-destructive" aria-hidden />
            {t("recentFailures")}
          </CardTitle>
        </CardHeader>
        <CardContent>
          <RecentFailures failures={summary.recent_failures} />
        </CardContent>
      </Card>

      <Card
        variant="bordered"
        className="animate-in fade-in slide-in-from-bottom-2 shadow-subtle transition-all duration-[250ms] ease-[cubic-bezier(0.23,1,0.32,1)] hover:shadow-md"
        style={{ animationDelay: "810ms", animationFillMode: "backwards" }}
      >
        <CardHeader>
          <CardTitle>{t("slowestMonitors")}</CardTitle>
        </CardHeader>
        <CardContent>
          <SlowestMonitors monitors={summary.slowest_monitors} />
        </CardContent>
      </Card>

      <Card
        variant="bordered"
        className="animate-in fade-in slide-in-from-bottom-2 shadow-subtle transition-all duration-[250ms] ease-[cubic-bezier(0.23,1,0.32,1)] hover:shadow-md"
        style={{ animationDelay: "900ms", animationFillMode: "backwards" }}
      >
        <CardHeader>
          <CardTitle>{t("attentionTitle")}</CardTitle>
        </CardHeader>
        <CardContent>
          <AttentionList attention={summary.attention_required} />
        </CardContent>
      </Card>
    </div>
  );
}
