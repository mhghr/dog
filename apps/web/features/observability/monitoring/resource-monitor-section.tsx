"use client";

import { useLocale, useTranslations } from "next-intl";

import { ResourceMonitorDashboard } from "@/entities/resource/ui/resource-monitor-dashboard";
import { useResourceMonitors } from "@/entities/resource/hooks/use-resource";

// ResourceMonitorSection renders the live monitoring dashboards for every
// enabled monitor on a resource.
export function ResourceMonitorSection({ resourceId }: { resourceId: string }) {
  const t = useTranslations("resources");
  const locale = useLocale();
  const isFa = locale === "fa";
  const { data, isPending, isError, refetch } = useResourceMonitors(resourceId);

  if (isPending) {
    return (
      <div className="space-y-4">
        <div className="h-40 animate-pulse rounded-xl bg-muted/40" />
        <div className="h-40 animate-pulse rounded-xl bg-muted/40" />
      </div>
    );
  }

  if (isError) {
    return (
      <button
        type="button"
        onClick={() => void refetch()}
        className="rounded-lg border border-destructive/30 px-4 py-2 text-sm text-destructive"
      >
        {t("errorTitle")}
      </button>
    );
  }

  const enabled = data?.items.filter((m) => m.enabled) ?? [];

  if (enabled.length === 0) {
    return (
      <div className="rounded-xl border border-dashed border-border/60 py-16 text-center text-sm text-muted-foreground">
        {isFa ? "مانیتور فعالی وجود ندارد" : "No active monitors"}
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-8">
      {enabled.map((monitor) => (
        <section key={monitor.id}>
          <h2 className="mb-3 text-sm font-semibold">{monitor.name}</h2>
          <ResourceMonitorDashboard
            resourceId={resourceId}
            monitorId={monitor.id}
            metricKeys={["avg_rtt_ms", "packet_loss_percent", "jitter_ms"]}
            isFa={isFa}
          />
        </section>
      ))}
    </div>
  );
}
