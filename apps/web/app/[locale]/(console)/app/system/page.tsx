"use client";

import { Server } from "lucide-react";
import { useLocale, useTranslations } from "next-intl";

import { ErrorState } from "@/components/common/error-state";
import { PageHeader } from "@/components/common/page-header";
import { RelativeTime } from "@/components/common/relative-time";
import { Badge } from "@/components/ui/badge";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { useSystemHealth } from "@/hooks/use-system-health";
import { formatNumber } from "@/lib/formatters";
import { cn } from "@/lib/utils";
import type { ComponentHealth } from "@/types/api";

function HealthBadge({ status }: { status: ComponentHealth["status"] }) {
  const t = useTranslations("system");

  const styles: Record<ComponentHealth["status"], string> = {
    healthy: "bg-success/10 text-success",
    unhealthy: "bg-destructive/10 text-destructive",
    degraded: "bg-warning/10 text-warning",
    unknown: "bg-muted text-muted-foreground",
  };

  return (
    <Badge className={cn("border-transparent font-medium", styles[status])}>
      <span className="inline-block size-2 rounded-full bg-current" aria-hidden />
      {t(status)}
    </Badge>
  );
}

function HealthRow({ component }: { component: ComponentHealth }) {
  const t = useTranslations("system");

  return (
    <li className="flex items-center justify-between gap-3 rounded-md px-3 py-2.5 hover:bg-muted/50">
      <div className="flex items-center gap-2">
        <Server className="size-4 text-muted-foreground" aria-hidden />
        <span dir="ltr" className="font-mono text-sm">
          {component.name}
        </span>
      </div>
      <div className="flex items-center gap-3">
        {component.last_seen ? (
          <span className="hidden text-xs text-muted-foreground sm:inline">
            {t("lastSeen")}: <RelativeTime value={component.last_seen} />
          </span>
        ) : null}
        <HealthBadge status={component.status} />
      </div>
    </li>
  );
}

export default function SystemPage() {
  const t = useTranslations("system");
  const locale = useLocale();

  const healthQuery = useSystemHealth();

  return (
    <div>
      <PageHeader
        title={t("title")}
        subtitle={t("subtitle")}
        actions={
          healthQuery.data ? <HealthBadge status={healthQuery.data.status} /> : null
        }
      />

      {healthQuery.isPending ? (
        <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
            <Skeleton className="h-[300px] rounded-xl" />
            <Skeleton className="h-[300px] rounded-xl" />
        </div>
      ) : healthQuery.isError ? (
        <ErrorState onRetry={() => void healthQuery.refetch()} />
      ) : (
        <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle className="text-base">{t("component")}</CardTitle>
            </CardHeader>
            <CardContent>
              <ul className="flex flex-col divide-y divide-border">
                {healthQuery.data.components.map((component) => (
                  <HealthRow key={component.name} component={component} />
                ))}
              </ul>
            </CardContent>
          </Card>

          <div className="flex flex-col gap-4">
            <Card>
              <CardHeader>
                <CardTitle className="text-base">{t("workers")}</CardTitle>
              </CardHeader>
              <CardContent>
                {healthQuery.data.workers.length === 0 ? (
                  <p className="py-6 text-center text-sm text-muted-foreground">
                    {t("noWorkers")}
                  </p>
                ) : (
                  <ul className="flex flex-col divide-y divide-border">
                    {healthQuery.data.workers.map((worker) => (
                      <HealthRow key={worker.name} component={worker} />
                    ))}
                  </ul>
                )}
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle className="text-base">{t("queue")}</CardTitle>
              </CardHeader>
              <CardContent className="grid grid-cols-2 gap-4">
                <div className="stat-card items-start">
                  <p className="text-sm text-muted-foreground">{t("lag")}</p>
                  <p className="mt-1 text-2xl font-semibold tabular-nums" dir="ltr">
                    {formatNumber(healthQuery.data.queue.lag, locale)}
                  </p>
                </div>
                <div className="stat-card items-start">
                  <p className="text-sm text-muted-foreground">{t("pending")}</p>
                  <p className="mt-1 text-2xl font-semibold tabular-nums" dir="ltr">
                    {formatNumber(healthQuery.data.queue.pending, locale)}
                  </p>
                </div>
              </CardContent>
            </Card>
          </div>
        </div>
      )}
    </div>
  );
}
