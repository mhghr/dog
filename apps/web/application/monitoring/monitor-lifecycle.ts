import { resourcesApi } from "@/entities/resource/api/resource.api";
import { monitorApi } from "@/entities/monitor/api/monitor.api";

import type { MonitorV2Input, MonitorV2 } from "@/entities/resource/model/monitor-v2";
import type { Monitor } from "@/entities/monitor/model/types";

// enableMonitor activates a resource-scoped monitor (resource API).
export async function enableResourceMonitor(
  resourceId: string,
  monitorId: string,
  input: MonitorV2Input,
): Promise<MonitorV2> {
  return resourcesApi.updateMonitor(resourceId, monitorId, { ...input, enabled: true });
}

// disableMonitor pauses a resource-scoped monitor (resource API).
export async function disableResourceMonitor(
  resourceId: string,
  monitorId: string,
  input: MonitorV2Input,
): Promise<MonitorV2> {
  return resourcesApi.updateMonitor(resourceId, monitorId, { ...input, enabled: false });
}

// enableMonitor activates a standalone monitor (legacy monitors API).
export async function enableMonitor(monitorId: string): Promise<Monitor> {
  return monitorApi.resume(monitorId);
}

// disableMonitor pauses a standalone monitor (legacy monitors API).
export async function disableMonitor(monitorId: string): Promise<Monitor> {
  return monitorApi.pause(monitorId);
}

// configureMonitor updates a monitor's configuration while keeping it in the
// same enabled/disabled state it already had.
export async function configureMonitor(
  monitorId: string,
  input: Partial<Monitor>,
): Promise<Monitor> {
  return monitorApi.update(monitorId, input as Parameters<typeof monitorApi.update>[1]);
}
