import { describe, expect, it } from "vitest";

import { readTcpConfig, readTcpThresholds } from "@/entities/resource/ui/monitoring/tcp/tcp-config";
import {
  evaluateTcpHealth,
  evaluateMetric,
  tcpHealthTone,
} from "@/entities/resource/ui/monitoring/tcp/tcp-health";
import {
  summarizeTcp,
  toTcpChartSeries,
  toTcpProbeStats,
} from "@/entities/resource/ui/monitoring/tcp/tcp-metrics";
import type { ProbeResult } from "@/entities/monitor/model/result";

describe("readTcpThresholds", () => {
  it("returns defaults when configuration is empty", () => {
    const t = readTcpThresholds(undefined);
    expect(t.connectTime).toEqual({ warning: 500, critical: 2000 });
  });

  it("reads thresholds from configuration.health_rules.connect_time_ms", () => {
    const t = readTcpThresholds({
      health_rules: { connect_time_ms: { warning: 300, critical: 900 } },
    });
    expect(t.connectTime).toEqual({ warning: 300, critical: 900 });
  });

  it("accepts legacy aliases", () => {
    const t = readTcpThresholds({
      health_rules: { connect_duration_ms: { warning: 100, critical: 400 } },
    });
    expect(t.connectTime).toEqual({ warning: 100, critical: 400 });
  });
});

describe("readTcpConfig", () => {
  it("reads execution parameters", () => {
    const cfg = readTcpConfig({
      port: 5432,
      timeout_ms: 1500,
      ip_version: "ipv4",
    });
    expect(cfg.port).toBe(5432);
    expect(cfg.timeoutMs).toBe(1500);
    expect(cfg.ipVersion).toBe("ipv4");
  });

  it("falls back to defaults", () => {
    const cfg = readTcpConfig(undefined);
    expect(cfg.port).toBe(0);
    expect(cfg.timeoutMs).toBe(5000);
    expect(cfg.ipVersion).toBe("auto");
  });
});

describe("evaluateTcpHealth", () => {
  it("is down when the backend reports down", () => {
    expect(evaluateTcpHealth({ lastStatus: "down", thresholds: readTcpThresholds(undefined) })).toBe("down");
  });

  it("is critical when the connection failed", () => {
    expect(evaluateTcpHealth({ success: false, thresholds: readTcpThresholds(undefined) })).toBe("critical");
  });

  it("is healthy when connect time is within thresholds", () => {
    expect(evaluateTcpHealth({
      lastStatus: "up",
      success: true,
      connectTimeMs: 50,
      thresholds: readTcpThresholds(undefined),
    })).toBe("healthy");
  });

  it("is warning when connect time exceeds the warning threshold", () => {
    expect(evaluateTcpHealth({
      lastStatus: "up",
      success: true,
      connectTimeMs: 800,
      thresholds: readTcpThresholds(undefined),
    })).toBe("warning");
  });

  it("is critical when connect time exceeds the critical threshold", () => {
    expect(evaluateTcpHealth({
      lastStatus: "up",
      success: true,
      connectTimeMs: 2500,
      thresholds: readTcpThresholds(undefined),
    })).toBe("critical");
  });
});

describe("evaluateMetric", () => {
  it("is unknown when there is no value", () => {
    expect(evaluateMetric(null, { warning: 100, critical: 300 })).toBe("unknown");
  });
});

describe("tcpHealthTone", () => {
  it("maps states to badge tones", () => {
    expect(tcpHealthTone("healthy")).toBe("success");
    expect(tcpHealthTone("warning")).toBe("warning");
    expect(tcpHealthTone("critical")).toBe("destructive");
    expect(tcpHealthTone("down")).toBe("destructive");
    expect(tcpHealthTone("unknown")).toBe("muted");
  });
});

const baseResult: ProbeResult = {
  id: "r1",
  job_id: "j1",
  monitor_id: "m1",
  probe_location_id: "loc-amsterdam",
  status: "up",
  success: true,
  duration_millis: 120,
  metrics: { reachability: 1, connect_time_ms: 45 },
  attributes: { probe_name: "Amsterdam", protocol: "tcp", port: 443, ip_family: "ipv4", remote_address: "203.0.113.10:443" },
  started_at: "2026-01-01T00:00:00Z",
  finished_at: "2026-01-01T00:00:00.1Z",
};

function failedResult(): ProbeResult {
  return {
    ...baseResult,
    id: "r2",
    probe_location_id: "loc-montreal",
    success: false,
    status: "down",
    error_code: "connection_refused",
    error_message: "dial tcp: connection refused",
    metrics: { reachability: 0 },
    attributes: { probe_name: "Montreal", error_type: "connection_refused" },
  };
}

describe("toTcpProbeStats", () => {
  it("maps successful results with connect time", () => {
    const [stat] = toTcpProbeStats([baseResult]);
    expect(stat.location).toBe("Amsterdam");
    expect(stat.success).toBe(true);
    expect(stat.connectTimeMs).toBe(45);
    expect(stat.errorCode).toBeNull();
  });

  it("maps failed results without fabricating timings", () => {
    const [stat] = toTcpProbeStats([failedResult()]);
    expect(stat.success).toBe(false);
    expect(stat.connectTimeMs).toBeNull();
    expect(stat.errorCode).toBe("connection_refused");
  });
});

describe("summarizeTcp", () => {
  it("computes availability and average connect time", () => {
    const summary = summarizeTcp([baseResult, failedResult()]);
    expect(summary.totalChecks).toBe(2);
    expect(summary.availability).toBe(50);
    expect(summary.connectTimeMs).toBe(45);
  });
});

describe("toTcpChartSeries", () => {
  it("normalizes series points to {time, value}", () => {
    const series = toTcpChartSeries(
      [
        {
          probe_id: "p1",
          probe_name: "Amsterdam",
          location: "Amsterdam",
          points: [{ timestamp: "2026-01-01T00:00:00Z", value: 45 }],
        },
      ],
      "connect_time_ms",
    );
    expect(series[0].probeName).toBe("Amsterdam");
    expect(series[0].points).toEqual([{ time: "2026-01-01T00:00:00Z", value: 45 }]);
  });
});
