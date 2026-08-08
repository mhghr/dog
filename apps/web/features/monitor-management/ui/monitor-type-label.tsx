"use client";

import { useTranslations } from "next-intl";

import { MONITOR_TYPE_ICONS } from "@/entities/monitor/model/monitor-meta";
import { cn } from "@/shared/utils/cn";
import type { MonitorType } from "@/entities/monitor/model/types";

export function MonitorTypeLabel({
  type,
  className,
  iconOnly = false,
}: {
  type: MonitorType;
  className?: string;
  iconOnly?: boolean;
}) {
  const t = useTranslations("types");
  const Icon = MONITOR_TYPE_ICONS[type];

  return (
    <span className={cn("inline-flex items-center gap-1.5 text-sm", className)}>
      <Icon className="size-4 shrink-0 text-muted-foreground" aria-hidden />
      {iconOnly ? <span className="sr-only">{t(type)}</span> : t(type)}
    </span>
  );
}
