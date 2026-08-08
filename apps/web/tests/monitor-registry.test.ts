import { describe, expect, it } from "vitest";

import {
  getMonitorDefinition,
  getMonitorFormField,
  MONITOR_REGISTRY,
  MONITOR_TYPE_GROUPS,
  MONITOR_TYPES,
} from "@/plugins/monitoring/core/registry";
import { MONITOR_TYPE_VALUES } from "@/entities/monitor/model/types";

describe("monitor registry", () => {
  it("registers every API monitor type exactly once", () => {
    expect(new Set(MONITOR_TYPES).size).toBe(MONITOR_TYPE_VALUES.length);
    expect([...MONITOR_TYPES].sort()).toEqual([...MONITOR_TYPE_VALUES].sort());
    expect(Object.keys(MONITOR_REGISTRY).sort()).toEqual([...MONITOR_TYPE_VALUES].sort());
  });

  it("assigns every monitor type to exactly one group", () => {
    const groupedTypes = MONITOR_TYPE_GROUPS.flatMap((group) => group.types);
    expect(new Set(groupedTypes).size).toBe(MONITOR_TYPE_VALUES.length);
    expect([...groupedTypes].sort()).toEqual([...MONITOR_TYPE_VALUES].sort());
  });

  it("provides required UI and scheduling metadata", () => {
    for (const type of MONITOR_TYPES) {
      const definition = getMonitorDefinition(type);
      expect(definition.type).toBe(type);
      expect(definition.icon).toBeTypeOf("function");
      expect(definition.ConfigFields).toBeTypeOf("function");
      expect(definition.defaultIntervalSeconds).toBeGreaterThanOrEqual(
        definition.minimumIntervalSeconds,
      );
    }
  });

  it("maps common and type-specific API validation fields", () => {
    expect(getMonitorFormField("http", "name")).toBe("name");
    expect(getMonitorFormField("http", "config.method")).toBe("http_method");
    expect(getMonitorFormField("tcp", "config.port")).toBe("tcp_port");
    expect(getMonitorFormField("ping", "config.unknown")).toBeNull();
  });
});
