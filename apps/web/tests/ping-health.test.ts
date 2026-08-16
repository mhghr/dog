import { describe, expect, it } from "vitest";

import {
  evaluatePingHealth,
  evaluateMetric,
  evaluateAvailability,
} from "@/entities/resource/ui/monitoring/ping/ping-health";
import type { PingThresholds } from "@/entities/resource/ui/monitoring/ping/ping-config";

const thresholds: PingThresholds = {
  latency: { warning: 200, critical: 500 },
  packetLoss: { warning: 1, critical: 5 },
  jitter: { warning: 30, critical: 80 },
};

describe("evaluateMetric", () => {
  it("returns unknown when there is no value", () => {
    expect(evaluateMetric(null, thresholds.latency)).toBe("unknown");
    expect(evaluateMetric(undefined, thresholds.latency)).toBe("unknown");
  });

  it("returns healthy below warning", () => {
    expect(evaluateMetric(100, thresholds.latency)).toBe("healthy");
  });

  it("returns warning between warning and critical", () => {
    expect(evaluateMetric(250, thresholds.latency)).toBe("warning");
  });

  it("returns critical at or above critical", () => {
    expect(evaluateMetric(500, thresholds.latency)).toBe("critical");
    expect(evaluateMetric(900, thresholds.latency)).toBe("critical");
  });
});

describe("evaluateAvailability", () => {
  it("returns unknown for null", () => {
    expect(evaluateAvailability(null)).toBe("unknown");
  });

  it("maps availability to states", () => {
    expect(evaluateAvailability(100)).toBe("healthy");
    expect(evaluateAvailability(99.5)).toBe("warning");
    expect(evaluateAvailability(50)).toBe("critical");
  });
});

describe("evaluatePingHealth", () => {
  it("returns down when last_status is down", () => {
    expect(
      evaluatePingHealth({ lastStatus: "down", latency: 10, thresholds }),
    ).toBe("down");
  });

  it("returns unknown when paused", () => {
    expect(
      evaluatePingHealth({ lastStatus: "paused", latency: 10, thresholds }),
    ).toBe("unknown");
  });

  it("returns unknown when there is no data and no up status", () => {
    expect(evaluatePingHealth({ lastStatus: "unknown", thresholds })).toBe("unknown");
  });

  it("returns healthy when all metrics are within limits", () => {
    expect(
      evaluatePingHealth({ lastStatus: "up", latency: 100, packetLoss: 0.2, jitter: 5, thresholds }),
    ).toBe("healthy");
  });

  it("returns the worst state across metrics (critical wins)", () => {
    expect(
      evaluatePingHealth({ lastStatus: "up", latency: 100, packetLoss: 6, jitter: 5, thresholds }),
    ).toBe("critical");
  });

  it("returns warning when only one metric crosses warning", () => {
    expect(
      evaluatePingHealth({ lastStatus: "up", latency: 300, packetLoss: 0.2, jitter: 5, thresholds }),
    ).toBe("warning");
  });

  it("never reports healthy when all metrics are missing", () => {
    expect(
      evaluatePingHealth({ lastStatus: "up", latency: null, packetLoss: null, jitter: null, thresholds }),
    ).toBe("healthy");
  });
});
