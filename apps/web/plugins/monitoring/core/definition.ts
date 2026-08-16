import type { ComponentType } from "react";
import type { UseFormReturn } from "react-hook-form";

import type { AppIcon } from "@/shared/ui/icons";
import type { MonitorFormValues } from "@/entities/monitor/model/form-values";
import type { Monitor, MonitorType } from "@/entities/monitor/model/types";
import type { ProbeLocation } from "@/entities/probe/model/types";
import type { MonitorMetrics, ProbeResult } from "@/entities/monitor/model/result";

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
