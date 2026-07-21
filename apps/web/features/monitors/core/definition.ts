import type { ComponentType } from "react";
import type { UseFormReturn } from "react-hook-form";

import type { AppIcon } from "@/lib/icons";
import type { MonitorFormValues } from "@/lib/schemas";
import type { Monitor, MonitorType, ProbeLocation } from "@/types/monitor";
import type { MonitorMetrics, ProbeResult } from "@/types/result";

export type MonitorTypeGroupKey = "web" | "network" | "domain_email";

export interface MonitorSummaryProps {
  monitor: Monitor;
  metrics: MonitorMetrics | undefined;
  latestResult: ProbeResult | null;
  recentResults: ProbeResult[];
  probeLocations: ProbeLocation[];
  locale: string;
  rangeLabel: string;
}

export interface MonitorConfigurationProps {
  monitor: Monitor;
  latestResult: ProbeResult | null;
  recentResults: ProbeResult[];
  probeLocations: ProbeLocation[];
  locale: string;
}

export interface MonitorTypeDefinition {
  type: MonitorType;
  group: MonitorTypeGroupKey;
  icon: AppIcon;
  defaultIntervalSeconds: number;
  minimumIntervalSeconds: number;
  defaultValues: Partial<MonitorFormValues>;
  ConfigFields: ComponentType<{ form: UseFormReturn<MonitorFormValues> }>;
  Summary?: ComponentType<MonitorSummaryProps>;
  Configuration?: ComponentType<MonitorConfigurationProps>;
  apiFieldMap: Readonly<Record<string, keyof MonitorFormValues>>;
}
