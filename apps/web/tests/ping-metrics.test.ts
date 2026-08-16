import { describe, expect, it } from "vitest";

import {
  getMetricValue,
  summarize,
  toProbeStats,
  toChartSeries,
  PING_METRIC_KEYS,
} from "@/entities/resource/ui/monitoring/ping/ping-metrics";
import type { ProbeResult } from "@/entities/monitor/model/result";

function probe(partial: Partial<ProbeResult>): ProbeResult {
  return {
    id: "id",
    job_id: "job",
    monitor_id: "mon",
    probe_location_id: "loc",
    status: "up",
    success: true,
    duration_millis: 40,
    metrics: {},
    attributes: {},
    started_at: new Date().toISOString(),
    finished_at: new Date().toISOString(),
    ...partial,
  };
}

describe("getMetricValue", () => {
  it("reads a numeric metric", () => {
    expect(getMetricValue(probe({ metrics: { rtt_ms: 42 } }), PING_METRIC_KEYS.rtt)).toBe(42);
  });

  it("reads a numeric string", () => {
    expect(getMetricValue(probe({ metrics: { rtt_ms: "42" } }), PING_METRIC_KEYS.rtt)).toBe(42);
  });

  it("tries alias keys in order", () => {
    expect(getMetricValue(probe({ metrics: { latency_ms: 55 } }), PING_METRIC_KEYS.rtt)).toBe(55);
  });

  it("returns null when missing", () => {
    expect(getMetricValue(probe({ metrics: {} }), PING_METRIC_KEYS.rtt)).toBeNull();
  });
});

describe("toProbeStats", () => {
  it("derives location from attributes.probe_name", () => {
    const stats = toProbeStats([
      probe({ attributes: { probe_name: "Amsterdam" } }),
    ]);
    expect(stats[0].location).toBe("Amsterdam");
  });

  it("falls back to probe_location_id", () => {
    const stats = toProbeStats([probe({ probe_location_id: "loc-1" })]);
    expect(stats[0].location).toBe("loc-1");
  });
});

describe("summarize", () => {
  it("returns empty summary for no results", () => {
    const s = summarize([]);
    expect(s.availability).toBeNull();
    expect(s.totalChecks).toBe(0);
    expect(s.latency).toBeNull();
  });

  it("computes availability and latency aggregates", () => {
    const s = summarize([
      probe({ success: true, duration_millis: 40, metrics: { rtt_ms: 40 } }),
      probe({ success: false, duration_millis: 80, metrics: { rtt_ms: 80 } }),
      probe({ success: true, duration_millis: 60, metrics: { rtt_ms: 60 } }),
    ]);
    expect(s.availability).toBeCloseTo((2 / 3) * 100);
    expect(s.totalChecks).toBe(3);
    expect(s.successChecks).toBe(2);
    expect(s.failedChecks).toBe(1);
    expect(s.latency).toBeCloseTo(60);
    expect(s.latencyMin).toBe(40);
    expect(s.latencyMax).toBe(80);
  });

  it("aggregates packet loss and jitter", () => {
    const s = summarize([
      probe({ metrics: { packet_loss_percent: 1, jitter_ms: 5 } }),
      probe({ metrics: { packet_loss_percent: 3, jitter_ms: 15 } }),
    ]);
    expect(s.packetLoss).toBeCloseTo(2);
    expect(s.jitter).toBeCloseTo(10);
    expect(s.jitterMax).toBe(15);
  });

  it("computes lost packets from sent/received attributes", () => {
    const s = summarize([
      probe({ attributes: { packets_sent: 4, packets_received: 4 } }),
      probe({ attributes: { packets_sent: 4, packets_received: 3 } }),
    ]);
    expect(s.packetsSent).toBe(8);
    expect(s.packetsReceived).toBe(7);
    expect(s.packetsLost).toBe(1);
  });
});

describe("toChartSeries", () => {
  it("normalizes series into a stable chart shape", () => {
    const series = toChartSeries(
      [
        {
          probe_id: "p1",
          probe_name: "Amsterdam",
          location: "Amsterdam",
          metric_key: "rtt_ms",
          points: [{ timestamp: "2026-01-01T00:00:00Z", value: 42 }],
        },
      ],
      "rtt_ms",
    );
    expect(series).toEqual([
      {
        metric: "rtt_ms",
        location: "Amsterdam",
        probeName: "Amsterdam",
        points: [{ time: "2026-01-01T00:00:00Z", value: 42 }],
      },
    ]);
  });
});
