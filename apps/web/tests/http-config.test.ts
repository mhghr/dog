import { describe, expect, it } from "vitest";

import {
  readHttpConfig,
  readHttpThresholds,
} from "@/entities/resource/ui/monitoring/http/http-config";

describe("readHttpThresholds", () => {
  it("returns defaults when configuration is empty", () => {
    const t = readHttpThresholds(undefined);
    expect(t.responseTime).toEqual({ warning: 2000, critical: 5000 });
    expect(t.ttfb).toEqual({ warning: 1000, critical: 3000 });
    expect(t.dnsDuration).toEqual({ warning: 500, critical: 2000 });
    expect(t.connectDuration).toEqual({ warning: 500, critical: 2000 });
    expect(t.tlsDuration).toEqual({ warning: 500, critical: 2000 });
  });

  it("reads thresholds from configuration.health_rules.response_time_ms", () => {
    const t = readHttpThresholds({
      health_rules: { response_time_ms: { warning: 500, critical: 1500 } },
    });
    expect(t.responseTime).toEqual({ warning: 500, critical: 1500 });
  });

  it("accepts ttfb_ms and dns_duration_ms aliases", () => {
    const t = readHttpThresholds({
      health_rules: {
        ttfb_ms: { warning: 400, critical: 1200 },
        dns_duration_ms: { warning: 100, critical: 400 },
      },
    });
    expect(t.ttfb).toEqual({ warning: 400, critical: 1200 });
    expect(t.dnsDuration).toEqual({ warning: 100, critical: 400 });
  });
});

describe("readHttpConfig", () => {
  it("reads execution parameters", () => {
    const cfg = readHttpConfig({
      method: "POST",
      follow_redirects: false,
      verify_tls: false,
      max_redirects: 2,
      expected_status_codes: "200, 204",
      body_contains: "healthy",
      request_body: '{"ping":1}',
      headers: { "X-Test": "abc" },
    });
    expect(cfg.method).toBe("POST");
    expect(cfg.followRedirects).toBe(false);
    expect(cfg.verifyTls).toBe(false);
    expect(cfg.maxRedirects).toBe(2);
    expect(cfg.expectedStatusCodes).toEqual([200, 204]);
    expect(cfg.bodyContains).toBe("healthy");
    expect(cfg.requestBody).toBe('{"ping":1}');
    expect(cfg.headers).toEqual({ "X-Test": "abc" });
  });

  it("falls back to executor defaults", () => {
    const cfg = readHttpConfig(undefined);
    expect(cfg.method).toBe("GET");
    expect(cfg.followRedirects).toBe(true);
    expect(cfg.verifyTls).toBe(true);
    expect(cfg.maxRedirects).toBe(5);
    expect(cfg.expectedStatusCodes).toEqual([200]);
    expect(cfg.bodyContains).toBe("");
  });

  it("accepts verify_ssl as a verify_tls alias", () => {
    const cfg = readHttpConfig({ verify_ssl: false });
    expect(cfg.verifyTls).toBe(false);
  });

  it("parses expected_status_codes as an array of numbers", () => {
    const cfg = readHttpConfig({ expected_status_codes: "200,201,204" });
    expect(cfg.expectedStatusCodes).toEqual([200, 201, 204]);
  });
});
