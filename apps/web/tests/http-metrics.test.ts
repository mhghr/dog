import { describe, expect, it } from "vitest";

import {
  summarizeHttp,
  toHttpChartSeries,
  toHttpProbeStats,
} from "@/entities/resource/ui/monitoring/http/http-metrics";
import type { ProbeResult } from "@/entities/monitor/model/result";

const baseResult: ProbeResult = {
  id: "r1",
  job_id: "j1",
  monitor_id: "m1",
  probe_location_id: "loc-amsterdam",
  status: "up",
  success: true,
  duration_millis: 120,
  metrics: { reachability: 1, response_time_ms: 120, ttfb_ms: 40, response_size_bytes: 2048, content_assertion: 1 },
  attributes: { status_code: 200, probe_name: "Amsterdam", method: "GET" },
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
    error_code: "connection_failed",
    error_message: "dial tcp: connection refused",
    metrics: { reachability: 0 },
    attributes: { probe_name: "Montreal", method: "GET", error_type: "connection_failed" },
  };
}

describe("toHttpProbeStats", () => {
  it("maps successful results with status code and timings", () => {
    const [stat] = toHttpProbeStats([baseResult]);
    expect(stat.location).toBe("Amsterdam");
    expect(stat.success).toBe(true);
    expect(stat.statusCode).toBe(200);
    expect(stat.responseTimeMs).toBe(120);
    expect(stat.ttfbMs).toBe(40);
    expect(stat.errorCode).toBeNull();
  });

  it("maps failed results preserving the error without fabricating timings", () => {
    const [stat] = toHttpProbeStats([failedResult()]);
    expect(stat.success).toBe(false);
    expect(stat.statusCode).toBeNull();
    expect(stat.responseTimeMs).toBeNull();
    expect(stat.errorCode).toBe("connection_failed");
    expect(stat.errorMessage).toContain("connection refused");
  });
});

describe("summarizeHttp", () => {
  it("computes availability and averages from available timings", () => {
    const summary = summarizeHttp([baseResult, failedResult()]);
    expect(summary.totalChecks).toBe(2);
    expect(summary.successChecks).toBe(1);
    expect(summary.failedChecks).toBe(1);
    expect(summary.availability).toBe(50);
    // Only the successful result has a response time.
    expect(summary.responseTimeMs).toBe(120);
    expect(summary.ttfbMs).toBe(40);
  });

  it("returns null averages when no result has timings", () => {
    const summary = summarizeHttp([failedResult()]);
    expect(summary.responseTimeMs).toBeNull();
    expect(summary.ttfbMs).toBeNull();
    expect(summary.availability).toBe(0);
  });
});

describe("toHttpChartSeries", () => {
  it("normalizes series points to {time, value}", () => {
    const series = toHttpChartSeries(
      [
        {
          probe_id: "p1",
          probe_name: "Amsterdam",
          location: "Amsterdam",
          points: [{ timestamp: "2026-01-01T00:00:00Z", value: 120 }],
        },
      ],
      "response_time_ms",
    );
    expect(series[0].probeName).toBe("Amsterdam");
    expect(series[0].points).toEqual([{ time: "2026-01-01T00:00:00Z", value: 120 }]);
  });
});
