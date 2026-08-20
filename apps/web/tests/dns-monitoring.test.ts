import { describe, expect, it } from "vitest";

import { readDnsConfig, readDnsThresholds } from "@/entities/resource/ui/monitoring/dns/dns-config";
import {
  evaluateDnsHealth,
  evaluateMetric,
  dnsHealthTone,
} from "@/entities/resource/ui/monitoring/dns/dns-health";
import {
  summarizeDns,
  toDnsChartSeries,
  toDnsProbeStats,
} from "@/entities/resource/ui/monitoring/dns/dns-metrics";
import type { ProbeResult } from "@/entities/monitor/model/result";

describe("readDnsThresholds", () => {
  it("returns defaults when configuration is empty", () => {
    const t = readDnsThresholds(undefined);
    expect(t.responseTime).toEqual({ warning: 500, critical: 2000 });
  });

  it("reads thresholds from configuration.health_rules.response_time_ms", () => {
    const t = readDnsThresholds({
      health_rules: { response_time_ms: { warning: 200, critical: 800 } },
    });
    expect(t.responseTime).toEqual({ warning: 200, critical: 800 });
  });
});

describe("readDnsConfig", () => {
  it("reads execution parameters", () => {
    const cfg = readDnsConfig({
      record_type: "AAAA",
      resolver: "8.8.8.8:53",
      expected_values: "2001:db8::1, 2001:db8::2",
      timeout_ms: 1200,
      ip_version: "ipv6",
    });
    expect(cfg.recordType).toBe("AAAA");
    expect(cfg.resolver).toBe("8.8.8.8:53");
    expect(cfg.expectedValues).toEqual(["2001:db8::1", "2001:db8::2"]);
    expect(cfg.timeoutMs).toBe(1200);
    expect(cfg.ipVersion).toBe("ipv6");
  });

  it("accepts legacy server/nameserver resolver keys", () => {
    expect(readDnsConfig({ server: "1.1.1.1:53" }).resolver).toBe("1.1.1.1:53");
    expect(readDnsConfig({ nameserver: "9.9.9.9" }).resolver).toBe("9.9.9.9");
    expect(readDnsConfig(undefined).resolver).toBe("");
  });

  it("falls back to defaults", () => {
    const cfg = readDnsConfig(undefined);
    expect(cfg.recordType).toBe("A");
    expect(cfg.timeoutMs).toBe(5000);
    expect(cfg.ipVersion).toBe("auto");
  });
});

describe("evaluateDnsHealth", () => {
  it("is down when the backend reports down", () => {
    expect(evaluateDnsHealth({ lastStatus: "down", thresholds: readDnsThresholds(undefined) })).toBe("down");
  });

  it("is critical when the query failed", () => {
    expect(evaluateDnsHealth({ success: false, thresholds: readDnsThresholds(undefined) })).toBe("critical");
  });

  it("is critical when the expected record does not match", () => {
    expect(evaluateDnsHealth({
      lastStatus: "up",
      success: true,
      expectedRecordMatch: 0,
      responseTimeMs: 20,
      thresholds: readDnsThresholds(undefined),
    })).toBe("critical");
  });

  it("is healthy when response time is within thresholds", () => {
    expect(evaluateDnsHealth({
      lastStatus: "up",
      success: true,
      expectedRecordMatch: 1,
      responseTimeMs: 40,
      thresholds: readDnsThresholds(undefined),
    })).toBe("healthy");
  });

  it("is warning when response time exceeds the warning threshold", () => {
    expect(evaluateDnsHealth({
      lastStatus: "up",
      success: true,
      expectedRecordMatch: 1,
      responseTimeMs: 900,
      thresholds: readDnsThresholds(undefined),
    })).toBe("warning");
  });

  it("is critical when response time exceeds the critical threshold", () => {
    expect(evaluateDnsHealth({
      lastStatus: "up",
      success: true,
      expectedRecordMatch: 1,
      responseTimeMs: 2500,
      thresholds: readDnsThresholds(undefined),
    })).toBe("critical");
  });
});

describe("evaluateMetric", () => {
  it("is unknown when there is no value", () => {
    expect(evaluateMetric(null, { warning: 100, critical: 300 })).toBe("unknown");
  });
});

describe("dnsHealthTone", () => {
  it("maps states to badge tones", () => {
    expect(dnsHealthTone("healthy")).toBe("success");
    expect(dnsHealthTone("warning")).toBe("warning");
    expect(dnsHealthTone("critical")).toBe("destructive");
    expect(dnsHealthTone("down")).toBe("destructive");
    expect(dnsHealthTone("unknown")).toBe("muted");
  });
});

const baseResult: ProbeResult = {
  id: "r1",
  job_id: "j1",
  monitor_id: "m1",
  probe_location_id: "loc-amsterdam",
  status: "up",
  success: true,
  duration_millis: 60,
  metrics: { reachability: 1, response_time_ms: 40, answer_count: 2, ttl_seconds: 300, expected_record_match: 1 },
  attributes: { probe_name: "Amsterdam", record_type: "A", resolver: "8.8.8.8:53" },
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
    error_code: "nxdomain",
    error_message: "DNS returned NXDOMAIN",
    metrics: { reachability: 0 },
    attributes: { probe_name: "Montreal", record_type: "A", error_type: "nxdomain" },
  };
}

describe("toDnsProbeStats", () => {
  it("maps successful results with response time and answers", () => {
    const [stat] = toDnsProbeStats([baseResult]);
    expect(stat.location).toBe("Amsterdam");
    expect(stat.success).toBe(true);
    expect(stat.responseTimeMs).toBe(40);
    expect(stat.answerCount).toBe(2);
    expect(stat.expectedMatch).toBe(true);
    expect(stat.resolver).toBe("8.8.8.8:53");
  });

  it("maps failed results without fabricating timings", () => {
    const [stat] = toDnsProbeStats([failedResult()]);
    expect(stat.success).toBe(false);
    expect(stat.responseTimeMs).toBeNull();
    expect(stat.errorCode).toBe("nxdomain");
  });
});

describe("summarizeDns", () => {
  it("computes availability and averages", () => {
    const summary = summarizeDns([baseResult, failedResult()]);
    expect(summary.totalChecks).toBe(2);
    expect(summary.availability).toBe(50);
    expect(summary.responseTimeMs).toBe(40);
    expect(summary.answerCount).toBe(2);
  });
});

describe("toDnsChartSeries", () => {
  it("normalizes series points to {time, value}", () => {
    const series = toDnsChartSeries(
      [
        {
          probe_id: "p1",
          probe_name: "Amsterdam",
          location: "Amsterdam",
          points: [{ timestamp: "2026-01-01T00:00:00Z", value: 40 }],
        },
      ],
      "response_time_ms",
    );
    expect(series[0].probeName).toBe("Amsterdam");
    expect(series[0].points).toEqual([{ time: "2026-01-01T00:00:00Z", value: 40 }]);
  });
});
