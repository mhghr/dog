"use client";

import { Switch } from "@/shared/ui/switch";
import { MonitorTypeIcon } from "../../components/monitor-type-icon";
import type { MonitoringTypeSchema } from "../monitoring-schema";

interface MonitoringTypeHeaderProps {
  schema: MonitoringTypeSchema;
  isFa: boolean;
  enabled: boolean;
  onToggle: (enabled: boolean) => void;
}

// Standard monitoring header: type icon + name and the enable/disable toggle.
export function MonitoringTypeHeader({
  schema,
  isFa,
  enabled,
  onToggle,
}: MonitoringTypeHeaderProps) {
  return (
    <div className="flex items-center gap-3 border-b border-border/50 bg-muted/20 px-7 py-4">
      <span className="grid size-10 shrink-0 place-items-center rounded-xl bg-primary/10 text-primary ring-1 ring-primary/15 shadow-[0_0_18px_-6px_var(--primary)]">
        <MonitorTypeIcon type={schema.type} className="size-5" />
      </span>
      <h2 className="min-w-0 flex-1 truncate text-[15px] font-semibold tracking-tight text-foreground">
        {isFa ? schema.title.fa : schema.title.en}
      </h2>
      <Switch checked={enabled} onCheckedChange={onToggle} aria-label="Enable monitoring" />
    </div>
  );
}
