"use client";

import { useTranslations } from "next-intl";

import { MONITOR_TYPE_ICONS } from "@/lib/monitor-meta";
import { cn } from "@/lib/utils";
import type { MonitorType } from "@/types/monitor";

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
