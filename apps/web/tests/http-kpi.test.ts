import { describe, expect, it } from "vitest";

import {
  probeHealthOf,
  summarizeHttp,
  toHttpProbeHealth,
  toHttpChartSeries,
  statusLabelOf,
  lastErrorOf,
} from "@/entities/resource/ui/monitoring/http/http-metrics";
import type { ProbeResult } from "@/entities/monitor/model/result";

const thresholds = { warning: 1000, critical: 3000 };

function result(overrides: Partial<ProbeResult>): ProbeResult {
  return {
    id: "r1",
    job_id: "j1",
    monitor_id: "m1",
    probe_location_id: "loc-ams",
    status: "up",
    success: true,
    duration_millis: 120,
    metrics: { reachability: 1, response_time_ms: 120, status_code: 200 },
    attributes: { probe_name: "Amsterdam", status_code: 200 },
    started_at: "2026-01-01T00:00:00Z",
    finished_at: "2026-01-01T00:00:00.1Z",
    ...overrides,
  };
}

describe("probeHealthOf", () => {
  it("is healthy for a successful fast check", () => {
    expect(probeHealthOf(result({}), thresholds)).toBe("healthy");
  });

  it("is warning above the warning threshold", () => {
    expect(probeHealthOf(result({ metrics: { response_time_ms: 1500 } }), thresholds)).toBe("warning");
  });

  it("is critical above the critical threshold", () => {
    expect(probeHealthOf(result({ metrics: { response_time_ms: 4000 } }), thresholds)).toBe("critical");
  });

  it("is critical for a failed check", () => {
    expect(probeHealthOf(result({ success: false }), thresholds)).toBe("critical");
  });
});

describe("toHttpProbeHealth", () => {
  it("attaches availability from the status series", () => {
    const stats = toHttpProbeHealth(
      [result({})],
      toHttpChartSeries(
        [
          {
            probe_id: "p1",
            probe_name: "Amsterdam",
            location: "Amsterdam",
            points: [
              { timestamp: "2026-01-01T00:00:00Z", value: 1 },
              { timestamp: "2026-01-01T00:01:00Z", value: 0.9 },
            ],
          },
        ],
        "status",
      ),
      thresholds,
    );
    expect(stats[0].availability).toBeCloseTo(95, 5);
    expect(stats[0].health).toBe("healthy");
    expect(stats[0].breakdown.dns).toBeNull();
  });
});

describe("summarizeHttp with series", () => {
  it("computes min/max/p95 from pooled series points", () => {
    const summary = summarizeHttp(
      [result({})],
      toHttpChartSeries(
        [
          {
            probe_id: "p1",
            probe_name: "Amsterdam",
            location: "Amsterdam",
            points: [
              { timestamp: "2026-01-01T00:00:00Z", value: 100 },
              { timestamp: "2026-01-01T00:01:00Z", value: 200 },
              { timestamp: "2026-01-01T00:02:00Z", value: 300 },
            ],
          },
        ],
        "response_time_ms",
      ),
    );
    expect(summary.minLatencyMs).toBe(100);
    expect(summary.maxLatencyMs).toBe(300);
    expect(summary.p95LatencyMs).toBe(300);
  });
});

describe("statusLabelOf / lastErrorOf", () => {
  it("formats known status codes readably", () => {
    expect(statusLabelOf(200, null)).toBe("200 OK");
    expect(statusLabelOf(503, null)).toBe("503 Service Unavailable");
    expect(statusLabelOf(418, null)).toBe("418");
  });

  it("falls back to the most recent error code when no status was recorded", () => {
    const probeHealth = [
      { errorCode: null, lastCheckedAt: "2026-01-01T00:00:00Z" },
      { errorCode: "unexpected_status_code", lastCheckedAt: "2026-01-01T00:02:00Z" },
      { errorCode: "timeout", lastCheckedAt: "2026-01-01T00:01:00Z" },
    ];
    expect(lastErrorOf(probeHealth)).toBe("unexpected_status_code");
    expect(statusLabelOf(null, lastErrorOf(probeHealth))).toBe("unexpected_status_code");
  });
});
