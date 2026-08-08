"use client";

import { useLocale, useTranslations } from "next-intl";

import { MonitorConfig } from "@/entities/resource/ui/components/monitor-config";
import { useMonitorTypes, useResourceMonitors } from "@/entities/resource/hooks/use-resource";
import { Skeleton } from "@/shared/ui/skeleton";

// ResourceSettingsView lets operators enable, configure and disable monitors
// for a resource through the resource-scoped monitor API.
export function ResourceSettingsView({ resourceId }: { resourceId: string }) {
  const t = useTranslations("resources");
  const locale = useLocale();
  const isFa = locale === "fa";
  const monitorsQuery = useResourceMonitors(resourceId);
  const typesQuery = useMonitorTypes();

  if (monitorsQuery.isPending || typesQuery.isPending) {
    return (
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        <Skeleton className="h-72 rounded-xl" />
        <Skeleton className="h-72 rounded-xl lg:col-span-2" />
      </div>
    );
  }

  const monitors = monitorsQuery.data?.items ?? [];
  const types = typesQuery.data?.items ?? [];

  return (
    <div className="flex flex-col gap-6">
      <p className="text-sm text-muted-foreground">
        {t("settingsSubtitle")}
      </p>
      <div className="flex flex-col gap-6">
        {types.map((type) => {
          const monitor = monitors.find((m) => m.monitor_type_id === type.id);
          return (
            <MonitorConfig
              key={type.id}
              resourceId={resourceId}
              type={type}
              monitor={monitor}
              isFa={isFa}
            />
          );
        })}
      </div>
    </div>
  );
}
