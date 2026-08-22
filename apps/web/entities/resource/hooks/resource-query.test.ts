import { describe, expect, it } from "vitest";

import {
  buildMetricsQueryString,
  isDnsMonitor,
  isHttpMonitor,
  isPingMonitor,
  isSnmpMonitor,
  isTcpMonitor,
  isTlsMonitor,
  RANGE_MILLIS,
  resourceListQueryString,
  resourceMonitorMetricsQueryKey,
  type MetricsRange,
} from "./resource-query";
import type { Monitor } from "./types";
import type { MonitorTypeDef } from "@/entities/resource/model/types";

const monitor = (monitorTypeId: string): Monitor =>
  ({ id: "m1", monitor_type_id: monitorTypeId } as unknown as Monitor);

const typeDef = (overrides: Partial<MonitorTypeDef>): MonitorTypeDef =>
  ({ id: "t1", name: "", slug: "", executor_key: "" as string, ...overrides }) as MonitorTypeDef;

describe("resourceListQueryString", () => {
  it("includes defaults for page and page size", () => {
    const qs = resourceListQueryString({});
    expect(qs).toContain("page=1");
    expect(qs).toContain("page_size=20");
  });

  it("omits empty optional filters", () => {
    const qs = resourceListQueryString({ search: "", status: undefined });
    expect(qs).not.toContain("search");
    expect(qs).not.toContain("status");
  });

  it("includes provided filters", () => {
    const qs = resourceListQueryString({ search: "web", status: "up", resourceTypeId: "rt1", page: 3, pageSize: 50 });
    expect(qs).toContain("search=web");
    expect(qs).toContain("status=up");
    expect(qs).toContain("resource_type_id=rt1");
    expect(qs).toContain("page=3");
    expect(qs).toContain("page_size=50");
  });
});

describe("resourceMonitorMetricsQueryKey", () => {
  it("builds a stable cache key", () => {
    expect(resourceMonitorMetricsQueryKey("r1", "m1", "1h")).toEqual([
      "resources", "r1", "monitors", "m1", "metrics", "1h", "",
    ]);
    expect(resourceMonitorMetricsQueryKey("r1", "m1", "1h", "status")).toEqual([
      "resources", "r1", "monitors", "m1", "metrics", "1h", "status",
    ]);
  });
});

describe("buildMetricsQueryString", () => {
  it("spans the configured range in milliseconds", () => {
    const qs = buildMetricsQueryString("1h");
    const params = new URLSearchParams(qs);
    const from = new Date(params.get("from")!);
    const to = new Date(params.get("to")!);
    expect(to.getTime() - from.getTime()).toBe(RANGE_MILLIS["1h"]);
    expect(params.get("step")).toBe("auto");
  });

  it("appends the metric key when provided", () => {
    const qs = buildMetricsQueryString("15m", "status");
    expect(new URLSearchParams(qs).get("metric")).toBe("status");
  });
});

describe("monitor type classifiers", () => {
  const types: MonitorTypeDef[] = [
    typeDef({ id: "t-ping", executor_key: "ping", slug: "ping", name: "Ping" }),
    typeDef({ id: "t-http", executor_key: "http", slug: "http", name: "HTTP Check" }),
    typeDef({ id: "t-tcp", executor_key: "tcp", slug: "tcp", name: "TCP Port" }),
    typeDef({ id: "t-dns", executor_key: "dns", slug: "dns", name: "DNS Resolution" }),
    typeDef({ id: "t-ssl", executor_key: "tls", slug: "ssl", name: "SSL Certificate" }),
    typeDef({ id: "t-snmp", executor_key: "snmp", slug: "snmp", name: "SNMP Device" }),
  ];

  it("classifies ping", () => {
    expect(isPingMonitor(monitor("t-ping"), types)).toBe(true);
    expect(isPingMonitor(monitor("t-http"), types)).toBe(false);
  });

  it("classifies http", () => {
    expect(isHttpMonitor(monitor("t-http"), types)).toBe(true);
    expect(isHttpMonitor(monitor("t-ping"), types)).toBe(false);
  });

  it("classifies tcp", () => {
    expect(isTcpMonitor(monitor("t-tcp"), types)).toBe(true);
    expect(isTcpMonitor(monitor("t-http"), types)).toBe(false);
  });

  it("classifies dns", () => {
    expect(isDnsMonitor(monitor("t-dns"), types)).toBe(true);
    expect(isDnsMonitor(monitor("t-tcp"), types)).toBe(false);
  });

  it("classifies tls/ssl", () => {
    expect(isTlsMonitor(monitor("t-ssl"), types)).toBe(true);
    expect(isTlsMonitor(monitor("t-dns"), types)).toBe(false);
  });

  it("classifies snmp", () => {
    expect(isSnmpMonitor(monitor("t-snmp"), types)).toBe(true);
    expect(isSnmpMonitor(monitor("t-http"), types)).toBe(false);
  });

  it("matches by name when id differs", () => {
    const byName = [
      typeDef({ id: "x1", executor_key: "", slug: "", name: "Ping" }),
      typeDef({ id: "x2", executor_key: "", slug: "", name: "HTTP Check" }),
      typeDef({ id: "x3", executor_key: "", slug: "", name: "TCP Port" }),
      typeDef({ id: "x4", executor_key: "", slug: "", name: "DNS Resolution" }),
      typeDef({ id: "x5", executor_key: "", slug: "", name: "SSL Certificate" }),
      typeDef({ id: "x6", executor_key: "", slug: "", name: "SNMP Device" }),
    ];
    expect(isPingMonitor(monitor("x1"), byName)).toBe(true);
    expect(isHttpMonitor(monitor("x2"), byName)).toBe(true);
    expect(isTcpMonitor(monitor("x3"), byName)).toBe(true);
    expect(isDnsMonitor(monitor("x4"), byName)).toBe(true);
    expect(isTlsMonitor(monitor("x5"), byName)).toBe(true);
    expect(isSnmpMonitor(monitor("x6"), byName)).toBe(true);
  });

  it("returns false when the type is unknown", () => {
    expect(isPingMonitor(monitor("nope"), types)).toBe(false);
  });
});

describe("RANGE_MILLIS", () => {
  it("defines all ranges in ascending order", () => {
    const ranges: MetricsRange[] = ["15m", "1h", "6h", "24h", "7d", "30d"];
    const values = ranges.map((r) => RANGE_MILLIS[r]);
    for (let i = 1; i < values.length; i++) {
      expect(values[i]).toBeGreaterThan(values[i - 1]);
    }
  });
});
