import { describe, expect, it } from "vitest";

import {
  evaluateHttpHealth,
  evaluateMetric,
  httpHealthTone,
} from "@/entities/resource/ui/monitoring/http/http-health";
import { readHttpThresholds } from "@/entities/resource/ui/monitoring/http/http-config";

describe("evaluateHttpHealth", () => {
  it("is down when the backend reports down", () => {
    const state = evaluateHttpHealth({
      lastStatus: "down",
      responseTimeMs: 10,
      thresholds: readHttpThresholds(undefined),
    });
    expect(state).toBe("down");
  });

  it("is critical when the request failed", () => {
    const state = evaluateHttpHealth({
      success: false,
      responseTimeMs: null,
      thresholds: readHttpThresholds(undefined),
    });
    expect(state).toBe("critical");
  });

  it("is critical when the content assertion failed", () => {
    const state = evaluateHttpHealth({
      success: true,
      contentAssertion: 0,
      responseTimeMs: 50,
      thresholds: readHttpThresholds(undefined),
    });
    expect(state).toBe("critical");
  });

  it("is healthy when response time is within thresholds", () => {
    const state = evaluateHttpHealth({
      lastStatus: "up",
      success: true,
      statusCode: 200,
      responseTimeMs: 50,
      ttfbMs: 20,
      thresholds: readHttpThresholds(undefined),
    });
    expect(state).toBe("healthy");
  });

  it("is warning when response time exceeds the warning threshold", () => {
    const state = evaluateHttpHealth({
      lastStatus: "up",
      success: true,
      responseTimeMs: 2500,
      ttfbMs: 20,
      thresholds: readHttpThresholds(undefined),
    });
    expect(state).toBe("warning");
  });

  it("is critical when response time exceeds the critical threshold", () => {
    const state = evaluateHttpHealth({
      lastStatus: "up",
      success: true,
      responseTimeMs: 6000,
      ttfbMs: 20,
      thresholds: readHttpThresholds(undefined),
    });
    expect(state).toBe("critical");
  });

  it("is unknown with no metric data", () => {
    const state = evaluateHttpHealth({
      lastStatus: "up",
      success: undefined,
      statusCode: null,
      responseTimeMs: null,
      ttfbMs: null,
      thresholds: readHttpThresholds(undefined),
    });
    expect(state).toBe("healthy");
  });
});

describe("evaluateMetric", () => {
  it("is unknown when there is no value", () => {
    expect(evaluateMetric(null, { warning: 100, critical: 300 })).toBe("unknown");
  });
});

describe("httpHealthTone", () => {
  it("maps states to badge tones", () => {
    expect(httpHealthTone("healthy")).toBe("success");
    expect(httpHealthTone("warning")).toBe("warning");
    expect(httpHealthTone("critical")).toBe("destructive");
    expect(httpHealthTone("down")).toBe("destructive");
    expect(httpHealthTone("unknown")).toBe("muted");
  });
});
