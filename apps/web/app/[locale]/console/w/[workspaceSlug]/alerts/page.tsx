"use client";

import { useTranslations } from "next-intl";

import { EmptyState } from "@/design-system/patterns/empty-state";
import { ErrorState } from "@/design-system/patterns/error-state";
import { PageHeader } from "@/design-system/patterns/page-header";
import { Badge } from "@/shared/ui/badge";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/shared/ui/card";
import { Skeleton } from "@/shared/ui/skeleton";
import { useAlerts } from "@/entities/alert/hooks/use-alert";
import { Warning } from "@/shared/ui/icons";
import { cn } from "@/shared/utils/cn";
import type { Alert } from "@/entities/alert/model/types";

const severityStyles: Record<Alert["severity"], string> = {
  info: "bg-muted text-muted-foreground",
  warning: "bg-warning/10 text-warning",
  critical: "bg-destructive/10 text-destructive",
};

const stateStyles: Record<Alert["state"], string> = {
  pending: "bg-muted text-muted-foreground",
  firing: "bg-destructive/10 text-destructive",
  recovering: "bg-warning/10 text-warning",
  resolved: "bg-success/10 text-success",
  suppressed: "bg-muted text-muted-foreground",
};

function AlertRow({ alert }: { alert: Alert }) {
  return (
    <div className="grid grid-cols-5 items-center gap-4 border-b border-border/60 px-6 py-3 text-sm last:border-b-0">
      <div className="flex items-center gap-2">
        <Badge
          variant="outline"
          className={cn("text-xs", severityStyles[alert.severity])}
        >
          {alert.severity}
        </Badge>
        <Badge
          variant="outline"
          className={cn("text-xs", stateStyles[alert.state])}
        >
          {alert.state}
        </Badge>
      </div>
      <span className="truncate font-medium">{alert.title}</span>
      <span className="truncate text-muted-foreground">{alert.monitor_id}</span>
      <span className="line-clamp-2 text-muted-foreground">
        {alert.description}
      </span>
      <span className="text-right text-muted-foreground">
        {alert.opened_at
          ? new Date(alert.opened_at).toLocaleString()
          : new Date(alert.created_at).toLocaleString()}
      </span>
    </div>
  );
}

function AlertsSkeleton() {
  return (
    <Card>
      <CardHeader>
        <Skeleton className="h-5 w-24" />
      </CardHeader>
      <CardContent className="space-y-3 p-0">
        <div className="border-b border-border/60 px-6 py-3">
          <div className="grid grid-cols-5 gap-4 text-xs font-medium text-muted-foreground">
            <Skeleton className="h-4 w-16" />
            <Skeleton className="h-4 w-20" />
            <Skeleton className="h-4 w-16" />
            <Skeleton className="h-4 w-24" />
            <Skeleton className="h-4 w-12" />
          </div>
        </div>
        {Array.from({ length: 5 }).map((_, i) => (
          <div key={i} className="grid grid-cols-5 gap-4 px-6 py-3">
            <Skeleton className="h-4 w-12" />
            <Skeleton className="h-4 w-24" />
            <Skeleton className="h-4 w-16" />
            <Skeleton className="h-4 w-32" />
            <Skeleton className="h-4 w-20" />
          </div>
        ))}
      </CardContent>
    </Card>
  );
}

export default function AlertsPage() {
  const t = useTranslations("alerts");
  const { data, isLoading, isError, refetch } = useAlerts();

  return (
    <div className="space-y-6">
      <PageHeader
        title={t("title")}
        subtitle={t("subtitle")}
      />

      {isLoading ? (
        <AlertsSkeleton />
      ) : isError ? (
        <ErrorState onRetry={() => refetch()} />
      ) : !data || data.items.length === 0 ? (
        <EmptyState
          icon={Warning}
          title={t("noAlerts")}
          description={t("subtitle")}
        />
      ) : (
        <Card>
          <CardHeader>
            <CardTitle>{t("title")}</CardTitle>
          </CardHeader>
          <CardContent className="p-0">
            <div className="border-b border-border/60 px-6 py-3 text-xs font-medium text-muted-foreground">
              <div className="grid grid-cols-5 gap-4">
                <span>{t("state")}</span>
                <span>{t("title")}</span>
                <span>{t("monitor")}</span>
                <span>{t("description")}</span>
                <span className="text-right">{t("opened")}</span>
              </div>
            </div>
            {data.items.map((alert) => (
              <AlertRow key={alert.id} alert={alert} />
            ))}
          </CardContent>
        </Card>
      )}
    </div>
  );
}
