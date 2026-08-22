// SNMP metric normalization. Turns raw ProbeResult payloads into stable,
// UI-friendly structures — device summary, interface snapshots, sensors.

import type { ProbeResult } from "@/entities/monitor/model/result";
import type { ProbeSeries } from "@/entities/resource/api/resource.api";
import type { SparklineSeries } from "@/shared/ui/charts/sparkline";

export interface SnmpDeviceSummary {
  cpuPercent: number | null;
  memoryPercent: number | null;
  uptimeSeconds: number | null;
  temperatureCelsius: number | null;
  availability: number | null;
  reachable: boolean;
  deviceHealth: number | null;
  interfacesDown: number;
  interfacesTotal: number;
  maxUtilization: number | null;
  state: string | null;
}

export interface SnmpInterfaceSnapshot {
  if_index: number;
  if_name?: string;
  if_descr?: string;
  if_alias?: string;
  if_speed_bps?: number;
  if_admin_status?: number;
  if_oper_status?: number;
  if_in_octets?: number;
  if_out_octets?: number;
  if_in_errors?: number;
  if_out_errors?: number;
  if_in_discards?: number;
  if_out_discards?: number;
  in_bps?: number;
  out_bps?: number;
  utilization_percent?: number;
}

export interface SnmpSensorInfo {
  name: string;
  sensor_type: string;
  value: number;
  unit: string;
  status: string;
}

export interface SnmpDeviceInfo {
  sys_name?: string;
  sys_descr?: string;
  sys_object_id?: string;
  sys_uptime?: string;
  vendor?: string;
  model?: string;
  location?: string;
}

function num(value: unknown): number | null {
  if (typeof value === "number" && !Number.isNaN(value)) return value;
  if (typeof value === "string" && value.trim() !== "") {
    const n = Number(value);
    return Number.isNaN(n) ? null : n;
  }
  return null;
}

function parseJSON<T>(value: unknown): T | null {
  if (typeof value !== "string" || value === "") return null;
  try {
    return JSON.parse(value) as T;
  } catch {
    return null;
  }
}

export function summarizeSnmp(latest: ProbeResult[]): SnmpDeviceSummary {
  const result = latest[0];
  const metrics = result?.metrics ?? {};

  const totalChecks = latest.length;
  const successChecks = latest.filter((r) => r.success).length;
  const reachable = num(metrics["snmp.reachability"]) ?? 0;
  const state =
    typeof result?.attributes?.["snmp.state"] === "string"
      ? (result.attributes["snmp.state"] as string)
      : null;

  const interfaces = toSnmpInterfaces(latest);
  const interfacesDown = interfaces.filter((i) => i.if_oper_status === 2).length;

  return {
    cpuPercent: num(metrics["device.cpu_percent"]),
    memoryPercent: num(metrics["device.memory_percent"]),
    uptimeSeconds: num(metrics["device.uptime_seconds"]),
    temperatureCelsius: num(metrics["device.temperature_celsius"]),
    availability: totalChecks > 0 ? (successChecks / totalChecks) * 100 : null,
    reachable: reachable === 1,
    deviceHealth: num(metrics["device.health"]),
    interfacesDown,
    interfacesTotal: interfaces.length,
    maxUtilization: num(metrics["snmp.interface_utilization_percent"]),
    state,
  };
}

export function toSnmpInterfaces(latest: ProbeResult[]): SnmpInterfaceSnapshot[] {
  const raw = latest[0]?.attributes?.["snmp.interfaces"];
  return parseJSON<SnmpInterfaceSnapshot[]>(raw) ?? [];
}

export function toSnmpSensors(latest: ProbeResult[]): SnmpSensorInfo[] {
  const raw = latest[0]?.attributes?.["snmp.sensors"];
  return parseJSON<SnmpSensorInfo[]>(raw) ?? [];
}

export function toSnmpDevice(latest: ProbeResult[]): SnmpDeviceInfo {
  const raw = latest[0]?.attributes?.["snmp.device"];
  return parseJSON<SnmpDeviceInfo>(raw) ?? {};
}

export function toSnmpEvents(latest: ProbeResult[]): string[] {
  const raw = latest[0]?.attributes?.["snmp.partial_failures"];
  if (Array.isArray(raw)) return raw.map(String);
  return [];
}

export function sparkOf(series: ProbeSeries[] | undefined, transform?: (v: number) => number): SparklineSeries[] {
  return (series ?? [])
    .map((s) => ({
      name: s.probe_name || s.location,
      points: transform
        ? s.points.map((p) => ({ time: p.timestamp, value: transform(p.value) }))
        : s.points.map((p) => ({ time: p.timestamp, value: p.value })),
    }))
    .filter((s) => s.points.length > 0);
}

export function operStatusLabel(status: number | undefined): string {
  switch (status) {
    case 1: return "up";
    case 2: return "down";
    case 3: return "testing";
    case 4: return "unknown";
    case 5: return "dormant";
    case 6: return "notPresent";
    case 7: return "lowerLayerDown";
    default: return "unknown";
  }
}

export function utilHealth(utilization: number | null, warning: number, critical: number): "healthy" | "warning" | "critical" | "unknown" {
  if (utilization == null) return "unknown";
  if (utilization >= critical) return "critical";
  if (utilization >= warning) return "warning";
  return "healthy";
}

export function cpuHealth(cpu: number | null, warning: number, critical: number): "healthy" | "warning" | "critical" | "unknown" {
  if (cpu == null) return "unknown";
  if (cpu >= critical) return "critical";
  if (cpu >= warning) return "warning";
  return "healthy";
}
