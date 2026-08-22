import { describe, expect, it } from "vitest";

import {
  isValidHostname,
  isValidPort,
  isValidUrl,
  parseStatusCodes,
  validateMonitoringConfig,
  type ValidationValues,
} from "@/entities/resource/ui/settings/validation";

function values(overrides: Partial<ValidationValues>): ValidationValues {
  return {
    type: "ping",
    target: "example.com",
    config: {},
    intervalSeconds: 30,
    timeoutMillis: 5000,
    retries: 2,
    ...overrides,
  };
}

describe("isValidPort", () => {
  it("accepts valid ports and rejects invalid ones", () => {
    expect(isValidPort(80)).toBe(true);
    expect(isValidPort(65535)).toBe(true);
    expect(isValidPort(0)).toBe(false);
    expect(isValidPort(65536)).toBe(false);
    expect(isValidPort("abc")).toBe(false);
    expect(isValidPort(1.5)).toBe(false);
  });
});

describe("isValidHostname", () => {
  it("accepts hostnames and IPs", () => {
    expect(isValidHostname("example.com")).toBe(true);
    expect(isValidHostname("192.168.1.1")).toBe(true);
    expect(isValidHostname("")).toBe(false);
    expect(isValidHostname("not a hostname!")).toBe(false);
  });
});

describe("isValidUrl", () => {
  it("accepts http(s) URLs only", () => {
    expect(isValidUrl("https://example.com/path")).toBe(true);
    expect(isValidUrl("http://example.com")).toBe(true);
    expect(isValidUrl("ftp://example.com")).toBe(false);
    expect(isValidUrl("not-a-url")).toBe(false);
  });
});

describe("parseStatusCodes", () => {
  it("parses comma-separated integers", () => {
    expect(parseStatusCodes("200, 404")).toEqual([200, 404]);
    expect(parseStatusCodes("200,abc")).toEqual([200]);
    expect(parseStatusCodes("")).toBeNull();
    expect(parseStatusCodes(200)).toBeNull();
  });
});

describe("validateMonitoringConfig", () => {
  it("validates common execution constraints", () => {
    const errors = validateMonitoringConfig(values({ intervalSeconds: 1, retries: 20, target: " " }));
    expect(errors.interval_seconds).toBeDefined();
    expect(errors.retries).toBeDefined();
    expect(errors.target).toBeDefined();
  });

  it("validates ping packet count", () => {
    const errors = validateMonitoringConfig(values({ type: "ping", config: { count: 99 } }));
    expect(errors.count).toBeDefined();
    expect(validateMonitoringConfig(values({ type: "ping", config: { count: 3 } })).count).toBeUndefined();
  });

  it("validates http method, url and status codes", () => {
    const errors = validateMonitoringConfig(
      values({ type: "http", config: { method: "PATCH", url: "not valid", expected_status_codes: "x" } }),
    );
    expect(errors.method).toBeUndefined();
    expect(errors.url).toBeDefined();
    expect(errors.expected_status_codes).toBeDefined();
  });

  it("validates snmp v3 credentials", () => {
    const errors = validateMonitoringConfig(
      values({
        type: "snmp",
        config: { host: "10.0.0.1", port: 161, version: "3", security_level: "authPriv" },
      }),
    );
    expect(errors.username).toBeDefined();
    expect(errors.authentication_secret).toBeDefined();
    expect(errors.privacy_secret).toBeDefined();
  });

  it("returns no errors for a valid config", () => {
    const errors = validateMonitoringConfig(
      values({ type: "http", config: { method: "GET", url: "https://example.com", expected_status_codes: "200" } }),
    );
    expect(errors).toEqual({});
  });
});
