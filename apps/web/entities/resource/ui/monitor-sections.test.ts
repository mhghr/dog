import { describe, expect, it } from "vitest";

import { partitionMonitorsByType } from "./monitor-sections";
import type { Monitor } from "@/entities/resource/hooks/types";
import type { MonitorTypeDef } from "@/entities/resource/model/types";

const monitor = (id: string, monitorTypeId: string): Monitor =>
  ({ id, monitor_type_id: monitorTypeId }) as unknown as Monitor;

const typeDef = (id: string, executorKey: string): MonitorTypeDef =>
  ({ id, name: "", slug: "", executor_key: executorKey }) as unknown as MonitorTypeDef;

const types: MonitorTypeDef[] = [
  typeDef("t-ping", "ping"),
  typeDef("t-http", "http"),
  typeDef("t-snmp", "snmp"),
];

describe("partitionMonitorsByType", () => {
  it("groups monitors into per-type buckets", () => {
    const monitors = [
      monitor("m1", "t-ping"),
      monitor("m2", "t-http"),
      monitor("m3", "t-ping"),
      monitor("m4", "t-snmp"),
    ];

    const sections = partitionMonitorsByType(monitors, types);
    const ping = sections.find((s) => s.type === "ping");
    const http = sections.find((s) => s.type === "http");
    const snmp = sections.find((s) => s.type === "snmp");

    expect(ping?.monitors.map((m) => m.id)).toEqual(["m1", "m3"]);
    expect(http?.monitors.map((m) => m.id)).toEqual(["m2"]);
    expect(snmp?.monitors.map((m) => m.id)).toEqual(["m4"]);
  });

  it("returns empty buckets for unknown monitor types", () => {
    const sections = partitionMonitorsByType([monitor("m1", "nope")], types);
    expect(sections.every((s) => s.monitors.length === 0)).toBe(true);
  });

  it("handles an empty monitor list", () => {
    const sections = partitionMonitorsByType([], types);
    expect(sections).toHaveLength(6);
    expect(sections.map((s) => s.type)).toEqual([
      "ping", "http", "tcp", "dns", "tls", "snmp",
    ]);
  });
});
