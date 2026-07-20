"use client";

import { useTranslations } from "next-intl";

import { Badge } from "@/components/ui/badge";
import { STATUS_STYLES } from "@/lib/monitor-meta";
import { cn } from "@/lib/utils";
import type { MonitorStatus } from "@/types/monitor";

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
