import { describe, expect, it } from "vitest";

import { readTlsConfig, readTlsThresholds } from "@/entities/resource/ui/monitoring/tls/tls-config";
import {
  compareLowerIsWorse,
  evaluateExpiry,
  evaluateMetric,
  evaluateTlsHealth,
  tlsHealthTone,
} from "@/entities/resource/ui/monitoring/tls/tls-health";
import {
  summarizeTls,
  toTlsChartSeries,
  toTlsProbeStats,
} from "@/entities/resource/ui/monitoring/tls/tls-metrics";
import type { ProbeResult } from "@/entities/monitor/model/result";

describe("readTlsThresholds", () => {
  it("returns defaults when configuration is empty", () => {
    const t = readTlsThresholds(undefined);
    expect(t.handshakeTime).toEqual({ warning: 500, critical: 2000 });
    expect(t.certificateExpiryDays).toEqual({ warning: 30, critical: 14 });
  });

  it("reads thresholds from configuration.health_rules", () => {
    const t = readTlsThresholds({
      health_rules: {
        handshake_time_ms: { warning: 400, critical: 1200 },
        certificate_expiry_days: { warning: 45, critical: 20 },
      },
    });
    expect(t.handshakeTime).toEqual({ warning: 400, critical: 1200 });
    expect(t.certificateExpiryDays).toEqual({ warning: 45, critical: 20 });
  });

  it("accepts legacy expiry aliases", () => {
    const t = readTlsThresholds({
      health_rules: { days_remaining: { warning: 60, critical: 30 } },
    });
    expect(t.certificateExpiryDays).toEqual({ warning: 60, critical: 30 });
  });
});

describe("readTlsConfig", () => {
  it("reads execution parameters", () => {
    const cfg = readTlsConfig({
      port: 8443,
      server_name: "api.example.com",
      verify_tls: false,
      min_tls_version: "1.3",
      timeout_ms: 8000,
      ip_version: "ipv6",
    });
    expect(cfg.port).toBe(8443);
    expect(cfg.serverName).toBe("api.example.com");
    expect(cfg.verifyTls).toBe(false);
    expect(cfg.minTlsVersion).toBe("1.3");
    expect(cfg.timeoutMs).toBe(8000);
    expect(cfg.ipVersion).toBe("ipv6");
  });

  it("accepts verify_ssl as a verify_tls alias", () => {
    expect(readTlsConfig({ verify_ssl: false }).verifyTls).toBe(false);
  });

  it("falls back to defaults", () => {
    const cfg = readTlsConfig(undefined);
    expect(cfg.port).toBe(443);
    expect(cfg.verifyTls).toBe(true);
    expect(cfg.minTlsVersion).toBe("1.2");
    expect(cfg.ipVersion).toBe("auto");
  });
});

describe("evaluateTlsHealth", () => {
  const thresholds = readTlsThresholds(undefined);

  it("is down when the backend reports down", () => {
    expect(evaluateTlsHealth({ lastStatus: "down", thresholds })).toBe("down");
  });

  it("is critical when the handshake failed", () => {
    expect(evaluateTlsHealth({ success: false, thresholds })).toBe("critical");
  });

  it("warns when verification is disabled", () => {
    const state = evaluateTlsHealth({
      lastStatus: "up",
      success: true,
      verified: false,
      handshakeTimeMs: 50,
      certificateExpiryDays: 90,
      thresholds,
    });
    expect(state).toBe("warning");
  });

  it("is healthy when all metrics are within thresholds", () => {
    expect(evaluateTlsHealth({
      lastStatus: "up",
      success: true,
      verified: true,
      handshakeTimeMs: 60,
      certificateExpiryDays: 90,
      thresholds,
    })).toBe("healthy");
  });

  it("is warning when handshake exceeds the warning threshold", () => {
    expect(evaluateTlsHealth({
      lastStatus: "up",
      success: true,
      verified: true,
      handshakeTimeMs: 900,
      certificateExpiryDays: 90,
      thresholds,
    })).toBe("warning");
  });

  it("is warning when expiry falls under the warning threshold", () => {
    expect(evaluateTlsHealth({
      lastStatus: "up",
      success: true,
      verified: true,
      handshakeTimeMs: 50,
      certificateExpiryDays: 20,
      thresholds,
    })).toBe("warning");
  });

  it("is critical when expiry falls under the critical threshold", () => {
    expect(evaluateTlsHealth({
      lastStatus: "up",
      success: true,
      verified: true,
      handshakeTimeMs: 50,
      certificateExpiryDays: 10,
      thresholds,
    })).toBe("critical");
  });
});

describe("expiry thresholds", () => {
  it("treats lower days as worse", () => {
    expect(compareLowerIsWorse(5, { warning: 30, critical: 14 })).toBe("critical");
    expect(compareLowerIsWorse(20, { warning: 30, critical: 14 })).toBe("warning");
    expect(compareLowerIsWorse(90, { warning: 30, critical: 14 })).toBe("healthy");
    expect(compareLowerIsWorse(null, { warning: 30, critical: 14 })).toBe("healthy");
  });

  it("is unknown when there is no expiry value", () => {
    expect(evaluateExpiry(null, { warning: 30, critical: 14 })).toBe("unknown");
  });

  it("is unknown when there is no handshake value", () => {
    expect(evaluateMetric(null, { warning: 500, critical: 2000 })).toBe("unknown");
  });
});

describe("tlsHealthTone", () => {
  it("maps states to badge tones", () => {
    expect(tlsHealthTone("healthy")).toBe("success");
    expect(tlsHealthTone("warning")).toBe("warning");
    expect(tlsHealthTone("critical")).toBe("destructive");
    expect(tlsHealthTone("down")).toBe("destructive");
    expect(tlsHealthTone("unknown")).toBe("muted");
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
  metrics: { reachability: 1, handshake_time_ms: 60, certificate_expiry_days: 90 },
  attributes: {
    probe_name: "Amsterdam",
    verified: true,
    tls_version: "1.3",
    certificate_issuer: "CN=R3,O=Let's Encrypt",
    port: 443,
  },
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
    error_code: "certificate_expired",
    error_message: "certificate expired",
    metrics: { reachability: 0, certificate_expiry_days: -3 },
    attributes: { probe_name: "Montreal", verified: false, error_type: "certificate_expired" },
  };
}

describe("toTlsProbeStats", () => {
  it("maps successful results with handshake, expiry and verification", () => {
    const [stat] = toTlsProbeStats([baseResult]);
    expect(stat.location).toBe("Amsterdam");
    expect(stat.success).toBe(true);
    expect(stat.handshakeTimeMs).toBe(60);
    expect(stat.certificateExpiryDays).toBe(90);
    expect(stat.verified).toBe(true);
    expect(stat.tlsVersion).toBe("1.3");
  });

  it("maps failed results preserving expiry and error", () => {
    const [stat] = toTlsProbeStats([failedResult()]);
    expect(stat.success).toBe(false);
    expect(stat.handshakeTimeMs).toBeNull();
    expect(stat.certificateExpiryDays).toBe(-3);
    expect(stat.verified).toBe(false);
    expect(stat.errorCode).toBe("certificate_expired");
  });
});

describe("summarizeTls", () => {
  it("computes availability and averages", () => {
    const summary = summarizeTls([baseResult, failedResult()]);
    expect(summary.totalChecks).toBe(2);
    expect(summary.availability).toBe(50);
    expect(summary.handshakeTimeMs).toBe(60);
    // Expired certificate on the failed probe carries -3 days; both values
    // are valid measurements, so the average includes them.
    expect(summary.certificateExpiryDays).toBe(43.5);
  });
});

describe("toTlsChartSeries", () => {
  it("normalizes series points to {time, value}", () => {
    const series = toTlsChartSeries(
      [
        {
          probe_id: "p1",
          probe_name: "Amsterdam",
          location: "Amsterdam",
          points: [{ timestamp: "2026-01-01T00:00:00Z", value: 90 }],
        },
      ],
      "certificate_expiry_days",
    );
    expect(series[0].probeName).toBe("Amsterdam");
    expect(series[0].points).toEqual([{ time: "2026-01-01T00:00:00Z", value: 90 }]);
  });
});
