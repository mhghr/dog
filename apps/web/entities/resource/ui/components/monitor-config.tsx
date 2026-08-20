"use client";

// Legacy adapter: routes the previous `MonitorConfig` entry point through the
// new schema-driven monitoring settings framework. Both the resource detail
// Settings tab and the observability resource-settings view consume the same
// standard layout without change.
import { MonitoringSettingsForm } from "../settings";
import type { MonitorTypeDef } from "@/entities/resource/model/types";
import type { Monitor } from "@/entities/resource/hooks/types";

interface Props {
  resourceId: string;
  type: MonitorTypeDef;
  monitor: Monitor | undefined;
  isFa: boolean;
}

export function MonitorConfig({ resourceId, type, monitor, isFa }: Props) {
  return (
    <MonitoringSettingsForm
      resourceId={resourceId}
      type={type}
      monitor={monitor}
      target={monitor?.resource_target ?? ""}
      isFa={isFa}
    />
  );
}
