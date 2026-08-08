"use client";

import { useTranslations } from "next-intl";

import { Badge } from "@/shared/ui/badge";
import { STATUS_STYLES } from "@/entities/monitor/model/monitor-meta";
import { cn } from "@/shared/utils/cn";
import type { MonitorStatus } from "@/entities/monitor/model/types";

export function MonitorStatusBadge({
  status,
  className,
}: {
  status: MonitorStatus;
  className?: string;
}) {
  const t = useTranslations("status");
  const styles = STATUS_STYLES[status];

  return (
    <Badge className={cn("gap-1.5 font-medium", styles.badge, className)}>
      <span
        className={cn("size-1.5 shrink-0 rounded-full", styles.dot)}
        aria-hidden
      />
      {t(status)}
    </Badge>
  );
}
